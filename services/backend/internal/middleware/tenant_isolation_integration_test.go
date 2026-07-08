package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestTenant represents a test tenant
type TestTenant struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name string    `gorm:"type:varchar(255);not null"`
}

// TestResource represents a test resource with tenant isolation
type TestResource struct {
	ID       uint      `gorm:"primaryKey"`
	TenantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name     string    `gorm:"type:varchar(255);not null"`
	Value    string    `gorm:"type:text"`
}

func setupIntegrationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database")

	// Auto-migrate schemas
	err = db.AutoMigrate(&TestTenant{}, &TestResource{}, &CrossTenantAccessLog{})
	require.NoError(t, err, "Failed to migrate test database")

	return db
}

func createTestTenants(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	tenant1 := TestTenant{
		ID:   uuid.New(),
		Name: "Tenant 1",
	}
	tenant2 := TestTenant{
		ID:   uuid.New(),
		Name: "Tenant 2",
	}

	err := db.Create(&tenant1).Error
	require.NoError(t, err, "Failed to create tenant 1")

	err = db.Create(&tenant2).Error
	require.NoError(t, err, "Failed to create tenant 2")

	return tenant1.ID, tenant2.ID
}

func createTestResources(t *testing.T, db *gorm.DB, tenant1ID, tenant2ID uuid.UUID) {
	resources := []TestResource{
		{TenantID: tenant1ID, Name: "Resource 1 for Tenant 1", Value: "Value 1"},
		{TenantID: tenant1ID, Name: "Resource 2 for Tenant 1", Value: "Value 2"},
		{TenantID: tenant2ID, Name: "Resource 1 for Tenant 2", Value: "Value 3"},
		{TenantID: tenant2ID, Name: "Resource 2 for Tenant 2", Value: "Value 4"},
	}

	for _, resource := range resources {
		err := db.Create(&resource).Error
		require.NoError(t, err, "Failed to create test resource")
	}
}

func setupIntegrationTestRouter(db *gorm.DB) *gin.Engine {
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
	router.GET("/api/tenants/:tenant_id/resources", func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
			return
		}

		var resources []TestResource
		if err := db.Where("tenant_id = ?", tenantUUID).Find(&resources).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"resources": resources,
		})
	})

	router.POST("/api/tenants/:tenant_id/resources", func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
			return
		}

		var input struct {
			Name  string `json:"name" binding:"required"`
			Value string `json:"value"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resource := TestResource{
			TenantID: tenantUUID,
			Name:     input.Name,
			Value:    input.Value,
		}

		if err := db.Create(&resource).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success":  true,
			"resource": resource,
		})
	})

	router.PUT("/api/tenants/:tenant_id/resources/:resource_id", func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
			return
		}

		resourceID := c.Param("resource_id")

		var input struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var resource TestResource
		if err := db.Where("id = ? AND tenant_id = ?", resourceID, tenantUUID).First(&resource).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return
		}

		if input.Name != "" {
			resource.Name = input.Name
		}
		if input.Value != "" {
			resource.Value = input.Value
		}

		if err := db.Save(&resource).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"resource": resource,
		})
	})

	router.DELETE("/api/tenants/:tenant_id/resources/:resource_id", func(c *gin.Context) {
		tenantID := c.Param("tenant_id")
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant ID"})
			return
		}

		resourceID := c.Param("resource_id")

		result := db.Where("id = ? AND tenant_id = ?", resourceID, tenantUUID).Delete(&TestResource{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "resource deleted",
		})
	})

	return router
}

// Property 16: Tenant Isolation في قاعدة البيانات
// Validates: Requirements 26.5
func TestProperty16_TenantIsolationInDatabase(t *testing.T) {
	db := setupIntegrationTestDB(t)
	tenant1ID, tenant2ID := createTestTenants(t, db)
	createTestResources(t, db, tenant1ID, tenant2ID)
	router := setupIntegrationTestRouter(db)

	t.Run("Tenant1 can only see their own resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant1ID.String()), nil)
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		resources := response["resources"].([]interface{})
		assert.Equal(t, 2, len(resources))

		// Verify all resources belong to tenant1
		for _, r := range resources {
			resource := r.(map[string]interface{})
			assert.Equal(t, tenant1ID.String(), resource["TenantID"])
		}
	})

	t.Run("Tenant2 can only see their own resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant2ID.String()), nil)
		req.Header.Set("X-Tenant-ID", tenant2ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		resources := response["resources"].([]interface{})
		assert.Equal(t, 2, len(resources))

		// Verify all resources belong to tenant2
		for _, r := range resources {
			resource := r.(map[string]interface{})
			assert.Equal(t, tenant2ID.String(), resource["TenantID"])
		}
	})

	t.Run("Tenant1 cannot access Tenant2 resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant2ID.String()), nil)
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
	})

	t.Run("Tenant2 cannot access Tenant1 resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant1ID.String()), nil)
		req.Header.Set("X-Tenant-ID", tenant2ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
	})
}

func TestIntegration_CreateResourceWithTenantIsolation(t *testing.T) {
	db := setupIntegrationTestDB(t)
	tenant1ID, tenant2ID := createTestTenants(t, db)
	router := setupIntegrationTestRouter(db)

	t.Run("Tenant1 can create resource for themselves", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "New Resource",
			"value": "New Value",
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/tenants/%s/resources", tenant1ID.String()), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		resource := response["resource"].(map[string]interface{})
		assert.Equal(t, tenant1ID.String(), resource["TenantID"])
		assert.Equal(t, "New Resource", resource["Name"])
	})

	t.Run("Tenant1 cannot create resource for Tenant2", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "Malicious Resource",
			"value": "Malicious Value",
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/tenants/%s/resources", tenant2ID.String()), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
	})
}

func TestIntegration_UpdateResourceWithTenantIsolation(t *testing.T) {
	db := setupIntegrationTestDB(t)
	tenant1ID, tenant2ID := createTestTenants(t, db)
	createTestResources(t, db, tenant1ID, tenant2ID)
	router := setupIntegrationTestRouter(db)

	// Get a resource from tenant1
	var tenant1Resource TestResource
	err := db.Where("tenant_id = ?", tenant1ID).First(&tenant1Resource).Error
	require.NoError(t, err)

	t.Run("Tenant1 can update their own resource", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "Updated Resource",
			"value": "Updated Value",
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/tenants/%s/resources/%d", tenant1ID.String(), tenant1Resource.ID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		resource := response["resource"].(map[string]interface{})
		assert.Equal(t, "Updated Resource", resource["Name"])
	})

	t.Run("Tenant2 cannot update Tenant1 resource", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "Malicious Update",
			"value": "Malicious Value",
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/tenants/%s/resources/%d", tenant1ID.String(), tenant1Resource.ID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", tenant2ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])
	})
}

func TestIntegration_DeleteResourceWithTenantIsolation(t *testing.T) {
	db := setupIntegrationTestDB(t)
	tenant1ID, tenant2ID := createTestTenants(t, db)
	createTestResources(t, db, tenant1ID, tenant2ID)
	router := setupIntegrationTestRouter(db)

	// Get a resource from tenant1
	var tenant1Resource TestResource
	err := db.Where("tenant_id = ?", tenant1ID).First(&tenant1Resource).Error
	require.NoError(t, err)

	t.Run("Tenant1 can delete their own resource", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/tenants/%s/resources/%d", tenant1ID.String(), tenant1Resource.ID), nil)
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))

		// Verify resource is deleted
		var count int64
		db.Model(&TestResource{}).Where("id = ?", tenant1Resource.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	// Get a resource from tenant2
	var tenant2Resource TestResource
	err = db.Where("tenant_id = ?", tenant2ID).First(&tenant2Resource).Error
	require.NoError(t, err)

	t.Run("Tenant1 cannot delete Tenant2 resource", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/tenants/%s/resources/%d", tenant2ID.String(), tenant2Resource.ID), nil)
		req.Header.Set("X-Tenant-ID", tenant1ID.String())
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "TENANT")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "CROSS_TENANT_ACCESS_DENIED", errorObj["code"])

		// Verify resource still exists
		var count int64
		db.Model(&TestResource{}).Where("id = ?", tenant2Resource.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestIntegration_SuperAdminCanAccessAllTenants(t *testing.T) {
	db := setupIntegrationTestDB(t)
	tenant1ID, tenant2ID := createTestTenants(t, db)
	createTestResources(t, db, tenant1ID, tenant2ID)
	router := setupIntegrationTestRouter(db)

	t.Run("Super Admin can access Tenant1 resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant1ID.String()), nil)
		req.Header.Set("X-Tenant-ID", uuid.New().String()) // Different tenant
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "SUPER_ADMIN")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		resources := response["resources"].([]interface{})
		assert.Equal(t, 2, len(resources))
	})

	t.Run("Super Admin can access Tenant2 resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/tenants/%s/resources", tenant2ID.String()), nil)
		req.Header.Set("X-Tenant-ID", uuid.New().String()) // Different tenant
		req.Header.Set("X-User-ID", uuid.New().String())
		req.Header.Set("X-User-Type", "SUPER_ADMIN")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		resources := response["resources"].([]interface{})
		assert.Equal(t, 2, len(resources))
	})
}
