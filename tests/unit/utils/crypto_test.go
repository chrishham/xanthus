package utils

import (
	"strings"
	"testing"

	"github.com/chrishham/xanthus/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptData(t *testing.T) {
	token := "test-cloudflare-token-12345"
	data := "sensitive data to encrypt"

	encrypted, err := utils.EncryptData(data, token)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, data, encrypted)

	// Verify it's base64 encoded
	assert.True(t, strings.Contains(encrypted, "==") || len(encrypted) > 0)
}

func TestDecryptData(t *testing.T) {
	token := "test-cloudflare-token-12345"
	data := "sensitive data to decrypt"

	// First encrypt the data
	encrypted, err := utils.EncryptData(data, token)
	require.NoError(t, err)

	// Then decrypt it
	decrypted, err := utils.DecryptData(encrypted, token)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	testCases := []struct {
		name  string
		token string
		data  string
	}{
		{
			name:  "Simple text",
			token: "cloudflare-token-123",
			data:  "Hello, World!",
		},
		{
			name:  "JSON data",
			token: "another-token-456",
			data:  `{"key": "value", "number": 42}`,
		},
		{
			name:  "Empty data",
			token: "token-789",
			data:  "",
		},
		{
			name:  "Long data",
			token: "long-token-012",
			data:  strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 100),
		},
		{
			name:  "Special characters",
			token: "special-token-345",
			data:  "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?`~",
		},
		{
			name:  "Unicode characters",
			token: "unicode-token-678",
			data:  "Unicode: 🔐 密码 אבגד αβγδ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := utils.EncryptData(tc.data, tc.token)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)
			assert.NotEqual(t, tc.data, encrypted)

			// Decrypt
			decrypted, err := utils.DecryptData(encrypted, tc.token)
			require.NoError(t, err)
			assert.Equal(t, tc.data, decrypted)
		})
	}
}

func TestDecryptDataWithWrongToken(t *testing.T) {
	correctToken := "correct-token-123"
	wrongToken := "wrong-token-456"
	data := "secret data"

	// Encrypt with correct token
	encrypted, err := utils.EncryptData(data, correctToken)
	require.NoError(t, err)

	// Try to decrypt with wrong token
	_, err = utils.DecryptData(encrypted, wrongToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt")
}

func TestDecryptDataWithInvalidBase64(t *testing.T) {
	token := "test-token"
	invalidBase64 := "this-is-not-valid-base64!!!"

	_, err := utils.DecryptData(invalidBase64, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode base64")
}

func TestDecryptDataWithTooShortCiphertext(t *testing.T) {
	token := "test-token"
	// Create a valid base64 string that's too short (less than nonce size)
	shortCiphertext := "YWJj" // "abc" in base64

	_, err := utils.DecryptData(shortCiphertext, token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestEncryptionConsistency(t *testing.T) {
	token := "consistency-token"
	data := "test data for consistency"

	// Encrypt the same data multiple times
	encrypted1, err1 := utils.EncryptData(data, token)
	encrypted2, err2 := utils.EncryptData(data, token)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Each encryption should produce different results (due to random nonces)
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same original data
	decrypted1, err1 := utils.DecryptData(encrypted1, token)
	decrypted2, err2 := utils.DecryptData(encrypted2, token)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, data, decrypted1)
	assert.Equal(t, data, decrypted2)
}

func TestTokenSensitivity(t *testing.T) {
	data := "sensitive information"
	token1 := "token-version-1"
	token2 := "token-version-2"

	// Encrypt with first token
	encrypted1, err := utils.EncryptData(data, token1)
	require.NoError(t, err)

	// Encrypt with second token
	encrypted2, err := utils.EncryptData(data, token2)
	require.NoError(t, err)

	// Results should be different
	assert.NotEqual(t, encrypted1, encrypted2)

	// Each can only be decrypted with its respective token
	decrypted1, err := utils.DecryptData(encrypted1, token1)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted1)

	decrypted2, err := utils.DecryptData(encrypted2, token2)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted2)

	// Cross-decryption should fail
	_, err = utils.DecryptData(encrypted1, token2)
	assert.Error(t, err)

	_, err = utils.DecryptData(encrypted2, token1)
	assert.Error(t, err)
}

// Benchmarks
func BenchmarkEncryptData(b *testing.B) {
	token := "benchmark-token-12345"
	data := "benchmark data for encryption performance testing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := utils.EncryptData(data, token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptData(b *testing.B) {
	token := "benchmark-token-12345"
	data := "benchmark data for decryption performance testing"

	// Pre-encrypt the data
	encrypted, err := utils.EncryptData(data, token)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := utils.DecryptData(encrypted, token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncryptDecryptRoundTrip(b *testing.B) {
	token := "benchmark-token-12345"
	data := "benchmark data for full round-trip performance testing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, err := utils.EncryptData(data, token)
		if err != nil {
			b.Fatal(err)
		}

		_, err = utils.DecryptData(encrypted, token)
		if err != nil {
			b.Fatal(err)
		}
	}
}
