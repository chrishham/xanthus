package middleware

import (
	"net/http"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// Global cache service instance
var cacheService = services.NewCacheService()

// AuthMiddleware validates Cloudflare token from cookies
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("cf_token")
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		// Check cache first for account info
		accountInfo, cached := cacheService.GetAccountInfo(token)
		if cached {
			// Use cached account info
			c.Set("cf_token", token)
			c.Set("account_id", accountInfo.AccountID)
			c.Set("namespace_id", accountInfo.NamespaceID)
			c.Next()
			return
		}

		// Cache miss - verify token and get account info
		if !utils.VerifyCloudflareToken(token) {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		// Get account ID and namespace info
		namespaceExists, accountID, err := utils.CheckKVNamespaceExists(token)
		if err != nil {
			c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
			c.Abort()
			return
		}

		// Get namespace ID for caching
		namespaceID := ""
		if namespaceExists {
			namespaceID, _ = utils.GetXanthusNamespaceID(&http.Client{}, token, accountID)
		}

		// Cache account info for 10 minutes
		cacheService.SetAccountInfo(token, &services.AccountInfo{
			AccountID:   accountID,
			NamespaceID: namespaceID,
		}, 10*time.Minute)

		// Store in context for handlers
		c.Set("cf_token", token)
		c.Set("account_id", accountID)
		c.Set("namespace_id", namespaceID)
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

		// Check cache first for account info
		accountInfo, cached := cacheService.GetAccountInfo(token)
		if cached {
			// Use cached account info
			c.Set("cf_token", token)
			c.Set("account_id", accountInfo.AccountID)
			c.Set("namespace_id", accountInfo.NamespaceID)
			c.Next()
			return
		}

		// Cache miss - verify token and get account info
		if !utils.VerifyCloudflareToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
			c.Abort()
			return
		}

		// Get account ID and namespace info
		namespaceExists, accountID, err := utils.CheckKVNamespaceExists(token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access account"})
			c.Abort()
			return
		}

		// Get namespace ID for caching
		namespaceID := ""
		if namespaceExists {
			namespaceID, _ = utils.GetXanthusNamespaceID(&http.Client{}, token, accountID)
		}

		// Cache account info for 10 minutes
		cacheService.SetAccountInfo(token, &services.AccountInfo{
			AccountID:   accountID,
			NamespaceID: namespaceID,
		}, 10*time.Minute)

		// Store in context for handlers
		c.Set("cf_token", token)
		c.Set("account_id", accountID)
		c.Set("namespace_id", namespaceID)
		c.Next()
	}
}
