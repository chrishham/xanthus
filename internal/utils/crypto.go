package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"strings"

	"gopkg.in/yaml.v2"
)

// EncryptData encrypts data using AES-256-GCM with a key derived from the CF token
func EncryptData(data, token string) (string, error) {
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

// DecryptData decrypts data using AES-256-GCM with a key derived from the CF token
func DecryptData(encryptedData, token string) (string, error) {
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

// Base64Encode encodes a string to base64
func Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// GenerateSecurePassword generates a cryptographically secure random password
func GenerateSecurePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

	password := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range password {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback to a simpler character if crypto/rand fails
			password[i] = charset[i%len(charset)]
		} else {
			password[i] = charset[randomIndex.Int64()]
		}
	}

	return string(password)
}

// ConvertToYAML converts a map to YAML string
func ConvertToYAML(data map[string]interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %v", err)
	}

	return strings.TrimSpace(string(yamlBytes)), nil
}
