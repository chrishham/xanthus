package services

import "github.com/chrishham/xanthus/internal/models"

// ApplicationServiceFactory provides a factory for creating application-related services
type ApplicationServiceFactory struct {
	versionService VersionService
}

// NewApplicationServiceFactory creates a new application service factory
func NewApplicationServiceFactory() *ApplicationServiceFactory {
	return &ApplicationServiceFactory{
		versionService: NewDefaultVersionService(),
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