package cache

import (
	"fmt"
	"time"
)

// Storage defines the interface for cache implementations.
type Storage interface {
	Set(key string, value []byte, ttl time.Duration)
	SetWithHeaders(key string, value []byte, headers map[string]string, statusCode int, ttl time.Duration)
	Get(key string) []byte
	GetItem(key string) *CacheItem
	Delete(key string)
	Clear()
	Stats() map[string]interface{}
	Stop()
}

// FactoryConfig represents cache factory configuration.
type FactoryConfig struct {
	StorageType     string
	CacheDir        string
	MaxSize         string
	DefaultTTL      int
	CleanupInterval int
	Persistent      bool // If true, cache never expires
}

// NewCacheStorage creates a new cache storage based on configuration.
func NewCacheStorage(config FactoryConfig) (Storage, error) {
	switch config.StorageType {
	case "file", "persistent":
		// File-based persistent cache
		return NewFileCache(config.CacheDir, config.MaxSize, config.DefaultTTL, config.Persistent)
	case "memory", "":
		// Memory-based cache (default)
		return NewCache(config.MaxSize, config.DefaultTTL, config.CleanupInterval), nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.StorageType)
	}
}
