package models

// VersionSourceConfig defines how to fetch version information
type VersionSourceConfig struct {
	Type    string `json:"type" yaml:"type"`
	Source  string `json:"source" yaml:"source"`
	Pattern string `json:"pattern" yaml:"pattern"`
	Chart   string `json:"chart" yaml:"chart"` // For helm type
}

// PredefinedApplication represents a curated application available for deployment
type PredefinedApplication struct {
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Icon          string                  `json:"icon"`
	Category      string                  `json:"category"`
	Version       string                  `json:"version"`
	VersionSource VersionSourceConfig     `json:"version_source"`
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
