package cache

import (
	"container/list"
	"sync"
	"time"

	"github.com/tentens-tech/shared-lock/internal/infrastructure/metrics"
)

type Cache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	maxSize  int
	lruList  *list.List
	keyToLRU map[string]*list.Element
}

type CacheItem struct {
	Value      interface{}
	Expiration time.Time
	key        string
}

type lruItem struct {
	key string
}

func New(cacheSize int) *Cache {
	cache := &Cache{
		items:    make(map[string]*CacheItem),
		maxSize:  cacheSize,
		lruList:  list.New(),
		keyToLRU: make(map[string]*list.Element),
	}

	go cache.cleanup()

	return cache
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		item.Value = value
		item.Expiration = time.Now().Add(ttl)
		if elem, ok := c.keyToLRU[key]; ok {
			c.lruList.MoveToFront(elem)
		}
		metrics.CacheOperations.WithLabelValues("set", "update").Inc()
		return
	}

	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &CacheItem{
		Value:      value,
		Expiration: time.Now().Add(ttl),
		key:        key,
	}

	elem := c.lruList.PushFront(&lruItem{key: key})
	c.keyToLRU[key] = elem

	metrics.CacheOperations.WithLabelValues("set", "success").Inc()
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		metrics.CacheOperations.WithLabelValues("get", "miss").Inc()
		return nil, false
	}

	if time.Now().After(item.Expiration) {
		c.removeItem(key)
		metrics.CacheOperations.WithLabelValues("get", "expired").Inc()
		return nil, false
	}

	if elem, ok := c.keyToLRU[key]; ok {
		c.lruList.MoveToFront(elem)
	}

	metrics.CacheOperations.WithLabelValues("get", "hit").Inc()
	return item.Value, true
}

func (c *Cache) evictOldest() {
	if elem := c.lruList.Back(); elem != nil {
		if lruItem, ok := elem.Value.(*lruItem); ok {
			c.removeItem(lruItem.key)
			metrics.CacheOperations.WithLabelValues("evict", "size_limit").Inc()
		}
	}
}

func (c *Cache) removeItem(key string) {
	delete(c.items, key)
	if elem, ok := c.keyToLRU[key]; ok {
		c.lruList.Remove(elem)
		delete(c.keyToLRU, key)
	}
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.Expiration) {
				c.removeItem(key)
				metrics.CacheOperations.WithLabelValues("cleanup", "expired").Inc()
			}
		}
		c.mu.Unlock()
	}
}

func (c *Cache) SetMaxSize(cacheSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxSize = cacheSize

	for len(c.items) > c.maxSize {
		c.evictOldest()
	}
}
