// Package cache provides in-memory and file-based caching implementations.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// generateHash generates a SHA256 hash for a string (shared utility)
func generateHash(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CacheItem represents a cached item with metadata.
type CacheItem struct {
	Key        string
	Value      []byte
	Headers    map[string]string
	StatusCode int
	ExpiresAt  time.Time
	Size       int
}

// Cache implements an in-memory caching system with automatic cleanup.
type Cache struct {
	items           map[string]*CacheItem
	mutex           sync.RWMutex
	maxSize         int64
	currentSize     int64
	ttl             time.Duration
	cleanupInterval time.Duration
	stopChan        chan bool
}

// NewCache creates a new in-memory cache instance.
func NewCache(maxSize string, defaultTTL int, cleanupInterval int) *Cache {
	sizeBytes, err := parseSize(maxSize)
	if err != nil {
		sizeBytes = 100 * 1024 * 1024 // Default 100MB
	}

	// Set defaults for zero values
	if defaultTTL <= 0 {
		defaultTTL = 300 // Default 5 minutes
	}
	if cleanupInterval <= 0 {
		cleanupInterval = 600 // Default 10 minutes
	}

	cache := &Cache{
		items:           make(map[string]*CacheItem),
		maxSize:         sizeBytes,
		currentSize:     0,
		ttl:             time.Duration(defaultTTL) * time.Second,
		cleanupInterval: time.Duration(cleanupInterval) * time.Second,
		stopChan:        make(chan bool),
	}

	// Start cleanup goroutine
	go cache.startCleanup()

	return cache
}

func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	var multiplier int64 = 1
	var numStr string

	if len(sizeStr) > 2 {
		suffix := sizeStr[len(sizeStr)-2:]
		switch suffix {
		case "KB", "kb":
			multiplier = 1024
			numStr = sizeStr[:len(sizeStr)-2]
		case "MB", "mb":
			multiplier = 1024 * 1024
			numStr = sizeStr[:len(sizeStr)-2]
		case "GB", "gb":
			multiplier = 1024 * 1024 * 1024
			numStr = sizeStr[:len(sizeStr)-2]
		default:
			numStr = sizeStr
		}
	} else {
		numStr = sizeStr
	}

	var size int64
	_, err := fmt.Sscanf(numStr, "%d", &size)
	if err != nil {
		return 0, err
	}

	return size * multiplier, nil
}

func (c *Cache) generateKey(key string) string {
	return generateHash(key)
}

// Set stores data in the cache with a specified TTL.
func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	c.SetWithHeaders(key, value, nil, 200, ttl)
}

// SetWithHeaders stores data with HTTP headers and status code in the cache.
func (c *Cache) SetWithHeaders(key string, value []byte, headers map[string]string, statusCode int, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hashKey := c.generateKey(key)

	// Remove existing item if it exists
	if item, exists := c.items[hashKey]; exists {
		c.currentSize -= int64(item.Size)
		delete(c.items, hashKey)
	}

	// Check if we need to evict items
	for c.currentSize+int64(len(value)) > c.maxSize && len(c.items) > 0 {
		c.evictLRU()
	}

	// Add new item
	expiresAt := time.Now().Add(ttl)
	if ttl == 0 {
		expiresAt = time.Now().Add(c.ttl)
	}

	item := &CacheItem{
		Key:        key,
		Value:      make([]byte, len(value)),
		Headers:    headers,
		StatusCode: statusCode,
		ExpiresAt:  expiresAt,
		Size:       len(value),
	}
	copy(item.Value, value)

	c.items[hashKey] = item
	c.currentSize += int64(len(value))
}

// Get retrieves cached data by key, returning nil if not found or expired.
func (c *Cache) Get(key string) []byte {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	hashKey := c.generateKey(key)

	if item, exists := c.items[hashKey]; exists {
		if time.Now().Before(item.ExpiresAt) {
			return item.Value
		}
	}

	return nil
}

// GetItem retrieves a complete cache item with metadata by key.
func (c *Cache) GetItem(key string) *CacheItem {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hashKey := c.generateKey(key)

	if item, exists := c.items[hashKey]; exists {
		if time.Now().Before(item.ExpiresAt) {
			return item
		}
		// Item expired, remove it
		delete(c.items, hashKey)
		c.currentSize -= int64(item.Size)
	}

	return nil
}

// Delete removes an item from the cache by key.
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	hashKey := c.generateKey(key)

	if item, exists := c.items[hashKey]; exists {
		delete(c.items, hashKey)
		c.currentSize -= int64(item.Size)
	}
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*CacheItem)
	c.currentSize = 0
}

func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.ExpiresAt
		}
	}

	if oldestKey != "" {
		if item, exists := c.items[oldestKey]; exists {
			c.currentSize -= int64(item.Size)
		}
		delete(c.items, oldestKey)
	}
}

func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

func (c *Cache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
			c.currentSize -= int64(item.Size)
		}
	}
}

// Stats returns current cache statistics.
func (c *Cache) Stats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"items_count":   len(c.items),
		"current_size":  c.currentSize,
		"max_size":      c.maxSize,
		"usage_percent": float64(c.currentSize) / float64(c.maxSize) * 100,
	}
}

// Stop stops the cache cleanup goroutine.
func (c *Cache) Stop() {
	close(c.stopChan)
}
