package cache

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Cache struct {
	redisClient *redis.Client
	logger      *zap.SugaredLogger
}

type NotFoundError struct{}

func (c *NotFoundError) Error() string {
	return "data not found in cache"
}

func NewCache(client *redis.Client, logger *zap.SugaredLogger) *Cache {
	return &Cache{
		redisClient: client,
		logger:      logger,
	}
}

func (c *Cache) InsertData(ctx context.Context, key, data string) error {
	_, err := c.redisClient.Set(ctx, key, data, 0).Result()
	if err != nil {
		err := fmt.Errorf("error set data on redis: %w", err)
		c.logger.With("context", ctx).Error(err)
		return err
	}

	return nil
}
func (c *Cache) GetData(ctx context.Context, key string) (string, error) {
	logger := c.logger.With("context", ctx)
	result, err := c.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		e := &NotFoundError{}
		logger.Error(e.Error())
		return "", e
	}

	if err != nil {
		err := fmt.Errorf("error retrieving data from redis: %w", err)
		logger.Error(err.Error())
		return "", err
	}
	logger.Debugw("retrieved data from redis cache", "data", result)
	return result, nil
}
func (c *Cache) DeleteData(ctx context.Context, key string) error {
	_, err := c.redisClient.Del(ctx, key).Result()
	if err != nil {
		err := fmt.Errorf("error deleting data from redis: %w", err)
		c.logger.With("context", ctx).Error(err.Error())
		return err
	}
	c.logger.With("context", ctx).Debugf("successful deleted %q from cache", key)
	return nil
}

func (c *Cache) GetSet(ctx context.Context, key string) (map[string]struct{}, error) {
	set, err := c.redisClient.SMembersMap(ctx, key).Result()
	if err != nil && err != redis.Nil {

		err := fmt.Errorf("error retriving set: %w", err)
		c.logger.Error(err)
		return nil, err
	}

	return set, nil
}

func (c *Cache) DeleteSet(ctx context.Context, key string) error {
	_, err := c.redisClient.Unlink(ctx, key).Result()
	if err != nil && err != redis.Nil {

		err := fmt.Errorf("error deleting set: %w", err)
		c.logger.Error(err)
		return err
	}

	return nil

}

func (c *Cache) DeleteFromSet(ctx context.Context, setName string, keys ...string) error {

	_, err := c.redisClient.SRem(ctx, setName, keys).Result()
	if err != nil && err != redis.Nil {

		err := fmt.Errorf("error removing elements from set: %w", err)
		c.logger.Error(err)
		return err
	}
	return nil
}

func (c *Cache) UpdateSet(ctx context.Context, setName string, keys ...string) error {

	if _, err := c.redisClient.SAdd(ctx, setName, keys).Result(); err != nil {
		err := fmt.Errorf("error updating set: %w", err)
		return err
	}
	return nil
}
