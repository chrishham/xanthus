package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {

	port := findAvailablePort()
	if port == "" {
		log.Fatal("Could not find an available port")
	}

	fmt.Printf("🚀 Xanthus is starting on http://localhost:%s\n", port)

	// Set Gin to release mode for production use
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Configure trusted proxies for security
	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "web/static")

	// Routes
	r.GET("/", handleRoot)
	r.GET("/login", handleLoginPage)
	r.POST("/login", handleLogin)
	r.GET("/main", handleMainPage)
	r.GET("/setup", handleSetupPage)
	r.POST("/setup/hetzner", handleSetupHetzner)
	r.GET("/setup/server", handleSetupServerPage)
	r.POST("/setup/server", handleSetupServer)
	r.GET("/dns", handleDNSConfigPage)
	r.POST("/dns/configure", handleDNSConfigure)
	r.POST("/dns/remove", handleDNSRemove)
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

// HetznerLocation represents a Hetzner datacenter location
type HetznerLocation struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// HetznerServerType represents a Hetzner server type/instance
type HetznerServerType struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Cores              int             `json:"cores"`
	Memory             float64         `json:"memory"`
	Disk               int             `json:"disk"`
	Prices             []HetznerPrice  `json:"prices"`
	StorageType        string          `json:"storage_type"`
	CPUType            string          `json:"cpu_type"`
	Architecture       string          `json:"architecture"`
	AvailableLocations map[string]bool `json:"available_locations,omitempty"`
}

// HetznerPrice represents pricing information for a server type
type HetznerPrice struct {
	Location     string             `json:"location"`
	PriceHourly  HetznerPriceDetail `json:"price_hourly"`
	PriceMonthly HetznerPriceDetail `json:"price_monthly"`
}

// HetznerPriceDetail represents price details
type HetznerPriceDetail struct {
	Net   string `json:"net"`
	Gross string `json:"gross"`
}

// HetznerLocationsResponse represents the API response for locations
type HetznerLocationsResponse struct {
	Locations []HetznerLocation `json:"locations"`
}

// HetznerServerTypesResponse represents the API response for server types
type HetznerServerTypesResponse struct {
	ServerTypes []HetznerServerType `json:"server_types"`
}

// HetznerDatacenter represents a Hetzner datacenter with availability info
type HetznerDatacenter struct {
	ID          int                          `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Location    HetznerLocation              `json:"location"`
	ServerTypes HetznerDatacenterServerTypes `json:"server_types"`
}

// HetznerDatacenterServerTypes represents server type availability in a datacenter
type HetznerDatacenterServerTypes struct {
	Supported             []int `json:"supported"`
	Available             []int `json:"available"`
	AvailableForMigration []int `json:"available_for_migration"`
}

// HetznerDatacentersResponse represents the API response for datacenters
type HetznerDatacentersResponse struct {
	Datacenters []HetznerDatacenter `json:"datacenters"`
}

// CloudflareDomain represents a domain zone in Cloudflare
type CloudflareDomain struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Paused     bool   `json:"paused"`
	Type       string `json:"type"`
	Managed    bool   `json:"managed"`
	CreatedOn  string `json:"created_on"`
	ModifiedOn string `json:"modified_on"`
}

// CloudflareDomainsResponse represents the API response for domain zones
type CloudflareDomainsResponse struct {
	Success bool               `json:"success"`
	Result  []CloudflareDomain `json:"result"`
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

		// Check and create CSR if not exists
		client := &http.Client{Timeout: 10 * time.Second}
		var existingCSR map[string]interface{}
		if err := getKVValue(client, token, accountID, "config:ssl:csr", &existingCSR); err != nil {
			log.Println("🔧 Generating new CSR for SSL certificates")
			
			cfService := services.NewCloudflareService()
			csrConfig, err := cfService.GenerateCSR()
			if err != nil {
				log.Printf("Error generating CSR: %v", err)
				c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error generating CSR: %s", err.Error())))
				return
			}

			// Store CSR in KV
			if err := putKVValue(client, token, accountID, "config:ssl:csr", csrConfig); err != nil {
				log.Printf("Error storing CSR: %v", err)
				c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error storing CSR: %s", err.Error())))
				return
			}

			log.Println("✅ CSR generated and stored successfully")
		} else {
			log.Println("✅ CSR already exists in KV")
		}

		// Check setup completion status
		// 1. Check if Hetzner API key exists
		var hetznerKey string
		if err := getKVValue(client, token, accountID, "config:hetzner:api_key", &hetznerKey); err != nil {
			log.Println("🔧 Setup step 1 required - Hetzner API key not found")
			// Set session cookie before redirecting to setup
			c.SetCookie("cf_token", token, 3600, "/", "", false, true)
			c.Header("HX-Redirect", "/setup")
			c.Status(http.StatusOK)
			return
		}

		// 2. Check if server configuration exists
		var serverConfig map[string]string
		if err := getKVValue(client, token, accountID, "config:server:selection", &serverConfig); err != nil {
			log.Println("🔧 Setup step 2 required - Server configuration not found")
			// Set session cookie before redirecting to server selection
			c.SetCookie("cf_token", token, 3600, "/", "", false, true)
			c.Header("HX-Redirect", "/setup/server")
			c.Status(http.StatusOK)
			return
		}

		// All setup complete, proceed to main app
		log.Printf("✅ Setup complete - Server: %s in %s", serverConfig["server_type"], serverConfig["location"])
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

func handleSetupPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	c.HTML(http.StatusOK, "setup.html", gin.H{
		"Step":  1,
		"Title": "Setup - Hetzner API Key",
	})
}

func handleSetupHetzner(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	hetznerKey := c.PostForm("hetzner_key")
	if hetznerKey == "" {
		c.Data(http.StatusBadRequest, "text/html", []byte("❌ Hetzner API key is required"))
		return
	}

	// Validate Hetzner API key
	if !validateHetznerAPIKey(hetznerKey) {
		c.Data(http.StatusOK, "text/html", []byte("❌ Invalid Hetzner API key. Please check your key and try again."))
		return
	}

	// Get account ID for KV storage
	_, accountID, err := checkKVNamespaceExists(token)
	if err != nil {
		log.Printf("Error getting account ID: %v", err)
		c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error: %s", err.Error())))
		return
	}

	// Store encrypted Hetzner API key in KV
	client := &http.Client{Timeout: 10 * time.Second}
	encryptedKey, err := encryptData(hetznerKey, token)
	if err != nil {
		log.Printf("Error encrypting Hetzner key: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error storing API key"))
		return
	}

	if err := putKVValue(client, token, accountID, "config:hetzner:api_key", encryptedKey); err != nil {
		log.Printf("Error storing Hetzner key: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error storing API key"))
		return
	}

	log.Println("✅ Hetzner API key validated and stored")
	c.Header("HX-Redirect", "/setup/server")
	c.Status(http.StatusOK)
}

func handleSetupServerPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	// Get account ID to fetch Hetzner key
	_, accountID, err := checkKVNamespaceExists(token)
	if err != nil {
		c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
		return
	}

	// Get decrypted Hetzner API key
	hetznerKey, err := getHetznerAPIKey(token, accountID)
	if err != nil {
		log.Printf("Error getting Hetzner API key: %v", err)
		c.Redirect(http.StatusTemporaryRedirect, "/setup")
		return
	}

	// Fetch locations, server types, and real availability from Hetzner API
	locations, err := fetchHetznerLocations(hetznerKey)
	if err != nil {
		log.Printf("Error fetching locations: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error fetching server locations"))
		return
	}

	serverTypes, err := fetchHetznerServerTypes(hetznerKey)
	if err != nil {
		log.Printf("Error fetching server types: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error fetching server types"))
		return
	}

	// Fetch real-time availability data
	availability, err := fetchServerAvailability(hetznerKey)
	if err != nil {
		log.Printf("Error fetching server availability: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error fetching server availability"))
		return
	}

	// Filter to only shared vCPU instances and add availability info
	sharedServerTypes := filterSharedVCPUServers(serverTypes)

	// Add availability information to each server type
	for i := range sharedServerTypes {
		sharedServerTypes[i].AvailableLocations = make(map[string]bool)
		for locationName, locationAvailability := range availability {
			if available, exists := locationAvailability[sharedServerTypes[i].ID]; exists && available {
				sharedServerTypes[i].AvailableLocations[locationName] = true
			}
		}
	}

	c.HTML(http.StatusOK, "setup-server.html", gin.H{
		"Step":        2,
		"Title":       "Server Selection",
		"Locations":   locations,
		"ServerTypes": sharedServerTypes,
	})
}

func handleSetupServer(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	location := c.PostForm("location")
	serverType := c.PostForm("server_type")

	if location == "" || serverType == "" {
		c.Data(http.StatusBadRequest, "text/html", []byte("❌ Please select both location and server type"))
		return
	}

	// Get account ID for KV storage
	_, accountID, err := checkKVNamespaceExists(token)
	if err != nil {
		c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
		return
	}

	// Store server configuration in KV
	client := &http.Client{Timeout: 10 * time.Second}
	serverConfig := map[string]string{
		"location":    location,
		"server_type": serverType,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}

	if err := putKVValue(client, token, accountID, "config:server:selection", serverConfig); err != nil {
		log.Printf("Error storing server config: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error storing server configuration"))
		return
	}

	log.Printf("✅ Server configuration stored: %s in %s", serverType, location)
	c.Header("HX-Redirect", "/setup/step3")
	c.Status(http.StatusOK)
}

func handleDNSConfigPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	// Get account ID
	_, accountID, err := checkKVNamespaceExists(token)
	if err != nil {
		c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
		return
	}

	// Fetch domains from Cloudflare
	domains, err := fetchCloudflareDomains(token)
	if err != nil {
		log.Printf("Error fetching domains: %v", err)
		c.Data(http.StatusOK, "text/html", []byte("❌ Error fetching domains"))
		return
	}

	// Check which domains are managed by Xanthus (exist in KV)
	kvService := services.NewKVService()
	for i := range domains {
		if _, err := kvService.GetDomainSSLConfig(token, accountID, domains[i].Name); err == nil {
			domains[i].Managed = true
		}
	}

	c.HTML(http.StatusOK, "dns-config.html", gin.H{
		"Domains": domains,
	})
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

// encryptData encrypts data using AES-256-GCM with a key derived from the CF token
func encryptData(data, token string) (string, error) {
	// Derive key from token using SHA256
	hash := sha256.Sum256([]byte(token))
	key := hash[:]

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptData decrypts data using AES-256-GCM with a key derived from the CF token
func decryptData(encryptedData, token string) (string, error) {
	// Derive key from token using SHA256
	hash := sha256.Sum256([]byte(token))
	key := hash[:]

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %v", err)
	}

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %v", err)
	}

	return string(plaintext), nil
}

// putKVValue stores a value in Cloudflare KV
func putKVValue(client *http.Client, token, accountID, key string, value interface{}) error {
	// First, get the Xanthus namespace ID
	namespaceID, err := getXanthusNamespaceID(client, token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get namespace ID: %v", err)
	}

	// Marshal value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %v", err)
	}

	// Create request
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		accountID, namespaceID, key)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(valueBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var kvResp CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&kvResp); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if !kvResp.Success {
		return fmt.Errorf("KV put failed: %v", kvResp.Errors)
	}

	return nil
}

// getXanthusNamespaceID retrieves the Xanthus namespace ID
func getXanthusNamespaceID(client *http.Client, token, accountID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces", accountID), nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	var kvResp KVNamespaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&kvResp); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	if !kvResp.Success {
		return "", fmt.Errorf("KV API failed: %v", kvResp.Errors)
	}

	// Find Xanthus namespace
	for _, ns := range kvResp.Result {
		if ns.Title == "Xanthus" {
			return ns.ID, nil
		}
	}

	return "", fmt.Errorf("Xanthus namespace not found")
}

// getKVValue retrieves a value from Cloudflare KV
func getKVValue(client *http.Client, token, accountID, key string, result interface{}) error {
	// Get the Xanthus namespace ID
	namespaceID, err := getXanthusNamespaceID(client, token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get namespace ID: %v", err)
	}

	// Create request
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		accountID, namespaceID, key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("key not found in KV")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("KV API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	return nil
}

// validateHetznerAPIKey validates a Hetzner Cloud API key by making a test API call
func validateHetznerAPIKey(apiKey string) bool {
	client := &http.Client{Timeout: 10 * time.Second}

	// Test the API key by fetching server types (minimal API call)
	req, err := http.NewRequest("GET", "https://api.hetzner.cloud/v1/server_types", nil)
	if err != nil {
		log.Printf("Error creating Hetzner API request: %v", err)
		return false
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making Hetzner API request: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Check if the response is successful (200 OK)
	if resp.StatusCode == 200 {
		log.Println("✅ Hetzner API key validated successfully")
		return true
	}

	log.Printf("❌ Hetzner API key validation failed with status: %d", resp.StatusCode)
	return false
}

// getHetznerAPIKey retrieves and decrypts the Hetzner API key from KV
func getHetznerAPIKey(token, accountID string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var encryptedKey string

	if err := getKVValue(client, token, accountID, "config:hetzner:api_key", &encryptedKey); err != nil {
		return "", fmt.Errorf("failed to get Hetzner API key: %v", err)
	}

	decryptedKey, err := decryptData(encryptedKey, token)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt Hetzner API key: %v", err)
	}

	return decryptedKey, nil
}

// fetchHetznerLocations fetches available datacenter locations from Hetzner API
func fetchHetznerLocations(apiKey string) ([]HetznerLocation, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", "https://api.hetzner.cloud/v1/locations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch locations: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var locationsResp HetznerLocationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&locationsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return locationsResp.Locations, nil
}

// fetchHetznerServerTypes fetches available server types from Hetzner API
func fetchHetznerServerTypes(apiKey string) ([]HetznerServerType, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", "https://api.hetzner.cloud/v1/server_types", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server types: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var serverTypesResp HetznerServerTypesResponse
	if err := json.NewDecoder(resp.Body).Decode(&serverTypesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return serverTypesResp.ServerTypes, nil
}

// fetchServerAvailability fetches real-time server availability for all datacenters
func fetchServerAvailability(apiKey string) (map[string]map[int]bool, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	// Use the datacenters endpoint to get real availability info
	req, err := http.NewRequest("GET", "https://api.hetzner.cloud/v1/datacenters", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch datacenters: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var datacentersResp HetznerDatacentersResponse
	if err := json.NewDecoder(resp.Body).Decode(&datacentersResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Build availability map: [location][serverTypeID] = available
	availability := make(map[string]map[int]bool)

	for _, datacenter := range datacentersResp.Datacenters {
		locationName := datacenter.Location.Name
		availability[locationName] = make(map[int]bool)

		// Mark available server types for this location
		for _, serverTypeID := range datacenter.ServerTypes.Available {
			availability[locationName][serverTypeID] = true
		}
	}

	return availability, nil
}

// filterSharedVCPUServers filters server types to only include shared vCPU instances
func filterSharedVCPUServers(serverTypes []HetznerServerType) []HetznerServerType {
	var sharedServers []HetznerServerType

	for _, server := range serverTypes {
		// Filter for shared vCPU types (typically start with "cpx" or "cx")
		if server.CPUType == "shared" {
			sharedServers = append(sharedServers, server)
		}
	}

	return sharedServers
}

// fetchCloudflareDomains fetches all domain zones from Cloudflare API
func fetchCloudflareDomains(token string) ([]CloudflareDomain, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/zones", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch domains: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var domainsResp CloudflareDomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&domainsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !domainsResp.Success {
		return nil, fmt.Errorf("API call failed: %v", domainsResp.Errors)
	}

	return domainsResp.Result, nil
}

// handleDNSConfigure handles the DNS configuration automation for a domain
func handleDNSConfigure(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required"})
		return
	}

	// Get account ID
	_, accountID, err := checkKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Initialize services
	cfService := services.NewCloudflareService()
	kvService := services.NewKVService()

	// Check if domain is already configured
	existingConfig, err := kvService.GetDomainSSLConfig(token, accountID, domain)
	if err == nil && existingConfig != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Domain already configured",
			"config": existingConfig,
		})
		return
	}

	// Get CSR from KV
	client := &http.Client{Timeout: 10 * time.Second}
	var csrConfig struct {
		CSR        string `json:"csr"`
		PrivateKey string `json:"private_key"`
		CreatedAt  string `json:"created_at"`
	}
	if err := getKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
		log.Printf("Error getting CSR from KV: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CSR not found. Please logout and login again."})
		return
	}

	// Configure SSL for the domain
	sslConfig, err := cfService.ConfigureDomainSSL(token, domain, csrConfig.CSR, csrConfig.PrivateKey)
	if err != nil {
		log.Printf("Error configuring SSL for domain %s: %v", domain, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("SSL configuration failed: %v", err)})
		return
	}

	// Store configuration in KV
	if err := kvService.StoreDomainSSLConfig(token, accountID, sslConfig); err != nil {
		log.Printf("Error storing SSL config for domain %s: %v", domain, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store configuration"})
		return
	}

	log.Printf("✅ SSL configuration completed for domain: %s", domain)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "SSL configuration completed successfully",
		"config":  sslConfig,
	})
}

// handleDNSRemove handles removing DNS configuration for a domain
func handleDNSRemove(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required"})
		return
	}

	// Get account ID
	_, accountID, err := checkKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Initialize services
	cfService := services.NewCloudflareService()
	kvService := services.NewKVService()

	// Get existing domain configuration
	domainConfig, err := kvService.GetDomainSSLConfig(token, accountID, domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Domain configuration not found"})
		return
	}

	// Revert all Cloudflare changes made by Xanthus
	if err := cfService.RemoveDomainFromXanthus(token, domain, domainConfig); err != nil {
		log.Printf("Error reverting Cloudflare changes for domain %s: %v", domain, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to revert Cloudflare changes: %v", err)})
		return
	}

	// Remove configuration from KV
	if err := kvService.DeleteDomainSSLConfig(token, accountID, domain); err != nil {
		log.Printf("Error removing SSL config for domain %s: %v", domain, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove configuration"})
		return
	}

	log.Printf("✅ SSL configuration and Cloudflare changes reverted for domain: %s", domain)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Domain configuration removed and Cloudflare changes reverted successfully",
	})
}
