package middleware

import (
	"net/http"
	"strings"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware validates JWT tokens for API endpoints
func JWTAuthMiddleware(jwtService *services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			if err == services.ErrExpiredToken {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		// Store claims in context for handlers
		c.Set("user_id", claims.UserID)
		c.Set("account_id", claims.AccountID)
		c.Set("namespace_id", claims.NamespaceID)
		c.Set("cf_token", claims.CFToken)
		c.Next()
	}
}

// JWTAuthMiddlewareHTML validates JWT tokens for HTML pages and redirects on failure
func JWTAuthMiddlewareHTML(jwtService *services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Fallback to cookie for backward compatibility during migration
			token, err := c.Cookie("jwt_token")
			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, "/login")
				c.Abort()
				return
			}
			tokenString = token
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/login")
			c.Abort()
			return
		}

		// Store claims in context for handlers
		c.Set("user_id", claims.UserID)
		c.Set("account_id", claims.AccountID)
		c.Set("namespace_id", claims.NamespaceID)
		c.Set("cf_token", claims.CFToken)
		c.Next()
	}
}

// JWTWebSocketAuthMiddleware validates JWT tokens for WebSocket connections
func JWTWebSocketAuthMiddleware(jwtService *services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// For WebSocket, token can be passed as query parameter
		tokenString := c.Query("token")
		if tokenString == "" {
			// Try Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
			c.Abort()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Store claims in context for handlers
		c.Set("user_id", claims.UserID)
		c.Set("account_id", claims.AccountID)
		c.Set("namespace_id", claims.NamespaceID)
		c.Set("cf_token", claims.CFToken)
		c.Next()
	}
}
