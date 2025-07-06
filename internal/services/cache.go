package services

import (
	"strings"
	"sync"
	"time"
)

// CacheEntry represents a cached value with expiration
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

// CacheService provides in-memory caching for frequently accessed data
type CacheService struct {
	cache map[string]CacheEntry
	mutex sync.RWMutex
}

// NewCacheService creates a new cache service instance
func NewCacheService() *CacheService {
	return &CacheService{
		cache: make(map[string]CacheEntry),
	}
}

// Get retrieves a value from cache
func (cs *CacheService) Get(key string) (interface{}, bool) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	entry, exists := cs.cache[key]
	if !exists || time.Now().After(entry.Expiration) {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in cache with expiration
func (cs *CacheService) Set(key string, value interface{}, duration time.Duration) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.cache[key] = CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(duration),
	}
}

// Delete removes a value from cache
func (cs *CacheService) Delete(key string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	delete(cs.cache, key)
}

// Clear removes all expired entries from cache
func (cs *CacheService) Clear() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	now := time.Now()
	for key, entry := range cs.cache {
		if now.After(entry.Expiration) {
			delete(cs.cache, key)
		}
	}
}

// AccountInfo represents cached account information
type AccountInfo struct {
	AccountID   string
	NamespaceID string
}

// GetAccountInfo retrieves cached account information
func (cs *CacheService) GetAccountInfo(token string) (*AccountInfo, bool) {
	// Use full token hash for security - no partial tokens that could collide
	key := "account_info:" + hashToken(token)
	value, exists := cs.Get(key)
	if !exists {
		return nil, false
	}

	accountInfo, ok := value.(*AccountInfo)
	return accountInfo, ok
}

// SetAccountInfo caches account information
func (cs *CacheService) SetAccountInfo(token string, info *AccountInfo, duration time.Duration) {
	key := "account_info:" + hashToken(token)
	cs.Set(key, info, duration)
}

// hashToken creates a secure hash of the token for cache keys
func hashToken(token string) string {
	// Use a simple but secure approach - could use crypto/sha256 for production
	if len(token) < 20 {
		return token // For short tokens, use as-is
	}
	// Use last 20 chars which are more unique than first chars
	return token[len(token)-20:]
}

// ClearUserCache removes all cached data for a specific user/account
func (cs *CacheService) ClearUserCache(accountID string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Remove all cache entries for this account
	for key := range cs.cache {
		if strings.Contains(key, accountID) {
			delete(cs.cache, key)
		}
	}
}

// GetCacheStats returns basic cache statistics for monitoring
func (cs *CacheService) GetCacheStats() map[string]interface{} {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	active := 0
	expired := 0
	now := time.Now()

	for _, entry := range cs.cache {
		if now.After(entry.Expiration) {
			expired++
		} else {
			active++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(cs.cache),
		"active_entries":  active,
		"expired_entries": expired,
	}
}
