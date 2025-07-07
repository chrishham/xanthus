package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateSecretKey(t *testing.T) {
	key, err := GenerateSecretKey()
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestJWTService_GenerateTokenPair(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	accessToken, refreshToken, err := jwtService.GenerateTokenPair(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEqual(t, accessToken, refreshToken)
}

func TestJWTService_ValidateToken(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	token, err := jwtService.GenerateAccessToken(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)

	claims, err := jwtService.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, accountID, claims.AccountID)
	assert.Equal(t, namespaceID, claims.NamespaceID)
	assert.Equal(t, cfToken, claims.CFToken)
}

func TestJWTService_ValidateToken_InvalidToken(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	_, err = jwtService.ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTService_ValidateToken_ExpiredToken(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, -1*time.Hour, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	token, err := jwtService.GenerateAccessToken(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)

	_, err = jwtService.ValidateToken(token)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestJWTService_RefreshToken(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	refreshToken, err := jwtService.GenerateRefreshToken(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)

	// Wait a bit to ensure different timestamps
	time.Sleep(1 * time.Second)

	newAccessToken, newRefreshToken, err := jwtService.RefreshToken(refreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)
	assert.NotEmpty(t, newRefreshToken)
	assert.NotEqual(t, refreshToken, newRefreshToken)
}

func TestJWTService_ExtractUserID(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	token, err := jwtService.GenerateAccessToken(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)

	extractedUserID, err := jwtService.ExtractUserID(token)
	require.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestJWTService_ExtractAccountInfo(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	token, err := jwtService.GenerateAccessToken(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)

	extractedAccountID, extractedNamespaceID, err := jwtService.ExtractAccountInfo(token)
	require.NoError(t, err)
	assert.Equal(t, accountID, extractedAccountID)
	assert.Equal(t, namespaceID, extractedNamespaceID)
}

func TestJWTService_ExtractCFToken(t *testing.T) {
	secretKey, err := GenerateSecretKey()
	require.NoError(t, err)

	jwtService := NewJWTService(secretKey, 15*time.Minute, 7*24*time.Hour)

	userID := "test-user"
	accountID := "test-account"
	namespaceID := "test-namespace"
	cfToken := "test-cf-token"

	token, err := jwtService.GenerateAccessToken(userID, accountID, namespaceID, cfToken)
	require.NoError(t, err)

	extractedCFToken, err := jwtService.ExtractCFToken(token)
	require.NoError(t, err)
	assert.Equal(t, cfToken, extractedCFToken)
}
