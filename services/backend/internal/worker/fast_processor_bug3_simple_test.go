package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/al-hudhud/backend/internal/db"
)

// TestBug3_MockChannelSelection_Simple is a simplified test that verifies the FIXED
// channel selection behavior against the database.
//
// **Validates: Requirements 1.3, 2.3**
//
// Fixed Behavior (P): getOrSelectChannel() queries the database for an
// available online channel and returns the SAME real channel consistently.
//
// On FIXED code: This test PASSES (real channels are queried from the database)
func TestBug3_MockChannelSelection_Simple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bug detection test in short mode")
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()

	ctx := context.Background()

	// Create a test tenant ID
	tenantID := uuid.New()

	// Insert an online dispatch channel for the tenant so the DB query succeeds
	channelID := uuid.New()
	err := db.GormDB.Exec(`
		INSERT INTO dispatch_channels (id, tenant_id, type, channel_key, config, gateway_id, current_load, status, priority, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, channelID, tenantID, "WEBSOCKET", "ws-channel-"+channelID.String(), nil, nil, 0, "ONLINE", 1, true, time.Now(), time.Now()).Error
	require.NoError(t, err, "Setup: Should insert test dispatch channel")
	t.Cleanup(func() {
		db.GormDB.Exec("DELETE FROM dispatch_channels WHERE id = ?", channelID)
	})

	// Initialize FastMessageProcessor without cache - should query database
	processor := &FastMessageProcessor{
		channelCache: nil, // No cache - should query database
		logger:       logger,
	}

	// Call getOrSelectChannel() twice
	channel1, err1 := processor.getOrSelectChannel(ctx, tenantID)
	channel2, err2 := processor.getOrSelectChannel(ctx, tenantID)

	// FIXED code: Both calls succeed and return real channels from the database
	require.NoError(t, err1, "First call should not return error (database-backed)")
	require.NoError(t, err2, "Second call should not return error (database-backed)")
	require.NotNil(t, channel1, "First call should return a channel")
	require.NotNil(t, channel2, "Second call should return a channel")

	t.Logf("Channel 1 ID: %s, Type: %s", channel1.ChannelID, channel1.Type)
	t.Logf("Channel 2 ID: %s, Type: %s", channel2.ChannelID, channel2.Type)

	// The channel returned must be the real channel stored in the database.
	assert.Equal(t, channelID, channel1.ChannelID,
		"FIXED: getOrSelectChannel() returns the real channel from the database")
	assert.Equal(t, "WEBSOCKET", channel1.Type,
		"FIXED: Channel type comes from the database")
	assert.Equal(t, 1, channel1.Priority,
		"FIXED: Channel priority comes from the database")

	// Both calls must return the SAME channel (consistent database-backed selection).
	assert.Equal(t, channel1.ChannelID, channel2.ChannelID,
		"FIXED: Two calls to getOrSelectChannel() return the SAME real channel from the database. "+
			"The mock uuid.New() behavior has been removed.")

	t.Logf("✓ FIXED: getOrSelectChannel() queries the database and returns a consistent channel")
}

// TestBug3_MockChannelSelection_CodeInspection documents the bug by inspecting the code
// This test always passes but documents the expected behavior
func TestBug3_MockChannelSelection_CodeInspection(t *testing.T) {
	t.Log("=== Bug 3: Mock Channel Selection - Code Inspection ===")
	t.Log("")
	t.Log("Current Implementation (BUGGY):")
	t.Log("  func (fp *FastMessageProcessor) getOrSelectChannel(ctx context.Context, tenantID uuid.UUID) (*redis.CachedChannel, error) {")
	t.Log("      // Try cache first")
	t.Log("      if fp.channelCache != nil {")
	t.Log("          cached, err := fp.channelCache.Get(ctx, tenantID)")
	t.Log("          if err == nil && cached.IsOnline {")
	t.Log("              return cached, nil")
	t.Log("          }")
	t.Log("      }")
	t.Log("")
	t.Log("      // Select new channel (simplified - should use routing engine)")
	t.Log("      // For now, return a mock channel")
	t.Log("      channel := &redis.CachedChannel{")
	t.Log("          ChannelID: uuid.New(),  // ← BUG: Generates random UUID instead of querying database")
	t.Log("          Type:      \"WEBSOCKET\",")
	t.Log("          Priority:  1,")
	t.Log("          IsOnline:  true,")
	t.Log("          CachedAt:  time.Now(),")
	t.Log("      }")
	t.Log("")
	t.Log("      return channel, nil")
	t.Log("  }")
	t.Log("")
	t.Log("Expected Implementation (FIXED):")
	t.Log("  - Query database for available channels")
	t.Log("  - Filter by tenant_id, is_online=true, status='ACTIVE'")
	t.Log("  - Order by priority ASC, current_load ASC")
	t.Log("  - Return real channel from database")
	t.Log("  - Cache the selected channel")
	t.Log("")
	t.Log("Impact:")
	t.Log("  - Messages are routed to non-existent channels")
	t.Log("  - Channel priority and load balancing are ignored")
	t.Log("  - Channel online status is not checked")
	t.Log("  - All messages fail to send")
}
