package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/utils"
)

// SimpleApplicationService provides core CRUD operations for applications using existing services
type SimpleApplicationService struct{}

// NewSimpleApplicationService creates a new SimpleApplicationService
func NewSimpleApplicationService() *SimpleApplicationService {
	return &SimpleApplicationService{}
}

// ListApplications returns all applications for the given account with real-time status updates
func (s *SimpleApplicationService) ListApplications(token, accountID string) ([]models.Application, error) {
	kvService := NewKVService()

	// Get the Xanthus namespace ID
	namespaceID, err := kvService.GetXanthusNamespaceID(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace ID: %w", err)
	}

	// List all keys with app: prefix
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/keys?prefix=app:",
		accountID, namespaceID)
	fmt.Printf("Listing keys from KV with URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var keysResp struct {
		Success bool `json:"success"`
		Result  []struct {
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keysResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !keysResp.Success {
		return nil, fmt.Errorf("KV API failed")
	}

	fmt.Printf("Found %d keys with app: prefix\n", len(keysResp.Result))
	for i, key := range keysResp.Result {
		fmt.Printf("Key %d: %s\n", i, key.Name)
	}

	applications := []models.Application{}

	// Fetch each application, but skip password keys
	for _, key := range keysResp.Result {
		// Skip password keys (they end with ":password")
		if strings.HasSuffix(key.Name, ":password") {
			fmt.Printf("Skipping password key: %s\n", key.Name)
			continue
		}

		fmt.Printf("Attempting to retrieve application: %s\n", key.Name)
		var app models.Application
		if err := kvService.GetValue(token, accountID, key.Name, &app); err == nil {
			fmt.Printf("Successfully retrieved application: %s (ID: %s, Name: %s, VPSName: %s, AppType: %s, Status: %s, URL: %s)\n",
				key.Name, app.ID, app.Name, app.VPSName, app.AppType, app.Status, app.URL)
			// Update application status with real-time Helm status
			if realTimeStatus, statusErr := s.GetApplicationRealTimeStatus(token, accountID, &app); statusErr == nil {
				app.Status = realTimeStatus
				fmt.Printf("Updated status for %s: %s\n", app.ID, realTimeStatus)
			} else {
				fmt.Printf("Could not get real-time status for %s: %v\n", app.ID, statusErr)
			}
			// If we can't get real-time status, keep the cached status
			app.UpdatedAt = time.Now().Format(time.RFC3339)

			applications = append(applications, app)
			fmt.Printf("Added application to list: %s\n", app.ID)
		} else {
			// Log error to help debug issues
			fmt.Printf("Error retrieving application %s: %v\n", key.Name, err)
		}
	}

	return applications, nil
}

// GetApplication returns a specific application by ID
func (s *SimpleApplicationService) GetApplication(token, accountID, appID string) (*models.Application, error) {
	applications, err := s.ListApplications(token, accountID)
	if err != nil {
		return nil, err
	}

	for _, app := range applications {
		if app.ID == appID {
			return &app, nil
		}
	}

	return nil, fmt.Errorf("application not found: %s", appID)
}

// CreateApplication creates a new application
func (s *SimpleApplicationService) CreateApplication(token, accountID string, appData interface{}, predefinedApp *models.PredefinedApplication) (*models.Application, error) {
	// Parse application data based on type
	var subdomain, domain, vpsID, vpsName, description string

	switch data := appData.(type) {
	case map[string]interface{}:
		if sub, ok := data["subdomain"].(string); ok {
			subdomain = sub
		}
		if dom, ok := data["domain"].(string); ok {
			domain = dom
		}
		if vps, ok := data["vps_id"].(string); ok {
			vpsID = vps
		}
		if name, ok := data["vps_name"].(string); ok {
			vpsName = name
		}
		if desc, ok := data["description"].(string); ok {
			description = desc
		}
	default:
		return nil, fmt.Errorf("invalid application data format")
	}

	// Generate application ID first
	appID := fmt.Sprintf("app-%d", time.Now().Unix())

	// Check for existing ArgoCD installation on this VPS before creating the application
	if predefinedApp.ID == "argocd" {
		if err := s.checkExistingArgoCDInstallation(token, accountID, vpsID, appID); err != nil {
			return nil, err
		}
	}

	// Create namespace based on application type
	namespace := predefinedApp.ID

	// Create application model
	app := &models.Application{
		ID:          appID,
		Name:        subdomain,
		Description: description,
		AppType:     predefinedApp.ID,
		AppVersion:  predefinedApp.Version,
		Subdomain:   subdomain,
		Domain:      domain,
		VPSID:       vpsID,
		VPSName:     vpsName,
		Namespace:   namespace,
		Status:      "Creating",
		URL:         fmt.Sprintf("https://%s.%s", subdomain, domain),
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	// Save individual application to KV store with app: prefix
	kvService := NewKVService()
	kvKey := fmt.Sprintf("app:%s", appID)
	fmt.Printf("Saving application with key: %s, appID: %s\n", kvKey, appID)
	if err := kvService.PutValue(token, accountID, kvKey, app); err != nil {
		fmt.Printf("Failed to save application to KV: %v\n", err)
		return nil, fmt.Errorf("failed to save application: %w", err)
	}
	fmt.Printf("Successfully saved application to KV\n")

	// Deploy the application using Helm
	fmt.Printf("Starting deployment for application %s\n", appID)
	// Convert back to map for deployment
	dataMap := appData.(map[string]interface{})
	err := s.deployApplication(token, accountID, dataMap, predefinedApp, appID)
	if err != nil {
		fmt.Printf("Deployment failed for %s: %v\n", appID, err)
		app.Status = "Failed"
		app.ErrorMsg = err.Error() // Store the detailed error message including resource exhaustion info
	} else {
		fmt.Printf("Deployment successful for %s\n", appID)
		app.Status = "Running"
		app.ErrorMsg = "" // Clear any previous error messages
	}

	// Update application status
	app.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := kvService.PutValue(token, accountID, kvKey, app); err != nil {
		fmt.Printf("Warning: Failed to update application status: %v\n", err)
	}

	return app, nil
}

// UpdateApplication updates an existing application
func (s *SimpleApplicationService) UpdateApplication(token, accountID string, app *models.Application) error {
	// Update timestamp
	app.UpdatedAt = time.Now().Format(time.RFC3339)

	// Save individual application to KV store with app: prefix
	kvService := NewKVService()
	kvKey := fmt.Sprintf("app:%s", app.ID)
	if err := kvService.PutValue(token, accountID, kvKey, app); err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	return nil
}

// DeleteApplication deletes an application and cleans up all resources
func (s *SimpleApplicationService) DeleteApplication(token, accountID, appID string) error {
	kvService := NewKVService()

	// First, get the application details before deletion
	app, err := s.GetApplication(token, accountID, appID)
	if err != nil {
		return fmt.Errorf("failed to get application details: %w", err)
	}

	// Delete Helm deployment from VPS
	if err := s.deleteApplicationDeployment(token, accountID, app); err != nil {
		fmt.Printf("Warning: Failed to delete Helm deployment for %s: %v\n", appID, err)
		// Continue with cleanup even if Helm deletion fails
	}

	// Delete DNS A record from Cloudflare
	if err := s.deleteApplicationDNS(token, app); err != nil {
		fmt.Printf("Warning: Failed to delete DNS record for %s: %v\n", appID, err)
		// Continue with cleanup even if DNS deletion fails
	}

	// Delete the main application key from KV
	kvKey := fmt.Sprintf("app:%s", appID)
	if err := kvService.DeleteValue(token, accountID, kvKey); err != nil {
		return fmt.Errorf("failed to delete application from KV: %w", err)
	}

	// Also delete the password key if it exists
	passwordKey := fmt.Sprintf("app:%s:password", appID)
	kvService.DeleteValue(token, accountID, passwordKey) // Ignore error - password key might not exist

	fmt.Printf("Successfully deleted application %s and cleaned up resources\n", appID)
	return nil
}

// deleteApplicationDeployment removes the Helm deployment from the VPS
func (s *SimpleApplicationService) deleteApplicationDeployment(token, accountID string, app *models.Application) error {
	if app.VPSID == "" {
		return fmt.Errorf("no VPS ID associated with application")
	}

	sshService := NewSSHService()

	// Get VPS configuration
	serverID, _ := strconv.Atoi(app.VPSID)
	vpsService := NewVPSService()
	vpsConfig, err := vpsService.ValidateVPSAccess(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Get SSH private key from KV
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return fmt.Errorf("failed to get SSH key: %w", err)
	}

	// Establish SSH connection
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, serverID)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %w", err)
	}

	// Uninstall Helm release
	// Release name starts with subdomain as specified in requirements
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.AppType)
	uninstallCmd := fmt.Sprintf("helm uninstall %s --namespace %s", releaseName, app.Namespace)

	result, err := sshService.ExecuteCommand(conn, uninstallCmd)
	if err != nil {
		return fmt.Errorf("failed to uninstall Helm release: %v, output: %s", err, result.Output)
	}

	// ArgoCD-specific cleanup: remove cluster-wide resources
	if app.AppType == "argocd" {
		if err := s.cleanupArgoCDResources(sshService, conn, app.Namespace); err != nil {
			fmt.Printf("Warning: Failed to clean up ArgoCD cluster resources: %v\n", err)
			// Continue with deletion even if cluster cleanup fails
		}
	}

	fmt.Printf("Successfully uninstalled Helm release %s from namespace %s\n", releaseName, app.Namespace)
	return nil
}

// deleteApplicationDNS removes the DNS A record from Cloudflare
func (s *SimpleApplicationService) deleteApplicationDNS(token string, app *models.Application) error {
	cfService := NewCloudflareService()

	// Get zone ID for the domain
	zoneID, err := cfService.GetZoneID(token, app.Domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID for domain %s: %w", app.Domain, err)
	}

	// Get existing DNS records
	records, err := cfService.GetDNSRecords(token, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get DNS records: %w", err)
	}

	// Find and delete the A record for this application
	var recordName string
	if app.Subdomain == "" || app.Subdomain == "*" {
		recordName = app.Domain
	} else {
		recordName = fmt.Sprintf("%s.%s", app.Subdomain, app.Domain)
	}

	recordsDeleted := 0
	for _, record := range records {
		// Normalize record name (remove trailing dot if present)
		normalizedRecordName := strings.TrimSuffix(record.Name, ".")

		// Check if this is the A record for our application
		if record.Type == "A" && (normalizedRecordName == recordName || record.Name == recordName) {
			fmt.Printf("Deleting DNS A record: %s -> %s (ID: %s)\n", record.Name, record.Content, record.ID)
			if err := cfService.DeleteDNSRecord(token, zoneID, record.ID); err != nil {
				return fmt.Errorf("failed to delete DNS record %s: %w", record.Name, err)
			}
			recordsDeleted++
		}
	}

	if recordsDeleted == 0 {
		fmt.Printf("Warning: No DNS A record found for %s\n", recordName)
	} else {
		fmt.Printf("Successfully deleted %d DNS record(s) for %s\n", recordsDeleted, recordName)
	}

	return nil
}

// GetApplicationRealTimeStatus fetches current deployment status from Helm
func (s *SimpleApplicationService) GetApplicationRealTimeStatus(token, accountID string, app *models.Application) (string, error) {
	if app.VPSID == "" {
		return "Unknown", nil
	}

	// Use existing VPS service
	vpsService := NewVPSService()
	sshService := NewSSHService()

	// Get VPS configuration
	serverID, _ := strconv.Atoi(app.VPSID)
	vpsConfig, err := vpsService.ValidateVPSAccess(token, accountID, serverID)
	if err != nil {
		return "Unknown", fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Get SSH private key from KV
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return "Unknown", fmt.Errorf("failed to get SSH key: %w", err)
	}

	// Establish SSH connection
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, serverID)
	if err != nil {
		return "Unknown", fmt.Errorf("failed to connect to VPS: %w", err)
	}

	// Check Helm deployment status
	// Release name starts with subdomain as specified in requirements
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.AppType)
	statusCmd := fmt.Sprintf("helm status %s -n %s --output json 2>/dev/null || echo '{\"info\":{\"status\":\"not-found\"}}'",
		releaseName, app.Namespace)

	result, err := sshService.ExecuteCommand(conn, statusCmd)
	if err != nil {
		return "Unknown", nil
	}
	output := result.Output

	// Parse Helm status
	var helmStatus struct {
		Info struct {
			Status string `json:"status"`
		} `json:"info"`
	}

	if err := json.Unmarshal([]byte(output), &helmStatus); err != nil {
		return "Unknown", nil
	}

	// Map Helm status to application status
	switch strings.ToLower(helmStatus.Info.Status) {
	case "deployed":
		return "Running", nil
	case "failed":
		return "Failed", nil
	case "pending-install", "pending-upgrade":
		return "Deploying", nil
	case "not-found":
		return "Not Deployed", nil
	default:
		return helmStatus.Info.Status, nil
	}
}

// checkExistingArgoCDInstallation checks if there's already an ArgoCD installation on the VPS
func (s *SimpleApplicationService) checkExistingArgoCDInstallation(token, accountID, vpsID, excludeAppID string) error {
	kvService := NewKVService()

	// Get all applications from KV store
	applications, err := s.getAllApplications(token, accountID, kvService)
	if err != nil {
		return fmt.Errorf("failed to check existing applications: %v", err)
	}

	// Check if any existing application is ArgoCD on the same VPS
	for _, app := range applications {
		// Skip the current application being created
		if app.ID == excludeAppID {
			continue
		}
		// Check if it's an ArgoCD application on the same VPS (ignore failed/not deployed)
		if app.AppType == "argocd" && app.VPSID == vpsID && (app.Status == "Running" || app.Status == "Creating" || app.Status == "Deploying") {
			return fmt.Errorf("ArgoCD is already installed on this VPS (application: %s, subdomain: %s.%s). Only one ArgoCD installation is allowed per VPS due to cluster-wide resources. Please use the existing ArgoCD instance or choose a different VPS", app.Name, app.Subdomain, app.Domain)
		}
	}

	return nil
}

// getAllApplications retrieves all applications from KV store
func (s *SimpleApplicationService) getAllApplications(token, accountID string, kvService *KVService) ([]models.Application, error) {
	// Get the Xanthus namespace ID
	namespaceID, err := kvService.GetXanthusNamespaceID(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace ID: %w", err)
	}

	// List all keys with app: prefix
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/keys?prefix=app:",
		accountID, namespaceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var keysResp struct {
		Success bool `json:"success"`
		Result  []struct {
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keysResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !keysResp.Success {
		return nil, fmt.Errorf("KV API failed")
	}

	applications := []models.Application{}

	// Fetch each application, but skip password keys
	for _, key := range keysResp.Result {
		// Skip password keys (they end with ":password")
		if strings.HasSuffix(key.Name, ":password") {
			continue
		}

		var app models.Application
		if err := kvService.GetValue(token, accountID, key.Name, &app); err == nil {
			applications = append(applications, app)
		}
	}

	return applications, nil
}

// cleanupArgoCDResources removes ArgoCD cluster-wide resources to prevent deployment conflicts
func (s *SimpleApplicationService) cleanupArgoCDResources(sshService *SSHService, conn *SSHConnection, namespace string) error {
	fmt.Printf("🧹 Starting ArgoCD cluster-wide resource cleanup...\n")

	var cleanupErrors []string

	// 1. Delete ArgoCD Custom Resource Definitions (CRDs)
	fmt.Printf("🧹 Cleaning up ArgoCD CRDs...\n")
	crdCommands := []string{
		"kubectl delete crd applications.argoproj.io --ignore-not-found=true",
		"kubectl delete crd applicationsets.argoproj.io --ignore-not-found=true", 
		"kubectl delete crd appprojects.argoproj.io --ignore-not-found=true",
	}

	for _, cmd := range crdCommands {
		_, err := sshService.ExecuteCommand(conn, cmd)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete CRD: %v", err))
		}
	}

	// 2. Delete ArgoCD ClusterRoles and ClusterRoleBindings
	fmt.Printf("🧹 Cleaning up ArgoCD cluster roles...\n")
	clusterRoleCommands := []string{
		"kubectl delete clusterrole argocd-application-controller --ignore-not-found=true",
		"kubectl delete clusterrole argocd-server --ignore-not-found=true",
		"kubectl delete clusterrolebinding argocd-application-controller --ignore-not-found=true",
		"kubectl delete clusterrolebinding argocd-server --ignore-not-found=true",
	}

	for _, cmd := range clusterRoleCommands {
		_, err := sshService.ExecuteCommand(conn, cmd)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete cluster role: %v", err))
		}
	}

	// 3. Delete any remaining ArgoCD-related secrets in the namespace
	fmt.Printf("🧹 Cleaning up ArgoCD secrets in namespace %s...\n", namespace)
	secretCommands := []string{
		fmt.Sprintf("kubectl delete secret argocd-initial-admin-secret -n %s --ignore-not-found=true", namespace),
		fmt.Sprintf("kubectl delete secret argocd-redis -n %s --ignore-not-found=true", namespace),
		fmt.Sprintf("kubectl delete secret argocd-secret -n %s --ignore-not-found=true", namespace),
	}

	for _, cmd := range secretCommands {
		_, err := sshService.ExecuteCommand(conn, cmd)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete secret: %v", err))
		}
	}

	// 4. Delete ArgoCD ConfigMaps
	fmt.Printf("🧹 Cleaning up ArgoCD configmaps in namespace %s...\n", namespace)
	configMapCommands := []string{
		fmt.Sprintf("kubectl delete configmap argocd-cm -n %s --ignore-not-found=true", namespace),
		fmt.Sprintf("kubectl delete configmap argocd-cmd-params-cm -n %s --ignore-not-found=true", namespace),
		fmt.Sprintf("kubectl delete configmap argocd-rbac-cm -n %s --ignore-not-found=true", namespace),
	}

	for _, cmd := range configMapCommands {
		_, err := sshService.ExecuteCommand(conn, cmd)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete configmap: %v", err))
		}
	}

	// 5. Force delete any remaining ArgoCD pods
	fmt.Printf("🧹 Force deleting remaining ArgoCD pods in namespace %s...\n", namespace)
	_, err := sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl delete pods -l app.kubernetes.io/part-of=argocd -n %s --ignore-not-found=true --force --grace-period=0", namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete ArgoCD pods: %v", err))
	}

	// 6. Clean up any ArgoCD-related ValidatingAdmissionWebhooks and MutatingAdmissionWebhooks
	fmt.Printf("🧹 Cleaning up ArgoCD admission webhooks...\n")
	webhookCommands := []string{
		"kubectl delete validatingadmissionwebhook argocd-notifications-webhook --ignore-not-found=true",
		"kubectl delete mutatingadmissionwebhook argocd-notifications-webhook --ignore-not-found=true",
	}

	for _, cmd := range webhookCommands {
		_, err := sshService.ExecuteCommand(conn, cmd)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete webhook: %v", err))
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("ArgoCD cleanup completed with errors: %s", strings.Join(cleanupErrors, "; "))
	}

	fmt.Printf("✅ Successfully cleaned up all ArgoCD cluster-wide resources\n")
	return nil
}
