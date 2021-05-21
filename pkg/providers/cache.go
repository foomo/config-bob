package providers

import (
	"fmt"
	"sync"
)

type Cache struct {
	lock sync.RWMutex
	data map[string]string
}

func NewCache() *Cache {
	return &Cache{
		lock: sync.RWMutex{},
		data: map[string]string{},
	}
}

func (c *Cache) Set(provider, path, value string) {
	c.lock.Lock()
	c.data[getCacheKey(provider, path)] = value
	c.lock.Unlock()
}

func (c *Cache) Get(provider, path string) (value string, ok bool) {
	c.lock.RLock()
	value, ok = c.data[getCacheKey(provider, path)]
	c.lock.RUnlock()
	return value, ok
}

func getCacheKey(provider, path string) string {
	return fmt.Sprintf("%s#%s", provider, path)
}
