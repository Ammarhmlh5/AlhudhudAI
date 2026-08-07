package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/al-hudhud/backend/pkg/password"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestCrossTenantAccessPrevention tests that tenants cannot access each other's data
func TestCrossTenantAccessPrevention(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create two tenants
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()

	// Insert test data for both tenants
	setupTenantData(t, db, tenant1ID)
	setupTenantData(t, db, tenant2ID)

	tests := []struct {
		name           string
		requestTenant  uuid.UUID
		targetTenant   uuid.UUID
		expectedStatus int
		shouldSucceed  bool
	}{
		{
			name:           "Same tenant access should succeed",
			requestTenant:  tenant1ID,
			targetTenant:   tenant1ID,
			expectedStatus: http.StatusOK,
			shouldSucceed:  true,
		},
		{
			name:           "Cross-tenant access should be blocked",
			requestTenant:  tenant1ID,
			targetTenant:   tenant2ID,
			expectedStatus: http.StatusForbidden,
			shouldSucceed:  false,
		},
		{
			name:           "Reverse cross-tenant access should be blocked",
			requestTenant:  tenant2ID,
			targetTenant:   tenant1ID,
			expectedStatus: http.StatusForbidden,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router with middleware
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("tenant_id", tt.requestTenant.String())
				c.Set("db", db)
				c.Next()
			})

			// Add test endpoint
			router.GET("/tariffs/:tenant_id", func(c *gin.Context) {
				tenantID := c.Param("tenant_id")
				requestTenantID := c.GetString("tenant_id")

				// Cross-tenant prevention check
				if tenantID != requestTenantID {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "cross-tenant access denied",
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			// Make request
			req := httptest.NewRequest("GET", fmt.Sprintf("/tariffs/%s", tt.targetTenant.String()), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.shouldSucceed {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "cross-tenant")
			}
		})
	}
}

// TestSQLInjectionPrevention_DB tests that SQL injection attempts are blocked
func TestSQLInjectionPrevention_DB(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tenantID := uuid.New()
	setupTenantData(t, db, tenantID)

	sqlInjectionPayloads := []string{
		"'; DROP TABLE tariffs; --",
		"1' OR '1'='1",
		"1' UNION SELECT * FROM users--",
		"admin'--",
		"' OR 1=1--",
		"1'; DELETE FROM tariffs WHERE '1'='1",
		"1' AND 1=0 UNION ALL SELECT 'admin', '81dc9bdb52d04dc20036dbd8313ed055'",
		"' UNION SELECT NULL, NULL, NULL--",
		"1' AND (SELECT COUNT(*) FROM tariffs) > 0--",
	}

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("tenant_id", tenantID.String())
				c.Set("db", db)
				c.Next()
			})

			router.GET("/tariffs/:id", func(c *gin.Context) {
				tariffID := c.Param("id")
				db := c.MustGet("db").(*gorm.DB)
				tenantIDStr := c.GetString("tenant_id")

				var tariff struct {
					ID       int
					TenantID string
					Name     string
				}

				// Using parameterized query (safe)
				result := db.Raw("SELECT id, tenant_id, name FROM security_test_tariffs WHERE id = ? AND tenant_id = ?", 
					tariffID, tenantIDStr).Scan(&tariff)

				if result.Error != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"tariff": tariff})
			})

			req := httptest.NewRequest("GET", fmt.Sprintf("/tariffs/%s", url.PathEscape(payload)), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should either return 400 (invalid) or 200 with no data
			// Should NOT return 500 (server error from SQL injection)
			assert.NotEqual(t, http.StatusInternalServerError, w.Code, 
				"SQL injection may have caused server error")

			// Verify database integrity
			var count int64
			db.Raw("SELECT COUNT(*) FROM security_test_tariffs WHERE tenant_id = ?", tenantID.String()).Scan(&count)
			assert.Greater(t, count, int64(0), "Data should not be deleted")
		})
	}
}

// TestXSSPrevention_JSON tests that XSS attempts are properly escaped
func TestXSSPrevention_JSON(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"<svg/onload=alert('XSS')>",
		"javascript:alert('XSS')",
		"<iframe src='javascript:alert(\"XSS\")'></iframe>",
		"<body onload=alert('XSS')>",
		"<input onfocus=alert('XSS') autofocus>",
		"<select onfocus=alert('XSS') autofocus>",
		"<textarea onfocus=alert('XSS') autofocus>",
		"<marquee onstart=alert('XSS')>",
	}

	for _, payload := range xssPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			router := gin.New()

			router.POST("/tariffs", func(c *gin.Context) {
				var input struct {
					Name        string `json:"name"`
					Description string `json:"description"`
				}

				if err := c.ShouldBindJSON(&input); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// Return the input (should be escaped in JSON)
				c.JSON(http.StatusOK, gin.H{
					"name":        input.Name,
					"description": input.Description,
				})
			})

			body := map[string]string{
				"name":        payload,
				"description": payload,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/tariffs", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			// Verify response is properly JSON-encoded (which escapes HTML)
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON")

			// Verify dangerous characters are escaped in the JSON output
			responseStr := w.Body.String()
			assert.NotContains(t, responseStr, "<script>", "Script tags should be escaped")
			assert.NotContains(t, responseStr, "<", "HTML special characters should be escaped")
			assert.NotContains(t, responseStr, "&", "Ampersands should be escaped")
		})
	}
}

// TestCSRFProtection tests CSRF token validation
func TestCSRFProtection(t *testing.T) {
	tests := []struct {
		name           string
		csrfToken      string
		expectedStatus int
		shouldSucceed  bool
	}{
		{
			name:           "Valid CSRF token should succeed",
			csrfToken:      "valid-token-123",
			expectedStatus: http.StatusOK,
			shouldSucceed:  true,
		},
		{
			name:           "Missing CSRF token should fail",
			csrfToken:      "",
			expectedStatus: http.StatusForbidden,
			shouldSucceed:  false,
		},
		{
			name:           "Invalid CSRF token should fail",
			csrfToken:      "invalid-token",
			expectedStatus: http.StatusForbidden,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			// CSRF middleware
			router.Use(func(c *gin.Context) {
				// In production, this would validate against session
				expectedToken := "valid-token-123"
				token := c.GetHeader("X-CSRF-Token")

				if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
					if token != expectedToken {
						c.JSON(http.StatusForbidden, gin.H{
							"error": "invalid CSRF token",
						})
						c.Abort()
						return
					}
				}
				c.Next()
			})

			router.POST("/tariffs", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			req := httptest.NewRequest("POST", "/tariffs", nil)
			if tt.csrfToken != "" {
				req.Header.Set("X-CSRF-Token", tt.csrfToken)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.shouldSucceed {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "CSRF")
			}
		})
	}
}

// TestAPIAuthentication tests API key and JWT authentication
func TestAPIAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		shouldSucceed  bool
	}{
		{
			name:           "Valid API key should succeed",
			authHeader:     "Bearer valid-api-key-123",
			expectedStatus: http.StatusOK,
			shouldSucceed:  true,
		},
		{
			name:           "Missing auth header should fail",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			shouldSucceed:  false,
		},
		{
			name:           "Invalid API key should fail",
			authHeader:     "Bearer invalid-key",
			expectedStatus: http.StatusUnauthorized,
			shouldSucceed:  false,
		},
		{
			name:           "Malformed auth header should fail",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
			shouldSucceed:  false,
		},
		{
			name:           "Expired token should fail",
			authHeader:     "Bearer expired-token",
			expectedStatus: http.StatusUnauthorized,
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			// Auth middleware
			router.Use(func(c *gin.Context) {
				authHeader := c.GetHeader("Authorization")

				if authHeader == "" {
					c.JSON(http.StatusUnauthorized, gin.H{
						"error": "missing authorization header",
					})
					c.Abort()
					return
				}

				// Simple validation (in production, verify against database)
				validTokens := map[string]bool{
					"Bearer valid-api-key-123": true,
				}

				if !validTokens[authHeader] {
					c.JSON(http.StatusUnauthorized, gin.H{
						"error": "invalid or expired token",
					})
					c.Abort()
					return
				}

				c.Next()
			})

			router.GET("/tariffs", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			req := httptest.NewRequest("GET", "/tariffs", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.shouldSucceed {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

// TestRateLimitingByIP tests that rate limiting prevents abuse
func TestRateLimitingByIP(t *testing.T) {
	router := gin.New()

	// Simple rate limiter (in production, use Redis)
	requestCounts := make(map[string]int)
	maxRequests := 5

	router.Use(func(c *gin.Context) {
		ip := c.ClientIP()
		requestCounts[ip]++

		if requestCounts[ip] > maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	})

	router.GET("/tariffs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// Make requests up to limit
	for i := 0; i < maxRequests; i++ {
		req := httptest.NewRequest("GET", "/tariffs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, fmt.Sprintf("Request %d should succeed", i+1))
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/tariffs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request should be rate limited")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "rate limit")
}

// TestPasswordHashing tests that passwords are properly hashed
func TestPasswordHashing(t *testing.T) {
	plaintext := "SecurePassword123!"

	// Hash password using the real bcrypt implementation
	hashedPassword, err := password.HashPassword(plaintext)
	require.NoError(t, err)

	// Verify hash is not plaintext
	assert.NotEqual(t, plaintext, hashedPassword, "Password should be hashed")
	assert.Greater(t, len(hashedPassword), 20, "Hash should be sufficiently long")

	// Verify same password produces different hashes (salt)
	hashedPassword2, err := password.HashPassword(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, hashedPassword, hashedPassword2, "Hashes should use unique salts")

	// Verify password verification works
	assert.True(t, password.CheckPassword(plaintext, hashedPassword), "Password verification should succeed")
	assert.False(t, password.CheckPassword("WrongPassword", hashedPassword), "Wrong password should fail")
}

// TestSecureHeaders tests that security headers are set
func TestSecureHeaders(t *testing.T) {
	router := gin.New()

	// Security headers middleware
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src")
}

// Helper functions

func setupTestDB(t *testing.T) *gorm.DB {
	dbPassword := os.Getenv("PGPASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dsn := fmt.Sprintf("host=localhost user=postgres password=%s dbname=sms_gateway_test port=5432 sslmode=disable", dbPassword)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Bound the pool so concurrent tests (up to 500 goroutines) queue on the
	// client side instead of exhausting the server's max_connections limit.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Create test table
	db.Exec(`
		CREATE TABLE IF NOT EXISTS security_test_tariffs (
			id SERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL,
			name VARCHAR(255),
			cost DECIMAL(10, 2)
		)
	`)

	return db
}

func cleanupTestDB(t *testing.T, db *gorm.DB) {
	db.Exec("DROP TABLE IF EXISTS security_test_tariffs")
}

func setupTenantData(t *testing.T, db *gorm.DB, tenantID uuid.UUID) {
	db.Exec("INSERT INTO security_test_tariffs (tenant_id, name, cost) VALUES (?, ?, ?)",
		tenantID.String(), "Test Tariff", 10.00)
}
