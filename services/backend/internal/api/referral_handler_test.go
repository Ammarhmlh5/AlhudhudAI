package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/al-hudhud/backend/internal/referral"
)

// Note: These tests require a test database connection
// Run with: go test -v ./internal/api/...

func setupReferralTestRouter(handler *ReferralHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Mock auth middleware
	r.Use(func(c *gin.Context) {
		// Set mock tenant ID
		c.Set("tenant_id", uuid.New())
		c.Set("role", "user")
		c.Next()
	})
	
	api := r.Group("/api")
	handler.RegisterRoutes(api)
	
	return r
}

func TestGetReferralCode(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/referrals/code", nil)
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	
	data := response["data"].(map[string]interface{})
	assert.NotEmpty(t, data["code"])
	assert.NotEmpty(t, data["share_url"])
}

func TestValidateReferralCode_Valid(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Create request body
	body := map[string]string{
		"code": "REF-TEST001",
	}
	jsonBody, _ := json.Marshal(body)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/referrals/validate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
}

func TestValidateReferralCode_Invalid(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Create request body with invalid code
	body := map[string]string{
		"code": "INVALID-CODE",
	}
	jsonBody, _ := json.Marshal(body)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/referrals/validate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "invalid referral code", response["error"])
}

func TestProcessReferral_Success(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Create request body
	body := map[string]interface{}{
		"referrer_tenant_id": uuid.New().String(),
		"referee_tenant_id":  uuid.New().String(),
		"referral_code":      "REF-TEST002",
	}
	jsonBody, _ := json.Marshal(body)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/referrals/process", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestProcessReferral_SelfReferral(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Create request body with same referrer and referee
	tenantID := uuid.New().String()
	body := map[string]interface{}{
		"referrer_tenant_id": tenantID,
		"referee_tenant_id":  tenantID, // Same as referrer
		"referral_code":      "REF-TEST003",
	}
	jsonBody, _ := json.Marshal(body)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/referrals/process", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "cannot refer yourself", response["error"])
}

func TestGetReferralStats(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/referrals/stats", nil)
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestListReferrals(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/referrals/list?limit=10&offset=0", nil)
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
	
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["referrals"])
	assert.Equal(t, float64(10), data["limit"])
	assert.Equal(t, float64(0), data["offset"])
}

func TestListReferrals_Pagination(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	// Execute with custom pagination
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/referrals/list?limit=5&offset=10", nil)
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(5), data["limit"])
	assert.Equal(t, float64(10), data["offset"])
}

func TestGetReferralPerformance_Admin(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Mock admin auth middleware
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uuid.New())
		c.Set("role", "admin") // Admin role
		c.Next()
	})
	
	api := r.Group("/api")
	handler.RegisterRoutes(api)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/referrals/performance", nil)
	r.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])
}

func TestGetReferralPerformance_NonAdmin(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler) // Uses "user" role
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/referrals/performance", nil)
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.False(t, response["success"].(bool))
	assert.Equal(t, "admin access required", response["error"])
}

func TestGrantReward_Admin(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Mock admin auth middleware
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uuid.New())
		c.Set("role", "admin")
		c.Next()
	})
	
	api := r.Group("/api")
	handler.RegisterRoutes(api)
	
	// Create request body
	referralID := uuid.New()
	body := map[string]string{
		"notes": "Test reward grant",
	}
	jsonBody, _ := json.Marshal(body)
	
	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/referrals/"+referralID.String()+"/grant-reward", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	
	// Assert (will fail because referral doesn't exist, but that's expected)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

func TestMarkConditionMet_Admin(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Mock admin auth middleware
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", uuid.New())
		c.Set("role", "admin")
		c.Next()
	})
	
	api := r.Group("/api")
	handler.RegisterRoutes(api)
	
	// Execute
	referralID := uuid.New()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/referrals/"+referralID.String()+"/mark-condition-met", nil)
	r.ServeHTTP(w, req)
	
	// Assert (will fail because referral doesn't exist, but that's expected)
	assert.NotEqual(t, http.StatusForbidden, w.Code)
}

// Benchmark tests
func BenchmarkGetReferralCode(b *testing.B) {
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/referrals/code", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkValidateReferralCode(b *testing.B) {
	manager := referral.NewManager(nil, nil)
	handler := NewReferralHandler(manager)
	router := setupReferralTestRouter(handler)
	
	body := map[string]string{
		"code": "REF-BENCH001",
	}
	jsonBody, _ := json.Marshal(body)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/referrals/validate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
	}
}
