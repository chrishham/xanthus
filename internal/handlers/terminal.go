package handlers

import (
	"fmt"
	"net/http"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// TerminalHandler contains dependencies for terminal-related operations
type TerminalHandler struct {
	terminalService   *services.TerminalService
	wsTerminalService *services.WebSocketTerminalService
}

// NewTerminalHandler creates a new terminal handler instance
func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{
		terminalService:   services.NewTerminalService(),
		wsTerminalService: services.NewWebSocketTerminalService(),
	}
}

// NewTerminalHandlerWithService creates a new terminal handler with shared WebSocket service
func NewTerminalHandlerWithService(wsService *services.WebSocketTerminalService) *TerminalHandler {
	return &TerminalHandler{
		terminalService:   services.NewTerminalService(),
		wsTerminalService: wsService,
	}
}

// HandleTerminalView gets terminal session details
func (h *TerminalHandler) HandleTerminalView(c *gin.Context) {
	sessionID := c.Param("session_id")

	session, err := h.terminalService.GetSession(sessionID)
	if err != nil {
		utils.JSONNotFound(c, "Terminal session not found")
		return
	}

	// Return terminal proxy URL
	terminalURL := fmt.Sprintf("http://localhost:%d", session.Port)
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"session_id":   session.ID,
		"terminal_url": terminalURL,
		"status":       session.Status,
		"server_id":    session.ServerID,
		"host":         session.Host,
	})
}

// HandleTerminalStop stops an active terminal session
func (h *TerminalHandler) HandleTerminalStop(c *gin.Context) {
	sessionID := c.Param("session_id")

	if err := h.terminalService.StopSession(sessionID); err != nil {
		utils.JSONNotFound(c, "Terminal session not found")
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"success": true,
		"message": "Terminal session stopped",
	})
}

// HandleTerminalPage renders the standalone terminal page
func (h *TerminalHandler) HandleTerminalPage(c *gin.Context) {
	sessionID := c.Param("session_id")

	// Try WebSocket terminal service first (new system)
	session, err := h.wsTerminalService.GetSession(sessionID)
	if err != nil {
		// Fallback to legacy terminal service for backward compatibility
		legacySession, legacyErr := h.terminalService.GetSession(sessionID)
		if legacyErr != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{
				"error":   "Terminal session not found",
				"message": "The requested terminal session does not exist or has expired.",
			})
			return
		}

		// Use legacy session data
		c.HTML(http.StatusOK, "terminal.html", gin.H{
			"SessionID":  legacySession.ID,
			"ServerName": legacySession.Host,
			"Title":      fmt.Sprintf("Terminal - %s", legacySession.Host),
		})
		return
	}

	// Use WebSocket session data
	c.HTML(http.StatusOK, "terminal.html", gin.H{
		"SessionID":  session.ID,
		"ServerName": session.Host,
		"Title":      fmt.Sprintf("Terminal - %s", session.Host),
	})
}
