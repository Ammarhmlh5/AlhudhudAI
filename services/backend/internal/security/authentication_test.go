package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtpkg "github.com/al-hudhud/backend/pkg/jwt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIAccessWithoutAuth tests API access without authentication
func TestAPIAccessWithoutAuth(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	endpoints := []string{
		"/api/admin/dashboard/overview",
		"/api/admin/companies/ratings",
		"/api/admin/companies/test-tenant/rating",
		"/api/admin/servers",
		"/api/admin/alerts",
		"/api/admin/distribution/test-tenant",
		"/api/admin/signalpush/test-tenant/statistics",
	}

	for _, endpoint := range endpoints {
		t.Run("No auth header - "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should return 401 Unauthorized
			assert.Equal(t, http.StatusUnauthorized, w.Code, 
				"Endpoint should require authentication: "+endpoint)
		})
	}
}

// TestAPIAccessWithInvalidToken tests API access with invalid tokens
func TestAPIAccessWithInvalidToken(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	invalidTokens := []string{
		"",
		"invalid-token",
		"Bearer ",
		"Bearer invalid",
		"Basic dGVzdDp0ZXN0", // Wrong auth type
		"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature", // Invalid JWT
	}

	endpoint := "/api/admin/dashboard/overview"

	for _, token := range invalidTokens {
		t.Run("Invalid token: "+token, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			if token != "" {
				req.Header.Set("Authorization", token)
			}
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should return 401 Unauthorized
			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"Invalid token should be rejected")
		})
	}
}

// TestAPIAccessWithExpiredToken tests API access with expired tokens
func TestAPIAccessWithExpiredToken(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	// Generate an expired token
	expiredToken := generateExpiredToken(t)
	
	req := httptest.NewRequest("GET", "/api/admin/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	// Should return 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"Expired token should be rejected")
	
	// Response should indicate token expiration
	assert.Contains(t, w.Body.String(), "expired",
		"Response should indicate token expiration")
}

// TestAPIAccessWithValidToken tests API access with valid tokens
func TestAPIAccessWithValidToken(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	// Generate a valid token
	validToken := generateValidToken(t, "admin-user", []string{"SUPER_ADMIN"})
	
	endpoints := []string{
		"/api/admin/dashboard/overview",
		"/api/admin/companies/ratings",
		"/api/admin/servers",
	}

	for _, endpoint := range endpoints {
		t.Run("Valid token - "+endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should return 200 OK or other success status
			assert.NotEqual(t, http.StatusUnauthorized, w.Code,
				"Valid token should be accepted")
			assert.NotEqual(t, http.StatusForbidden, w.Code,
				"Valid admin token should have access")
		})
	}
}

// TestTokenRefresh tests token refresh mechanism
func TestTokenRefresh(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	// Generate a token that's about to expire
	almostExpiredToken := generateAlmostExpiredToken(t)
	
	req := httptest.NewRequest("POST", "/api/admin/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+almostExpiredToken)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	// Should return 200 OK with new token
	assert.Equal(t, http.StatusOK, w.Code,
		"Token refresh should succeed")
	
	// Response should contain new token
	assert.Contains(t, w.Body.String(), "token",
		"Response should contain new token")
}

// TestRateLimiting tests rate limiting on authentication endpoints
func TestRateLimiting(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	// Make multiple failed login attempts
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/admin/auth/login", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		if i < 5 {
			// First 5 attempts should return 401
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		} else {
			// After 5 attempts, should be rate limited
			assert.Equal(t, http.StatusTooManyRequests, w.Code,
				"Should be rate limited after multiple failed attempts")
		}
	}
}

// Helper functions

func generateExpiredToken(t *testing.T) string {
	t.Helper()
	claims := &jwtpkg.Claims{
		UserID:   "admin",
		Email:    "admin@test.com",
		UserType: "SUPER_ADMIN",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, err := jwtpkg.SignToken(claims)
	require.NoError(t, err)
	return token
}

func generateValidToken(t *testing.T, userID string, roles []string) string {
	t.Helper()
	token, err := jwtpkg.GenerateToken(userID, userID+"@example.com", roleToUserType(roles), tenantIDForUser(userID), "", nil)
	require.NoError(t, err)
	return token
}

func generateAlmostExpiredToken(t *testing.T) string {
	t.Helper()
	claims := &jwtpkg.Claims{
		UserID:   "admin",
		Email:    "admin@test.com",
		UserType: "SUPER_ADMIN",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwtpkg.SignToken(claims)
	require.NoError(t, err)
	return token
}

// roleToUserType maps the role names used by the security suite to the
// user_type values enforced by the real RBAC middleware.
func roleToUserType(roles []string) string {
	for _, role := range roles {
		switch role {
		case "SUPER_ADMIN":
			return "SUPER_ADMIN"
		case "STAFF":
			return "STAFF"
		case "TENANT_ADMIN":
			return "TENANT"
		}
	}
	return "USER"
}

// tenantIDForUser derives a tenant id from a user id (user-1 -> tenant-1).
func tenantIDForUser(userID string) string {
	if strings.HasPrefix(userID, "user-") {
		return "tenant-" + strings.TrimPrefix(userID, "user-")
	}
	return ""
}
