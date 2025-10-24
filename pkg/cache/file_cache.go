package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileCacheItem represents a persistent cache item
type FileCacheItem struct {
	Key        string            `json:"key"`
	Headers    map[string]string `json:"headers"`
	StatusCode int               `json:"status_code"`
	CreatedAt  time.Time         `json:"created_at"`
	ExpiresAt  time.Time         `json:"expires_at"` // For compatibility, but will use zero value for never expire
	Size       int               `json:"size"`
	DataFile   string            `json:"data_file"` // Path to the data file
}

// FileCache implements persistent file-based caching
type FileCache struct {
	cacheDir    string
	items       map[string]*FileCacheItem
	mutex       sync.RWMutex
	maxSize     int64
	currentSize int64
	ttl         time.Duration
	persistent  bool // If true, cache never expires
}

// NewFileCache creates a new persistent file cache
func NewFileCache(cacheDir string, maxSize string, defaultTTL int, persistent bool) (*FileCache, error) {
	sizeBytes, err := parseSize(maxSize)
	if err != nil {
		sizeBytes = 500 * 1024 * 1024 // Default 500MB
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Create data subdirectory for storing actual cache data
	dataDir := filepath.Join(cacheDir, "data")
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	cache := &FileCache{
		cacheDir:    cacheDir,
		items:       make(map[string]*FileCacheItem),
		maxSize:     sizeBytes,
		currentSize: 0,
		ttl:         time.Duration(defaultTTL) * time.Second,
		persistent:  persistent,
	}

	// Load existing cache from disk
	if err := cache.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %v", err)
	}

	return cache, nil
}

// loadFromDisk loads cache metadata from disk
func (fc *FileCache) loadFromDisk() error {
	indexFile := filepath.Join(fc.cacheDir, "index.json")

	// If index file doesn't exist, start with empty cache
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(indexFile)
	if err != nil {
		return err
	}

	var items map[string]*FileCacheItem
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	now := time.Now()
	for key, item := range items {
		// Check if data file exists
		dataFile := filepath.Join(fc.cacheDir, "data", item.DataFile)
		if _, err := os.Stat(dataFile); os.IsNotExist(err) {
			continue // Skip items with missing data files
		}

		// If not persistent mode, check expiration
		if !fc.persistent && !item.ExpiresAt.IsZero() && now.After(item.ExpiresAt) {
			// Remove expired item
			_ = os.Remove(dataFile) //nolint:errcheck
			continue
		}

		fc.items[key] = item
		fc.currentSize += int64(item.Size)
	}

	return nil
}

// saveIndex saves cache metadata to disk
func (fc *FileCache) saveIndex() error {
	indexFile := filepath.Join(fc.cacheDir, "index.json")

	data, err := json.MarshalIndent(fc.items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, data, 0600)
}

// generateKey generates a hash key for the cache
func (fc *FileCache) generateKey(key string) string {
	return generateHash(key)
}

// SetWithHeaders stores data with headers in persistent cache
func (fc *FileCache) SetWithHeaders(key string, value []byte, headers map[string]string, statusCode int, ttl time.Duration) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	hashKey := fc.generateKey(key)

	// Remove existing item if it exists
	if item, exists := fc.items[hashKey]; exists {
		fc.currentSize -= int64(item.Size)
		// Remove old data file
		oldDataFile := filepath.Join(fc.cacheDir, "data", item.DataFile)
		_ = os.Remove(oldDataFile) //nolint:errcheck
		delete(fc.items, hashKey)
	}

	// Check if we need to evict items
	for fc.currentSize+int64(len(value)) > fc.maxSize && len(fc.items) > 0 {
		fc.evictOldest()
	}

	// Write data to file
	dataFileName := fmt.Sprintf("%s.bin", hashKey)
	dataFilePath := filepath.Join(fc.cacheDir, "data", dataFileName)

	if err := os.WriteFile(dataFilePath, value, 0600); err != nil {
		// Failed to write, skip this cache item
		return
	}

	// Create cache item
	var expiresAt time.Time
	if fc.persistent {
		// Use zero value to indicate never expires
		expiresAt = time.Time{}
	} else {
		if ttl == 0 {
			ttl = fc.ttl
		}
		expiresAt = time.Now().Add(ttl)
	}

	item := &FileCacheItem{
		Key:        key,
		Headers:    headers,
		StatusCode: statusCode,
		CreatedAt:  time.Now(),
		ExpiresAt:  expiresAt,
		Size:       len(value),
		DataFile:   dataFileName,
	}

	fc.items[hashKey] = item
	fc.currentSize += int64(len(value))

	// Save index
	_ = fc.saveIndex() //nolint:errcheck
}

// Set stores data in persistent cache (legacy method)
func (fc *FileCache) Set(key string, value []byte, ttl time.Duration) {
	fc.SetWithHeaders(key, value, nil, 200, ttl)
}

// GetItem retrieves a cache item with full metadata
func (fc *FileCache) GetItem(key string) *CacheItem {
	fc.mutex.RLock()
	hashKey := fc.generateKey(key)
	item, exists := fc.items[hashKey]
	fc.mutex.RUnlock()

	if !exists {
		return nil
	}

	// Check expiration (only if not persistent mode)
	if !fc.persistent && !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt) {
		// Item expired
		fc.Delete(key)
		return nil
	}

	// Read data from file
	dataFilePath := filepath.Join(fc.cacheDir, "data", item.DataFile)
	data, err := os.ReadFile(dataFilePath)
	if err != nil {
		// File not found or error, remove from index
		fc.Delete(key)
		return nil
	}

	return &CacheItem{
		Key:        item.Key,
		Value:      data,
		Headers:    item.Headers,
		StatusCode: item.StatusCode,
		ExpiresAt:  item.ExpiresAt,
		Size:       item.Size,
	}
}

// Get retrieves cached data (legacy method)
func (fc *FileCache) Get(key string) []byte {
	item := fc.GetItem(key)
	if item != nil {
		return item.Value
	}
	return nil
}

// Delete removes an item from cache
func (fc *FileCache) Delete(key string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	hashKey := fc.generateKey(key)

	if item, exists := fc.items[hashKey]; exists {
		// Remove data file
		dataFilePath := filepath.Join(fc.cacheDir, "data", item.DataFile)
		_ = os.Remove(dataFilePath) //nolint:errcheck

		fc.currentSize -= int64(item.Size)
		delete(fc.items, hashKey)

		// Save index
		_ = fc.saveIndex() //nolint:errcheck
	}
}

// Clear removes all items from cache
func (fc *FileCache) Clear() {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Remove all data files
	dataDir := filepath.Join(fc.cacheDir, "data")
	if files, err := os.ReadDir(dataDir); err == nil {
		for _, file := range files {
			_ = os.Remove(filepath.Join(dataDir, file.Name())) //nolint:errcheck
		}
	}

	fc.items = make(map[string]*FileCacheItem)
	fc.currentSize = 0

	// Save index
	_ = fc.saveIndex() //nolint:errcheck
}

// evictOldest removes the oldest cache item
func (fc *FileCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range fc.items {
		if oldestKey == "" || item.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.CreatedAt
		}
	}

	if oldestKey != "" {
		if item, exists := fc.items[oldestKey]; exists {
			// Remove data file
			dataFilePath := filepath.Join(fc.cacheDir, "data", item.DataFile)
			_ = os.Remove(dataFilePath) //nolint:errcheck

			fc.currentSize -= int64(item.Size)
		}
		delete(fc.items, oldestKey)
	}
}

// Stats returns cache statistics
func (fc *FileCache) Stats() map[string]interface{} {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	return map[string]interface{}{
		"items_count":   len(fc.items),
		"current_size":  fc.currentSize,
		"max_size":      fc.maxSize,
		"usage_percent": float64(fc.currentSize) / float64(fc.maxSize) * 100,
		"storage_type":  "file",
		"persistent":    fc.persistent,
		"cache_dir":     fc.cacheDir,
	}
}

// Stop performs cleanup (for file cache, just ensure index is saved)
func (fc *FileCache) Stop() {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	_ = fc.saveIndex() //nolint:errcheck
}
