package cache

import (
	"context"
	"sync"
	"time"
)

// Cache represents a simple in-memory cache with TTL support
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*item
	ttl     time.Duration
	enabled bool
}

type item struct {
	value     interface{}
	expiresAt time.Time
}

// New creates a new cache instance
func New(ttl time.Duration, enabled bool) *Cache {
	c := &Cache{
		items:   make(map[string]*item),
		ttl:     ttl,
		enabled: enabled,
	}
	
	if enabled {
		// Start cleanup goroutine
		go c.cleanup()
	}
	
	return c
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}
	
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(item.expiresAt) {
		// Item has expired, remove it
		delete(c.items, key)
		return nil, false
	}
	
	return item.value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	if !c.enabled {
		return
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = &item{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	if !c.enabled {
		return
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	if !c.enabled {
		return
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items = make(map[string]*item)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	if !c.enabled {
		return 0
	}
	
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.items)
}

// cleanup removes expired items from the cache
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2) // Clean up twice per TTL period
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// CacheableFunc represents a function that can be cached
type CacheableFunc func(ctx context.Context, args ...interface{}) (interface{}, error)

// WithCache wraps a function with caching capability
func (c *Cache) WithCache(key string, fn CacheableFunc) CacheableFunc {
	return func(ctx context.Context, args ...interface{}) (interface{}, error) {
		// Try to get from cache first
		if cached, found := c.Get(key); found {
			return cached, nil
		}
		
		// Execute the function
		result, err := fn(ctx, args...)
		if err != nil {
			return nil, err
		}
		
		// Cache the result
		c.Set(key, result)
		return result, nil
	}
}
