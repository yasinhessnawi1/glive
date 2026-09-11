package cache

import (
	"container/list"
	"sync"
	"time"

	"github.com/glive/core/types"
)

// LRUCache is a thread-safe LRU cache with TTL
type LRUCache struct {
	capacity int
	ttl      time.Duration
	mu       sync.RWMutex
	items    map[string]*list.Element
	order    *list.List
}

type cacheItem struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// NewLRUCache creates a new LRU cache with the specified capacity and TTL
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	c := &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}

	// Start cleanup goroutine
	go c.cleanup()

	return c
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	elem, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item := elem.Value.(*cacheItem)
	if time.Now().After(item.expiresAt) {
		c.removeElement(elem)
		return nil, false
	}

	c.order.MoveToFront(elem)
	return item.value, true
}

// Set stores a value in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		c.order.MoveToFront(elem)
		item := elem.Value.(*cacheItem)
		item.value = value
		item.expiresAt = time.Now().Add(c.ttl)
		return
	}

	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.removeElement(oldest)
		}
	}

	item := &cacheItem{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.order.PushFront(item)
	c.items[key] = elem
}

// Delete removes a value from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		c.removeElement(elem)
	}
}

// removeElement removes an element from both the list and map
func (c *LRUCache) removeElement(elem *list.Element) {
	c.order.Remove(elem)
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
}

// cleanup periodically removes expired entries
func (c *LRUCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		var toRemove []*list.Element
		for _, elem := range c.items {
			item := elem.Value.(*cacheItem)
			if now.After(item.expiresAt) {
				toRemove = append(toRemove, elem)
			}
		}
		for _, elem := range toRemove {
			c.removeElement(elem)
		}
		c.mu.Unlock()
	}
}

// AnalysisCache caches project analysis results
type AnalysisCache struct {
	cache *LRUCache
}

// NewAnalysisCache creates a new analysis cache
func NewAnalysisCache() *AnalysisCache {
	return &AnalysisCache{
		cache: NewLRUCache(100, 24*time.Hour),
	}
}

// GetAnalysis retrieves an analysis result from the cache
func (c *AnalysisCache) GetAnalysis(repoURL string, commitHash string) (*types.AnalysisResult, bool) {
	key := repoURL + "@" + commitHash
	if val, ok := c.cache.Get(key); ok {
		return val.(*types.AnalysisResult), true
	}
	return nil, false
}

// SetAnalysis stores an analysis result in the cache
func (c *AnalysisCache) SetAnalysis(repoURL string, commitHash string, result *types.AnalysisResult) {
	key := repoURL + "@" + commitHash
	c.cache.Set(key, result)
}
