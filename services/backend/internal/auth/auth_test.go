package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/pkg/jwt"
	"github.com/al-hudhud/backend/pkg/password"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthErrors(t *testing.T) {
	// Test error constants exist
	assert.Error(t, auth.ErrInvalidCredentials)
	assert.Error(t, auth.ErrUserNotActive)
	assert.Error(t, auth.ErrInvalidToken)
}

func TestPasswordHashing(t *testing.T) {
	// Arrange
	passwordStr := "TestPassword123!"

	// Act
	hash, err := password.HashPassword(passwordStr)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, passwordStr, hash)

	// Test password check
	isValid := password.CheckPassword(passwordStr, hash)
	assert.True(t, isValid)

	// Test wrong password
	isInvalid := password.CheckPassword("WrongPassword", hash)
	assert.False(t, isInvalid)
}

func TestPasswordValidation(t *testing.T) {
	// Test valid password
	validPassword := "SecureP@ss123!"
	assert.True(t, password.ValidatePasswordStrength(validPassword))

	// Test invalid passwords - only length is checked
	assert.False(t, password.ValidatePasswordStrength("short")) // Too short (less than 8 chars)
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create middleware - AuthMiddleware is a function, not a constructor
	router.GET("/protected", auth.AuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Test without token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create middleware
	router.GET("/protected", auth.AuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Test with invalid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a valid token
	userID := uuid.New().String()
	token, _ := jwt.GenerateToken(userID, "admin@alhudhud.com", "staff", "", uuid.New().String(), []string{"read"})

	// Create middleware
	router.GET("/protected", auth.AuthMiddleware(), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})

	// Test with valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGenerateRefreshToken(t *testing.T) {
	// Act
	token, err := jwt.GenerateRefreshToken()

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	// Base64 encoded 32 bytes can be 43-44 characters depending on padding
	assert.GreaterOrEqual(t, len(token), 40)
	assert.LessOrEqual(t, len(token), 44)
}

func TestServiceCanBeCreated(t *testing.T) {
	// This test verifies the service can be created
	service := auth.NewService()
	assert.NotNil(t, service)
}

func TestJWTClaims_Parsing(t *testing.T) {
	// Arrange
	userID := uuid.New().String()
	email := "admin@alhudhud.com"
	userType := "staff"
	tenantID := ""
	staffID := uuid.New().String()
	permissions := []string{"read", "write"}

	// Act
	token, err := jwt.GenerateToken(userID, email, userType, tenantID, staffID, permissions)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse the token
	claims, err := jwt.ValidateToken(token)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, userType, claims.UserType)
	assert.Equal(t, tenantID, claims.TenantID)
	assert.Equal(t, staffID, claims.StaffID)
	assert.Equal(t, permissions, claims.Permissions)
}

func TestJWTClaims_InvalidToken(t *testing.T) {
	// Arrange - Invalid token format
	_, err := jwt.ValidateToken("invalid-token-format")

	// Assert
	assert.Error(t, err)
}

func TestTokenExpiry(t *testing.T) {
	// Arrange
	userID := uuid.New().String()
	email := "admin@alhudhud.com"
	userType := "staff"
	tenantID := ""
	staffID := uuid.New().String()
	permissions := []string{"read"}

	// Act
	token, err := jwt.GenerateToken(userID, email, userType, tenantID, staffID, permissions)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify the token is valid now
	claims, err := jwt.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Verify expiry is set to the configured access-token lifetime
	expectedExpiry := time.Now().Add(jwt.AccessTokenExpiry)
	actualExpiry := claims.ExpiresAt.Time
	assert.WithinDuration(t, expectedExpiry, actualExpiry, time.Minute)
}

func TestAuthMiddleware_RequireUserType(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create middleware
	router.GET("/admin", auth.AuthMiddleware(), auth.RequireUserType("SUPER_ADMIN", "ADMIN"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test without proper user type (no token)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	router.ServeHTTP(w, req)

	// Should fail due to no token
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_RequirePermission(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create middleware
	router.GET("/secure", auth.AuthMiddleware(), auth.RequirePermission("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test without proper permissions (no token)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/secure", nil)
	router.ServeHTTP(w, req)

	// Should fail due to no token
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_TenantIsolation(t *testing.T) {
	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create middleware
	router.GET("/tenant-data", auth.AuthMiddleware(), auth.TenantIsolationMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test without proper tenant context (no token)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/tenant-data", nil)
	router.ServeHTTP(w, req)

	// Should fail due to no token
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}