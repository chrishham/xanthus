package services

import (
	"fmt"
	"log"
	"sync"

	"github.com/chrishham/xanthus/internal/models"
)

// EnhancedVersionService interface extends VersionService with additional capabilities
type EnhancedVersionService interface {
	VersionService
	RefreshVersionFromSource(appID string) (string, error)
	GetVersionHistory(appID string) ([]string, error)
	GetCacheStats() CacheStats
	ClearCache()
	SetVersionSource(appID string, source VersionSource)
	GetVersionSource(appID string) VersionSource
}

// EnhancedDefaultVersionService implements EnhancedVersionService with pluggable sources
type EnhancedDefaultVersionService struct {
	cache          VersionCache
	sources        map[string]VersionSource
	sourcesMutex   sync.RWMutex
	factory        *VersionSourceFactory
	config         DefaultVersionCacheConfig
	configLoader   models.ConfigLoader
}

// NewEnhancedDefaultVersionService creates a new enhanced version service
func NewEnhancedDefaultVersionService(configLoader models.ConfigLoader) EnhancedVersionService {
	config := NewDefaultVersionCacheConfig()
	cache := NewInMemoryVersionCache(config.CleanupInterval)
	factory := NewVersionSourceFactory()
	
	service := &EnhancedDefaultVersionService{
		cache:        cache,
		sources:      make(map[string]VersionSource),
		factory:      factory,
		config:       config,
		configLoader: configLoader,
	}
	
	// Initialize default sources for existing applications
	service.initializeDefaultSources()
	
	return service
}

// initializeDefaultSources sets up version sources for existing hardcoded applications
func (s *EnhancedDefaultVersionService) initializeDefaultSources() {
	s.sourcesMutex.Lock()
	defer s.sourcesMutex.Unlock()
	
	// Add source for code-server (existing GitHub integration)
	s.sources["code-server"] = NewGitHubVersionSource("coder/code-server")
	
	log.Println("Initialized default version sources")
}

// GetLatestVersion retrieves the latest version for an application with caching
func (s *EnhancedDefaultVersionService) GetLatestVersion(appID string) (string, error) {
	// Check cache first
	if version, found := s.cache.Get(appID); found {
		return version, nil
	}
	
	// Get version from source
	return s.RefreshVersionFromSource(appID)
}

// RefreshVersionFromSource forces a refresh from the version source
func (s *EnhancedDefaultVersionService) RefreshVersionFromSource(appID string) (string, error) {
	s.sourcesMutex.RLock()
	source, exists := s.sources[appID]
	s.sourcesMutex.RUnlock()
	
	if !exists {
		// Try to create source from configuration
		if err := s.loadVersionSourceFromConfig(appID); err != nil {
			log.Printf("Warning: No version source configured for %s, using 'latest': %v", appID, err)
			version := "latest"
			s.cache.Set(appID, version, s.config.DefaultTTL)
			return version, nil
		}
		
		// Retry after loading from config
		s.sourcesMutex.RLock()
		source, exists = s.sources[appID]
		s.sourcesMutex.RUnlock()
		
		if !exists {
			return "latest", fmt.Errorf("no version source available for %s", appID)
		}
	}
	
	log.Printf("Fetching latest version for %s from %s source", appID, source.GetSourceType())
	
	version, err := source.GetLatestVersion()
	if err != nil {
		log.Printf("Warning: Failed to fetch latest version for %s: %v", appID, err)
		
		// Try to return cached version if available
		if cachedVersion, found := s.cache.Get(appID); found {
			log.Printf("Using cached version for %s: %s", appID, cachedVersion)
			return cachedVersion, nil
		}
		
		// Fallback to 'latest'
		version = "latest"
	}
	
	// Update cache
	s.cache.Set(appID, version, s.config.DefaultTTL)
	
	log.Printf("Updated %s version to %s", appID, version)
	return version, err
}

// RefreshVersion forces a refresh of the version cache for a specific app
func (s *EnhancedDefaultVersionService) RefreshVersion(appID string) error {
	// Invalidate cache entry
	s.cache.Invalidate(appID)
	
	// Fetch new version
	_, err := s.RefreshVersionFromSource(appID)
	return err
}

// GetVersionHistory retrieves version history for an application
func (s *EnhancedDefaultVersionService) GetVersionHistory(appID string) ([]string, error) {
	s.sourcesMutex.RLock()
	source, exists := s.sources[appID]
	s.sourcesMutex.RUnlock()
	
	if !exists {
		return []string{}, fmt.Errorf("no version source configured for %s", appID)
	}
	
	return source.GetVersionHistory()
}

// GetCacheStats returns cache statistics
func (s *EnhancedDefaultVersionService) GetCacheStats() CacheStats {
	return s.cache.GetStats()
}

// ClearCache clears all cached versions
func (s *EnhancedDefaultVersionService) ClearCache() {
	s.cache.Clear()
	log.Println("Version cache cleared")
}

// SetVersionSource manually sets a version source for an application
func (s *EnhancedDefaultVersionService) SetVersionSource(appID string, source VersionSource) {
	s.sourcesMutex.Lock()
	defer s.sourcesMutex.Unlock()
	
	s.sources[appID] = source
	log.Printf("Set version source for %s: %s (%s)", appID, source.GetSourceType(), source.GetSourceName())
}

// GetVersionSource retrieves the version source for an application
func (s *EnhancedDefaultVersionService) GetVersionSource(appID string) VersionSource {
	s.sourcesMutex.RLock()
	defer s.sourcesMutex.RUnlock()
	
	return s.sources[appID]
}

// loadVersionSourceFromConfig attempts to load version source from configuration
func (s *EnhancedDefaultVersionService) loadVersionSourceFromConfig(appID string) error {
	// This would typically load from the configuration files
	// For now, we'll implement a basic fallback mechanism
	
	// Try to load from config files if configLoader is available
	if s.configLoader != nil {
		// Load all applications and find the matching one
		apps, err := s.configLoader.LoadApplications("configs/applications")
		if err != nil {
			return fmt.Errorf("failed to load applications config: %w", err)
		}
		
		for _, app := range apps {
			if app.ID == appID {
				// Found the application in config, extract version source
				return s.createVersionSourceFromConfig(appID, app)
			}
		}
	}
	
	return fmt.Errorf("application %s not found in configuration", appID)
}

// createVersionSourceFromConfig creates a version source from application configuration
func (s *EnhancedDefaultVersionService) createVersionSourceFromConfig(appID string, app models.PredefinedApplication) error {
	// For applications loaded from config, we need to access the original config
	// This is a simplified implementation - in practice, we'd store the version source config
	
	// For now, we'll use some heuristics based on the application
	var source VersionSource
	var err error
	
	switch appID {
	case "code-server":
		source = NewGitHubVersionSource("coder/code-server")
	case "argocd":
		source = NewHelmVersionSource("https://argoproj.github.io/argo-helm", "argo-cd")
	default:
		source = NewStaticVersionSource("latest", appID)
	}
	
	if err != nil {
		return fmt.Errorf("failed to create version source: %w", err)
	}
	
	s.sourcesMutex.Lock()
	s.sources[appID] = source
	s.sourcesMutex.Unlock()
	
	log.Printf("Created version source for %s from configuration: %s", appID, source.GetSourceType())
	return nil
}