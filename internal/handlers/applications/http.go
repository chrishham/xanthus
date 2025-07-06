package applications

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/gin-gonic/gin"
)

// HandleApplicationsPage renders the applications management page
func (h *Handler) HandleApplicationsPage(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	// Get applications list using service
	appService := h.GetApplicationService()
	applications, err := appService.ListApplications(token, accountID)
	if err != nil {
		log.Printf("Error getting applications: %v", err)
		applications = []models.Application{}
	}

	// Get predefined applications catalog
	predefinedApps := h.catalog.GetApplications()

	c.HTML(http.StatusOK, "applications.html", gin.H{
		"Applications":   applications,
		"PredefinedApps": predefinedApps,
		"ActivePage":     "applications",
	})
}

// HandleApplicationsList returns a JSON list of applications
func (h *Handler) HandleApplicationsList(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	// Get applications list using service
	appService := h.GetApplicationService()
	applications, err := appService.ListApplications(token, accountID)
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
func (h *Handler) HandleApplicationsPrerequisites(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

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
	predefinedApps := h.catalog.GetApplications()

	c.JSON(http.StatusOK, gin.H{
		"domains":         managedDomains,
		"servers":         managedServers,
		"predefined_apps": predefinedApps,
	})
}

// HandleApplicationsCreate creates new applications from predefined catalog
func (h *Handler) HandleApplicationsCreate(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

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

	// Validate application data
	validator := NewValidationHelper()
	if err := validator.ValidateApplicationData(appData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if subdomain is already taken
	if err := validator.ValidateSubdomainAvailability(token, accountID, appData.Subdomain, appData.Domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that app type exists in catalog
	predefinedApp, exists := h.catalog.GetApplicationByID(appData.AppType)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application type"})
		return
	}

	// Look up VPS name from VPS ID
	vpsHelper := NewVPSConnectionHelper()
	vpsConfig, err := vpsHelper.GetVPSConfigByID(token, accountID, appData.VPS)
	if err != nil {
		log.Printf("Failed to get VPS config for ID %s: %v", appData.VPS, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid VPS selection"})
		return
	}

	// Convert struct to map for service compatibility
	appDataMap := map[string]interface{}{
		"subdomain":   appData.Subdomain,
		"domain":      appData.Domain,
		"vps_id":      appData.VPS,
		"vps_name":    vpsConfig.Name,
		"description": appData.Description,
	}

	// Create application using service
	appService := h.GetApplicationService()
	app, err := appService.CreateApplication(token, accountID, appDataMap, predefinedApp)
	if err != nil {
		log.Printf("Error creating application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application"})
		return
	}

	response := gin.H{
		"success":     true,
		"message":     SuccessMessages.ApplicationCreated,
		"application": app,
	}

	// For code-server applications, include the initial password
	if appData.AppType == string(TypeCodeServer) && app.Status == string(StatusDeployed) {
		passwordHelper := NewPasswordHelper()
		password, err := passwordHelper.GetDecryptedPassword(token, accountID, app.ID)
		if err == nil {
			response["initial_password"] = password
			response["password_info"] = "Save this password - you'll need it to access your code-server instance"
		}
	}

	// For ArgoCD applications, include the initial admin password
	if appData.AppType == string(TypeArgoCD) && app.Status == string(StatusDeployed) {
		passwordHelper := NewPasswordHelper()
		password, err := passwordHelper.GetDecryptedPassword(token, accountID, app.ID)
		if err == nil {
			response["initial_password"] = password
			response["password_info"] = "Save this admin password - you'll need it to access your ArgoCD instance (username: admin)"
			response["username"] = "admin"
		}
	}

	c.JSON(http.StatusOK, response)
}

// HandleApplicationUpgrade upgrades existing applications to new versions
func (h *Handler) HandleApplicationUpgrade(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	appID := c.Param("id")
	var upgradeData struct {
		Version string `json:"version"`
	}

	if err := c.ShouldBindJSON(&upgradeData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Get application details using helper
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	// Validate version for code-server applications
	if app.AppType == string(TypeCodeServer) && upgradeData.Version != "latest" {
		codeServerHandler := NewCodeServerHandlers()
		valid, err := codeServerHandler.ValidateVersion(upgradeData.Version)
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

	// Upgrade application using service
	deploymentService := services.NewApplicationDeploymentService()
	err = deploymentService.UpgradeApplication(token, accountID, appID, upgradeData.Version)
	if err != nil {
		log.Printf("Error upgrading application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": SuccessMessages.ApplicationUpdated,
	})
}

// HandleApplicationVersions returns available versions for an application type
func (h *Handler) HandleApplicationVersions(c *gin.Context) {
	appType := c.Param("app_type")

	// Currently only supporting code-server version lookup
	if appType != string(TypeCodeServer) {
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
func (h *Handler) HandleApplicationPasswordChange(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	appID := c.Param("id")
	var passwordData struct {
		NewPassword string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&passwordData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate password
	validator := NewValidationHelper()
	if err := validator.ValidatePassword(passwordData.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != string(TypeCodeServer) && app.AppType != string(TypeArgoCD) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password change is only supported for code-server and ArgoCD applications"})
		return
	}

	// Update password based on application type using specific handlers
	if app.AppType == string(TypeCodeServer) {
		codeServerHandler := NewCodeServerHandlers()
		err = codeServerHandler.UpdatePassword(token, accountID, appID, passwordData.NewPassword, struct {
			VPSID     string
			Subdomain string
			ID        string
			Namespace string
		}{app.VPSID, app.Subdomain, app.ID, app.Namespace})
	} else if app.AppType == string(TypeArgoCD) {
		argoCDHandler := NewArgoCDHandlers()
		err = argoCDHandler.UpdatePassword(token, accountID, appID, passwordData.NewPassword, struct {
			VPSID     string
			Subdomain string
			ID        string
			Namespace string
		}{app.VPSID, app.Subdomain, app.ID, app.Namespace})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrorMessages.InvalidApplicationType})
		return
	}

	if err != nil {
		log.Printf("Error updating %s password: %v", app.AppType, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": SuccessMessages.PasswordUpdated,
	})
}

// HandleApplicationPasswordGet retrieves the current password for code-server applications
func (h *Handler) HandleApplicationPasswordGet(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	appID := c.Param("id")

	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != string(TypeCodeServer) && app.AppType != string(TypeArgoCD) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password retrieval is only supported for code-server and ArgoCD applications"})
		return
	}

	var password string

	// Use application-specific password retrieval
	if app.AppType == string(TypeArgoCD) {
		argoCDHandler := NewArgoCDHandlers()
		password, err = argoCDHandler.GetPassword(token, accountID, appID, struct {
			VPSID     string
			Namespace string
		}{app.VPSID, app.Namespace})
	} else {
		// For other applications, use the generic password helper
		passwordHelper := NewPasswordHelper()
		password, err = passwordHelper.GetDecryptedPassword(token, accountID, appID)
	}

	if err != nil {
		log.Printf("Error retrieving %s password: %v", app.AppType, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve password"})
		return
	}

	response := gin.H{
		"success":  true,
		"password": password,
	}

	// For ArgoCD applications, include username information
	if app.AppType == string(TypeArgoCD) {
		response["username"] = "admin"
		response["login_info"] = "Username: admin (default ArgoCD admin user)"
	}

	c.JSON(http.StatusOK, response)
}

// HandleApplicationDelete deletes applications and cleans up resources
func (h *Handler) HandleApplicationDelete(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	appID := c.Param("id")

	// Delete application using service
	appService := h.GetApplicationService()
	err := appService.DeleteApplication(token, accountID, appID)
	if err != nil {
		log.Printf("Error deleting application: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": SuccessMessages.ApplicationDeleted,
	})
}

// HandlePortForwardsList returns the list of port forwards for an application
func (h *Handler) HandlePortForwardsList(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")
	appID := c.Param("id")

	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != string(TypeCodeServer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Port forwarding is only supported for code-server applications"})
		return
	}

	// Get port forwards using service
	portForwardService := NewPortForwardService()
	portForwards, err := portForwardService.ListPortForwards(token, accountID, appID)
	if err != nil {
		log.Printf("Error getting port forwards: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get port forwards"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"port_forwards": portForwards,
	})
}

// HandlePortForwardsCreate creates a new port forward for an application
func (h *Handler) HandlePortForwardsCreate(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")
	appID := c.Param("id")

	var portForwardData struct {
		Port      int    `json:"port"`
		Subdomain string `json:"subdomain"`
	}

	if err := c.ShouldBindJSON(&portForwardData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate input
	if portForwardData.Port < 1 || portForwardData.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Port must be between 1 and 65535"})
		return
	}

	if portForwardData.Subdomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subdomain is required"})
		return
	}

	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != string(TypeCodeServer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Port forwarding is only supported for code-server applications"})
		return
	}

	// Create port forward using service
	portForwardService := NewPortForwardService()
	portForward, err := portForwardService.CreatePortForward(token, accountID, appID, portForwardData.Port, portForwardData.Subdomain)
	if err != nil {
		log.Printf("Error creating port forward: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create port forward"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Port forward created successfully",
		"port_forward": portForward,
	})
}

// HandlePortForwardsDelete removes a port forward for an application
func (h *Handler) HandlePortForwardsDelete(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")
	appID := c.Param("id")
	portForwardID := c.Param("port_id")

	// Get application details
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	if app.AppType != string(TypeCodeServer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Port forwarding is only supported for code-server applications"})
		return
	}

	// Delete port forward using service
	portForwardService := NewPortForwardService()
	err = portForwardService.DeletePortForward(token, accountID, appID, portForwardID)
	if err != nil {
		log.Printf("Error deleting port forward: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete port forward"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Port forward removed successfully",
	})
}

// HandleApplicationToken retrieves authentication token for headlamp applications
func (h *Handler) HandleApplicationToken(c *gin.Context) {
	token := c.GetString("cf_token")
	accountID := c.GetString("account_id")

	appID := c.Param("id")

	// Get application details using helper
	appHelper := NewApplicationHelper()
	app, err := appHelper.GetApplicationByID(token, accountID, appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	// Only support token retrieval for headlamp applications
	if app.AppType != "headlamp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token retrieval is only supported for headlamp applications"})
		return
	}

	// Get VPS configuration using helper
	vpsHelper := NewVPSConnectionHelper()
	conn, err := vpsHelper.GetVPSConnection(token, accountID, app.VPSID)
	if err != nil {
		log.Printf("Error connecting to VPS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to VPS"})
		return
	}
	defer conn.Close()

	// Generate service account token for headlamp
	sshService := services.NewSSHService()
	createTokenCmd := fmt.Sprintf("kubectl create token headlamp-headlamp -n headlamp --duration=8760h")
	result, err := sshService.ExecuteCommand(conn, createTokenCmd)
	if err != nil {
		log.Printf("Error creating token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication token"})
		return
	}

	// Clean up token (remove any newlines)
	authToken := strings.TrimSpace(result.Output)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   authToken,
		"message": "Authentication token retrieved successfully",
	})
}
