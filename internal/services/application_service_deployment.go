package services

import (
	"fmt"
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

	// Configure SSL certificates on VPS - this is required for HTTPS access
	if err := s.configureVPSSSL(token, accountID, domain, vpsConfig, csrConfig); err != nil {
		return fmt.Errorf("failed to configure SSL certificates on VPS: %v", err)
	}

	// Configure DNS record for the application
	if err := s.configureApplicationDNS(token, subdomain, domain, vpsConfig.PublicIPv4); err != nil {
		return fmt.Errorf("failed to configure DNS for application: %v", err)
	}

	// Retrieve and store password for applications that generate them
	if err := s.retrieveApplicationPassword(token, accountID, predefinedApp.ID, appID, vpsID, releaseName, namespace, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey); err != nil {
		// Log the error but don't fail the deployment since the app is successfully deployed
		fmt.Printf("Warning: Failed to retrieve application password: %v\n", err)
	}

	return nil
}

// configureVPSSSL configures SSL certificates on the VPS for the given domain
func (s *SimpleApplicationService) configureVPSSSL(token, accountID, domain string, vpsConfig struct {
	PublicIPv4 string `json:"public_ipv4"`
	SSHUser    string `json:"ssh_user"`
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
