package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// VPSHandler contains dependencies for VPS-related operations
type VPSHandler struct {
	// Add dependencies here as needed
}

// NewVPSHandler creates a new VPS handler instance
func NewVPSHandler() *VPSHandler {
	return &VPSHandler{}
}

// HandleVPSManagePage renders the VPS management page
func (h *VPSHandler) HandleVPSManagePage(c *gin.Context) {
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

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		log.Printf("Error getting Hetzner API key: %v", err)
		c.Redirect(http.StatusTemporaryRedirect, "/setup")
		return
	}

	// Initialize Hetzner service and list servers
	hetznerService := services.NewHetznerService()
	servers, err := hetznerService.ListServers(hetznerKey)
	if err != nil {
		log.Printf("Error listing servers: %v", err)
		servers = []services.HetznerServer{}
	}

	// Enhance servers with cost information for initial page load
	kvService := services.NewKVService()
	for i := range servers {
		if vpsConfig, err := kvService.GetVPSConfig(token, accountID, servers[i].ID); err == nil {
			// Calculate accumulated cost
			if accumulatedCost, err := kvService.CalculateVPSCosts(vpsConfig); err == nil {
				if servers[i].Labels == nil {
					servers[i].Labels = make(map[string]string)
				}
				servers[i].Labels["accumulated_cost"] = fmt.Sprintf("%.2f", accumulatedCost)
				servers[i].Labels["monthly_cost"] = fmt.Sprintf("%.2f", vpsConfig.MonthlyRate)
				servers[i].Labels["hourly_cost"] = fmt.Sprintf("%.4f", vpsConfig.HourlyRate)
			}
		}
	}

	c.HTML(http.StatusOK, "vps-manage.html", gin.H{
		"Servers":    servers,
		"ActivePage": "vps",
	})
}

// HandleVPSList returns a JSON list of VPS instances
func (h *VPSHandler) HandleVPSList(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// List servers
	hetznerService := services.NewHetznerService()
	servers, err := hetznerService.ListServers(hetznerKey)
	if err != nil {
		log.Printf("Error listing servers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list servers"})
		return
	}

	// Enhance servers with cost information
	kvService := services.NewKVService()
	for i := range servers {
		if vpsConfig, err := kvService.GetVPSConfig(token, accountID, servers[i].ID); err == nil {
			// Calculate accumulated cost
			if accumulatedCost, err := kvService.CalculateVPSCosts(vpsConfig); err == nil {
				servers[i].Labels["accumulated_cost"] = fmt.Sprintf("%.2f", accumulatedCost)
				servers[i].Labels["monthly_cost"] = fmt.Sprintf("%.2f", vpsConfig.MonthlyRate)
				servers[i].Labels["hourly_cost"] = fmt.Sprintf("%.4f", vpsConfig.HourlyRate)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

// HandleVPSCreate creates a new VPS instance on Hetzner Cloud with K3s setup
func (h *VPSHandler) HandleVPSCreate(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	name := c.PostForm("name")
	domain := c.PostForm("domain")
	location := c.PostForm("location")
	serverType := c.PostForm("server_type")

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server name is required"})
		return
	}
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required"})
		return
	}
	if location == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server location is required"})
		return
	}
	if serverType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server type is required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Check if Hetzner API key exists - if not, guide user to setup
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Hetzner API key not configured",
			"setup_required": true,
			"setup_step":     "hetzner_api",
			"message":        "Please configure your Hetzner API key first in the setup section"})
		return
	}

	// Get SSL CSR configuration
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		CSR        string `json:"csr"`
		PrivateKey string `json:"private_key"`
		CreatedAt  string `json:"created_at"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Error getting CSR from KV: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SSL CSR configuration not found. Please logout and login again."})
		return
	}

	// Convert CSR private key to SSH public key
	cfService := services.NewCloudflareService()
	sshPublicKey, err := cfService.ConvertPrivateKeyToSSH(csrConfig.PrivateKey)
	if err != nil {
		log.Printf("Error converting private key to SSH: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate SSH key from CSR"})
		return
	}

	// Validate SSH public key format
	if !strings.HasPrefix(sshPublicKey, "ssh-rsa ") {
		keyPreview := sshPublicKey
		if len(keyPreview) > 50 {
			keyPreview = keyPreview[:50] + "..."
		}
		log.Printf("Invalid SSH public key format: %s", keyPreview)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generated SSH key has invalid format"})
		return
	}
	log.Printf("✅ Generated SSH public key (length: %d)", len(sshPublicKey))

	// Create or find SSH key in Hetzner Cloud
	hetznerService := services.NewHetznerService()
	sshKeyName := fmt.Sprintf("xanthus-key-%d", time.Now().Unix())
	sshKey, err := hetznerService.CreateOrFindSSHKey(hetznerKey, sshKeyName, sshPublicKey)
	if err != nil {
		log.Printf("Error creating/finding SSH key in Hetzner: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create SSH key in Hetzner Cloud: %v", err)})
		return
	}

	// Use the actual key name from the found/created key
	sshKeyName = sshKey.Name
	log.Printf("✅ Using SSH key: %s (ID: %d)", sshKeyName, sshKey.ID)

	// Create server using cloud-init script
	server, err := hetznerService.CreateServer(hetznerKey, name, serverType, location, sshKeyName)
	if err != nil {
		log.Printf("Error creating server: %v", err)

		// Check for specific error types and provide user-friendly messages
		errorStr := err.Error()
		if strings.Contains(errorStr, "server name is already used") || strings.Contains(errorStr, "uniqueness_error") {
			c.JSON(http.StatusConflict, gin.H{"error": "A server with this name already exists. Please choose a different name."})
			return
		}

		// Generic error for other cases
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create server: %v", err)})
		return
	}

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

	// Initialize KV service for storing VPS configuration
	kvService := services.NewKVService()

	// Store VPS configuration in KV
	vpsConfig := &services.VPSConfig{
		ServerID:    server.ID,
		Name:        server.Name,
		ServerType:  serverType,
		Location:    location,
		PublicIPv4:  server.PublicNet.IPv4.IP,
		Status:      server.Status,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		SSHKeyName:  sshKeyName,
		SSHUser:     "root",
		SSHPort:     22,
		HourlyRate:  hourlyRate,
		MonthlyRate: monthlyRate,
	}

	if err := kvService.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		log.Printf("Error storing VPS config: %v", err)
		// Don't fail the creation, just log the error
	}

	// Configure DNS records for the VPS
	log.Printf("🔧 Starting DNS configuration for domain: %s, VPS IP: %s", domain, server.PublicNet.IPv4.IP)
	cfService = services.NewCloudflareService()
	if err := cfService.ConfigureDNSForVPS(token, domain, server.PublicNet.IPv4.IP); err != nil {
		log.Printf("❌ Failed to configure DNS for domain %s: %v", domain, err)
		// Don't fail the creation, but log the warning
	} else {
		log.Printf("✅ DNS configured for domain %s pointing to %s", domain, server.PublicNet.IPv4.IP)
	}

	// Configure TLS certificates and ArgoCD ingress (async operation)
	go func() {
		// Wait a bit for the server to be fully ready
		time.Sleep(2 * time.Minute)

		// Get domain SSL configuration
		domainConfig, err := kvService.GetDomainSSLConfig(token, accountID, domain)
		if err != nil {
			log.Printf("Warning: Could not get SSL config for domain %s: %v", domain, err)
			return
		}

		// Connect to VPS via SSH
		sshService := services.NewSSHService()
		conn, err := sshService.ConnectToVPS(server.PublicNet.IPv4.IP, "root", csrConfig.PrivateKey)
		if err != nil {
			log.Printf("Warning: Could not connect to VPS for TLS setup: %v", err)
			return
		}
		defer conn.Close()

		// Create TLS secret for the domain
		if err := sshService.CreateTLSSecret(conn, domain, domainConfig.Certificate, domainConfig.PrivateKey); err != nil {
			log.Printf("Warning: Failed to create TLS secret for domain %s: %v", domain, err)
		} else {
			log.Printf("✅ TLS secret created for domain %s", domain)
		}

		// Create ArgoCD ingress
		if err := sshService.CreateArgoCDIngress(conn, domain); err != nil {
			log.Printf("Warning: Failed to create ArgoCD ingress for domain %s: %v", domain, err)
		} else {
			log.Printf("✅ ArgoCD ingress configured for https://argocd.%s", domain)
		}
	}()

	// Update VPS config to include domain
	vpsConfig.Domain = domain

	// Update stored VPS configuration with domain
	if err := kvService.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		log.Printf("Warning: Error updating VPS config with domain: %v", err)
	}

	log.Printf("✅ Created server: %s (ID: %d) with IPv4: %s", server.Name, server.ID, server.PublicNet.IPv4.IP)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Server created successfully with K3s, Helm, ArgoCD, and DNS configuration",
		"server":  server,
		"config":  vpsConfig,
	})
}

// HandleVPSDelete deletes a VPS instance and cleans up configuration
func (h *VPSHandler) HandleVPSDelete(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	serverID := c.PostForm("server_id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server ID is required"})
		return
	}

	// Convert serverID to int
	var id int
	if _, err := fmt.Sscanf(serverID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// Get VPS configuration before deletion (for logging)
	kvService := services.NewKVService()
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, id)
	if err != nil {
		log.Printf("Warning: Could not get VPS config for server %d: %v", id, err)
	}

	// Delete server from Hetzner
	hetznerService := services.NewHetznerService()
	if err := hetznerService.DeleteServer(hetznerKey, id); err != nil {
		log.Printf("Error deleting server %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete server: %v", err)})
		return
	}

	// Clean up VPS configuration from KV
	if err := kvService.DeleteVPSConfig(token, accountID, id); err != nil {
		log.Printf("Warning: Could not delete VPS config for server %d: %v", id, err)
		// Don't fail the deletion, just log the warning
	}

	serverName := fmt.Sprintf("Server %d", id)
	if vpsConfig != nil {
		serverName = vpsConfig.Name
	}

	log.Printf("✅ Deleted server: %s (ID: %d) and cleaned up configuration", serverName, id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Server deleted successfully and configuration cleaned up",
	})
}

// HandleVPSCreatePage renders the VPS creation page
func (h *VPSHandler) HandleVPSCreatePage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	c.HTML(http.StatusOK, "vps-create.html", gin.H{
		"ActivePage": "vps",
	})
}

// HandleVPSServerOptions fetches available server types and locations with filtering/sorting
func (h *VPSHandler) HandleVPSServerOptions(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// Fetch locations and server types
	locations, err := utils.FetchHetznerLocations(hetznerKey)
	if err != nil {
		log.Printf("Error fetching locations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch locations"})
		return
	}

	serverTypes, err := utils.FetchHetznerServerTypes(hetznerKey)
	if err != nil {
		log.Printf("Error fetching server types: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch server types"})
		return
	}

	// Filter to only shared vCPU servers for cost efficiency
	sharedServerTypes := utils.FilterSharedVCPUServers(serverTypes)

	// Apply architecture filter if requested
	architectureFilter := c.Query("architecture")
	if architectureFilter != "" {
		var filteredTypes []models.HetznerServerType
		for _, serverType := range sharedServerTypes {
			if serverType.Architecture == architectureFilter {
				filteredTypes = append(filteredTypes, serverType)
			}
		}
		sharedServerTypes = filteredTypes
	}

	// Get sort parameter and sort
	sortBy := c.Query("sort")
	switch sortBy {
	case "price_desc":
		utils.SortServerTypesByPriceDesc(sharedServerTypes)
	case "price_asc":
		utils.SortServerTypesByPriceAsc(sharedServerTypes)
	case "cpu_desc":
		utils.SortServerTypesByCPUDesc(sharedServerTypes)
	case "cpu_asc":
		utils.SortServerTypesByCPUAsc(sharedServerTypes)
	case "memory_desc":
		utils.SortServerTypesByMemoryDesc(sharedServerTypes)
	case "memory_asc":
		utils.SortServerTypesByMemoryAsc(sharedServerTypes)
	default:
		// Default to price ascending
		utils.SortServerTypesByPriceAsc(sharedServerTypes)
	}

	c.JSON(http.StatusOK, gin.H{
		"locations":   locations,
		"serverTypes": sharedServerTypes,
	})
}

// HandleVPSConfigure configures VPS with SSL certificates for a specific domain
func (h *VPSHandler) HandleVPSConfigure(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	serverIDStr := c.Param("id")
	var serverID int
	if _, err := fmt.Sscanf(serverIDStr, "%d", &serverID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required for SSL configuration"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get VPS configuration
	kvService := services.NewKVService()
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VPS configuration not found"})
		return
	}

	// Get SSL configuration for the domain
	domainConfig, err := kvService.GetDomainSSLConfig(token, accountID, domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SSL configuration not found for domain"})
		return
	}

	// Get CSR configuration for SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SSH private key not found"})
		return
	}

	// Connect to VPS and configure SSL
	sshService := services.NewSSHService()
	conn, err := sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("SSH connection failed: %v", err)})
		return
	}
	defer conn.Close()

	// Configure K3s with new SSL certificates
	if err := sshService.ConfigureK3s(conn, domainConfig.Certificate, domainConfig.PrivateKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to configure K3s: %v", err)})
		return
	}

	log.Printf("✅ Successfully configured VPS %d with SSL for domain %s", serverID, domain)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("VPS successfully configured with SSL certificates for %s", domain),
	})
}

// HandleVPSDeploy deploys Kubernetes manifests to a VPS
func (h *VPSHandler) HandleVPSDeploy(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	serverIDStr := c.Param("id")
	var serverID int
	if _, err := fmt.Sscanf(serverIDStr, "%d", &serverID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	manifest := c.PostForm("manifest")
	name := c.PostForm("name")
	if manifest == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manifest and name are required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get VPS configuration
	kvService := services.NewKVService()
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VPS configuration not found"})
		return
	}

	// Get CSR configuration for SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SSH private key not found"})
		return
	}

	// Connect to VPS and deploy manifest
	sshService := services.NewSSHService()
	conn, err := sshService.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("SSH connection failed: %v", err)})
		return
	}
	defer conn.Close()

	// Deploy the Kubernetes manifest
	if err := sshService.DeployManifest(conn, manifest, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to deploy manifest: %v", err)})
		return
	}

	log.Printf("✅ Successfully deployed %s to VPS %d", name, serverID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Successfully deployed %s to VPS", name),
	})
}

// HandleVPSLocations fetches available VPS locations from Hetzner
func (h *VPSHandler) HandleVPSLocations(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// Fetch locations
	locations, err := utils.FetchHetznerLocations(hetznerKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch locations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"locations": locations})
}

// HandleVPSServerTypes fetches available server types for a specific location
func (h *VPSHandler) HandleVPSServerTypes(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	location := c.Query("location")
	if location == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Location parameter is required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// Fetch server types
	serverTypes, err := utils.FetchHetznerServerTypes(hetznerKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch server types"})
		return
	}

	// Filter to only shared vCPU servers
	sharedServerTypes := utils.FilterSharedVCPUServers(serverTypes)

	// Get availability for the selected location
	availability, err := utils.FetchServerAvailability(hetznerKey)
	if err != nil {
		log.Printf("Warning: Could not fetch availability: %v", err)
		availability = make(map[string]map[int]bool)
	}

	// Add availability and pricing information
	for i := range sharedServerTypes {
		// Check availability in the selected location
		if locationAvailability, exists := availability[location]; exists {
			sharedServerTypes[i].AvailableLocations = map[string]bool{location: locationAvailability[sharedServerTypes[i].ID]}
		} else {
			// Default to available if we can't check
			sharedServerTypes[i].AvailableLocations = map[string]bool{location: true}
		}

		// Calculate monthly price from hourly
		monthlyPrice := utils.GetServerTypeMonthlyPrice(sharedServerTypes[i])
		// Add a monthlyPrice field for easy access in frontend
		sharedServerTypes[i].Prices = append(sharedServerTypes[i].Prices, models.HetznerPrice{
			Location: "monthly_calc",
			PriceMonthly: models.HetznerPriceDetail{
				Gross: fmt.Sprintf("%.2f", monthlyPrice),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"serverTypes": sharedServerTypes})
}

// HandleVPSValidateName validates VPS names against existing servers
func (h *VPSHandler) HandleVPSValidateName(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// Check if name already exists by listing servers
	hetznerService := services.NewHetznerService()
	servers, err := hetznerService.ListServers(hetznerKey)
	if err != nil {
		log.Printf("Error checking existing servers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing servers"})
		return
	}

	// Check if name is already in use
	for _, server := range servers {
		if server.Name == name {
			c.JSON(http.StatusConflict, gin.H{
				"available": false,
				"error":     "A VPS with this name already exists in your Hetzner account",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"available": true,
		"message":   "Name is available",
	})
}

// HandleVPSPowerOff powers off a VPS instance
func (h *VPSHandler) HandleVPSPowerOff(c *gin.Context) {
	h.performVPSAction(c, "poweroff", "powered off")
}

// HandleVPSPowerOn powers on a VPS instance
func (h *VPSHandler) HandleVPSPowerOn(c *gin.Context) {
	h.performVPSAction(c, "poweron", "powered on")
}

// HandleVPSReboot reboots a VPS instance
func (h *VPSHandler) HandleVPSReboot(c *gin.Context) {
	h.performVPSAction(c, "reboot", "rebooted")
}

// performVPSAction is a generic function that performs power management actions on VPS servers
func (h *VPSHandler) performVPSAction(c *gin.Context, action, actionText string) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	serverID := c.PostForm("server_id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server ID is required"})
		return
	}

	// Convert serverID to int
	var id int
	if _, err := fmt.Sscanf(serverID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Get Hetzner API key
	hetznerKey, err := utils.GetHetznerAPIKey(token, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Hetzner API key"})
		return
	}

	// Perform action
	hetznerService := services.NewHetznerService()
	var actionErr error
	switch action {
	case "poweroff":
		actionErr = hetznerService.PowerOffServer(hetznerKey, id)
	case "poweron":
		actionErr = hetznerService.PowerOnServer(hetznerKey, id)
	case "reboot":
		actionErr = hetznerService.RebootServer(hetznerKey, id)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	if actionErr != nil {
		log.Printf("Error performing %s on server %d: %v", action, id, actionErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to perform %s: %v", action, actionErr)})
		return
	}

	log.Printf("✅ Server %d %s successfully", id, actionText)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Server %s successfully", actionText),
	})
}

// HandleVPSCheckKey checks if Hetzner API key exists in KV storage
func (h *VPSHandler) HandleVPSCheckKey(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Invalid token")
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONResponse(c, http.StatusOK, gin.H{"exists": false})
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
func (h *VPSHandler) HandleVPSValidateKey(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Invalid token")
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

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Store the key
	client := &http.Client{Timeout: 10 * time.Second}
	encryptedKey, err := utils.EncryptData(apiKey, token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to encrypt API key")
		return
	}

	if err := utils.PutKVValue(client, token, accountID, "config:hetzner:api_key", encryptedKey); err != nil {
		utils.JSONInternalServerError(c, "Failed to store API key")
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{"success": true})
}

// HandleVPSSSHKey returns SSH private key for VPS access
func (h *VPSHandler) HandleVPSSSHKey(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Invalid token")
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
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
		utils.JSONInternalServerError(c, "SSH private key not found. Please logout and login again.")
		return
	}

	// Check if user wants to download the key
	download := c.Query("download")
	if download == "true" {
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", "attachment; filename=xanthus-ssh-key.pem")
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

// HandleVPSStatus gets VPS health status via SSH
func (h *VPSHandler) HandleVPSStatus(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Invalid token")
		return
	}

	serverIDStr := c.Param("id")
	var serverID int
	if _, err := fmt.Sscanf(serverIDStr, "%d", &serverID); err != nil {
		utils.JSONBadRequest(c, "Invalid server ID")
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration
	kvService := services.NewKVService()
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		utils.JSONNotFound(c, "VPS configuration not found")
		return
	}

	// Get CSR configuration for SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		utils.JSONInternalServerError(c, "SSH private key not found")
		return
	}

	// Check VPS health via SSH
	sshService := services.NewSSHService()
	status, err := sshService.CheckVPSHealth(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, serverID)
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to check VPS status: %v", err))
		return
	}

	utils.JSONResponse(c, http.StatusOK, status)
}

// HandleVPSLogs fetches VPS logs via SSH connection
func (h *VPSHandler) HandleVPSLogs(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Invalid token")
		return
	}

	serverIDStr := c.Param("id")
	var serverID int
	if _, err := fmt.Sscanf(serverIDStr, "%d", &serverID); err != nil {
		utils.JSONBadRequest(c, "Invalid server ID")
		return
	}

	// Get number of lines (default 100)
	lines := 100
	if linesStr := c.Query("lines"); linesStr != "" {
		if parsedLines, err := fmt.Sscanf(linesStr, "%d", &lines); err == nil && parsedLines > 0 {
			if lines > 1000 {
				lines = 1000 // Limit to prevent overwhelming response
			}
		}
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration
	kvService := services.NewKVService()
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		utils.JSONNotFound(c, "VPS configuration not found")
		return
	}

	// Get CSR configuration for SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		utils.JSONInternalServerError(c, "SSH private key not found")
		return
	}

	// Connect to VPS and get logs
	sshService := services.NewSSHService()
	logs, err := sshService.GetVPSLogs(vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey, lines)
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

// HandleVPSTerminal creates a web terminal session for VPS
func (h *VPSHandler) HandleVPSTerminal(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		utils.JSONUnauthorized(c, "Invalid token")
		return
	}

	serverIDStr := c.Param("id")
	var serverID int
	if _, err := fmt.Sscanf(serverIDStr, "%d", &serverID); err != nil {
		utils.JSONBadRequest(c, "Invalid server ID")
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to get account ID")
		return
	}

	// Get VPS configuration
	kvService := services.NewKVService()
	vpsConfig, err := kvService.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		utils.JSONNotFound(c, "VPS configuration not found")
		return
	}

	// Get CSR configuration for SSH private key
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		PrivateKey string `json:"private_key"`
	}
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		utils.JSONInternalServerError(c, "SSH private key not found")
		return
	}

	// Create terminal session
	terminalService := services.NewTerminalService()
	session, err := terminalService.CreateSession(serverID, vpsConfig.PublicIPv4, vpsConfig.SSHUser, csrConfig.PrivateKey)
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

// HandleSetupHetzner configures Hetzner API key in setup
func (h *VPSHandler) HandleSetupHetzner(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	hetznerKey := c.PostForm("hetzner_key")

	// Get account ID for checking existing key
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		log.Printf("Error getting account ID: %v", err)
		c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error: %s", err.Error())))
		return
	}

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
