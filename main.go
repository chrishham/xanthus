package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/chrishham/xanthus/internal/handlers/applications"
	"github.com/chrishham/xanthus/internal/handlers/vps"
	"github.com/chrishham/xanthus/internal/router"
	"github.com/chrishham/xanthus/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
	// Set port to 8081
	port := "8081"

	// Get version info for startup logging
	version := getVersion()
	goVersion := runtime.Version()
	platform := runtime.GOOS + "/" + runtime.GOARCH

	fmt.Printf("🚀 Xanthus %s is starting on http://localhost:%s\n", version, port)
	fmt.Printf("📊 Platform: %s | Go: %s\n", platform, goVersion)

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Configure security
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// Setup templates with cache busting
	setupTemplates(r)

	// Setup static files from embedded filesystem
	staticFS, err := fs.Sub(StaticFiles, "web/static")
	if err != nil {
		log.Fatal("Failed to create static files sub-filesystem:", err)
	}
	r.StaticFS("/static", http.FS(staticFS))

	// Setup SvelteKit files from embedded filesystem
	svelteFS, err := fs.Sub(SvelteFiles, "svelte-app/build")
	if err != nil {
		log.Fatal("Failed to create SvelteKit files sub-filesystem:", err)
	}
	// Note: SvelteHandler will handle /app routes including static files

	// Initialize shared services
	wsTerminalService := services.NewWebSocketTerminalService()

	// Initialize JWT service with 32-byte secret key
	jwtSecretKey, err := services.GenerateSecretKey()
	if err != nil {
		log.Fatal("Failed to generate JWT secret key:", err)
	}
	jwtService := services.NewJWTService(jwtSecretKey, 15*time.Minute, 7*24*time.Hour)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(jwtService)
	dnsHandler := handlers.NewDNSHandler()
	vpsLifecycleHandler := vps.NewVPSLifecycleHandler()
	vpsInfoHandler := vps.NewVPSInfoHandler()
	vpsConfigHandler := vps.NewVPSConfigHandler()
	vpsMetaHandler := vps.NewVPSMetaHandler()
	appsHandler := applications.NewHandlerWithEmbedFS(&AllApplicationFiles)
	terminalHandler := handlers.NewTerminalHandlerWithService(wsTerminalService)
	webSocketTerminalHandler := handlers.NewWebSocketTerminalHandlerWithService(wsTerminalService)
	pagesHandler := handlers.NewPagesHandler()
	versionHandler := handlers.NewVersionHandler()
	svelteHandler := handlers.NewSvelteHandler(svelteFS)

	// Configure routes
	routeConfig := router.RouteConfig{
		AuthHandler:              authHandler,
		DNSHandler:               dnsHandler,
		VPSLifecycleHandler:      vpsLifecycleHandler,
		VPSInfoHandler:           vpsInfoHandler,
		VPSConfigHandler:         vpsConfigHandler,
		VPSMetaHandler:           vpsMetaHandler,
		AppsHandler:              appsHandler,
		TerminalHandler:          terminalHandler,
		WebSocketTerminalHandler: webSocketTerminalHandler,
		PagesHandler:             pagesHandler,
		VersionHandler:           versionHandler,
		SvelteHandler:            svelteHandler,
		JWTService:               jwtService,
	}

	router.SetupRoutes(r, routeConfig)

	// Start server
	log.Fatal(r.Run(":" + port))
}

// setupTemplates configures HTML templates with helper functions
func setupTemplates(r *gin.Engine) {
	// Generate cache busting timestamp
	cacheBuster := strconv.FormatInt(time.Now().Unix(), 10)

	funcMap := template.FuncMap{
		"toJSON": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				panic("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					panic("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
		"cacheBuster": func() string {
			return cacheBuster
		},
	}

	// Pre-compile templates once at startup for better performance using embedded files
	log.Println("📋 Pre-compiling HTML templates from embedded files...")
	tmpl := template.New("").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseFS(HTMLTemplates, "web/templates/*.html"))
	tmpl = template.Must(tmpl.ParseFS(HTMLTemplates, "web/templates/partials/*/*.html"))
	r.SetHTMLTemplate(tmpl)
	log.Println("✅ Templates pre-compiled successfully from embedded files")
}

// getVersion returns the current version from environment or default
func getVersion() string {
	if version := os.Getenv("XANTHUS_VERSION"); version != "" {
		return "v" + version
	}
	return "v2.0-dev"
}
