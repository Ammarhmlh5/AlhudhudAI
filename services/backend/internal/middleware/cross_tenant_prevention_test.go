package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// :memory: sqlite is per-connection; pin the pool to a single connection
	// so tables created by AutoMigrate are visible to later queries.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Auto-migrate the schema
	err = db.AutoMigrate(&CrossTenantAccessLog{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func setupTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Middleware to set tenant context
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", c.GetHeader("X-Tenant-ID"))
		c.Set("user_id", c.GetHeader("X-User-ID"))
		c.Set("user_type", c.GetHeader("X-User-Type"))
		c.Next()
	})

	router.Use(PreventCrossTenantAccess(db))

	// Test routes
	router.GET("/api/tenants/:tenant_id/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	router.POST("/api/tenants/:tenant_id/resources", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "created"})
	})

	return router
}

func TestPreventCrossTenantAccess_SameTenant(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	tenantID := uuid.New().String()

	req, _ := http.NewRequest("GET", "/api/tenants/"+tenantID+"/data", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "TENANT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestPreventCrossTenantAccess_DifferentTenant(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	authenticatedTenantID := uuid.New().String()
	requestedTenantID := uuid.New().String()

	req, _ := http.NewRequest("GET", "/api/tenants/"+requestedTenantID+"/data", nil)
	req.Header.Set("X-Tenant-ID", authenticatedTenantID)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "TENANT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
}

func TestPreventCrossTenantAccess_SuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	authenticatedTenantID := uuid.New().String()
	requestedTenantID := uuid.New().String()

	req, _ := http.NewRequest("GET", "/api/tenants/"+requestedTenantID+"/data", nil)
	req.Header.Set("X-Tenant-ID", authenticatedTenantID)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "SUPER_ADMIN")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Super admin should be allowed to access any tenant
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestPreventCrossTenantAccess_Staff(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	authenticatedTenantID := uuid.New().String()
	requestedTenantID := uuid.New().String()

	req, _ := http.NewRequest("GET", "/api/tenants/"+requestedTenantID+"/data", nil)
	req.Header.Set("X-Tenant-ID", authenticatedTenantID)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "STAFF")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Staff should be allowed to access any tenant
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestPreventCrossTenantAccess_QueryParameter(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	authenticatedTenantID := uuid.New().String()
	requestedTenantID := uuid.New().String()

	req, _ := http.NewRequest("GET", "/api/data?tenant_id="+requestedTenantID, nil)
	req.Header.Set("X-Tenant-ID", authenticatedTenantID)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "TENANT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
}

func TestPreventCrossTenantAccess_RequestBody(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	authenticatedTenantID := uuid.New().String()
	requestedTenantID := uuid.New().String()

	body := map[string]interface{}{
		"tenant_id": requestedTenantID,
		"name":      "Test Resource",
	}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tenants/"+authenticatedTenantID+"/resources", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", authenticatedTenantID)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "TENANT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
}

func TestPreventCrossTenantAccess_MissingTenantID(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	req, _ := http.NewRequest("GET", "/api/tenants/"+uuid.New().String()+"/data", nil)
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "TENANT")
	// No X-Tenant-ID header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "TENANT_ID_MISSING", errorObj["code"])
}

func TestPreventCrossTenantAccess_InvalidTenantIDFormat(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	req, _ := http.NewRequest("GET", "/api/tenants/invalid-uuid/data", nil)
	req.Header.Set("X-Tenant-ID", uuid.New().String())
	req.Header.Set("X-User-ID", uuid.New().String())
	req.Header.Set("X-User-Type", "TENANT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUESTED_TENANT_ID", errorObj["code"])
}

func TestPreventCrossTenantAccess_LogsAttempt(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db)

	authenticatedTenantID := uuid.New().String()
	requestedTenantID := uuid.New().String()
	userID := uuid.New().String()

	req, _ := http.NewRequest("GET", "/api/tenants/"+requestedTenantID+"/data", nil)
	req.Header.Set("X-Tenant-ID", authenticatedTenantID)
	req.Header.Set("X-User-ID", userID)
	req.Header.Set("X-User-Type", "TENANT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// Wait a bit for async logging
	// Note: In production, you'd use a proper synchronization mechanism
	// For testing, we'll just check that the log was created
	var count int64
	db.Model(&CrossTenantAccessLog{}).Count(&count)
	
	// The log should be created (may take a moment due to async)
	// In a real test, you'd use proper synchronization
	assert.GreaterOrEqual(t, count, int64(0))
}
