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
			// Update application status with real-time Helm status
			if realTimeStatus, statusErr := s.GetApplicationRealTimeStatus(token, accountID, &app); statusErr == nil {
				app.Status = realTimeStatus
			}
			// If we can't get real-time status, keep the cached status
			app.UpdatedAt = time.Now().Format(time.RFC3339)

			applications = append(applications, app)
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
	
	// Create namespace from subdomain
	namespace := fmt.Sprintf("app-%s", strings.ReplaceAll(subdomain, ".", "-"))
	
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
	if err := kvService.PutValue(token, accountID, kvKey, app); err != nil {
		return nil, fmt.Errorf("failed to save application: %w", err)
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

// DeleteApplication deletes an application
func (s *SimpleApplicationService) DeleteApplication(token, accountID, appID string) error {
	kvService := NewKVService()
	
	// Delete the main application key
	kvKey := fmt.Sprintf("app:%s", appID)
	if err := kvService.DeleteValue(token, accountID, kvKey); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	// Also delete the password key if it exists
	passwordKey := fmt.Sprintf("app:%s:password", appID)
	kvService.DeleteValue(token, accountID, passwordKey) // Ignore error - password key might not exist

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

