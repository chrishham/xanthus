package router

import (
	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/chrishham/xanthus/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RouteConfig holds all the handler instances
type RouteConfig struct {
	AuthHandler         *handlers.AuthHandler
	DNSHandler          *handlers.DNSHandler
	VPSLifecycleHandler *handlers.VPSLifecycleHandler
	VPSInfoHandler      *handlers.VPSInfoHandler
	VPSConfigHandler    *handlers.VPSConfigHandler
	VPSMetaHandler      *handlers.VPSMetaHandler
	AppsHandler         *handlers.ApplicationsHandler
	TerminalHandler     *handlers.TerminalHandler
	PagesHandler        *handlers.PagesHandler
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
	protected.POST("/setup/hetzner", config.VPSConfigHandler.HandleSetupHetzner)
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
		// Meta/UI routes
		vps.GET("", config.VPSMetaHandler.HandleVPSManagePage)
		vps.GET("/create", config.VPSMetaHandler.HandleVPSCreatePage)
		vps.GET("/locations", config.VPSMetaHandler.HandleVPSLocations)
		vps.GET("/server-types", config.VPSMetaHandler.HandleVPSServerTypes)
		vps.GET("/server-options", config.VPSMetaHandler.HandleVPSServerOptions)
		vps.POST("/validate-name", config.VPSMetaHandler.HandleVPSValidateName)

		// Info/monitoring routes
		vps.GET("/list", config.VPSInfoHandler.HandleVPSList)
		vps.GET("/ssh-key", config.VPSInfoHandler.HandleVPSSSHKey)
		vps.GET("/:id/status", config.VPSInfoHandler.HandleVPSStatus)
		vps.GET("/:id/info", config.VPSInfoHandler.HandleVPSInfo)
		vps.GET("/:id/logs", config.VPSInfoHandler.HandleVPSLogs)
		vps.POST("/:id/terminal", config.VPSInfoHandler.HandleVPSTerminal)

		// Lifecycle routes
		vps.POST("/create", config.VPSLifecycleHandler.HandleVPSCreate)
		vps.POST("/delete", config.VPSLifecycleHandler.HandleVPSDelete)
		vps.POST("/poweroff", config.VPSLifecycleHandler.HandleVPSPowerOff)
		vps.POST("/poweron", config.VPSLifecycleHandler.HandleVPSPowerOn)
		vps.POST("/reboot", config.VPSLifecycleHandler.HandleVPSReboot)

		// Configuration routes
		vps.GET("/check-key", config.VPSConfigHandler.HandleVPSCheckKey)
		vps.POST("/validate-key", config.VPSConfigHandler.HandleVPSValidateKey)
		vps.POST("/:id/configure", config.VPSConfigHandler.HandleVPSConfigure)
		vps.POST("/:id/deploy", config.VPSConfigHandler.HandleVPSDeploy)
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
		apps.GET("/versions/:app_type", config.AppsHandler.HandleApplicationVersions)
		apps.POST("/:id/upgrade", config.AppsHandler.HandleApplicationUpgrade)
		apps.GET("/:id/password", config.AppsHandler.HandleApplicationPasswordGet)
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
