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
type ApplicationDeploymentService struct {
	helmService *HelmService
	sshService  *SSHService
	kvService   *KVService
}

// NewApplicationDeploymentService creates a new ApplicationDeploymentService
func NewApplicationDeploymentService() *ApplicationDeploymentService {
	return &ApplicationDeploymentService{
		helmService: NewHelmService(),
		sshService:  NewSSHService(),
		kvService:   NewKVService(),
	}
}

// DeployApplication deploys an application using Helm and appropriate handlers
func (ads *ApplicationDeploymentService) DeployApplication(token, accountID string, appData interface{}, predefinedApp *models.PredefinedApplication, appID string) error {
	log.Printf("🚀 CLAUDE DEBUG: DeployApplication called for %s with type %s", appID, predefinedApp.ID)
	log.Printf("🚀 CLAUDE DEBUG: Helm config: %+v", predefinedApp.HelmChart)

	// Convert appData to map for easier access
	appDataMap, ok := appData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid application data format")
	}

	kvService := ads.kvService
	sshService := ads.sshService

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

	// Deploy based on application type and repository configuration
	var deployErr error
	helmConfig := predefinedApp.HelmChart

	log.Printf("DEBUG: Deployment decision - App ID: %s, Repository: '%s'", predefinedApp.ID, helmConfig.Repository)

	if predefinedApp.ID == "code-server" && helmConfig.Repository == "local" {
		log.Printf("DEBUG: Using LOCAL CHART for code-server")
		deployErr = ads.deployCodeServerWithLocalChart(conn, predefinedApp, releaseName, namespace, subdomain, domain, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	} else {
		log.Printf("DEBUG: Using EXTERNAL CHART - App: %s, Repo: %s", predefinedApp.ID, helmConfig.Repository)
		deployErr = ads.deployWithExternalChart(conn, predefinedApp, releaseName, namespace, subdomain, domain)
	}

	if deployErr != nil {
		return deployErr
	}

	// Retrieve and store password for applications that generate them
	if err := ads.retrieveApplicationPassword(token, accountID, predefinedApp.ID, appID, vpsID, releaseName, namespace, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey); err != nil {
		// Log the error but don't fail the deployment since the app is successfully deployed
		log.Printf("Warning: Failed to retrieve application password: %v", err)
	}

	return nil
}

// deployCodeServerWithLocalChart deploys code-server using the local Helm chart
func (ads *ApplicationDeploymentService) deployCodeServerWithLocalChart(conn *SSHConnection, predefinedApp *models.PredefinedApplication, releaseName, namespace, subdomain, domain, vpsIP, sshUser, privateKey string) error {
	// Get latest version
	versionService := NewDefaultVersionService()
	version, err := versionService.GetLatestVersion(predefinedApp.ID)
	if err != nil {
		return fmt.Errorf("failed to get latest version: %v", err)
	}

	// Generate password
	password := utils.GenerateSecurePassword(20)

	// Prepare values for local chart
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "codercom/code-server",
			"tag":        version,
		},
		"password": password,
		"persistence": map[string]interface{}{
			"enabled": true,
			"size":    "10Gi",
		},
		"setupScript": map[string]interface{}{
			"enabled": true,
		},
		"vscodeSettings": map[string]interface{}{
			"enabled": true,
		},
	}

	// Copy local chart to VPS
	chartPath := "/tmp/xanthus-code-server"
	if err := ads.copyLocalChartToVPS(conn, chartPath); err != nil {
		return fmt.Errorf("failed to copy local chart to VPS: %v", err)
	}

	// Write values file
	valuesPath := fmt.Sprintf("/tmp/%s-values.yaml", releaseName)
	if err := ads.writeValuesFile(conn, valuesPath, values); err != nil {
		return fmt.Errorf("failed to write values file: %v", err)
	}

	// Install using Helm (first check if release exists)
	checkCmd := fmt.Sprintf("helm list -n %s | grep %s", namespace, releaseName)
	result, err := ads.sshService.ExecuteCommand(conn, checkCmd)

	if err != nil || strings.TrimSpace(result.Output) == "" {
		// Release doesn't exist, install it
		return ads.helmService.InstallChart(
			vpsIP,
			sshUser,
			privateKey,
			releaseName,
			chartPath,
			version,
			namespace,
			valuesPath,
		)
	} else {
		// Release exists, upgrade it
		return ads.helmService.UpgradeChart(
			vpsIP,
			sshUser,
			privateKey,
			releaseName,
			chartPath,
			version,
			namespace,
			valuesPath,
		)
	}
}

// deployWithExternalChart deploys applications using external charts (ArgoCD, etc.)
func (ads *ApplicationDeploymentService) deployWithExternalChart(conn *SSHConnection, predefinedApp *models.PredefinedApplication, releaseName, namespace, subdomain, domain string) error {
	var chartName string

	// Handle different chart repository types based on HelmChart configuration
	helmConfig := predefinedApp.HelmChart

	if strings.Contains(helmConfig.Repository, "github.com") {
		// Clone GitHub repository for the chart
		repoDir := fmt.Sprintf("/tmp/%s-chart", predefinedApp.ID)
		_, err := ads.sshService.ExecuteCommand(conn, fmt.Sprintf("rm -rf %s && git clone %s %s", repoDir, helmConfig.Repository, repoDir))
		if err != nil {
			return fmt.Errorf("failed to clone chart repository: %v", err)
		}
		chartName = fmt.Sprintf("%s/%s", repoDir, helmConfig.Chart)
	} else {
		// Add Helm repository
		repoName := predefinedApp.ID
		_, err := ads.sshService.ExecuteCommand(conn, fmt.Sprintf("helm repo add %s %s", repoName, helmConfig.Repository))
		if err != nil {
			return fmt.Errorf("failed to add Helm repository: %v", err)
		}

		// Update Helm repositories
		_, err = ads.sshService.ExecuteCommand(conn, "helm repo update")
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
	_, err = ads.sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", valuesPath, valuesContent))
	if err != nil {
		return fmt.Errorf("failed to upload values file: %v", err)
	}

	// Create ConfigMaps for code-server applications before Helm deployment
	if predefinedApp.ID == "code-server" {
		if err := ads.createCodeServerConfigMaps(conn, releaseName, namespace); err != nil {
			return fmt.Errorf("failed to create code-server ConfigMaps: %v", err)
		}
	}

	// Install via Helm
	installCmd := fmt.Sprintf("helm install %s %s --namespace %s --values %s --wait --timeout 10m",
		releaseName, chartName, namespace, valuesPath)

	result, err := ads.sshService.ExecuteCommand(conn, installCmd)
	if err != nil {
		return fmt.Errorf("helm install failed: %v, output: %s", err, result.Output)
	}

	return nil
}

// writeValuesFile writes a values map to a YAML file on the remote server
func (ads *ApplicationDeploymentService) writeValuesFile(conn *SSHConnection, filePath string, values map[string]interface{}) error {
	// Convert values to YAML
	yamlContent, err := utils.ConvertToYAML(values)
	if err != nil {
		return fmt.Errorf("failed to convert values to YAML: %v", err)
	}

	// Write to remote file
	_, err = ads.sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", filePath, yamlContent))
	if err != nil {
		return fmt.Errorf("failed to write values file: %v", err)
	}

	return nil
}

// copyLocalChartToVPS copies the local Helm chart to the VPS
func (ads *ApplicationDeploymentService) copyLocalChartToVPS(conn *SSHConnection, remotePath string) error {
	localChartPath := "charts/xanthus-code-server"

	// Create remote directory
	_, err := ads.sshService.ExecuteCommand(conn, fmt.Sprintf("mkdir -p %s", remotePath))
	if err != nil {
		return fmt.Errorf("failed to create remote chart directory: %v", err)
	}

	// Copy Chart.yaml
	if err := ads.copyFileToVPS(conn, fmt.Sprintf("%s/Chart.yaml", localChartPath), fmt.Sprintf("%s/Chart.yaml", remotePath)); err != nil {
		return fmt.Errorf("failed to copy Chart.yaml: %v", err)
	}

	// Copy values.yaml
	if err := ads.copyFileToVPS(conn, fmt.Sprintf("%s/values.yaml", localChartPath), fmt.Sprintf("%s/values.yaml", remotePath)); err != nil {
		return fmt.Errorf("failed to copy values.yaml: %v", err)
	}

	// Create templates directory
	_, err = ads.sshService.ExecuteCommand(conn, fmt.Sprintf("mkdir -p %s/templates", remotePath))
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
	}

	for _, file := range templateFiles {
		localFile := fmt.Sprintf("%s/templates/%s", localChartPath, file)
		remoteFile := fmt.Sprintf("%s/templates/%s", remotePath, file)
		if err := ads.copyFileToVPS(conn, localFile, remoteFile); err != nil {
			return fmt.Errorf("failed to copy template %s: %v", file, err)
		}
	}

	return nil
}

// copyFileToVPS copies a local file to the VPS via SSH
func (ads *ApplicationDeploymentService) copyFileToVPS(conn *SSHConnection, localPath, remotePath string) error {
	// Read local file
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file %s: %v", localPath, err)
	}

	// Write to remote file using cat with heredoc
	_, err = ads.sshService.ExecuteCommand(conn, fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", remotePath, string(content)))
	if err != nil {
		return fmt.Errorf("failed to write remote file %s: %v", remotePath, err)
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

	// For code-server applications, override configuration to use shared ConfigMap and external chart format
	if predefinedApp.ID == "code-server" {
		content += `

# External chart overrides - use extraConfigmapMounts for ConfigMaps
extraConfigmapMounts:
  - name: vscode-settings
    mountPath: /tmp/vscode-settings
    configMap: "` + releaseName + `-vscode-settings"
    readOnly: true
  - name: setup-script
    mountPath: /tmp/setup-script
    configMap: "code-server-setup-script"
    readOnly: true

# External chart overrides - use extraInitContainers
extraInitContainers: |
  - name: setup-environment
    image: ubuntu:22.04
    imagePullPolicy: IfNotPresent
    command:
      - bash
      - -c
      - |
        set -e
        echo "🚀 Starting basic code-server environment setup..."
        
        # Create user if it doesn't exist and setup home directory
        if ! id -u coder > /dev/null 2>&1; then
          useradd -m -u 1000 -s /bin/bash coder
        fi
        
        # Copy development setup script to user home directory
        if [ -f /tmp/setup-script/setup-dev-environment.sh ]; then
          echo "📝 Copying development setup script..."
          cp /tmp/setup-script/setup-dev-environment.sh /home/coder/setup-dev-environment.sh
          chmod +x /home/coder/setup-dev-environment.sh
        fi
        
        # Setup basic environment in bashrc
        cat >> /home/coder/.bashrc << 'BASHRC_EOF'
        # Xanthus Code-Server Environment
        echo "🎉 Welcome to your Xanthus Code-Server environment!"
        echo "📝 To install additional development tools, run:"
        echo "    ./setup-dev-environment.sh"
        echo ""
        BASHRC_EOF
        
        # Create basic directories
        mkdir -p /home/coder/workspace /home/coder/.local/share/code-server/User
        
        # Setup VS Code settings if available
        if [ -d /tmp/vscode-settings ]; then
          echo "📝 Copying VS Code settings..."
          cp -f /tmp/vscode-settings/settings.json /home/coder/.local/share/code-server/User/settings.json 2>/dev/null || echo "No settings.json found, skipping..."
          cp -f /tmp/vscode-settings/keybindings.json /home/coder/.local/share/code-server/User/keybindings.json 2>/dev/null || echo "No keybindings.json found, skipping..."
        fi
        
        # Fix all permissions
        echo "🔒 Fixing permissions..."
        chown -R 1000:1000 /home/coder
        
        echo "🎉 Basic environment setup complete!"
        echo "📝 Development tools can be installed by running: ./setup-dev-environment.sh"
    securityContext:
      runAsUser: 0
    volumeMounts:
      - name: data
        mountPath: /home/coder
      - name: vscode-settings
        mountPath: /tmp/vscode-settings
        readOnly: true
      - name: setup-script
        mountPath: /tmp/setup-script
        readOnly: true
`
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

// createCodeServerConfigMaps creates ConfigMaps for code-server applications
func (ads *ApplicationDeploymentService) createCodeServerConfigMaps(conn *SSHConnection, releaseName, namespace string) error {
	sshService := NewSSHService()

	// Create VS Code settings ConfigMap (per-instance)
	settingsJSON := `{
    "workbench.colorTheme": "Default Dark+",
    "workbench.iconTheme": "vs-seti",
    "editor.fontSize": 14,
    "editor.tabSize": 4,
    "editor.insertSpaces": true,
    "editor.detectIndentation": true,
    "editor.renderWhitespace": "selection",
    "editor.rulers": [80, 120],
    "files.autoSave": "afterDelay",
    "files.autoSaveDelay": 1000,
    "explorer.confirmDelete": false,
    "explorer.confirmDragAndDrop": false,
    "git.enableSmartCommit": true,
    "git.confirmSync": false,
    "terminal.integrated.fontSize": 14,
    "workbench.startupEditor": "newUntitledFile"
}`

	settingsConfigMapName := fmt.Sprintf("%s-vscode-settings", releaseName)
	createSettingsConfigMapCmd := fmt.Sprintf(`kubectl create configmap %s -n %s --from-literal=settings.json='%s' --dry-run=client -o yaml | kubectl apply -f -`,
		settingsConfigMapName, namespace, settingsJSON)

	_, err := sshService.ExecuteCommand(conn, createSettingsConfigMapCmd)
	if err != nil {
		return fmt.Errorf("failed to create VS Code settings ConfigMap: %v", err)
	}

	log.Printf("Created VS Code settings ConfigMap: %s in namespace %s", settingsConfigMapName, namespace)

	// Create shared setup script ConfigMap (once per namespace)
	sharedScriptConfigMapName := "code-server-setup-script"

	// Check if shared ConfigMap already exists
	checkConfigMapCmd := fmt.Sprintf("kubectl get configmap %s -n %s", sharedScriptConfigMapName, namespace)
	_, err = sshService.ExecuteCommand(conn, checkConfigMapCmd)

	if err != nil {
		// ConfigMap doesn't exist, create it
		setupScriptPath := "/home/coder/Projects/xanthus/internal/templates/applications/setup-dev-environment.sh"
		setupScriptContent, err := os.ReadFile(setupScriptPath)
		if err != nil {
			return fmt.Errorf("failed to read setup script template: %v", err)
		}

		createScriptConfigMapCmd := fmt.Sprintf(`kubectl create configmap %s -n %s --from-literal=setup-dev-environment.sh=%s`,
			sharedScriptConfigMapName, namespace, string(setupScriptContent))

		_, err = sshService.ExecuteCommand(conn, createScriptConfigMapCmd)
		if err != nil {
			return fmt.Errorf("failed to create shared setup script ConfigMap: %v", err)
		}

		log.Printf("Created shared setup script ConfigMap: %s in namespace %s", sharedScriptConfigMapName, namespace)
	} else {
		log.Printf("Shared setup script ConfigMap already exists: %s in namespace %s", sharedScriptConfigMapName, namespace)
	}

	return nil
}
