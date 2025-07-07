package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
