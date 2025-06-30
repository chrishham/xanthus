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

	// Generate application ID
	appID := fmt.Sprintf("app-%d", time.Now().Unix())

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
	} else {
		fmt.Printf("Deployment successful for %s\n", appID)
		app.Status = "Running"
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
	releaseName := fmt.Sprintf("%s-%s", app.AppType, app.ID)
	uninstallCmd := fmt.Sprintf("helm uninstall %s --namespace %s", releaseName, app.Namespace)

	result, err := sshService.ExecuteCommand(conn, uninstallCmd)
	if err != nil {
		return fmt.Errorf("failed to uninstall Helm release: %v, output: %s", err, result.Output)
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
	releaseName := fmt.Sprintf("%s-%s", app.AppType, app.ID)
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
