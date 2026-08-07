package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/al-hudhud/backend/internal/rollover"
	"github.com/al-hudhud/backend/internal/subscription"
)

func TestSubscribe_Success(t *testing.T) {
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
	subManager := subscription.NewManager(pool, logger)
	rolloverManager := rollover.NewManager(pool, logger)
	handler := NewSubscriptionHandler(subManager, rolloverManager, pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.POST("/subscriptions", handler.Subscribe)

	// Create test tenant with sufficient balance
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, 'Test Tenant', 'Test Contact', 'test@example.com', 100.00, NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	// Create test package
	includedMessages := int64(1000)
	var packageID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, is_active, display_order
		) VALUES ($1, 2, 'Test Package', 'TEST_SUB', 10.00, $2, 0.01, false, 'monthly', false, true, 1)
		RETURNING id
	`, tenantID, includedMessages).Scan(&packageID)
	require.NoError(t, err)

	// Subscribe request
	req := SubscribeRequest{
		PackageID: packageID,
		AutoRenew: true,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/subscriptions", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute
	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp SubscribeResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Greater(t, resp.SubscriptionID, int64(0))
	assert.Contains(t, resp.Message, "Test Package")

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", resp.SubscriptionID)
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}

func TestSubscribe_InsufficientBalance(t *testing.T) {
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
	subManager := subscription.NewManager(pool, logger)
	rolloverManager := rollover.NewManager(pool, logger)
	handler := NewSubscriptionHandler(subManager, rolloverManager, pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.POST("/subscriptions", handler.Subscribe)

	// Create test tenant with insufficient balance
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, 'Test Tenant', 'Test Contact', 'test@example.com', 5.00, NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	// Create test package
	includedMessages := int64(1000)
	var packageID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, is_active, display_order
		) VALUES ($1, 2, 'Test Package', 'TEST_SUB_INSUF', 10.00, $2, 0.01, false, 'monthly', false, true, 1)
		RETURNING id
	`, tenantID, includedMessages).Scan(&packageID)
	require.NoError(t, err)

	// Subscribe request
	req := SubscribeRequest{
		PackageID: packageID,
		AutoRenew: false,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/subscriptions", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute
	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusPaymentRequired, w.Code)

	var resp SubscribeResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "Insufficient balance")

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}

func TestSubscribe_InactivePackage(t *testing.T) {
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
	subManager := subscription.NewManager(pool, logger)
	rolloverManager := rollover.NewManager(pool, logger)
	handler := NewSubscriptionHandler(subManager, rolloverManager, pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.POST("/subscriptions", handler.Subscribe)

	// Create test tenant
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, 'Test Tenant', 'Test Contact', 'test@example.com', 100.00, NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	// Create inactive package
	includedMessages := int64(1000)
	var packageID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, is_active, display_order
		) VALUES ($1, 2, 'Inactive Package', 'TEST_SUB_INACTIVE', 10.00, $2, 0.01, false, 'monthly', false, false, 1)
		RETURNING id
	`, tenantID, includedMessages).Scan(&packageID)
	require.NoError(t, err)

	// Subscribe request
	req := SubscribeRequest{
		PackageID: packageID,
		AutoRenew: false,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/subscriptions", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute
	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp SubscribeResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "not active")

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}

func TestGetUsage_Success(t *testing.T) {
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
	subManager := subscription.NewManager(pool, logger)
	rolloverManager := rollover.NewManager(pool, logger)
	handler := NewSubscriptionHandler(subManager, rolloverManager, pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.GET("/subscriptions/usage", handler.GetUsage)

	// Create test tenant
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, 'Test Tenant', 'Test Contact', 'test@example.com', 100.00, NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	// Create test package
	includedMessages := int64(1000)
	var packageID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, is_active, display_order
		) VALUES ($1, 2, 'Test Package', 'TEST_USAGE', 10.00, $2, 0.01, false, 'monthly', false, true, 1)
		RETURNING id
	`, tenantID, includedMessages).Scan(&packageID)
	require.NoError(t, err)

	// Create active subscription
	var subscriptionID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO subscriptions (
			tenant_id, package_id, start_date, end_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days, next_renewal_date
		) VALUES (
			$1, $2, NOW(), NOW() + INTERVAL '30 days',
			100, 900, 0, 0,
			'active', true, 3, NOW() + INTERVAL '30 days'
		)
		RETURNING id
	`, tenantID, packageID).Scan(&subscriptionID)
	require.NoError(t, err)

	// Execute
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/subscriptions/usage", nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp GetUsageResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Usage)
	assert.Equal(t, subscriptionID, resp.Usage.SubscriptionID)
	assert.Equal(t, int64(100), resp.Usage.MessagesUsed)
	assert.Equal(t, int64(900), *resp.Usage.MessagesRemaining)

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", subscriptionID)
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}

func TestGetUsage_NoActiveSubscription(t *testing.T) {
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
	subManager := subscription.NewManager(pool, logger)
	rolloverManager := rollover.NewManager(pool, logger)
	handler := NewSubscriptionHandler(subManager, rolloverManager, pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.GET("/subscriptions/usage", handler.GetUsage)

	// Create test tenant without subscription
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, 'Test Tenant', 'Test Contact', 'test@example.com', 100.00, NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	// Execute
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/subscriptions/usage", nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.False(t, resp["success"].(bool))
	assert.Contains(t, resp["error"].(string), "No active subscription")

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}

func TestCancelSubscription_Success(t *testing.T) {
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
	subManager := subscription.NewManager(pool, logger)
	rolloverManager := rollover.NewManager(pool, logger)
	handler := NewSubscriptionHandler(subManager, rolloverManager, pool, logger)

	tenantID := uuid.New()
	router := setupTestRouter(tenantID, "user")
	router.PUT("/subscriptions/:id/cancel", handler.CancelSubscription)

	// Create test data
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, business_name, contact_name, email, balance, created_at, updated_at)
		VALUES ($1, 'Test Tenant', 'Test Contact', 'test@example.com', 100.00, NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	includedMessages := int64(1000)
	var packageID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, service_id, name, code, price_usd,
			included_messages, overage_rate, is_unlimited, billing_cycle,
			rollover_enabled, is_active, display_order
		) VALUES ($1, 2, 'Test Package', 'TEST_CANCEL', 10.00, $2, 0.01, false, 'monthly', false, true, 1)
		RETURNING id
	`, tenantID, includedMessages).Scan(&packageID)
	require.NoError(t, err)

	var subscriptionID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO subscriptions (
			tenant_id, package_id, start_date, end_date,
			messages_used, messages_remaining, overage_messages, overage_cost,
			status, auto_renew, grace_period_days, next_renewal_date
		) VALUES (
			$1, $2, NOW(), NOW() + INTERVAL '30 days',
			0, 1000, 0, 0,
			'active', true, 3, NOW() + INTERVAL '30 days'
		)
		RETURNING id
	`, tenantID, packageID).Scan(&subscriptionID)
	require.NoError(t, err)

	// Execute
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/subscriptions/"+strconv.FormatInt(subscriptionID, 10)+"/cancel", nil)

	router.ServeHTTP(w, httpReq)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp["success"].(bool))
	assert.Contains(t, resp["message"].(string), "cancelled")

	// Verify auto_renew is false
	var autoRenew bool
	err = pool.QueryRow(ctx, "SELECT auto_renew FROM subscriptions WHERE id = $1", subscriptionID).Scan(&autoRenew)
	require.NoError(t, err)
	assert.False(t, autoRenew)

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", subscriptionID)
	_, _ = pool.Exec(ctx, "DELETE FROM packages WHERE id = $1", packageID)
	_, _ = pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
}
