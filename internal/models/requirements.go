package models

import (
	"fmt"
	"net/url"
	"strings"
)

// ApplicationValidator interface defines methods for validating applications
type ApplicationValidator interface {
	ValidateRequirements(app PredefinedApplication) error
	ValidateHelmChart(chart HelmChartConfig) error
}

// DefaultApplicationValidator implements basic validation logic
type DefaultApplicationValidator struct{}

// NewDefaultApplicationValidator creates a new instance of DefaultApplicationValidator
func NewDefaultApplicationValidator() ApplicationValidator {
	return &DefaultApplicationValidator{}
}

// ValidateRequirements validates that application requirements are reasonable
func (v *DefaultApplicationValidator) ValidateRequirements(app PredefinedApplication) error {
	// Validate basic application fields
	if app.ID == "" {
		return fmt.Errorf("application ID cannot be empty")
	}
	if app.Name == "" {
		return fmt.Errorf("application name cannot be empty")
	}
	if app.Version == "" {
		return fmt.Errorf("application version cannot be empty")
	}

	// Validate resource requirements
	if app.Requirements.MinCPU < 0 {
		return fmt.Errorf("minimum CPU requirement cannot be negative: %.2f", app.Requirements.MinCPU)
	}
	if app.Requirements.MinCPU > 16 {
		return fmt.Errorf("minimum CPU requirement too high (max 16 cores): %.2f", app.Requirements.MinCPU)
	}
	if app.Requirements.MinMemory < 0 {
		return fmt.Errorf("minimum memory requirement cannot be negative: %d GB", app.Requirements.MinMemory)
	}
	if app.Requirements.MinMemory > 64 {
		return fmt.Errorf("minimum memory requirement too high (max 64 GB): %d GB", app.Requirements.MinMemory)
	}
	if app.Requirements.MinDisk < 0 {
		return fmt.Errorf("minimum disk requirement cannot be negative: %d GB", app.Requirements.MinDisk)
	}
	if app.Requirements.MinDisk > 1000 {
		return fmt.Errorf("minimum disk requirement too high (max 1000 GB): %d GB", app.Requirements.MinDisk)
	}

	// Validate port ranges
	if app.DefaultPort < 1 || app.DefaultPort > 65535 {
		return fmt.Errorf("default port must be between 1 and 65535: %d", app.DefaultPort)
	}

	// Validate category
	validCategories := []string{"Development", "Productivity", "DevOps", "Monitoring", "Database", "Utilities", "Other"}
	if app.Category != "" {
		validCategory := false
		for _, valid := range validCategories {
			if app.Category == valid {
				validCategory = true
				break
			}
		}
		if !validCategory {
			return fmt.Errorf("invalid category '%s', must be one of: %s", app.Category, strings.Join(validCategories, ", "))
		}
	}

	return nil
}

// ValidateHelmChart validates Helm chart configuration
func (v *DefaultApplicationValidator) ValidateHelmChart(chart HelmChartConfig) error {
	// Validate repository URL
	if chart.Repository == "" {
		return fmt.Errorf("helm chart repository cannot be empty")
	}

	// Check if repository is a valid URL
	if !strings.Contains(chart.Repository, "github.com") {
		// For non-GitHub repositories, validate as URL
		if _, err := url.Parse(chart.Repository); err != nil {
			return fmt.Errorf("invalid repository URL '%s': %v", chart.Repository, err)
		}
		// Check for common Helm repository patterns
		if !strings.HasPrefix(chart.Repository, "http://") && !strings.HasPrefix(chart.Repository, "https://") {
			return fmt.Errorf("repository URL must use http:// or https:// protocol: %s", chart.Repository)
		}
	} else {
		// For GitHub repositories, validate format
		if !strings.HasPrefix(chart.Repository, "https://github.com/") {
			return fmt.Errorf("GitHub repository must use https://github.com/ format: %s", chart.Repository)
		}
	}

	// Validate chart name
	if chart.Chart == "" {
		return fmt.Errorf("helm chart name cannot be empty")
	}
	if strings.Contains(chart.Chart, " ") {
		return fmt.Errorf("helm chart name cannot contain spaces: '%s'", chart.Chart)
	}

	// Validate version format (basic semantic versioning check)
	if chart.Version != "" {
		if !isValidVersion(chart.Version) {
			return fmt.Errorf("invalid chart version format '%s', should be semantic version (e.g., 1.2.3)", chart.Version)
		}
	}

	// Validate values template path
	if chart.ValuesTemplate == "" {
		return fmt.Errorf("values template path cannot be empty")
	}
	if !strings.HasSuffix(chart.ValuesTemplate, ".yaml") && !strings.HasSuffix(chart.ValuesTemplate, ".yml") {
		return fmt.Errorf("values template must be a YAML file (.yaml or .yml): %s", chart.ValuesTemplate)
	}

	// Validate namespace format
	if chart.Namespace != "" {
		if !isValidKubernetesName(chart.Namespace) {
			return fmt.Errorf("invalid namespace format '%s', must be a valid Kubernetes name", chart.Namespace)
		}
	}

	// Validate placeholder keys (no spaces or special characters)
	for key := range chart.Placeholders {
		if strings.Contains(key, " ") || !isValidPlaceholderKey(key) {
			return fmt.Errorf("invalid placeholder key '%s', must contain only letters, numbers, and underscores", key)
		}
	}

	return nil
}

// CheckApplicationRequirements checks if system meets application requirements
func CheckApplicationRequirements(app PredefinedApplication, availableCPU float64, availableMemoryGB int, availableDiskGB int) bool {
	return availableCPU >= app.Requirements.MinCPU &&
		availableMemoryGB >= app.Requirements.MinMemory &&
		availableDiskGB >= app.Requirements.MinDisk
}

// isValidVersion checks if a version string follows basic semantic versioning
func isValidVersion(version string) bool {
	if version == "latest" {
		return true
	}

	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Split by dots
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	// Check each part is numeric (allowing pre-release suffixes)
	for _, part := range parts {
		if part == "" {
			return false
		}
		// Allow pre-release versions like "1.2.3-beta"
		numPart := strings.SplitN(part, "-", 2)[0]
		for _, char := range numPart {
			if char < '0' || char > '9' {
				return false
			}
		}
	}

	return true
}

// isValidKubernetesName checks if a name follows Kubernetes naming conventions
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	// Must start and end with alphanumeric character
	if !isAlphaNumeric(rune(name[0])) || !isAlphaNumeric(rune(name[len(name)-1])) {
		return false
	}

	// Can contain lowercase letters, numbers, and hyphens
	for _, char := range name {
		if !isAlphaNumeric(char) && char != '-' {
			return false
		}
		// No uppercase letters
		if char >= 'A' && char <= 'Z' {
			return false
		}
	}

	return true
}

// isValidPlaceholderKey checks if a placeholder key is valid
func isValidPlaceholderKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	for _, char := range key {
		if !isAlphaNumeric(char) && char != '_' {
			return false
		}
	}

	return true
}

// isAlphaNumeric checks if a character is alphanumeric
func isAlphaNumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}
