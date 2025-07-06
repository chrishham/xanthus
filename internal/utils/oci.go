package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OCICredentials represents OCI authentication credentials
type OCICredentials struct {
	Tenancy     string `json:"tenancy"`
	User        string `json:"user"`
	Region      string `json:"region"`
	Fingerprint string `json:"fingerprint"`
	PrivateKey  string `json:"private_key"`
}

// DecodeOCIAuthToken decodes a base64-encoded OCI auth token
func DecodeOCIAuthToken(token string) (*OCICredentials, error) {
	if token == "" {
		return nil, fmt.Errorf("OCI auth token is empty")
	}

	// Decode base64
	jsonData, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OCI auth token: %w", err)
	}

	// Parse JSON
	var creds OCICredentials
	if err := json.Unmarshal(jsonData, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse OCI credentials: %w", err)
	}

	// Validate required fields
	if creds.Tenancy == "" {
		return nil, fmt.Errorf("tenancy OCID is required")
	}
	if creds.User == "" {
		return nil, fmt.Errorf("user OCID is required")
	}
	if creds.Region == "" {
		return nil, fmt.Errorf("region is required")
	}
	if creds.Fingerprint == "" {
		return nil, fmt.Errorf("fingerprint is required")
	}
	if creds.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required")
	}

	return &creds, nil
}

// EncodeOCIAuthToken encodes OCI credentials into a base64 auth token
func EncodeOCIAuthToken(creds *OCICredentials) (string, error) {
	if creds == nil {
		return "", fmt.Errorf("OCI credentials cannot be nil")
	}

	// Validate required fields
	if creds.Tenancy == "" {
		return "", fmt.Errorf("tenancy OCID is required")
	}
	if creds.User == "" {
		return "", fmt.Errorf("user OCID is required")
	}
	if creds.Region == "" {
		return "", fmt.Errorf("region is required")
	}
	if creds.Fingerprint == "" {
		return "", fmt.Errorf("fingerprint is required")
	}
	if creds.PrivateKey == "" {
		return "", fmt.Errorf("private key is required")
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(creds)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OCI credentials: %w", err)
	}

	// Encode to base64
	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// GetOCIAuthToken retrieves OCI auth token from Cloudflare KV
func GetOCIAuthToken(token, accountID string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("cloudflare token is required")
	}
	if accountID == "" {
		return "", fmt.Errorf("cloudflare account ID is required")
	}

	// Use the same pattern as GetHetznerAPIKey
	client := &http.Client{Timeout: 10 * time.Second}
	var ociToken string
	err := GetKVValue(client, token, accountID, "oci_token", &ociToken)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve OCI auth token: %w", err)
	}

	if ociToken == "" {
		return "", fmt.Errorf("OCI auth token not found in KV store")
	}

	return ociToken, nil
}

// SetOCIAuthToken stores OCI auth token in Cloudflare KV
func SetOCIAuthToken(token, accountID, ociToken string) error {
	if token == "" {
		return fmt.Errorf("cloudflare token is required")
	}
	if accountID == "" {
		return fmt.Errorf("cloudflare account ID is required")
	}
	if ociToken == "" {
		return fmt.Errorf("OCI auth token is required")
	}

	// Validate the token before storing
	_, err := DecodeOCIAuthToken(ociToken)
	if err != nil {
		return fmt.Errorf("invalid OCI auth token: %w", err)
	}

	// Store in KV using the same pattern as Hetzner
	client := &http.Client{Timeout: 10 * time.Second}
	err = PutKVValue(client, token, accountID, "oci_token", ociToken)
	if err != nil {
		return fmt.Errorf("failed to store OCI auth token: %w", err)
	}

	return nil
}

// ValidateOCICredentials validates OCI credentials format
func ValidateOCICredentials(creds *OCICredentials) error {
	if creds == nil {
		return fmt.Errorf("OCI credentials cannot be nil")
	}

	// Validate Tenancy OCID format
	if !strings.HasPrefix(creds.Tenancy, "ocid1.tenancy.") {
		return fmt.Errorf("invalid tenancy OCID format")
	}

	// Validate User OCID format
	if !strings.HasPrefix(creds.User, "ocid1.user.") {
		return fmt.Errorf("invalid user OCID format")
	}

	// Validate region format (basic validation)
	if !strings.Contains(creds.Region, "-") {
		return fmt.Errorf("invalid region format")
	}

	// Validate fingerprint format (basic validation)
	if !strings.Contains(creds.Fingerprint, ":") {
		return fmt.Errorf("invalid fingerprint format")
	}

	// Validate private key format
	if !strings.Contains(creds.PrivateKey, "-----BEGIN") || !strings.Contains(creds.PrivateKey, "-----END") {
		return fmt.Errorf("invalid private key format")
	}

	return nil
}

// ValidateOCIToken validates an OCI auth token format and structure
func ValidateOCIToken(token string) error {
	_, err := DecodeOCIAuthToken(token)
	return err
}
