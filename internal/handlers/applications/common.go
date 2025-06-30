package applications

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
)

// VPSConnectionHelper manages VPS connection setup
type VPSConnectionHelper struct {
	kvService  *services.KVService
	sshService *services.SSHService
}

// NewVPSConnectionHelper creates a new VPS connection helper
func NewVPSConnectionHelper() *VPSConnectionHelper {
	return &VPSConnectionHelper{
		kvService:  services.NewKVService(),
		sshService: services.NewSSHService(),
	}
}

// GetVPSConnection establishes an SSH connection to a VPS
func (v *VPSConnectionHelper) GetVPSConnection(token, accountID, vpsID string) (*services.SSHConnection, error) {
	// Get VPS configuration
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err := v.kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", vpsID), &vpsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return nil, fmt.Errorf("failed to get SSH private key: %v", err)
	}

	// Create SSH connection
	vpsIDInt, _ := strconv.Atoi(vpsID)
	conn, err := v.sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, vpsIDInt)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VPS: %v", err)
	}

	return conn, nil
}

// GetVPSConfigByID retrieves VPS configuration by ID
func (v *VPSConnectionHelper) GetVPSConfigByID(token, accountID, vpsID string) (*VPSConfig, error) {
	var vpsConfig VPSConfig
	err := v.kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", vpsID), &vpsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPS configuration: %v", err)
	}
	return &vpsConfig, nil
}

// VPSConfig represents VPS configuration structure
type VPSConfig struct {
	PublicIPv4 string `json:"public_ipv4"`
	SSHUser    string `json:"ssh_user"`
	Name       string `json:"name"`
}

// SSHKeyHelper manages SSH key operations
type SSHKeyHelper struct {
	kvService *services.KVService
}

// NewSSHKeyHelper creates a new SSH key helper
func NewSSHKeyHelper() *SSHKeyHelper {
	return &SSHKeyHelper{
		kvService: services.NewKVService(),
	}
}

// GetSSHPrivateKey retrieves the SSH private key for VPS connections
func (s *SSHKeyHelper) GetSSHPrivateKey(token, accountID string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return "", fmt.Errorf("failed to get SSH private key: %v", err)
	}
	return csrConfig.PrivateKey, nil
}

// ValidationHelper provides common validation functions
type ValidationHelper struct{}

// NewValidationHelper creates a new validation helper
func NewValidationHelper() *ValidationHelper {
	return &ValidationHelper{}
}

// ValidateApplicationData validates basic application creation data
func (v *ValidationHelper) ValidateApplicationData(data interface{}) error {
	appData, ok := data.(struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		AppType     string `json:"app_type"`
		Subdomain   string `json:"subdomain"`
		Domain      string `json:"domain"`
		VPS         string `json:"vps"`
	})
	if !ok {
		return fmt.Errorf("invalid application data structure")
	}

	if appData.Name == "" {
		return fmt.Errorf("application name is required")
	}
	if appData.AppType == "" {
		return fmt.Errorf("application type is required")
	}
	if appData.Subdomain == "" {
		return fmt.Errorf("subdomain is required")
	}
	if appData.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if appData.VPS == "" {
		return fmt.Errorf("VPS selection is required")
	}

	return nil
}

// ValidateSubdomainAvailability checks if subdomain is already taken for the domain
func (v *ValidationHelper) ValidateSubdomainAvailability(token, accountID, subdomain, domain string) error {
	kvService := services.NewKVService()

	// Get all existing applications
	applications, err := v.getExistingApplications(token, accountID, kvService)
	if err != nil {
		return fmt.Errorf("failed to check existing applications: %v", err)
	}

	// Check if subdomain + domain combination is already taken
	targetURL := fmt.Sprintf("https://%s.%s", subdomain, domain)
	for _, app := range applications {
		if app.URL == targetURL {
			return fmt.Errorf("subdomain '%s' is already taken for domain '%s'", subdomain, domain)
		}
	}

	return nil
}

// getExistingApplications retrieves all existing applications from KV store
func (v *ValidationHelper) getExistingApplications(token, accountID string, kvService *services.KVService) ([]models.Application, error) {
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

// ValidatePassword validates password requirements
func (v *ValidationHelper) ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	return nil
}

// ValidateApplicationType checks if application type is supported
func (v *ValidationHelper) ValidateApplicationType(appType string) error {
	supportedTypes := []string{"code-server", "argocd"}
	for _, supportedType := range supportedTypes {
		if appType == supportedType {
			return nil
		}
	}
	return fmt.Errorf("unsupported application type: %s", appType)
}

// ApplicationHelper provides common application operations
type ApplicationHelper struct {
	kvService *services.KVService
}

// NewApplicationHelper creates a new application helper
func NewApplicationHelper() *ApplicationHelper {
	return &ApplicationHelper{
		kvService: services.NewKVService(),
	}
}

// GetApplicationByID retrieves an application by ID
func (a *ApplicationHelper) GetApplicationByID(token, accountID, appID string) (*models.Application, error) {
	var app models.Application
	err := a.kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		return nil, fmt.Errorf("application not found: %v", err)
	}
	return &app, nil
}

// UpdateApplicationStatus updates an application's status
func (a *ApplicationHelper) UpdateApplicationStatus(token, accountID, appID, status string) error {
	app, err := a.GetApplicationByID(token, accountID, appID)
	if err != nil {
		return err
	}

	app.Status = status
	app.UpdatedAt = time.Now().Format(time.RFC3339)

	return a.kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)
}

// GenerateApplicationID generates a unique application ID
func (a *ApplicationHelper) GenerateApplicationID() string {
	return fmt.Sprintf("app-%d", time.Now().Unix())
}

// GenerateReleaseName generates a Helm release name for an application
func (a *ApplicationHelper) GenerateReleaseName(subdomain, appID string) string {
	return fmt.Sprintf("%s-%s", subdomain, appID)
}

// PasswordHelper manages application password operations
type PasswordHelper struct {
	kvService *services.KVService
}

// NewPasswordHelper creates a new password helper
func NewPasswordHelper() *PasswordHelper {
	return &PasswordHelper{
		kvService: services.NewKVService(),
	}
}

// StoreEncryptedPassword stores an encrypted password in KV
func (p *PasswordHelper) StoreEncryptedPassword(token, accountID, appID, password string) error {
	encryptedPassword, err := utils.EncryptData(password, token)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}

	return p.kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:password", appID), map[string]string{
		"password": encryptedPassword,
	})
}

// GetDecryptedPassword retrieves and decrypts a stored password
// If not found in KV, attempts to retrieve from VPS and store it
func (p *PasswordHelper) GetDecryptedPassword(token, accountID, appID string) (string, error) {
	var passwordData struct {
		Password string `json:"password"`
	}

	// First try to get from KV store
	err := p.kvService.GetValue(token, accountID, fmt.Sprintf("app:%s:password", appID), &passwordData)
	if err == nil {
		// Found in KV, decrypt and return
		password, err := utils.DecryptData(passwordData.Password, token)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt password: %v", err)
		}
		return password, nil
	}

	// Password not found in KV, try to retrieve from VPS
	fmt.Printf("Password not found in KV for app %s, attempting to retrieve from VPS\n", appID)
	password, err := p.retrievePasswordFromVPS(token, accountID, appID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve password from VPS: %v", err)
	}

	// Store the retrieved password in KV for future use
	if storeErr := p.StoreEncryptedPassword(token, accountID, appID, password); storeErr != nil {
		fmt.Printf("Warning: Failed to store retrieved password in KV: %v\n", storeErr)
	} else {
		fmt.Printf("Successfully stored retrieved password in KV for app %s\n", appID)
	}

	return password, nil
}

// retrievePasswordFromVPS retrieves password directly from VPS for a given application
func (p *PasswordHelper) retrievePasswordFromVPS(token, accountID, appID string) (string, error) {
	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		return "", fmt.Errorf("failed to get application details: %v", err)
	}

	// Get VPS connection
	vpsHelper := NewVPSConnectionHelper()
	conn, err := vpsHelper.GetVPSConnection(token, accountID, app.VPSID)
	if err != nil {
		return "", fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Generate release name (should match the deployment logic)
	releaseName := fmt.Sprintf("%s-%s", app.AppType, app.ID)

	// Retrieve password based on application type
	switch app.AppType {
	case "code-server":
		return p.retrieveCodeServerPasswordFromVPS(conn, releaseName, app.Namespace)
	case "argocd":
		return p.retrieveArgoCDPasswordFromVPS(conn, releaseName, app.Namespace)
	default:
		return "", fmt.Errorf("password retrieval not supported for application type: %s", app.AppType)
	}
}

// retrieveCodeServerPasswordFromVPS retrieves code-server password from VPS
func (p *PasswordHelper) retrieveCodeServerPasswordFromVPS(conn *services.SSHConnection, releaseName, namespace string) (string, error) {
	sshService := services.NewSSHService()

	// Retrieve password from Kubernetes secret
	// The secret name is the same as the release name for code-server
	secretName := releaseName
	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' | base64 --decode", namespace, secretName)

	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve code-server password from secret '%s' in namespace '%s': %v", secretName, namespace, err)
	}

	password := strings.TrimSpace(result.Output)
	if password == "" {
		return "", fmt.Errorf("retrieved empty password from code-server secret")
	}

	return password, nil
}

// retrieveArgoCDPasswordFromVPS retrieves ArgoCD admin password from VPS
func (p *PasswordHelper) retrieveArgoCDPasswordFromVPS(conn *services.SSHConnection, releaseName, namespace string) (string, error) {
	sshService := services.NewSSHService()

	// Retrieve admin password from ArgoCD initial admin secret
	secretName := "argocd-initial-admin-secret"
	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", namespace, secretName)

	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve ArgoCD password from secret '%s' in namespace '%s': %v", secretName, namespace, err)
	}

	password := strings.TrimSpace(result.Output)
	if password == "" {
		return "", fmt.Errorf("retrieved empty password from ArgoCD secret")
	}

	return password, nil
}

// DeletePassword removes a stored password
func (p *PasswordHelper) DeletePassword(token, accountID, appID string) error {
	return p.kvService.DeleteValue(token, accountID, fmt.Sprintf("app:%s:password", appID))
}
