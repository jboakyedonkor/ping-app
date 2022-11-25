package mock

import "fmt"

type CacherStore struct {
	Cache           map[string]string
	WantInsertError bool
	WantDeleteError bool
}

func (c *CacherStore) InsertData(key, data string) error {
	if c.WantInsertError {
		return fmt.Errorf("insert error")
	}
	c.Cache[key] = data
	return nil
}

func (c *CacherStore) GetData(key string) (string, error) {
	data, ok := c.Cache[key]
	if !ok {
		return "", fmt.Errorf("data not found")
	}
	return data, nil
}

func (c *CacherStore) DeleteData(key string) error {
	if c.WantDeleteError {
		return fmt.Errorf("delete error")
	}

	delete(c.Cache, key)
	return nil
}
