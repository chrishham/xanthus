package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/ssh"
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
	r.GET("/setup", handleSetupPage)
	r.POST("/setup/hetzner", handleSetupHetzner)
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

// SSHKeyPair represents an SSH key pair
type SSHKeyPair struct {
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	KeyName     string `json:"key_name"`
	CreatedAt   string `json:"created_at"`
}

// SSHKeyKVData represents SSH key data stored in KV (encrypted)
type SSHKeyKVData struct {
	KeyData     string `json:"key_data"`
	KeyName     string `json:"key_name"`
	CreatedAt   string `json:"created_at"`
	Fingerprint string `json:"fingerprint"`
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

		// Initialize SSH key management
		_, err = getOrCreateSSHKey(token, accountID)
		if err != nil {
			log.Printf("Error managing SSH keys: %v", err)
			c.Data(http.StatusOK, "text/html", []byte(fmt.Sprintf("❌ Error setting up SSH keys: %s", err.Error())))
			return
		}

		// Check if this is first-time setup (Hetzner API key missing)
		client := &http.Client{Timeout: 10 * time.Second}
		var hetznerKey string
		if err := getKVValue(client, token, accountID, "config:hetzner:api_key", &hetznerKey); err != nil {
			log.Println("🔧 First-time setup detected - Hetzner API key not found")
			// Set session cookie before redirecting to setup
			c.SetCookie("cf_token", token, 3600, "/", "", false, true)
			c.Header("HX-Redirect", "/setup")
			c.Status(http.StatusOK)
			return
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

func handleSetupPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !verifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	c.HTML(http.StatusOK, "setup.html", gin.H{
		"Step": 1,
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
	c.Header("HX-Redirect", "/main")
	c.Status(http.StatusOK)
}

func handleLogout(c *gin.Context) {
	// Clean up local SSH keys on logout
	if err := removeLocalSSHKeys(); err != nil {
		log.Printf("Warning: failed to clean up SSH keys on logout: %v", err)
	}
	
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

// generateSSHKeyPair generates a new RSA SSH key pair
func generateSSHKeyPair() (*SSHKeyPair, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Convert private key to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %v", err)
	}

	// Format public key for SSH
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	publicKeyString := string(publicKeyBytes)

	// Calculate fingerprint
	fingerprint := ssh.FingerprintSHA256(publicKey)

	// Create key pair
	keyPair := &SSHKeyPair{
		PrivateKey:  string(privateKeyBytes),
		PublicKey:   publicKeyString,
		Fingerprint: fingerprint,
		KeyName:     "xanthus-deploy-key",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	return keyPair, nil
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

// storeSSHKeyInKV stores SSH key data in Cloudflare KV (encrypted)
func storeSSHKeyInKV(token, accountID string, keyPair *SSHKeyPair) error {
	client := &http.Client{Timeout: 10 * time.Second}

	// Encrypt private key
	encryptedPrivateKey, err := encryptData(keyPair.PrivateKey, token)
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %v", err)
	}

	// Store private key
	privateKeyData := SSHKeyKVData{
		KeyData:     encryptedPrivateKey,
		KeyName:     keyPair.KeyName,
		CreatedAt:   keyPair.CreatedAt,
		Fingerprint: keyPair.Fingerprint,
	}

	if err := putKVValue(client, token, accountID, "config:ssh:private_key", privateKeyData); err != nil {
		return fmt.Errorf("failed to store private key: %v", err)
	}

	// Store public key (no encryption needed)
	publicKeyData := SSHKeyKVData{
		KeyData:     keyPair.PublicKey,
		KeyName:     keyPair.KeyName,
		CreatedAt:   keyPair.CreatedAt,
		Fingerprint: keyPair.Fingerprint,
	}

	if err := putKVValue(client, token, accountID, "config:ssh:public_key", publicKeyData); err != nil {
		return fmt.Errorf("failed to store public key: %v", err)
	}

	return nil
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

// getXanthusDir returns the Xanthus configuration directory path
func getXanthusDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	
	xanthusDir := filepath.Join(homeDir, ".xanthus")
	return xanthusDir, nil
}

// ensureXanthusDir creates the Xanthus directory structure if it doesn't exist
func ensureXanthusDir() error {
	xanthusDir, err := getXanthusDir()
	if err != nil {
		return err
	}
	
	// Create main directory
	if err := os.MkdirAll(xanthusDir, 0700); err != nil {
		return fmt.Errorf("failed to create xanthus directory: %v", err)
	}
	
	// Create SSH subdirectory
	sshDir := filepath.Join(xanthusDir, "ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create ssh directory: %v", err)
	}
	
	return nil
}

// saveSSHKeyLocally saves SSH key pair to local cache
func saveSSHKeyLocally(keyPair *SSHKeyPair) error {
	if err := ensureXanthusDir(); err != nil {
		return err
	}
	
	xanthusDir, err := getXanthusDir()
	if err != nil {
		return err
	}
	
	sshDir := filepath.Join(xanthusDir, "ssh")
	
	// Save private key
	privateKeyPath := filepath.Join(sshDir, "xanthus_key")
	if err := os.WriteFile(privateKeyPath, []byte(keyPair.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}
	
	// Save public key
	publicKeyPath := filepath.Join(sshDir, "xanthus_key.pub")
	if err := os.WriteFile(publicKeyPath, []byte(keyPair.PublicKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %v", err)
	}
	
	// Save metadata
	metadata := map[string]string{
		"fingerprint": keyPair.Fingerprint,
		"key_name":    keyPair.KeyName,
		"created_at":  keyPair.CreatedAt,
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}
	
	metadataPath := filepath.Join(sshDir, "key_metadata.json")
	if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %v", err)
	}
	
	log.Printf("✅ SSH keys saved locally to %s", sshDir)
	return nil
}

// loadSSHKeyFromLocal loads SSH key pair from local cache
func loadSSHKeyFromLocal() (*SSHKeyPair, error) {
	xanthusDir, err := getXanthusDir()
	if err != nil {
		return nil, err
	}
	
	sshDir := filepath.Join(xanthusDir, "ssh")
	
	// Check if private key exists
	privateKeyPath := filepath.Join(sshDir, "xanthus_key")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("local SSH key not found")
	}
	
	// Read private key
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}
	
	// Read public key
	publicKeyPath := filepath.Join(sshDir, "xanthus_key.pub")
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %v", err)
	}
	
	// Read metadata
	metadataPath := filepath.Join(sshDir, "key_metadata.json")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %v", err)
	}
	
	var metadata map[string]string
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}
	
	keyPair := &SSHKeyPair{
		PrivateKey:  string(privateKeyBytes),
		PublicKey:   string(publicKeyBytes),
		Fingerprint: metadata["fingerprint"],
		KeyName:     metadata["key_name"],
		CreatedAt:   metadata["created_at"],
	}
	
	return keyPair, nil
}

// removeLocalSSHKeys removes SSH keys from local cache
func removeLocalSSHKeys() error {
	xanthusDir, err := getXanthusDir()
	if err != nil {
		return err
	}
	
	sshDir := filepath.Join(xanthusDir, "ssh")
	
	// Remove all SSH related files
	files := []string{"xanthus_key", "xanthus_key.pub", "key_metadata.json"}
	
	for _, file := range files {
		filePath := filepath.Join(sshDir, file)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to remove %s: %v", filePath, err)
		}
	}
	
	log.Println("✅ Local SSH keys removed")
	return nil
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

// loadSSHKeyFromKV loads SSH key pair from Cloudflare KV
func loadSSHKeyFromKV(token, accountID string) (*SSHKeyPair, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Load private key data
	var privateKeyData SSHKeyKVData
	if err := getKVValue(client, token, accountID, "config:ssh:private_key", &privateKeyData); err != nil {
		return nil, fmt.Errorf("failed to load private key from KV: %v", err)
	}

	// Load public key data
	var publicKeyData SSHKeyKVData
	if err := getKVValue(client, token, accountID, "config:ssh:public_key", &publicKeyData); err != nil {
		return nil, fmt.Errorf("failed to load public key from KV: %v", err)
	}

	// Decrypt private key
	decryptedPrivateKey, err := decryptData(privateKeyData.KeyData, token)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %v", err)
	}

	keyPair := &SSHKeyPair{
		PrivateKey:  decryptedPrivateKey,
		PublicKey:   publicKeyData.KeyData, // Public key is not encrypted
		Fingerprint: privateKeyData.Fingerprint,
		KeyName:     privateKeyData.KeyName,
		CreatedAt:   privateKeyData.CreatedAt,
	}

	return keyPair, nil
}

// getOrCreateSSHKey retrieves SSH key with hybrid approach (local first, KV fallback, generate if none)
func getOrCreateSSHKey(token, accountID string) (*SSHKeyPair, error) {
	// Try to load from local cache first
	if keyPair, err := loadSSHKeyFromLocal(); err == nil {
		log.Println("✅ SSH key loaded from local cache")
		return keyPair, nil
	} else {
		log.Printf("Local SSH key not found: %v", err)
	}

	// Try to load from KV
	if keyPair, err := loadSSHKeyFromKV(token, accountID); err == nil {
		log.Println("✅ SSH key loaded from Cloudflare KV")
		
		// Save to local cache for future use
		if err := saveSSHKeyLocally(keyPair); err != nil {
			log.Printf("Warning: failed to cache SSH key locally: %v", err)
		}
		
		return keyPair, nil
	} else {
		log.Printf("SSH key not found in KV: %v", err)
	}

	// Neither local nor KV has the key, generate new one
	log.Println("🔑 Generating new SSH key pair...")
	keyPair, err := generateSSHKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SSH key: %v", err)
	}

	// Store in KV
	if err := storeSSHKeyInKV(token, accountID, keyPair); err != nil {
		log.Printf("Warning: failed to store SSH key in KV: %v", err)
	} else {
		log.Println("✅ SSH key stored in Cloudflare KV")
	}

	// Store locally
	if err := saveSSHKeyLocally(keyPair); err != nil {
		log.Printf("Warning: failed to store SSH key locally: %v", err)
	}

	log.Printf("✅ New SSH key generated with fingerprint: %s", keyPair.Fingerprint)
	return keyPair, nil
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
