package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
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
	catalog        services.ApplicationCatalog
	validator      models.ApplicationValidator
	serviceFactory *services.ApplicationServiceFactory
}

// NewApplicationsHandler creates a new applications handler instance using the service layer
func NewApplicationsHandler() *ApplicationsHandler {
	factory := services.NewApplicationServiceFactory()
	return &ApplicationsHandler{
		catalog:        factory.CreateCatalogService(),
		validator:      factory.CreateValidatorService(),
		serviceFactory: factory,
	}
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
	predefinedApps := h.catalog.GetApplications()

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
	predefinedApps := h.catalog.GetApplications()

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
	predefinedApp, exists := h.catalog.GetApplicationByID(appData.AppType)
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

	// For ArgoCD applications, include the initial admin password
	if appData.AppType == "argocd" && app.Status == "deployed" {
		password, err := h.getArgoCDPassword(token, accountID, app.ID)
		if err == nil {
			response["initial_password"] = password
			response["password_info"] = "Save this admin password - you'll need it to access your ArgoCD instance (username: admin)"
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

	if app.AppType != "code-server" && app.AppType != "argocd" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password change is only supported for code-server and ArgoCD applications"})
		return
	}

	// Update password based on application type
	if app.AppType == "code-server" {
		err = h.updateCodeServerPassword(token, accountID, appID, passwordData.NewPassword, &app)
	} else if app.AppType == "argocd" {
		err = h.updateArgoCDPassword(token, accountID, appID, passwordData.NewPassword, &app)
	}

	if err != nil {
		log.Printf("Error updating %s password: %v", app.AppType, err)
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

	if app.AppType != "code-server" && app.AppType != "argocd" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password retrieval is only supported for code-server and ArgoCD applications"})
		return
	}

	// Get current password based on application type
	var password string
	if app.AppType == "code-server" {
		password, err = h.getCodeServerPassword(token, accountID, appID)
	} else if app.AppType == "argocd" {
		password, err = h.getArgoCDPassword(token, accountID, appID)
	}

	if err != nil {
		log.Printf("Error retrieving %s password: %v", app.AppType, err)
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

// getApplicationsList retrieves all applications from Cloudflare KV with real-time status
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
			// Update application status with real-time Helm status
			if realTimeStatus, statusErr := h.getRealTimeStatus(token, accountID, &app); statusErr == nil {
				app.Status = realTimeStatus
			}
			// If we can't get real-time status, keep the cached status

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

	// Generate values file from template with placeholder substitution
	releaseName := fmt.Sprintf("%s-%s", data.Subdomain, appID)
	valuesFilePath, err := h.generateValuesFile(conn, predefinedApp, data.Subdomain, data.Domain, releaseName)
	if err != nil {
		return fmt.Errorf("failed to generate values file: %v", err)
	}

	var chartName string

	// Handle different application deployment types
	switch data.AppType {
	case "code-server":
		// Clone GitHub repository for code-server chart
		repoDir := "/tmp/code-server-chart"
		_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("rm -rf %s && git clone %s %s", repoDir, predefinedApp.HelmChart.Repository, repoDir))
		if err != nil {
			return fmt.Errorf("failed to clone chart repository: %v", err)
		}
		chartName = fmt.Sprintf("%s/%s", repoDir, predefinedApp.HelmChart.Chart)

		// Create VS Code settings ConfigMap
		err = h.createVSCodeSettingsConfigMap(conn, releaseName, predefinedApp.HelmChart.Namespace)
		if err != nil {
			log.Printf("Warning: Failed to create VS Code settings ConfigMap: %v", err)
		}

	case "argocd":
		// Add Helm repository for ArgoCD
		repoName := "argo"
		_, err = sshService.ExecuteCommand(conn, fmt.Sprintf("helm repo add %s %s", repoName, predefinedApp.HelmChart.Repository))
		if err != nil {
			return fmt.Errorf("failed to add Helm repository: %v", err)
		}

		// Update Helm repositories
		_, err = sshService.ExecuteCommand(conn, "helm repo update")
		if err != nil {
			return fmt.Errorf("failed to update Helm repositories: %v", err)
		}
		chartName = fmt.Sprintf("%s/%s", repoName, predefinedApp.HelmChart.Chart)

		// Install ArgoCD CLI
		_, err = sshService.ExecuteCommand(conn, `
			ARCH=$(uname -m)
			case $ARCH in
				x86_64) ARGOCD_ARCH="amd64" ;;
				aarch64) ARGOCD_ARCH="arm64" ;;
				armv7l) ARGOCD_ARCH="armv7" ;;
				*) echo "Warning: Unsupported architecture $ARCH, defaulting to amd64"; ARGOCD_ARCH="amd64" ;;
			esac
			curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-${ARGOCD_ARCH}
			chmod +x /usr/local/bin/argocd
		`)
		if err != nil {
			log.Printf("Warning: Failed to install ArgoCD CLI: %v", err)
			// Don't fail the deployment, just log the warning
		}

	default:
		return fmt.Errorf("unsupported application type: %s", data.AppType)
	}

	err = helmService.InstallChart(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		csrConfig.PrivateKey,
		releaseName,
		chartName,
		predefinedApp.HelmChart.Version,
		predefinedApp.HelmChart.Namespace,
		valuesFilePath,
	)

	if err != nil {
		return fmt.Errorf("failed to install Helm chart: %v", err)
	}

	// Create TLS secret for ingress if domain SSL config is available
	if data.Domain != "" {
		// Get domain SSL configuration using the KV service
		kvService := services.NewKVService()
		domainConfig, err := kvService.GetDomainSSLConfig(token, accountID, data.Domain)
		if err == nil && domainConfig != nil {
			// Create TLS secret in the application namespace
			err = sshService.CreateTLSSecret(conn, data.Domain, domainConfig.Certificate, domainConfig.PrivateKey, predefinedApp.HelmChart.Namespace)
			if err != nil {
				log.Printf("Warning: Failed to create TLS secret for domain %s: %v", data.Domain, err)
				// Don't fail the deployment, but log the warning
			} else {
				log.Printf("✅ Created TLS secret for domain %s in namespace %s", data.Domain, predefinedApp.HelmChart.Namespace)
			}
		} else {
			log.Printf("Warning: No SSL configuration found for domain %s: %v", data.Domain, err)
			// Try to list available SSL configurations for debugging
			if configs, listErr := kvService.ListDomainSSLConfigs(token, accountID); listErr == nil {
				log.Printf("Debug: Available SSL configurations:")
				for domain := range configs {
					log.Printf("  - %s", domain)
				}
			} else {
				log.Printf("Debug: Failed to list SSL configurations: %v", listErr)
			}
		}
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

	// For ArgoCD apps, retrieve and store the auto-generated admin password
	if data.AppType == "argocd" {
		// Wait a bit for the ArgoCD deployment to be ready
		time.Sleep(10 * time.Second)
		err = h.retrieveAndStoreArgoCDPassword(token, accountID, appID, releaseName, predefinedApp.HelmChart.Namespace, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
		if err != nil {
			log.Printf("Warning: Failed to retrieve ArgoCD password: %v", err)
		}
	}

	return nil
}

// generateValuesFile creates a values file from template with placeholder substitution
func (h *ApplicationsHandler) generateValuesFile(conn *services.SSHConnection, predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName string) (string, error) {
	sshService := services.NewSSHService()

	// Read the values template file
	templatePath := filepath.Join("internal/templates/applications", predefinedApp.HelmChart.ValuesTemplate)
	templateContent, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read values template: %v", err)
	}

	// Perform placeholder substitution
	valuesContent := string(templateContent)
	valuesContent = strings.ReplaceAll(valuesContent, "{{SUBDOMAIN}}", subdomain)
	valuesContent = strings.ReplaceAll(valuesContent, "{{DOMAIN}}", domain)
	valuesContent = strings.ReplaceAll(valuesContent, "{{RELEASE_NAME}}", releaseName)

	// Apply additional placeholders from the predefined app configuration
	for key, value := range predefinedApp.HelmChart.Placeholders {
		valuesContent = strings.ReplaceAll(valuesContent, fmt.Sprintf("{{%s}}", key), value)
	}

	// Create a temporary values file on the VPS
	valuesFileName := fmt.Sprintf("/tmp/%s-values.yaml", releaseName)

	// Write the values content directly to the VPS using SSH
	createFileCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", valuesFileName, valuesContent)

	_, err = sshService.ExecuteCommand(conn, createFileCmd)
	if err != nil {
		return "", fmt.Errorf("failed to create values file on VPS: %v", err)
	}

	return valuesFileName, nil
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
	predefinedApp, exists := h.catalog.GetApplicationByID(app.AppType)
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

	// Generate values file from template with placeholder substitution and version update
	releaseName := fmt.Sprintf("%s-%s", app.Subdomain, app.ID)

	// Create a modified predefined app with updated version for values generation
	modifiedPredefinedApp := *predefinedApp
	if modifiedPredefinedApp.HelmChart.Placeholders == nil {
		modifiedPredefinedApp.HelmChart.Placeholders = make(map[string]string)
	}

	// Update the version placeholder for upgrade
	if app.AppType == "code-server" && version != "latest" {
		modifiedPredefinedApp.HelmChart.Placeholders["VERSION"] = version
	} else if version == "latest" {
		// For "latest", fetch the actual latest version
		githubService := services.NewGitHubService()
		if latestRelease, err := githubService.GetCodeServerLatestVersion(); err == nil {
			latestTag := strings.TrimPrefix(latestRelease.TagName, "v")
			modifiedPredefinedApp.HelmChart.Placeholders["VERSION"] = latestTag
		}
	}

	valuesFilePath, err := h.generateValuesFile(conn, &modifiedPredefinedApp, app.Subdomain, app.Domain, releaseName)
	if err != nil {
		return fmt.Errorf("failed to generate values file for upgrade: %v", err)
	}

	// For code-server apps, ensure VS Code settings ConfigMap exists
	if app.AppType == "code-server" {
		err = h.createVSCodeSettingsConfigMap(conn, releaseName, app.Namespace)
		if err != nil {
			log.Printf("Warning: Failed to create/update VS Code settings ConfigMap: %v", err)
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
		valuesFilePath,
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

	// Also delete the password if it's a code-server or ArgoCD application
	if app.AppType == "code-server" || app.AppType == "argocd" {
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

// getRealTimeStatus fetches the current Helm deployment status for an application
func (h *ApplicationsHandler) getRealTimeStatus(token, accountID string, app *models.Application) (string, error) {
	kvService := services.NewKVService()
	helmService := services.NewHelmService()

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}

	err := kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s", app.VPSID), &vpsConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	var privateKeyData map[string]string
	err = kvService.GetValue(token, accountID, "ssh_private_key", &privateKeyData)
	if err != nil {
		return "", fmt.Errorf("failed to get SSH private key: %v", err)
	}

	privateKey, ok := privateKeyData["private_key"]
	if !ok {
		return "", fmt.Errorf("SSH private key not found")
	}

	// Get real-time Helm status
	status, err := helmService.GetReleaseStatus(
		vpsConfig.PublicIPv4,
		vpsConfig.SSHUser,
		privateKey,
		app.Name,      // release name
		app.Namespace, // namespace
	)
	if err != nil {
		return "", fmt.Errorf("failed to get release status: %v", err)
	}

	return status, nil
}

// getArgoCDPassword retrieves the current password for an ArgoCD application directly from the VPS
func (h *ApplicationsHandler) getArgoCDPassword(token, accountID, appID string) (string, error) {
	kvService := services.NewKVService()

	// First try to get the stored password from KV
	var passwordData struct {
		Password string `json:"password"`
	}
	err := kvService.GetValue(token, accountID, fmt.Sprintf("app:%s:password", appID), &passwordData)
	if err == nil {
		// Decrypt password from KV storage
		password, decryptErr := utils.DecryptData(passwordData.Password, token)
		if decryptErr == nil {
			return password, nil
		}
		log.Printf("Warning: Failed to decrypt stored ArgoCD password, fetching from VPS: %v", decryptErr)
	}

	// If not in KV or decryption failed, fetch directly from VPS
	log.Printf("ArgoCD password not found in KV, fetching from VPS for app %s", appID)

	// Get application details
	var app models.Application
	err = kvService.GetValue(token, accountID, fmt.Sprintf("app:%s", appID), &app)
	if err != nil {
		return "", fmt.Errorf("failed to get application details: %v", err)
	}

	// Get VPS configuration for SSH details
	var vpsConfig struct {
		PublicIPv4 string `json:"public_ipv4"`
		SSHUser    string `json:"ssh_user"`
	}
	err = kvService.GetValue(token, accountID, fmt.Sprintf("vps:%s:config", app.VPSID), &vpsConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get VPS configuration: %v", err)
	}

	// Get SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		return "", fmt.Errorf("failed to get SSH private key: %v", err)
	}

	// Connect to VPS
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(app.VPSID)
	conn, err := sshService.GetOrCreateConnection(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, vpsIDInt)
	if err != nil {
		return "", fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Retrieve admin password from ArgoCD initial admin secret
	secretName := "argocd-initial-admin-secret"
	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", app.Namespace, secretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve password from secret '%s' in namespace '%s': %v", secretName, app.Namespace, err)
	}

	password := strings.TrimSpace(result.Output)
	if password == "" {
		return "", fmt.Errorf("no password found in ArgoCD secret '%s'", secretName)
	}

	// Store the retrieved password in KV for future use
	encryptedPassword, err := utils.EncryptData(password, token)
	if err != nil {
		log.Printf("Warning: Failed to encrypt password for storage: %v", err)
		// Still return the password even if we can't store it
		return password, nil
	}

	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:password", appID), map[string]string{
		"password": encryptedPassword,
	})
	if err != nil {
		log.Printf("Warning: Failed to store ArgoCD password in KV: %v", err)
		// Still return the password even if we can't store it
	}

	return password, nil
}

// updateArgoCDPassword updates the password for an ArgoCD application
func (h *ApplicationsHandler) updateArgoCDPassword(token, accountID, appID, newPassword string, app *models.Application) error {
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

	// Update the ArgoCD admin password using Helm values
	// Generate bcrypt hash of the new password using htpasswd
	hashCmd := fmt.Sprintf("htpasswd -nbBC 10 \"\" %s | tr -d ':\\n' | sed 's/$2y/$2a/'", newPassword)
	hashResult, err := sshService.ExecuteCommand(conn, hashCmd)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %v", err)
	}
	hashedPassword := strings.TrimSpace(hashResult.Output)

	// Get the Helm release name from the application namespace (it should match the app ID pattern)
	releaseName := fmt.Sprintf("%s-app-%s", app.Subdomain, strings.Split(app.ID, "-")[1])

	// Update ArgoCD using Helm values
	upgradeCmd := fmt.Sprintf("helm upgrade %s oci://ghcr.io/argoproj/argo-helm/argo-cd --version 8.1.2 --namespace %s --set configs.secret.argocdServerAdminPassword=%s --reuse-values",
		releaseName, app.Namespace, hashedPassword)
	_, err = sshService.ExecuteCommand(conn, upgradeCmd)
	if err != nil {
		return fmt.Errorf("failed to update ArgoCD with new password: %v", err)
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

// retrieveAndStoreArgoCDPassword retrieves the auto-generated admin password from ArgoCD
func (h *ApplicationsHandler) retrieveAndStoreArgoCDPassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// List all secrets in the namespace for debugging
	listCmd := fmt.Sprintf("kubectl get secrets --namespace %s --no-headers", namespace)
	listResult, err := sshService.ExecuteCommand(conn, listCmd)
	if err != nil {
		log.Printf("Debug: Failed to list secrets in namespace %s: %v", namespace, err)
	} else {
		log.Printf("Debug: Available secrets in namespace %s:\n%s", namespace, listResult.Output)
	}

	// Try to find ArgoCD admin secret with different possible names
	secretNames := []string{
		"argocd-initial-admin-secret",
		fmt.Sprintf("%s-argocd-initial-admin-secret", releaseName),
		"argocd-secret",
		fmt.Sprintf("%s-argocd-secret", releaseName),
	}

	var password string
	var foundSecret string

	for _, secretName := range secretNames {
		cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", namespace, secretName)
		result, err := sshService.ExecuteCommand(conn, cmd)
		if err == nil && strings.TrimSpace(result.Output) != "" {
			password = strings.TrimSpace(result.Output)
			foundSecret = secretName
			log.Printf("Found ArgoCD admin password in secret: %s", secretName)
			break
		}
	}

	// If no password found in any secret, try to get it from ArgoCD server pod logs or generate one
	if password == "" {
		log.Printf("Warning: No ArgoCD admin password found in standard secrets, checking server logs...")

		// Try to get the initial password from ArgoCD server logs
		logCmd := fmt.Sprintf("kubectl logs --namespace %s -l app.kubernetes.io/name=argocd-server --tail=100 2>/dev/null | grep -i 'password' | head -5", namespace)
		logResult, err := sshService.ExecuteCommand(conn, logCmd)
		if err == nil && strings.TrimSpace(logResult.Output) != "" {
			log.Printf("ArgoCD server logs (password related):\n%s", logResult.Output)
		}

		// As a last resort, generate a secure password and set it
		password = "admin" + fmt.Sprintf("%d", time.Now().Unix())
		log.Printf("Warning: No ArgoCD admin password found, using generated password")

		// Try to create the initial admin secret with our generated password
		encodedPassword := utils.Base64Encode(password)
		createSecretCmd := fmt.Sprintf(`kubectl create secret generic argocd-initial-admin-secret --namespace %s --from-literal=password=%s --dry-run=client -o yaml | kubectl apply -f -`, namespace, encodedPassword)
		_, err = sshService.ExecuteCommand(conn, createSecretCmd)
		if err != nil {
			log.Printf("Warning: Failed to create ArgoCD admin secret: %v", err)
		} else {
			log.Printf("Created ArgoCD admin secret with generated password")
		}
	}

	// Store password in KV storage (encrypted)
	kvService := services.NewKVService()
	encryptedPassword, err := utils.EncryptData(password, token)
	if err != nil {
		log.Printf("Warning: Failed to encrypt password: %v", err)
		return nil // Don't fail the deployment for this
	}

	err = kvService.PutValue(token, accountID, fmt.Sprintf("app:%s:password", appID), map[string]string{
		"password": encryptedPassword,
	})
	if err != nil {
		log.Printf("Warning: Failed to store ArgoCD password in KV: %v", err)
		return nil // Don't fail the deployment for this
	}

	if foundSecret != "" {
		log.Printf("Successfully stored ArgoCD admin password from secret: %s", foundSecret)
	} else {
		log.Printf("Successfully stored generated ArgoCD admin password")
	}

	return nil
}
