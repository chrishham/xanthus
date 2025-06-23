package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	baseURL        = "https://api.cloudflare.com/client/v4"
	cfRootCertURL  = "https://developers.cloudflare.com/ssl/static/origin_ca_rsa_root.pem"
	colorRed       = "\033[0;31m"
	colorGreen     = "\033[0;32m"
	colorYellow    = "\033[1;33m"
	colorReset     = "\033[0m"
)

type Config struct {
	Domain       string
	APIToken     string
	ZoneID       string
	K8sNamespace string
	K8sSecretName string
	CertFile     string
	KeyFile      string
}

type CloudflareAPI struct {
	config *Config
	client *http.Client
}

// Response structures
type CFResponse struct {
	Success bool        `json:"success"`
	Errors  []CFError   `json:"errors"`
	Result  interface{} `json:"result"`
}

type CFError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Certificate struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

func main() {
	// Configuration
	config := &Config{
		Domain:        "kantoliana.gr",
		APIToken:      "your_cloudflare_api_token_here",
		ZoneID:        "your_zone_id_here", // Leave empty to auto-fetch
		K8sNamespace:  "default",
		K8sSecretName: "kantoliana-client-cloudflare-tls",
		CertFile:      "kantoliana-cert.crt",
		KeyFile:       "kantoliana-private.key",
	}

	// Validate configuration
	if config.APIToken == "your_cloudflare_api_token_here" {
		printError("Please set your Cloudflare API token in the configuration")
		os.Exit(1)
	}

	// Create API client
	api := &CloudflareAPI{
		config: config,
		client: &http.Client{},
	}

	printInfo(fmt.Sprintf("Starting Cloudflare domain setup automation for %s", config.Domain))

	// Get Zone ID if not provided
	if config.ZoneID == "" || config.ZoneID == "your_zone_id_here" {
		if err := api.getZoneID(); err != nil {
			printError(fmt.Sprintf("Failed to get zone ID: %v", err))
			os.Exit(1)
		}
	}

	// Execute all steps
	steps := []struct {
		name string
		fn   func() error
	}{
		{"Set SSL/TLS mode to Full (strict)", api.setSSLMode},
		{"Create Origin Server Certificate", api.createOriginCertificate},
		{"Append Cloudflare Root Certificate", api.appendRootCertificate},
		{"Enable Always Use HTTPS", api.enableAlwaysHTTPS},
		{"Create Page Rule for www redirect", api.createPageRule},
		{"Create Kubernetes TLS Secret", api.createK8sSecret},
	}

	for _, step := range steps {
		printInfo(step.name + "...")
		if err := step.fn(); err != nil {
			printError(fmt.Sprintf("%s failed: %v", step.name, err))
			os.Exit(1)
		}
		printSuccess(step.name + " completed successfully")
	}

	printSuccess("All steps completed successfully!")
	printInfo(fmt.Sprintf("Certificate files created: %s, %s", config.CertFile, config.KeyFile))
}

// Helper functions for colored output
func printInfo(msg string) {
	fmt.Printf("%s[INFO]%s %s\n", colorGreen, colorReset, msg)
}

func printError(msg string) {
	fmt.Printf("%s[ERROR]%s %s\n", colorRed, colorReset, msg)
}

func printWarning(msg string) {
	fmt.Printf("%s[WARNING]%s %s\n", colorYellow, colorReset, msg)
}

func printSuccess(msg string) {
	fmt.Printf("%s[SUCCESS]%s %s\n", colorGreen, colorReset, msg)
}

// API request helper
func (api *CloudflareAPI) makeRequest(method, endpoint string, body interface{}) (*CFResponse, error) {
	url := baseURL + endpoint
	
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

	req.Header.Set("Authorization", "Bearer "+api.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
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
			return nil, fmt.Errorf("API error: %s", cfResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	return &cfResp, nil
}

// Get Zone ID for the domain
func (api *CloudflareAPI) getZoneID() error {
	resp, err := api.makeRequest("GET", "/zones?name="+api.config.Domain, nil)
	if err != nil {
		return err
	}

	// Parse zones from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	var zones []Zone
	if err := json.Unmarshal(resultBytes, &zones); err != nil {
		return fmt.Errorf("failed to parse zones: %w", err)
	}

	if len(zones) == 0 {
		return fmt.Errorf("no zone found for domain %s", api.config.Domain)
	}

	api.config.ZoneID = zones[0].ID
	printInfo(fmt.Sprintf("Zone ID found: %s", api.config.ZoneID))
	return nil
}

// Set SSL/TLS mode to Full (strict)
func (api *CloudflareAPI) setSSLMode() error {
	body := map[string]string{"value": "strict"}
	_, err := api.makeRequest("PATCH", 
		fmt.Sprintf("/zones/%s/settings/ssl", api.config.ZoneID), 
		body)
	return err
}

// Create Origin Server Certificate
func (api *CloudflareAPI) createOriginCertificate() error {
	body := map[string]interface{}{
		"hostnames":         []string{api.config.Domain, "*." + api.config.Domain},
		"requested_validity": 5475, // 15 years (maximum)
		"request_type":      "origin-rsa",
		"csr":              "",
	}

	resp, err := api.makeRequest("POST", "/certificates", body)
	if err != nil {
		return err
	}

	// Parse certificate from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(resultBytes, &cert); err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Save certificate and key to files
	if err := os.WriteFile(api.config.CertFile, []byte(cert.Certificate), 0644); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	if err := os.WriteFile(api.config.KeyFile, []byte(cert.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	printInfo(fmt.Sprintf("Certificate saved to %s", api.config.CertFile))
	printInfo(fmt.Sprintf("Private key saved to %s", api.config.KeyFile))
	return nil
}

// Append Cloudflare Root Certificate
func (api *CloudflareAPI) appendRootCertificate() error {
	// Download root certificate
	resp, err := http.Get(cfRootCertURL)
	if err != nil {
		return fmt.Errorf("failed to download root certificate: %w", err)
	}
	defer resp.Body.Close()

	rootCert, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read root certificate: %w", err)
	}

	// Append to existing certificate file
	f, err := os.OpenFile(api.config.CertFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open certificate file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(rootCert); err != nil {
		return fmt.Errorf("failed to append root certificate: %w", err)
	}

	return nil
}

// Enable Always Use HTTPS
func (api *CloudflareAPI) enableAlwaysHTTPS() error {
	body := map[string]string{"value": "on"}
	_, err := api.makeRequest("PATCH",
		fmt.Sprintf("/zones/%s/settings/always_use_https", api.config.ZoneID),
		body)
	return err
}

// Create Page Rule for www redirect
func (api *CloudflareAPI) createPageRule() error {
	body := map[string]interface{}{
		"targets": []map[string]interface{}{
			{
				"target": "url",
				"constraint": map[string]string{
					"operator": "matches",
					"value":    fmt.Sprintf("www.%s/*", api.config.Domain),
				},
			},
		},
		"actions": []map[string]interface{}{
			{
				"id": "forwarding_url",
				"value": map[string]interface{}{
					"url":         fmt.Sprintf("https://$1.%s/$2", api.config.Domain),
					"status_code": 301,
				},
			},
		},
		"priority": 1,
		"status": "active",
	}

	_, err := api.makeRequest("POST",
		fmt.Sprintf("/zones/%s/pagerules", api.config.ZoneID),
		body)
	return err
}

// Create Kubernetes TLS Secret
func (api *CloudflareAPI) createK8sSecret() error {
	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		printWarning("kubectl is not installed. Please run the following command manually:")
		fmt.Printf("\nkubectl create secret tls %s \\\n", api.config.K8sSecretName)
		fmt.Printf("  --cert=%s \\\n", api.config.CertFile)
		fmt.Printf("  --key=%s \\\n", api.config.KeyFile)
		fmt.Printf("  -n %s\n\n", api.config.K8sNamespace)
		return nil
	}

	// Create the secret
	cmd := exec.Command("kubectl", "create", "secret", "tls", api.config.K8sSecretName,
		"--cert="+api.config.CertFile,
		"--key="+api.config.KeyFile,
		"-n", api.config.K8sNamespace)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl command failed: %w\nOutput: %s", err, output)
	}

	return nil
}