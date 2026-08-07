package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestRouter creates a test router with middleware
func setupTestRouter(tenantID uuid.UUID, userRole string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Mock middleware to set tenant_id and user_type
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		userType := "USER"
		if userRole == "admin" {
			userType = "TENANT"
		}
		c.Set("user_type", userType)
		c.Next()
	})

	return router
}

func TestCreatePackage_Success(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "admin")
	router.POST("/packages", handler.CreatePackage)

	// Create test package
	includedMessages := int64(1000)
	req := CreatePackageRequest{
		ServiceID:        2, // Mobile Gateway Service
		Name:             "Test Package",
		Code:             "TEST_PKG",
		PriceUSD:         decimal.NewFromFloat(10.00),
		IncludedMessages: &includedMessages,
		OverageRate:      decimal.NewFromFloat(0.01),
		IsUnlimited:      false,
		BillingCycle:     "monthly",
		RolloverEnabled:  true,
		DisplayOrder:     1,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute
	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp CreatePackageResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Greater(t, resp.PackageID, int64(0))
	assert.Contains(t, resp.Message, "Test Package")

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", resp.PackageID)
}

func TestCreatePackage_NonAdmin(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user") // Non-admin role
	router.POST("/packages", handler.CreatePackage)

	// Create test package
	includedMessages := int64(1000)
	req := CreatePackageRequest{
		ServiceID:        2,
		Name:             "Test Package",
		Code:             "TEST_PKG",
		PriceUSD:         decimal.NewFromFloat(10.00),
		IncludedMessages: &includedMessages,
		OverageRate:      decimal.NewFromFloat(0.01),
		IsUnlimited:      false,
		BillingCycle:     "monthly",
		RolloverEnabled:  false,
		DisplayOrder:     1,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute
	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp["success"].(bool))
	assert.Contains(t, resp["error"].(string), "Admin access required")
}

func TestCreatePackage_InvalidRequest(t *testing.T) {
	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "admin")
	router.POST("/packages", handler.CreatePackage)

	tests := []struct {
		name        string
		req         CreatePackageRequest
		expectedErr string
	}{
		{
			name: "Negative price",
			req: CreatePackageRequest{
				ServiceID:    2,
				Name:         "Test",
				Code:         "TEST",
				PriceUSD:     decimal.NewFromFloat(-10.00),
				BillingCycle: "monthly",
			},
			expectedErr: "price_usd must be greater than 0",
		},
		{
			name: "Unlimited with included messages",
			req: CreatePackageRequest{
				ServiceID:        2,
				Name:             "Test",
				Code:             "TEST",
				PriceUSD:         decimal.NewFromFloat(10.00),
				IncludedMessages: func() *int64 { v := int64(1000); return &v }(),
				IsUnlimited:      true,
				BillingCycle:     "monthly",
			},
			expectedErr: "unlimited packages should not have included_messages",
		},
		{
			name: "Limited without included messages",
			req: CreatePackageRequest{
				ServiceID:    2,
				Name:         "Test",
				Code:         "TEST",
				PriceUSD:     decimal.NewFromFloat(10.00),
				IsUnlimited:  false,
				BillingCycle: "monthly",
			},
			expectedErr: "limited packages must have included_messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			w := httptest.NewRecorder()
			httpReq, _ := http.NewRequest("POST", "/packages", bytes.NewBuffer(body))
			httpReq.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, httpReq)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var resp map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.False(t, resp["success"].(bool))
			assert.Contains(t, resp["error"].(string), tt.expectedErr)
		})
	}
}

func TestListPackages_Success(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.GET("/packages", handler.ListPackages)

	// Execute
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/packages?page=1&page_size=10", nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp ListPackagesResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.GreaterOrEqual(t, resp.TotalCount, 0)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 10, resp.PageSize)
}

func TestListPackages_WithFilters(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.GET("/packages", handler.ListPackages)

	// Execute with filters
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/packages?service_id=2&is_active=true", nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp ListPackagesResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	
	// All returned packages should match filters
	for _, pkg := range resp.Packages {
		assert.Equal(t, int64(2), pkg.ServiceID)
		assert.True(t, pkg.IsActive)
	}
}

func TestGetPackage_Success(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.GET("/packages/:id", handler.GetPackage)

	// Create a test package first
	includedMessages := int64(1000)
	_, err = pool.Exec(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, is_active, display_order
		) VALUES ($1, 2, 'Test Package', 'TEST_GET', 10.00, $2, 0.01, false, 'monthly', false, true, 1)
	`, tenantID, includedMessages)
	require.NoError(t, err)

	// Get the package ID
	var packageID int64
	err = pool.QueryRow(ctx, "SELECT id FROM packages WHERE tenant_id = $1 AND code = 'TEST_GET'", tenantID).Scan(&packageID)
	require.NoError(t, err)

	// Execute
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/packages/"+strconv.FormatInt(packageID, 10), nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["package"])

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
}

func TestGetPackage_NotFound(t *testing.T) {
	// Skip if no database connection
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	require.NoError(t, err)
	defer pool.Close()

	logger := zap.NewNop()
	handler := NewPackageHandler(pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.GET("/packages/:id", handler.GetPackage)

	// Execute with non-existent ID
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/packages/999999", nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp["success"].(bool))
	assert.Contains(t, resp["error"].(string), "not found")
}
