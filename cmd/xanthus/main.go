package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"

	"github.com/chrishham/xanthus/internal/handlers"
	"github.com/chrishham/xanthus/internal/router"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	// Find available port
	port := utils.FindAvailablePort()
	if port == "" {
		log.Fatal("Could not find an available port")
	}

	fmt.Printf("🚀 Xanthus is starting on http://localhost:%s\n", port)

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Configure security
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// Setup templates
	setupTemplates(r)

	// Setup static files
	r.Static("/static", "web/static")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	dnsHandler := handlers.NewDNSHandler()
	vpsHandler := handlers.NewVPSHandler()
	appsHandler := handlers.NewApplicationsHandler()

	// Configure routes
	routeConfig := router.RouteConfig{
		AuthHandler: authHandler,
		DNSHandler:  dnsHandler,
		VPSHandler:  vpsHandler,
		AppsHandler: appsHandler,
	}

	router.SetupRoutes(r, routeConfig)

	// Start server
	log.Fatal(r.Run(":" + port))
}

// setupTemplates configures HTML templates with helper functions
func setupTemplates(r *gin.Engine) {
	r.SetFuncMap(template.FuncMap{
		"toJSON": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	})
	r.LoadHTMLGlob("web/templates/*")
}
