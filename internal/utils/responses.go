package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
}

// JSONSuccess sends a standardized JSON success response
func JSONSuccess(c *gin.Context, message string, data interface{}) {
	response := SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	c.JSON(http.StatusOK, response)
}

// JSONSuccessSimple sends a simple success response with just a message
func JSONSuccessSimple(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// JSONError sends a standardized JSON error response
func JSONError(c *gin.Context, statusCode int, message string) {
	response := ErrorResponse{
		Success: false,
		Error:   message,
		Code:    statusCode,
	}
	c.JSON(statusCode, response)
}

// JSONBadRequest sends a 400 Bad Request error response
func JSONBadRequest(c *gin.Context, message string) {
	JSONError(c, http.StatusBadRequest, message)
}

// JSONUnauthorized sends a 401 Unauthorized error response
func JSONUnauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	JSONError(c, http.StatusUnauthorized, message)
}

// JSONForbidden sends a 403 Forbidden error response
func JSONForbidden(c *gin.Context, message string) {
	JSONError(c, http.StatusForbidden, message)
}

// JSONNotFound sends a 404 Not Found error response
func JSONNotFound(c *gin.Context, message string) {
	JSONError(c, http.StatusNotFound, message)
}

// JSONInternalServerError sends a 500 Internal Server Error response
func JSONInternalServerError(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	JSONError(c, http.StatusInternalServerError, message)
}

// JSONServiceUnavailable sends a 503 Service Unavailable error response
func JSONServiceUnavailable(c *gin.Context, message string) {
	JSONError(c, http.StatusServiceUnavailable, message)
}

// JSONResponse sends a generic JSON response with custom status code
func JSONResponse(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// HTMLError sends an HTML error response (for HTMX requests)
func HTMLError(c *gin.Context, message string) {
	c.Data(http.StatusOK, "text/html", []byte("❌ "+message))
}

// HTMLSuccess sends an HTML success response (for HTMX requests)
func HTMLSuccess(c *gin.Context, message string) {
	c.Data(http.StatusOK, "text/html", []byte("✅ "+message))
}

// HTMXRedirect sends an HTMX redirect header
func HTMXRedirect(c *gin.Context, url string) {
	c.Header("HX-Redirect", url)
	c.Status(http.StatusOK)
}

// HTMXRefresh triggers an HTMX page refresh
func HTMXRefresh(c *gin.Context) {
	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

// ValidationError represents field validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// JSONValidationError sends validation error responses
func JSONValidationError(c *gin.Context, errors []ValidationError) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error":   "Validation failed",
		"errors":  errors,
	})
}

// Common response helpers for specific use cases

// VPSCreationSuccess sends a success response for VPS creation
func VPSCreationSuccess(c *gin.Context, serverName string) {
	JSONSuccessSimple(c, "Server created successfully with K3s, Helm, and ArgoCD")
}

// VPSDeletionSuccess sends a success response for VPS deletion
func VPSDeletionSuccess(c *gin.Context) {
	JSONSuccessSimple(c, "Server deleted successfully and configuration cleaned up")
}

// VPSConfigurationSuccess sends a success response for VPS configuration
func VPSConfigurationSuccess(c *gin.Context, domain string) {
	JSONSuccessSimple(c, "VPS successfully configured with SSL certificates for "+domain)
}

// ApplicationSuccess sends success responses for application operations
func ApplicationSuccess(c *gin.Context, action, appName string) {
	var message string
	switch action {
	case "create":
		message = "Application created successfully"
	case "upgrade":
		message = "Application upgraded successfully"
	case "delete":
		message = "Application deleted successfully"
	default:
		message = "Operation completed successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        message,
		"application_id": appName,
	})
}

// DNSConfigurationSuccess sends success response for DNS operations
func DNSConfigurationSuccess(c *gin.Context, domain, action string) {
	var message string
	switch action {
	case "configure":
		message = "SSL certificate configured successfully"
	case "remove":
		message = "SSL configuration removed successfully"
	default:
		message = "DNS operation completed successfully"
	}

	JSONSuccess(c, message, gin.H{"domain": domain})
}

// SetupSuccess sends success response for setup operations
func SetupSuccess(c *gin.Context, service string) {
	JSONSuccessSimple(c, service+" configuration saved successfully")
}
