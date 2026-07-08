package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFToken represents a CSRF token with its metadata
type CSRFToken struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CSRFTokenStore manages CSRF tokens in memory
// In production, consider using Redis for distributed systems
type CSRFTokenStore struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
}

var (
	tokenStore     *CSRFTokenStore
	tokenStoreOnce sync.Once
)

// GetTokenStore returns the singleton token store instance
func GetTokenStore() *CSRFTokenStore {
	tokenStoreOnce.Do(func() {
		tokenStore = &CSRFTokenStore{
			tokens: make(map[string]*CSRFToken),
		}
		// Start cleanup goroutine
		go tokenStore.cleanupExpiredTokens()
	})
	return tokenStore
}

// GenerateToken generates a new CSRF token
func (s *CSRFTokenStore) GenerateToken() (string, error) {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode to base64
	token := base64.URLEncoding.EncodeToString(b)

	// Store token with expiration
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.tokens[token] = &CSRFToken{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour), // Token valid for 24 hours
	}

	return token, nil
}

// ValidateToken validates a CSRF token
func (s *CSRFTokenStore) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	storedToken, exists := s.tokens[token]
	if !exists {
		return false
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		return false
	}

	return true
}

// DeleteToken removes a token from the store
func (s *CSRFTokenStore) DeleteToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

// cleanupExpiredTokens periodically removes expired tokens
func (s *CSRFTokenStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, csrfToken := range s.tokens {
			if now.After(csrfToken.ExpiresAt) {
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}

// CSRFProtection middleware validates CSRF tokens using double-submit cookie pattern
// It checks both the cookie and the header to prevent CSRF attacks
func CSRFProtection() gin.HandlerFunc {
	store := GetTokenStore()

	return func(c *gin.Context) {
		method := c.Request.Method

		// Skip CSRF validation for safe methods (GET, HEAD, OPTIONS)
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip CSRF validation for API key authentication
		// API keys are used for server-to-server communication where CSRF is not a concern
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			c.Next()
			return
		}

		// Skip CSRF validation for Bearer token auth (mobile apps and dynamic API requests)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			c.Next()
			return
		}

		// For state-changing methods (POST, PUT, DELETE, PATCH), validate CSRF token
		// Double-submit cookie pattern: token must be in both cookie and header

		// Get token from cookie
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil || cookieToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CSRF_TOKEN_MISSING",
					"message": "CSRF token missing in cookie",
				},
			})
			c.Abort()
			return
		}

		// Get token from header
		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CSRF_TOKEN_MISSING",
					"message": "CSRF token missing in header",
				},
			})
			c.Abort()
			return
		}

		// Tokens must match (double-submit pattern)
		if cookieToken != headerToken {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CSRF_TOKEN_MISMATCH",
					"message": "CSRF token mismatch between cookie and header",
				},
			})
			c.Abort()
			return
		}

		// Validate token exists and is not expired
		if !store.ValidateToken(cookieToken) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CSRF_TOKEN_INVALID",
					"message": "CSRF token is invalid or expired",
				},
			})
			c.Abort()
			return
		}

		// Token is valid, proceed with request
		c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token and returns it
// This is used by the /api/csrf-token endpoint
func GenerateCSRFToken() (string, error) {
	store := GetTokenStore()
	return store.GenerateToken()
}
