package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ApplicationConfig represents the YAML configuration structure for applications
type ApplicationConfig struct {
	ID            string                     `yaml:"id" validate:"required"`
	Name          string                     `yaml:"name" validate:"required"`
	Description   string                     `yaml:"description" validate:"required"`
	Icon          string                     `yaml:"icon,omitempty"`
	Category      string                     `yaml:"category" validate:"required"`
	VersionSource VersionSourceConfig        `yaml:"version_source"`
	HelmChart     HelmChartConfigYAML        `yaml:"helm_chart"`
	DefaultPort   int                        `yaml:"default_port" validate:"required,min=1,max=65535"`
	Requirements  ApplicationRequirementsYAML `yaml:"requirements"`
	Features      []string                   `yaml:"features,omitempty"`
	Documentation string                     `yaml:"documentation,omitempty"`
	Metadata      ApplicationMetadata        `yaml:"metadata,omitempty"`
}

// VersionSourceConfig defines how to fetch version information
type VersionSourceConfig struct {
	Type    string `yaml:"type" validate:"required,oneof=github dockerhub helm static"`
	Source  string `yaml:"source" validate:"required"`
	Pattern string `yaml:"pattern,omitempty"`
	Chart   string `yaml:"chart,omitempty"` // For helm type
}

// HelmChartConfigYAML represents Helm configuration in YAML
type HelmChartConfigYAML struct {
	Repository     string            `yaml:"repository" validate:"required,url"`
	Chart          string            `yaml:"chart" validate:"required"`
	Version        string            `yaml:"version" validate:"required"`
	Namespace      string            `yaml:"namespace" validate:"required"`
	ValuesTemplate string            `yaml:"values_template,omitempty"`
	Placeholders   map[string]string `yaml:"placeholders,omitempty"`
}

// ApplicationRequirementsYAML defines minimum system requirements in YAML
type ApplicationRequirementsYAML struct {
	MinCPU      float64 `yaml:"min_cpu" validate:"required,min=0"`
	MinMemoryGB int     `yaml:"min_memory_gb" validate:"required,min=0"`
	MinDiskGB   int     `yaml:"min_disk_gb" validate:"required,min=0"`
}

// ApplicationMetadata contains additional application information
type ApplicationMetadata struct {
	Maintainer string `yaml:"maintainer,omitempty"`
	Support    string `yaml:"support,omitempty"`
	License    string `yaml:"license,omitempty"`
}

// ConfigLoader interface defines how to load application configurations
type ConfigLoader interface {
	LoadApplications(configPath string) ([]PredefinedApplication, error)
	LoadApplication(configFile string) (*PredefinedApplication, error)
	ValidateConfig(config ApplicationConfig) error
}

// YAMLConfigLoader implements ConfigLoader for YAML files
type YAMLConfigLoader struct {
	validator ApplicationValidator
}

// NewYAMLConfigLoader creates a new YAML configuration loader
func NewYAMLConfigLoader(validator ApplicationValidator) ConfigLoader {
	return &YAMLConfigLoader{
		validator: validator,
	}
}

// LoadApplications loads all application configurations from a directory
func (l *YAMLConfigLoader) LoadApplications(configPath string) ([]PredefinedApplication, error) {
	var applications []PredefinedApplication

	// Check if directory exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration directory does not exist: %s", configPath)
	}

	// Read all YAML files in the directory
	files, err := filepath.Glob(filepath.Join(configPath, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, err := filepath.Glob(filepath.Join(configPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration files: %w", err)
	}
	files = append(files, ymlFiles...)

	for _, file := range files {
		// Skip template files
		if strings.Contains(filepath.Base(file), "template") {
			continue
		}

		app, err := l.LoadApplication(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load application from %s: %w", file, err)
		}

		if app != nil {
			applications = append(applications, *app)
		}
	}

	return applications, nil
}

// LoadApplication loads a single application configuration from a YAML file
func (l *YAMLConfigLoader) LoadApplication(configFile string) (*PredefinedApplication, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ApplicationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate configuration
	if err := l.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Convert to PredefinedApplication
	app := l.convertToPredefinedApplication(config)
	
	return &app, nil
}

// ValidateConfig validates an application configuration
func (l *YAMLConfigLoader) ValidateConfig(config ApplicationConfig) error {
	// Basic validation
	if config.ID == "" {
		return fmt.Errorf("application ID is required")
	}
	if config.Name == "" {
		return fmt.Errorf("application name is required")
	}
	if config.Description == "" {
		return fmt.Errorf("application description is required")
	}
	if config.Category == "" {
		return fmt.Errorf("application category is required")
	}

	// Validate version source
	if err := l.validateVersionSource(config.VersionSource); err != nil {
		return fmt.Errorf("invalid version source: %w", err)
	}

	// Validate Helm configuration
	if err := l.validateHelmConfig(config.HelmChart); err != nil {
		return fmt.Errorf("invalid Helm configuration: %w", err)
	}

	// Validate port
	if config.DefaultPort <= 0 || config.DefaultPort > 65535 {
		return fmt.Errorf("default port must be between 1 and 65535")
	}

	// Validate requirements
	if config.Requirements.MinCPU < 0 {
		return fmt.Errorf("minimum CPU cannot be negative")
	}
	if config.Requirements.MinMemoryGB < 0 {
		return fmt.Errorf("minimum memory cannot be negative")
	}
	if config.Requirements.MinDiskGB < 0 {
		return fmt.Errorf("minimum disk space cannot be negative")
	}

	return nil
}

// validateVersionSource validates version source configuration
func (l *YAMLConfigLoader) validateVersionSource(vs VersionSourceConfig) error {
	validTypes := []string{"github", "dockerhub", "helm", "static"}
	
	if vs.Type == "" {
		return fmt.Errorf("version source type is required")
	}

	for _, validType := range validTypes {
		if vs.Type == validType {
			break
		}
	}

	if vs.Source == "" {
		return fmt.Errorf("version source is required")
	}

	// Type-specific validation
	switch vs.Type {
	case "helm":
		if vs.Chart == "" {
			return fmt.Errorf("chart name is required for helm version source")
		}
	}

	return nil
}

// validateHelmConfig validates Helm chart configuration
func (l *YAMLConfigLoader) validateHelmConfig(hc HelmChartConfigYAML) error {
	if hc.Repository == "" {
		return fmt.Errorf("Helm repository is required")
	}
	if hc.Chart == "" {
		return fmt.Errorf("Helm chart is required")
	}
	if hc.Version == "" {
		return fmt.Errorf("Helm chart version is required")
	}
	if hc.Namespace == "" {
		return fmt.Errorf("Helm namespace is required")
	}

	return nil
}

// convertToPredefinedApplication converts ApplicationConfig to PredefinedApplication
func (l *YAMLConfigLoader) convertToPredefinedApplication(config ApplicationConfig) PredefinedApplication {
	return PredefinedApplication{
		ID:          config.ID,
		Name:        config.Name,
		Description: config.Description,
		Icon:        config.Icon,
		Category:    config.Category,
		Version:     "dynamic", // Will be resolved by version service
		HelmChart: HelmChartConfig{
			Repository:     config.HelmChart.Repository,
			Chart:          config.HelmChart.Chart,
			Version:        config.HelmChart.Version,
			ValuesTemplate: config.HelmChart.ValuesTemplate,
			Placeholders:   config.HelmChart.Placeholders,
			Namespace:      config.HelmChart.Namespace,
		},
		DefaultPort: config.DefaultPort,
		Requirements: ApplicationRequirements{
			MinCPU:    config.Requirements.MinCPU,
			MinMemory: config.Requirements.MinMemoryGB,
			MinDisk:   config.Requirements.MinDiskGB,
		},
		Features:      config.Features,
		Documentation: config.Documentation,
	}
}