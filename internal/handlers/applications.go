package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// ApplicationsHandler contains dependencies for application-related operations
type ApplicationsHandler struct {
	// Add dependencies here as needed
}

// NewApplicationsHandler creates a new applications handler instance
func NewApplicationsHandler() *ApplicationsHandler {
	return &ApplicationsHandler{}
}

// HandleApplicationsPage renders the applications management page
func (h *ApplicationsHandler) HandleApplicationsPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
		return
	}

	// Get applications list
	applications, err := h.getApplicationsList(token, accountID)
	if err != nil {
		log.Printf("Error getting applications: %v", err)
		applications = []models.Application{}
	}

	// Get predefined applications catalog
	predefinedApps := models.GetPredefinedApplications()

	c.HTML(http.StatusOK, "applications.html", gin.H{
		"Applications":   applications,
		"PredefinedApps": predefinedApps,
		"ActivePage":     "applications",
	})
}

// HandleApplicationsList returns a JSON list of applications
func (h *ApplicationsHandler) HandleApplicationsList(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Get applications list
	applications, err := h.getApplicationsList(token, accountID)
	if err != nil {
		log.Printf("Error getting applications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
	})
}

// HandleApplicationsPrerequisites returns prerequisites for creating applications
func (h *ApplicationsHandler) HandleApplicationsPrerequisites(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	kvService := services.NewKVService()

	// Get managed domains from KV (those with SSL config)
	sslConfigs, err := kvService.ListDomainSSLConfigs(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get SSL configs"})
		return
	}

	// Convert to domain list
	managedDomains := []gin.H{}
	for domain := range sslConfigs {
		managedDomains = append(managedDomains, gin.H{
			"name": domain,
		})
	}

	// Get managed VPS from KV (those with VPS config)
	vpsConfigs, err := kvService.ListVPSConfigs(token, accountID)
	if err != nil {
		log.Printf("Error getting VPS configs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get VPS configs: %v", err)})
		return
	}

	// Convert to server list
	managedServers := []gin.H{}
	for serverID, config := range vpsConfigs {
		managedServers = append(managedServers, gin.H{
			"id":   fmt.Sprintf("%d", serverID),
			"name": config.Name,
			"public_net": gin.H{
				"ipv4": gin.H{
					"ip": config.PublicIPv4,
				},
			},
		})
	}

	// Get predefined applications catalog
	predefinedApps := models.GetPredefinedApplications()

	c.JSON(http.StatusOK, gin.H{
		"domains":         managedDomains,
		"servers":         managedServers,
		"predefined_apps": predefinedApps,
	})
}

// HandleApplicationsCreate creates new applications from predefined catalog
func (h *ApplicationsHandler) HandleApplicationsCreate(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Parse request body
	var appData struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		AppType     string `json:"app_type"`
		Subdomain   string `json:"subdomain"`
		Domain      string `json:"domain"`
		VPS         string `json:"vps"`
	}

	if err := c.ShouldBindJSON(&appData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate that app type exists in catalog
	predefinedApp, exists := models.GetPredefinedApplicationByID(appData.AppType)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application type"})
		return
	}

	// Create application
	app, err := h.createApplication(token, accountID, appData, predefinedApp)
	if err != nil {
		log.Printf("Error creating application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application"})
		return
	}

	response := gin.H{
		"success":     true,
		"message":     "Application created successfully",
		"application": app,
	}

	// For code-server applications, include the initial password
	if appData.AppType == "code-server" && app.Status == "deployed" {
		password, err := h.getCodeServerPassword(token, accountID, app.ID)
		if err == nil {
			response["initial_password"] = password
			response["password_info"] = "Save this password - you'll need it to access your code-server instance"
		}
	}

	c.JSON(http.StatusOK, response)
}

// HandleApplicationUpgrade upgrades existing applications to new versions
func (h *ApplicationsHandler) HandleApplicationUpgrade(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	appID := c.Param("id")
	var upgradeData struct {
		Version string `json:"version"`
	}

	if err := c.ShouldBindJSON(&upgradeData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Get application details to determine app type
	kvService := services.NewKVService()
	var app models.Application
	err = kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	// Validate version for code-server applications
	if app.AppType == "code-server" && upgradeData.Version != "latest" {
		valid, err := h.validateCodeServerVersion(upgradeData.Version)
		if err != nil {
			log.Printf("Error validating version: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate version"})
			return
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid version specified"})
			return
		}
	}

	// Upgrade application
	err = h.upgradeApplication(token, accountID, appID, upgradeData.Version)
	if err != nil {
		log.Printf("Error upgrading application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Application upgraded successfully",
	})
}

// HandleApplicationVersions returns available versions for an application type
func (h *ApplicationsHandler) HandleApplicationVersions(c *gin.Context) {
	appType := c.Param("app_type")

	// Currently only supporting code-server version lookup
	if appType != "code-server" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Version lookup only supported for code-server applications",
		})
		return
	}

	githubService := services.NewGitHubService()
	releases, err := githubService.GetCodeServerVersions(20) // Get last 20 releases
	if err != nil {
		log.Printf("Error fetching code-server versions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch version information",
		})
		return
	}

	// Convert to VersionInfo format
	var versions []models.VersionInfo
	for i, release := range releases {
		// GitHub releases use tags like "v4.101.2", but Docker images use "4.101.2"
		// Strip the "v" prefix for Docker compatibility
		dockerTag := strings.TrimPrefix(release.TagName, "v")
		
		versionInfo := models.VersionInfo{
			Version:     dockerTag, // Use Docker-compatible version
			Name:        release.Name,
			IsLatest:    i == 0, // First release is latest
			IsStable:    !release.Prerelease,
			PublishedAt: release.PublishedAt,
			URL:         release.HTMLURL,
		}
		versions = append(versions, versionInfo)
	}

	response := models.VersionsResponse{
		Success:  true,
		Versions: versions,
	}

	c.JSON(http.StatusOK, response)
}

// HandleApplicationPasswordChange changes the password for code-server applications
func (h *ApplicationsHandler) HandleApplicationPasswordChange(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	appID := c.Param("id")
	var passwordData struct {
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&passwordData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if len(passwordData.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 8 characters long"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Get application details
	kvService := services.NewKVService()
	var app models.Application
	err = kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != "code-server" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password change is only supported for code-server applications"})
		return
	}

	// Update password
	err = h.updateCodeServerPassword(token, accountID, appID, passwordData.NewPassword, &app)
	if err != nil {
		log.Printf("Error updating code-server password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password updated successfully",
	})
}

// HandleApplicationPasswordGet retrieves the current password for code-server applications
func (h *ApplicationsHandler) HandleApplicationPasswordGet(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	appID := c.Param("id")

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Get application details
	kvService := services.NewKVService()
	var app models.Application
	err = kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != "code-server" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password retrieval is only supported for code-server applications"})
		return
	}

	// Get current password
	password, err := h.getCodeServerPassword(token, accountID, appID)
	if err != nil {
		log.Printf("Error retrieving code-server password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"password": password,
	})
}

// HandleApplicationDelete deletes applications and cleans up resources
func (h *ApplicationsHandler) HandleApplicationDelete(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	appID := c.Param("id")

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Delete application
	err = h.deleteApplication(token, accountID, appID)
	if err != nil {
		log.Printf("Error deleting application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Application deleted successfully",
	})
}

// getApplicationsList retrieves all applications from Cloudflare KV
func (h *ApplicationsHandler) getApplicationsList(token, accountID string) ([]models.Application, error) {
	kvService := services.NewKVService()

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

// createApplication implements core application creation logic with predefined apps
func (h *ApplicationsHandler) createApplication(token, accountID string, appData interface{}, predefinedApp *models.PredefinedApplication) (*models.Application, error) {
	// Generate unique ID for application
	appID := fmt.Sprintf("app-%d", time.Now().Unix())

	data := appData.(struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		AppType     string `json:"app_type"`
		Subdomain   string `json:"subdomain"`
		Domain      string `json:"domain"`
		VPS         string `json:"vps"`
	})

	// Get VPS details
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		return nil, err
	}

	hetznerService := services.NewHetznerService()
	servers, err := hetznerService.ListServers(hetznerKey)
	if err != nil {
		return nil, err
	}

	var vpsName string
	for _, server := range servers {
		if fmt.Sprintf("%d", server.ID) == data.VPS {
			vpsName = server.Name
			break
		}
	}

	app := &models.Application{
		ID:          appID,
		Name:        data.Name,
		Description: data.Description,
		AppType:     data.AppType,
		AppVersion:  predefinedApp.Version,
		Subdomain:   data.Subdomain,
		Domain:      data.Domain,
		VPSID:       data.VPS,
		VPSName:     vpsName,
		Namespace:   predefinedApp.HelmChart.Namespace,
		Status:      "pending",
		URL:         fmt.Sprintf("https://%s.%s", data.Subdomain, data.Domain),
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	// Store in KV
	kvService := services.NewKVService()

	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)
	if err != nil {
		return nil, err
	}

	// Deploy via Helm using predefined configuration
	err = h.deployPredefinedApplication(token, accountID, data, predefinedApp, appID)
	if err != nil {
		app.Status = "failed"
		log.Printf("Predefined application deployment failed: %v", err)
	} else {
		app.Status = "deployed"
	}

	// Update application status
	app.UpdatedAt = time.Now().Format(time.RFC3339)
	kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)

	return app, nil
}

// deployPredefinedApplication deploys a predefined application using its Helm configuration
func (h *ApplicationsHandler) deployPredefinedApplication(token, accountID string, appData interface{}, predefinedApp *models.PredefinedApplication, appID string) error {
	data := appData.(struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		AppType     string `json:"app_type"`
		Subdomain   string `json:"subdomain"`
		Domain      string `json:"domain"`
		VPS         string `json:"vps"`
	})

	kvService := services.NewKVService()
	helmService := services.NewHelmService()

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err := kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", data.VPS), &vpsConfig)
	if err != nil {
		return fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return fmt.Errorf("failed to get SSH private key: %v", err)
	}

	sshService := services.NewSSHService()
	vpsID, _ := strconv.Atoi(data.VPS)
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, vpsID)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Clone GitHub repository for code-server chart
	repoDir := "/tmp/code-server-chart"
	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("rm -rf %s && git clone %s %s", repoDir, predefinedApp.HelmChart.Repository, repoDir))
	if err != nil {
		return fmt.Errorf("failed to clone chart repository: %v", err)
	}

	// Prepare Helm values, replacing placeholders
	values := make(map[string]string)
	releaseName := fmt.Sprintf("%s-%s", data.Subdomain, appID)
	
	for key, value := range predefinedApp.HelmChart.Values {
		valueStr := h.convertValueToString(value)
		// Replace placeholders
		valueStr = strings.ReplaceAll(valueStr, "{{SUBDOMAIN}}", data.Subdomain)
		valueStr = strings.ReplaceAll(valueStr, "{{DOMAIN}}", data.Domain)
		valueStr = strings.ReplaceAll(valueStr, "{{RELEASE_NAME}}", releaseName)
		values[key] = valueStr
	}

	// For code-server apps, create VS Code settings ConfigMap for persistence
	if data.AppType == "code-server" {
		err = h.createVSCodeSettingsConfigMap(conn, releaseName, predefinedApp.HelmChart.Namespace)
		if err != nil {
			log.Printf("Warning: Failed to create VS Code settings ConfigMap: %v", err)
		}
	}

	// Install Helm chart
	chartName := fmt.Sprintf("%s/%s", repoDir, predefinedApp.HelmChart.Chart)

	err = helmService.InstallChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		chartName,
		predefinedApp.HelmChart.Version,
		predefinedApp.HelmChart.Namespace,
		values,
	)

	if err != nil {
		return fmt.Errorf("failed to install Helm chart: %v", err)
	}

	// For code-server apps, retrieve and store the auto-generated password
	if data.AppType == "code-server" {
		// Wait a bit for the secret to be fully created
		time.Sleep(5 * time.Second)
		err = h.retrieveAndStoreCodeServerPassword(token, accountID, appID, releaseName, predefinedApp.HelmChart.Namespace, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
		if err != nil {
			log.Printf("Warning: Failed to retrieve code-server password: %v", err)
		}
	}

	return nil
}

// convertValueToString converts interface{} to string for Helm values
func (h *ApplicationsHandler) convertValueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case map[string]interface{}:
		// For nested objects, convert to JSON
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes)
	case []interface{}:
		// For arrays, convert to Helm --set format: {value1,value2}
		if len(v) == 0 {
			return "{}"
		}
		var items []string
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return "{" + strings.Join(items, ",") + "}"
	case []string:
		// For string arrays, convert to Helm --set format: {value1,value2}
		if len(v) == 0 {
			return "{}"
		}
		return "{" + strings.Join(v, ",") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// createVSCodeSettingsConfigMap creates a ConfigMap with default VS Code settings for persistence
func (h *ApplicationsHandler) createVSCodeSettingsConfigMap(conn *services.SSHConnection, releaseName, namespace string) error {
	sshService := services.NewSSHService()
	
	// Default VS Code settings with theme persistence and other user preferences
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

	// Create ConfigMap with the settings
	configMapName := fmt.Sprintf("%s-vscode-settings", releaseName)
	createConfigMapCmd := fmt.Sprintf(`kubectl create configmap %s -n %s --from-literal=settings.json='%s' --dry-run=client -o yaml | kubectl apply -f -`,
		configMapName, namespace, settingsJSON)
	
	_, err := sshService.ExecuteCommand(conn, createConfigMapCmd)
	if err != nil {
		return fmt.Errorf("failed to create VS Code settings ConfigMap: %v", err)
	}
	
	log.Printf("Created VS Code settings ConfigMap: %s in namespace %s", configMapName, namespace)
	return nil
}

// upgradeApplication implements core application upgrade logic
func (h *ApplicationsHandler) upgradeApplication(token, accountID, appID, version string) error {
	kvService := services.NewKVService()

	// Get current application
	var app models.Application
	err := kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		return err
	}

	// Get predefined app configuration
	predefinedApp, exists := models.GetPredefinedApplicationByID(app.AppType)
	if !exists {
		return fmt.Errorf("predefined application type not found: %s", app.AppType)
	}

	// Update version
	app.AppVersion = version
	app.Status = "pending"
	app.UpdatedAt = time.Now().Format(time.RFC3339)

	// Store updated app
	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)
	if err != nil {
		return err
	}

	// Perform Helm upgrade
	helmService := services.NewHelmService()

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err = kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", app.VPSID), &vpsConfig)
	if err != nil {
		return fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return fmt.Errorf("failed to get SSH private key: %v", err)
	}

	// Clone GitHub repository for code-server chart (needed for upgrade)
	repoDir := "/tmp/code-server-chart-upgrade"
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(app.VPSID)
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("rm -rf %s && git clone %s %s", repoDir, predefinedApp.HelmChart.Repository, repoDir))
	if err != nil {
		return fmt.Errorf("failed to clone chart repository for upgrade: %v", err)
	}

	// Prepare Helm values with updated version
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.ID)
	values := make(map[string]string)
	for key, value := range predefinedApp.HelmChart.Values {
		valueStr := h.convertValueToString(value)
		// Replace placeholders
		valueStr = strings.ReplaceAll(valueStr, "{{SUBDOMAIN}}", app.Subdomain)
		valueStr = strings.ReplaceAll(valueStr, "{{DOMAIN}}", app.Domain)
		valueStr = strings.ReplaceAll(valueStr, "{{RELEASE_NAME}}", releaseName)
		values[key] = valueStr
	}

	// For code-server apps, ensure VS Code settings ConfigMap exists
	if app.AppType == "code-server" {
		err = h.createVSCodeSettingsConfigMap(conn, releaseName, app.Namespace)
		if err != nil {
			log.Printf("Warning: Failed to create/update VS Code settings ConfigMap: %v", err)
		}
	}

	// Update the image tag with the new version for code-server
	if app.AppType == "code-server" && version != "latest" {
		values["image.tag"] = version
	} else if version == "latest" {
		// For "latest", we can remove the explicit tag to use chart default
		// or fetch the actual latest version
		githubService := services.NewGitHubService()
		if latestRelease, err := githubService.GetCodeServerLatestVersion(); err == nil {
			latestTag := strings.TrimPrefix(latestRelease.TagName, "v")
			values["image.tag"] = latestTag
		}
	}

	// Upgrade Helm chart
	chartName := fmt.Sprintf("%s/%s", repoDir, predefinedApp.HelmChart.Chart)

	err = helmService.UpgradeChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		chartName,
		predefinedApp.HelmChart.Version, // Use chart version, not app version
		app.Namespace,
		values,
	)

	if err != nil {
		app.Status = "failed"
		log.Printf("Helm upgrade failed: %v", err)
	} else {
		app.Status = "deployed"
	}

	// Update application status
	app.UpdatedAt = time.Now().Format(time.RFC3339)
	kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)

	return nil
}

// retrieveAndStoreCodeServerPassword retrieves the auto-generated password from Kubernetes secret
func (h *ApplicationsHandler) retrieveAndStoreCodeServerPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Retrieve password from Kubernetes secret
	// The secret name follows the pattern: {release-name}-code-server
	secretName := fmt.Sprintf("%s-code-server", releaseName)

	// First, let's check if the secret exists and list available secrets for debugging
	listCmd := fmt.Sprintf("kubectl get secrets --namespace %s", namespace)
	listResult, err := sshService.ExecuteCommand(conn, listCmd)
	if err != nil {
		log.Printf("Debug: Failed to list secrets in namespace %s: %v", namespace, err)
	} else {
		log.Printf("Debug: Available secrets in namespace %s: %s", namespace, listResult.Output)
	}

	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' | base64 --decode", namespace, secretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to retrieve password from secret '%s' in namespace '%s': %v", secretName, namespace, err)
	}

	// Store password in KV storage (encrypted)
	kvService := services.NewKVService()
	encryptedPassword, err := utils.EncryptData(strings.TrimSpace(result.Output), token)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}

	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:password", appID), map[string]string{
		"password": encryptedPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to store password: %v", err)
	}

	return nil
}

// getCodeServerPassword retrieves the stored password for a code-server application
func (h *ApplicationsHandler) getCodeServerPassword(token, accountID, appID string) (string, error) {
	kvService := services.NewKVService()

	var passwordData struct {
		Password string `json:"password"`
	}

	err := kvService.GetValue(token, accountID, fmt.Sprintf("app:%s:password", appID), &passwordData)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve password: %v", err)
	}

	// Decrypt password
	password, err := utils.DecryptData(passwordData.Password, token)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %v", err)
	}

	return password, nil
}

// updateCodeServerPassword updates the password for a code-server application
func (h *ApplicationsHandler) updateCodeServerPassword(token, accountID, appID, newPassword string, app *models.Application) error {
	// Get VPS configuration for SSH details
	kvService := services.NewKVService()
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err := kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", app.VPSID), &vpsConfig)
	if err != nil {
		return fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return fmt.Errorf("failed to get SSH private key: %v", err)
	}

	// Connect to VPS
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(app.VPSID)
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Update the Kubernetes secret with new password
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.ID)
	secretName := fmt.Sprintf("%s-code-server", releaseName)
	encodedPassword := utils.Base64Encode(newPassword)
	cmd := fmt.Sprintf("kubectl patch secret --namespace %s %s -p '{\"data\":{\"password\":\"%s\"}}'", app.Namespace, secretName, encodedPassword)
	_, err = sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to update Kubernetes secret: %v", err)
	}

	// Restart the code-server deployment to pick up the new password
	// The deployment name follows the pattern: {release-name}-code-server
	deploymentName := fmt.Sprintf("%s-code-server", releaseName)
	restartCmd := fmt.Sprintf("kubectl rollout restart deployment --namespace %s %s", app.Namespace, deploymentName)
	_, err = sshService.ExecuteCommand(conn, restartCmd)
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %v", err)
	}

	// Update stored password in KV
	encryptedPassword, err := utils.EncryptData(newPassword, token)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %v", err)
	}

	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:password", appID), map[string]string{
		"password": encryptedPassword,
	})
	if err != nil {
		return fmt.Errorf("failed to store password: %v", err)
	}

	return nil
}

// deleteApplication implements core application deletion logic
func (h *ApplicationsHandler) deleteApplication(token, accountID, appID string) error {
	kvService := services.NewKVService()

	// Get application details before deletion
	var app models.Application
	err := kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		return fmt.Errorf("failed to get application: %v", err)
	}

	// Uninstall Helm chart before deleting from KV
	helmService := services.NewHelmService()

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err = kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", app.VPSID), &vpsConfig)
	if err != nil {
		log.Printf("Warning: Failed to get VPS configuration for cleanup: %v", err)
	} else {
		// Get SSH private key
		client := &http.Client{Timeout: 10 * time.Second}
		var csrConfig struct {
			PrivateKey string `json:"private_key"`
		}
		if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
			log.Printf("Warning: Failed to get SSH private key for cleanup: %v", err)
		} else {
			// Uninstall Helm chart
			releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.ID)
			err = helmService.UninstallChart(
				vpsConfig.PublicIPv4,
				vpsConfig.SSHUser,
				csrConfig.PrivateKey,
				releaseName,
				app.Namespace,
			)

			if err != nil {
				log.Printf("Warning: Failed to uninstall Helm chart: %v", err)
				// Continue with KV deletion even if Helm uninstall fails
			}
		}
	}

	// Delete from KV
	err = kvService.DeleteValue(token, accountID, fmt.Sprintf("app:%s", appID))
	if err != nil {
		return err
	}

	// Also delete the password if it's a code-server application
	if app.AppType == "code-server" {
		kvService.DeleteValue(token, accountID, fmt.Sprintf("app:%s:password", appID))
		// Don't fail if password deletion fails - it's not critical
	}

	return nil
}

// validateCodeServerVersion checks if a given version exists in GitHub releases
func (h *ApplicationsHandler) validateCodeServerVersion(version string) (bool, error) {
	githubService := services.NewGitHubService()
	releases, err := githubService.GetCodeServerVersions(50) // Check last 50 releases
	if err != nil {
		return false, err
	}

	// Check if the version exists in the releases
	// Handle both Docker format (4.101.2) and GitHub format (v4.101.2)
	for _, release := range releases {
		dockerTag := strings.TrimPrefix(release.TagName, "v")
		
		if dockerTag == version || release.TagName == version {
			return true, nil
		}
	}

	return false, nil
}
