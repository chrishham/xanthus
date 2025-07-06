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

	// Invalidate VPS cache to ensure immediate UI update
	h.vpsService.InvalidateVPSCache(accountID)

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

// HandleOCICreate creates a new OCI compute instance with K3s setup
func (h *VPSLifecycleHandler) HandleOCICreate(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Parse request data
	var req struct {
		Name     string  `json:"name" binding:"required"`
		Shape    string  `json:"shape" binding:"required"`
		Region   string  `json:"region"`
		Timezone string  `json:"timezone"`
		OCPU     float32 `json:"ocpu"`      // OCPU count for flexible shapes
		Memory   float32 `json:"memory"`    // Memory in GB for flexible shapes
		OCIToken string  `json:"oci_token"` // Optional - for first-time setup
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	// Set defaults for OCPU and Memory if not provided
	if req.OCPU == 0 {
		req.OCPU = 1.0 // Default to 1 OCPU
	}
	if req.Memory == 0 {
		req.Memory = 6.0 // Default to 6GB
	}

	// Validate Always Free tier limits for VM.Standard.A1.Flex
	if req.Shape == "VM.Standard.A1.Flex" {
		if req.OCPU > 4 {
			utils.JSONBadRequest(c, "OCPU count cannot exceed 4 for Always Free tier (VM.Standard.A1.Flex)")
			return
		}
		if req.Memory > 24 {
			utils.JSONBadRequest(c, "Memory cannot exceed 24GB for Always Free tier (VM.Standard.A1.Flex)")
			return
		}
		if req.OCPU < 1 {
			utils.JSONBadRequest(c, "OCPU count must be at least 1")
			return
		}
		if req.Memory < 1 {
			utils.JSONBadRequest(c, "Memory must be at least 1GB")
			return
		}
	}

	// Get or store OCI auth token
	var ociToken string
	var err error

	// First try to get existing token from KV store
	ociToken, err = utils.GetOCIAuthToken(token, accountID)
	if err != nil {
		// If no existing token and one is provided in request, store and use it
		if req.OCIToken != "" {
			log.Printf("OCI Create: No existing token found, storing provided token for account %s", accountID)
			if storeErr := utils.SetOCIAuthToken(token, accountID, req.OCIToken); storeErr != nil {
				log.Printf("OCI Create: Failed to store OCI auth token for account %s: %v", accountID, storeErr)
				utils.JSONInternalServerError(c, fmt.Sprintf("Failed to store OCI auth token: %v", storeErr))
				return
			}
			ociToken = req.OCIToken
			log.Printf("✅ Stored OCI auth token for account %s for future reuse", accountID)
		} else {
			log.Printf("OCI Create: Failed to retrieve OCI auth token for account %s: %v", accountID, err)
			utils.JSONInternalServerError(c, "OCI auth token not found. Please provide oci_token in the request or configure your OCI credentials first.")
			return
		}
	} else {
		log.Printf("✅ Reusing existing OCI auth token for account %s", accountID)
	}

	// Create OCI service
	ociService, err := services.NewOCIService(ociToken)
	if err != nil {
		log.Printf("OCI Create: Failed to create OCI service: %v", err)
		utils.JSONInternalServerError(c, "Failed to initialize OCI service")
		return
	}

	// Test connection
	ctx := c.Request.Context()
	if err := ociService.TestConnection(ctx); err != nil {
		log.Printf("OCI Create: Connection test failed: %v", err)
		utils.JSONInternalServerError(c, "Failed to connect to OCI")
		return
	}

	// Get SSH keys
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

	// Convert to SSH public key
	sshPublicKey, err := h.cfService.ConvertPrivateKeyToSSH(csrConfig.PrivateKey)
	if err != nil {
		log.Printf("Error converting private key to SSH: %v", err)
		utils.JSONInternalServerError(c, "Failed to generate SSH key from CSR")
		return
	}

	// Use default timezone if not provided
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Create VPS with K3s using OCI service
	log.Printf("Creating OCI instance: %s with shape %s (%v OCPU, %v GB RAM)", req.Name, req.Shape, req.OCPU, req.Memory)
	ociInstance, err := ociService.CreateVPSWithK3s(ctx, req.Name, req.Shape, sshPublicKey, timezone, req.OCPU, req.Memory)
	if err != nil {
		log.Printf("Error creating OCI instance: %v", err)
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to create OCI instance: %v", err))
		return
	}

	// Create VPS config in KV store
	serverID := int(time.Now().Unix()) // Generate unique ID
	vpsConfig, err := h.vpsService.CreateOCIVPSConfig(
		token, accountID,
		ociInstance.DisplayName, ociInstance.PublicIP, "ubuntu", ociInstance.Shape,
		serverID, csrConfig.PrivateKey, sshPublicKey,
	)
	if err != nil {
		log.Printf("Error creating OCI VPS config: %v", err)
		// Instance was created but config failed - log this for cleanup
		log.Printf("WARNING: OCI instance %s (%s) was created but config storage failed", ociInstance.ID, ociInstance.PublicIP)
		utils.JSONInternalServerError(c, "Instance created but configuration storage failed")
		return
	}

	// Also store the OCI instance ID for future management
	vpsConfig.ProviderInstanceID = ociInstance.ID
	if err := h.vpsService.UpdateVPSConfig(token, accountID, serverID, vpsConfig); err != nil {
		log.Printf("Warning: Failed to update VPS config with OCI instance ID: %v", err)
	}

	log.Printf("✅ Created OCI instance: %s (ID: %s) with IP: %s", ociInstance.DisplayName, ociInstance.ID, ociInstance.PublicIP)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OCI instance created successfully with K3s and Helm",
		"server": gin.H{
			"id":     serverID,
			"name":   ociInstance.DisplayName,
			"oci_id": ociInstance.ID,
			"public_net": gin.H{
				"ipv4": gin.H{
					"ip": ociInstance.PublicIP,
				},
			},
			"lifecycle_state":     ociInstance.LifecycleState,
			"shape":               ociInstance.Shape,
			"availability_domain": ociInstance.AvailabilityDomain,
		},
		"config": vpsConfig,
	})
}

// HandleOCIDelete deletes an OCI instance and cleans up configuration
func (h *VPSLifecycleHandler) HandleOCIDelete(c *gin.Context) {
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

	// Get VPS config to get the OCI instance ID
	vpsConfig, err := h.vpsService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Error getting VPS config for deletion: %v", err)
		utils.JSONInternalServerError(c, "Failed to get VPS configuration")
		return
	}

	// Verify it's an OCI instance
	if vpsConfig.Provider != "oci" && vpsConfig.Provider != "OCI" && vpsConfig.Provider != "Oracle Cloud Infrastructure (OCI)" {
		utils.JSONBadRequest(c, "This endpoint is only for OCI instances")
		return
	}

	// Get OCI auth token
	ociToken, err := utils.GetOCIAuthToken(token, accountID)
	if err != nil {
		log.Printf("OCI Delete: Failed to retrieve OCI auth token: %v", err)
		// Still try to clean up local config even if we can't delete the instance
		log.Printf("Cleaning up local configuration for server %d", serverID)
		if err := h.vpsService.DeleteVPSConfig(token, accountID, serverID); err != nil {
			log.Printf("Error cleaning up VPS config: %v", err)
		}
		utils.JSONInternalServerError(c, "OCI auth token not found, but local configuration cleaned up")
		return
	}

	// Create OCI service
	ociService, err := services.NewOCIService(ociToken)
	if err != nil {
		log.Printf("OCI Delete: Failed to create OCI service: %v", err)
		utils.JSONInternalServerError(c, "Failed to initialize OCI service")
		return
	}

	ctx := c.Request.Context()

	// Delete the OCI instance if we have the instance ID
	if vpsConfig.ProviderInstanceID != "" {
		log.Printf("Deleting OCI instance: %s", vpsConfig.ProviderInstanceID)
		if err := ociService.DeleteVPSWithCleanup(ctx, vpsConfig.ProviderInstanceID, false); err != nil {
			// Check if it's a 404 error (instance not found)
			if strings.Contains(err.Error(), "not_found") || strings.Contains(err.Error(), "404") {
				log.Printf("OCI instance %s not found (already deleted), continuing with local cleanup", vpsConfig.ProviderInstanceID)
			} else {
				log.Printf("Error deleting OCI instance %s: %v", vpsConfig.ProviderInstanceID, err)
			}
			// Continue with local cleanup even if cloud deletion fails
		} else {
			log.Printf("✅ Deleted OCI instance: %s", vpsConfig.ProviderInstanceID)
		}
	} else {
		log.Printf("Warning: No OCI instance ID found for server %d, skipping cloud deletion", serverID)
	}

	// Clean up local configuration
	if err := h.vpsService.DeleteVPSConfig(token, accountID, serverID); err != nil {
		log.Printf("Error cleaning up VPS config: %v", err)
		utils.JSONInternalServerError(c, "Failed to clean up VPS configuration")
		return
	}

	// Invalidate VPS cache to ensure immediate UI update
	h.vpsService.InvalidateVPSCache(accountID)

	log.Printf("✅ Deleted OCI instance and cleaned up configuration for server: %s (ID: %d)", vpsConfig.Name, serverID)
	utils.VPSDeletionSuccess(c)
}

// HandleOCIPowerOff powers off an OCI instance
func (h *VPSLifecycleHandler) HandleOCIPowerOff(c *gin.Context) {
	h.performOCIServerAction(c, "poweroff")
}

// HandleOCIPowerOn powers on an OCI instance
func (h *VPSLifecycleHandler) HandleOCIPowerOn(c *gin.Context) {
	h.performOCIServerAction(c, "poweron")
}

// HandleOCIReboot reboots an OCI instance
func (h *VPSLifecycleHandler) HandleOCIReboot(c *gin.Context) {
	h.performOCIServerAction(c, "reboot")
}

// performOCIServerAction is a helper for OCI server power management actions
func (h *VPSLifecycleHandler) performOCIServerAction(c *gin.Context, action string) {
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

	// Get VPS config to get the OCI instance ID
	vpsConfig, err := h.vpsService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Error getting VPS config for action %s: %v", action, err)
		utils.JSONInternalServerError(c, "Failed to get VPS configuration")
		return
	}

	// Verify it's an OCI instance
	if vpsConfig.Provider != "oci" && vpsConfig.Provider != "OCI" && vpsConfig.Provider != "Oracle Cloud Infrastructure (OCI)" {
		utils.JSONBadRequest(c, "This endpoint is only for OCI instances")
		return
	}

	if vpsConfig.ProviderInstanceID == "" {
		utils.JSONInternalServerError(c, "OCI instance ID not found in configuration")
		return
	}

	// Get OCI auth token
	ociToken, err := utils.GetOCIAuthToken(token, accountID)
	if err != nil {
		log.Printf("OCI %s: Failed to retrieve OCI auth token: %v", action, err)
		utils.JSONInternalServerError(c, "OCI auth token not found")
		return
	}

	// Create OCI service
	ociService, err := services.NewOCIService(ociToken)
	if err != nil {
		log.Printf("OCI %s: Failed to create OCI service: %v", action, err)
		utils.JSONInternalServerError(c, "Failed to initialize OCI service")
		return
	}

	ctx := c.Request.Context()

	// Perform the action
	var actionErr error
	switch action {
	case "poweroff":
		actionErr = ociService.PowerOffInstance(ctx, vpsConfig.ProviderInstanceID)
	case "poweron":
		actionErr = ociService.PowerOnInstance(ctx, vpsConfig.ProviderInstanceID)
	case "reboot":
		actionErr = ociService.RebootInstance(ctx, vpsConfig.ProviderInstanceID)
	default:
		utils.JSONBadRequest(c, "Invalid action")
		return
	}

	if actionErr != nil {
		log.Printf("Error performing %s on OCI instance %s: %v", action, vpsConfig.ProviderInstanceID, actionErr)
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to %s instance", action))
		return
	}

	log.Printf("✅ Successfully performed %s on OCI instance: %s", action, vpsConfig.ProviderInstanceID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Server %s completed successfully", action),
	})
}

// HandleOCIValidateToken validates an OCI auth token
func (h *VPSLifecycleHandler) HandleOCIValidateToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "Invalid request data")
		return
	}

	// Validate the token format
	if err := utils.ValidateOCIToken(req.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Try to create service and test connection
	ociService, err := services.NewOCIService(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": fmt.Sprintf("Failed to create OCI service: %v", err),
		})
		return
	}

	ctx := c.Request.Context()
	if err := ociService.TestConnection(ctx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": fmt.Sprintf("Connection test failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "OCI auth token is valid",
	})
}

// HandleOCICheckToken checks if an OCI auth token exists in KV store
func (h *VPSLifecycleHandler) HandleOCICheckToken(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Try to get existing OCI token
	_, err := utils.GetOCIAuthToken(token, accountID)
	exists := err == nil

	log.Printf("OCI token check for account %s: exists=%v", accountID, exists)

	c.JSON(http.StatusOK, gin.H{
		"exists": exists,
	})
}

// HandleOCIStoreToken stores an OCI auth token in KV
func (h *VPSLifecycleHandler) HandleOCIStoreToken(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	var req struct {
		OCIToken string `json:"oci_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "Invalid request data")
		return
	}

	// Validate and store the token
	if err := utils.SetOCIAuthToken(token, accountID, req.OCIToken); err != nil {
		log.Printf("Error storing OCI auth token: %v", err)
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to store OCI auth token: %v", err))
		return
	}

	log.Printf("✅ Stored OCI auth token for account %s", accountID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OCI auth token stored successfully",
	})
}

// HandleOCIGetHomeRegion retrieves the home region for the tenancy
func (h *VPSLifecycleHandler) HandleOCIGetHomeRegion(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Get stored OCI auth token
	ociAuthToken, err := utils.GetOCIAuthToken(token, accountID)
	if err != nil {
		log.Printf("❌ Failed to retrieve OCI auth token for account %s: %v", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to retrieve OCI auth token",
		})
		return
	}

	if ociAuthToken == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "OCI auth token not found",
		})
		return
	}

	// Create OCI service
	ociService, err := services.NewOCIService(ociAuthToken)
	if err != nil {
		log.Printf("❌ Failed to create OCI service for account %s: %v", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create OCI service",
		})
		return
	}

	// Get home region
	homeRegion, err := ociService.GetTenancyHomeRegion(c.Request.Context())
	if err != nil {
		log.Printf("❌ Failed to get home region for account %s: %v", accountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get home region",
		})
		return
	}

	log.Printf("✅ Retrieved home region %s for account %s", homeRegion, accountID)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"home_region": homeRegion,
	})
}

// HandleUpdateVPSConfig allows manual updates to VPS configuration
func (h *VPSLifecycleHandler) HandleUpdateVPSConfig(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	serverIDStr := c.Param("id")
	serverID, err := utils.ParseServerID(serverIDStr)
	if err != nil {
		utils.JSONServerIDInvalid(c)
		return
	}

	// Parse request body
	var updateRequest struct {
		Region   string  `json:"region,omitempty"`
		OCPU     float32 `json:"ocpu,omitempty"`
		Memory   float32 `json:"memory,omitempty"`
		Timezone string  `json:"timezone,omitempty"`
		Location string  `json:"location,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		utils.JSONBadRequest(c, "Invalid request format")
		return
	}

	// Get existing VPS config
	vpsConfig, err := h.vpsService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Error getting VPS config for update: %v", err)
		utils.JSONInternalServerError(c, "Failed to get VPS configuration")
		return
	}

	// Update fields if provided
	if updateRequest.Region != "" {
		vpsConfig.Region = updateRequest.Region
	}
	if updateRequest.OCPU > 0 {
		vpsConfig.OCPU = updateRequest.OCPU
	}
	if updateRequest.Memory > 0 {
		vpsConfig.Memory = updateRequest.Memory
	}
	if updateRequest.Timezone != "" {
		vpsConfig.Timezone = updateRequest.Timezone
	}
	if updateRequest.Location != "" {
		vpsConfig.Location = updateRequest.Location
	}

	// Store updated config
	if err := h.vpsService.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		log.Printf("Error storing updated VPS config: %v", err)
		utils.JSONInternalServerError(c, "Failed to update VPS configuration")
		return
	}

	// Invalidate cache
	h.vpsService.InvalidateVPSCache(accountID)

	log.Printf("✅ Updated VPS config for server %d", serverID)
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "VPS configuration updated successfully",
		"config":  vpsConfig,
	})
}
