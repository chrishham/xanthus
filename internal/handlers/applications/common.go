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
	// Release name format: subdomain-apptype (e.g., final-test-code-server)
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.AppType)

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
	// For custom xanthus-code-server chart, the secret name follows the fullname pattern: releaseName-chartName
	// For official code-server chart, the secret name is the same as release name
	secretName := releaseName
	
	// Try the custom chart secret name first (release-name + "-xanthus-code-server")
	customSecretName := fmt.Sprintf("%s-xanthus-code-server", releaseName)
	cmd := fmt.Sprintf("set -o pipefail; kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", namespace, customSecretName)

	fmt.Printf("DEBUG: Trying custom chart secret: %s\n", customSecretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err == nil && strings.TrimSpace(result.Output) != "" && !strings.Contains(result.Output, "Error from server") {
		// Success with custom chart secret - make sure it's not an error message
		return strings.TrimSpace(result.Output), nil
	}
	
	// Fall back to original secret name for official chart
	cmd = fmt.Sprintf("set -o pipefail; kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", namespace, secretName)
	fmt.Printf("DEBUG: Trying official chart secret: %s\n", secretName)
	result, err = sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		fmt.Printf("DEBUG: Both secret lookups failed. Last error: %v\n", err)
		fmt.Printf("DEBUG: Last command output: %s\n", result.Output)
		return "", fmt.Errorf("failed to retrieve code-server password from secret '%s' or '%s' in namespace '%s': %v", customSecretName, secretName, namespace, err)
	}

	password := strings.TrimSpace(result.Output)
	if password == "" || strings.Contains(password, "Error from server") {
		return "", fmt.Errorf("retrieved empty password or error from code-server secret")
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
	if password == "" || strings.Contains(password, "Error from server") {
		return "", fmt.Errorf("retrieved empty password or error from ArgoCD secret")
	}

	return password, nil
}

// DeletePassword removes a stored password
func (p *PasswordHelper) DeletePassword(token, accountID, appID string) error {
	return p.kvService.DeleteValue(token, accountID, fmt.Sprintf("app:%s:password", appID))
}

// PortForwardService manages port forwarding operations
type PortForwardService struct {
	kvService  *services.KVService
	sshService *services.SSHService
}

// NewPortForwardService creates a new port forward service
func NewPortForwardService() *PortForwardService {
	return &PortForwardService{
		kvService:  services.NewKVService(),
		sshService: services.NewSSHService(),
	}
}

// PortForward represents a port forwarding configuration
type PortForward struct {
	ID          string `json:"id"`
	AppID       string `json:"app_id"`
	Port        int    `json:"port"`
	Subdomain   string `json:"subdomain"`
	Domain      string `json:"domain"`
	URL         string `json:"url"`
	ServiceName string `json:"service_name"`
	IngressName string `json:"ingress_name"`
	CreatedAt   string `json:"created_at"`
}

// ListPortForwards retrieves all port forwards for an application
func (p *PortForwardService) ListPortForwards(token, accountID, appID string) ([]PortForward, error) {
	var portForwards []PortForward

	// Get port forwards from KV store
	err := p.kvService.GetValue(token, accountID, fmt.Sprintf("app:%s:port-forwards", appID), &portForwards)
	if err != nil {
		// Return empty list if not found
		return []PortForward{}, nil
	}

	return portForwards, nil
}

// CreatePortForward creates a new port forward with Kubernetes service and ingress
func (p *PortForwardService) CreatePortForward(token, accountID, appID string, port int, subdomain string) (*PortForward, error) {
	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %v", err)
	}

	// Extract domain from application URL
	domain, err := p.extractDomainFromURL(app.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract domain: %v", err)
	}

	// Generate unique names
	portForwardID := fmt.Sprintf("pf-%d", time.Now().Unix())
	serviceName := fmt.Sprintf("%s-port-%d", app.ID, port)
	ingressName := fmt.Sprintf("%s-port-%d-ingress", app.ID, port)
	url := fmt.Sprintf("https://%s.%s", subdomain, domain)

	// Create the port forward configuration
	portForward := &PortForward{
		ID:          portForwardID,
		AppID:       appID,
		Port:        port,
		Subdomain:   subdomain,
		Domain:      domain,
		URL:         url,
		ServiceName: serviceName,
		IngressName: ingressName,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// Get VPS connection
	vpsHelper := NewVPSConnectionHelper()
	conn, err := vpsHelper.GetVPSConnection(token, accountID, app.VPSID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Create Kubernetes service
	if err := p.createKubernetesService(conn, app, portForward); err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes service: %v", err)
	}

	// Create Kubernetes ingress
	if err := p.createKubernetesIngress(conn, app, portForward); err != nil {
		// Cleanup service if ingress creation fails
		p.deleteKubernetesService(conn, app.Namespace, serviceName)
		return nil, fmt.Errorf("failed to create Kubernetes ingress: %v", err)
	}

	// Create DNS A record for the port forward subdomain
	if err := p.createPortForwardDNS(token, accountID, app.VPSID, portForward); err != nil {
		// Cleanup resources if DNS creation fails
		p.deleteKubernetesIngress(conn, app.Namespace, ingressName)
		p.deleteKubernetesService(conn, app.Namespace, serviceName)
		return nil, fmt.Errorf("failed to create DNS record: %v", err)
	}

	// Store port forward in KV
	if err := p.storePortForward(token, accountID, appID, portForward); err != nil {
		// Cleanup resources if storage fails
		p.deletePortForwardDNS(token, portForward.Subdomain, portForward.Domain)
		p.deleteKubernetesIngress(conn, app.Namespace, ingressName)
		p.deleteKubernetesService(conn, app.Namespace, serviceName)
		return nil, fmt.Errorf("failed to store port forward: %v", err)
	}

	return portForward, nil
}

// DeletePortForward removes a port forward and cleans up Kubernetes resources
func (p *PortForwardService) DeletePortForward(token, accountID, appID, portForwardID string) error {
	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		return fmt.Errorf("failed to get application: %v", err)
	}

	// Get existing port forwards
	portForwards, err := p.ListPortForwards(token, accountID, appID)
	if err != nil {
		return fmt.Errorf("failed to get port forwards: %v", err)
	}

	// Find the port forward to delete
	var targetPortForward *PortForward
	var updatedPortForwards []PortForward

	for _, pf := range portForwards {
		if pf.ID == portForwardID {
			targetPortForward = &pf
		} else {
			updatedPortForwards = append(updatedPortForwards, pf)
		}
	}

	if targetPortForward == nil {
		return fmt.Errorf("port forward not found")
	}

	// Get VPS connection
	vpsHelper := NewVPSConnectionHelper()
	conn, err := vpsHelper.GetVPSConnection(token, accountID, app.VPSID)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Delete Kubernetes resources
	p.deleteKubernetesIngress(conn, app.Namespace, targetPortForward.IngressName)
	p.deleteKubernetesService(conn, app.Namespace, targetPortForward.ServiceName)

	// Delete DNS A record
	p.deletePortForwardDNS(token, targetPortForward.Subdomain, targetPortForward.Domain)

	// Update KV store
	return p.kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:port-forwards", appID), updatedPortForwards)
}

// extractDomainFromURL extracts the domain from a URL
func (p *PortForwardService) extractDomainFromURL(urlStr string) (string, error) {
	// Simple domain extraction - assumes URL format is https://subdomain.domain.tld
	if !strings.HasPrefix(urlStr, "https://") {
		return "", fmt.Errorf("invalid URL format")
	}

	hostname := strings.TrimPrefix(urlStr, "https://")
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid hostname format")
	}

	// Return domain without the first subdomain part
	return strings.Join(parts[1:], "."), nil
}

// storePortForward stores a port forward in the KV store
func (p *PortForwardService) storePortForward(token, accountID, appID string, portForward *PortForward) error {
	// Get existing port forwards
	existingPortForwards, _ := p.ListPortForwards(token, accountID, appID)

	// Add new port forward
	existingPortForwards = append(existingPortForwards, *portForward)

	// Store updated list
	return p.kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:port-forwards", appID), existingPortForwards)
}

// createKubernetesService creates a Kubernetes service for port forwarding
func (p *PortForwardService) createKubernetesService(conn *services.SSHConnection, app *models.Application, portForward *PortForward) error {
	// Use the actual deployment's release name pattern: subdomain-apptype
	// This matches how applications are actually deployed
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.AppType)

	serviceYAML := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    port-forward: "true"
spec:
  selector:
    app.kubernetes.io/name: %s
    app.kubernetes.io/instance: %s
  ports:
  - name: port-%d
    port: 80
    targetPort: %d
    protocol: TCP
  type: ClusterIP
`, portForward.ServiceName, app.Namespace, portForward.ServiceName, app.AppType, releaseName, portForward.Port, portForward.Port)

	// Apply the service using kubectl
	cmd := fmt.Sprintf("cat <<'EOF' | kubectl apply -f -\n%s\nEOF", serviceYAML)
	result, err := p.sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to create service: %v, output: %s", err, result.Output)
	}

	return nil
}

// createKubernetesIngress creates a Kubernetes ingress for port forwarding
func (p *PortForwardService) createKubernetesIngress(conn *services.SSHConnection, app *models.Application, portForward *PortForward) error {
	ingressYAML := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    port-forward: "true"
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  tls:
  - secretName: %s-tls
    hosts:
    - %s.%s
  rules:
  - host: %s.%s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: 80
`, portForward.IngressName, app.Namespace, portForward.ServiceName, portForward.Domain, portForward.Subdomain, portForward.Domain, portForward.Subdomain, portForward.Domain, portForward.ServiceName)

	// Apply the ingress using kubectl
	cmd := fmt.Sprintf("cat <<'EOF' | kubectl apply -f -\n%s\nEOF", ingressYAML)
	result, err := p.sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to create ingress: %v, output: %s", err, result.Output)
	}

	return nil
}

// deleteKubernetesService deletes a Kubernetes service
func (p *PortForwardService) deleteKubernetesService(conn *services.SSHConnection, namespace, serviceName string) error {
	cmd := fmt.Sprintf("kubectl delete service --namespace %s %s --ignore-not-found=true", namespace, serviceName)
	_, err := p.sshService.ExecuteCommand(conn, cmd)
	return err
}

// deleteKubernetesIngress deletes a Kubernetes ingress
func (p *PortForwardService) deleteKubernetesIngress(conn *services.SSHConnection, namespace, ingressName string) error {
	cmd := fmt.Sprintf("kubectl delete ingress --namespace %s %s --ignore-not-found=true", namespace, ingressName)
	_, err := p.sshService.ExecuteCommand(conn, cmd)
	return err
}

// createPortForwardDNS creates a DNS A record for the port forward subdomain
func (p *PortForwardService) createPortForwardDNS(token, accountID, vpsID string, portForward *PortForward) error {
	// Get VPS configuration to retrieve the public IP
	vpsHelper := NewVPSConnectionHelper()
	vpsConfig, err := vpsHelper.GetVPSConfigByID(token, accountID, vpsID)
	if err != nil {
		return fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Create Cloudflare DNS service
	cfService := services.NewCloudflareService()

	// Get zone ID for the domain
	zoneID, err := cfService.GetZoneID(token, portForward.Domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID for domain %s: %v", portForward.Domain, err)
	}

	// Create A record for subdomain
	recordName := fmt.Sprintf("%s.%s", portForward.Subdomain, portForward.Domain)
	_, err = cfService.CreateDNSRecord(token, zoneID, "A", recordName, vpsConfig.PublicIPv4, true)
	if err != nil {
		return fmt.Errorf("failed to create DNS A record for %s: %v", recordName, err)
	}

	return nil
}

// deletePortForwardDNS deletes the DNS A record for a port forward subdomain
func (p *PortForwardService) deletePortForwardDNS(token, subdomain, domain string) error {
	// Create Cloudflare DNS service
	cfService := services.NewCloudflareService()

	// Get zone ID for the domain
	zoneID, err := cfService.GetZoneID(token, domain)
	if err != nil {
		// Log error but don't fail the deletion - DNS cleanup is best effort
		fmt.Printf("Warning: Failed to get zone ID for domain %s during DNS cleanup: %v\n", domain, err)
		return nil
	}

	// Get all DNS records for the domain
	records, err := cfService.GetDNSRecords(token, zoneID)
	if err != nil {
		// Log error but don't fail the deletion - DNS cleanup is best effort
		fmt.Printf("Warning: Failed to list DNS records for domain %s during cleanup: %v\n", domain, err)
		return nil
	}

	// Find and delete A records matching the subdomain
	recordName := fmt.Sprintf("%s.%s", subdomain, domain)
	for _, record := range records {
		// Normalize record names by removing trailing dots
		normalizedRecordName := strings.TrimSuffix(record.Name, ".")
		normalizedTargetName := strings.TrimSuffix(recordName, ".")

		if record.Type == "A" && normalizedRecordName == normalizedTargetName {
			err := cfService.DeleteDNSRecord(token, zoneID, record.ID)
			if err != nil {
				// Log error but don't fail the deletion - DNS cleanup is best effort
				fmt.Printf("Warning: Failed to delete DNS record %s during cleanup: %v\n", record.ID, err)
			} else {
				fmt.Printf("Successfully deleted DNS A record for %s\n", recordName)
			}
		}
	}

	return nil
}
