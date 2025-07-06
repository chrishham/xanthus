package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// KVService handles Cloudflare KV storage operations
type KVService struct {
	client *http.Client
	// Cache for namespace ID to avoid repeated API calls
	namespaceIDCache map[string]string
	cacheService     *CacheService
}

// NewKVService creates a new KV service instance
func NewKVService() *KVService {
	return &KVService{
		client:           &http.Client{Timeout: 10 * time.Second}, // Reduced timeout
		namespaceIDCache: make(map[string]string),
		cacheService:     NewCacheService(),
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
	// Try to get cached namespace ID first - use accountID for isolation
	cacheKey := "namespace_id:" + accountID
	namespaceID := ""

	if cached, exists := kvs.cacheService.Get(cacheKey); exists {
		namespaceID = cached.(string)
	} else {
		// Fallback to API call
		var err error
		namespaceID, err = kvs.GetXanthusNamespaceID(token, accountID)
		if err != nil {
			return fmt.Errorf("failed to get namespace ID: %w", err)
		}
		// Cache for 10 minutes
		kvs.cacheService.Set(cacheKey, namespaceID, 10*time.Minute)
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
	// Try to get cached namespace ID first - use accountID for isolation
	cacheKey := "namespace_id:" + accountID
	namespaceID := ""

	if cached, exists := kvs.cacheService.Get(cacheKey); exists {
		namespaceID = cached.(string)
	} else {
		// Fallback to API call
		var err error
		namespaceID, err = kvs.GetXanthusNamespaceID(token, accountID)
		if err != nil {
			return fmt.Errorf("failed to get namespace ID: %w", err)
		}
		// Cache for 10 minutes
		kvs.cacheService.Set(cacheKey, namespaceID, 10*time.Minute)
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
	// Get the Xanthus namespace ID (with caching)
	cacheKey := "legacy_ns:" + accountID
	namespaceID, exists := kvs.namespaceIDCache[cacheKey]

	if !exists {
		var err error
		namespaceID, err = kvs.GetXanthusNamespaceID(token, accountID)
		if err != nil {
			return fmt.Errorf("failed to get namespace ID: %w", err)
		}
		kvs.namespaceIDCache[cacheKey] = namespaceID
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
	// Get the Xanthus namespace ID (with caching)
	cacheKey := "legacy_ns:" + accountID // Use first 10 chars of token as cache key
	namespaceID, exists := kvs.namespaceIDCache[cacheKey]

	if !exists {
		var err error
		namespaceID, err = kvs.GetXanthusNamespaceID(token, accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get namespace ID: %w", err)
		}
		kvs.namespaceIDCache[cacheKey] = namespaceID
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

	// Filter SSL config keys and extract domains
	type sslKeyInfo struct {
		keyName string
		domain  string
	}

	var sslKeys []sslKeyInfo
	for _, key := range keysResp.Result {
		if len(key.Name) > 20 && key.Name[len(key.Name)-11:] == ":ssl_config" {
			// Extract domain from key format: domain:example.com:ssl_config
			parts := key.Name[7:]           // Remove "domain:" prefix
			domain := parts[:len(parts)-11] // Remove ":ssl_config" suffix
			sslKeys = append(sslKeys, sslKeyInfo{keyName: key.Name, domain: domain})
		}
	}

	// Parallel fetch of SSL configs using goroutines
	configs := make(map[string]*DomainSSLConfig)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent requests to avoid overwhelming Cloudflare KV API
	maxConcurrency := 5
	semaphore := make(chan struct{}, maxConcurrency)

	for _, sslKey := range sslKeys {
		wg.Add(1)
		go func(keyName, domain string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var config DomainSSLConfig
			if err := kvs.GetValue(token, accountID, keyName, &config); err == nil {
				// Thread-safe map write
				mu.Lock()
				configs[domain] = &config
				mu.Unlock()
			}
		}(sslKey.keyName, sslKey.domain)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return configs, nil
}

// DeleteDomainSSLConfig removes SSL configuration for a domain from KV
func (kvs *KVService) DeleteDomainSSLConfig(token, accountID, domain string) error {
	key := fmt.Sprintf("domain:%s:ssl_config", domain)
	return kvs.DeleteValue(token, accountID, key)
}

// VPSConfig represents VPS configuration stored in KV
type VPSConfig struct {
	ServerID           int     `json:"server_id"`
	Name               string  `json:"name"`
	ServerType         string  `json:"server_type"`
	Location           string  `json:"location"`
	PublicIPv4         string  `json:"public_ipv4"`
	CreatedAt          string  `json:"created_at"`
	SSHKeyName         string  `json:"ssh_key_name"`
	SSHUser            string  `json:"ssh_user"`
	SSHPort            int     `json:"ssh_port"`
	HourlyRate         float64 `json:"hourly_rate"`          // EUR per hour
	MonthlyRate        float64 `json:"monthly_rate"`         // EUR per month (including IPv4)
	Timezone           string  `json:"timezone"`             // e.g., "Europe/Berlin", "UTC"
	Provider           string  `json:"provider"`             // VPS provider (e.g., "Hetzner", "OCI", "AWS", "DigitalOcean")
	ProviderInstanceID string  `json:"provider_instance_id"` // Provider-specific instance ID (e.g., OCI instance OCID)
	// OCI-specific fields
	OCPU         float32 `json:"ocpu,omitempty"`         // Number of OCPUs (for OCI flexible shapes)
	Memory       float32 `json:"memory,omitempty"`       // Memory in GB (for OCI flexible shapes)
	Region       string  `json:"region,omitempty"`       // Cloud provider region (e.g., "eu-zurich-1")
	Architecture string  `json:"architecture,omitempty"` // CPU architecture (e.g., "ARM64", "x86_64")
}

// StoreVPSConfig stores VPS configuration in KV
func (kvs *KVService) StoreVPSConfig(token, accountID string, config *VPSConfig) error {
	key := fmt.Sprintf("vps:%d:config", config.ServerID)
	return kvs.PutValue(token, accountID, key, config)
}

// GetVPSConfig retrieves VPS configuration from KV
func (kvs *KVService) GetVPSConfig(token, accountID string, serverID int) (*VPSConfig, error) {
	key := fmt.Sprintf("vps:%d:config", serverID)
	var config VPSConfig
	if err := kvs.GetValue(token, accountID, key, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ListVPSConfigs retrieves all VPS configurations
func (kvs *KVService) ListVPSConfigs(token, accountID string) (map[int]*VPSConfig, error) {
	// Get the Xanthus namespace ID (with caching)
	cacheKey := "legacy_ns:" + accountID // Use first 10 chars of token as cache key
	namespaceID, exists := kvs.namespaceIDCache[cacheKey]

	if !exists {
		var err error
		namespaceID, err = kvs.GetXanthusNamespaceID(token, accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get namespace ID: %w", err)
		}
		kvs.namespaceIDCache[cacheKey] = namespaceID
	}

	// List all keys with vps:*:config prefix
	url := fmt.Sprintf("%s/accounts/%s/storage/kv/namespaces/%s/keys?prefix=vps:",
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

	// Filter VPS config keys
	var vpsKeys []string
	for _, key := range keysResp.Result {
		if len(key.Name) > 8 && key.Name[len(key.Name)-7:] == ":config" {
			vpsKeys = append(vpsKeys, key.Name)
		}
	}

	// Parallel fetch of VPS configs using goroutines
	configs := make(map[int]*VPSConfig)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrent requests to avoid overwhelming Cloudflare KV API
	maxConcurrency := 5
	semaphore := make(chan struct{}, maxConcurrency)

	for _, keyName := range vpsKeys {
		wg.Add(1)
		go func(keyName string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var config VPSConfig
			if err := kvs.GetValue(token, accountID, keyName, &config); err == nil {
				// Thread-safe map write
				mu.Lock()
				configs[config.ServerID] = &config
				mu.Unlock()
			}
		}(keyName)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return configs, nil
}

// DeleteVPSConfig removes VPS configuration from KV
func (kvs *KVService) DeleteVPSConfig(token, accountID string, serverID int) error {
	key := fmt.Sprintf("vps:%d:config", serverID)
	return kvs.DeleteValue(token, accountID, key)
}

// UpdateVPSConfig updates specific fields in VPS configuration
func (kvs *KVService) UpdateVPSConfig(token, accountID string, serverID int, updates map[string]interface{}) error {
	// Get existing config
	config, err := kvs.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get existing config: %w", err)
	}

	// Apply updates
	for field, value := range updates {
		switch field {
		case "public_ipv4":
			if ip, ok := value.(string); ok {
				config.PublicIPv4 = ip
			}
		case "ssh_key_name":
			if key, ok := value.(string); ok {
				config.SSHKeyName = key
			}
		case "ssh_user":
			if user, ok := value.(string); ok {
				config.SSHUser = user
			}
		case "ssh_port":
			if port, ok := value.(int); ok {
				config.SSHPort = port
			}
		}
	}

	// Store updated config
	return kvs.StoreVPSConfig(token, accountID, config)
}

// CalculateVPSCosts calculates accumulated cost for a VPS based on creation time and current time
func (kvs *KVService) CalculateVPSCosts(config *VPSConfig) (float64, error) {
	if config.HourlyRate <= 0 {
		return 0, fmt.Errorf("hourly rate not set for VPS %d", config.ServerID)
	}

	// Parse creation time
	createdAt, err := time.Parse(time.RFC3339, config.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("failed to parse creation time: %w", err)
	}

	// Calculate hours since creation
	now := time.Now().UTC()
	hoursSinceCreation := now.Sub(createdAt).Hours()

	// Calculate accumulated cost (hourly rate includes IPv4 cost)
	accumulatedCost := hoursSinceCreation * config.HourlyRate

	return accumulatedCost, nil
}
