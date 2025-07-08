package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthHandler contains dependencies for auth-related operations
type AuthHandler struct {
	jwtService *services.JWTService
}

// NewAuthHandler creates a new auth handler instance
func NewAuthHandler(jwtService *services.JWTService) *AuthHandler {
	return &AuthHandler{
		jwtService: jwtService,
	}
}


// HandleLogin processes login with Cloudflare token
func (h *AuthHandler) HandleLogin(c *gin.Context) {
	token := c.PostForm("cf_token")
	if token == "" {
		c.Data(http.StatusBadRequest, "text/html", []byte("API token is required"))
		return
	}

	if utils.VerifyCloudflareToken(token) {
		// Check if Xanthus KV namespace exists, create if not
		exists, accountID, err := utils.CheckKVNamespaceExists(token)
		if err != nil {
			log.Printf("Error checking KV namespace: %v", err)
			c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error checking KV namespace: %s", err.Error())))
			return
		}

		if !exists {
			if err := utils.CreateKVNamespace(token, accountID); err != nil {
				log.Printf("Error creating KV namespace: %v", err)
				c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error creating KV namespace: %s", err.Error())))
				return
			}
		} else {
			log.Println("✅ Xanthus KV namespace already exists")
		}

		// Check and create CSR if not exists
		client := &http.Client{Timeout: 10 * time.Second}
		var existingCSR map[string]interface{}
		if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &existingCSR); err != nil {
			log.Println("🔧 Generating new CSR for SSL certificates")

			cfService := services.NewCloudflareService()
			csrConfig, err := cfService.GenerateCSR()
			if err != nil {
				log.Printf("Error generating CSR: %v", err)
				c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error generating CSR: %s", err.Error())))
				return
			}

			// Store CSR in KV
			if err := utils.PutKVValue(client, token, accountID, "config:ssl:csr", csrConfig); err != nil {
				log.Printf("Error storing CSR: %v", err)
				c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error storing CSR: %s", err.Error())))
				return
			}

			log.Println("✅ CSR generated and stored successfully")
		} else {
			log.Println("✅ CSR already exists in KV")
		}

		// Valid token - proceed to main page (24 hours = 86400 seconds)
		c.SetCookie("cf_token", token, 86400, "/", "", false, true)
		c.Redirect(http.StatusTemporaryRedirect, "/")
	} else {
		c.Data(http.StatusOK, "text/html", []byte("❌ Invalid Cloudflare API token. Please check your token and try again."))
	}
}

// HandleLogout clears authentication and redirects to login
func (h *AuthHandler) HandleLogout(c *gin.Context) {
	c.SetCookie("cf_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

// HandleHealth returns health status
func (h *AuthHandler) HandleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}

// HandleAPILogin processes login with Cloudflare token and returns JWT tokens
func (h *AuthHandler) HandleAPILogin(c *gin.Context) {
	var req struct {
		CFToken string `json:"cf_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !utils.VerifyCloudflareToken(req.CFToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Cloudflare API token"})
		return
	}

	// Check if Xanthus KV namespace exists, create if not
	exists, accountID, err := utils.CheckKVNamespaceExists(req.CFToken)
	if err != nil {
		log.Printf("Error checking KV namespace: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check KV namespace"})
		return
	}

	if !exists {
		if err := utils.CreateKVNamespace(req.CFToken, accountID); err != nil {
			log.Printf("Error creating KV namespace: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create KV namespace"})
			return
		}
	}

	// Get namespace ID for JWT claims
	namespaceID := ""
	if exists {
		namespaceID, _ = utils.GetXanthusNamespaceID(&http.Client{}, req.CFToken, accountID)
	}

	// Check and create CSR if not exists
	client := &http.Client{Timeout: 10 * time.Second}
	var existingCSR map[string]interface{}
	if err := utils.GetKVValue(client, req.CFToken, accountID, "config:ssl:csr", &existingCSR); err != nil {
		log.Println("Generating new CSR for SSL certificates")

		cfService := services.NewCloudflareService()
		csrConfig, err := cfService.GenerateCSR()
		if err != nil {
			log.Printf("Error generating CSR: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSR"})
			return
		}

		// Store CSR in KV
		if err := utils.PutKVValue(client, req.CFToken, accountID, "config:ssl:csr", csrConfig); err != nil {
			log.Printf("Error storing CSR: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store CSR"})
			return
		}
	}

	// Generate JWT tokens
	userID := accountID // Use account ID as user ID for now
	accessToken, refreshToken, err := h.jwtService.GenerateTokenPair(userID, accountID, namespaceID, req.CFToken)
	if err != nil {
		log.Printf("Error generating JWT tokens: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900, // 15 minutes
	})
}

// HandleAPILogout invalidates the current JWT token
func (h *AuthHandler) HandleAPILogout(c *gin.Context) {
	// For now, just return success - in a production system, you'd add the token to a blacklist
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// HandleAPIRefreshToken generates new tokens using a refresh token
func (h *AuthHandler) HandleAPIRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	newAccessToken, newRefreshToken, err := h.jwtService.RefreshToken(req.RefreshToken)
	if err != nil {
		if err == services.ErrExpiredToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has expired"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"token_type":    "Bearer",
		"expires_in":    900, // 15 minutes
	})
}

// HandleAPIAuthStatus returns current authentication status
func (h *AuthHandler) HandleAPIAuthStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	accountID, _ := c.Get("account_id")
	namespaceID, _ := c.Get("namespace_id")

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"user": gin.H{
			"id":           userID,
			"account_id":   accountID,
			"namespace_id": namespaceID,
		},
	})
}
