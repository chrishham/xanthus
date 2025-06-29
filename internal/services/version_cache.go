package services

import (
	"sync"
	"time"
)

// VersionCache interface defines methods for caching version information
type VersionCache interface {
	Get(key string) (string, bool)
	Set(key string, value string, ttl time.Duration)
	Invalidate(key string)
	Clear()
	GetStats() CacheStats
}

// CacheStats provides statistics about cache usage
type CacheStats struct {
	Hits        int64
	Misses      int64
	Entries     int
	LastCleanup time.Time
}

// InMemoryVersionCache implements VersionCache with in-memory storage
type InMemoryVersionCache struct {
	entries     map[string]versionCacheEntry
	mutex       sync.RWMutex
	stats       CacheStats
	cleanupTTL  time.Duration
	lastCleanup time.Time
}

type versionCacheEntry struct {
	value     string
	expiresAt time.Time
}

// NewInMemoryVersionCache creates a new in-memory version cache
func NewInMemoryVersionCache(cleanupInterval time.Duration) VersionCache {
	cache := &InMemoryVersionCache{
		entries:     make(map[string]versionCacheEntry),
		cleanupTTL:  cleanupInterval,
		lastCleanup: time.Now(),
	}

	// Start cleanup goroutine
	go cache.startCleanupWorker()

	return cache
}

// Get retrieves a value from the cache
func (c *InMemoryVersionCache) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return "", false
	}

	// Check if entry has expired
	if time.Now().After(entry.expiresAt) {
		c.stats.Misses++
		// Don't delete here to avoid write lock, cleanup worker will handle it
		return "", false
	}

	c.stats.Hits++
	return entry.value, true
}

// Set stores a value in the cache with TTL
func (c *InMemoryVersionCache) Set(key string, value string, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries[key] = versionCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Invalidate removes a specific key from the cache
func (c *InMemoryVersionCache) Invalidate(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *InMemoryVersionCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]versionCacheEntry)
	c.stats.Hits = 0
	c.stats.Misses = 0
}

// GetStats returns cache statistics
func (c *InMemoryVersionCache) GetStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := c.stats
	stats.Entries = len(c.entries)
	stats.LastCleanup = c.lastCleanup
	return stats
}

// startCleanupWorker runs a background goroutine to clean up expired entries
func (c *InMemoryVersionCache) startCleanupWorker() {
	ticker := time.NewTicker(c.cleanupTTL)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpiredEntries()
	}
}

// cleanupExpiredEntries removes expired entries from the cache
func (c *InMemoryVersionCache) cleanupExpiredEntries() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}

	c.lastCleanup = now
}

// DefaultVersionCacheConfig provides default cache configuration
type DefaultVersionCacheConfig struct {
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
}

// NewDefaultVersionCacheConfig creates default cache configuration
func NewDefaultVersionCacheConfig() DefaultVersionCacheConfig {
	return DefaultVersionCacheConfig{
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 30 * time.Minute,
	}
}
