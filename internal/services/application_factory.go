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
	registry              ApplicationRegistry
	enhancedValidator     *EnhancedApplicationValidator
}

// NewApplicationServiceFactory creates a new application service factory
func NewApplicationServiceFactory() *ApplicationServiceFactory {
	// Create config loader for enhanced version service
	validator := models.NewDefaultApplicationValidator()
	configLoader := models.NewYAMLConfigLoader(validator)
	
	// Create enhanced version service
	enhancedVersionService := NewEnhancedDefaultVersionService(configLoader)
	
	// Create enhanced validator
	enhancedValidator := NewEnhancedApplicationValidator(validator)
	
	// Create application registry
	registry := NewInMemoryApplicationRegistry(enhancedValidator)
	
	return &ApplicationServiceFactory{
		versionService:         enhancedVersionService, // Use enhanced service as default
		enhancedVersionService: enhancedVersionService,
		configLoader:          configLoader,
		registry:              registry,
		enhancedValidator:     enhancedValidator,
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

// CreateApplicationRegistry creates a new application registry
func (f *ApplicationServiceFactory) CreateApplicationRegistry() ApplicationRegistry {
	return f.registry
}

// CreateEnhancedValidator creates a new enhanced application validator
func (f *ApplicationServiceFactory) CreateEnhancedValidator() *EnhancedApplicationValidator {
	return f.enhancedValidator
}

// CreateRegistryCatalogBridge creates a registry-catalog bridge
func (f *ApplicationServiceFactory) CreateRegistryCatalogBridge() *RegistryWithCatalogBridge {
	return NewRegistryWithCatalogBridge(f.registry)
}

// CreateRegistryBasedCatalogService creates a catalog service backed by the registry
func (f *ApplicationServiceFactory) CreateRegistryBasedCatalogService() ApplicationCatalog {
	bridge := NewRegistryWithCatalogBridge(f.registry)
	return bridge
}

// CreateRegistryWithDefaults creates a registry pre-populated with default applications
func (f *ApplicationServiceFactory) CreateRegistryWithDefaults() (ApplicationRegistry, error) {
	// Load default applications from configuration
	configPath := GetDefaultConfigPath()
	configCatalog := NewConfigDrivenCatalogService(configPath, f.versionService)
	
	// Get default applications
	defaultApps := configCatalog.GetApplications()
	
	// Register default applications
	for _, app := range defaultApps {
		if err := f.registry.Register(app); err != nil {
			return nil, err
		}
	}
	
	return f.registry, nil
}

// ValidateApplicationWithCluster validates an application against cluster capabilities
func (f *ApplicationServiceFactory) ValidateApplicationWithCluster(app models.PredefinedApplication, cluster ClusterInfo) error {
	return f.enhancedValidator.ValidateWithCluster(app, cluster)
}