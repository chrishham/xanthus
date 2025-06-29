package services

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/chrishham/xanthus/internal/models"
)

// ConfigDrivenCatalogService implements ApplicationCatalog using YAML configuration files
type ConfigDrivenCatalogService struct {
	configPath     string
	configLoader   models.ConfigLoader
	versionService VersionService
	applications   []models.PredefinedApplication
	categories     []string
	mutex          sync.RWMutex
	loaded         bool
}

// NewConfigDrivenCatalogService creates a new configuration-driven catalog service
func NewConfigDrivenCatalogService(configPath string, versionService VersionService) ApplicationCatalog {
	validator := models.NewDefaultApplicationValidator()
	configLoader := models.NewYAMLConfigLoader(validator)

	service := &ConfigDrivenCatalogService{
		configPath:     configPath,
		configLoader:   configLoader,
		versionService: versionService,
		applications:   []models.PredefinedApplication{},
		categories:     []string{},
	}

	// Load applications on startup
	if err := service.loadApplications(); err != nil {
		log.Printf("Warning: Failed to load applications from config: %v", err)
		log.Printf("Falling back to empty catalog")
	}

	return service
}

// GetApplications returns all configured applications with current versions
func (s *ConfigDrivenCatalogService) GetApplications() []models.PredefinedApplication {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Ensure applications are loaded
	if !s.loaded {
		s.mutex.RUnlock()
		if err := s.RefreshCatalog(); err != nil {
			log.Printf("Failed to load applications: %v", err)
			return []models.PredefinedApplication{}
		}
		s.mutex.RLock()
	}

	// Create a copy with current versions
	apps := make([]models.PredefinedApplication, len(s.applications))
	for i, app := range s.applications {
		apps[i] = app

		// Get current version from version service
		if version, err := s.versionService.GetLatestVersion(app.ID); err == nil {
			apps[i].Version = version

			// Update placeholders with current version
			if apps[i].HelmChart.Placeholders == nil {
				apps[i].HelmChart.Placeholders = make(map[string]string)
			}
			apps[i].HelmChart.Placeholders["VERSION"] = version
		} else {
			log.Printf("Warning: Failed to get version for %s: %v", app.ID, err)
			// Keep the version as "dynamic" or provide a fallback
			if apps[i].Version == "dynamic" {
				apps[i].Version = "latest"
			}
		}
	}

	return apps
}

// GetApplicationByID returns a specific application by ID with current version
func (s *ConfigDrivenCatalogService) GetApplicationByID(id string) (*models.PredefinedApplication, bool) {
	apps := s.GetApplications()
	for _, app := range apps {
		if app.ID == id {
			return &app, true
		}
	}
	return nil, false
}

// GetCategories returns unique categories from all applications
func (s *ConfigDrivenCatalogService) GetCategories() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Ensure applications are loaded
	if !s.loaded {
		s.mutex.RUnlock()
		if err := s.RefreshCatalog(); err != nil {
			log.Printf("Failed to load applications: %v", err)
			return []string{}
		}
		s.mutex.RLock()
	}

	return s.categories
}

// RefreshCatalog reloads applications from configuration files
func (s *ConfigDrivenCatalogService) RefreshCatalog() error {
	return s.loadApplications()
}

// loadApplications loads applications from configuration files
func (s *ConfigDrivenCatalogService) loadApplications() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Load applications from configuration files
	apps, err := s.configLoader.LoadApplications(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to load applications: %w", err)
	}

	// Extract unique categories
	categoryMap := make(map[string]bool)
	for _, app := range apps {
		categoryMap[app.Category] = true
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	// Update internal state
	s.applications = apps
	s.categories = categories
	s.loaded = true

	log.Printf("Loaded %d applications from configuration files", len(apps))
	return nil
}

// HybridCatalogService combines configuration-driven and hardcoded applications
type HybridCatalogService struct {
	configCatalog   ApplicationCatalog
	fallbackCatalog ApplicationCatalog
}

// NewHybridCatalogService creates a catalog that tries configuration first, then falls back to hardcoded
func NewHybridCatalogService(configPath string, versionService VersionService) ApplicationCatalog {
	configCatalog := NewConfigDrivenCatalogService(configPath, versionService)
	fallbackCatalog := NewApplicationCatalogService(versionService)

	return &HybridCatalogService{
		configCatalog:   configCatalog,
		fallbackCatalog: fallbackCatalog,
	}
}

// GetApplications returns applications from config, falling back to hardcoded if config fails
func (s *HybridCatalogService) GetApplications() []models.PredefinedApplication {
	// Try configuration-driven catalog first
	configApps := s.configCatalog.GetApplications()
	if len(configApps) > 0 {
		return configApps
	}

	// Fall back to hardcoded catalog
	log.Printf("Using fallback catalog due to empty configuration")
	return s.fallbackCatalog.GetApplications()
}

// GetApplicationByID returns application from config, falling back to hardcoded
func (s *HybridCatalogService) GetApplicationByID(id string) (*models.PredefinedApplication, bool) {
	// Try configuration-driven catalog first
	if app, found := s.configCatalog.GetApplicationByID(id); found {
		return app, true
	}

	// Fall back to hardcoded catalog
	return s.fallbackCatalog.GetApplicationByID(id)
}

// GetCategories returns categories from config, falling back to hardcoded
func (s *HybridCatalogService) GetCategories() []string {
	// Try configuration-driven catalog first
	configCategories := s.configCatalog.GetCategories()
	if len(configCategories) > 0 {
		return configCategories
	}

	// Fall back to hardcoded catalog
	return s.fallbackCatalog.GetCategories()
}

// RefreshCatalog refreshes both catalogs
func (s *HybridCatalogService) RefreshCatalog() error {
	// Refresh configuration catalog
	if err := s.configCatalog.RefreshCatalog(); err != nil {
		log.Printf("Failed to refresh config catalog: %v", err)
	}

	// Refresh fallback catalog
	if err := s.fallbackCatalog.RefreshCatalog(); err != nil {
		log.Printf("Failed to refresh fallback catalog: %v", err)
	}

	return nil
}

// GetDefaultConfigPath returns the default path for application configurations
func GetDefaultConfigPath() string {
	return filepath.Join("configs", "applications")
}
