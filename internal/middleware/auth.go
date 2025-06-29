package middleware

import (
	"net/http"

	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates Cloudflare token from cookies
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("cf_token")
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		if !utils.VerifyCloudflareToken(token) {
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

		// Store token and account ID in context for handlers to use
		c.Set("cf_token", token)
		c.Set("account_id", accountID)
		c.Next()
	}
}

// APIAuthMiddleware validates Cloudflare token for API endpoints
func APIAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("cf_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !utils.VerifyCloudflareToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
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

		// Store token and account ID in context for handlers to use
		c.Set("cf_token", token)
		c.Set("account_id", accountID)
		c.Next()
	}
}
