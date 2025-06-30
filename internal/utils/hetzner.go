package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/chrishham/xanthus/internal/models"
)

// Temporary in-memory cache for validated Hetzner API keys
// This allows immediate use of validated keys without waiting for KV store propagation
var (
	tempHetznerKeys = make(map[string]string)
	tempKeysMutex   sync.RWMutex
)

// SetTempHetznerKey stores a Hetzner API key temporarily in memory
func SetTempHetznerKey(accountID, apiKey string) {
	tempKeysMutex.Lock()
	defer tempKeysMutex.Unlock()
	tempHetznerKeys[accountID] = apiKey
}

// GetTempHetznerKey retrieves a temporarily stored Hetzner API key
func GetTempHetznerKey(accountID string) (string, bool) {
	tempKeysMutex.RLock()
	defer tempKeysMutex.RUnlock()
	key, exists := tempHetznerKeys[accountID]
	return key, exists
}

// ClearTempHetznerKey removes a temporarily stored Hetzner API key
func ClearTempHetznerKey(accountID string) {
	tempKeysMutex.Lock()
	defer tempKeysMutex.Unlock()
	delete(tempHetznerKeys, accountID)
}

// ValidateHetznerAPIKey validates a Hetzner Cloud API key by making a test API call
func ValidateHetznerAPIKey(apiKey string) bool {
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

// GetHetznerAPIKey retrieves and decrypts the Hetzner API key, checking temporary cache first
func GetHetznerAPIKey(token, accountID string) (string, error) {
	log.Printf("GetHetznerAPIKey: Attempting to retrieve key for account %s", accountID)
	
	// First, check temporary cache for recently validated keys
	if tempKey, exists := GetTempHetznerKey(accountID); exists {
		log.Printf("GetHetznerAPIKey: Found key in temporary cache for account %s", accountID)
		return tempKey, nil
	}
	
	log.Printf("GetHetznerAPIKey: Key not in temporary cache, checking KV store for account %s", accountID)
	
	client := &http.Client{Timeout: 10 * time.Second}
	var encryptedKey string
	
	// Retry logic to handle Cloudflare KV eventual consistency
	maxRetries := 3
	retryDelay := 2 * time.Second
	
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("GetHetznerAPIKey: KV attempt %d/%d for account %s", attempt, maxRetries, accountID)
		
		if err := GetKVValue(client, token, accountID, "config:hetzner:api_key", &encryptedKey); err != nil {
			lastErr = err
			log.Printf("GetHetznerAPIKey: KV retrieval attempt %d failed for account %s: %v", attempt, accountID, err)
			
			if attempt < maxRetries {
				log.Printf("GetHetznerAPIKey: Waiting %v before retry for account %s", retryDelay, accountID)
				time.Sleep(retryDelay)
				continue
			}
		} else {
			log.Printf("GetHetznerAPIKey: Successfully retrieved encrypted key for account %s on attempt %d, attempting decryption", accountID, attempt)
			
			decryptedKey, err := DecryptData(encryptedKey, token)
			if err != nil {
				log.Printf("GetHetznerAPIKey: Decryption failed for account %s: %v", accountID, err)
				return "", fmt.Errorf("failed to decrypt Hetzner API key: %v", err)
			}

			log.Printf("GetHetznerAPIKey: Successfully decrypted key for account %s", accountID)
			return decryptedKey, nil
		}
	}
	
	log.Printf("GetHetznerAPIKey: All retry attempts failed for account %s", accountID)
	return "", fmt.Errorf("failed to get Hetzner API key after %d attempts: %v", maxRetries, lastErr)
}

// FetchHetznerLocations fetches available datacenter locations from Hetzner API
func FetchHetznerLocations(apiKey string) ([]models.HetznerLocation, error) {
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

	var locationsResp models.HetznerLocationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&locationsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return locationsResp.Locations, nil
}

// FetchHetznerServerTypes fetches available server types from Hetzner API
func FetchHetznerServerTypes(apiKey string) ([]models.HetznerServerType, error) {
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

	var serverTypesResp models.HetznerServerTypesResponse
	if err := json.NewDecoder(resp.Body).Decode(&serverTypesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return serverTypesResp.ServerTypes, nil
}

// FetchServerAvailability fetches real-time server availability for all datacenters
func FetchServerAvailability(apiKey string) (map[string]map[int]bool, error) {
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

	var datacentersResp models.HetznerDatacentersResponse
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

// FilterSharedVCPUServers filters server types to only include shared vCPU instances
func FilterSharedVCPUServers(serverTypes []models.HetznerServerType) []models.HetznerServerType {
	var sharedServers []models.HetznerServerType

	for _, server := range serverTypes {
		// Filter for shared vCPU types (typically start with "cpx" or "cx")
		if server.CPUType == "shared" {
			sharedServers = append(sharedServers, server)
		}
	}

	return sharedServers
}

// SortServerTypesByPriceDesc sorts server types by price in descending order (highest first)
func SortServerTypesByPriceDesc(serverTypes []models.HetznerServerType) {
	n := len(serverTypes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			priceJ := GetServerTypeMonthlyPrice(serverTypes[j])
			priceJ1 := GetServerTypeMonthlyPrice(serverTypes[j+1])

			// For descending order: if current price < next price, swap
			if priceJ < priceJ1 {
				serverTypes[j], serverTypes[j+1] = serverTypes[j+1], serverTypes[j]
			}
		}
	}
}

// SortServerTypesByPriceAsc sorts server types by price in ascending order (lowest first)
func SortServerTypesByPriceAsc(serverTypes []models.HetznerServerType) {
	n := len(serverTypes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			priceJ := GetServerTypeMonthlyPrice(serverTypes[j])
			priceJ1 := GetServerTypeMonthlyPrice(serverTypes[j+1])

			// For ascending order: if current price > next price, swap
			if priceJ > priceJ1 {
				serverTypes[j], serverTypes[j+1] = serverTypes[j+1], serverTypes[j]
			}
		}
	}
}

// GetServerTypeMonthlyPrice gets the monthly price for a server type (uses first available location)
func GetServerTypeMonthlyPrice(serverType models.HetznerServerType) float64 {
	if len(serverType.Prices) == 0 {
		return 0.0
	}

	// Use the first available price location
	priceStr := serverType.Prices[0].PriceMonthly.Gross

	// Parse price string - it might be in format like "4.90" or "4.90 EUR"
	// Remove any non-numeric characters except decimal point
	cleanPrice := ""
	foundDecimal := false
	for _, char := range priceStr {
		if char >= '0' && char <= '9' {
			cleanPrice += string(char)
		} else if char == '.' && !foundDecimal {
			cleanPrice += string(char)
			foundDecimal = true
		}
	}

	if cleanPrice == "" {
		return 0.0
	}

	var priceFloat float64
	if _, err := fmt.Sscanf(cleanPrice, "%f", &priceFloat); err != nil {
		return 0.0
	}

	return priceFloat
}

// SortServerTypesByCPUDesc sorts server types by CPU cores in descending order
func SortServerTypesByCPUDesc(serverTypes []models.HetznerServerType) {
	n := len(serverTypes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if serverTypes[j].Cores < serverTypes[j+1].Cores {
				serverTypes[j], serverTypes[j+1] = serverTypes[j+1], serverTypes[j]
			}
		}
	}
}

// SortServerTypesByCPUAsc sorts server types by CPU cores in ascending order
func SortServerTypesByCPUAsc(serverTypes []models.HetznerServerType) {
	n := len(serverTypes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if serverTypes[j].Cores > serverTypes[j+1].Cores {
				serverTypes[j], serverTypes[j+1] = serverTypes[j+1], serverTypes[j]
			}
		}
	}
}

// SortServerTypesByMemoryDesc sorts server types by memory in descending order
func SortServerTypesByMemoryDesc(serverTypes []models.HetznerServerType) {
	n := len(serverTypes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if serverTypes[j].Memory < serverTypes[j+1].Memory {
				serverTypes[j], serverTypes[j+1] = serverTypes[j+1], serverTypes[j]
			}
		}
	}
}

// SortServerTypesByMemoryAsc sorts server types by memory in ascending order
func SortServerTypesByMemoryAsc(serverTypes []models.HetznerServerType) {
	n := len(serverTypes)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if serverTypes[j].Memory > serverTypes[j+1].Memory {
				serverTypes[j], serverTypes[j+1] = serverTypes[j+1], serverTypes[j]
			}
		}
	}
}
