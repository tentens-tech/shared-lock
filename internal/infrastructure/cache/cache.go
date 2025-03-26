package cache

import (
	"sync"
	"time"

	"github.com/tentens-tech/shared-lock/internal/infrastructure/metrics"
)

type Cache struct {
	mu    sync.RWMutex
	items map[string]*CacheItem
}

type CacheItem struct {
	Value      interface{}
	Expiration time.Time
}

func New() *Cache {
	cache := &Cache{
		items: make(map[string]*CacheItem),
	}

	go cache.cleanup()

	return cache
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}

	metrics.CacheOperations.WithLabelValues("set", "success").Inc()
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		metrics.CacheOperations.WithLabelValues("get", "miss").Inc()
		return nil, false
	}

	if time.Now().After(item.Expiration) {
		delete(c.items, key)
		metrics.CacheOperations.WithLabelValues("get", "expired").Inc()
		return nil, false
	}

	metrics.CacheOperations.WithLabelValues("get", "hit").Inc()
	return item.Value, true
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
