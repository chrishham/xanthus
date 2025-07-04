package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"

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

	fmt.Printf("🚀 Xanthus v2.0 (with shared ConfigMap support) is starting on http://localhost:%s\n", port)

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Configure security
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// Setup templates
	setupTemplates(r)

	// Setup static files
	r.Static("/static", "web/static")

	// Initialize shared services
	wsTerminalService := services.NewWebSocketTerminalService()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	dnsHandler := handlers.NewDNSHandler()
	vpsLifecycleHandler := vps.NewVPSLifecycleHandler()
	vpsInfoHandler := vps.NewVPSInfoHandler()
	vpsConfigHandler := vps.NewVPSConfigHandler()
	vpsMetaHandler := vps.NewVPSMetaHandler()
	appsHandler := applications.NewHandler()
	terminalHandler := handlers.NewTerminalHandlerWithService(wsTerminalService)
	webSocketTerminalHandler := handlers.NewWebSocketTerminalHandlerWithService(wsTerminalService)
	pagesHandler := handlers.NewPagesHandler()

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
	}

	router.SetupRoutes(r, routeConfig)

	// Start server
	log.Fatal(r.Run(":" + port))
}

// setupTemplates configures HTML templates with helper functions
func setupTemplates(r *gin.Engine) {
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
	}

	// Load main templates and partials
	tmpl := template.New("").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseGlob("web/templates/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/partials/*/*.html"))
	r.SetHTMLTemplate(tmpl)
}
