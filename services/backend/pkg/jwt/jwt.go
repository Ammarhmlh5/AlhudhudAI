package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// JWT secret key - should be loaded from environment in production
	jwtSecret = []byte("your-secret-key-change-this-in-production")

	// Token expiration times
	AccessTokenExpiry  = 1 * time.Hour      // 1 hour
	RefreshTokenExpiry = 7 * 24 * time.Hour // 7 days
)

// Claims represents the JWT claims
type Claims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	UserType    string   `json:"user_type"`
	TenantID    string   `json:"tenant_id,omitempty"`
	StaffID     string   `json:"staff_id,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT access token
func GenerateToken(userID, email, userType string, tenantID, staffID string, permissions []string) (string, error) {
	claims := Claims{
		UserID:      userID,
		Email:       email,
		UserType:    userType,
		TenantID:    tenantID,
		StaffID:     staffID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateRefreshToken creates a random refresh token
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateToken validates and parses a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshAccessToken generates a new access token from a refresh token
func RefreshAccessToken(userID, email, userType string, tenantID, staffID string, permissions []string) (string, error) {
	return GenerateToken(userID, email, userType, tenantID, staffID, permissions)
}

// SetJWTSecret sets the JWT secret key (should be called on app initialization)
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}
