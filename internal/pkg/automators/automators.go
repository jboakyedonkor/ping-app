package automators

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Automator struct {
	cache     Cacher
	secretKey []byte
	scheduler *gocron.Scheduler
	logger    *zap.SugaredLogger
}

type Cacher interface {
	InsertData(key, data string) error
	GetData(key string) (string, error)
	DeleteData(key string) error
}

func NewAutomator(cache Cacher, secretKey []byte, scheduler *gocron.Scheduler, logger *zap.SugaredLogger) *Automator {
	return &Automator{
		cache:     cache,
		secretKey: secretKey,
		scheduler: scheduler,
		logger:    logger,
	}
}

func (a *Automator) CreateNewJob(ctx context.Context, cronExpression string, task *Task) (string, error) {
	logger := a.logger.With("context", ctx)
	UUID := uuid.New()

	jobConfig := JobConfig{
		CronExpression: cronExpression,
		UID:            UUID,
		Task:           *task,
	}
	bytes, err := json.Marshal(jobConfig)
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

	jobFunc, err := createJobFunc(jobConfig)
	if err != nil {
		err := fmt.Errorf("error creating job function: %w", err)
		logger.Error(err)
		return "", err

	}

	if _, err := a.scheduler.Cron(cronExpression).Tag(jobConfig.UID.String()).Do(jobFunc); err != nil {
		err := fmt.Errorf("error scheduling job: %w", err)
		logger.Error(err)
		return "", err
	}

	if err := a.cache.InsertData(jobConfig.UID.String(), encryptedJob); err != nil {
		if err := a.scheduler.RemoveByTag(jobConfig.UID.String()); err != nil {

			logger.Errorf("error deleting job after error insert job config into cache: %w", err)
			return "", fmt.Errorf("error removing job: %w", err)
		}
		logger.Errorf("error inserting new job into cache: %w")
		return "", fmt.Errorf("error inserting new job into cache: %w", err)
	}

	logger.Debugw("created new job", "jobUID", jobConfig.UID.String())
	return jobConfig.UID.String(), nil
}

func (a *Automator) DeleteJob(ctx context.Context, jobUID uuid.UUID) error {
	logger := a.logger.With("context", ctx)

	if err := a.scheduler.RemoveByTag(jobUID.String()); err != nil {
		err := fmt.Errorf("error removing job from scheduler: %w", err)
		logger.Error(err)
		return err
	}

	if err := a.cache.DeleteData(jobUID.String()); err != nil {
		err := fmt.Errorf("error removing job config cache: %w", err)
		logger.Error(err)
		return err
	}
	return nil
}

func createJobFunc(config JobConfig) (func() (any, error), error) {

	scheme := config.Task.AuthHeader.Scheme
	auth := ""
	if scheme != "" {
		if scheme != "Bearer" && scheme != "Basic" && scheme != "Digest" {
			return nil, fmt.Errorf("invalid")
		}
		auth = fmt.Sprintf("%s %s", scheme, config.Task.AuthHeader.Parameters)
	}

	return func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), config.Task.Timeout)
		defer cancel()

		client := http.Client{}

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, config.Task.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		request.Header.Add("Authorization", auth)
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("error making request: %w", err)
		}
		defer response.Body.Close()

		var r string
		err = json.NewDecoder(response.Body).Decode(&r)
		if err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}
		return r, nil
	}, nil

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
