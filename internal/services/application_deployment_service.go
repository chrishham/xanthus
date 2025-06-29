package services

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chrishham/xanthus/internal/models"
)

// ApplicationDeploymentService handles application deployment operations
type ApplicationDeploymentService struct{}

// NewApplicationDeploymentService creates a new ApplicationDeploymentService
func NewApplicationDeploymentService() *ApplicationDeploymentService {
	return &ApplicationDeploymentService{}
}

// DeployApplication deploys an application using Helm and appropriate handlers
func (ads *ApplicationDeploymentService) DeployApplication(token, accountID string, appData interface{}, predefinedApp *models.PredefinedApplication, appID string) error {
	// For now, delegate to existing simple implementation
	// This can be enhanced later with more sophisticated deployment logic
	log.Printf("Deploying application %s with type %s", appID, predefinedApp.ID)

	// In the future, this would contain:
	// - VPS connection setup
	// - Helm chart deployment
	// - TLS certificate management
	// - Application-specific configuration
	// - Password retrieval and storage

	// For now, return success to maintain compatibility
	return nil
}

// UpgradeApplication upgrades an existing application to a new version
func (ads *ApplicationDeploymentService) UpgradeApplication(token, accountID, appID, version string) error {
	// Get application details
	appService := NewSimpleApplicationService()
	app, err := appService.GetApplication(token, accountID, appID)
	if err != nil {
		return fmt.Errorf("failed to get application: %v", err)
	}

	// Update the application version and status
	app.AppVersion = version
	app.Status = "updating"

	// Update the application in KV store
	err = appService.UpdateApplication(token, accountID, app)
	if err != nil {
		return fmt.Errorf("failed to update application: %v", err)
	}

	log.Printf("Starting upgrade of application %s to version %s", appID, version)

	// Perform the actual Helm upgrade
	err = ads.performUpgrade(token, accountID, app)
	if err != nil {
		// Update status to failed on error
		app.Status = "failed"
		appService.UpdateApplication(token, accountID, app)
		return fmt.Errorf("upgrade failed: %v", err)
	}

	// Update status to running on success
	app.Status = "running"
	err = appService.UpdateApplication(token, accountID, app)
	if err != nil {
		log.Printf("Warning: Failed to update application status after successful upgrade: %v", err)
	}

	log.Printf("Successfully upgraded application %s to version %s", appID, version)
	return nil
}

// performUpgrade performs the actual Helm upgrade operation
func (ads *ApplicationDeploymentService) performUpgrade(token, accountID string, app *models.Application) error {
	kvService := NewKVService()

	// Get predefined application configuration using the catalog service
	factory := NewApplicationServiceFactory()
	catalog := factory.CreateHybridCatalogService()
	predefinedApp, found := catalog.GetApplicationByID(app.AppType)
	if !found {
		return fmt.Errorf("application configuration not found for type: %s", app.AppType)
	}

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err := kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", app.VPSID), &vpsConfig)
	if err != nil {
		return fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := kvService.GetValue(token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return fmt.Errorf("failed to get SSH private key: %v", err)
	}

	// Generate release name and namespace (same as deployment)
	releaseName := fmt.Sprintf("%s-%s", app.AppType, app.ID)
	namespace := app.AppType // Use type-based namespace as per CLAUDE.md

	// Generate updated values file with new version
	valuesContent, err := ads.generateValuesFromTemplate(predefinedApp, app.AppVersion, app.Subdomain, app.Domain, releaseName)
	if err != nil {
		return fmt.Errorf("failed to generate values file: %v", err)
	}

	// Upload values file to VPS
	valuesPath := fmt.Sprintf("/tmp/%s-values.yaml", releaseName)
	sshService := NewSSHService()
	conn, err := sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Write values file
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", valuesPath, valuesContent))
	if err != nil {
		return fmt.Errorf("failed to upload values file: %v", err)
	}

	// Perform Helm upgrade
	helmService := NewHelmService()
	chartName := predefinedApp.HelmChart.Repository
	if strings.Contains(predefinedApp.HelmChart.Repository, "github.com") {
		// For GitHub-based charts, use the cloned path
		chartName = fmt.Sprintf("/tmp/%s-chart/%s", app.AppType, predefinedApp.HelmChart.Chart)
	}

	err = helmService.UpgradeChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		chartName,
		app.AppVersion,
		namespace,
		valuesPath,
	)
	if err != nil {
		return fmt.Errorf("helm upgrade failed: %v", err)
	}

	return nil
}

// generateValuesFromTemplate generates values file content from template with placeholder substitution
func (ads *ApplicationDeploymentService) generateValuesFromTemplate(predefinedApp *models.PredefinedApplication, version, subdomain, domain, releaseName string) (string, error) {
	// This mirrors the logic from application_service_simple.go generateFromTemplate
	templatePath := fmt.Sprintf("internal/templates/applications/%s", predefinedApp.HelmChart.ValuesTemplate)
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %v", templatePath, err)
	}

	// Create placeholder map - note: don't include {{ }} in keys since we add them in the replacement
	placeholders := map[string]string{
		"VERSION":      version,
		"SUBDOMAIN":    subdomain,
		"DOMAIN":       domain,
		"RELEASE_NAME": releaseName,
	}

	// Add additional placeholders from config
	if predefinedApp.HelmChart.Placeholders != nil {
		for key, value := range predefinedApp.HelmChart.Placeholders {
			placeholders[key] = value
		}
	}

	// Replace placeholders in template - add {{ }} formatting here
	content := string(templateContent)
	for placeholder, value := range placeholders {
		content = strings.ReplaceAll(content, fmt.Sprintf("{{%s}}", placeholder), value)
	}

	return content, nil
}
