package services

import (
	"log"

	"github.com/chrishham/xanthus/internal/models"
)

// ApplicationCatalog interface defines methods for managing application catalog
type ApplicationCatalog interface {
	GetApplications() []models.PredefinedApplication
	GetApplicationByID(id string) (*models.PredefinedApplication, bool)
	GetCategories() []string
	RefreshCatalog() error
}

// ApplicationCatalogService provides application catalog functionality with external dependencies
type ApplicationCatalogService struct {
	versionService VersionService
}

// NewApplicationCatalogService creates a new application catalog service with dependencies
func NewApplicationCatalogService(versionService VersionService) ApplicationCatalog {
	return &ApplicationCatalogService{
		versionService: versionService,
	}
}

// GetApplications returns the catalog of available applications
func (s *ApplicationCatalogService) GetApplications() []models.PredefinedApplication {
	codeServerVersion, err := s.versionService.GetLatestVersion("code-server")
	if err != nil {
		log.Printf("Warning: Failed to fetch latest code-server version: %v", err)
		codeServerVersion = "4.101.1" // fallback version
	}

	return []models.PredefinedApplication{
		{
			ID:          "code-server",
			Name:        "Code Server",
			Description: "VS Code in your browser - a full development environment accessible from anywhere",
			Icon:        "💻",
			Category:    "Development",
			Version:     codeServerVersion,
			HelmChart: models.HelmChartConfig{
				Repository:     "https://github.com/coder/code-server",
				Chart:          "ci/helm-chart",
				Version:        "main",
				Namespace:      "code-server",
				ValuesTemplate: "code-server.yaml",
				Placeholders: map[string]string{
					"VERSION": codeServerVersion,
				},
			},
			DefaultPort: 8080,
			Requirements: models.ApplicationRequirements{
				MinCPU:    0.5,
				MinMemory: 1,
				MinDisk:   10,
			},
			Features: []string{
				"Full VS Code experience in browser",
				"Git integration",
				"Terminal access",
				"Extension support",
				"Docker integration",
				"Persistent workspace",
			},
			Documentation: "https://coder.com/docs/code-server",
		},
		{
			ID:          "argocd",
			Name:        "Argo CD",
			Description: "Declarative, GitOps continuous delivery tool for Kubernetes",
			Icon:        "🚀",
			Category:    "DevOps",
			Version:     "stable",
			HelmChart: models.HelmChartConfig{
				Repository:     "https://argoproj.github.io/argo-helm",
				Chart:          "argo-cd",
				Version:        "stable",
				Namespace:      "argocd",
				ValuesTemplate: "argocd.yaml",
				Placeholders:   map[string]string{},
			},
			DefaultPort: 80,
			Requirements: models.ApplicationRequirements{
				MinCPU:    1.0,
				MinMemory: 2,
				MinDisk:   5,
			},
			Features: []string{
				"GitOps application delivery",
				"Declarative configuration",
				"Web UI and CLI",
				"Multi-cluster support",
				"RBAC and SSO integration",
				"Automated synchronization",
			},
			Documentation: "https://argo-cd.readthedocs.io/",
		},
	}
}

// GetApplicationByID returns a specific predefined application by ID
func (s *ApplicationCatalogService) GetApplicationByID(id string) (*models.PredefinedApplication, bool) {
	apps := s.GetApplications()
	for _, app := range apps {
		if app.ID == id {
			return &app, true
		}
	}
	return nil, false
}

// GetCategories returns unique categories
func (s *ApplicationCatalogService) GetCategories() []string {
	apps := s.GetApplications()
	categoryMap := make(map[string]bool)

	for _, app := range apps {
		categoryMap[app.Category] = true
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// RefreshCatalog forces a refresh of the application catalog
func (s *ApplicationCatalogService) RefreshCatalog() error {
	return s.versionService.RefreshVersion("code-server")
}