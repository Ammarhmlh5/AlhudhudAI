package routing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestBug23_VerifyChannelOnline_Online tests health check for online channel
// Validates: Bug 23 - Task 23.2.2 - Health check verification
func TestBug23_VerifyChannelOnline_Online(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenantID := uuid.New()
	
	// Create an online channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	
	// Verify channel is online
	isOnline, err := engine.VerifyChannelOnline(ctx, channelID, tenantID)
	require.NoError(t, err)
	assert.True(t, isOnline, "Channel should be online")
	
	t.Log("✅ Health check correctly identifies online channel")
}

// TestBug23_VerifyChannelOnline_Offline tests health check for offline channel
// Validates: Bug 23 - Task 23.2.2 - Health check verification
func TestBug23_VerifyChannelOnline_Offline(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenantID := uuid.New()
	
	// Create an offline channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 10)
	
	// Verify channel is offline
	isOnline, err := engine.VerifyChannelOnline(ctx, channelID, tenantID)
	require.NoError(t, err)
	assert.False(t, isOnline, "Channel should be offline")
	
	t.Log("✅ Health check correctly identifies offline channel")
}

// TestBug23_VerifyChannelOnline_WaitMode tests health check for channel in wait mode
// Validates: Bug 23 - Task 23.2.2 - Health check verification
func TestBug23_VerifyChannelOnline_WaitMode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenantID := uuid.New()
	
	// Create a channel in wait mode
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusWaitMode, 10)
	
	// Verify channel is not online (wait mode is not online)
	isOnline, err := engine.VerifyChannelOnline(ctx, channelID, tenantID)
	require.NoError(t, err)
	assert.False(t, isOnline, "Channel in wait mode should not be considered online")
	
	t.Log("✅ Health check correctly identifies channel in wait mode as not online")
}

// TestBug23_VerifyChannelOnline_NotFound tests health check for non-existent channel
// Validates: Bug 23 - Task 23.2.2 - Health check verification
func TestBug23_VerifyChannelOnline_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenantID := uuid.New()
	nonExistentChannelID := uuid.New()
	
	// Verify non-existent channel
	isOnline, err := engine.VerifyChannelOnline(ctx, nonExistentChannelID, tenantID)
	require.NoError(t, err)
	assert.False(t, isOnline, "Non-existent channel should return false")
	
	t.Log("✅ Health check correctly handles non-existent channel")
}

// TestBug23_VerifyChannelOnline_TenantIsolation tests that health check respects tenant isolation
// Validates: Bug 23 - Task 23.2.2 - Health check verification with tenant isolation
func TestBug23_VerifyChannelOnline_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	
	// Create channel for tenant1
	channelID := createTestChannel(t, db, tenant1ID, "PHONE", ChannelStatusOnline, 10)
	
	// Try to verify channel with wrong tenant ID
	isOnline, err := engine.VerifyChannelOnline(ctx, channelID, tenant2ID)
	require.NoError(t, err)
	assert.False(t, isOnline, "Channel should not be found for different tenant")
	
	// Verify with correct tenant ID
	isOnline, err = engine.VerifyChannelOnline(ctx, channelID, tenant1ID)
	require.NoError(t, err)
	assert.True(t, isOnline, "Channel should be found for correct tenant")
	
	t.Log("✅ Health check respects tenant isolation")
}

// TestBug23_HealthCheckScenario_StaleCache tests the complete scenario of stale cache detection
// Validates: Bug 23 - Task 23.2.2 - Complete health check scenario
func TestBug23_HealthCheckScenario_StaleCache(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenantID := uuid.New()
	
	// Step 1: Create an online channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	
	// Step 2: Verify channel is online
	isOnline, err := engine.VerifyChannelOnline(ctx, channelID, tenantID)
	require.NoError(t, err)
	assert.True(t, isOnline, "Channel should be online initially")
	
	// Step 3: Simulate channel going offline (update database)
	_, err = db.Exec(`
		UPDATE dispatch_channels 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND tenant_id = $3
	`, ChannelStatusOffline, channelID, tenantID)
	require.NoError(t, err, "Failed to update channel status")
	
	// Step 4: Health check should now detect channel is offline
	isOnline, err = engine.VerifyChannelOnline(ctx, channelID, tenantID)
	require.NoError(t, err)
	assert.False(t, isOnline, "Channel should be offline after status change")
	
	// Step 5: In real implementation, this would trigger cache invalidation
	err = engine.invalidateChannelCache(ctx, tenantID)
	require.NoError(t, err, "Failed to invalidate cache")
	
	t.Log("✅ Complete health check scenario works correctly")
	t.Log("   - Channel status change detected")
	t.Log("   - Cache invalidation triggered")
	t.Log("   - Next request will query database for online channels")
}

// TestBug23_Integration_HealthCheckWithCacheInvalidation tests integration of health check and cache invalidation
// Validates: Bug 23 - Tasks 23.2.1 and 23.2.2 - Integration test
func TestBug23_Integration_HealthCheckWithCacheInvalidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger, _ := zap.NewDevelopment()
	engine := NewEngine(db, nil, logger, "test-secret")
	
	ctx := context.Background()
	tenantID := uuid.New()
	
	// Create multiple channels
	channel1ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	channel2ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 8)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 5)
	
	// Select channel (should cache results)
	selectedChannel, err := engine.SelectChannel(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, selectedChannel)
	assert.Equal(t, channel1ID, selectedChannel.ID, "Should select highest priority online channel")
	
	// Simulate: channel1 goes offline
	_, err = db.Exec(`
		UPDATE dispatch_channels 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2
	`, ChannelStatusOffline, channel1ID)
	require.NoError(t, err)
	
	// Health check detects channel1 is offline
	isOnline, err := engine.VerifyChannelOnline(ctx, channel1ID, tenantID)
	require.NoError(t, err)
	assert.False(t, isOnline, "Channel1 should be offline")
	
	// Invalidate cache (active invalidation)
	err = engine.invalidateChannelCache(ctx, tenantID)
	require.NoError(t, err)
	
	// Select channel again (should query database and get channel2)
	selectedChannel, err = engine.SelectChannel(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, selectedChannel)
	assert.Equal(t, channel2ID, selectedChannel.ID, "Should select next highest priority online channel")
	
	t.Log("✅ Integration test passed:")
	t.Log("   - Health check detected offline channel")
	t.Log("   - Cache invalidation triggered")
	t.Log("   - Next selection correctly chose different online channel")
}

// TestBug23_Documentation_HealthCheckStrategy documents the health check strategy
func TestBug23_Documentation_HealthCheckStrategy(t *testing.T) {
	t.Log("=== Bug 23: Health Check Verification Strategy ===")
	t.Log("")
	t.Log("IMPLEMENTATION:")
	t.Log("  1. VerifyChannelOnline(channelID, tenantID) queries database")
	t.Log("  2. Returns true if channel status is ONLINE")
	t.Log("  3. Returns false if channel is OFFLINE or WAIT_MODE")
	t.Log("  4. Respects tenant isolation")
	t.Log("")
	t.Log("USAGE SCENARIOS:")
	t.Log("  A. Periodic Health Check:")
	t.Log("     - Every Nth request (e.g., 1 in 10)")
	t.Log("     - Verify cached channel is still online")
	t.Log("     - Invalidate cache if offline")
	t.Log("")
	t.Log("  B. On-Demand Verification:")
	t.Log("     - When message send fails")
	t.Log("     - Verify channel status")
	t.Log("     - Invalidate cache if offline")
	t.Log("")
	t.Log("  C. Event-Driven Invalidation:")
	t.Log("     - Channel status update event")
	t.Log("     - Immediately invalidate cache")
	t.Log("     - No need to wait for health check")
	t.Log("")
	t.Log("BENEFITS:")
	t.Log("  - Reduces stale data window from 5 min to 1 min (or less)")
	t.Log("  - Active invalidation provides immediate updates")
	t.Log("  - Health checks catch edge cases")
	t.Log("  - Maintains performance with caching")
	t.Log("")
	t.Log("✅ Health check strategy documented")
}
