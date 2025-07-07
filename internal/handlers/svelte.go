package handlers

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SvelteHandler handles SvelteKit SPA routing
type SvelteHandler struct {
	svelteFS fs.FS
}

// NewSvelteHandler creates a new SvelteKit handler
func NewSvelteHandler(svelteFS fs.FS) *SvelteHandler {
	return &SvelteHandler{
		svelteFS: svelteFS,
	}
}

// HandleSPAFallback serves the SvelteKit index.html for client-side routing
func (h *SvelteHandler) HandleSPAFallback(c *gin.Context) {
	// Get the requested path
	path := c.Request.URL.Path

	// Remove the /app prefix
	if strings.HasPrefix(path, "/app") {
		path = strings.TrimPrefix(path, "/app")
		if path == "" {
			path = "/"
		}
	}

	// If the path starts with /_app (SvelteKit assets), let the static file handler deal with it
	if strings.HasPrefix(path, "/_app") {
		c.Next()
		return
	}

	// For all other paths under /app/*, serve the SvelteKit index.html
	// This enables client-side routing
	indexFile, err := h.svelteFS.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "SvelteKit app not found")
		return
	}
	defer indexFile.Close()

	// Read the index.html content
	indexData, err := fs.ReadFile(h.svelteFS, "index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read SvelteKit app")
		return
	}

	// Serve the index.html with correct content type
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.String(http.StatusOK, string(indexData))
}
