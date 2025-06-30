package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketTerminalHandler handles WebSocket terminal connections
type WebSocketTerminalHandler struct {
	terminalService *services.WebSocketTerminalService
	upgrader        websocket.Upgrader
}

// NewWebSocketTerminalHandler creates a new WebSocket terminal handler
func NewWebSocketTerminalHandler() *WebSocketTerminalHandler {
	return &WebSocketTerminalHandler{
		terminalService: services.NewWebSocketTerminalService(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from same origin
				return true
			},
		},
	}
}

// NewWebSocketTerminalHandlerWithService creates a new WebSocket terminal handler with shared service
func NewWebSocketTerminalHandlerWithService(wsService *services.WebSocketTerminalService) *WebSocketTerminalHandler {
	return &WebSocketTerminalHandler{
		terminalService: wsService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from same origin
				return true
			},
		},
	}
}

// HandleWebSocketTerminal handles WebSocket terminal connections
func (h *WebSocketTerminalHandler) HandleWebSocketTerminal(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	// Authenticate WebSocket connection
	token := h.authenticateWebSocket(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}
	defer conn.Close()

	// Validate session ownership
	session, err := h.terminalService.GetSession(sessionID)
	if err != nil {
		h.sendErrorMessage(conn, "Session not found")
		return
	}

	// Ensure user owns the session (using account ID for validation)
	if session.AccountID != accountID {
		h.sendErrorMessage(conn, "Unauthorized session access")
		return
	}

	// Handle WebSocket terminal session
	if err := h.terminalService.HandleWebSocketConnection(sessionID, conn); err != nil {
		log.Printf("WebSocket terminal session error: %v", err)
		h.sendErrorMessage(conn, "Terminal session error")
	}
}

// authenticateWebSocket authenticates WebSocket connections
func (h *WebSocketTerminalHandler) authenticateWebSocket(c *gin.Context) string {
	// Try to get token from multiple sources

	// 1. Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]
			if utils.VerifyCloudflareToken(token) {
				return token
			}
		}
	}

	// 2. Try query parameter (for WebSocket connections from frontend)
	token := c.Query("token")
	if token != "" && utils.VerifyCloudflareToken(token) {
		return token
	}

	// 3. Try cookie (fallback for same-origin requests)
	token, err := c.Cookie("cf_token")
	if err == nil && utils.VerifyCloudflareToken(token) {
		return token
	}

	return ""
}

// sendErrorMessage sends an error message over WebSocket
func (h *WebSocketTerminalHandler) sendErrorMessage(conn *websocket.Conn, message string) {
	errorMsg := map[string]string{
		"type":    "error",
		"message": message,
	}
	if data, err := json.Marshal(errorMsg); err == nil {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

// HandleTerminalCreate creates a new WebSocket terminal session
func (h *WebSocketTerminalHandler) HandleTerminalCreate(c *gin.Context) {
	// Get authentication context (set by middleware)
	token, exists := c.Get("cf_token")
	if !exists {
		utils.JSONUnauthorized(c, "Authentication token not found")
		return
	}

	accountID, exists := c.Get("account_id")
	if !exists {
		utils.JSONError(c, http.StatusInternalServerError, "Account ID not found")
		return
	}

	// Parse request
	var req struct {
		ServerID   int    `json:"server_id" binding:"required"`
		Host       string `json:"host" binding:"required"`
		User       string `json:"user" binding:"required"`
		PrivateKey string `json:"private_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Create terminal session
	session, err := h.terminalService.CreateSession(
		req.ServerID,
		req.Host,
		req.User,
		req.PrivateKey,
		token.(string),
		accountID.(string),
	)
	if err != nil {
		utils.JSONError(c, http.StatusInternalServerError, "Failed to create terminal session: "+err.Error())
		return
	}

	// Return session info for WebSocket connection
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"session_id":    session.ID,
		"websocket_url": fmt.Sprintf("/ws/terminal/%s", session.ID),
		"status":        session.Status,
		"server_id":     session.ServerID,
		"host":          session.Host,
	})
}

// HandleTerminalList lists active terminal sessions for the authenticated user
func (h *WebSocketTerminalHandler) HandleTerminalList(c *gin.Context) {
	accountID, exists := c.Get("account_id")
	if !exists {
		utils.JSONError(c, http.StatusInternalServerError, "Account ID not found")
		return
	}

	sessions := h.terminalService.ListSessionsForAccount(accountID.(string))
	utils.JSONResponse(c, http.StatusOK, gin.H{
		"sessions": sessions,
	})
}

// HandleTerminalStop stops a WebSocket terminal session
func (h *WebSocketTerminalHandler) HandleTerminalStop(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.JSONError(c, http.StatusBadRequest, "Session ID required")
		return
	}

	accountID, exists := c.Get("account_id")
	if !exists {
		utils.JSONError(c, http.StatusInternalServerError, "Account ID not found")
		return
	}

	// Validate session ownership before stopping
	session, err := h.terminalService.GetSession(sessionID)
	if err != nil {
		utils.JSONNotFound(c, "Terminal session not found")
		return
	}

	if session.AccountID != accountID.(string) {
		utils.JSONError(c, http.StatusForbidden, "Unauthorized session access")
		return
	}

	// Stop the session
	if err := h.terminalService.StopSession(sessionID); err != nil {
		utils.JSONError(c, http.StatusInternalServerError, "Failed to stop session: "+err.Error())
		return
	}

	utils.JSONResponse(c, http.StatusOK, gin.H{
		"success": true,
		"message": "Terminal session stopped",
	})
}
