package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// deployApplication deploys a predefined application using its Helm configuration
func (s *SimpleApplicationService) deployApplication(token, accountID string, appData map[string]interface{}, predefinedApp *models.PredefinedApplication, appID string) error {
	kvService := NewKVService()
	sshService := NewSSHService()

	subdomain := appData["subdomain"].(string)
	domain := appData["domain"].(string)
	vpsID := appData["vps_id"].(string)

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
	vpsIDInt, _ := strconv.Atoi(vpsID)
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Generate release name and namespace
	releaseName := fmt.Sprintf("%s-%s", predefinedApp.ID, appID)
	namespace := fmt.Sprintf("app-%s", strings.ReplaceAll(subdomain, ".", "-"))

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
	valuesContent, err := s.generateValuesFile(predefinedApp, subdomain, domain, releaseName)
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
	if err := s.retrieveApplicationPassword(token, accountID, predefinedApp.ID, appID, vpsID, releaseName, namespace, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey); err != nil {
		// Log the error but don't fail the deployment since the app is successfully deployed
		fmt.Printf("Warning: Failed to retrieve application password: %v\n", err)
	}

	return nil
}

// generateValuesFile generates a Helm values file using template-based approach
func (s *SimpleApplicationService) generateValuesFile(predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName string) (string, error) {
	// Check if a values template is specified in the configuration
	if predefinedApp.HelmChart.ValuesTemplate != "" {
		return s.generateFromTemplate(predefinedApp, subdomain, domain, releaseName)
	}
	
	// Fallback to minimal values if no template is specified
	return s.generateMinimalValues(predefinedApp, subdomain, domain, releaseName)
}

// generateFromTemplate generates values from a template file with placeholder substitution
func (s *SimpleApplicationService) generateFromTemplate(predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName string) (string, error) {
	templatePath := fmt.Sprintf("internal/templates/applications/%s", predefinedApp.HelmChart.ValuesTemplate)
	
	// Read the template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}
	
	// Prepare placeholder values
	placeholders := map[string]string{
		"VERSION":      predefinedApp.Version,
		"SUBDOMAIN":    subdomain,
		"DOMAIN":       domain,
		"RELEASE_NAME": releaseName,
	}
	
	// Add any additional placeholders from the configuration
	for key, value := range predefinedApp.HelmChart.Placeholders {
		placeholders[key] = value
	}
	
	// Replace placeholders in the template
	content := string(templateContent)
	for placeholder, value := range placeholders {
		content = strings.ReplaceAll(content, fmt.Sprintf("{{%s}}", placeholder), value)
	}
	
	return content, nil
}


// generateMinimalValues generates minimal values when no template is available
func (s *SimpleApplicationService) generateMinimalValues(predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName string) (string, error) {
	// Generate basic ingress configuration for any application
	return fmt.Sprintf(`
# Minimal values generated for %s
ingress:
  enabled: true
  className: "traefik"
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
  hosts:
    - host: %s.%s
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: %s-tls
      hosts:
        - %s.%s

# Application version
image:
  tag: "%s"
`, predefinedApp.ID, subdomain, domain, releaseName, subdomain, domain, predefinedApp.Version), nil
}

// retrieveApplicationPassword retrieves and stores the auto-generated password for applications that create them
func (s *SimpleApplicationService) retrieveApplicationPassword(token, accountID, appType, appID, vpsID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	switch appType {
	case "code-server":
		return s.retrieveCodeServerPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey)
	case "argocd":
		return s.retrieveArgoCDPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey)
	default:
		// No password retrieval needed for this application type
		return nil
	}
}

// retrieveCodeServerPassword retrieves the auto-generated password from code-server Kubernetes secret
func (s *SimpleApplicationService) retrieveCodeServerPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

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
	return s.storeEncryptedPassword(token, accountID, appID, password)
}

// retrieveArgoCDPassword retrieves the auto-generated admin password from ArgoCD
func (s *SimpleApplicationService) retrieveArgoCDPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

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
	return s.storeEncryptedPassword(token, accountID, appID, password)
}

// storeEncryptedPassword encrypts and stores the password in KV store
func (s *SimpleApplicationService) storeEncryptedPassword(token, accountID, appID, password string) error {
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

