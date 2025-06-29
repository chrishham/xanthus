package services

import (
	"time"
	"github.com/chrishham/xanthus/internal/models"
)

// ApplicationServiceFactory provides a factory for creating application-related services
type ApplicationServiceFactory struct {
	versionService         VersionService
	enhancedVersionService EnhancedVersionService
	configLoader          models.ConfigLoader
}

// NewApplicationServiceFactory creates a new application service factory
func NewApplicationServiceFactory() *ApplicationServiceFactory {
	// Create config loader for enhanced version service
	validator := models.NewDefaultApplicationValidator()
	configLoader := models.NewYAMLConfigLoader(validator)
	
	// Create enhanced version service
	enhancedVersionService := NewEnhancedDefaultVersionService(configLoader)
	
	return &ApplicationServiceFactory{
		versionService:         enhancedVersionService, // Use enhanced service as default
		enhancedVersionService: enhancedVersionService,
		configLoader:          configLoader,
	}
}

// CreateCatalogService creates a new application catalog service
func (f *ApplicationServiceFactory) CreateCatalogService() ApplicationCatalog {
	return NewApplicationCatalogService(f.versionService)
}

// CreateConfigCatalogService creates a new configuration-driven catalog service
func (f *ApplicationServiceFactory) CreateConfigCatalogService() ApplicationCatalog {
	configPath := GetDefaultConfigPath()
	return NewConfigDrivenCatalogService(configPath, f.versionService)
}

// CreateHybridCatalogService creates a hybrid catalog service (config + fallback)
func (f *ApplicationServiceFactory) CreateHybridCatalogService() ApplicationCatalog {
	configPath := GetDefaultConfigPath()
	return NewHybridCatalogService(configPath, f.versionService)
}

// CreateValidatorService creates a new application validator service
func (f *ApplicationServiceFactory) CreateValidatorService() models.ApplicationValidator {
	return models.NewDefaultApplicationValidator()
}

// GetVersionService returns the version service instance
func (f *ApplicationServiceFactory) GetVersionService() VersionService {
	return f.versionService
}

// GetEnhancedVersionService returns the enhanced version service instance
func (f *ApplicationServiceFactory) GetEnhancedVersionService() EnhancedVersionService {
	return f.enhancedVersionService
}

// CreateBackgroundRefreshService creates a new background refresh service
func (f *ApplicationServiceFactory) CreateBackgroundRefreshService(config BackgroundRefreshConfig) *BackgroundRefreshService {
	return NewBackgroundRefreshService(f.enhancedVersionService, config)
}

// CreatePeriodicRefreshManager creates a new periodic refresh manager
func (f *ApplicationServiceFactory) CreatePeriodicRefreshManager(backgroundService *BackgroundRefreshService, catalogService ApplicationCatalog, interval time.Duration) *PeriodicRefreshManager {
	return NewPeriodicRefreshManager(backgroundService, catalogService, interval)
}