package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/chrishham/xanthus/internal/models"
)

// ApplicationRegistry manages the lifecycle of predefined applications
// providing registration, unregistration, updates, and listing capabilities
type ApplicationRegistry interface {
	Register(app models.PredefinedApplication) error
	Unregister(id string) error
	Update(id string, app models.PredefinedApplication) error
	List() []models.PredefinedApplication
	Get(id string) (*models.PredefinedApplication, bool)
	Validate(app models.PredefinedApplication) error
	Clear() error
	Count() int
	IsRegistered(id string) bool
}

// InMemoryApplicationRegistry is a thread-safe in-memory implementation
// of the ApplicationRegistry interface
type InMemoryApplicationRegistry struct {
	applications map[string]models.PredefinedApplication
	mutex        sync.RWMutex
	validator    models.ApplicationValidator
	createdAt    time.Time
	lastModified time.Time
}

// NewInMemoryApplicationRegistry creates a new in-memory application registry
func NewInMemoryApplicationRegistry(validator models.ApplicationValidator) *InMemoryApplicationRegistry {
	now := time.Now()
	return &InMemoryApplicationRegistry{
		applications: make(map[string]models.PredefinedApplication),
		validator:    validator,
		createdAt:    now,
		lastModified: now,
	}
}

// Register adds a new application to the registry
func (r *InMemoryApplicationRegistry) Register(app models.PredefinedApplication) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate the application before registration
	if err := r.validator.ValidateRequirements(app); err != nil {
		return fmt.Errorf("application validation failed: %w", err)
	}

	if err := r.validator.ValidateHelmChart(app.HelmChart); err != nil {
		return fmt.Errorf("helm chart validation failed: %w", err)
	}

	// Check if application already exists
	if _, exists := r.applications[app.ID]; exists {
		return fmt.Errorf("application with ID '%s' already exists", app.ID)
	}

	// Register the application
	r.applications[app.ID] = app
	r.lastModified = time.Now()

	return nil
}

// Unregister removes an application from the registry
func (r *InMemoryApplicationRegistry) Unregister(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.applications[id]; !exists {
		return fmt.Errorf("application with ID '%s' not found", id)
	}

	delete(r.applications, id)
	r.lastModified = time.Now()

	return nil
}

// Update modifies an existing application in the registry
func (r *InMemoryApplicationRegistry) Update(id string, app models.PredefinedApplication) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate the updated application
	if err := r.validator.ValidateRequirements(app); err != nil {
		return fmt.Errorf("application validation failed: %w", err)
	}

	if err := r.validator.ValidateHelmChart(app.HelmChart); err != nil {
		return fmt.Errorf("helm chart validation failed: %w", err)
	}

	// Check if application exists
	if _, exists := r.applications[id]; !exists {
		return fmt.Errorf("application with ID '%s' not found", id)
	}

	// Ensure the ID matches
	if app.ID != id {
		return fmt.Errorf("application ID mismatch: expected '%s', got '%s'", id, app.ID)
	}

	// Update the application
	r.applications[id] = app
	r.lastModified = time.Now()

	return nil
}

// List returns all registered applications
func (r *InMemoryApplicationRegistry) List() []models.PredefinedApplication {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	apps := make([]models.PredefinedApplication, 0, len(r.applications))
	for _, app := range r.applications {
		apps = append(apps, app)
	}

	return apps
}

// Get retrieves a specific application by ID
func (r *InMemoryApplicationRegistry) Get(id string) (*models.PredefinedApplication, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	app, exists := r.applications[id]
	if !exists {
		return nil, false
	}

	return &app, true
}

// Validate validates an application without registering it
func (r *InMemoryApplicationRegistry) Validate(app models.PredefinedApplication) error {
	if err := r.validator.ValidateRequirements(app); err != nil {
		return fmt.Errorf("application validation failed: %w", err)
	}

	if err := r.validator.ValidateHelmChart(app.HelmChart); err != nil {
		return fmt.Errorf("helm chart validation failed: %w", err)
	}

	return nil
}

// Clear removes all applications from the registry
func (r *InMemoryApplicationRegistry) Clear() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.applications = make(map[string]models.PredefinedApplication)
	r.lastModified = time.Now()

	return nil
}

// Count returns the number of registered applications
func (r *InMemoryApplicationRegistry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.applications)
}

// IsRegistered checks if an application with the given ID is registered
func (r *InMemoryApplicationRegistry) IsRegistered(id string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.applications[id]
	return exists
}

// RegistryWithCatalogBridge provides a bridge between ApplicationRegistry and ApplicationCatalog
// This allows the registry to be used as a catalog source while maintaining registry functionality
type RegistryWithCatalogBridge struct {
	registry ApplicationRegistry
}

// NewRegistryWithCatalogBridge creates a new registry-catalog bridge
func NewRegistryWithCatalogBridge(registry ApplicationRegistry) *RegistryWithCatalogBridge {
	return &RegistryWithCatalogBridge{
		registry: registry,
	}
}

// GetApplications implements ApplicationCatalog interface
func (b *RegistryWithCatalogBridge) GetApplications() []models.PredefinedApplication {
	return b.registry.List()
}

// GetApplicationByID implements ApplicationCatalog interface
func (b *RegistryWithCatalogBridge) GetApplicationByID(id string) (*models.PredefinedApplication, bool) {
	return b.registry.Get(id)
}

// GetCategories implements ApplicationCatalog interface
func (b *RegistryWithCatalogBridge) GetCategories() []string {
	applications := b.registry.List()
	categorySet := make(map[string]bool)

	for _, app := range applications {
		if app.Category != "" {
			categorySet[app.Category] = true
		}
	}

	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}

	return categories
}

// RefreshCatalog implements ApplicationCatalog interface
func (b *RegistryWithCatalogBridge) RefreshCatalog() error {
	// Registry doesn't need refreshing as it's always current
	return nil
}

// GetRegistry provides access to the underlying registry
func (b *RegistryWithCatalogBridge) GetRegistry() ApplicationRegistry {
	return b.registry
}
