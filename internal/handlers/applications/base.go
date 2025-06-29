package applications

import (
	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
)

// Handler contains dependencies for application-related operations
type Handler struct {
	catalog        services.ApplicationCatalog
	validator      models.ApplicationValidator
	serviceFactory *services.ApplicationServiceFactory
}

// NewHandler creates a new applications handler instance using the service layer
func NewHandler() *Handler {
	factory := services.NewApplicationServiceFactory()
	return &Handler{
		catalog:        factory.CreateHybridCatalogService(),
		validator:      factory.CreateValidatorService(),
		serviceFactory: factory,
	}
}