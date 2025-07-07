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

// BaseHandler contains shared dependencies and methods for all VPS handlers
type BaseHandler struct {
	hetznerService *services.HetznerService
	kvService      *services.KVService
	sshService     *services.SSHService
	cfService      *services.CloudflareService
}

// NewBaseHandler creates a new base handler instance with initialized services
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{
		hetznerService: services.NewHetznerService(),
		kvService:      services.NewKVService(),
		sshService:     services.NewSSHService(),
		cfService:      services.NewCloudflareService(),
	}
}

// validateTokenAndAccount validates the token and returns account info
// Returns true if valid, false if invalid (and sends appropriate error response)
// This method auto-detects JWT vs cookie-based authentication
func (h *BaseHandler) validateTokenAndAccount(c *gin.Context) (token, accountID string, valid bool) {
	// Check if we have JWT context (from JWT middleware)
	if _, hasJWT := c.Get("cf_token"); hasJWT {
		return utils.ValidateJWTAndGetAccountJSON(c)
	}
	// Fallback to cookie-based authentication
	return utils.ValidateTokenAndGetAccountJSON(c)
}

// validateTokenAndAccountHTML validates token for HTML pages
// Returns true if valid, false if invalid (and redirects to login)
func (h *BaseHandler) validateTokenAndAccountHTML(c *gin.Context) (token, accountID string, valid bool) {
	return utils.ValidateTokenAndGetAccountHTML(c)
}

// getVPSConfig retrieves VPS configuration for a given server ID
// Handles authentication, server ID parsing, and error responses
func (h *BaseHandler) getVPSConfig(c *gin.Context, serverIDStr string) (*services.VPSConfig, bool) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return nil, false
	}

	serverID, err := utils.ParseServerID(serverIDStr)
	if err != nil {
		utils.JSONServerIDInvalid(c)
		return nil, false
	}

	vpsConfig, err := h.kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		utils.JSONVPSNotFound(c)
		return nil, false
	}

	return vpsConfig, true
}

// getHetznerKey retrieves and validates Hetzner API key
func (h *BaseHandler) getHetznerKey(c *gin.Context, token, accountID string) (string, bool) {
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		log.Printf("getHetznerKey failed for account %s: %v", accountID, err)
		utils.JSONHetznerKeyMissing(c)
		return "", false
	}
	return hetznerKey, true
}

// getSSHPrivateKey retrieves SSH private key from CSR config
func (h *BaseHandler) getSSHPrivateKey(c *gin.Context, token, accountID string) (string, bool) {
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}

	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		utils.JSONSSHKeyNotFound(c)
		return "", false
	}

	return csrConfig.PrivateKey, true
}

// getServerIDFromRequest gets server_id from either form data or JSON
func (h *BaseHandler) getServerIDFromRequest(c *gin.Context) (string, error) {
	// Try form data first (for backward compatibility)
	if serverIDStr := c.PostForm("server_id"); serverIDStr != "" {
		return serverIDStr, nil
	}
	
	// Try JSON body
	var reqBody struct {
		ServerID string `json:"server_id"`
	}
	if err := c.ShouldBindJSON(&reqBody); err == nil && reqBody.ServerID != "" {
		return reqBody.ServerID, nil
	}
	
	return "", fmt.Errorf("server_id is required")
}

// performServerAction is a generic helper for server power management actions
func (h *BaseHandler) performServerAction(c *gin.Context, action string) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	serverIDStr, err := h.getServerIDFromRequest(c)
	if err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	
	serverID, err := utils.ParseServerID(serverIDStr)
	if err != nil {
		utils.JSONServerIDInvalid(c)
		return
	}

	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		return
	}

	var actionErr error
	switch action {
	case "poweroff":
		actionErr = h.hetznerService.PowerOffServer(hetznerKey, serverID)
	case "poweron":
		actionErr = h.hetznerService.PowerOnServer(hetznerKey, serverID)
	case "reboot":
		actionErr = h.hetznerService.RebootServer(hetznerKey, serverID)
	default:
		utils.JSONBadRequest(c, "Invalid action")
		return
	}

	if actionErr != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to perform %s: %v", action, actionErr))
		return
	}

	// Convert action to past tense for response
	actionText := action
	switch action {
	case "poweroff":
		actionText = "powered off"
	case "poweron":
		actionText = "powered on"
	case "reboot":
		actionText = "rebooted"
	}

	utils.JSONVPSPowerActionSuccess(c, actionText, serverIDStr)
}
