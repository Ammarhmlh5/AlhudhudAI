package routing

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use test database connection
	dsn := "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	
	// Test connection
	err = db.Ping()
	require.NoError(t, err)
	
	return db
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, db *sql.DB, tenantID uuid.UUID) {
	_, err := db.Exec("DELETE FROM messages WHERE tenant_id = $1", tenantID)
	require.NoError(t, err)
	
	_, err = db.Exec("DELETE FROM dispatch_channels WHERE tenant_id = $1", tenantID)
	require.NoError(t, err)
}

// createTestChannel creates a test channel in the database
func createTestChannel(t *testing.T, db *sql.DB, tenantID uuid.UUID, channelType, status string, priority int) uuid.UUID {
	channelID := uuid.New()
	channelKey := uuid.New().String()
	
	query := `
		INSERT INTO dispatch_channels (id, tenant_id, type, channel_key, status, priority, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := db.Exec(
		query,
		channelID,
		tenantID,
		channelType,
		channelKey,
		status,
		priority,
		"{}",
		time.Now(),
		time.Now(),
	)
	require.NoError(t, err)
	
	return channelID
}

// TestSelectChannel_OnlineOnly tests that only ONLINE channels are selected
// Validates Requirement 5.3
func TestSelectChannel_OnlineOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create channels with different statuses
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 10)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusWaitMode, 8)
	onlineChannelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	
	// Select channel
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenantID)
	
	// Verify
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, ChannelStatusOnline, channel.Status, "Should only select ONLINE channels")
	assert.Equal(t, onlineChannelID, channel.ID)
}

// TestSelectChannel_PriorityOrdering tests that channels are ordered by priority DESC
// Validates Requirement 5.4 (Priority ordering)
func TestSelectChannel_PriorityOrdering(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create channels with different priorities (all ONLINE)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 3)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	highPriorityID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	
	// Select channel
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenantID)
	
	// Verify
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 10, channel.Priority, "Should select highest priority channel")
	assert.Equal(t, highPriorityID, channel.ID)
}

// TestSelectChannel_LoadBalancing tests that channels with same priority are ordered by load
// Validates Requirement 5.4 (Load ordering)
func TestSelectChannel_LoadBalancing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create channels with same priority
	channel1ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	channel2ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	
	// Create messages to simulate load on channel1
	messageID := uuid.New()
	_, err := db.Exec(`
		INSERT INTO messages (id, tenant_id, channel_id, phone_number, content, status, cost, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, messageID, tenantID, channel1ID, "+1234567890", "Test", "PROCESSING", 1.0, time.Now(), time.Now())
	require.NoError(t, err)
	
	// Select channel
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenantID)
	
	// Verify - should select channel2 (lower load)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, channel2ID, channel.ID, "Should select channel with lower load")
	assert.Equal(t, 0, channel.CurrentLoad, "Selected channel should have 0 load")
}

// TestSelectChannel_NoChannelAvailable tests error when no online channels exist
func TestSelectChannel_NoChannelAvailable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create only offline channels
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 10)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 5)
	
	// Select channel
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenantID)
	
	// Verify
	assert.Error(t, err)
	assert.Nil(t, channel)
	assert.Equal(t, ErrNoChannelAvailable, err)
}

// TestSelectChannel_TenantIsolation tests that channels are isolated by tenant
func TestSelectChannel_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	defer cleanupTestData(t, db, tenant1ID)
	defer cleanupTestData(t, db, tenant2ID)
	
	// Create channels for different tenants
	tenant1ChannelID := createTestChannel(t, db, tenant1ID, "PHONE", ChannelStatusOnline, 10)
	createTestChannel(t, db, tenant2ID, "PHONE", ChannelStatusOnline, 10)
	
	// Select channel for tenant1
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenant1ID)
	
	// Verify - should only get tenant1's channel
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, tenant1ID, channel.TenantID)
	assert.Equal(t, tenant1ChannelID, channel.ID)
}

// TestGetAvailableChannels_ReturnsAllOnline tests getting all online channels
func TestGetAvailableChannels_ReturnsAllOnline(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create mix of online and offline channels
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 8)
	
	// Get available channels
	ctx := context.Background()
	channels, err := engine.GetAvailableChannels(ctx, tenantID)
	
	// Verify
	require.NoError(t, err)
	assert.Len(t, channels, 2, "Should return only ONLINE channels")
	
	// Verify all are online
	for _, ch := range channels {
		assert.Equal(t, ChannelStatusOnline, ch.Status)
		assert.Equal(t, tenantID, ch.TenantID)
	}
}

// TestGetAvailableChannels_OrderedCorrectly tests that channels are ordered properly
func TestGetAvailableChannels_OrderedCorrectly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create channels with different priorities
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 3)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	
	// Get available channels
	ctx := context.Background()
	channels, err := engine.GetAvailableChannels(ctx, tenantID)
	
	// Verify
	require.NoError(t, err)
	require.Len(t, channels, 3)
	
	// Verify ordering (Priority DESC)
	assert.Equal(t, 10, channels[0].Priority)
	assert.Equal(t, 5, channels[1].Priority)
	assert.Equal(t, 3, channels[2].Priority)
}

// TestGetAvailableChannels_EmptyWhenNoOnline tests empty result when no online channels
func TestGetAvailableChannels_EmptyWhenNoOnline(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create only offline channels
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 10)
	
	// Get available channels
	ctx := context.Background()
	channels, err := engine.GetAvailableChannels(ctx, tenantID)
	
	// Verify
	require.NoError(t, err)
	assert.Empty(t, channels, "Should return empty list when no online channels")
}

// TestGetAvailableChannels_TenantIsolation tests tenant isolation
func TestGetAvailableChannels_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // nil Redis client for basic tests
	
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	defer cleanupTestData(t, db, tenant1ID)
	defer cleanupTestData(t, db, tenant2ID)
	
	// Create channels for different tenants
	createTestChannel(t, db, tenant1ID, "PHONE", ChannelStatusOnline, 10)
	createTestChannel(t, db, tenant1ID, "PHONE", ChannelStatusOnline, 5)
	createTestChannel(t, db, tenant2ID, "PHONE", ChannelStatusOnline, 10)
	
	// Get available channels for tenant1
	ctx := context.Background()
	channels, err := engine.GetAvailableChannels(ctx, tenant1ID)
	
	// Verify - should only get tenant1's channels
	require.NoError(t, err)
	assert.Len(t, channels, 2)
	
	for _, ch := range channels {
		assert.Equal(t, tenant1ID, ch.TenantID)
	}
}

// TestSelectChannel_CacheHit tests that cache is used when available
// Validates Requirement 5.1 and 5.2
func TestSelectChannel_CacheHit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	// Setup Redis client for testing
	// Note: This requires Redis to be running on localhost:6379
	// Skip test if Redis is not available
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})
	
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping cache test")
	}
	defer redisClient.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, redisClient, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Clear cache before test
	cacheKey := fmt.Sprintf("channels:cache:%s", tenantID.String())
	redisClient.Del(ctx, cacheKey)
	
	// Create test channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	
	// First call - should query database and cache result
	channel1, err := engine.SelectChannel(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, channel1)
	assert.Equal(t, channelID, channel1.ID)
	
	// Verify cache was set
	exists, err := redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "Cache should be set after first call")
	
	// Second call - should use cache (no database query)
	channel2, err := engine.SelectChannel(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, channel2)
	assert.Equal(t, channelID, channel2.ID)
	
	// Verify TTL is set correctly (should be close to 5 minutes)
	ttl, err := redisClient.TTL(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl.Seconds(), float64(290), "TTL should be close to 5 minutes")
	assert.LessOrEqual(t, ttl.Seconds(), float64(300), "TTL should not exceed 5 minutes")
	
	// Clean up
	redisClient.Del(ctx, cacheKey)
}

// TestGetAvailableChannels_CacheHit tests that cache is used for GetAvailableChannels
// Validates Requirement 5.1 and 5.2
func TestGetAvailableChannels_CacheHit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	// Setup Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})
	
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping cache test")
	}
	defer redisClient.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, redisClient, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Clear cache before test
	cacheKey := fmt.Sprintf("channels:cache:%s", tenantID.String())
	redisClient.Del(ctx, cacheKey)
	
	// Create test channels
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 5)
	
	// First call - should query database and cache result
	channels1, err := engine.GetAvailableChannels(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, channels1, 2)
	
	// Verify cache was set
	exists, err := redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "Cache should be set after first call")
	
	// Second call - should use cache
	channels2, err := engine.GetAvailableChannels(ctx, tenantID)
	require.NoError(t, err)
	assert.Len(t, channels2, 2)
	
	// Verify channels are the same
	assert.Equal(t, channels1[0].ID, channels2[0].ID)
	assert.Equal(t, channels1[1].ID, channels2[1].ID)
	
	// Clean up
	redisClient.Del(ctx, cacheKey)
}

// TestSelectChannel_CacheMiss tests behavior when cache is empty
// Validates Requirement 5.2 (fallback to database)
func TestSelectChannel_CacheMiss(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	// Setup Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})
	
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping cache test")
	}
	defer redisClient.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, redisClient, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Clear cache to ensure miss
	cacheKey := fmt.Sprintf("channels:cache:%s", tenantID.String())
	redisClient.Del(ctx, cacheKey)
	
	// Create test channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	
	// Call with empty cache - should query database
	channel, err := engine.SelectChannel(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, channelID, channel.ID)
	
	// Verify cache was populated
	exists, err := redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "Cache should be populated after database query")
	
	// Clean up
	redisClient.Del(ctx, cacheKey)
}

// TestSelectChannel_CacheIsolation tests that cache is isolated per tenant
// Validates tenant isolation in caching (Requirement 5.2)
func TestSelectChannel_CacheIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	// Setup Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})
	
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping cache test")
	}
	defer redisClient.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, redisClient, logger, "test-secret-key")
	
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	defer cleanupTestData(t, db, tenant1ID)
	defer cleanupTestData(t, db, tenant2ID)
	
	// Clear caches
	cache1Key := fmt.Sprintf("channels:cache:%s", tenant1ID.String())
	cache2Key := fmt.Sprintf("channels:cache:%s", tenant2ID.String())
	redisClient.Del(ctx, cache1Key)
	redisClient.Del(ctx, cache2Key)
	
	// Create channels for different tenants
	channel1ID := createTestChannel(t, db, tenant1ID, "PHONE", ChannelStatusOnline, 10)
	channel2ID := createTestChannel(t, db, tenant2ID, "PHONE", ChannelStatusOnline, 10)
	
	// Get channels for tenant1
	channel1, err := engine.SelectChannel(ctx, tenant1ID)
	require.NoError(t, err)
	assert.Equal(t, channel1ID, channel1.ID)
	assert.Equal(t, tenant1ID, channel1.TenantID)
	
	// Get channels for tenant2
	channel2, err := engine.SelectChannel(ctx, tenant2ID)
	require.NoError(t, err)
	assert.Equal(t, channel2ID, channel2.ID)
	assert.Equal(t, tenant2ID, channel2.TenantID)
	
	// Verify both caches exist independently
	exists1, err := redisClient.Exists(ctx, cache1Key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists1)
	
	exists2, err := redisClient.Exists(ctx, cache2Key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists2)
	
	// Clean up
	redisClient.Del(ctx, cache1Key)
	redisClient.Del(ctx, cache2Key)
}

// TestSelectChannel_WithoutRedis tests that engine works without Redis
// Validates graceful degradation when Redis is unavailable
func TestSelectChannel_WithoutRedis(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key") // No Redis client
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create test channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	
	// Should work without Redis (direct database query)
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenantID)
	
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, channelID, channel.ID)
}

// TestIncrementChannelLoad tests incrementing channel load
// Validates Requirement 5.6 (Load Balancing)
func TestIncrementChannelLoad(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create test channel with initial load = 0
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	// Increment load
	ctx := context.Background()
	err := engine.IncrementChannelLoad(ctx, channelID, tenantID)
	require.NoError(t, err)

	// Verify load was incremented
	var currentLoad int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 1, currentLoad, "Load should be incremented to 1")

	// Increment again
	err = engine.IncrementChannelLoad(ctx, channelID, tenantID)
	require.NoError(t, err)

	// Verify load was incremented again
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 2, currentLoad, "Load should be incremented to 2")
}

// TestDecrementChannelLoad tests decrementing channel load
// Validates Requirement 5.6 (Load Balancing)
func TestDecrementChannelLoad(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create test channel and set initial load
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	_, err := db.Exec("UPDATE dispatch_channels SET current_load = 3 WHERE id = $1", channelID)
	require.NoError(t, err)

	// Decrement load
	ctx := context.Background()
	err = engine.DecrementChannelLoad(ctx, channelID, tenantID)
	require.NoError(t, err)

	// Verify load was decremented
	var currentLoad int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 2, currentLoad, "Load should be decremented to 2")
}

// TestDecrementChannelLoad_PreventNegative tests that load doesn't go negative
// Validates Requirement 5.6 (Load Balancing)
func TestDecrementChannelLoad_PreventNegative(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create test channel with load = 0
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	// Try to decrement when load is already 0
	ctx := context.Background()
	err := engine.DecrementChannelLoad(ctx, channelID, tenantID)
	// Should not return error, but should not decrement below 0
	require.NoError(t, err)

	// Verify load is still 0
	var currentLoad int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 0, currentLoad, "Load should not go below 0")
}

// TestUpdateChannelLoad tests updating channel load to specific value
// Validates Requirement 5.6 (Load Balancing)
func TestUpdateChannelLoad(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create test channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	// Update load to specific value
	ctx := context.Background()
	err := engine.UpdateChannelLoad(ctx, channelID, tenantID, 5)
	require.NoError(t, err)

	// Verify load was updated
	var currentLoad int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 5, currentLoad, "Load should be updated to 5")

	// Update to different value
	err = engine.UpdateChannelLoad(ctx, channelID, tenantID, 10)
	require.NoError(t, err)

	// Verify load was updated again
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 10, currentLoad, "Load should be updated to 10")
}

// TestUpdateChannelLoad_RejectNegative tests that negative load is rejected
// Validates Requirement 5.6 (Load Balancing)
func TestUpdateChannelLoad_RejectNegative(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create test channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	// Try to update load to negative value
	ctx := context.Background()
	err := engine.UpdateChannelLoad(ctx, channelID, tenantID, -5)
	assert.Error(t, err, "Should reject negative load value")

	// Verify load was not changed
	var currentLoad int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channelID).Scan(&currentLoad)
	require.NoError(t, err)
	assert.Equal(t, 0, currentLoad, "Load should remain 0")
}

// TestLoadBalancing_TenantIsolation tests that load updates are isolated by tenant
// Validates tenant isolation in load balancing
func TestLoadBalancing_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	defer cleanupTestData(t, db, tenant1ID)
	defer cleanupTestData(t, db, tenant2ID)

	// Create channels for different tenants
	channel1ID := createTestChannel(t, db, tenant1ID, "PHONE", ChannelStatusOnline, 10)
	channel2ID := createTestChannel(t, db, tenant2ID, "PHONE", ChannelStatusOnline, 10)

	ctx := context.Background()

	// Try to increment tenant1's channel with tenant2's ID (should fail)
	err := engine.IncrementChannelLoad(ctx, channel1ID, tenant2ID)
	assert.Error(t, err, "Should fail when tenant ID doesn't match")

	// Verify tenant1's channel load was not changed
	var load1 int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channel1ID).Scan(&load1)
	require.NoError(t, err)
	assert.Equal(t, 0, load1, "Tenant1's channel load should remain 0")

	// Increment with correct tenant ID should work
	err = engine.IncrementChannelLoad(ctx, channel1ID, tenant1ID)
	require.NoError(t, err)

	// Verify load was incremented
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channel1ID).Scan(&load1)
	require.NoError(t, err)
	assert.Equal(t, 1, load1, "Tenant1's channel load should be 1")

	// Verify tenant2's channel was not affected
	var load2 int
	err = db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channel2ID).Scan(&load2)
	require.NoError(t, err)
	assert.Equal(t, 0, load2, "Tenant2's channel load should remain 0")
}

// TestLoadBalancing_CacheInvalidation tests that cache is invalidated after load updates
// Validates Requirement 5.6 (Cache invalidation)
func TestLoadBalancing_CacheInvalidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Setup Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping cache invalidation test")
	}
	defer redisClient.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, redisClient, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Clear cache
	cacheKey := fmt.Sprintf("channels:cache:%s", tenantID.String())
	redisClient.Del(ctx, cacheKey)

	// Create test channel
	channelID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	// Get channels to populate cache
	_, err := engine.GetAvailableChannels(ctx, tenantID)
	require.NoError(t, err)

	// Verify cache exists
	exists, err := redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "Cache should exist after GetAvailableChannels")

	// Increment load - should invalidate cache
	err = engine.IncrementChannelLoad(ctx, channelID, tenantID)
	require.NoError(t, err)

	// Verify cache was invalidated
	exists, err = redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "Cache should be invalidated after IncrementChannelLoad")

	// Populate cache again
	_, err = engine.GetAvailableChannels(ctx, tenantID)
	require.NoError(t, err)

	// Decrement load - should invalidate cache
	err = engine.DecrementChannelLoad(ctx, channelID, tenantID)
	require.NoError(t, err)

	// Verify cache was invalidated
	exists, err = redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "Cache should be invalidated after DecrementChannelLoad")

	// Populate cache again
	_, err = engine.GetAvailableChannels(ctx, tenantID)
	require.NoError(t, err)

	// Update load - should invalidate cache
	err = engine.UpdateChannelLoad(ctx, channelID, tenantID, 5)
	require.NoError(t, err)

	// Verify cache was invalidated
	exists, err = redisClient.Exists(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "Cache should be invalidated after UpdateChannelLoad")

	// Clean up
	redisClient.Del(ctx, cacheKey)
}

// TestLoadBalancing_SelectsLeastLoaded tests that SelectChannel picks channel with lowest load
// Validates Requirement 5.6 (Load Balancing)
func TestLoadBalancing_SelectsLeastLoaded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create 3 channels with same priority
	channel1ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	channel2ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	channel3ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	ctx := context.Background()

	// Set different loads
	_, err := db.Exec("UPDATE dispatch_channels SET current_load = 5 WHERE id = $1", channel1ID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE dispatch_channels SET current_load = 2 WHERE id = $1", channel2ID)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE dispatch_channels SET current_load = 8 WHERE id = $1", channel3ID)
	require.NoError(t, err)

	// Select channel - should pick channel2 (lowest load = 2)
	channel, err := engine.SelectChannel(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, channel2ID, channel.ID, "Should select channel with lowest load")
	assert.Equal(t, 2, channel.CurrentLoad)
}

// TestLoadBalancing_Distribution tests even distribution across channels
// Validates Requirement 5.6 (Load Balancing)
func TestLoadBalancing_Distribution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")

	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)

	// Create 3 channels with same priority and zero load
	channel1ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	channel2ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)
	channel3ID := createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOnline, 10)

	ctx := context.Background()

	// Simulate sending 9 messages and track distribution
	channelCounts := make(map[uuid.UUID]int)

	for i := 0; i < 9; i++ {
		// Select channel
		channel, err := engine.SelectChannel(ctx, tenantID)
		require.NoError(t, err)
		require.NotNil(t, channel)

		// Track selection
		channelCounts[channel.ID]++

		// Increment load to simulate message assignment
		err = engine.IncrementChannelLoad(ctx, channel.ID, tenantID)
		require.NoError(t, err)
	}

	// Verify distribution is relatively even (each channel should get ~3 messages)
	// Allow some variance due to timing
	assert.Greater(t, channelCounts[channel1ID], 0, "Channel 1 should receive messages")
	assert.Greater(t, channelCounts[channel2ID], 0, "Channel 2 should receive messages")
	assert.Greater(t, channelCounts[channel3ID], 0, "Channel 3 should receive messages")

	// Total should be 9
	total := channelCounts[channel1ID] + channelCounts[channel2ID] + channelCounts[channel3ID]
	assert.Equal(t, 9, total, "Total messages should be 9")

	// Verify final loads match the counts
	var load1, load2, load3 int
	db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channel1ID).Scan(&load1)
	db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channel2ID).Scan(&load2)
	db.QueryRow("SELECT current_load FROM dispatch_channels WHERE id = $1", channel3ID).Scan(&load3)

	assert.Equal(t, channelCounts[channel1ID], load1)
	assert.Equal(t, channelCounts[channel2ID], load2)
	assert.Equal(t, channelCounts[channel3ID], load3)
}

// createTestMessage creates a test message in the database
func createTestMessage(t *testing.T, db *sql.DB, tenantID uuid.UUID, status string) uuid.UUID {
	messageID := uuid.New()
	
	query := `
		INSERT INTO messages (id, tenant_id, phone_number, content, status, cost, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := db.Exec(
		query,
		messageID,
		tenantID,
		"+1234567890",
		"Test message",
		status,
		0.05,
		time.Now(),
		time.Now(),
	)
	require.NoError(t, err)
	
	return messageID
}

// getMessageStatus retrieves the status of a message
func getMessageStatus(t *testing.T, db *sql.DB, messageID uuid.UUID, tenantID uuid.UUID) string {
	var status string
	query := `SELECT status FROM messages WHERE id = $1 AND tenant_id = $2`
	err := db.QueryRow(query, messageID, tenantID).Scan(&status)
	require.NoError(t, err)
	return status
}

// TestSetMessageWaiting_Success tests successful update of message status to WAITING
// Validates Requirement 5.5
func TestSetMessageWaiting_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create a message with PENDING status
	messageID := createTestMessage(t, db, tenantID, "PENDING")
	
	// Set message to WAITING
	ctx := context.Background()
	err := engine.SetMessageWaiting(ctx, messageID, tenantID)
	
	// Verify
	require.NoError(t, err)
	status := getMessageStatus(t, db, messageID, tenantID)
	assert.Equal(t, "WAITING", status, "Message status should be updated to WAITING")
}

// TestSetMessageWaiting_ProcessingStatus tests update from PROCESSING status
// Validates Requirement 5.5
func TestSetMessageWaiting_ProcessingStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create a message with PROCESSING status
	messageID := createTestMessage(t, db, tenantID, "PROCESSING")
	
	// Set message to WAITING
	ctx := context.Background()
	err := engine.SetMessageWaiting(ctx, messageID, tenantID)
	
	// Verify
	require.NoError(t, err)
	status := getMessageStatus(t, db, messageID, tenantID)
	assert.Equal(t, "WAITING", status, "Message status should be updated to WAITING from PROCESSING")
}

// TestSetMessageWaiting_TenantIsolation tests that tenant isolation is enforced
// Validates tenant isolation requirement
func TestSetMessageWaiting_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")
	
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()
	defer cleanupTestData(t, db, tenant1ID)
	defer cleanupTestData(t, db, tenant2ID)
	
	// Create a message for tenant1
	messageID := createTestMessage(t, db, tenant1ID, "PENDING")
	
	// Try to update message using tenant2 ID (should fail)
	ctx := context.Background()
	err := engine.SetMessageWaiting(ctx, messageID, tenant2ID)
	
	// Verify
	require.Error(t, err, "Should fail when tenant ID doesn't match")
	assert.Contains(t, err.Error(), "message not found, tenant mismatch, or invalid status")
	
	// Verify original status unchanged
	status := getMessageStatus(t, db, messageID, tenant1ID)
	assert.Equal(t, "PENDING", status, "Message status should remain unchanged")
}

// TestSetMessageWaiting_InvalidStatus tests that only PENDING/PROCESSING messages can be updated
// Validates Requirement 5.5
func TestSetMessageWaiting_InvalidStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create a message with SENT status (should not be updatable to WAITING)
	messageID := createTestMessage(t, db, tenantID, "SENT")
	
	// Try to set message to WAITING
	ctx := context.Background()
	err := engine.SetMessageWaiting(ctx, messageID, tenantID)
	
	// Verify
	require.Error(t, err, "Should fail when message status is not PENDING or PROCESSING")
	assert.Contains(t, err.Error(), "message not found, tenant mismatch, or invalid status")
	
	// Verify original status unchanged
	status := getMessageStatus(t, db, messageID, tenantID)
	assert.Equal(t, "SENT", status, "Message status should remain unchanged")
}

// TestSetMessageWaiting_NonExistentMessage tests handling of non-existent message
func TestSetMessageWaiting_NonExistentMessage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")
	
	tenantID := uuid.New()
	nonExistentMessageID := uuid.New()
	
	// Try to update non-existent message
	ctx := context.Background()
	err := engine.SetMessageWaiting(ctx, nonExistentMessageID, tenantID)
	
	// Verify
	require.Error(t, err, "Should fail when message doesn't exist")
	assert.Contains(t, err.Error(), "message not found, tenant mismatch, or invalid status")
}

// TestSelectChannel_NoChannelAvailable_Integration tests the complete flow
// when no channels are available
// Validates Requirement 5.5 (integration test)
func TestSelectChannel_NoChannelAvailable_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	logger := zap.NewNop()
	engine := NewEngine(db, nil, logger, "test-secret-key")
	
	tenantID := uuid.New()
	defer cleanupTestData(t, db, tenantID)
	
	// Create only OFFLINE channels
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 10)
	createTestChannel(t, db, tenantID, "PHONE", ChannelStatusOffline, 8)
	
	// Create a message
	messageID := createTestMessage(t, db, tenantID, "PENDING")
	
	// Try to select channel
	ctx := context.Background()
	channel, err := engine.SelectChannel(ctx, tenantID)
	
	// Verify no channel available
	require.Error(t, err)
	assert.Equal(t, ErrNoChannelAvailable, err)
	assert.Nil(t, channel)
	
	// Now set message to WAITING
	err = engine.SetMessageWaiting(ctx, messageID, tenantID)
	require.NoError(t, err)
	
	// Verify message status updated
	status := getMessageStatus(t, db, messageID, tenantID)
	assert.Equal(t, "WAITING", status, "Message should be in WAITING status when no channels available")
}
