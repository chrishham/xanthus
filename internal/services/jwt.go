package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	ErrTokenNotFound = errors.New("token not found")
)

type Claims struct {
	UserID      string `json:"user_id"`
	AccountID   string `json:"account_id"`
	NamespaceID string `json:"namespace_id"`
	CFToken     string `json:"cf_token"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey       []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
	issuer          string
}

func NewJWTService(secretKey []byte, accessDuration, refreshDuration time.Duration) *JWTService {
	return &JWTService{
		secretKey:       secretKey,
		accessDuration:  accessDuration,
		refreshDuration: refreshDuration,
		issuer:          "xanthus",
	}
}

func GenerateSecretKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret key: %w", err)
	}
	return key, nil
}

func (j *JWTService) GenerateTokenPair(userID, accountID, namespaceID, cfToken string) (string, string, error) {
	accessToken, err := j.GenerateAccessToken(userID, accountID, namespaceID, cfToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := j.GenerateRefreshToken(userID, accountID, namespaceID, cfToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (j *JWTService) GenerateAccessToken(userID, accountID, namespaceID, cfToken string) (string, error) {
	claims := &Claims{
		UserID:      userID,
		AccountID:   accountID,
		NamespaceID: namespaceID,
		CFToken:     cfToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) GenerateRefreshToken(userID, accountID, namespaceID, cfToken string) (string, error) {
	claims := &Claims{
		UserID:      userID,
		AccountID:   accountID,
		NamespaceID: namespaceID,
		CFToken:     cfToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (j *JWTService) RefreshToken(refreshTokenString string) (string, string, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	return j.GenerateTokenPair(claims.UserID, claims.AccountID, claims.NamespaceID, claims.CFToken)
}

func (j *JWTService) ExtractUserID(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (j *JWTService) ExtractAccountInfo(tokenString string) (string, string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", "", err
	}
	return claims.AccountID, claims.NamespaceID, nil
}

func (j *JWTService) ExtractCFToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.CFToken, nil
}