package main

import (
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
		// Set a simple session cookie
		c.SetCookie("cf_token", token, 3600, "/", "", false, true)
		c.Header("HX-Redirect", "/main")
		c.Status(http.StatusOK)
	} else {
		c.Data(http.StatusUnauthorized, "text/html", []byte("Invalid Cloudflare API token. Please check your token and try again."))
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