package models

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
	// Add validation logic here as needed
	// For now, this is a placeholder for future validation
	return nil
}

// ValidateHelmChart validates Helm chart configuration
func (v *DefaultApplicationValidator) ValidateHelmChart(chart HelmChartConfig) error {
	// Add validation logic here as needed
	// For now, this is a placeholder for future validation
	return nil
}

// CheckApplicationRequirements checks if system meets application requirements
func CheckApplicationRequirements(app PredefinedApplication, availableCPU float64, availableMemoryGB int, availableDiskGB int) bool {
	return availableCPU >= app.Requirements.MinCPU &&
		availableMemoryGB >= app.Requirements.MinMemory &&
		availableDiskGB >= app.Requirements.MinDisk
}