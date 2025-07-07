package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chrishham/xanthus/internal/models"
)

// VerifyCloudflareToken verifies the validity of a Cloudflare API token
func VerifyCloudflareToken(token string) bool {
	client := &http.Client{Timeout: 5 * time.Second}

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

	var cfResp models.CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		log.Printf("Error decoding response: %v", err)
		return false
	}

	return cfResp.Success
}

// CheckKVNamespaceExists checks if the "Xanthus" KV namespace exists
func CheckKVNamespaceExists(token string) (bool, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

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

	var kvNamespaceResp models.KVNamespaceResponse
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

// CreateKVNamespace creates the "Xanthus" KV namespace
func CreateKVNamespace(token, accountID string) error {
	client := &http.Client{Timeout: 5 * time.Second}

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

	var createResp models.CloudflareResponse
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

// PutKVValue stores a value in Cloudflare KV
func PutKVValue(client *http.Client, token, accountID, key string, value interface{}) error {
	// First, get the Xanthus namespace ID
	namespaceID, err := GetXanthusNamespaceID(client, token, accountID)
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

	var kvResp models.CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&kvResp); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if !kvResp.Success {
		return fmt.Errorf("KV put failed: %v", kvResp.Errors)
	}

	return nil
}

// GetXanthusNamespaceID retrieves the Xanthus namespace ID
func GetXanthusNamespaceID(client *http.Client, token, accountID string) (string, error) {
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

	var kvResp models.KVNamespaceResponse
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

// GetKVValue retrieves a value from Cloudflare KV
func GetKVValue(client *http.Client, token, accountID, key string, result interface{}) error {
	// Get the Xanthus namespace ID
	namespaceID, err := GetXanthusNamespaceID(client, token, accountID)
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

// FetchCloudflareDomains fetches all domain zones from Cloudflare
func FetchCloudflareDomains(token string) ([]models.CloudflareDomain, error) {
	client := &http.Client{Timeout: 8 * time.Second}

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

	var domainsResp models.CloudflareDomainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&domainsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !domainsResp.Success {
		return nil, fmt.Errorf("API call failed: %v", domainsResp.Errors)
	}

	return domainsResp.Result, nil
}
