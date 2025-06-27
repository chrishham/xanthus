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

	c.HTML(http.StatusOK, "applications.html", gin.H{
		"Applications": applications,
		"ActivePage":   "applications",
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get VPS configs"})
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

	c.JSON(http.StatusOK, gin.H{
		"domains": managedDomains,
		"servers": managedServers,
	})
}

// HandleApplicationsCreate creates new applications with Helm deployment
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
		Subdomain   string `json:"subdomain"`
		Domain      string `json:"domain"`
		VPS         string `json:"vps"`
		Chart       string `json:"chart"`
		Version     string `json:"version"`
	}

	if err := c.ShouldBindJSON(&appData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Create application
	app, err := h.createApplication(token, accountID, appData)
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

// createApplication implements core application creation logic with VPS integration
func (h *ApplicationsHandler) createApplication(token, accountID string, appData interface{}) (*models.Application, error) {
	// Generate unique ID for application
	appID := fmt.Sprintf("app-%d", time.Now().Unix())

	data := appData.(struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Subdomain   string `json:"subdomain"`
		Domain      string `json:"domain"`
		VPS         string `json:"vps"`
		Chart       string `json:"chart"`
		Version     string `json:"version"`
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
		ID:           appID,
		Name:         data.Name,
		Description:  data.Description,
		Subdomain:    data.Subdomain,
		Domain:       data.Domain,
		VPSID:        data.VPS,
		VPSName:      vpsName,
		ChartName:    data.Chart,
		ChartVersion: data.Version,
		Namespace:    data.Name, // Use app name as namespace
		Status:       "pending",
		CreatedAt:    time.Now().Format(time.RFC3339),
	}

	// Store in KV
	kvService := services.NewKVService()

	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)
	if err != nil {
		return nil, err
	}

	// Deploy via Helm
	helmService := services.NewHelmService()

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err = kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", data.VPS), &vpsConfig)
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

	// Install Helm chart
	releaseName := fmt.Sprintf("%s-%s", data.Subdomain, app.ID)
	values := map[string]string{
		"ingress.enabled":                    "true",
		"ingress.hosts[0].host":              fmt.Sprintf("%s.%s", data.Subdomain, data.Domain),
		"ingress.hosts[0].paths[0].path":     "/",
		"ingress.hosts[0].paths[0].pathType": "Prefix",
	}

	err = helmService.InstallChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		data.Chart,
		data.Version,
		app.Namespace,
		values,
	)

	if err != nil {
		app.Status = "failed"
		log.Printf("Helm deployment failed: %v", err)
	} else {
		app.Status = "deployed"
	}

	// Update application status
	kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)

	return app, nil
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

	// Update version
	app.ChartVersion = version
	app.Status = "pending"

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

	// Upgrade Helm chart
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.ID)
	values := map[string]string{
		"ingress.enabled":                    "true",
		"ingress.hosts[0].host":              fmt.Sprintf("%s.%s", app.Subdomain, app.Domain),
		"ingress.hosts[0].paths[0].path":     "/",
		"ingress.hosts[0].paths[0].pathType": "Prefix",
	}

	err = helmService.UpgradeChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		app.ChartName,
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

// HandleVPSRepositories lists Helm repositories on a VPS
func (h *ApplicationsHandler) HandleVPSRepositories(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Unauthorized")
		return
	}

	vpsID := c.Param("id")
	serverID, err := strconv.Atoi(vpsID)
	if err != nil {
		utils.JSONBadRequest(c, "Invalid VPS ID")
		return
	}

	log.Printf("Getting Helm repositories for VPS %s", vpsID)

	// Initialize services
	kvService := services.NewKVService()
	sshService := services.NewSSHService()

	// Get account ID from token
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration from KV
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Failed to get VPS config: %v", err)
		utils.JSONNotFound(c, "VPS not found")
		return
	}

	// Get SSH private key from KV
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Failed to get SSH private key: %v", err)
		utils.JSONInternalServerError(c, "SSH key not found")
		return
	}

	// Check for existing SSH session first
	sessionID := c.GetHeader("X-SSH-Session-ID")
	var conn *services.SSHConnection

	sessionManager := services.GetGlobalSessionManager()
	if sessionID != "" {
		if sessionConn, exists := sessionManager.GetSessionConnection(sessionID); exists {
			conn = sessionConn
			log.Printf("Using existing SSH session %s for VPS %s", sessionID, vpsID)
		}
	}

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, serverID)
		if err != nil {
			log.Printf("Failed to connect to VPS %s: %v", vpsID, err)
			utils.JSONServiceUnavailable(c, "Cannot connect to VPS")
			return
		}
		log.Printf("Created new SSH connection for VPS %s", vpsID)
	}

	// Get list of Helm repositories
	repositories, err := sshService.ListHelmRepositories(conn)
	if err != nil {
		log.Printf("Failed to list repositories: %v", err)
		utils.JSONInternalServerError(c, "Failed to list Helm repositories")
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"repositories": repositories,
		"vps_id":       vpsID,
	})
}

// HandleVPSAddRepository adds a new Helm repository to a VPS
func (h *ApplicationsHandler) HandleVPSAddRepository(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Unauthorized")
		return
	}

	vpsID := c.Param("id")
	serverID, err := strconv.Atoi(vpsID)
	if err != nil {
		utils.JSONBadRequest(c, "Invalid VPS ID")
		return
	}

	var repoData struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	if err := c.ShouldBindJSON(&repoData); err != nil {
		utils.JSONBadRequest(c, "Invalid request data")
		return
	}

	// Validate repository name and URL
	if repoData.Name == "" || repoData.URL == "" {
		utils.JSONBadRequest(c, "Repository name and URL are required")
		return
	}

	// Basic URL validation
	if !strings.HasPrefix(repoData.URL, "http://") && !strings.HasPrefix(repoData.URL, "https://") {
		utils.JSONBadRequest(c, "Invalid repository URL")
		return
	}

	// Sanitize repository name to prevent command injection
	if strings.ContainsAny(repoData.Name, ";|&$`(){}[]<>\"'\\") {
		utils.JSONBadRequest(c, "Invalid characters in repository name")
		return
	}

	log.Printf("Adding repository %s (%s) to VPS %s", repoData.Name, repoData.URL, vpsID)

	// Initialize services
	kvService := services.NewKVService()
	sshService := services.NewSSHService()

	// Get account ID from token
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration from KV
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Failed to get VPS config: %v", err)
		utils.JSONNotFound(c, "VPS not found")
		return
	}

	// Get SSH private key from KV
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Failed to get SSH private key: %v", err)
		utils.JSONInternalServerError(c, "SSH key not found")
		return
	}

	// Check for existing SSH session first
	sessionID := c.GetHeader("X-SSH-Session-ID")
	var conn *services.SSHConnection

	sessionManager := services.GetGlobalSessionManager()
	if sessionID != "" {
		if sessionConn, exists := sessionManager.GetSessionConnection(sessionID); exists {
			conn = sessionConn
			log.Printf("Using existing SSH session %s for VPS %s", sessionID, vpsID)
		}
	}

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, serverID)
		if err != nil {
			log.Printf("Failed to connect to VPS %s: %v", vpsID, err)
			utils.JSONServiceUnavailable(c, "Cannot connect to VPS")
			return
		}
		log.Printf("Created new SSH connection for VPS %s", vpsID)
	}

	// Add the repository
	if err := sshService.AddHelmRepository(conn, repoData.Name, repoData.URL); err != nil {
		log.Printf("Failed to add repository: %v", err)
		utils.JSONInternalServerError(c, "Failed to add Helm repository")
		return
	}

	log.Printf("Successfully added repository %s to VPS %s", repoData.Name, vpsID)
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Repository %s added successfully", repoData.Name),
	})
}

// HandleVPSCharts lists Helm charts from a repository
func (h *ApplicationsHandler) HandleVPSCharts(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Unauthorized")
		return
	}

	vpsID := c.Param("id")
	repo := c.Param("repo")

	serverID, err := strconv.Atoi(vpsID)
	if err != nil {
		utils.JSONBadRequest(c, "Invalid VPS ID")
		return
	}

	// Sanitize repository name to prevent command injection
	if strings.ContainsAny(repo, ";|&$`(){}[]<>\"'\\") {
		utils.JSONBadRequest(c, "Invalid characters in repository name")
		return
	}

	log.Printf("Getting charts for repository %s on VPS %s", repo, vpsID)

	// Initialize services
	kvService := services.NewKVService()
	sshService := services.NewSSHService()

	// Get account ID from token
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration from KV
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Failed to get VPS config: %v", err)
		utils.JSONNotFound(c, "VPS not found")
		return
	}

	// Get SSH private key from KV
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Failed to get SSH private key: %v", err)
		utils.JSONInternalServerError(c, "SSH key not found")
		return
	}

	// Check for existing SSH session first
	sessionID := c.GetHeader("X-SSH-Session-ID")
	var conn *services.SSHConnection

	sessionManager := services.GetGlobalSessionManager()
	if sessionID != "" {
		if sessionConn, exists := sessionManager.GetSessionConnection(sessionID); exists {
			conn = sessionConn
			log.Printf("Using existing SSH session %s for VPS %s", sessionID, vpsID)
		}
	}

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, serverID)
		if err != nil {
			log.Printf("Failed to connect to VPS %s: %v", vpsID, err)
			utils.JSONServiceUnavailable(c, "Cannot connect to VPS")
			return
		}
		log.Printf("Created new SSH connection for VPS %s", vpsID)
	}

	// List charts from the repository
	charts, err := sshService.ListHelmCharts(conn, repo)
	if err != nil {
		log.Printf("Failed to list charts: %v", err)
		utils.JSONInternalServerError(c, "Failed to list Helm charts")
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"charts":     charts,
		"repository": repo,
		"vps_id":     vpsID,
	})
}

// HandleCreateSSHSession creates a new SSH session for wizard operations
func (h *ApplicationsHandler) HandleCreateSSHSession(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Unauthorized")
		return
	}

	var sessionData struct {
		VPSID string `json:"vps_id"`
	}

	if err := c.ShouldBindJSON(&sessionData); err != nil {
		utils.JSONBadRequest(c, "Invalid request data")
		return
	}

	if sessionData.VPSID == "" {
		utils.JSONBadRequest(c, "VPS ID is required")
		return
	}

	serverID, err := strconv.Atoi(sessionData.VPSID)
	if err != nil {
		utils.JSONBadRequest(c, "Invalid VPS ID")
		return
	}

	log.Printf("Creating SSH session for VPS %s", sessionData.VPSID)

	// Initialize services
	kvService := services.NewKVService()
	sessionManager := services.GetGlobalSessionManager()

	// Get account ID from token
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration from KV
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Failed to get VPS config: %v", err)
		utils.JSONNotFound(c, "VPS not found")
		return
	}

	// Get SSH private key from KV
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Failed to get SSH private key: %v", err)
		utils.JSONInternalServerError(c, "SSH key not found")
		return
	}

	// Create SSH session
	sessionID, err := sessionManager.CreateSession(
		serverID,
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		token,
	)
	if err != nil {
		log.Printf("Failed to create SSH session: %v", err)
		utils.JSONInternalServerError(c, "Failed to create SSH session")
		return
	}

	log.Printf("Successfully created SSH session %s for VPS %s", sessionID, sessionData.VPSID)
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"session_id": sessionID,
		"vps_id":     sessionData.VPSID,
		"message":    "SSH session created successfully",
	})
}

// HandleCloseSSHSession closes an existing SSH session
func (h *ApplicationsHandler) HandleCloseSSHSession(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Unauthorized")
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.JSONBadRequest(c, "Session ID is required")
		return
	}

	sessionManager := services.GetGlobalSessionManager()
	sessionManager.RemoveSession(sessionID)

	log.Printf("Closed SSH session %s", sessionID)
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"message": "SSH session closed successfully",
	})
}
