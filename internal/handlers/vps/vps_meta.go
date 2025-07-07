package vps

import (
	"fmt"
	"log"
	"net/http"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// VPSMetaHandler handles VPS metadata and validation operations
type VPSMetaHandler struct {
	*BaseHandler
	vpsService *services.VPSService
}

// NewVPSMetaHandler creates a new VPS meta handler instance
func NewVPSMetaHandler() *VPSMetaHandler {
	return &VPSMetaHandler{
		BaseHandler: NewBaseHandler(),
		vpsService:  services.NewVPSService(),
	}
}

// HandleVPSManagePage redirects to Svelte VPS page
func (h *VPSMetaHandler) HandleVPSManagePage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app/vps")
}

// HandleVPSCreatePage redirects to Svelte VPS page  
func (h *VPSMetaHandler) HandleVPSCreatePage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app/vps")
}

// HandleVPSServerOptions fetches available server types and locations with filtering/sorting
func (h *VPSMetaHandler) HandleVPSServerOptions(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Get Hetzner API key
	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		return
	}

	// Fetch locations and server types
	locations, err := utils.FetchHetznerLocations(hetznerKey)
	if err != nil {
		log.Printf("Error fetching locations: %v", err)
		utils.JSONInternalServerError(c, "Failed to fetch locations")
		return
	}

	serverTypes, err := utils.FetchHetznerServerTypes(hetznerKey)
	if err != nil {
		log.Printf("Error fetching server types: %v", err)
		utils.JSONInternalServerError(c, "Failed to fetch server types")
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

	c.JSON(200, gin.H{ // http.StatusOK
		"locations":   locations,
		"serverTypes": sharedServerTypes,
	})
}

// HandleVPSLocations fetches available VPS locations from Hetzner
func (h *VPSMetaHandler) HandleVPSLocations(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Get Hetzner API key
	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		return
	}

	// Fetch locations
	locations, err := utils.FetchHetznerLocations(hetznerKey)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to fetch locations")
		return
	}

	c.JSON(200, gin.H{"locations": locations}) // http.StatusOK
}

// HandleVPSServerTypes fetches available server types for a specific location
func (h *VPSMetaHandler) HandleVPSServerTypes(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	location := c.Query("location")
	if location == "" {
		utils.JSONBadRequest(c, "Location parameter is required")
		return
	}

	// Get Hetzner API key
	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		return
	}

	// Fetch server types
	serverTypes, err := utils.FetchHetznerServerTypes(hetznerKey)
	if err != nil {
		utils.JSONInternalServerError(c, "Failed to fetch server types")
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

	c.JSON(200, gin.H{"serverTypes": sharedServerTypes}) // http.StatusOK
}

// HandleVPSValidateName validates VPS names against existing servers
func (h *VPSMetaHandler) HandleVPSValidateName(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	name := c.PostForm("name")
	if name == "" {
		utils.JSONBadRequest(c, "Name is required")
		return
	}

	// Get Hetzner API key
	hetznerKey, valid := h.getHetznerKey(c, token, accountID)
	if !valid {
		return
	}

	// Check if name already exists by listing servers
	servers, err := h.hetznerService.ListServers(hetznerKey)
	if err != nil {
		log.Printf("Error checking existing servers: %v", err)
		utils.JSONInternalServerError(c, "Failed to check existing servers")
		return
	}

	// Check if name is already in use
	for _, server := range servers {
		if server.Name == name {
			utils.JSONVPSNameUnavailable(c)
			return
		}
	}

	utils.JSONVPSNameAvailable(c)
}
