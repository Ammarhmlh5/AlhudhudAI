package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/al-hudhud/backend/internal/db"
)

// TestBug3_MockChannelSelection_ExplorationTest tests the bug condition:
// getOrSelectChannel() returns mock UUID instead of real channel from database
//
// **Validates: Requirements 1.3, 2.3**
//
// Bug Condition (C): getOrSelectChannel(tenantID) is called
//                    AND real channels exist in database for tenant
//                    AND function returns mock channel with uuid.New()
//                    AND messages are routed to non-existent channel
//
// Expected Behavior (P): When getOrSelectChannel() is called,
//                        the system SHALL query the database for available online channels,
//                        check cache first, and select based on priority and load
//
// CRITICAL: This test MUST FAIL on unfixed code - failure confirms the bug exists
// DO NOT attempt to fix the test or the code when it fails
// NOTE: This test encodes the expected behavior - it will validate the fix when it passes after implementation
// GOAL: Surface counterexamples that demonstrate the bug exists
func TestBug3_MockChannelSelection_ExplorationTest(t *testing.T) {
	// Initialize database
	db.InitGormDB()
	defer db.Close()

	// Initialize logger
	logger, _ := zap.NewDevelopment()

	ctx := context.Background()

	// Create test tenant
	tenantID := uuid.New()
	userID := uuid.New()
	err := db.GormDB.Exec(`
		INSERT INTO users (id, email, password_hash, user_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "bug3user@test.com", "hash", "CLIENT", time.Now(), time.Now()).Error
	require.NoError(t, err, "Failed to create test user")

	defer func() {
		db.GormDB.Exec("DELETE FROM users WHERE id = ?", userID)
	}()

	err = db.GormDB.Exec(`
		INSERT INTO tenants (id, user_id, business_name, contact_name, email, phone, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tenantID, userID, "Test Business Bug3", "Test Contact", "bug3@test.com", "+1234567890", "ACTIVE", time.Now(), time.Now()).Error
	require.NoError(t, err, "Failed to create test tenant")

	// Clean up tenant after test
	defer func() {
		db.GormDB.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	}()

	// Create test gateway for the tenant
	gatewayID := uuid.New()
	err = db.GormDB.Exec(`
		INSERT INTO dedicated_gateways (id, tenant_id, gateway_name, gateway_code, telecom_provider, telecom_api_endpoint, status, contract_start_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, gatewayID, tenantID, "Test Gateway", "test-gw-code", "Test Provider", "http://test.com", "active", time.Now(), time.Now(), time.Now()).Error
	require.NoError(t, err, "Failed to create test gateway")

	// Clean up gateway after test
	defer func() {
		db.GormDB.Exec("DELETE FROM dedicated_gateways WHERE id = ?", gatewayID)
	}()

	// Create real channels in database for this tenant
	realChannelIDs := []uuid.UUID{
		uuid.New(),
		uuid.New(),
		uuid.New(),
	}

	for i, channelID := range realChannelIDs {
		err = db.GormDB.Exec(`
			INSERT INTO dispatch_channels (id, tenant_id, gateway_id, type, channel_key, status, priority, current_load, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, channelID, tenantID, gatewayID, "WEBSOCKET", fmt.Sprintf("channel-%d", i), "ONLINE", i+1, 0, time.Now(), time.Now()).Error
		require.NoError(t, err, "Failed to create test channel %d", i)
	}

	// Clean up channels after test
	defer func() {
		for _, channelID := range realChannelIDs {
			db.GormDB.Exec("DELETE FROM dispatch_channels WHERE id = ?", channelID)
		}
	}()

	// Verify channels exist in database
	var channelCount int64
	err = db.GormDB.Raw(`
		SELECT COUNT(*) 
		FROM dispatch_channels c
		WHERE c.tenant_id = ? AND c.status = 'ONLINE'
	`, tenantID).Scan(&channelCount).Error
	require.NoError(t, err, "Failed to count channels")
	require.Equal(t, int64(3), channelCount, "Expected 3 channels in database")

	// Initialize FastMessageProcessor without cache (to force database query)
	processor := &FastMessageProcessor{
		channelCache: nil, // No cache - should query database
		logger:       logger,
	}

	// Call getOrSelectChannel() - this should query database but currently returns mock
	selectedChannel, err := processor.getOrSelectChannel(ctx, tenantID)

	// On UNFIXED code: This will return a mock channel with uuid.New()
	// The bug is that selectedChannel.ChannelID will NOT be one of realChannelIDs
	
	// Expected behavior (after fix): selectedChannel.ChannelID should be one of realChannelIDs
	// Current behavior (bug): selectedChannel.ChannelID is a random UUID not in database

	if err != nil {
		t.Logf("ERROR: getOrSelectChannel returned error: %v", err)
		t.Logf("This might indicate the bug is already fixed (database query implemented)")
		t.Logf("Or it might indicate a different issue (database connection, schema mismatch)")
		require.NoError(t, err, "getOrSelectChannel should not return error when channels exist")
	}

	require.NotNil(t, selectedChannel, "getOrSelectChannel should return a channel")

	// Check if the returned channel ID is one of the real channels
	isRealChannel := false
	for _, realID := range realChannelIDs {
		if selectedChannel.ChannelID == realID {
			isRealChannel = true
			break
		}
	}

	// Log the result for debugging
	t.Logf("Selected Channel ID: %s", selectedChannel.ChannelID)
	t.Logf("Real Channel IDs: %v", realChannelIDs)
	t.Logf("Is Real Channel: %v", isRealChannel)

	// CRITICAL ASSERTION: This MUST FAIL on unfixed code
	// On unfixed code: selectedChannel.ChannelID will be a random UUID (mock)
	// After fix: selectedChannel.ChannelID will be one of realChannelIDs
	assert.True(t, isRealChannel,
		"BUG DETECTED: getOrSelectChannel() returned mock channel ID %s instead of real channel from database. "+
			"Real channel IDs: %v. This confirms the bug exists.",
		selectedChannel.ChannelID, realChannelIDs)

	// Additional verification: Check that the mock channel is not in database
	if !isRealChannel {
		var existsInDB bool
		err = db.GormDB.Raw("SELECT EXISTS(SELECT 1 FROM dispatch_channels WHERE id = ?)", selectedChannel.ChannelID).Scan(&existsInDB).Error
		require.NoError(t, err, "Failed to check if channel exists in database")
		
		t.Logf("Mock channel exists in database: %v", existsInDB)
		assert.False(t, existsInDB,
			"Mock channel ID %s should NOT exist in database, but it does. This is unexpected.",
			selectedChannel.ChannelID)
	}
}
