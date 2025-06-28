package models

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
	Repository string                 `json:"repository"`
	Chart      string                 `json:"chart"`
	Version    string                 `json:"version"`
	Values     map[string]interface{} `json:"values"`
	Namespace  string                 `json:"namespace"`
}

// ApplicationRequirements defines minimum system requirements
type ApplicationRequirements struct {
	MinCPU    float64 `json:"min_cpu"`
	MinMemory int     `json:"min_memory_gb"`
	MinDisk   int     `json:"min_disk_gb"`
}

// GetPredefinedApplications returns the catalog of available applications
func GetPredefinedApplications() []PredefinedApplication {
	return []PredefinedApplication{
		{
			ID:          "code-server",
			Name:        "Code Server",
			Description: "VS Code in your browser - a full development environment accessible from anywhere",
			Icon:        "💻",
			Category:    "Development",
			Version:     "4.101.1",
			HelmChart: HelmChartConfig{
				Repository: "https://github.com/coder/code-server",
				Chart:      "ci/helm-chart",
				Version:    "main",
				Namespace:  "code-server",
				Values: map[string]interface{}{
					// Basic configuration
					"image.repository": "codercom/code-server",
					"image.tag":        "4.101.1",
					"service.type":     "ClusterIP",
					"service.port":     8080,

					// Ingress configuration with Traefik
					"ingress.enabled": true,
					"ingress.annotations.traefik\\.ingress\\.kubernetes\\.io/router\\.entrypoints": "websecure",
					"ingress.annotations.traefik\\.ingress\\.kubernetes\\.io/router\\.tls":         "true",
					"ingress.hosts[0].host":     "{{SUBDOMAIN}}.{{DOMAIN}}",
					"ingress.hosts[0].paths":    []string{"/"},
					"ingress.tls[0].secretName": "{{DOMAIN}}-tls",
					"ingress.tls[0].hosts":      []string{"{{SUBDOMAIN}}.{{DOMAIN}}"},

					// Persistence
					"persistence.enabled": true,
					"persistence.size":    "10Gi",

					// Resources
					"resources.limits.cpu":      "2",
					"resources.limits.memory":   "4Gi",
					"resources.requests.cpu":    "100m",
					"resources.requests.memory": "128Mi",

					// Security
					"securityContext.enabled":   true,
					"securityContext.fsGroup":   1000,
					"securityContext.runAsUser": 1000,
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
