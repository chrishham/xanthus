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
	// Add dependencies here as needed
}

// NewAuthHandler creates a new auth handler instance
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// HandleRoot redirects to login page
func (h *AuthHandler) HandleRoot(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

// HandleLoginPage renders the login page
func (h *AuthHandler) HandleLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
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
		if err := utils.GetKVValue(token, accountID, "config:ssl:csr", &existingCSR); err != nil {
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

		// Valid token - proceed to main app (24 hours = 86400 seconds)
		c.SetCookie("cf_token", token, 86400, "/", "", false, true)
		c.Header("HX-Redirect", "/main")
		c.Status(http.StatusOK)
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

// TODO: These utility functions have been moved to internal/utils/placeholders.go
// They need to be properly implemented and moved to domain-specific utils files