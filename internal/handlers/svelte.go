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

	// If the path starts with /app/_app (SvelteKit assets), serve the static file directly
	if strings.HasPrefix(path, "/app/_app") {
		// Remove the /app prefix and leading slash for filesystem access
		filePath := strings.TrimPrefix(path, "/app/")
		
		// Try to open the file
		file, err := h.svelteFS.Open(filePath)
		if err != nil {
			c.String(http.StatusNotFound, "File not found")
			return
		}
		defer file.Close()

		// Read the file content
		data, err := fs.ReadFile(h.svelteFS, filePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to read file")
			return
		}

		// Set appropriate content type based on file extension
		contentType := "text/plain"
		if strings.HasSuffix(filePath, ".js") {
			contentType = "application/javascript"
		} else if strings.HasSuffix(filePath, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(filePath, ".json") {
			contentType = "application/json"
		}

		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// For all other paths, serve the SvelteKit index.html
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
