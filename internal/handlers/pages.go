package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PagesHandler contains dependencies for static page operations
type PagesHandler struct {
	// Add dependencies here as needed
}

// NewPagesHandler creates a new pages handler instance
func NewPagesHandler() *PagesHandler {
	return &PagesHandler{}
}

// HandleMainPage redirects to Svelte app dashboard
func (h *PagesHandler) HandleMainPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app")
}

// HandleDashboardPage redirects to Svelte app dashboard  
func (h *PagesHandler) HandleDashboardPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app")
}

// HandleDNSPage redirects to Svelte DNS page
func (h *PagesHandler) HandleDNSPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app/dns")
}

// HandleSetupPage redirects to Svelte setup page
func (h *PagesHandler) HandleSetupPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app/setup")
}

// HandleVPSPage redirects to Svelte VPS page
func (h *PagesHandler) HandleVPSPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app/vps")
}

// HandleApplicationsPage redirects to Svelte applications page
func (h *PagesHandler) HandleApplicationsPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app/applications")
}

// HandleVersionPage redirects to dashboard with version info modal
func (h *PagesHandler) HandleVersionPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/app?show=version")
}

// HandleTerminalPage renders a standalone terminal page for WebSocket terminals
func (h *PagesHandler) HandleTerminalPage(c *gin.Context) {
	sessionID := c.Param("session_id")
	serverName := c.Query("server")

	if sessionID == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error": "Session ID required",
		})
		return
	}

	c.HTML(http.StatusOK, "terminal.html", gin.H{
		"session_id":  sessionID,
		"server_name": serverName,
		"ActivePage":  "terminal",
	})
}
