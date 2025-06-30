package vps

import (
	"fmt"
	"log"
	"net/http"
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
			c.Header("HX-Redirect", "/setup/server")
			c.Status(http.StatusOK)
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
	c.Header("HX-Redirect", "/setup/server")
	c.Status(http.StatusOK)
}
