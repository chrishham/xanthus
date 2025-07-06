package applications

import (
	"embed"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
)

// Handler contains dependencies for application-related operations
type Handler struct {
	catalog        services.ApplicationCatalog
	validator      models.ApplicationValidator
	serviceFactory *services.ApplicationServiceFactory
	embedFS        *embed.FS
}

// NewHandler creates a new applications handler instance using the service layer
func NewHandler() *Handler {
	factory := services.NewApplicationServiceFactory()
	return &Handler{
		catalog:        factory.CreateHybridCatalogService(),
		validator:      factory.CreateValidatorService(),
		serviceFactory: factory,
		embedFS:        nil,
	}
}

// NewHandlerWithEmbedFS creates a new applications handler instance with embedded FS
func NewHandlerWithEmbedFS(embedFS *embed.FS) *Handler {
	factory := services.NewApplicationServiceFactoryWithEmbedFS(embedFS)
	return &Handler{
		catalog:        factory.CreateHybridCatalogService(),
		validator:      factory.CreateValidatorService(),
		serviceFactory: factory,
		embedFS:        embedFS,
	}
}

// GetApplicationService returns a SimpleApplicationService instance with embedded FS if available
func (h *Handler) GetApplicationService() *services.SimpleApplicationService {
	if h.embedFS != nil {
		return services.NewSimpleApplicationServiceWithEmbedFS(h.embedFS)
	}
	return services.NewSimpleApplicationService()
}
