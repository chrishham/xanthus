package router

import (
	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/chrishham/xanthus/internal/middleware"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// RouteConfig holds all the handler instances
type RouteConfig struct {
	AuthHandler *handlers.AuthHandler
	DNSHandler  *handlers.DNSHandler
	VPSHandler  *handlers.VPSHandler
	AppsHandler *handlers.ApplicationsHandler
}

// SetupRoutes configures all application routes
func SetupRoutes(r *gin.Engine, config RouteConfig) {
	setupPublicRoutes(r, config)
	setupProtectedRoutes(r, config)
	setupAPIRoutes(r, config)
}

// setupPublicRoutes configures routes that don't require authentication
func setupPublicRoutes(r *gin.Engine, config RouteConfig) {
	// Authentication routes
	r.GET("/", config.AuthHandler.HandleRoot)
	r.GET("/login", config.AuthHandler.HandleLoginPage)
	r.POST("/login", config.AuthHandler.HandleLogin)
	r.GET("/health", config.AuthHandler.HandleHealth)
}

// setupProtectedRoutes configures routes that require authentication
func setupProtectedRoutes(r *gin.Engine, config RouteConfig) {
	// Apply authentication middleware to protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())

	// Main application pages
	protected.GET("/main", handleMainPage)
	protected.GET("/setup", handleSetupPage)
	protected.POST("/setup/hetzner", handleSetupHetzner)
	protected.GET("/logout", config.AuthHandler.HandleLogout)

	// DNS management routes
	dns := protected.Group("/dns")
	{
		dns.GET("", config.DNSHandler.HandleDNSConfigPage)
		dns.GET("/list", config.DNSHandler.HandleDNSList)
		dns.POST("/configure", config.DNSHandler.HandleDNSConfigure)
		dns.POST("/remove", config.DNSHandler.HandleDNSRemove)
	}

	// VPS management routes
	vps := protected.Group("/vps")
	{
		vps.GET("", config.VPSHandler.HandleVPSManagePage)
		vps.GET("/list", config.VPSHandler.HandleVPSList)
		vps.GET("/create", config.VPSHandler.HandleVPSCreatePage)
		vps.GET("/check-key", handleVPSCheckKey)
		vps.POST("/validate-key", handleVPSValidateKey)
		vps.GET("/locations", config.VPSHandler.HandleVPSLocations)
		vps.GET("/server-types", config.VPSHandler.HandleVPSServerTypes)
		vps.GET("/server-options", config.VPSHandler.HandleVPSServerOptions)
		vps.POST("/validate-name", config.VPSHandler.HandleVPSValidateName)
		vps.POST("/create", config.VPSHandler.HandleVPSCreate)
		vps.POST("/delete", config.VPSHandler.HandleVPSDelete)
		vps.POST("/poweroff", config.VPSHandler.HandleVPSPowerOff)
		vps.POST("/poweron", config.VPSHandler.HandleVPSPowerOn)
		vps.POST("/reboot", config.VPSHandler.HandleVPSReboot)
		vps.GET("/ssh-key", handleVPSSSHKey)
		vps.GET("/:id/status", handleVPSStatus)
		vps.POST("/:id/configure", config.VPSHandler.HandleVPSConfigure)
		vps.POST("/:id/deploy", config.VPSHandler.HandleVPSDeploy)
		vps.GET("/:id/logs", handleVPSLogs)
		vps.POST("/:id/terminal", handleVPSTerminal)
	}

	// Terminal management routes
	terminal := protected.Group("/terminal")
	{
		terminal.GET("/:session_id", handleTerminalView)
		terminal.DELETE("/:session_id", handleTerminalStop)
	}

	// Applications management routes
	apps := protected.Group("/applications")
	{
		apps.GET("", config.AppsHandler.HandleApplicationsPage)
		apps.GET("/list", config.AppsHandler.HandleApplicationsList)
		apps.GET("/prerequisites", config.AppsHandler.HandleApplicationsPrerequisites)
		apps.GET("/vps/:id/repositories", handleVPSRepositories)
		apps.POST("/vps/:id/repositories", handleVPSAddRepository)
		apps.GET("/vps/:id/charts/:repo", handleVPSCharts)
		apps.POST("/create", config.AppsHandler.HandleApplicationsCreate)
		apps.POST("/:id/upgrade", config.AppsHandler.HandleApplicationUpgrade)
		apps.DELETE("/:id", config.AppsHandler.HandleApplicationDelete)
	}
}

// setupAPIRoutes configures API routes with appropriate middleware
func setupAPIRoutes(r *gin.Engine, config RouteConfig) {
	// API routes could be added here if needed
	api := r.Group("/api")
	api.Use(middleware.APIAuthMiddleware())

	// Add API routes here when needed
}

// Legacy handlers that still need to be migrated
// These should be moved to appropriate handler packages

func handleMainPage(c *gin.Context) {
	c.HTML(200, "main.html", nil)
}

func handleSetupPage(c *gin.Context) {
	// Try to get existing Hetzner API key for prepopulation
	var existingKey string
	token := c.GetString("cf_token")

	// Import necessary utility functions
	exists, accountID, err := utils.CheckKVNamespaceExists(token)
	if err == nil && exists {
		// If we can get the account ID, try to retrieve the existing key
		if hetznerKey, err := utils.GetHetznerAPIKey(token, accountID); err == nil {
			// Mask the key for security (show only first 4 and last 4 characters)
			if len(hetznerKey) > 8 {
				existingKey = hetznerKey[:4] + "..." + hetznerKey[len(hetznerKey)-4:]
			}
		}
	}

	c.HTML(200, "setup.html", map[string]interface{}{
		"existing_key": existingKey,
	})
}

func handleSetupHetzner(c *gin.Context) {
	// TODO: Move to setup handler or VPS handler - this needs full implementation
}

func handleVPSCheckKey(c *gin.Context) {
	// TODO: Move to VPS handler
}

func handleVPSValidateKey(c *gin.Context) {
	// TODO: Move to VPS handler
}

func handleVPSSSHKey(c *gin.Context) {
	// TODO: Move to VPS handler
}

func handleVPSStatus(c *gin.Context) {
	// TODO: Move to VPS handler
}

func handleVPSLogs(c *gin.Context) {
	// TODO: Move to VPS handler
}

func handleVPSTerminal(c *gin.Context) {
	// TODO: Move to VPS handler
}

func handleTerminalView(c *gin.Context) {
	// TODO: Move to terminal handler
}

func handleTerminalStop(c *gin.Context) {
	// TODO: Move to terminal handler
}

func handleVPSRepositories(c *gin.Context) {
	// TODO: Move to applications handler
}

func handleVPSAddRepository(c *gin.Context) {
	// TODO: Move to applications handler
}

func handleVPSCharts(c *gin.Context) {
	// TODO: Move to applications handler
}
