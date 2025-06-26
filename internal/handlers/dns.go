package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// DNSHandler contains dependencies for DNS-related operations
type DNSHandler struct {
	// Add dependencies here as needed
}

// NewDNSHandler creates a new DNS handler instance
func NewDNSHandler() *DNSHandler {
	return &DNSHandler{}
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

// HandleDNSConfigPage renders the DNS configuration page
func (h *DNSHandler) HandleDNSConfigPage(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.Data(http.StatusOK, "text/html", []byte("❌ Error accessing account"))
		return
	}

	// Fetch domains from Cloudflare
	domains, err := h.fetchCloudflareDomains(token)
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

// HandleDNSList returns a JSON list of domains
func (h *DNSHandler) HandleDNSList(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account ID"})
		return
	}

	// Fetch domains from Cloudflare
	domains, err := h.fetchCloudflareDomains(token)
	if err != nil {
		log.Printf("Error fetching domains: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch domains"})
		return
	}

	// Check which domains are managed by Xanthus (exist in KV)
	kvService := services.NewKVService()
	for i := range domains {
		if _, err := kvService.GetDomainSSLConfig(token, accountID, domains[i].Name); err == nil {
			domains[i].Managed = true
		}
	}

	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

// HandleDNSConfigure handles the DNS configuration automation for a domain
func (h *DNSHandler) HandleDNSConfigure(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
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
	if err := utils.GetKVValue(client, token, accountID, "config:ssl:csr", &csrConfig); err != nil {
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

// HandleDNSRemove handles removing DNS configuration for a domain
func (h *DNSHandler) HandleDNSRemove(c *gin.Context) {
	token, err := c.Cookie("cf_token")
	if err != nil || !utils.VerifyCloudflareToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain is required"})
		return
	}

	// Get account ID
	_, accountID, err := utils.CheckKVNamespaceExists(token)
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

// fetchCloudflareDomains fetches all domain zones from Cloudflare API
func (h *DNSHandler) fetchCloudflareDomains(token string) ([]CloudflareDomain, error) {
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
