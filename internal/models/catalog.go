package models

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chrishham/xanthus/internal/services"
)

var (
	latestCodeServerVersion string
	lastVersionCheck        time.Time
	versionMutex            sync.RWMutex
	versionCacheTTL         = 10 * time.Minute
)

// ApplicationCatalog interface defines methods for managing application catalog
type ApplicationCatalog interface {
	GetApplications() []PredefinedApplication
	GetApplicationByID(id string) (*PredefinedApplication, bool)
	GetCategories() []string
	RefreshCatalog() error
}

// DefaultApplicationCatalog implements ApplicationCatalog with hardcoded applications
type DefaultApplicationCatalog struct{}

// NewDefaultApplicationCatalog creates a new instance of DefaultApplicationCatalog
func NewDefaultApplicationCatalog() ApplicationCatalog {
	return &DefaultApplicationCatalog{}
}

// GetApplications returns the catalog of available applications
func (c *DefaultApplicationCatalog) GetApplications() []PredefinedApplication {
	codeServerVersion := getLatestCodeServerVersion()

	return []PredefinedApplication{
		{
			ID:          "code-server",
			Name:        "Code Server",
			Description: "VS Code in your browser - a full development environment accessible from anywhere",
			Icon:        "💻",
			Category:    "Development",
			Version:     codeServerVersion,
			HelmChart: HelmChartConfig{
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
			Requirements: ApplicationRequirements{
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
			HelmChart: HelmChartConfig{
				Repository:     "https://argoproj.github.io/argo-helm",
				Chart:          "argo-cd",
				Version:        "stable",
				Namespace:      "argocd",
				ValuesTemplate: "argocd.yaml",
				Placeholders:   map[string]string{},
			},
			DefaultPort: 80,
			Requirements: ApplicationRequirements{
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
func (c *DefaultApplicationCatalog) GetApplicationByID(id string) (*PredefinedApplication, bool) {
	apps := c.GetApplications()
	for _, app := range apps {
		if app.ID == id {
			return &app, true
		}
	}
	return nil, false
}

// GetCategories returns unique categories
func (c *DefaultApplicationCatalog) GetCategories() []string {
	apps := c.GetApplications()
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
func (c *DefaultApplicationCatalog) RefreshCatalog() error {
	RefreshVersionCache()
	return nil
}

// getLatestCodeServerVersion fetches the latest code-server version with caching
func getLatestCodeServerVersion() string {
	versionMutex.RLock()
	if latestCodeServerVersion != "" && time.Since(lastVersionCheck) < versionCacheTTL {
		defer versionMutex.RUnlock()
		return latestCodeServerVersion
	}
	versionMutex.RUnlock()

	versionMutex.Lock()
	defer versionMutex.Unlock()

	if latestCodeServerVersion != "" && time.Since(lastVersionCheck) < versionCacheTTL {
		return latestCodeServerVersion
	}

	githubService := services.NewGitHubService()
	release, err := githubService.GetCodeServerLatestVersion()
	if err != nil {
		log.Printf("Warning: Failed to fetch latest code-server version: %v", err)
		if latestCodeServerVersion != "" {
			return latestCodeServerVersion
		}
		return "4.101.1"
	}

	version := strings.TrimPrefix(release.TagName, "v")

	latestCodeServerVersion = version
	lastVersionCheck = time.Now()

	log.Printf("Updated code-server version to %s", version)
	return version
}

// RefreshVersionCache forces a refresh of the version cache
func RefreshVersionCache() {
	versionMutex.Lock()
	defer versionMutex.Unlock()
	lastVersionCheck = time.Time{}
}

// Legacy functions for backward compatibility
// GetPredefinedApplications returns the catalog of available applications
func GetPredefinedApplications() []PredefinedApplication {
	catalog := NewDefaultApplicationCatalog()
	return catalog.GetApplications()
}

// GetPredefinedApplicationByID returns a specific predefined application by ID
func GetPredefinedApplicationByID(id string) (*PredefinedApplication, bool) {
	catalog := NewDefaultApplicationCatalog()
	return catalog.GetApplicationByID(id)
}

// GetPredefinedApplicationCategories returns unique categories
func GetPredefinedApplicationCategories() []string {
	catalog := NewDefaultApplicationCatalog()
	return catalog.GetCategories()
}