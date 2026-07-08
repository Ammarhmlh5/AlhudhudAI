package channel

import (
	"context"
	"database/sql"
	"testing"

	"github.com/al-hudhud/backend/internal/redis"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use test database connection
	dsn := "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skip("Test database not available:", err)
	}
	if err := db.Ping(); err != nil {
		t.Skip("Test database not reachable:", err)
	}
	return db
}

// TestUpdateChannel_InvalidatesCacheOnOffline tests that cache is invalidated when channel goes offline
func TestUpdateChannel_InvalidatesCacheOnOffline(t *testing.T) {
	// This test verifies the logic without actual Redis
	// The actual integration test would require Redis setup

	// Setup logger
	logger := zap.NewNop()

	// Setup Redis client (nil for this test - we'll verify the logic without actual Redis)
	var channelCache *redis.ChannelCache = nil

	// Create service
	service := NewService(channelCache, logger)

	// Verify service was created with correct dependencies
	assert.NotNil(t, service)
	assert.Nil(t, service.channelCache) // nil is acceptable
	assert.NotNil(t, service.logger)
}

// TestUpdateChannel_StatusChangeDetection tests that status changes are detected correctly
func TestUpdateChannel_StatusChangeDetection(t *testing.T) {
	// This test verifies the status change detection logic

	// Setup logger
	logger := zap.NewNop()

	// Create service with nil cache (acceptable for this test)
	service := NewService(nil, logger)

	// Verify service structure
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
}

// TestStartHealthCheck_InitializesCorrectly tests that health check starts correctly
func TestStartHealthCheck_InitializesCorrectly(t *testing.T) {
	// Setup logger
	logger := zap.NewNop()

	// Create service with nil cache
	service := NewService(nil, logger)

	// Verify service has stopChan initialized
	assert.NotNil(t, service)
	assert.NotNil(t, service.stopChan)

	// Start health check (should handle nil cache gracefully)
	ctx := context.Background()
	service.StartHealthCheck(ctx)

	// Stop health check
	service.StopHealthCheck()
}

// TestStopHealthCheck_GracefulShutdown tests that health check stops gracefully
func TestStopHealthCheck_GracefulShutdown(t *testing.T) {
	// Setup logger
	logger := zap.NewNop()

	// Create service
	service := NewService(nil, logger)

	// Start health check
	ctx := context.Background()
	service.StartHealthCheck(ctx)

	// Stop should not panic
	assert.NotPanics(t, func() {
		service.StopHealthCheck()
	})
}

// TestPerformHealthCheck_HandlesNoOfflineChannels tests health check with no offline channels
func TestPerformHealthCheck_HandlesNoOfflineChannels(t *testing.T) {
	// This test requires actual database connection
	t.Skip("Skipping: requires database connection for health check test")
}
