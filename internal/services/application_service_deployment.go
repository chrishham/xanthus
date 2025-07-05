package services

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/utils"
)

// deployApplication deploys a predefined application using its Helm configuration
func (s *SimpleApplicationService) deployApplication(token, accountID string, appData map[string]interface{}, predefinedApp *models.PredefinedApplication, appID string) error {
	kvService := NewKVService()
	sshService := NewSSHService()

	subdomain := appData["subdomain"].(string)
	domain := appData["domain"].(string)
	vpsID := appData["vps_id"].(string)

	// Check for existing ArgoCD installation on this VPS
	if predefinedApp.ID == "argocd" {
		if err := s.checkExistingArgoCDInstallation(token, accountID, vpsID, appID); err != nil {
			return err
		}
	}

	// Get VPS configuration for SSH details and timezone
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
		Timezone   string `json:"timezone"`
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

	if predefinedApp.ID == "code-server" && helmConfig.Repository == "local" {
		// Use local chart for code-server
		chartPath := "/tmp/xanthus-code-server"
		if err := s.copyLocalChartToVPS(conn, sshService, chartPath); err != nil {
			return fmt.Errorf("failed to copy local chart to VPS: %v", err)
		}
		chartName = chartPath
	} else if strings.Contains(helmConfig.Repository, "github.com") {
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

	// Handle missing timezone with VPS location-based fallback
	timezone := vpsConfig.Timezone
	if timezone == "" {
		// Fallback to detecting timezone from SSH connection to VPS
		if timezone = s.detectVPSTimezone(token, accountID, vpsID); timezone == "" {
			timezone = "Europe/Berlin" // Ultimate fallback for nbg1 datacenter
		}
	}

	// Generate and upload values file
	valuesContent, err := s.generateValuesFile(predefinedApp, subdomain, domain, releaseName, timezone)
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
		// Check for resource exhaustion and clean up if detected
		if deploymentErr := s.handleDeploymentFailure(conn, sshService, releaseName, namespace, result.Output, err); deploymentErr != nil {
			return deploymentErr
		}
		return fmt.Errorf("helm install failed: %v, output: %s", err, result.Output)
	}

	// Configure SSL certificates on VPS - this is required for HTTPS access
	if err := s.configureVPSSSL(token, accountID, domain, vpsConfig, csrConfig); err != nil {
		return fmt.Errorf("failed to configure SSL certificates on VPS: %v", err)
	}

	// Create TLS secret in Kubernetes namespace for ingress
	domainConfig, err := kvService.GetDomainSSLConfig(token, accountID, domain)
	if err != nil {
		return fmt.Errorf("failed to get domain SSL config for TLS secret creation: %v", err)
	}

	if err := sshService.CreateTLSSecret(conn, domain, domainConfig.Certificate, domainConfig.PrivateKey, namespace); err != nil {
		return fmt.Errorf("failed to create TLS secret in namespace %s: %v", namespace, err)
	}

	// Configure DNS record for the application
	if err := s.configureApplicationDNS(token, subdomain, domain, vpsConfig.PublicIPv4); err != nil {
		return fmt.Errorf("failed to configure DNS for application: %v", err)
	}

	// Password retrieval is now handled on-demand when user requests it
	// No need to retrieve and store passwords during deployment

	return nil
}

// configureVPSSSL configures SSL certificates on the VPS for the given domain
func (s *SimpleApplicationService) configureVPSSSL(token, accountID, domain string, vpsConfig struct {
	PublicIPv4 string `json:"public_ipv4"`
	SSHUser    string `json:"ssh_user"`
	Timezone   string `json:"timezone"`
}, csrConfig struct {
	PrivateKey string `json:"private_key"`
}) error {
	kvService := NewKVService()
	sshService := NewSSHService()
	cfService := NewCloudflareService()

	// Check if SSL is already configured for this domain on this VPS
	sslConfigKey := fmt.Sprintf("vps:%s:ssl:%s", vpsConfig.PublicIPv4, domain)
	var existingSSLConfig map[string]interface{}
	if err := kvService.GetValue(token, accountID, sslConfigKey, &existingSSLConfig); err == nil {
		fmt.Printf("SSL already configured for domain %s on VPS %s\n", domain, vpsConfig.PublicIPv4)
		return nil
	}

	// Get SSL configuration for the domain from Cloudflare
	domainConfig, err := kvService.GetDomainSSLConfig(token, accountID, domain)
	if err != nil {
		// Domain SSL not configured, create it
		fmt.Printf("Creating SSL configuration for domain %s\n", domain)

		// Generate CSR if needed
		var csrData struct {
			CSR        string `json:"csr"`
			PrivateKey string `json:"private_key"`
			CreatedAt  string `json:"created_at"`
		}
		if err := kvService.GetValue(token, accountID, "config:ssl:csr", &csrData); err != nil {
			return fmt.Errorf("failed to get CSR configuration: %v", err)
		}

		// Configure domain SSL with Cloudflare
		domainConfig, err = cfService.ConfigureDomainSSL(token, domain, csrData.CSR, csrData.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to configure domain SSL: %v", err)
		}

		// Store domain SSL configuration
		err = kvService.StoreDomainSSLConfig(token, accountID, domainConfig)
		if err != nil {
			return fmt.Errorf("failed to store domain SSL config: %v", err)
		}
	}

	// Connect to VPS and configure SSL certificates
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, 0)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Configure K3s with SSL certificates
	if err := sshService.ConfigureK3s(conn, domainConfig.Certificate, domainConfig.PrivateKey); err != nil {
		return fmt.Errorf("failed to configure K3s with SSL: %v", err)
	}

	// Mark SSL as configured for this VPS/domain combination
	sslStatus := map[string]interface{}{
		"configured_at": fmt.Sprintf("%d", time.Now().Unix()),
		"domain":        domain,
		"vps_ip":        vpsConfig.PublicIPv4,
	}
	if err := kvService.PutValue(token, accountID, sslConfigKey, sslStatus); err != nil {
		fmt.Printf("Warning: Failed to store SSL configuration status: %v\n", err)
	}

	fmt.Printf("Successfully configured SSL certificates for domain %s on VPS %s\n", domain, vpsConfig.PublicIPv4)
	return nil
}

// configureApplicationDNS creates DNS A record for the application subdomain
func (s *SimpleApplicationService) configureApplicationDNS(token, subdomain, domain, vpsIP string) error {
	cfService := NewCloudflareService()

	// Get zone ID for the domain
	zoneID, err := cfService.GetZoneID(token, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID for domain %s: %v", domain, err)
	}

	// Handle bare domain (blank or asterisk subdomain)
	if subdomain == "" || subdomain == "*" {
		// Create A record for bare domain
		_, err := cfService.CreateDNSRecord(token, zoneID, "A", domain, vpsIP, true)
		return err
	}

	// Create A record for subdomain
	recordName := fmt.Sprintf("%s.%s", subdomain, domain)
	_, err = cfService.CreateDNSRecord(token, zoneID, "A", recordName, vpsIP, true)
	return err
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

// retrieveCodeServerPassword retrieves the auto-generated password from code-server config file
func (s *SimpleApplicationService) retrieveCodeServerPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Find the pod name for the code-server deployment
	podNameCmd := fmt.Sprintf("kubectl get pods -n %s -l app.kubernetes.io/name=xanthus-code-server -o jsonpath='{.items[0].metadata.name}'", namespace)
	result, err := sshService.ExecuteCommand(conn, podNameCmd)
	if err != nil {
		return fmt.Errorf("failed to get pod name: %v", err)
	}

	podName := strings.TrimSpace(result.Output)
	if podName == "" {
		return fmt.Errorf("no code-server pod found in namespace %s", namespace)
	}

	var password string
	var retrievalErr error

	// Retry up to 30 times with 2-second intervals (1 minute total)
	// Wait for code-server to start and generate its config file
	for attempt := 1; attempt <= 30; attempt++ {
		fmt.Printf("Password retrieval attempt %d/30 for app %s from pod %s\n", attempt, appID, podName)

		// Try to read password from code-server config file
		configCmd := fmt.Sprintf("kubectl exec -n %s %s -- cat /home/coder/.config/code-server/config.yaml 2>/dev/null | grep '^password:' | awk '{print $2}'", namespace, podName)
		result, err := sshService.ExecuteCommand(conn, configCmd)
		if err == nil && strings.TrimSpace(result.Output) != "" && !strings.Contains(result.Output, "Error from server") {
			password = strings.TrimSpace(result.Output)
			fmt.Printf("Successfully retrieved password from config file on attempt %d: %s\n", attempt, password)
			break
		}

		if attempt == 30 {
			retrievalErr = fmt.Errorf("failed to retrieve password from config file in pod '%s' in namespace '%s' after 30 attempts", podName, namespace)
		} else {
			fmt.Printf("Config file not ready yet, waiting 2 seconds before retry %d...\n", attempt+1)
			time.Sleep(2 * time.Second)
		}
	}

	if retrievalErr != nil {
		return retrievalErr
	}

	if password == "" {
		return fmt.Errorf("retrieved empty password from config file")
	}

	// No longer storing code-server passwords - they are retrieved on-demand from config file
	fmt.Printf("Successfully retrieved password from config file (not storing in KV): %s\n", password)
	return nil
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

	fmt.Printf("Successfully retrieved ArgoCD admin password for application %s\n", appID)
	return nil
}

// detectVPSTimezone detects the current timezone from the VPS via SSH
func (s *SimpleApplicationService) detectVPSTimezone(token, accountID, vpsID string) string {
	kvService := NewKVService()
	sshService := NewSSHService()

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err := kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", vpsID), &vpsConfig)
	if err != nil {
		return ""
	}

	// Get SSH private key
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := kvService.GetValue(token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return ""
	}

	// Create SSH connection
	conn, err := sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	if err != nil {
		return ""
	}
	defer conn.Close()

	// Get current timezone from VPS
	result, err := sshService.ExecuteCommand(conn, "timedatectl show --property=Timezone --value")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(result.Output)
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

// copyLocalChartToVPS copies the local Helm chart to the VPS
func (s *SimpleApplicationService) copyLocalChartToVPS(conn *SSHConnection, sshService *SSHService, remotePath string) error {
	localChartPath := "charts/xanthus-code-server"

	// Create remote directory
	_, err := sshService.ExecuteCommand(conn, fmt.Sprintf("mkdir -p %s", remotePath))
	if err != nil {
		return fmt.Errorf("failed to create remote chart directory: %v", err)
	}

	// Copy Chart.yaml
	if err := s.copyFileToVPS(conn, sshService, fmt.Sprintf("%s/Chart.yaml", localChartPath), fmt.Sprintf("%s/Chart.yaml", remotePath)); err != nil {
		return fmt.Errorf("failed to copy Chart.yaml: %v", err)
	}

	// Copy values.yaml
	if err := s.copyFileToVPS(conn, sshService, fmt.Sprintf("%s/values.yaml", localChartPath), fmt.Sprintf("%s/values.yaml", remotePath)); err != nil {
		return fmt.Errorf("failed to copy values.yaml: %v", err)
	}

	// Create templates directory
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("mkdir -p %s/templates", remotePath))
	if err != nil {
		return fmt.Errorf("failed to create templates directory: %v", err)
	}

	// Copy all template files
	templateFiles := []string{
		"_helpers.tpl",
		"deployment.yaml",
		"service.yaml",
		"pvc.yaml",
		"configmap.yaml",
		"ingress.yaml",
		"secret.yaml",
	}

	for _, file := range templateFiles {
		localFile := fmt.Sprintf("%s/templates/%s", localChartPath, file)
		remoteFile := fmt.Sprintf("%s/templates/%s", remotePath, file)
		if err := s.copyFileToVPS(conn, sshService, localFile, remoteFile); err != nil {
			return fmt.Errorf("failed to copy template %s: %v", file, err)
		}
	}

	return nil
}

// copyFileToVPS copies a local file to the VPS via SSH
func (s *SimpleApplicationService) copyFileToVPS(conn *SSHConnection, sshService *SSHService, localPath, remotePath string) error {
	// Read local file
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file %s: %v", localPath, err)
	}

	// Write to remote file using cat with heredoc
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", remotePath, string(content)))
	if err != nil {
		return fmt.Errorf("failed to write remote file %s: %v", remotePath, err)
	}

	return nil
}

// handleDeploymentFailure detects resource exhaustion and automatically cleans up failed deployments
func (s *SimpleApplicationService) handleDeploymentFailure(conn *SSHConnection, sshService *SSHService, releaseName, namespace, output string, originalErr error) error {
	// Check for resource exhaustion patterns
	resourceExhaustion := s.isResourceExhaustion(output)
	if !resourceExhaustion {
		// Not a resource issue, return original error
		return nil
	}

	fmt.Printf("🚨 Resource exhaustion detected for deployment %s. Initiating automatic cleanup...\n", releaseName)

	// Perform cleanup
	cleanupErr := s.cleanupFailedDeployment(conn, sshService, releaseName, namespace)
	if cleanupErr != nil {
		fmt.Printf("❌ Warning: Failed to clean up deployment %s: %v\n", releaseName, cleanupErr)
	} else {
		fmt.Printf("✅ Successfully cleaned up failed deployment %s\n", releaseName)
	}

	// Return user-friendly error message
	return fmt.Errorf("deployment failed due to insufficient resources on VPS. " +
		"Current VPS has reached CPU/memory capacity limits. " +
		"Please consider: 1) Upgrading VPS resources, 2) Removing unused applications, or 3) Optimizing resource allocation. " +
		"The failed deployment has been automatically cleaned up")
}

// isResourceExhaustion checks if the error output indicates resource exhaustion
func (s *SimpleApplicationService) isResourceExhaustion(output string) bool {
	resourcePatterns := []string{
		"Insufficient cpu",
		"Insufficient memory",
		"nodes are available",
		"preemption: 0/1 nodes are available",
		"No preemption victims found",
		"pod didn't trigger scale-up",
		"Insufficient ephemeral-storage",
		"Too many pods",
	}

	outputLower := strings.ToLower(output)
	for _, pattern := range resourcePatterns {
		if strings.Contains(outputLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// cleanupFailedDeployment removes all resources associated with a failed deployment
func (s *SimpleApplicationService) cleanupFailedDeployment(conn *SSHConnection, sshService *SSHService, releaseName, namespace string) error {
	var cleanupErrors []string

	// 1. Uninstall Helm release (if it exists)
	fmt.Printf("🧹 Cleaning up Helm release %s...\n", releaseName)
	_, err := sshService.ExecuteCommand(conn, fmt.Sprintf("helm uninstall %s -n %s", releaseName, namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to uninstall Helm release: %v", err))
	}

	// 2. Delete PVC (if it exists)
	fmt.Printf("🧹 Cleaning up PVC %s...\n", releaseName)
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl delete pvc %s -n %s --ignore-not-found=true", releaseName, namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete PVC: %v", err))
	}

	// 3. Delete secrets (if they exist)
	fmt.Printf("🧹 Cleaning up secrets for %s...\n", releaseName)
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl delete secret %s -n %s --ignore-not-found=true", releaseName, namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete secret: %v", err))
	}

	// 4. Delete configmaps (if they exist)
	fmt.Printf("🧹 Cleaning up configmaps for %s...\n", releaseName)
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl delete configmap %s-setup-script -n %s --ignore-not-found=true", releaseName, namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete setup-script configmap: %v", err))
	}

	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl delete configmap %s-vscode-settings -n %s --ignore-not-found=true", releaseName, namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete vscode-settings configmap: %v", err))
	}

	// 5. Delete any remaining pods (force delete if necessary)
	fmt.Printf("🧹 Cleaning up remaining pods for %s...\n", releaseName)
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("kubectl delete pods -l app.kubernetes.io/instance=%s -n %s --ignore-not-found=true --force --grace-period=0", releaseName, namespace))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete pods: %v", err))
	}

	// 6. Clean up temporary files
	fmt.Printf("🧹 Cleaning up temporary files...\n")
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("rm -f /tmp/%s-values.yaml", releaseName))
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete values file: %v", err))
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup completed with errors: %s", strings.Join(cleanupErrors, "; "))
	}

	return nil
}
