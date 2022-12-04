package mock

import (
	"context"
	"fmt"
)

type CacherStore struct {
	Cache           map[string]string
	CacheSet        map[string]struct{}
	SetName         string
	WantInsertError bool
	WantDeleteError bool
	WantGetError    bool
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

func (c *CacherStore) GetSet(ctx context.Context, key string) (map[string]struct{}, error) {
	if key != c.SetName {
		return nil, fmt.Errorf("set not found")
	}

	if c.CacheSet != nil {
		return nil, fmt.Errorf("nil cache set")
	}

	if c.WantGetError {
		return nil, fmt.Errorf("get error")
	}

	return c.CacheSet, nil

}

func (c *CacherStore) DeleteSet(ctx context.Context, key string) error {

	if key != c.SetName {
		return fmt.Errorf("error deleting set")
	}

	c.CacheSet = nil
	return nil

}

func (c *CacherStore) DeleteFromSet(ctx context.Context, setName string, keys ...string) error {
	if setName != c.SetName {
		return fmt.Errorf("setName not found")
	}

	if c.WantDeleteError {
		return fmt.Errorf("delete error")
	}

	for _, k := range keys {
		delete(c.CacheSet, k)
	}
	return nil
}

func (c *CacherStore) UpdateSet(ctx context.Context, setName string, keys ...string) error {

	if setName != c.SetName {
		return fmt.Errorf("setName not found")
	}

	if c.WantDeleteError {
		return fmt.Errorf("delete error")
	}

	for _, k := range keys {
		c.CacheSet[k] = struct{}{}
	}
	return nil
}
