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
	terminalService *services.TerminalService
}

// NewTerminalHandler creates a new terminal handler instance
func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{
		terminalService: services.NewTerminalService(),
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
