package vps

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// VPSInfoHandler handles VPS information retrieval and monitoring
type VPSInfoHandler struct {
	*BaseHandler
	vpsService *services.VPSService
}

// NewVPSInfoHandler creates a new VPS info handler instance
func NewVPSInfoHandler() *VPSInfoHandler {
	return &VPSInfoHandler{
		BaseHandler: NewBaseHandler(),
		vpsService:  services.NewVPSService(),
	}
}

// HandleVPSList returns a JSON list of VPS instances
func (h *VPSInfoHandler) HandleVPSList(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Get servers from KV store (includes both Hetzner and Oracle VPS)
	servers, err := h.vpsService.GetServersFromKV(token, accountID)
	if err != nil {
		log.Printf("Error getting servers from KV: %v", err)
		utils.JSONInternalServerError(c, "Failed to list servers")
		return
	}

	// Enhance servers with cost information using VPS service
	if err := h.vpsService.EnhanceServersWithCosts(token, accountID, servers); err != nil {
		log.Printf("Warning: Failed to enhance servers with costs: %v", err)
		// Continue without costs rather than failing
	}

	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

// HandleVPSStatus gets VPS health status via SSH
func (h *VPSInfoHandler) HandleVPSStatus(c *gin.Context) {
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

	// Parse server ID for logging
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Check VPS health via SSH
	status, err := h.sshService.CheckVPSHealth(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey, serverID)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to check VPS status: %v", err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, status)
}

// HandleVPSLogs fetches VPS logs via SSH connection
func (h *VPSInfoHandler) HandleVPSLogs(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)

	// Get number of lines (default 100)
	lines := 100
	if linesStr := c.Query("lines"); linesStr != "" {
		if parsedLines, err := fmt.Sscanf(linesStr, "%d", &lines); err == nil && parsedLines > 0 {
			if lines > 1000 {
				lines = 1000 // Limit to prevent overwhelming response
			}
		}
	}

	// Get SSH private key
	privateKey, valid := h.getSSHPrivateKey(c, token, accountID)
	if !valid {
		return
	}

	// Parse server ID for response
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Connect to VPS and get logs
	logs, err := h.sshService.GetVPSLogs(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey, lines)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to fetch logs: %v", err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"logs":      logs,
		"server_id": serverID,
		"lines":     lines,
	})
}

// HandleK3sLogs fetches K3s service logs via SSH connection
func (h *VPSInfoHandler) HandleK3sLogs(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)

	// Get number of lines (default 100)
	lines := 100
	if linesStr := c.Query("lines"); linesStr != "" {
		if parsedLines, err := fmt.Sscanf(linesStr, "%d", &lines); err == nil && parsedLines > 0 {
			if lines > 1000 {
				lines = 1000 // Limit to prevent overwhelming response
			}
		}
	}

	// Get SSH private key
	privateKey, valid := h.getSSHPrivateKey(c, token, accountID)
	if !valid {
		return
	}

	// Parse server ID for response
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Connect to VPS and get K3s logs
	logs, err := h.sshService.GetVPSK3sLogs(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey, lines)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to fetch K3s logs: %v", err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"logs":      logs,
		"server_id": serverID,
		"lines":     lines,
	})
}

// HandleVPSInfo retrieves VPS information including ArgoCD credentials
func (h *VPSInfoHandler) HandleVPSInfo(c *gin.Context) {
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

	// Parse server ID for response
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Connect to VPS and get info file
	conn, err := h.sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("SSH connection failed: %v", err))
		return
	}
	defer conn.Close()

	// Get VPS info including ArgoCD credentials
	info, err := h.sshService.GetVPSInfo(conn)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to get VPS info: %v", err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"server_id": serverID,
		"info":      info,
		"config":    vpsConfig,
	})
}

// HandleVPSSSHKey returns SSH private key for VPS access
func (h *VPSInfoHandler) HandleVPSSSHKey(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Get CSR configuration which contains the SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		CSR        string `json:"csr"`
		PrivateKey string `json:"private_key"`
		CreatedAt  string `json:"created_at"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Error getting CSR config: %v", err)
		utils.JSONSSHKeyNotFound(c)
		return
	}

	// Check if user wants to download the key
	download := c.Query("download")
	if download == "true" {
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", "attachment; filename=xanthus-key.pem")
		c.String(http.StatusOK, csrConfig.PrivateKey)
		return
	}

	// Return SSH private key and usage instructions
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"private_key": csrConfig.PrivateKey,
		"instructions": map[string]interface{}{
			"save_to_file":    "Save the private key to a file (e.g., ~/.ssh/xanthus-key.pem)",
			"set_permissions": "chmod 600 ~/.ssh/xanthus-key.pem",
			"ssh_command":     "ssh -i ~/.ssh/xanthus-key.pem root@<server-ip>",
		},
	})
}

// HandleVPSTerminal creates a web terminal session for VPS
func (h *VPSInfoHandler) HandleVPSTerminal(c *gin.Context) {
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

	// Parse server ID for terminal session
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Update VPS SSH user configuration if needed (for migration/correction)
	if err := h.vpsService.UpdateVPSSSHUser(token, accountID, serverID); err != nil {
		log.Printf("Warning: Failed to update VPS SSH user configuration: %v", err)
	}

	// Resolve SSH user using provider resolver
	resolvedSSHUser, err := h.vpsService.ResolveSSHUser(token, accountID, serverID)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to resolve SSH user: %v", err))
		return
	}

	log.Printf("🔍 VPS Terminal Debug - ServerID: %d, Provider: %s, StoredUser: %s, ResolvedUser: %s",
		serverID, vpsConfig.Provider, vpsConfig.SSHUser, resolvedSSHUser)

	// Create terminal session
	terminalService := services.NewTerminalService()
	session, err := terminalService.CreateSession(serverID, vpsConfig.PublicIPv4, resolvedSSHUser, privateKey)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to create terminal session: %v", err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"success":    true,
		"session_id": session.ID,
		"url":        fmt.Sprintf("/terminal/%s", session.ID),
		"port":       session.Port,
	})
}

// HandleVPSSSHUserDebug provides debug information about SSH user resolution for a VPS
func (h *VPSInfoHandler) HandleVPSSSHUserDebug(c *gin.Context) {
	serverIDStr := c.Param("id")
	vpsConfig, valid := h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	token, accountID, _ := h.validateTokenAndAccount(c)
	serverID, _ := utils.ParseServerID(serverIDStr)

	// Get current configuration
	storedSSHUser := vpsConfig.SSHUser

	// Get provider defaults
	providerDefaults := h.vpsService.GetProviderDefaults(vpsConfig.Provider)

	// Get resolved SSH user
	resolvedSSHUser, err := h.vpsService.ResolveSSHUser(token, accountID, serverID)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to resolve SSH user: %v", err))
		return
	}

	// Get correct SSH user (ignoring stored value)
	correctSSHUser := h.vpsService.GetCorrectSSHUserFromProvider(vpsConfig.Provider)

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"server_id":                 serverID,
		"provider":                  vpsConfig.Provider,
		"stored_ssh_user":           storedSSHUser,
		"provider_default_ssh_user": providerDefaults.DefaultSSHUser,
		"resolved_ssh_user":         resolvedSSHUser,
		"correct_ssh_user":          correctSSHUser,
		"needs_update":              storedSSHUser != correctSSHUser,
		"debug_info": gin.H{
			"provider_supports_api": providerDefaults.SupportsAPICreation,
			"provider_default_port": providerDefaults.DefaultSSHPort,
		},
	})
}

// HandleVPSApplications fetches all applications deployed on a specific VPS
func (h *VPSInfoHandler) HandleVPSApplications(c *gin.Context) {
	serverIDStr := c.Param("id")
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Parse server ID
	serverID, err := utils.ParseServerID(serverIDStr)
	if err != nil {
		utils.JSONBadRequest(c, "Invalid server ID")
		return
	}

	// Validate VPS access
	_, valid = h.getVPSConfig(c, serverIDStr)
	if !valid {
		return
	}

	// Use application service to get applications for this VPS
	appService := services.NewSimpleApplicationService()

	// Get all applications for this account
	allApplications, err := appService.ListApplications(token, accountID)
	if err != nil {
		log.Printf("Error listing applications for VPS %d: %v", serverID, err)
		utils.JSONInternalServerError(c, "Failed to fetch applications")
		return
	}

	// Filter applications that belong to this VPS
	var vpsApplications []models.Application
	vpsIDStr := fmt.Sprintf("%d", serverID)

	for _, app := range allApplications {
		if app.VPSID == vpsIDStr {
			vpsApplications = append(vpsApplications, app)
		}
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"server_id":    serverID,
		"applications": vpsApplications,
		"count":        len(vpsApplications),
	})
}
