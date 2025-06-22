package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := findAvailablePort()
	if port == "" {
		log.Fatal("Could not find an available port")
	}

	fmt.Printf("🚀 Xanthus is starting on http://localhost:%s\n", port)

	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*")

	// Routes
	r.GET("/", handleRoot)
	r.GET("/login", handleLoginPage)
	r.POST("/login", handleLogin)
	r.GET("/main", handleMainPage)
	r.GET("/logout", handleLogout)
	r.GET("/health", handleHealth)

	log.Fatal(r.Run(":" + port))
}

// CloudflareResponse represents the API response structure
type CloudflareResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// KVNamespace represents a Cloudflare KV namespace
type KVNamespace struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// KVNamespaceResponse represents the API response for KV namespaces
type KVNamespaceResponse struct {
	Success bool          `json:"success"`
	Result  []KVNamespace `json:"result"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func handleRoot(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

func handleLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func handleLogin(c *gin.Context) {
	token := c.PostForm("cf_token")
	if token == "" {
		c.Data(http.StatusBadRequest, "text/html", []byte("API token is required"))
		return
	}

	if verifyCloudflareToken(token) {
		// Check if Xanthus KV namespace exists, create if not
		exists, accountID, err := checkKVNamespaceExists(token)
		if err != nil {
			log.Printf("Error checking KV namespace: %v", err)
			c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error checking KV namespace: %s", err.Error())))
			return
		}

		if !exists {
			if err := createKVNamespace(token, accountID); err != nil {
				log.Printf("Error creating KV namespace: %v", err)
				c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error creating KV namespace: %s", err.Error())))
				return
			}
		} else {
			log.Println("✅ Xanthus KV namespace already exists")
		}

		// Set a simple session cookie
		c.SetCookie("cf_token", token, 3600, "/", "", false, true)
		c.Header("HX-Redirect", "/main")
		c.Status(http.StatusOK)
	} else {
		c.Data(http.StatusOK, "text/html", []byte("❌ Invalid Cloudflare API token. Please check your token and try again."))
	}
}

func handleMainPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	c.HTML(http.StatusOK, "main.html", nil)
}

func handleLogout(c *gin.Context) {
	c.SetCookie("cf_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}

func verifyCloudflareToken(token string) bool {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/user/tokens/verify", nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return false
	}
	defer resp.Body.Close()

	var cfResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		log.Printf("Error decoding response: %v", err)
		return false
	}

	return cfResp.Success
}

// checkKVNamespaceExists checks if the "Xanthus" KV namespace exists
func checkKVNamespaceExists(token string) (bool, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Get account memberships to find account ID
	req, err := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/memberships", nil)
	if err != nil {
		return false, "", fmt.Errorf("error creating memberships request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("error getting memberships: %v", err)
	}
	defer resp.Body.Close()

	var membershipResp struct {
		Success bool `json:"success"`
		Result  []struct {
			Account struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"account"`
		} `json:"result"`
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&membershipResp); err != nil {
		return false, "", fmt.Errorf("error decoding membership response: %v", err)
	}

	if !membershipResp.Success {
		return false, "", fmt.Errorf("memberships API call failed: %v", membershipResp.Errors)
	}

	if len(membershipResp.Result) == 0 {
		return false, "", fmt.Errorf("no account memberships found - token needs Account:Cloudflare Workers:Edit permission")
	}

	accountID := membershipResp.Result[0].Account.ID

	// Check KV namespaces for this account
	kvReq, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces", accountID), nil)
	if err != nil {
		return false, "", fmt.Errorf("error creating KV request: %v", err)
	}

	kvReq.Header.Set("Authorization", "Bearer "+token)
	kvReq.Header.Set("Content-Type", "application/json")

	kvResp, err := client.Do(kvReq)
	if err != nil {
		return false, "", fmt.Errorf("error getting KV namespaces: %v", err)
	}
	defer kvResp.Body.Close()

	var kvNamespaceResp KVNamespaceResponse
	if err := json.NewDecoder(kvResp.Body).Decode(&kvNamespaceResp); err != nil {
		return false, "", fmt.Errorf("error decoding KV response: %v", err)
	}

	if !kvNamespaceResp.Success {
		return false, "", fmt.Errorf("KV API call failed: %v", kvNamespaceResp.Errors)
	}

	// Check if "Xanthus" namespace exists
	for _, ns := range kvNamespaceResp.Result {
		if ns.Title == "Xanthus" {
			return true, accountID, nil
		}
	}

	return false, accountID, nil
}

// createKVNamespace creates the "Xanthus" KV namespace
func createKVNamespace(token, accountID string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	payload := map[string]string{
		"title": "Xanthus",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces", accountID), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error creating namespace: %v", err)
	}
	defer resp.Body.Close()

	var createResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	log.Printf("DEBUG: Create KV Namespace API response: %+v", createResp)

	if !createResp.Success {
		return fmt.Errorf("failed to create namespace: %v", createResp.Errors)
	}

	log.Println("✅ Created Xanthus KV namespace successfully")
	return nil
}

func findAvailablePort() string {
	for port := 8080; port <= 8090; port++ {
		address := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			listener.Close()
			return fmt.Sprintf("%d", port)
		}
	}
	return ""
}
