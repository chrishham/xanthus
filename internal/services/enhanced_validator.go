package services

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/chrishham/xanthus/internal/models"
)

// EnhancedApplicationValidator provides comprehensive validation for applications
// with detailed checks for all application components
type EnhancedApplicationValidator struct {
	baseValidator models.ApplicationValidator
}

// ClusterInfo represents cluster information for resource validation
type ClusterInfo struct {
	AvailableCPU      float64
	AvailableMemoryGB int
	AvailableDiskGB   int
	KubernetesVersion string
	HelmVersion       string
	NodeCount         int
}

// NewEnhancedApplicationValidator creates a new enhanced validator
func NewEnhancedApplicationValidator(baseValidator models.ApplicationValidator) *EnhancedApplicationValidator {
	return &EnhancedApplicationValidator{
		baseValidator: baseValidator,
	}
}

// ValidateRequirements validates application requirements with enhanced checks
func (v *EnhancedApplicationValidator) ValidateRequirements(app models.PredefinedApplication) error {
	// First run the base validator
	if err := v.baseValidator.ValidateRequirements(app); err != nil {
		return err
	}

	// Enhanced validation checks
	if err := v.validateBasicFields(app); err != nil {
		return err
	}

	if err := v.validateRequirements(app.Requirements); err != nil {
		return err
	}

	if err := v.validateFeatures(app.Features); err != nil {
		return err
	}

	if err := v.validateDocumentation(app.Documentation); err != nil {
		return err
	}

	return nil
}

// ValidateHelmChart validates Helm chart configuration with enhanced checks
func (v *EnhancedApplicationValidator) ValidateHelmChart(chart models.HelmChartConfig) error {
	// First run the base validator
	if err := v.baseValidator.ValidateHelmChart(chart); err != nil {
		return err
	}

	// Enhanced validation checks
	if err := v.validateChartBasics(chart); err != nil {
		return err
	}

	if err := v.validateRepository(chart.Repository); err != nil {
		return err
	}

	if err := v.validateNamespace(chart.Namespace); err != nil {
		return err
	}

	if err := v.validateVersion(chart.Version); err != nil {
		return err
	}

	return nil
}

// ValidateWithCluster validates application against cluster capabilities
func (v *EnhancedApplicationValidator) ValidateWithCluster(app models.PredefinedApplication, cluster ClusterInfo) error {
	// Basic application validation first
	if err := v.ValidateRequirements(app); err != nil {
		return err
	}

	if err := v.ValidateHelmChart(app.HelmChart); err != nil {
		return err
	}

	// Cluster-specific validation
	if err := v.validateResourceRequirements(app.Requirements, cluster); err != nil {
		return err
	}

	if err := v.validateKubernetesCompatibility(app, cluster); err != nil {
		return err
	}

	return nil
}

// ValidateConfig validates application configuration from YAML
func (v *EnhancedApplicationValidator) ValidateConfig(app models.PredefinedApplication) error {
	// This method provides additional validation specifically for YAML-loaded applications
	if err := v.ValidateRequirements(app); err != nil {
		return err
	}

	if err := v.ValidateHelmChart(app.HelmChart); err != nil {
		return err
	}

	// Additional config-specific validation
	if err := v.validateConfigFields(app); err != nil {
		return err
	}

	return nil
}

// validateBasicFields validates core application fields
func (v *EnhancedApplicationValidator) validateBasicFields(app models.PredefinedApplication) error {
	if strings.TrimSpace(app.ID) == "" {
		return fmt.Errorf("application ID cannot be empty")
	}

	if strings.TrimSpace(app.Name) == "" {
		return fmt.Errorf("application name cannot be empty")
	}

	if strings.TrimSpace(app.Description) == "" {
		return fmt.Errorf("application description cannot be empty")
	}

	if strings.TrimSpace(app.Category) == "" {
		return fmt.Errorf("application category cannot be empty")
	}

	// ID should be a valid identifier (letters, numbers, hyphens, underscores)
	validID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validID.MatchString(app.ID) {
		return fmt.Errorf("application ID '%s' contains invalid characters (only letters, numbers, hyphens, and underscores allowed)", app.ID)
	}

	// ID should not be too long
	if len(app.ID) > 50 {
		return fmt.Errorf("application ID '%s' is too long (maximum 50 characters)", app.ID)
	}

	// Name should not be too long
	if len(app.Name) > 100 {
		return fmt.Errorf("application name is too long (maximum 100 characters)")
	}

	// Description should not be too long
	if len(app.Description) > 500 {
		return fmt.Errorf("application description is too long (maximum 500 characters)")
	}

	return nil
}

// validateRequirements validates resource requirements
func (v *EnhancedApplicationValidator) validateRequirements(req models.ApplicationRequirements) error {
	if req.MinCPU < 0 {
		return fmt.Errorf("minimum CPU requirement cannot be negative")
	}

	if req.MinCPU > 32 {
		return fmt.Errorf("minimum CPU requirement %.2f is too high (maximum 32 cores)", req.MinCPU)
	}

	if req.MinMemory < 0 {
		return fmt.Errorf("minimum memory requirement cannot be negative")
	}

	if req.MinMemory > 128 {
		return fmt.Errorf("minimum memory requirement %d GB is too high (maximum 128 GB)", req.MinMemory)
	}

	if req.MinDisk < 0 {
		return fmt.Errorf("minimum disk requirement cannot be negative")
	}

	if req.MinDisk > 1000 {
		return fmt.Errorf("minimum disk requirement %d GB is too high (maximum 1000 GB)", req.MinDisk)
	}

	// Reasonable minimums
	if req.MinCPU > 0 && req.MinCPU < 0.1 {
		return fmt.Errorf("minimum CPU requirement %.2f is too low (minimum 0.1 cores)", req.MinCPU)
	}

	if req.MinMemory > 0 && req.MinMemory < 1 {
		return fmt.Errorf("minimum memory requirement %d GB is too low (minimum 1 GB)", req.MinMemory)
	}

	if req.MinDisk > 0 && req.MinDisk < 1 {
		return fmt.Errorf("minimum disk requirement %d GB is too low (minimum 1 GB)", req.MinDisk)
	}

	return nil
}

// validateFeatures validates feature list
func (v *EnhancedApplicationValidator) validateFeatures(features []string) error {
	if len(features) == 0 {
		return fmt.Errorf("application must have at least one feature")
	}

	if len(features) > 20 {
		return fmt.Errorf("too many features listed (maximum 20)")
	}

	for i, feature := range features {
		if strings.TrimSpace(feature) == "" {
			return fmt.Errorf("feature %d cannot be empty", i+1)
		}

		if len(feature) > 100 {
			return fmt.Errorf("feature %d is too long (maximum 100 characters)", i+1)
		}
	}

	return nil
}

// validateDocumentation validates documentation URL
func (v *EnhancedApplicationValidator) validateDocumentation(documentation string) error {
	if strings.TrimSpace(documentation) == "" {
		return fmt.Errorf("documentation URL cannot be empty")
	}

	if _, err := url.Parse(documentation); err != nil {
		return fmt.Errorf("invalid documentation URL: %w", err)
	}

	// Ensure it's a valid HTTP/HTTPS URL
	if !strings.HasPrefix(documentation, "http://") && !strings.HasPrefix(documentation, "https://") {
		return fmt.Errorf("documentation URL must start with http:// or https://")
	}

	return nil
}

// validateChartBasics validates basic Helm chart fields
func (v *EnhancedApplicationValidator) validateChartBasics(chart models.HelmChartConfig) error {
	if strings.TrimSpace(chart.Chart) == "" {
		return fmt.Errorf("helm chart name cannot be empty")
	}

	if strings.TrimSpace(chart.Namespace) == "" {
		return fmt.Errorf("helm chart namespace cannot be empty")
	}

	return nil
}

// validateRepository validates Helm repository URL
func (v *EnhancedApplicationValidator) validateRepository(repository string) error {
	if strings.TrimSpace(repository) == "" {
		return fmt.Errorf("helm repository URL cannot be empty")
	}

	if _, err := url.Parse(repository); err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	// Ensure it's a valid HTTP/HTTPS URL or Git repository
	if !strings.HasPrefix(repository, "http://") && 
	   !strings.HasPrefix(repository, "https://") && 
	   !strings.HasPrefix(repository, "git@") {
		return fmt.Errorf("repository URL must be HTTP, HTTPS, or Git SSH format")
	}

	return nil
}

// validateNamespace validates Kubernetes namespace
func (v *EnhancedApplicationValidator) validateNamespace(namespace string) error {
	// Kubernetes namespace naming rules
	validNamespace := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validNamespace.MatchString(namespace) {
		return fmt.Errorf("invalid namespace '%s' (must contain only lowercase letters, numbers, and hyphens, and start/end with alphanumeric character)", namespace)
	}

	if len(namespace) > 63 {
		return fmt.Errorf("namespace '%s' is too long (maximum 63 characters)", namespace)
	}

	// Reserved namespaces
	reservedNamespaces := []string{"kube-system", "kube-public", "kube-node-lease", "default"}
	for _, reserved := range reservedNamespaces {
		if namespace == reserved {
			return fmt.Errorf("namespace '%s' is reserved and cannot be used", namespace)
		}
	}

	return nil
}

// validateVersion validates chart version
func (v *EnhancedApplicationValidator) validateVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("helm chart version cannot be empty")
	}

	// Allow semantic versioning or branch names like "main", "master"
	if version != "main" && version != "master" && version != "latest" {
		// Basic semantic version pattern
		semverPattern := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
		if !semverPattern.MatchString(version) {
			return fmt.Errorf("invalid version '%s' (must be semantic version like '1.0.0' or branch name like 'main')", version)
		}
	}

	return nil
}

// validateResourceRequirements validates requirements against cluster capacity
func (v *EnhancedApplicationValidator) validateResourceRequirements(req models.ApplicationRequirements, cluster ClusterInfo) error {
	if req.MinCPU > cluster.AvailableCPU {
		return fmt.Errorf("application requires %.2f CPU cores but cluster only has %.2f available", req.MinCPU, cluster.AvailableCPU)
	}

	if req.MinMemory > cluster.AvailableMemoryGB {
		return fmt.Errorf("application requires %d GB memory but cluster only has %d GB available", req.MinMemory, cluster.AvailableMemoryGB)
	}

	if req.MinDisk > cluster.AvailableDiskGB {
		return fmt.Errorf("application requires %d GB disk but cluster only has %d GB available", req.MinDisk, cluster.AvailableDiskGB)
	}

	return nil
}

// validateKubernetesCompatibility validates Kubernetes version compatibility
func (v *EnhancedApplicationValidator) validateKubernetesCompatibility(app models.PredefinedApplication, cluster ClusterInfo) error {
	// Basic cluster health checks
	if cluster.NodeCount == 0 {
		return fmt.Errorf("cluster has no available nodes")
	}

	if strings.TrimSpace(cluster.KubernetesVersion) == "" {
		return fmt.Errorf("cluster Kubernetes version is unknown")
	}

	if strings.TrimSpace(cluster.HelmVersion) == "" {
		return fmt.Errorf("cluster Helm version is unknown")
	}

	// Could add specific version compatibility checks here
	// For now, just ensure basic requirements are met

	return nil
}

// validateConfigFields validates fields specific to configuration-loaded applications
func (v *EnhancedApplicationValidator) validateConfigFields(app models.PredefinedApplication) error {
	// Additional validation for config-loaded applications
	// This could include checks for required configuration fields,
	// validation of custom properties, etc.

	// Check default port if specified
	if app.DefaultPort > 0 {
		if app.DefaultPort < 1 || app.DefaultPort > 65535 {
			return fmt.Errorf("invalid default port %d (must be between 1 and 65535)", app.DefaultPort)
		}

		// Warn about well-known ports
		if app.DefaultPort < 1024 {
			// This is just a warning for now, could be made configurable
			// return fmt.Errorf("default port %d is a well-known port (consider using port >= 1024)", app.DefaultPort)
		}
	}

	return nil
}