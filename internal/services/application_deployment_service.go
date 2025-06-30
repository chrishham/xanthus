package services

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/utils"
)

// ApplicationDeploymentService handles application deployment operations
type ApplicationDeploymentService struct{}

// NewApplicationDeploymentService creates a new ApplicationDeploymentService
func NewApplicationDeploymentService() *ApplicationDeploymentService {
	return &ApplicationDeploymentService{}
}

// DeployApplication deploys an application using Helm and appropriate handlers
func (ads *ApplicationDeploymentService) DeployApplication(token, accountID string, appData interface{}, predefinedApp *models.PredefinedApplication, appID string) error {
	log.Printf("Deploying application %s with type %s", appID, predefinedApp.ID)

	// Convert appData to map for easier access
	appDataMap, ok := appData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid application data format")
	}

	kvService := NewKVService()
	sshService := NewSSHService()

	subdomain := appDataMap["subdomain"].(string)
	domain := appDataMap["domain"].(string)
	vpsID := appDataMap["vps_id"].(string)

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err := kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", vpsID), &vpsConfig)
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

	// Create SSH connection
	conn, err := sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Generate release name and namespace (using type-based namespace as per CLAUDE.md)
	// Release name starts with subdomain as specified in requirements
	releaseName := fmt.Sprintf("%s-%s", subdomain, predefinedApp.ID)
	namespace := predefinedApp.ID

	// Create namespace if it doesn't exist
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -", namespace))
	if err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	var chartName string

	// Handle different chart repository types based on HelmChart configuration
	helmConfig := predefinedApp.HelmChart

	if strings.Contains(helmConfig.Repository, "github.com") {
		// Clone GitHub repository for the chart
		repoDir := fmt.Sprintf("/tmp/%s-chart", predefinedApp.ID)
		_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("rm -rf %s && git clone %s %s", repoDir, helmConfig.Repository, repoDir))
		if err != nil {
			return fmt.Errorf("failed to clone chart repository: %v", err)
		}
		chartName = fmt.Sprintf("%s/%s", repoDir, helmConfig.Chart)
	} else {
		// Add Helm repository
		repoName := predefinedApp.ID
		_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("helm repo add %s %s", repoName, helmConfig.Repository))
		if err != nil {
			return fmt.Errorf("failed to add Helm repository: %v", err)
		}

		// Update Helm repositories
		_, err = sshService.ExecuteCommand(conn, "helm repo update")
		if err != nil {
			return fmt.Errorf("failed to update Helm repositories: %v", err)
		}
		chartName = fmt.Sprintf("%s/%s", repoName, helmConfig.Chart)
	}

	// Generate and upload values file
	valuesContent, err := ads.generateValuesFromTemplate(predefinedApp, predefinedApp.Version, subdomain, domain, releaseName)
	if err != nil {
		return fmt.Errorf("failed to generate values file: %v", err)
	}

	valuesPath := fmt.Sprintf("/tmp/%s-values.yaml", releaseName)
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", valuesPath, valuesContent))
	if err != nil {
		return fmt.Errorf("failed to upload values file: %v", err)
	}

	// Install via Helm
	installCmd := fmt.Sprintf("helm install %s %s --namespace %s --values %s --wait --timeout 10m",
		releaseName, chartName, namespace, valuesPath)

	result, err := sshService.ExecuteCommand(conn, installCmd)
	if err != nil {
		return fmt.Errorf("helm install failed: %v, output: %s", err, result.Output)
	}

	// Retrieve and store password for applications that generate them
	if err := ads.retrieveApplicationPassword(token, accountID, predefinedApp.ID, appID, vpsID, releaseName, namespace, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey); err != nil {
		// Log the error but don't fail the deployment since the app is successfully deployed
		log.Printf("Warning: Failed to retrieve application password: %v", err)
	}

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
	// Release name starts with subdomain as specified in requirements
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.AppType)
	namespace := app.AppType // Use type-based namespace as per CLAUDE.md

	// Generate updated values file with new version
	valuesContent, err := ads.generateValuesFromTemplate(predefinedApp, app.AppVersion, app.Subdomain, app.Domain, releaseName)
	if err != nil {
		return fmt.Errorf("failed to generate values file: %v", err)
	}

	// Establish SSH connection
	sshService := NewSSHService()
	conn, err := sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Handle chart repository setup (same as deployment)
	var chartName string
	helmConfig := predefinedApp.HelmChart

	if strings.Contains(helmConfig.Repository, "github.com") {
		// Clone GitHub repository for the chart
		repoDir := fmt.Sprintf("/tmp/%s-chart", predefinedApp.ID)
		_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("rm -rf %s && git clone %s %s", repoDir, helmConfig.Repository, repoDir))
		if err != nil {
			return fmt.Errorf("failed to clone chart repository: %v", err)
		}
		chartName = fmt.Sprintf("%s/%s", repoDir, helmConfig.Chart)
	} else {
		// Add/update Helm repository
		repoName := predefinedApp.ID
		_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("helm repo add %s %s", repoName, helmConfig.Repository))
		if err != nil {
			// If repo add fails, it might already exist, try to update
			_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("helm repo update %s", repoName))
			if err != nil {
				return fmt.Errorf("failed to add/update Helm repository: %v", err)
			}
		} else {
			// Update Helm repositories after successful add
			_, err = sshService.ExecuteCommand(conn, "helm repo update")
			if err != nil {
				return fmt.Errorf("failed to update Helm repositories: %v", err)
			}
		}
		chartName = fmt.Sprintf("%s/%s", repoName, helmConfig.Chart)
	}

	// Upload values file to VPS
	valuesPath := fmt.Sprintf("/tmp/%s-values.yaml", releaseName)
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", valuesPath, valuesContent))
	if err != nil {
		return fmt.Errorf("failed to upload values file: %v", err)
	}

	// Perform Helm upgrade
	helmService := NewHelmService()

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

// retrieveApplicationPassword retrieves and stores the auto-generated password for applications that create them
func (ads *ApplicationDeploymentService) retrieveApplicationPassword(token, accountID, appType, appID, vpsID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	switch appType {
	case "code-server":
		return ads.retrieveCodeServerPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey)
	case "argocd":
		return ads.retrieveArgoCDPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey)
	default:
		// No password retrieval needed for this application type
		return nil
	}
}

// retrieveCodeServerPassword retrieves the auto-generated password from code-server Kubernetes secret
func (ads *ApplicationDeploymentService) retrieveCodeServerPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := NewSSHService()

	conn, err := sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Retrieve password from Kubernetes secret
	// The secret name is the same as the release name for code-server
	secretName := releaseName
	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' | base64 --decode", namespace, secretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to retrieve password from secret '%s' in namespace '%s': %v", secretName, namespace, err)
	}

	// Store password in KV store
	password := strings.TrimSpace(result.Output)
	return ads.storeEncryptedPassword(token, accountID, appID, password)
}

// retrieveArgoCDPassword retrieves the auto-generated admin password from ArgoCD
func (ads *ApplicationDeploymentService) retrieveArgoCDPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := NewSSHService()

	conn, err := sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Retrieve admin password from ArgoCD initial admin secret
	secretName := "argocd-initial-admin-secret"
	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", namespace, secretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to retrieve password from secret '%s' in namespace '%s': %v", secretName, namespace, err)
	}

	password := strings.TrimSpace(result.Output)
	if password == "" {
		return fmt.Errorf("retrieved empty password from ArgoCD secret")
	}

	// Store password in KV store
	return ads.storeEncryptedPassword(token, accountID, appID, password)
}

// storeEncryptedPassword encrypts and stores the password in KV store
func (ads *ApplicationDeploymentService) storeEncryptedPassword(token, accountID, appID, password string) error {
	kvService := NewKVService()

	// Encrypt the password
	encryptedPassword, err := utils.EncryptData(password, token)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}

	// Store in KV with the key format: app:{appID}:password
	// Use the same format as PasswordHelper.StoreEncryptedPassword
	key := fmt.Sprintf("app:%s:password", appID)
	err = kvService.PutValue(token, accountID, key, map[string]string{
		"password": encryptedPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to store encrypted password: %v", err)
	}

	return nil
}
