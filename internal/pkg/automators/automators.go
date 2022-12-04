package automators

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/jboakyedonkor/ping-app/internal/pkg/cache"
	"go.uber.org/zap"
)

type Automator struct {
	cache      Cacher
	secretKey  []byte
	jobSetName string
	scheduler  *gocron.Scheduler
	logger     *zap.SugaredLogger
}

type Cacher interface {
	InsertData(ctx context.Context, key, data string) error
	GetData(ctx context.Context, key string) (string, error)
	DeleteData(ctx context.Context, key string) error
	GetSet(ctx context.Context, key string) (map[string]struct{}, error)
	DeleteSet(ctx context.Context, key string) error
	DeleteFromSet(ctx context.Context, setName string, keys ...string) error
	UpdateSet(ctx context.Context, setName string, keys ...string) error
}

type jobFunc func(config *JobConfig, logger *zap.SugaredLogger) (any, error)

const (
	reconcileTickerDuration = 10 * time.Second
)

func NewAutomator(cache Cacher, secretKey []byte, scheduler *gocron.Scheduler, logger *zap.SugaredLogger) *Automator {
	return &Automator{
		cache:      cache,
		secretKey:  secretKey,
		scheduler:  scheduler,
		logger:     logger,
		jobSetName: "jobs_set",
	}
}

func (a *Automator) CreateNewJob(ctx context.Context, config JobConfig) (string, error) {
	logger := a.logger.With("context", ctx)
	UUID := uuid.New()

	config.UID = UUID
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error marshalling job config: %w", err)
	}

	jobInfo := string(bytes)

	encryptedJob, err := EncryptJobInfo(a.secretKey, jobInfo)
	if err != nil {
		err := fmt.Errorf("error encrypting job config: %w", err)
		logger.Error(err)
		return "", err
	}

	_, err = a.scheduler.CronWithSeconds(config.CronExpression).Tag(config.UID.String()).Do(templateJobFunc, &config, a.logger)
	if err != nil {
		err := fmt.Errorf("error scheduling job: %w", err)
		logger.Error(err)
		return "", err
	}

	if err := a.cache.InsertData(ctx, config.UID.String(), encryptedJob); err != nil {
		if err := a.scheduler.RemoveByTag(config.UID.String()); err != nil {

			logger.Errorf("error deleting job after error insert job config into cache: %w", err)
			return "", fmt.Errorf("error removing job: %w", err)
		}
		logger.Errorf("error inserting new job into cache: %w")
		return "", fmt.Errorf("error inserting new job into cache: %w", err)
	}

	if err := a.cache.UpdateSet(ctx, a.jobSetName, config.UID.String()); err != nil {
		logger.Errorf("error updating job set: %w", err)
		return "", fmt.Errorf("error updating job set: %w", err)
	}

	logger.Debugw("created new job", "jobUID", config.UID.String())
	return config.UID.String(), nil
}

func (a *Automator) DeleteJob(ctx context.Context, jobUID uuid.UUID) error {
	logger := a.logger.With("context", ctx)
	jobID := jobUID.String()
	if err := a.scheduler.RemoveByTag(jobID); err != nil {
		err := fmt.Errorf("error removing job from scheduler: %w", err)
		logger.Error(err)
		return err
	}

	if err := a.cache.DeleteData(ctx, jobID); err != nil {
		err := fmt.Errorf("error removing job config cache: %w", err)
		logger.Error(err)
		return err
	}

	if err := a.cache.DeleteFromSet(ctx, a.jobSetName, jobID); err != nil {
		err := fmt.Errorf("error removing job config cache set: %w", err)
		logger.Error(err)
		return err
	}
	return nil
}

func (a *Automator) GetJob(ctx context.Context, jobUID uuid.UUID) (*JobConfig, error) {
	logger := a.logger.With("context", ctx)
	data, err := a.cache.GetData(ctx, jobUID.String())
	if err != nil {
		if _, ok := err.(*cache.NotFoundError); ok {
			return nil, err
		}
		err := fmt.Errorf("error retrieving job data: %w", err)
		logger.Error(err)
		return nil, err
	}

	config, err := DecryptJobInfo(a.secretKey, data)
	if err != nil {
		err := fmt.Errorf("error decrypting job data: %w", err)
		logger.Error(err)
		return nil, err
	}

	return config, nil
}

func (a *Automator) ReconcileJobs() {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(reconcileTickerDuration)

	isUp := true
	for isUp {
		select {

		case <-ticker.C:
			a.logger.Info("start reconcile ticker jobs")
			go a.reconcileJobs()

		case <-quit:
			ticker.Stop()
			a.scheduler.Stop()
			a.logger.Info("ticker and scheduler stopped")
			isUp = false
		}
	}

	a.logger.Info("reconciling of jobs stopped")
}

func (a *Automator) reconcileJobs() {

	ctx := context.Background()

	jobSet, err := a.cache.GetSet(ctx, a.jobSetName)
	if err != nil {
		a.logger.Errorf("error getting job set: ", err)
		return
	}

	tags := make(map[string]struct{}, 0)

	for _, job := range a.scheduler.Jobs() {
		for _, tag := range job.Tags() {
			tags[tag] = struct{}{}
		}
	}

	jobUUIDs := make([]string, 0)

	for key := range jobSet {
		if _, ok := tags[key]; !ok {
			jobUUIDs = append(jobUUIDs, key)
		}
	}
	if len(jobUUIDs) > 0 {
		a.logger.Infow("missing jobs", "ids", jobUUIDs)
	}

	for _, uuid := range jobUUIDs {
		data, err := a.cache.GetData(ctx, uuid)
		if err != nil {
			continue
		}

		config, err := DecryptJobInfo(a.secretKey, data)
		if err != nil {
			continue
		}

		a.scheduler.CronWithSeconds(config.CronExpression).Tag(config.UID.String()).Do(templateJobFunc, config, a.logger)
	}
}

func (a *Automator) GetRunningJobs(ctx context.Context) ([]*JobConfig, error) {
	possibleTags := make([]string, 0)
	for _, job := range a.scheduler.Jobs() {
		possibleTags = append(possibleTags, job.Tags()...)
	}

	jobConfigs := make([]*JobConfig, 0)

	for _, tag := range possibleTags {
		uid, err := uuid.Parse(tag)
		if err != nil {
			continue
		}

		data, err := a.cache.GetData(ctx, uid.String())
		if err != nil {
			if _, ok := err.(*cache.NotFoundError); ok {
				continue
			}

			return nil, fmt.Errorf("error retrieving job data: %w", err)
		}

		config, err := DecryptJobInfo(a.secretKey, data)
		if err != nil {
			return nil, fmt.Errorf("error decrypting job data: %w", err)
		}

		jobConfigs = append(jobConfigs, config)
	}
	return jobConfigs, nil
}

func templateJobFunc(config *JobConfig, joblogger *zap.SugaredLogger) (any, error) {

	logger := joblogger.With("job_id", config.UID)
	start := time.Now()
	ctx := context.Background()

	client := http.Client{}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, config.Task.URL, nil)
	if err != nil {
		err := fmt.Errorf("error creating request: %w", err)
		logger.Error(err)
		return nil, err
	}

	scheme := config.Task.AuthHeader.Scheme
	auth := ""
	if scheme != "" {
		if scheme != "Bearer" && scheme != "Basic" && scheme != "Digest" {
			err := fmt.Errorf("invalid scheme")
			logger.Error(err)
			return nil, err
		}
		auth = fmt.Sprintf("%s %s", scheme, config.Task.AuthHeader.Parameters)
		request.Header.Add("Authorization", auth)
	}

	response, err := client.Do(request)
	if err != nil {
		err := fmt.Errorf("error making request: %w", err)
		logger.Error(err)
		return nil, err
	}
	defer response.Body.Close()

	var r any

	if response.Body != nil && strings.Contains(response.Header.Get("Content-Type"), "application/json") {
		err = json.NewDecoder(response.Body).Decode(&r)
		if err != nil {
			err := fmt.Errorf("error decoding response: %w", err)
			logger.Error(err)
			return nil, err
		}
	}
	jobDuration := time.Since(start)

	logger.Infow("job completed", "body", r, "status_code", response.StatusCode, "duration_ns", jobDuration.Nanoseconds())
	return r, nil
}

func EncryptJobInfo(key []byte, jobInfo string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("error creating cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error creating GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())

	cipherText := aesGCM.Seal(nonce, nonce, []byte(jobInfo), nil)

	return hex.EncodeToString(cipherText), nil

}

func DecryptJobInfo(key []byte, encryptedData string) (*JobConfig, error) {

	encryptedBytes, err := hex.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("error decoding encrypted data: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("error creating GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	nonce := encryptedBytes[:nonceSize]
	encryptedJobInfo := encryptedBytes[nonceSize:]

	cipherText, err := aesGCM.Open(nil, nonce, encryptedJobInfo, nil)
	if err != nil {
		return nil, fmt.Errorf("error decrypting config data: %s", err)
	}
	var config JobConfig
	if err := json.Unmarshal(cipherText, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config data: %w", err)
	}
	return &config, nil

}
