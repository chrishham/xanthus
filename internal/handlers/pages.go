package handlers

import (
	"net/http"

	"github.com/chrishham/xanthus/internal/utils"
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

// HandleMainPage renders the main application page
func (h *PagesHandler) HandleMainPage(c *gin.Context) {
	// Check Hetzner API status
	hetznerStatus := "Not configured"
	token := c.GetString("cf_token")

	if token != "" {
		// Get account ID to check for existing Hetzner key
		if exists, accountID, err := utils.CheckKVNamespaceExists(token); err == nil && exists {
			if hetznerKey, err := utils.GetHetznerAPIKey(token, accountID); err == nil && hetznerKey != "" {
				// Test the API key
				if utils.ValidateHetznerAPIKey(hetznerKey) {
					hetznerStatus = "Connected"
				} else {
					hetznerStatus = "Invalid key"
				}
			}
		}
	}

	c.HTML(http.StatusOK, "main.html", gin.H{
		"ActivePage":    "main",
		"HetznerStatus": hetznerStatus,
	})
}

// HandleSetupPage renders the setup page with existing Hetzner key info
func (h *PagesHandler) HandleSetupPage(c *gin.Context) {
	// Try to get existing Hetzner API key for prepopulation
	var existingKey string
	token := c.GetString("cf_token")

	// Get account and check for existing key
	exists, accountID, err := utils.CheckKVNamespaceExists(token)
	if err == nil && exists {
		// If we can get the account ID, try to retrieve the existing key
		if hetznerKey, err := utils.GetHetznerAPIKey(token, accountID); err == nil {
			// Mask the key for security (show only first 4 and last 4 characters)
			if len(hetznerKey) > 8 {
				existingKey = hetznerKey[:4] + "..." + hetznerKey[len(hetznerKey)-4:]
			}
		}
	}

	c.HTML(http.StatusOK, "setup.html", gin.H{
		"existing_key": existingKey,
	})
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
