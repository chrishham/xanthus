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

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Application created successfully",
		"application": app,
	})
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

	// Fetch each application
	for _, key := range keysResp.Result {
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
	for key, value := range predefinedApp.HelmChart.Values {
		valueStr := h.convertValueToString(value)
		// Replace placeholders
		valueStr = strings.ReplaceAll(valueStr, "{{SUBDOMAIN}}", data.Subdomain)
		valueStr = strings.ReplaceAll(valueStr, "{{DOMAIN}}", data.Domain)
		values[key] = valueStr
	}

	// Install Helm chart
	releaseName := fmt.Sprintf("%s-%s", data.Subdomain, appID)
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

	// Prepare Helm values
	values := make(map[string]string)
	for key, value := range predefinedApp.HelmChart.Values {
		valueStr := h.convertValueToString(value)
		// Replace placeholders
		valueStr = strings.ReplaceAll(valueStr, "{{SUBDOMAIN}}", app.Subdomain)
		valueStr = strings.ReplaceAll(valueStr, "{{DOMAIN}}", app.Domain)
		values[key] = valueStr
	}

	// Upgrade Helm chart
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.ID)
	chartName := fmt.Sprintf("coder/%s", predefinedApp.HelmChart.Chart)

	err = helmService.UpgradeChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		chartName,
		version,
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
	return kvService.DeleteValue(token, accountID, fmt.Sprintf("app:%s", appID))
}
