package applications

import (
	"net/http"

	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates Cloudflare token and sets account ID in context
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("cf_token")
		if err != nil || !utils.VerifyCloudflareToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Get account ID and set in context
		_, accountID, err := utils.CheckKVNamespaceExists(token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
			c.Abort()
			return
		}

		c.Set("cf_token", token)
		c.Set("account_id", accountID)
		c.Next()
	}
}

// AuthMiddlewareHTML validates Cloudflare token for HTML pages (redirects to login)
func (h *Handler) AuthMiddlewareHTML() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("cf_token")
		if err != nil || !utils.VerifyCloudflareToken(token) {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		// Get account ID and set in context
		_, accountID, err := utils.CheckKVNamespaceExists(token)
		if err != nil {
			c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
			c.Abort()
			return
		}

		c.Set("cf_token", token)
		c.Set("account_id", accountID)
		c.Next()
	}
}

// VPSConfigMiddleware validates VPS configuration exists
func (h *Handler) VPSConfigMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetString("cf_token")
		accountID := c.GetString("account_id")
		
		if token == "" || accountID == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Missing authentication context"})
			c.Abort()
			return
		}

		// VPS validation logic would go here if needed
		// For now, just pass through
		c.Next()
	}
}

// ErrorHandlingMiddleware provides common error handling
func (h *Handler) ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle any errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			switch err.Type {
			case gin.ErrorTypePublic:
				c.JSON(c.Writer.Status(), gin.H{"error": err.Error()})
			case gin.ErrorTypeBind:
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
		}
	}
}