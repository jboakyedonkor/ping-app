package mock

import (
	"context"
	"fmt"
)

type CacherStore struct {
	Cache           map[string]string
	WantInsertError bool
	WantDeleteError bool
}

func (c *CacherStore) InsertData(ctx context.Context, key, data string) error {
	if c.WantInsertError {
		return fmt.Errorf("insert error")
	}
	c.Cache[key] = data
	return nil
}

func (c *CacherStore) GetData(ctx context.Context, key string) (string, error) {
	data, ok := c.Cache[key]
	if !ok {
		return "", fmt.Errorf("data not found")
	}
	return data, nil
}

func (c *CacherStore) DeleteData(ctx context.Context, key string) error {
	if c.WantDeleteError {
		return fmt.Errorf("delete error")
	}

	delete(c.Cache, key)
	return nil
}
