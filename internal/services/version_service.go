package services

import (
	"log"
	"strings"
	"sync"
	"time"
)

// VersionService interface defines methods for version management
type VersionService interface {
	GetLatestVersion(app string) (string, error)
	RefreshVersion(app string) error
}

// DefaultVersionService implements VersionService with caching and GitHub integration
type DefaultVersionService struct {
	githubService *GitHubService
	cache         map[string]legacyVersionCacheEntry
	cacheMutex    sync.RWMutex
	cacheTTL      time.Duration
}

type legacyVersionCacheEntry struct {
	version   string
	timestamp time.Time
}

// NewDefaultVersionService creates a new version service with caching
func NewDefaultVersionService() VersionService {
	return &DefaultVersionService{
		githubService: NewGitHubService(),
		cache:         make(map[string]legacyVersionCacheEntry),
		cacheTTL:      10 * time.Minute,
	}
}

// GetLatestVersion retrieves the latest version for an application with caching
func (s *DefaultVersionService) GetLatestVersion(app string) (string, error) {
	s.cacheMutex.RLock()
	if entry, exists := s.cache[app]; exists && time.Since(entry.timestamp) < s.cacheTTL {
		s.cacheMutex.RUnlock()
		return entry.version, nil
	}
	s.cacheMutex.RUnlock()

	// Double-check locking pattern
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Check again after acquiring write lock
	if entry, exists := s.cache[app]; exists && time.Since(entry.timestamp) < s.cacheTTL {
		return entry.version, nil
	}

	// Fetch new version based on app type
	var version string
	var err error

	switch app {
	case "code-server":
		release, fetchErr := s.githubService.GetCodeServerLatestVersion()
		if fetchErr != nil {
			log.Printf("Warning: Failed to fetch latest code-server version: %v", fetchErr)
			// Return cached version if available, otherwise fallback
			if entry, exists := s.cache[app]; exists {
				return entry.version, nil
			}
			return "4.101.1", fetchErr
		}
		version = strings.TrimPrefix(release.TagName, "v")
	default:
		log.Printf("Warning: Version fetching not supported for app: %s", app)
		return "latest", nil
	}

	// Update cache
	s.cache[app] = legacyVersionCacheEntry{
		version:   version,
		timestamp: time.Now(),
	}

	log.Printf("Updated %s version to %s", app, version)
	return version, err
}

// RefreshVersion forces a refresh of the version cache for a specific app
func (s *DefaultVersionService) RefreshVersion(app string) error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Clear cache entry to force refresh
	delete(s.cache, app)
	
	// Fetch new version (will update cache)
	_, err := s.GetLatestVersion(app)
	return err
}