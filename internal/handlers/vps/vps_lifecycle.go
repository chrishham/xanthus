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

// VPSLifecycleHandler handles VPS creation, deletion, and power management
type VPSLifecycleHandler struct {
	*BaseHandler
	vpsService *services.VPSService
}

// NewVPSLifecycleHandler creates a new VPS lifecycle handler instance
func NewVPSLifecycleHandler() *VPSLifecycleHandler {
	return &VPSLifecycleHandler{
		BaseHandler: NewBaseHandler(),
		vpsService:  services.NewVPSService(),
	}
}

// HandleVPSCreate creates a new VPS instance on Hetzner Cloud with K3s setup
func (h *VPSLifecycleHandler) HandleVPSCreate(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Validate required parameters
	name := c.PostForm("name")
	location := c.PostForm("location")
	serverType := c.PostForm("server_type")

	if name == "" {
		utils.JSONBadRequest(c, "Server name is required")
		return
	}
	if location == "" {
		utils.JSONBadRequest(c, "Server location is required")
		return
	}
	if serverType == "" {
		utils.JSONBadRequest(c, "Server type is required")
		return
	}

	// Get Hetzner API key
	log.Printf("VPS Create: Attempting to retrieve Hetzner API key for account %s", accountID)
	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		log.Printf("VPS Create: Failed to retrieve Hetzner API key for account %s", accountID)
		return
	}
	log.Printf("VPS Create: Successfully retrieved Hetzner API key for account %s", accountID)

	// Get SSL CSR configuration
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		CSR        string `json:"csr"`
		PrivateKey string `json:"private_key"`
		CreatedAt  string `json:"created_at"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Error getting CSR from KV: %v", err)
		utils.JSONInternalServerError(c, "SSL CSR configuration not found. Please logout and login again.")
		return
	}

	// Convert CSR private key to SSH public key
	sshPublicKey, err := h.cfService.ConvertPrivateKeyToSSH(csrConfig.PrivateKey)
	if err != nil {
		log.Printf("Error converting private key to SSH: %v", err)
		utils.JSONInternalServerError(c, "Failed to generate SSH key from CSR")
		return
	}

	// Validate SSH public key format
	if !strings.HasPrefix(sshPublicKey, "ssh-rsa ") {
		keyPreview := sshPublicKey
		if len(keyPreview) > 50 {
			keyPreview = keyPreview[:50] + "..."
		}
		log.Printf("Invalid SSH public key format: %s", keyPreview)
		utils.JSONInternalServerError(c, "Generated SSH key has invalid format")
		return
	}
	log.Printf("✅ Generated SSH public key (length: %d)", len(sshPublicKey))

	// Create or find SSH key in Hetzner Cloud
	sshKeyName := fmt.Sprintf("xanthus-key-%d", time.Now().Unix())
	sshKey, err := h.hetznerService.CreateOrFindSSHKey(hetznerKey, sshKeyName, sshPublicKey)
	if err != nil {
		log.Printf("Error creating/finding SSH key in Hetzner: %v", err)
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to create SSH key in Hetzner Cloud: %v", err))
		return
	}

	// Use the actual key name from the found/created key
	sshKeyName = sshKey.Name
	log.Printf("✅ Using SSH key: %s (ID: %d)", sshKeyName, sshKey.ID)

	// SSL certificates will be configured when applications are deployed
	var domainCert, domainKey string
	log.Printf("✅ VPS will be created without SSL certificates. SSL will be configured during application deployment")

	// Get server type pricing information
	serverTypes, err := utils.FetchHetznerServerTypes(hetznerKey)
	if err != nil {
		log.Printf("Warning: Could not fetch server types for pricing: %v", err)
	}

	var hourlyRate, monthlyRate float64
	for _, st := range serverTypes {
		if st.Name == serverType {
			if len(st.Prices) > 0 {
				// Use gross prices (including VAT)
				if hourlyGross := st.Prices[0].PriceHourly.Gross; hourlyGross != "" {
					if _, err := fmt.Sscanf(hourlyGross, "%f", &hourlyRate); err == nil {
						// Add IPv4 cost: €0.50/month = €0.00069444/hour (30.41 days avg per month)
						hourlyRate += 0.50 / (30.41 * 24)
					}
				}
				if monthlyGross := st.Prices[0].PriceMonthly.Gross; monthlyGross != "" {
					if _, err := fmt.Sscanf(monthlyGross, "%f", &monthlyRate); err == nil {
						// Add IPv4 cost
						monthlyRate += 0.50
					}
				}
			}
			break
		}
	}

	// Create VPS using the VPS service
	server, vpsConfig, err := h.vpsService.CreateVPSWithConfig(
		token, accountID, hetznerKey,
		name, serverType, location, "",
		sshKeyName, sshPublicKey,
		domainCert, domainKey,
		hourlyRate, monthlyRate,
	)
	if err != nil {
		log.Printf("Error creating server: %v", err)

		// Check for specific error types and provide user-friendly messages
		errorStr := err.Error()
		if strings.Contains(errorStr, "server name is already used") || strings.Contains(errorStr, "uniqueness_error") {
			utils.ClearTempHetznerKey(accountID) // Clean up on error
			c.JSON(http.StatusConflict, gin.H{"error": "A server with this name already exists. Please choose a different name."})
			return
		}

		// Generic error for other cases
		utils.ClearTempHetznerKey(accountID) // Clean up on error
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to create server: %v", err))
		return
	}

	// DNS records will be configured when applications are deployed
	log.Printf("✅ VPS created successfully. DNS records will be configured during application deployment")

	log.Printf("✅ Created server: %s (ID: %d) with IPv4: %s", server.Name, server.ID, server.PublicNet.IPv4.IP)

	// Clean up temporary Hetzner key cache after successful VPS creation
	utils.ClearTempHetznerKey(accountID)
	log.Printf("Cleaned up temporary Hetzner key cache for account %s", accountID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Server created successfully with K3s and Helm. DNS will be configured when applications are deployed",
		"server":  server,
		"config":  vpsConfig,
	})
}

// HandleVPSDelete deletes a VPS instance and cleans up configuration
func (h *VPSLifecycleHandler) HandleVPSDelete(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	serverIDStr := c.PostForm("server_id")
	serverID, err := utils.ParseServerID(serverIDStr)
	if err != nil {
		utils.JSONServerIDInvalid(c)
		return
	}

	// Get Hetzner API key
	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		return
	}

	// Delete VPS and cleanup using VPS service
	vpsConfig, err := h.vpsService.DeleteVPSAndCleanup(token, accountID, hetznerKey, serverID)
	if err != nil {
		log.Printf("Error deleting server %d: %v", serverID, err)
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to delete server: %v", err))
		return
	}

	serverName := fmt.Sprintf("Server %d", serverID)
	if vpsConfig != nil {
		serverName = vpsConfig.Name
	}

	log.Printf("✅ Deleted server: %s (ID: %d) and cleaned up configuration", serverName, serverID)
	utils.VPSDeletionSuccess(c)
}

// HandleSSHKey returns the SSH public key for manual VPS setup
func (h *VPSLifecycleHandler) HandleSSHKey(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Get SSL CSR configuration which contains the private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		CSR        string `json:"csr"`
		PrivateKey string `json:"private_key"`
		CreatedAt  string `json:"created_at"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Error getting CSR from KV: %v", err)
		utils.JSONInternalServerError(c, "SSL CSR configuration not found. Please logout and login again.")
		return
	}

	// Convert CSR private key to SSH public key
	sshPublicKey, err := h.cfService.ConvertPrivateKeyToSSH(csrConfig.PrivateKey)
	if err != nil {
		log.Printf("Error converting private key to SSH: %v", err)
		utils.JSONInternalServerError(c, "Failed to generate SSH key from CSR")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_key": sshPublicKey,
	})
}

// HandleAddOCI adds a manually created OCI instance to Xanthus management
func (h *VPSLifecycleHandler) HandleAddOCI(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Parse JSON request
	var req struct {
		Name     string `json:"name" binding:"required"`
		PublicIP string `json:"public_ip" binding:"required"`
		Username string `json:"username" binding:"required"`
		Shape    string `json:"shape" binding:"required"`
		Provider string `json:"provider" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "Invalid request data")
		return
	}

	// Validate provider
	if req.Provider != "oci" {
		utils.JSONBadRequest(c, "Only OCI provider is supported for manual addition")
		return
	}

	// Get SSH private key for connection
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		CSR        string `json:"csr"`
		PrivateKey string `json:"private_key"`
		CreatedAt  string `json:"created_at"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Error getting CSR from KV: %v", err)
		utils.JSONInternalServerError(c, "SSL CSR configuration not found. Please logout and login again.")
		return
	}

	// Convert to SSH public key for validation
	sshPublicKey, err := h.cfService.ConvertPrivateKeyToSSH(csrConfig.PrivateKey)
	if err != nil {
		log.Printf("Error converting private key to SSH: %v", err)
		utils.JSONInternalServerError(c, "Failed to generate SSH key from CSR")
		return
	}

	// Generate a mock server ID for OCI (using timestamp)
	serverID := int(time.Now().Unix())

	// Create VPS config for OCI instance
	vpsConfig, err := h.vpsService.CreateOCIVPSConfig(
		token, accountID,
		req.Name, req.PublicIP, req.Username, req.Shape,
		serverID, csrConfig.PrivateKey, sshPublicKey,
	)
	if err != nil {
		log.Printf("Error creating OCI VPS config: %v", err)
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to add OCI instance: %v", err))
		return
	}

	log.Printf("✅ Added OCI instance: %s (IP: %s) with ID: %d", req.Name, req.PublicIP, serverID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OCI instance added successfully. K3s setup is running in the background.",
		"server": gin.H{
			"id":   serverID,
			"name": req.Name,
			"public_net": gin.H{
				"ipv4": gin.H{
					"ip": req.PublicIP,
				},
			},
		},
		"config": vpsConfig,
	})
}

// HandleVPSPowerOff powers off a VPS instance
func (h *VPSLifecycleHandler) HandleVPSPowerOff(c *gin.Context) {
	h.performServerAction(c, "poweroff")
}

// HandleVPSPowerOn powers on a VPS instance
func (h *VPSLifecycleHandler) HandleVPSPowerOn(c *gin.Context) {
	h.performServerAction(c, "poweron")
}

// HandleVPSReboot reboots a VPS instance
func (h *VPSLifecycleHandler) HandleVPSReboot(c *gin.Context) {
	h.performServerAction(c, "reboot")
}
