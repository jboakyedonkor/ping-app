package cache_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/jboakyedonkor/ping-app/internal/pkg/cache"
	"go.uber.org/zap"
)

func TestCache_DeleteData(t *testing.T) {
	t.Parallel()

	type args struct {
		key string
	}
	tests := []struct {
		name          string
		args          args
		existingValue string
		wantErr       bool
	}{
		// TODO: Add test cases.
		{
			name: "successful deletion",
			args: args{
				key: "test-id",
			},
			existingValue: "test-value",
			wantErr:       false,
		},
		{
			name: "deletion error",
			args: args{
				key: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			miniRed := miniredis.RunT(t)

			if tt.wantErr {
				miniRed.SetError(fmt.Sprintf("%s error", tt.name))
			}

			if tt.existingValue != "" && !tt.wantErr {
				if err := miniRed.Set(tt.args.key, tt.existingValue); err != nil {
					t.Fatal(err)
				}
			}
			c := cache.NewCache(redis.NewClient(&redis.Options{
				Addr: miniRed.Addr(),
			}), zap.NewExample().Sugar())

			if err := c.DeleteData(ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Cache.DeleteData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCache_InsertData(t *testing.T) {
	t.Parallel()

	type args struct {
		key  string
		data string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "successful insertion",
			args: args{
				key:  "test-key",
				data: "test-value",
			},
			wantErr: false,
		},
		{
			name: "insertion error",
			args: args{
				key:  "test-key",
				data: "test-value",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			miniRed := miniredis.RunT(t)

			if tt.wantErr {
				miniRed.SetError(fmt.Sprintf("%s error", tt.name))
			}

			c := cache.NewCache(redis.NewClient(&redis.Options{
				Addr: miniRed.Addr(),
			}), zap.NewExample().Sugar())

			if err := c.InsertData(ctx, tt.args.key, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Cache.InsertData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCache_GetData(t *testing.T) {
	t.Parallel()

	type args struct {
		key string
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		notFoundErr bool
	}{
		// TODO: Add test cases.
		{
			name: "successful retrieval",
			args: args{
				key: "test-key",
			},
			want:    "test-value",
			wantErr: false,
		},
		{
			name: "not found error",
			args: args{
				key: "test",
			},
			want:        "",
			wantErr:     true,
			notFoundErr: true,
		},
		{
			name: "retrieval error",
			args: args{
				key: "test-key",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			miniRed := miniredis.RunT(t)

			if tt.wantErr && !tt.notFoundErr {
				miniRed.SetError(fmt.Sprintf("%s error", tt.name))
			} else if !tt.notFoundErr {
				if err := miniRed.Set(tt.args.key, tt.want); err != nil {
					t.Fatal(err)
				}
			}

			c := cache.NewCache(redis.NewClient(&redis.Options{
				Addr: miniRed.Addr(),
			}), zap.NewExample().Sugar())

			got, err := c.GetData(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				if _, ok := err.(*cache.NotFoundError); tt.notFoundErr && !ok {
					t.Errorf("Cache.GetData() error = %v, wantErr %v", err, tt.notFoundErr)
					return
				}
				t.Errorf("Cache.GetData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Cache.GetData() = %v, want %v", got, tt.want)
			}
		})
	}
}
