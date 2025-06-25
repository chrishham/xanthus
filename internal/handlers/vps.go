package handlers

import (
	"fmt"
	"log"
	"net/http"

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
		"Servers": servers,
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

// TODO: Complete VPS handler implementations
// The following handlers need to be extracted from main.go and implemented:

// Core VPS Lifecycle Management:
// - HandleVPSCreate (line 1440) - Create new VPS instance
// - HandleVPSDelete (line 1609) - Delete VPS instance  
// - HandleVPSCreatePage (line 2064) - VPS creation page

// VPS Configuration and Setup:
// - HandleVPSServerOptions (line 1363) - Server configuration options
// - HandleVPSConfigure (line 1847) - Configure VPS settings
// - HandleVPSDeploy (line 1921) - Deploy applications to VPS
// - HandleVPSLocations (line 2148) - Available VPS locations
// - HandleVPSServerTypes (line 2232) - Available server types
// - HandleVPSValidateName (line 2179) - Validate VPS name

// VPS Power Management:
// - HandleVPSPowerOff (line 1676) - Power off VPS
// - HandleVPSPowerOn (line 1680) - Power on VPS
// - HandleVPSReboot (line 1684) - Reboot VPS
// - performVPSAction (line 1688) - Generic VPS action handler

// VPS SSH and Access:
// - HandleVPSSSHKey (line 1750) - SSH key management
// - HandleVPSCheckKey (line 2073) - Check SSH key status
// - HandleVPSValidateKey (line 2106) - Validate SSH key

// VPS Monitoring and Status:
// - HandleVPSStatus (line 1797) - Get VPS status
// - HandleVPSLogs (line 1989) - View VPS logs

// VPS Terminal Access:
// - HandleVPSTerminal (line 2305) - Web terminal interface
// - HandleTerminalView (line 2359) - Terminal view handler
// - HandleTerminalStop (line 2379) - Stop terminal session

// VPS Application Management:
// - HandleVPSRepositories (line 2778) - Manage Helm repositories
// - HandleVPSAddRepository (line 2878) - Add new repository
// - HandleVPSCharts (line 3009) - List available Helm charts

// TODO: These utility functions have been moved to internal/utils/placeholders.go
// They need to be properly implemented and moved to domain-specific utils files