package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/jboakyedonkor/ping-app/internal/pkg/automators"
	"github.com/jboakyedonkor/ping-app/internal/pkg/cache"
	"github.com/jboakyedonkor/ping-app/internal/pkg/routes"
)

type envConfig struct {
	loggingMode string
	redisHost   string
	redisPort   string
	secretKey   string
	appPort     string
}

func main() {

	config := getEnvConfig()
	logger := getLogger(config)
	defer logger.Sync()

	scheduler := gocron.NewScheduler(time.UTC)

	redisCache := cache.NewCache(getRedisClient(config), logger)
	automator := automators.NewAutomator(redisCache, []byte(config.secretKey), scheduler, logger)

	jobRoute := routes.NewJobRoute(logger, automator)
	app := gin.Default()

	app.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "up",
		})
	})

	jobGroup := app.Group("/jobs")
	jobGroup.DELETE("/:id", jobRoute.DeleteJob)
	jobGroup.GET("/:id/config", jobRoute.GetJobConfig)
	jobGroup.GET("", jobRoute.GetJobs)
	jobGroup.POST("", jobRoute.CreateJob)

	scheduler.StartAsync()
	port := config.appPort
	if port == "" {
		port = "8080"
	}
	go automator.ReconcileJobs()

	logger.Infof("listening on port %s", port)
	app.Run(fmt.Sprintf(":%s", port))
}

func getLogger(config envConfig) *zap.SugaredLogger {

	if config.loggingMode == "" || config.loggingMode != "PROD" && config.loggingMode != "prod" {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		return logger.Sugar()
	}
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	return logger.Sugar()

}

func getRedisClient(config envConfig) *redis.Client {

	if config.redisHost == "" {
		panic("no redis host defined")
	}

	if config.redisPort == "" {
		panic("no redis port defined")
	}

	addr := fmt.Sprintf("%s:%s", config.redisHost, config.redisPort)

	redisClient := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		panic(err)
	}

	return redisClient
}

func getEnvConfig() envConfig {
	return envConfig{
		loggingMode: os.Getenv("MODE"),
		redisHost:   os.Getenv("REDIS_HOST"),
		redisPort:   os.Getenv("REDIS_PORT"),
		secretKey:   os.Getenv("SECRET_KEY"),
	}
}
