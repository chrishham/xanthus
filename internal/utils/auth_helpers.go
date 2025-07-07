package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// AuthResult contains the authentication validation result
type AuthResult struct {
	Token     string
	AccountID string
	Valid     bool
	Error     error
}

// ValidateTokenAndGetAccount validates the Cloudflare token and returns account ID
// This extracts the common pattern used across VPS handlers
func ValidateTokenAndGetAccount(c *gin.Context) (*AuthResult, error) {
	token, err := c.Cookie("cf_token")
	if err != nil {
		return &AuthResult{Valid: false, Error: fmt.Errorf("missing token cookie")}, err
	}

	if !VerifyCloudflareToken(token) {
		return &AuthResult{Valid: false, Error: fmt.Errorf("invalid token")}, fmt.Errorf("invalid token")
	}

	_, accountID, err := CheckKVNamespaceExists(token)
	if err != nil {
		return &AuthResult{Valid: false, Error: fmt.Errorf("failed to get account ID: %w", err)}, err
	}

	return &AuthResult{
		Token:     token,
		AccountID: accountID,
		Valid:     true,
	}, nil
}

// ValidateTokenAndGetAccountJSON validates token and sends JSON error if invalid
// Returns true if valid, false if invalid (and sends error response)
func ValidateTokenAndGetAccountJSON(c *gin.Context) (token, accountID string, valid bool) {
	result, err := ValidateTokenAndGetAccount(c)
	if err != nil || !result.Valid {
		if result.Error.Error() == "missing token cookie" || result.Error.Error() == "invalid token" {
			JSONUnauthorized(c, "Invalid token")
		} else {
			JSONInternalServerError(c, "Failed to get account ID")
		}
		return "", "", false
	}
	return result.Token, result.AccountID, true
}

// ValidateTokenAndGetAccountHTML validates token and redirects if invalid (for HTML pages)
// Returns true if valid, false if invalid (and redirects to login)
func ValidateTokenAndGetAccountHTML(c *gin.Context) (token, accountID string, valid bool) {
	result, err := ValidateTokenAndGetAccount(c)
	if err != nil || !result.Valid {
		c.Redirect(302, "/login") // http.StatusTemporaryRedirect
		return "", "", false
	}
	return result.Token, result.AccountID, true
}

// ValidateJWTAndGetAccountJSON validates JWT token from context and sends JSON error if invalid
// This should be used for API endpoints protected by JWT middleware
// Returns true if valid, false if invalid (and sends error response)
func ValidateJWTAndGetAccountJSON(c *gin.Context) (token, accountID string, valid bool) {
	// Get values set by JWT middleware
	cfToken, tokenExists := c.Get("cf_token")
	accID, accountExists := c.Get("account_id")

	if !tokenExists || !accountExists {
		JSONUnauthorized(c, "Missing authentication context")
		return "", "", false
	}

	cfTokenStr, tokenOk := cfToken.(string)
	accountIDStr, accountOk := accID.(string)

	if !tokenOk || !accountOk || cfTokenStr == "" || accountIDStr == "" {
		JSONUnauthorized(c, "Invalid authentication context")
		return "", "", false
	}

	return cfTokenStr, accountIDStr, true
}
