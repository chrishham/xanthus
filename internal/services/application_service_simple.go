package services

import (
	"encoding/json"
	"fmt"
	"log"
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
	// Get applications from KV store
	client := &http.Client{Timeout: 10 * time.Second}
	kvKey := fmt.Sprintf("applications_%s", accountID)
	
	var applicationsJSON string
	err := utils.GetKVValue(client, token, accountID, kvKey, &applicationsJSON)
	if err != nil {
		// If key doesn't exist, return empty list
		return []models.Application{}, nil
	}

	var applications []models.Application
	if applicationsJSON == "" {
		return applications, nil
	}

	if err := json.Unmarshal([]byte(applicationsJSON), &applications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal applications: %w", err)
	}

	// Update status for each application
	for i := range applications {
		if status, err := s.GetApplicationRealTimeStatus(token, accountID, &applications[i]); err == nil {
			applications[i].Status = status
		}
		applications[i].UpdatedAt = time.Now().Format(time.RFC3339)
	}

	// Save updated applications back to KV
	if err := s.saveApplications(token, accountID, applications); err != nil {
		log.Printf("Warning: failed to save updated applications: %v", err)
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

	// Get existing applications
	applications, err := s.ListApplications(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing applications: %w", err)
	}

	// Add new application
	applications = append(applications, *app)

	// Save to KV store
	if err := s.saveApplications(token, accountID, applications); err != nil {
		return nil, fmt.Errorf("failed to save application: %w", err)
	}

	return app, nil
}

// UpdateApplication updates an existing application
func (s *SimpleApplicationService) UpdateApplication(token, accountID string, app *models.Application) error {
	applications, err := s.ListApplications(token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get applications: %w", err)
	}

	// Find and update the application
	found := false
	for i, existing := range applications {
		if existing.ID == app.ID {
			app.UpdatedAt = time.Now().Format(time.RFC3339)
			applications[i] = *app
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("application not found: %s", app.ID)
	}

	// Save updated applications
	return s.saveApplications(token, accountID, applications)
}

// DeleteApplication deletes an application
func (s *SimpleApplicationService) DeleteApplication(token, accountID, appID string) error {
	applications, err := s.ListApplications(token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get applications: %w", err)
	}

	// Find and remove the application
	found := false
	for i, app := range applications {
		if app.ID == appID {
			applications = append(applications[:i], applications[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("application not found: %s", appID)
	}

	// Save updated applications
	return s.saveApplications(token, accountID, applications)
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

// saveApplications saves applications list to KV store
func (s *SimpleApplicationService) saveApplications(token, accountID string, applications []models.Application) error {
	client := &http.Client{Timeout: 10 * time.Second}
	kvKey := fmt.Sprintf("applications_%s", accountID)
	applicationsJSON, err := json.Marshal(applications)
	if err != nil {
		return fmt.Errorf("failed to marshal applications: %w", err)
	}

	return utils.PutKVValue(client, token, accountID, kvKey, string(applicationsJSON))
}