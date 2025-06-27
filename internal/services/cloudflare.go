package services

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	CloudflareBaseURL = "https://api.cloudflare.com/client/v4"
	CFRootCertURL     = "https://developers.cloudflare.com/ssl/static/origin_ca_rsa_root.pem"
)

// CloudflareService handles Cloudflare API operations
type CloudflareService struct {
	client *http.Client
}

// NewCloudflareService creates a new Cloudflare service instance
func NewCloudflareService() *CloudflareService {
	return &CloudflareService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// CFResponse represents the standard Cloudflare API response structure
type CFResponse struct {
	Success bool        `json:"success"`
	Errors  []CFError   `json:"errors"`
	Result  interface{} `json:"result"`
}

// CFError represents a Cloudflare API error
type CFError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Certificate represents an origin certificate from Cloudflare
type Certificate struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
	ID          string `json:"id"`
}

// DomainSSLConfig represents SSL configuration for a domain
type DomainSSLConfig struct {
	Domain          string `json:"domain"`
	ZoneID          string `json:"zone_id"`
	CertificateID   string `json:"certificate_id"`
	Certificate     string `json:"certificate"`
	PrivateKey      string `json:"private_key"`
	ConfiguredAt    string `json:"configured_at"`
	SSLMode         string `json:"ssl_mode"`
	AlwaysUseHTTPS  bool   `json:"always_use_https"`
	PageRuleCreated bool   `json:"page_rule_created"`
}

// CSRConfig represents a Certificate Signing Request configuration
type CSRConfig struct {
	CSR        string `json:"csr"`
	PrivateKey string `json:"private_key"`
	CreatedAt  string `json:"created_at"`
}

// GenerateCSR generates a new CSR and private key for Cloudflare origin certificates
func (cs *CloudflareService) GenerateCSR() (*CSRConfig, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate request template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"Xanthus K3s Deployment"},
			OrganizationalUnit: []string{"IT"},
		},
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	// Create CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate request: %w", err)
	}

	// Encode CSR to PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	// Encode private key to PEM
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return &CSRConfig{
		CSR:        string(csrPEM),
		PrivateKey: string(privateKeyPEM),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// makeRequest makes an authenticated request to the Cloudflare API
func (cs *CloudflareService) makeRequest(method, endpoint, token string, body interface{}) (*CFResponse, error) {
	url := CloudflareBaseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CFResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			var errorMessages []string
			for _, err := range cfResp.Errors {
				errorMessages = append(errorMessages, fmt.Sprintf("Code %d: %s", err.Code, err.Message))
			}
			return nil, fmt.Errorf("API error: %s. Full response: %s", strings.Join(errorMessages, "; "), string(respBody))
		}
		return nil, fmt.Errorf("API request failed. Full response: %s", string(respBody))
	}

	return &cfResp, nil
}

// GetZoneID retrieves the zone ID for a given domain
func (cs *CloudflareService) GetZoneID(token, domain string) (string, error) {
	resp, err := cs.makeRequest("GET", "/zones?name="+domain, token, nil)
	if err != nil {
		return "", err
	}

	// Parse zones from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	var zones []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resultBytes, &zones); err != nil {
		return "", fmt.Errorf("failed to parse zones: %w", err)
	}

	if len(zones) == 0 {
		return "", fmt.Errorf("no zone found for domain %s", domain)
	}

	return zones[0].ID, nil
}

// SetSSLMode sets SSL/TLS mode to Full (strict)
func (cs *CloudflareService) SetSSLMode(token, zoneID string) error {
	body := map[string]string{"value": "strict"}
	_, err := cs.makeRequest("PATCH",
		fmt.Sprintf("/zones/%s/settings/ssl", zoneID),
		token, body)
	return err
}

// CreateOriginCertificate creates an origin server certificate for the domain using stored CSR
func (cs *CloudflareService) CreateOriginCertificate(token, domain, csr string) (*Certificate, error) {
	body := map[string]interface{}{
		"hostnames":          []string{domain, "*." + domain},
		"requested_validity": 5475, // 15 years (maximum)
		"request_type":       "origin-rsa",
		"csr":                csr,
	}

	resp, err := cs.makeRequest("POST", "/certificates", token, body)
	if err != nil {
		return nil, err
	}

	// Parse certificate from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(resultBytes, &cert); err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &cert, nil
}

// AppendRootCertificate downloads and appends the Cloudflare root certificate
func (cs *CloudflareService) AppendRootCertificate(certificate string) (string, error) {
	// Download root certificate
	resp, err := http.Get(CFRootCertURL)
	if err != nil {
		return "", fmt.Errorf("failed to download root certificate: %w", err)
	}
	defer resp.Body.Close()

	rootCert, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read root certificate: %w", err)
	}

	// Append root certificate to the original certificate
	fullCertificate := certificate + string(rootCert)
	return fullCertificate, nil
}

// EnableAlwaysHTTPS enables the "Always Use HTTPS" setting
func (cs *CloudflareService) EnableAlwaysHTTPS(token, zoneID string) error {
	body := map[string]string{"value": "on"}
	_, err := cs.makeRequest("PATCH",
		fmt.Sprintf("/zones/%s/settings/always_use_https", zoneID),
		token, body)
	return err
}

// CreatePageRule creates a page rule for www to non-www redirect
func (cs *CloudflareService) CreatePageRule(token, zoneID, domain string) error {
	body := map[string]interface{}{
		"targets": []map[string]interface{}{
			{
				"target": "url",
				"constraint": map[string]string{
					"operator": "matches",
					"value":    fmt.Sprintf("www.%s/*", domain),
				},
			},
		},
		"actions": []map[string]interface{}{
			{
				"id": "forwarding_url",
				"value": map[string]interface{}{
					"url":         fmt.Sprintf("https://%s/$1", domain),
					"status_code": 301,
				},
			},
		},
		"priority": 1,
		"status":   "active",
	}

	_, err := cs.makeRequest("POST",
		fmt.Sprintf("/zones/%s/pagerules", zoneID),
		token, body)
	return err
}

// DeleteOriginCertificate removes an origin certificate
func (cs *CloudflareService) DeleteOriginCertificate(token, certificateID string) error {
	_, err := cs.makeRequest("DELETE",
		fmt.Sprintf("/certificates/%s", certificateID),
		token, nil)
	return err
}

// GetPageRules retrieves page rules for a zone
func (cs *CloudflareService) GetPageRules(token, zoneID string) ([]map[string]interface{}, error) {
	resp, err := cs.makeRequest("GET",
		fmt.Sprintf("/zones/%s/pagerules", zoneID),
		token, nil)
	if err != nil {
		return nil, err
	}

	// Parse page rules from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var pageRules []map[string]interface{}
	if err := json.Unmarshal(resultBytes, &pageRules); err != nil {
		return nil, fmt.Errorf("failed to parse page rules: %w", err)
	}

	return pageRules, nil
}

// DeletePageRule removes a page rule
func (cs *CloudflareService) DeletePageRule(token, zoneID, pageRuleID string) error {
	_, err := cs.makeRequest("DELETE",
		fmt.Sprintf("/zones/%s/pagerules/%s", zoneID, pageRuleID),
		token, nil)
	return err
}

// ResetSSLMode sets SSL/TLS mode back to Flexible
func (cs *CloudflareService) ResetSSLMode(token, zoneID string) error {
	body := map[string]string{"value": "flexible"}
	_, err := cs.makeRequest("PATCH",
		fmt.Sprintf("/zones/%s/settings/ssl", zoneID),
		token, body)
	return err
}

// DisableAlwaysHTTPS disables the "Always Use HTTPS" setting
func (cs *CloudflareService) DisableAlwaysHTTPS(token, zoneID string) error {
	body := map[string]string{"value": "off"}
	_, err := cs.makeRequest("PATCH",
		fmt.Sprintf("/zones/%s/settings/always_use_https", zoneID),
		token, body)
	return err
}

// RemoveDomainFromXanthus reverts all SSL changes made by Xanthus
func (cs *CloudflareService) RemoveDomainFromXanthus(token, domain string, config *DomainSSLConfig) error {
	// Step 1: Delete origin certificate
	if config.CertificateID != "" {
		if err := cs.DeleteOriginCertificate(token, config.CertificateID); err != nil {
			return fmt.Errorf("failed to delete origin certificate: %w", err)
		}
	}

	// Step 2: Delete page rules created for this domain
	if config.PageRuleCreated {
		pageRules, err := cs.GetPageRules(token, config.ZoneID)
		if err != nil {
			return fmt.Errorf("failed to get page rules: %w", err)
		}

		// Find and delete page rules for www redirect
		for _, rule := range pageRules {
			if targets, ok := rule["targets"].([]interface{}); ok {
				for _, target := range targets {
					if targetMap, ok := target.(map[string]interface{}); ok {
						if constraint, ok := targetMap["constraint"].(map[string]interface{}); ok {
							if value, ok := constraint["value"].(string); ok {
								expectedPattern := fmt.Sprintf("www.%s/*", domain)
								if value == expectedPattern {
									if ruleID, ok := rule["id"].(string); ok {
										if err := cs.DeletePageRule(token, config.ZoneID, ruleID); err != nil {
											return fmt.Errorf("failed to delete page rule: %w", err)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Step 3: Reset SSL mode to Flexible
	if config.SSLMode == "strict" {
		if err := cs.ResetSSLMode(token, config.ZoneID); err != nil {
			return fmt.Errorf("failed to reset SSL mode: %w", err)
		}
	}

	// Step 4: Disable Always Use HTTPS
	if config.AlwaysUseHTTPS {
		if err := cs.DisableAlwaysHTTPS(token, config.ZoneID); err != nil {
			return fmt.Errorf("failed to disable always HTTPS: %w", err)
		}
	}

	return nil
}

// ConfigureDomainSSL performs all SSL configuration steps for a domain
func (cs *CloudflareService) ConfigureDomainSSL(token, domain, csr, csrPrivateKey string) (*DomainSSLConfig, error) {
	config := &DomainSSLConfig{
		Domain:       domain,
		ConfiguredAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Step 1: Get Zone ID
	zoneID, err := cs.GetZoneID(token, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}
	config.ZoneID = zoneID

	// Step 2: Set SSL mode to Full (strict)
	if err := cs.SetSSLMode(token, zoneID); err != nil {
		return nil, fmt.Errorf("failed to set SSL mode: %w", err)
	}
	config.SSLMode = "strict"

	// Step 3: Create Origin Server Certificate
	cert, err := cs.CreateOriginCertificate(token, domain, csr)
	if err != nil {
		return nil, fmt.Errorf("failed to create origin certificate: %w", err)
	}
	config.CertificateID = cert.ID
	config.PrivateKey = csrPrivateKey

	// Step 4: Append Cloudflare Root Certificate
	fullCert, err := cs.AppendRootCertificate(cert.Certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to append root certificate: %w", err)
	}
	config.Certificate = fullCert

	// Step 5: Enable Always Use HTTPS
	if err := cs.EnableAlwaysHTTPS(token, zoneID); err != nil {
		return nil, fmt.Errorf("failed to enable always HTTPS: %w", err)
	}
	config.AlwaysUseHTTPS = true

	// Step 6: Create Page Rule for www redirect
	if err := cs.CreatePageRule(token, zoneID, domain); err != nil {
		return nil, fmt.Errorf("failed to create page rule: %w", err)
	}
	config.PageRuleCreated = true

	return config, nil
}

// ConvertPrivateKeyToSSH converts a PEM-encoded RSA private key to SSH public key format
func (cs *CloudflareService) ConvertPrivateKeyToSSH(privateKeyPEM string) (string, error) {
	// Parse the PEM private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the private key")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Ensure it's an RSA private key
	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key is not an RSA key")
	}

	// Generate SSH public key from RSA private key
	sshPublicKey, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Format as SSH authorized_keys format
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey))), nil
}

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	Content string                 `json:"content"`
	Proxied bool                   `json:"proxied"`
	TTL     int                    `json:"ttl"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// DNSRecordsResponse represents the API response for DNS records
type DNSRecordsResponse struct {
	Success bool        `json:"success"`
	Result  []DNSRecord `json:"result"`
	Errors  []CFError   `json:"errors"`
}

// GetDNSRecords retrieves all DNS records for a zone
func (cs *CloudflareService) GetDNSRecords(token, zoneID string) ([]DNSRecord, error) {
	resp, err := cs.makeRequest("GET", fmt.Sprintf("/zones/%s/dns_records", zoneID), token, nil)
	if err != nil {
		return nil, err
	}

	// Parse DNS records from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var records []DNSRecord
	if err := json.Unmarshal(resultBytes, &records); err != nil {
		return nil, fmt.Errorf("failed to parse DNS records: %w", err)
	}

	return records, nil
}

// DeleteDNSRecord removes a DNS record by ID
func (cs *CloudflareService) DeleteDNSRecord(token, zoneID, recordID string) error {
	_, err := cs.makeRequest("DELETE", fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID), token, nil)
	return err
}

// CreateDNSRecord creates a new DNS record
func (cs *CloudflareService) CreateDNSRecord(token, zoneID, recordType, name, content string, proxied bool) (*DNSRecord, error) {
	body := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"proxied": proxied,
		"ttl":     1, // Auto TTL when proxied, otherwise minimum
	}

	resp, err := cs.makeRequest("POST", fmt.Sprintf("/zones/%s/dns_records", zoneID), token, body)
	if err != nil {
		return nil, err
	}

	// Parse DNS record from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var record DNSRecord
	if err := json.Unmarshal(resultBytes, &record); err != nil {
		return nil, fmt.Errorf("failed to parse DNS record: %w", err)
	}

	return &record, nil
}

// ConfigureDNSForVPS configures DNS records for a VPS deployment
func (cs *CloudflareService) ConfigureDNSForVPS(token, domain, vpsIP string) error {
	// Get zone ID for the domain
	zoneID, err := cs.GetZoneID(token, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Get existing DNS records
	existingRecords, err := cs.GetDNSRecords(token, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get existing DNS records: %w", err)
	}

	log.Printf("📋 Found %d existing DNS records for domain %s", len(existingRecords), domain)
	for _, record := range existingRecords {
		if record.Type == "A" {
			log.Printf("📍 Existing A record: %s -> %s", record.Name, record.Content)
		}
	}

	// Delete all existing A records for the domain
	recordsDeleted := 0
	for _, record := range existingRecords {
		// Normalize record name (remove trailing dot if present)
		recordName := strings.TrimSuffix(record.Name, ".")
		
		// Check if this is an A record we should delete
		shouldDelete := record.Type == "A" && (
			recordName == domain ||
			recordName == "*."+domain ||
			recordName == "www."+domain ||
			record.Name == domain ||
			record.Name == "*."+domain ||
			record.Name == "www."+domain)
			
		if shouldDelete {
			log.Printf("🗑️ Deleting existing A record: %s -> %s", record.Name, record.Content)
			if err := cs.DeleteDNSRecord(token, zoneID, record.ID); err != nil {
				return fmt.Errorf("failed to delete existing A record %s: %w", record.Name, err)
			}
			recordsDeleted++
		}
	}
	log.Printf("🗑️ Deleted %d existing A records", recordsDeleted)

	// Create new A records pointing to the VPS IP
	recordsToCreate := []struct {
		name    string
		proxied bool
	}{
		{domain, true},          // Root domain with proxy
		{"*." + domain, true},   // Wildcard subdomain with proxy
		{"www." + domain, true}, // www subdomain with proxy
	}

	for _, recordInfo := range recordsToCreate {
		log.Printf("➕ Creating A record: %s -> %s (proxied: %v)", recordInfo.name, vpsIP, recordInfo.proxied)
		if _, err := cs.CreateDNSRecord(token, zoneID, "A", recordInfo.name, vpsIP, recordInfo.proxied); err != nil {
			return fmt.Errorf("failed to create A record for %s: %w", recordInfo.name, err)
		}
		log.Printf("✅ Created A record: %s -> %s", recordInfo.name, vpsIP)
	}

	// Verify DNS records were created successfully
	log.Printf("🔍 Verifying DNS records were created...")
	updatedRecords, err := cs.GetDNSRecords(token, zoneID)
	if err != nil {
		log.Printf("Warning: Could not verify DNS records: %v", err)
	} else {
		foundRecords := 0
		for _, record := range updatedRecords {
			if record.Type == "A" && record.Content == vpsIP {
				log.Printf("✅ Verified A record: %s -> %s", record.Name, record.Content)
				foundRecords++
			}
		}
		log.Printf("✅ Verified %d DNS records pointing to VPS IP %s", foundRecords, vpsIP)
	}

	return nil
}
