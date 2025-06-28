package models

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/chrishham/xanthus/internal/services"
)

// PredefinedApplication represents a curated application available for deployment
type PredefinedApplication struct {
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Icon          string                  `json:"icon"`
	Category      string                  `json:"category"`
	Version       string                  `json:"version"`
	HelmChart     HelmChartConfig         `json:"helm_chart"`
	DefaultPort   int                     `json:"default_port"`
	Requirements  ApplicationRequirements `json:"requirements"`
	Features      []string                `json:"features"`
	Documentation string                  `json:"documentation"`
}

// HelmChartConfig contains Helm chart deployment configuration
type HelmChartConfig struct {
	Repository     string            `json:"repository"`
	Chart          string            `json:"chart"`
	Version        string            `json:"version"`
	ValuesTemplate string            `json:"values_template"` // Path to values template file
	Placeholders   map[string]string `json:"placeholders"`    // Additional placeholder values
	Namespace      string            `json:"namespace"`
}

// ApplicationRequirements defines minimum system requirements
type ApplicationRequirements struct {
	MinCPU    float64 `json:"min_cpu"`
	MinMemory int     `json:"min_memory_gb"`
	MinDisk   int     `json:"min_disk_gb"`
}

var (
	// Cache for the latest code-server version
	latestCodeServerVersion string
	lastVersionCheck        time.Time
	versionMutex            sync.RWMutex
	versionCacheTTL         = 10 * time.Minute // Cache TTL for version checks
)

// GetPredefinedApplications returns the catalog of available applications
func GetPredefinedApplications() []PredefinedApplication {
	// Get the latest code-server version
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
	}
}

// GetPredefinedApplicationByID returns a specific predefined application by ID
func GetPredefinedApplicationByID(id string) (*PredefinedApplication, bool) {
	apps := GetPredefinedApplications()
	for _, app := range apps {
		if app.ID == id {
			return &app, true
		}
	}
	return nil, false
}

// GetPredefinedApplicationCategories returns unique categories
func GetPredefinedApplicationCategories() []string {
	apps := GetPredefinedApplications()
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

// getLatestCodeServerVersion fetches the latest code-server version with caching
func getLatestCodeServerVersion() string {
	versionMutex.RLock()
	// Check if we have a cached version that's still valid
	if latestCodeServerVersion != "" && time.Since(lastVersionCheck) < versionCacheTTL {
		defer versionMutex.RUnlock()
		return latestCodeServerVersion
	}
	versionMutex.RUnlock()

	// Need to fetch new version
	versionMutex.Lock()
	defer versionMutex.Unlock()

	// Double-check in case another goroutine updated it while we were waiting
	if latestCodeServerVersion != "" && time.Since(lastVersionCheck) < versionCacheTTL {
		return latestCodeServerVersion
	}

	// Fetch latest version from GitHub
	githubService := services.NewGitHubService()
	release, err := githubService.GetCodeServerLatestVersion()
	if err != nil {
		log.Printf("Warning: Failed to fetch latest code-server version: %v", err)
		// Return fallback version if we can't fetch from GitHub
		if latestCodeServerVersion != "" {
			return latestCodeServerVersion
		}
		return "4.101.1" // Fallback version
	}

	// Convert GitHub tag format (v4.101.2) to Docker format (4.101.2)
	version := strings.TrimPrefix(release.TagName, "v")

	// Update cache
	latestCodeServerVersion = version
	lastVersionCheck = time.Now()

	log.Printf("Updated code-server version to %s", version)
	return version
}

// RefreshVersionCache forces a refresh of the version cache
func RefreshVersionCache() {
	versionMutex.Lock()
	defer versionMutex.Unlock()

	// Reset the cache timestamp to force a refresh on next call
	lastVersionCheck = time.Time{}
}
