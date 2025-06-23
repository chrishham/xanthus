package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// KVService handles Cloudflare KV storage operations
type KVService struct {
	client *http.Client
}

// NewKVService creates a new KV service instance
func NewKVService() *KVService {
	return &KVService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
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
	Errors  []CFError     `json:"errors"`
}

// GetXanthusNamespaceID retrieves the Xanthus namespace ID
func (kvs *KVService) GetXanthusNamespaceID(token, accountID string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces", CloudflareBaseURL, accountID), nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := kvs.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	var kvResp KVNamespaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&kvResp); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
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

// PutValue stores a value in Cloudflare KV
func (kvs *KVService) PutValue(token, accountID, key string, value interface{}) error {
	// Get the Xanthus namespace ID
	namespaceID, err := kvs.GetXanthusNamespaceID(token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get namespace ID: %w", err)
	}

	// Marshal value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s", 
		CloudflareBaseURL, accountID, namespaceID, key)
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(valueBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := kvs.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var kvResp CFResponse
	if err := json.NewDecoder(resp.Body).Decode(&kvResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !kvResp.Success {
		return fmt.Errorf("KV put failed: %v", kvResp.Errors)
	}

	return nil
}

// GetValue retrieves a value from Cloudflare KV
func (kvs *KVService) GetValue(token, accountID, key string, result interface{}) error {
	// Get the Xanthus namespace ID
	namespaceID, err := kvs.GetXanthusNamespaceID(token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get namespace ID: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s", 
		CloudflareBaseURL, accountID, namespaceID, key)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kvs.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("key not found in KV")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("KV API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// DeleteValue deletes a value from Cloudflare KV
func (kvs *KVService) DeleteValue(token, accountID, key string) error {
	// Get the Xanthus namespace ID
	namespaceID, err := kvs.GetXanthusNamespaceID(token, accountID)
	if err != nil {
		return fmt.Errorf("failed to get namespace ID: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/values/%s", 
		CloudflareBaseURL, accountID, namespaceID, key)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kvs.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		return fmt.Errorf("KV API returned status %d", resp.StatusCode)
	}

	return nil
}

// StoreDomainSSLConfig stores SSL configuration for a domain in KV
func (kvs *KVService) StoreDomainSSLConfig(token, accountID string, config *DomainSSLConfig) error {
	key := fmt.Sprintf("domain:%s:ssl_config", config.Domain)
	return kvs.PutValue(token, accountID, key, config)
}

// GetDomainSSLConfig retrieves SSL configuration for a domain from KV
func (kvs *KVService) GetDomainSSLConfig(token, accountID, domain string) (*DomainSSLConfig, error) {
	key := fmt.Sprintf("domain:%s:ssl_config", domain)
	var config DomainSSLConfig
	if err := kvs.GetValue(token, accountID, key, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ListDomainSSLConfigs retrieves all domain SSL configurations
func (kvs *KVService) ListDomainSSLConfigs(token, accountID string) (map[string]*DomainSSLConfig, error) {
	// Get the Xanthus namespace ID
	namespaceID, err := kvs.GetXanthusNamespaceID(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace ID: %w", err)
	}

	// List all keys with domain:*:ssl_config prefix
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/keys?prefix=domain:", 
		CloudflareBaseURL, accountID, namespaceID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kvs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var keysResp struct {
		Success bool `json:"success"`
		Result  []struct {
			Name string `json:"name"`
		} `json:"result"`
		Errors []CFError `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keysResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !keysResp.Success {
		return nil, fmt.Errorf("KV API failed: %v", keysResp.Errors)
	}

	configs := make(map[string]*DomainSSLConfig)
	
	// Fetch each SSL config
	for _, key := range keysResp.Result {
		if len(key.Name) > 20 && key.Name[len(key.Name)-11:] == ":ssl_config" {
			// Extract domain from key format: domain:example.com:ssl_config
			parts := key.Name[7:] // Remove "domain:" prefix
			domain := parts[:len(parts)-11] // Remove ":ssl_config" suffix
			
			var config DomainSSLConfig
			if err := kvs.GetValue(token, accountID, key.Name, &config); err == nil {
				configs[domain] = &config
			}
		}
	}

	return configs, nil
}

// DeleteDomainSSLConfig removes SSL configuration for a domain from KV
func (kvs *KVService) DeleteDomainSSLConfig(token, accountID, domain string) error {
	key := fmt.Sprintf("domain:%s:ssl_config", domain)
	return kvs.DeleteValue(token, accountID, key)
}