package automators_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jboakyedonkor/ping-app/internal/pkg/automators"
	"github.com/jboakyedonkor/ping-app/internal/pkg/mock"
)

func TestDecryptJobInfo(t *testing.T) {
	t.Parallel()

	type args struct {
		key           []byte
		encryptedData string
	}
	tests := []struct {
		name    string
		args    args
		want    *automators.JobConfig
		wantErr bool
	}{

		{
			name: "sucess decrypt case",
			args: args{
				key:           []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				encryptedData: "0000000000000000000000005410d9c086dddbb49fd8b82f36a7a821bb5915f0033fa1a6304544d7a443532fd89e56b93f494e6e6057608cd33060d8483e27a66596b019a76079586d72037c32805945278d4b0080e078fbd5131e7d3be654b56f33091f1e2719434f19f886da96d82edd9dc726c96612f1eefb20abd40574423d7c4209d796515b06cf13f9eaadc537a4f214e8c15dcc3fc0173ec0a4e59740308b69c6fffb04f92224a5000b211ff7864476c014faf9b7888aa22d85b8c151ab726764a8c8b86aa8921a1bebb731f2da8d0e0d99d013996ef020e2566da8",
			},
			want: &automators.JobConfig{
				CronExpression: "* */1 * * *",
				UID:            uuid.MustParse("a8c14eb0-0fba-4b75-a461-e4d380317ab7"),
				Task: automators.Task{
					URL:     "http://test.org/ping",
					Timeout: time.Minute,
					AuthHeader: automators.AuthHeader{
						Scheme:     "Bearer",
						Parameters: "test-token",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid key",
			args: args{
				key:           []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				encryptedData: "000000000000000000000000d8032ef78af7220811b0a6c21d38b3913598f72387b2d0c87be75e58780b3668ca7f79772b43457e725b27e1f62debf08d299299fca88d3e7144fda4b2c7b5c8690bc7e5346dea8c7c49e66dd2a7edbbecbc30a498e8f038944f81fa34e4f07010b4863554dcab48cf703f202142861653bb950e09e0cd6a17eca0bae1cdc8e6896f2d7eab1b50e34bacf28f44808eb9c19b28ad908f097d1027f5671af81aca4dbb9871b4ade51f8b6edc494b19696db87d95ec975f1b4f45e527ac4f3a7e93b9757e46a418b9584511fcce7eaf574f0707a59dd0c55b5e5fb874e425d4ae3cf90ae567b8dff022cee33f7c",
			},
			want:    nil,
			wantErr: true,
		},

		{
			name: "unmarshalling error",
			args: args{
				key:           []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				encryptedData: "0000000000000000000000005b57c9c6c4c0f0a38ec6ade46da7bb65d6ccc730f846988b41195a",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := automators.DecryptJobInfo(tt.args.key, tt.args.encryptedData)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecryptJobInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DecryptJobInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptJobInfo(t *testing.T) {
	t.Parallel()
	config := automators.JobConfig{
		CronExpression: "* */1 * * *",
		UID:            uuid.MustParse("a8c14eb0-0fba-4b75-a461-e4d380317ab7"),
		Task: automators.Task{
			URL:     "http://test.org/ping",
			Timeout: time.Minute,
			AuthHeader: automators.AuthHeader{
				Scheme:     "Bearer",
				Parameters: "test-token",
			},
		},
	}
	jobInfo, err := json.Marshal(config)

	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		key     []byte
		jobInfo string
	}
	tests := []struct {
		name    string
		args    args
		want    automators.JobConfig
		wantErr bool
	}{

		{
			name: "sucess encrypt",
			args: args{
				key:     []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				jobInfo: string(jobInfo),
			},
			want:    config,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := automators.EncryptJobInfo(tt.args.key, tt.args.jobInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptJobInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			block, err := aes.NewCipher(tt.args.key)
			if err != nil {
				t.Fatal(err)
			}

			aesGCM, err := cipher.NewGCM(block)
			if err != nil {
				t.Fatal(err)
			}
			encryptedBytes, err := hex.DecodeString(got)
			if err != nil {
				t.Fatal(err)
			}

			nonceSize := aesGCM.NonceSize()
			cipherText, err := aesGCM.Open(nil, encryptedBytes[:nonceSize], encryptedBytes[nonceSize:], nil)
			if err != nil {
				t.Fatal(err)
			}

			var actualConfig automators.JobConfig

			if err := json.Unmarshal(cipherText, &actualConfig); err != nil {
				t.Fatal(fmt.Errorf("error unmarshalling config data: %w", err))
			}

			if !reflect.DeepEqual(actualConfig, tt.want) {
				t.Errorf("EncryptJobInfo() = %v, want %v", actualConfig, tt.want)
			}
		})
	}
}

func TestAutomator_CreateNewJob(t *testing.T) {
	type fields struct {
		cache     automators.Cacher
		secretKey []byte
		scheduler *gocron.Scheduler
		logger    *zap.SugaredLogger
	}
	type args struct {
		cronExpression string
		task           *automators.Task
		config         *automators.JobConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "sucess new job",
			fields: fields{
				cache: &mock.CacherStore{
					Cache:    make(map[string]string),
					CacheSet: make(map[string]struct{}),
					SetName:  "jobs_set",
				},
				secretKey: []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				scheduler: gocron.NewScheduler(time.Local),
				logger:    zap.NewExample().Sugar(),
			},
			args: args{
				config: &automators.JobConfig{
					CronExpression: "* * * * * *",
					Task: automators.Task{
						URL:     "http:/127.0.0.1/ping",
						Timeout: time.Minute,
					},
				},
			},
			wantErr: false,
		},

		{
			name: "invalid cron expression",
			fields: fields{
				cache: &mock.CacherStore{
					Cache: make(map[string]string),
				},
				secretKey: []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				scheduler: gocron.NewScheduler(time.Local),
				logger:    zap.NewExample().Sugar(),
			},
			args: args{
				config: &automators.JobConfig{
					CronExpression: "* * * rv *",
					Task: automators.Task{
						URL:     "http:/127.0.0.1/ping",
						Timeout: time.Minute,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cache error",
			fields: fields{
				cache: &mock.CacherStore{
					WantInsertError: true,
				},
				secretKey: []byte("kHQXeA!12mR56<OVDC0G7ZNEi(WiecmZ"),
				scheduler: gocron.NewScheduler(time.Local),
				logger:    zap.NewExample().Sugar(),
			},
			args: args{
				config: &automators.JobConfig{
					CronExpression: "* * * * * *",
					Task: automators.Task{
						URL:     "http:/127.0.0.1/ping",
						Timeout: time.Minute,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.fields.scheduler.StartAsync()
			ctx := context.Background()

			a := automators.NewAutomator(tt.fields.cache, tt.fields.secretKey, tt.fields.scheduler, tt.fields.logger)

			got, err := a.CreateNewJob(ctx, *tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Automator.CreateNewJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.fields.scheduler.Stop()

			if got == "" && !tt.wantErr {
				t.Errorf("Automator.CreateNewJob() = %v", got)
			}

			if tt.wantErr && got != "" {
				t.Errorf("Automator.CreateNewJob() = %v", got)
			}

		})
	}
}

func TestAutomator_DeleteJob(t *testing.T) {
	t.Parallel()
	type fields struct {
		cache     automators.Cacher
		secretKey []byte
		scheduler *gocron.Scheduler
		logger    *zap.SugaredLogger
	}
	type args struct {
		jobUID uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		jobFunc func()
		wantErr bool
	}{

		{
			name: "successful delete",
			fields: fields{
				cache: &mock.CacherStore{
					Cache: map[string]string{
						"a8c14eb0-0fba-4b75-a461-e4d380317ab7": "job-config",
					},
					SetName: "jobs_set",
				},
				scheduler: gocron.NewScheduler(time.Local),
				logger:    zap.NewExample().Sugar(),
			},
			args: args{
				jobUID: uuid.MustParse("a8c14eb0-0fba-4b75-a461-e4d380317ab7"),
			},
			jobFunc: func() {
				fmt.Println("success job")
			},
			wantErr: false,
		},
		{
			name: "cache error",
			fields: fields{
				cache: &mock.CacherStore{
					WantDeleteError: true,
				},
				scheduler: gocron.NewScheduler(time.Local),
				logger:    zap.NewExample().Sugar(),
			},
			args: args{
				jobUID: uuid.MustParse("a8c14eb0-0fba-4b75-a461-e4d380317ab7"),
			},
			jobFunc: func() {
				fmt.Println("failed job")
			},
			wantErr: true,
		},
		{
			name: "scheduler error",
			fields: fields{
				cache: &mock.CacherStore{
					WantDeleteError: true,
				},
				scheduler: gocron.NewScheduler(time.Local),
				logger:    zap.NewExample().Sugar(),
			},
			args: args{
				jobUID: uuid.MustParse("a8c14eb0-0fba-4b75-a461-e4d380317ab7"),
			},
			jobFunc: nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fields.scheduler.StartAsync()

			_, err := tt.fields.scheduler.Tag(tt.args.jobUID.String()).Every("1us").Do(tt.jobFunc)
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()

			a := automators.NewAutomator(tt.fields.cache, tt.fields.secretKey, tt.fields.scheduler, tt.fields.logger)

			if err := a.DeleteJob(ctx, tt.args.jobUID); (err != nil) != tt.wantErr {
				t.Errorf("Automator.DeleteJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
