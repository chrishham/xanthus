package router

import (
	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/chrishham/xanthus/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RouteConfig holds all the handler instances
type RouteConfig struct {
	AuthHandler     *handlers.AuthHandler
	DNSHandler      *handlers.DNSHandler
	VPSHandler      *handlers.VPSHandler
	AppsHandler     *handlers.ApplicationsHandler
	TerminalHandler *handlers.TerminalHandler
	PagesHandler    *handlers.PagesHandler
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
	protected.GET("/main", config.PagesHandler.HandleMainPage)
	protected.GET("/setup", config.PagesHandler.HandleSetupPage)
	protected.POST("/setup/hetzner", config.VPSHandler.HandleSetupHetzner)
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
		vps.GET("/check-key", config.VPSHandler.HandleVPSCheckKey)
		vps.POST("/validate-key", config.VPSHandler.HandleVPSValidateKey)
		vps.GET("/locations", config.VPSHandler.HandleVPSLocations)
		vps.GET("/server-types", config.VPSHandler.HandleVPSServerTypes)
		vps.GET("/server-options", config.VPSHandler.HandleVPSServerOptions)
		vps.POST("/validate-name", config.VPSHandler.HandleVPSValidateName)
		vps.POST("/create", config.VPSHandler.HandleVPSCreate)
		vps.POST("/delete", config.VPSHandler.HandleVPSDelete)
		vps.POST("/poweroff", config.VPSHandler.HandleVPSPowerOff)
		vps.POST("/poweron", config.VPSHandler.HandleVPSPowerOn)
		vps.POST("/reboot", config.VPSHandler.HandleVPSReboot)
		vps.GET("/ssh-key", config.VPSHandler.HandleVPSSSHKey)
		vps.GET("/:id/status", config.VPSHandler.HandleVPSStatus)
		vps.GET("/:id/info", config.VPSHandler.HandleVPSInfo)
		vps.GET("/:id/argocd-credentials", config.VPSHandler.HandleVPSArgoCDCredentials)
		vps.POST("/:id/configure", config.VPSHandler.HandleVPSConfigure)
		vps.POST("/:id/deploy", config.VPSHandler.HandleVPSDeploy)
		vps.GET("/:id/logs", config.VPSHandler.HandleVPSLogs)
		vps.POST("/:id/terminal", config.VPSHandler.HandleVPSTerminal)
	}

	// Terminal management routes
	terminal := protected.Group("/terminal")
	{
		terminal.GET("/:session_id", config.TerminalHandler.HandleTerminalView)
		terminal.DELETE("/:session_id", config.TerminalHandler.HandleTerminalStop)
	}

	// Applications management routes
	apps := protected.Group("/applications")
	{
		apps.GET("", config.AppsHandler.HandleApplicationsPage)
		apps.GET("/list", config.AppsHandler.HandleApplicationsList)
		apps.GET("/prerequisites", config.AppsHandler.HandleApplicationsPrerequisites)
		apps.POST("/create", config.AppsHandler.HandleApplicationsCreate)
		apps.POST("/:id/upgrade", config.AppsHandler.HandleApplicationUpgrade)
		apps.POST("/:id/password", config.AppsHandler.HandleApplicationPasswordChange)
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
