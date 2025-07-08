package router

import (
	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/chrishham/xanthus/internal/handlers/applications"
	"github.com/chrishham/xanthus/internal/handlers/vps"
	"github.com/chrishham/xanthus/internal/middleware"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/gin-gonic/gin"
)

// RouteConfig holds all the handler instances
type RouteConfig struct {
	AuthHandler              *handlers.AuthHandler
	DNSHandler               *handlers.DNSHandler
	VPSLifecycleHandler      *vps.VPSLifecycleHandler
	VPSInfoHandler           *vps.VPSInfoHandler
	VPSConfigHandler         *vps.VPSConfigHandler
	VPSMetaHandler           *vps.VPSMetaHandler
	AppsHandler              *applications.Handler
	TerminalHandler          *handlers.TerminalHandler
	WebSocketTerminalHandler *handlers.WebSocketTerminalHandler
	PagesHandler             *handlers.PagesHandler
	VersionHandler           *handlers.VersionHandler
	SvelteHandler            *handlers.SvelteHandler
	JWTService               *services.JWTService
}

// SetupRoutes configures all application routes
func SetupRoutes(r *gin.Engine, config RouteConfig) {
	setupPublicRoutes(r, config)
	setupProtectedRoutes(r, config)
	setupAPIRoutes(r, config)
	
	// Global SPA catch-all for any unmatched routes
	if config.SvelteHandler != nil {
		r.NoRoute(config.SvelteHandler.HandleSPAFallback)
	}
}

// setupPublicRoutes configures routes that don't require authentication
func setupPublicRoutes(r *gin.Engine, config RouteConfig) {
	// Authentication routes
	r.POST("/login", config.AuthHandler.HandleLogin)
	r.GET("/health", config.AuthHandler.HandleHealth)
	
	// Serve login page and assets via Svelte handler for unauthenticated users
	if config.SvelteHandler != nil {
		r.GET("/", config.SvelteHandler.HandleSPAFallback)
		r.GET("/login", config.SvelteHandler.HandleSPAFallback)  
		r.GET("/login/*path", config.SvelteHandler.HandleSPAFallback)
		r.GET("/_app/*path", config.SvelteHandler.HandleSPAFallback)
	}
}

// setupProtectedRoutes configures routes that require authentication
func setupProtectedRoutes(r *gin.Engine, config RouteConfig) {
	// Apply authentication middleware to protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())

	// Legacy page redirects to SvelteKit frontend
	protected.GET("/main", config.PagesHandler.HandleMainPage)
	protected.GET("/dashboard", config.PagesHandler.HandleDashboardPage)
	protected.GET("/setup", config.PagesHandler.HandleSetupPage)
	protected.GET("/dns", config.PagesHandler.HandleDNSPage)
	protected.GET("/vps", config.PagesHandler.HandleVPSPage)
	protected.GET("/applications", config.PagesHandler.HandleApplicationsPage)
	protected.GET("/version", config.PagesHandler.HandleVersionPage)
	protected.GET("/about", config.VersionHandler.GetAboutInfo)
	protected.GET("/logout", config.AuthHandler.HandleLogout)

	// Legacy terminal page (will be migrated to SvelteKit)
	protected.GET("/terminal-page/:session_id", config.TerminalHandler.HandleTerminalPage)

	// WebSocket endpoint (with JWT auth)
	ws := r.Group("/ws")
	if config.JWTService != nil {
		ws.Use(middleware.JWTWebSocketAuthMiddleware(config.JWTService))
	}
	{
		ws.GET("/terminal/:session_id", config.WebSocketTerminalHandler.HandleWebSocketTerminal)
	}

	// SvelteKit SPA routes - all frontend routes under /app prefix
	if config.SvelteHandler != nil {
		// Handle all /app routes as Svelte territory
		protected.GET("/app", config.SvelteHandler.HandleSPAFallback)
		protected.GET("/app/*path", config.SvelteHandler.HandleSPAFallback)
	}

	// Note: NoRoute is handled at the global level, not on router groups
}

// setupAPIRoutes configures API routes with appropriate middleware
func setupAPIRoutes(r *gin.Engine, config RouteConfig) {
	// Public API routes (no authentication required)
	publicAPI := r.Group("/api")

	// Authentication endpoints
	auth := publicAPI.Group("/auth")
	auth.POST("/login", config.AuthHandler.HandleAPILogin)
	auth.POST("/refresh", config.AuthHandler.HandleAPIRefreshToken)
	auth.POST("/logout", config.AuthHandler.HandleAPILogout)

	// Protected API routes (JWT authentication required)
	protectedAPI := r.Group("/api")
	if config.JWTService != nil {
		protectedAPI.Use(middleware.JWTAuthMiddleware(config.JWTService))
	}

	// User endpoints
	user := protectedAPI.Group("/user")
	user.GET("/profile", config.AuthHandler.HandleAPIAuthStatus)

	// Applications API endpoints
	apps := protectedAPI.Group("/applications")
	apps.GET("", config.AppsHandler.HandleApplicationsList)
	apps.GET("/prerequisites", config.AppsHandler.HandleApplicationsPrerequisites)
	apps.POST("", config.AppsHandler.HandleApplicationsCreate)
	apps.GET("/versions/:app_type", config.AppsHandler.HandleApplicationVersions)
	apps.POST("/:id/upgrade", config.AppsHandler.HandleApplicationUpgrade)
	apps.GET("/:id/password", config.AppsHandler.HandleApplicationPasswordGet)
	apps.POST("/:id/password", config.AppsHandler.HandleApplicationPasswordChange)
	apps.GET("/:id/token", config.AppsHandler.HandleApplicationToken)
	apps.GET("/:id/port-forwards", config.AppsHandler.HandlePortForwardsList)
	apps.POST("/:id/port-forwards", config.AppsHandler.HandlePortForwardsCreate)
	apps.DELETE("/:id/port-forwards/:port_id", config.AppsHandler.HandlePortForwardsDelete)
	apps.DELETE("/:id", config.AppsHandler.HandleApplicationDelete)

	// VPS API endpoints
	vps := protectedAPI.Group("/vps")
	{
		// Meta/UI endpoints
		vps.GET("/locations", config.VPSMetaHandler.HandleVPSLocations)
		vps.GET("/server-types", config.VPSMetaHandler.HandleVPSServerTypes)
		vps.GET("/server-options", config.VPSMetaHandler.HandleVPSServerOptions)
		vps.POST("/validate-name", config.VPSMetaHandler.HandleVPSValidateName)

		// Info/monitoring endpoints
		vps.GET("", config.VPSInfoHandler.HandleVPSList)
		vps.GET("/ssh-key", config.VPSInfoHandler.HandleVPSSSHKey)
		vps.GET("/:id/status", config.VPSInfoHandler.HandleVPSStatus)
		vps.GET("/:id/info", config.VPSInfoHandler.HandleVPSInfo)
		vps.GET("/:id/logs", config.VPSInfoHandler.HandleVPSLogs)
		vps.GET("/:id/k3s-logs", config.VPSInfoHandler.HandleK3sLogs)
		vps.GET("/:id/applications", config.VPSInfoHandler.HandleVPSApplications)
		vps.POST("/:id/terminal", config.VPSInfoHandler.HandleVPSTerminal)
		vps.GET("/:id/ssh-debug", config.VPSInfoHandler.HandleVPSSSHUserDebug)

		// Lifecycle endpoints
		vps.POST("", config.VPSLifecycleHandler.HandleVPSCreate)
		vps.POST("/delete", config.VPSLifecycleHandler.HandleVPSDelete)
		vps.POST("/poweroff", config.VPSLifecycleHandler.HandleVPSPowerOff)
		vps.POST("/poweron", config.VPSLifecycleHandler.HandleVPSPowerOn)
		vps.POST("/reboot", config.VPSLifecycleHandler.HandleVPSReboot)

		// Provider-specific endpoints
		vps.GET("/oci-ssh-key", config.VPSLifecycleHandler.HandleSSHKey)
		vps.POST("/add-oci", config.VPSLifecycleHandler.HandleAddOCI)

		// OCI automation endpoints
		oci := vps.Group("/oci")
		{
			oci.GET("/check-token", config.VPSLifecycleHandler.HandleOCICheckToken)
			oci.POST("/validate-token", config.VPSLifecycleHandler.HandleOCIValidateToken)
			oci.POST("/store-token", config.VPSLifecycleHandler.HandleOCIStoreToken)
			oci.GET("/home-region", config.VPSLifecycleHandler.HandleOCIGetHomeRegion)
			oci.POST("/create", config.VPSLifecycleHandler.HandleOCICreate)
			oci.POST("/delete", config.VPSLifecycleHandler.HandleOCIDelete)
			oci.POST("/poweroff", config.VPSLifecycleHandler.HandleOCIPowerOff)
			oci.POST("/poweron", config.VPSLifecycleHandler.HandleOCIPowerOn)
			oci.POST("/reboot", config.VPSLifecycleHandler.HandleOCIReboot)
		}

		// Configuration endpoints
		vps.GET("/check-key", config.VPSConfigHandler.HandleVPSCheckKey)
		vps.POST("/validate-key", config.VPSConfigHandler.HandleVPSValidateKey)
		vps.POST("/:id/configure", config.VPSConfigHandler.HandleVPSConfigure)
		vps.POST("/:id/deploy", config.VPSConfigHandler.HandleVPSDeploy)

		// Timezone endpoints
		vps.GET("/timezones", config.VPSConfigHandler.HandleVPSListTimezones)
		vps.GET("/:id/timezone", config.VPSConfigHandler.HandleVPSGetTimezone)
		vps.POST("/:id/timezone", config.VPSConfigHandler.HandleVPSSetTimezone)

		// Configuration update endpoint
		vps.PATCH("/:id/config", config.VPSLifecycleHandler.HandleUpdateVPSConfig)
	}

	// DNS API endpoints
	dns := protectedAPI.Group("/dns")
	{
		dns.GET("", config.DNSHandler.HandleDNSListAPI)
		dns.POST("/configure", config.DNSHandler.HandleDNSConfigureAPI)
		dns.POST("/remove", config.DNSHandler.HandleDNSRemoveAPI)
		dns.GET("/config/:domain", config.DNSHandler.HandleDNSConfigGetAPI)
	}

	// Setup API endpoints
	setup := protectedAPI.Group("/setup")
	{
		setup.GET("/status", config.VPSConfigHandler.HandleSetupStatusAPI)
		setup.POST("/hetzner", config.VPSConfigHandler.HandleSetupHetznerAPI)
		// Note: Other setup endpoints can reuse existing VPS API endpoints:
		// - /api/vps/locations for server locations
		// - /api/vps/server-types for server types
		// - /api/vps/server-options for server options
	}


	// Additional protected API routes will be added here in later phases
	
	// About endpoint
	protectedAPI.GET("/about", config.VersionHandler.GetAboutInfo)
}
