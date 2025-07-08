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

// HandleMainPage redirects to Svelte app
func (h *PagesHandler) HandleMainPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

// HandleSetupPage redirects to Svelte setup page
func (h *PagesHandler) HandleSetupPage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/setup")
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
