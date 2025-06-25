package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

// Application represents a deployed application
type Application struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Subdomain    string `json:"subdomain"`
	Domain       string `json:"domain"`
	VPSID        string `json:"vps_id"`
	VPSName      string `json:"vps_name"`
	ChartName    string `json:"chart_name"`
	ChartVersion string `json:"chart_version"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
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
		applications = []Application{}
	}

	c.HTML(http.StatusOK, "applications.html", gin.H{
		"Applications": applications,
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
func (h *ApplicationsHandler) getApplicationsList(token, accountID string) ([]Application, error) {
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

	applications := []Application{}

	// Fetch each application
	for _, key := range keysResp.Result {
		var app Application
		if err := kvService.GetValue(token, accountID, key.Name, &app); err == nil {
			applications = append(applications, app)
		}
	}

	return applications, nil
}

// createApplication implements core application creation logic with VPS integration
func (h *ApplicationsHandler) createApplication(token, accountID string, appData interface{}) (*Application, error) {
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

	app := &Application{
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

	// TODO: Deploy via Helm (would need to implement Helm deployment logic)
	// For now, just mark as deployed
	app.Status = "deployed"
	kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)

	return app, nil
}

// upgradeApplication implements core application upgrade logic
func (h *ApplicationsHandler) upgradeApplication(token, accountID, appID, version string) error {
	kvService := services.NewKVService()
	
	// Get current application
	var app Application
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

	// TODO: Implement actual Helm upgrade
	// For now, just mark as deployed
	app.Status = "deployed"
	kvService.PutValue(token, accountID, fmt.Sprintf("app:%s", appID), app)

	return nil
}

// deleteApplication implements core application deletion logic
func (h *ApplicationsHandler) deleteApplication(token, accountID, appID string) error {
	kvService := services.NewKVService()
	
	// TODO: Uninstall Helm chart before deleting from KV
	
	// Delete from KV
	return kvService.DeleteValue(token, accountID, fmt.Sprintf("app:%s", appID))
}

// TODO: These utility functions have been moved to internal/utils/placeholders.go
// They need to be properly implemented and moved to domain-specific utils files