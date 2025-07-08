package vps

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// VPSConfigHandler handles VPS configuration and deployment operations
type VPSConfigHandler struct {
	*BaseHandler
	vpsService *services.VPSService
}

// NewVPSConfigHandler creates a new VPS config handler instance
func NewVPSConfigHandler() *VPSConfigHandler {
	return &VPSConfigHandler{
		BaseHandler: NewBaseHandler(),
		vpsService:  services.NewVPSService(),
	}
}

// HandleVPSConfigure configures VPS with SSL certificates for a specific domain
func (h *VPSConfigHandler) HandleVPSConfigure(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)

	domain := c.PostForm("domain")
	if domain == "" {
		utils.JSONBadRequest(c, "Domain is required for SSL configuration")
		return
	}

	// Get SSL configuration for the domain
	domainConfig, err := h.kvService.GetDomainSSLConfig(token, accountID, domain)
	if err != nil {
		utils.JSONNotFound(c, "SSL configuration not found for domain")
		return
	}

	// Get SSH private key
	privateKey, valid := h.getSSHPrivateKey(c, token, accountID)
	if !valid {
		return
	}

	// Parse server ID for logging
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Connect to VPS and configure SSL
	conn, err := h.sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer conn.Close()

	// Configure K3s with new SSL certificates
	if err := h.sshService.ConfigureK3s(conn, domainConfig.Certificate, domainConfig.PrivateKey); err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to configure K3s: %v", err))
		return
	}

	log.Printf("✅ Successfully configured VPS %d with SSL for domain %s", serverID, domain)
	utils.VPSConfigurationSuccess(c, domain)
}

// HandleVPSDeploy deploys Kubernetes manifests to a VPS
func (h *VPSConfigHandler) HandleVPSDeploy(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)

	manifest := c.PostForm("manifest")
	name := c.PostForm("name")
	if manifest == "" || name == "" {
		utils.JSONBadRequest(c, "Manifest and name are required")
		return
	}

	// Get SSH private key
	privateKey, valid := h.getSSHPrivateKey(c, token, accountID)
	if !valid {
		return
	}

	// Parse server ID for logging
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Connect to VPS and deploy manifest
	conn, err := h.sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer conn.Close()

	// Deploy the Kubernetes manifest
	if err := h.sshService.DeployManifest(conn, manifest, name); err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to deploy manifest: %v", err))
		return
	}

	log.Printf("✅ Successfully deployed %s to VPS %d", name, serverID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Successfully deployed %s to VPS", name),
	})
}

// HandleVPSCheckKey checks if Hetzner API key exists in KV storage
func (h *VPSConfigHandler) HandleVPSCheckKey(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Check if Hetzner API key exists
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil || hetznerKey == "" {
		utils.JSONResponse(c, http.StatusOK, gin.H{"exists": false})
		return
	}

	// Mask the key for security (show only first 4 and last 4 characters)
	maskedKey := ""
	if len(hetznerKey) > 8 {
		maskedKey = hetznerKey[:4] + "..." + hetznerKey[len(hetznerKey)-4:]
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"exists":     true,
		"masked_key": maskedKey,
	})
}

// HandleVPSValidateKey validates and stores Hetzner API key
func (h *VPSConfigHandler) HandleVPSValidateKey(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	apiKey := c.PostForm("key")
	if apiKey == "" {
		utils.JSONBadRequest(c, "API key is required")
		return
	}

	// Validate the key
	if !utils.ValidateHetznerAPIKey(apiKey) {
		utils.JSONBadRequest(c, "Invalid Hetzner API key")
		return
	}

	// Store the key
	log.Printf("HandleVPSValidateKey: Storing API key for account %s", accountID)
	client := &http.Client{Timeout: 10 * time.Second}
	encryptedKey, err := utils.EncryptData(apiKey, token)
	if err != nil {
		log.Printf("HandleVPSValidateKey: Encryption failed for account %s: %v", accountID, err)
		utils.JSONInternalServerError(c, "Failed to encrypt API key")
		return
	}

	if err := utils.PutKVValue(client, token, accountID, "config:hetzner:api_key", encryptedKey); err != nil {
		log.Printf("HandleVPSValidateKey: KV storage failed for account %s: %v", accountID, err)
		utils.JSONInternalServerError(c, "Failed to store API key")
		return
	}

	log.Printf("HandleVPSValidateKey: Successfully stored API key for account %s", accountID)

	// Store the key temporarily in memory cache for immediate use (e.g., name validation)
	utils.SetTempHetznerKey(accountID, apiKey)
	log.Printf("HandleVPSValidateKey: Stored temporary key in cache for account %s", accountID)

	utils.JSONResponse(c, http.StatusOK, gin.H{"success": true})
}

// HandleSetupHetzner configures Hetzner API key in setup
func (h *VPSConfigHandler) HandleSetupHetzner(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccountHTML(c)
	if !valid {
		return
	}

	hetznerKey := c.PostForm("hetzner_key")

	// If no key provided, check if there's an existing key
	if hetznerKey == "" {
		if existingKey, err := utils.GetHetznerAPIKey(token, accountID); err == nil && existingKey != "" {
			// Use existing key - proceed to next step
			log.Println("✅ Using existing Hetzner API key")
			c.Redirect(http.StatusTemporaryRedirect, "/setup")
			return
		} else {
			c.Data(http.StatusBadRequest, "text/html", []byte("❌ Hetzner API key is required"))
			return
		}
	}

	// Validate Hetzner API key
	if !utils.ValidateHetznerAPIKey(hetznerKey) {
		c.Data(http.StatusOK, "text/html", []byte("❌ Invalid Hetzner API key. Please check your key and try again."))
		return
	}

	// Store encrypted Hetzner API key in KV
	client := &http.Client{Timeout: 10 * time.Second}
	encryptedKey, err := utils.EncryptData(hetznerKey, token)
	if err != nil {
		log.Printf("Error encrypting Hetzner key: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error storing API key"))
		return
	}

	if err := utils.PutKVValue(client, token, accountID, "config:hetzner:api_key", encryptedKey); err != nil {
		log.Printf("Error storing Hetzner key: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error storing API key"))
		return
	}

	log.Println("✅ Hetzner API key stored successfully")
	c.Redirect(http.StatusTemporaryRedirect, "/setup")
}

// HandleVPSGetTimezone retrieves the current timezone for a VPS
func (h *VPSConfigHandler) HandleVPSGetTimezone(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)

	// Get SSH private key
	privateKey, valid := h.getSSHPrivateKey(c, token, accountID)
	if !valid {
		return
	}

	// Connect to VPS and get timezone
	conn, err := h.sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer conn.Close()

	// Get current timezone from VPS - try sudo first, then fall back
	result, err := h.sshService.ExecuteCommand(conn, "sudo timedatectl show --property=Timezone --value")
	if err != nil {
		log.Printf("Sudo get timezone failed: %v, trying direct command", err)
		// Try without sudo
		result, err = h.sshService.ExecuteCommand(conn, "timedatectl show --property=Timezone --value")
		if err != nil {
			utils.JSONInternalServerError(c, fmt.Sprintf("Failed to get timezone: %v", err))
			return
		}
	}

	currentTimezone := strings.TrimSpace(result.Output)

	// Also return the timezone from config if available
	configTimezone := vpsConfig.Timezone
	if configTimezone == "" {
		configTimezone = currentTimezone
	}

	c.JSON(http.StatusOK, gin.H{
		"current_timezone": currentTimezone,
		"config_timezone":  configTimezone,
		"success":          true,
	})
}

// HandleVPSSetTimezone sets the timezone for a VPS
func (h *VPSConfigHandler) HandleVPSSetTimezone(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)

	timezone := c.PostForm("timezone")
	if timezone == "" {
		utils.JSONBadRequest(c, "Timezone is required")
		return
	}

	// Validate timezone format
	if !h.isValidTimezone(timezone) {
		utils.JSONBadRequest(c, "Invalid timezone format")
		return
	}

	// Get SSH private key
	privateKey, valid := h.getSSHPrivateKey(c, token, accountID)
	if !valid {
		return
	}

	// Parse server ID for logging
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Connect to VPS and set timezone
	conn, err := h.sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer conn.Close()

	// Set timezone on VPS - try sudo first, then fall back to direct command
	cmd := fmt.Sprintf("sudo timedatectl set-timezone %s", timezone)
	result, err := h.sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		log.Printf("Sudo command failed: %v, trying direct command", err)
		// Try without sudo (for root user or if sudo is not available)
		cmd = fmt.Sprintf("timedatectl set-timezone %s", timezone)
		if _, err := h.sshService.ExecuteCommand(conn, cmd); err != nil {
			utils.JSONInternalServerError(c, fmt.Sprintf("Failed to set timezone: %v", err))
			return
		}
	} else {
		log.Printf("Timezone set successfully with sudo: %s", result.Output)
	}

	// Verify the change - try sudo first, then fall back
	result, err = h.sshService.ExecuteCommand(conn, "sudo timedatectl show --property=Timezone --value")
	if err != nil {
		log.Printf("Sudo verify failed: %v, trying direct command", err)
		// Try without sudo
		result, err = h.sshService.ExecuteCommand(conn, "timedatectl show --property=Timezone --value")
		if err != nil {
			utils.JSONInternalServerError(c, fmt.Sprintf("Failed to verify timezone: %v", err))
			return
		}
	}

	actualTimezone := strings.TrimSpace(result.Output)
	if actualTimezone != timezone {
		utils.JSONInternalServerError(c, fmt.Sprintf("Timezone verification failed: expected %s, got %s", timezone, actualTimezone))
		return
	}

	// Update VPS config with new timezone
	vpsConfig.Timezone = timezone

	// For Oracle Cloud VPS with generic "oracle-cloud" location, update to actual region
	if strings.Contains(vpsConfig.Provider, "Oracle") && vpsConfig.Location == "oracle-cloud" {
		// Default to Frankfurt region for European users
		vpsConfig.Location = "eu-frankfurt-1"
		log.Printf("🔧 Updated Oracle Cloud VPS location from 'oracle-cloud' to 'eu-frankfurt-1'")
	}

	if err := h.kvService.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		log.Printf("Warning: Failed to store timezone in VPS config: %v", err)
	}

	log.Printf("✅ Successfully set timezone %s for VPS %d", timezone, serverID)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  fmt.Sprintf("Successfully set timezone to %s", timezone),
		"timezone": actualTimezone,
	})
}

// HandleVPSListTimezones returns a list of available timezones
func (h *VPSConfigHandler) HandleVPSListTimezones(c *gin.Context) {
	timezones := []string{
		"UTC",
		"America/New_York",    // US Eastern
		"America/Chicago",     // US Central
		"America/Denver",      // US Mountain
		"America/Los_Angeles", // US Pacific
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Europe/Rome",
		"Europe/Madrid",
		"Europe/Athens",
		"Europe/Amsterdam",
		"Europe/Brussels",
		"Europe/Vienna",
		"Europe/Prague",
		"Europe/Warsaw",
		"Europe/Budapest",
		"Europe/Helsinki",
		"Europe/Stockholm",
		"Europe/Oslo",
		"Europe/Copenhagen",
		"Europe/Zurich",
		"Asia/Tokyo",
		"Asia/Shanghai",
		"Asia/Hong_Kong",
		"Asia/Singapore",
		"Asia/Mumbai",
		"Asia/Dubai",
		"Asia/Seoul",
		"Asia/Bangkok",
		"Asia/Jakarta",
		"Australia/Sydney",
		"Australia/Melbourne",
		"Australia/Perth",
		"America/New_York",
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"America/Toronto",
		"America/Vancouver",
		"America/Mexico_City",
		"America/Sao_Paulo",
		"America/Buenos_Aires",
		"Africa/Cairo",
		"Africa/Johannesburg",
		"Africa/Lagos",
	}

	c.JSON(http.StatusOK, gin.H{
		"timezones": timezones,
		"success":   true,
	})
}

// isValidTimezone validates if a timezone string is valid
func (h *VPSConfigHandler) isValidTimezone(timezone string) bool {
	// Basic validation - could be enhanced with more comprehensive checks
	if timezone == "" {
		return false
	}

	// Check for common timezone patterns (IANA timezone database format)
	validPatterns := []string{
		"UTC",
		"GMT",
		"Europe/",
		"Asia/",
		"Australia/",
		"America/",
		"Africa/",
		"Pacific/",
		"Indian/",
		"Atlantic/",
		"Arctic/",
		"Antarctica/",
	}

	for _, pattern := range validPatterns {
		if timezone == pattern || strings.HasPrefix(timezone, pattern) {
			return true
		}
	}

	return false
}

// Setup API endpoints for Svelte SPA

// HandleSetupStatusAPI returns the current setup status
func (h *VPSConfigHandler) HandleSetupStatusAPI(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	status := gin.H{
		"hetzner_configured": false,
		"cloudflare_configured": true, // Always true if user is authenticated
		"setup_complete": false,
	}

	// Check if Hetzner key is configured
	if existingKey, err := utils.GetHetznerAPIKey(token, accountID); err == nil && existingKey != "" {
		status["hetzner_configured"] = true
		status["setup_complete"] = true
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"success": true,
	})
}

// HandleSetupHetznerAPI configures Hetzner API key via JSON API
func (h *VPSConfigHandler) HandleSetupHetznerAPI(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Parse JSON request body
	var req struct {
		HetznerKey string `json:"hetzner_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "Invalid request format")
		return
	}

	hetznerKey := req.HetznerKey

	// If no key provided, check if there's an existing key
	if hetznerKey == "" {
		if existingKey, err := utils.GetHetznerAPIKey(token, accountID); err == nil && existingKey != "" {
			// Use existing key - setup is complete
			log.Println("✅ Using existing Hetzner API key")
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Using existing Hetzner API key",
				"action": "existing_key",
			})
			return
		} else {
			utils.JSONBadRequest(c, "Hetzner API key is required")
			return
		}
	}

	// Validate Hetzner API key
	if !utils.ValidateHetznerAPIKey(hetznerKey) {
		utils.JSONBadRequest(c, "Invalid Hetzner API key. Please check your key and try again.")
		return
	}

	// Store encrypted Hetzner API key in KV
	client := &http.Client{Timeout: 10 * time.Second}
	encryptedKey, err := utils.EncryptData(hetznerKey, token)
	if err != nil {
		log.Printf("Error encrypting Hetzner key: %v", err)
		utils.JSONInternalServerError(c, "Error storing API key")
		return
	}

	if err := utils.PutKVValue(client, token, accountID, "config:hetzner:api_key", encryptedKey); err != nil {
		log.Printf("Error storing Hetzner key: %v", err)
		utils.JSONInternalServerError(c, "Error storing API key")
		return
	}

	log.Println("✅ Hetzner API key stored successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hetzner API key configured successfully",
		"action": "key_configured",
	})
}
