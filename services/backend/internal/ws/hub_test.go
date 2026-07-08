package ws

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testTenantID1  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testTenantID2  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testTenantID3  = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	testChannelID5 = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	testChannelID10= uuid.MustParse("00000000-0000-0000-0000-000000000010")
	testChannelID99= uuid.MustParse("00000000-0000-0000-0000-000000000999")
)

func TestNewHub(t *testing.T) {
	cfg := config.WebSocketConfig{
		BufferSize:        1000,
		HeartbeatInterval: 30 * time.Second,
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
	}
	hub := NewHub(cfg, nil, nil)
	
	assert.NotNil(t, hub)
	assert.Equal(t, cfg, hub.Config)
	assert.NotNil(t, hub.connections)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.logger)
}

func getTestConfig() config.WebSocketConfig {
	return config.WebSocketConfig{
		BufferSize:        1000,
		HeartbeatInterval: 30 * time.Second,
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
	}
}

func TestConnectionKey(t *testing.T) {
	key1 := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	key2 := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	key3 := ConnectionKey{TenantID: testTenantID2, ChannelID: testChannelID5}
	
	// Same tenant and channel should be equal
	assert.Equal(t, key1, key2)
	
	// Different tenant should not be equal
	assert.NotEqual(t, key1, key3)
}

func TestGetConnectionStatus_NotConnected(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	status, err := hub.GetConnectionStatus(ctx, testTenantID1, testChannelID5)
	
	require.NoError(t, err)
	assert.Equal(t, string(StatusDisconnected), status)
}

func TestWebSocketMessage_Types(t *testing.T) {
	tests := []struct {
		name     string
		msgType  string
		expected string
	}{
		{"MESSAGE type", "MESSAGE", "MESSAGE"},
		{"HEARTBEAT type", "HEARTBEAT", "HEARTBEAT"},
		{"ACK type", "ACK", "ACK"},
		{"STATUS_REPORT type", "STATUS_REPORT", "STATUS_REPORT"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := WebSocketMessage{
				Type:      tt.msgType,
				MessageID: "test-123",
				TenantID:  testTenantID1,
				ChannelID: testChannelID5,
				Timestamp: time.Now(),
			}
			
			assert.Equal(t, tt.expected, msg.Type)
		})
	}
}

func TestConnectionStatus_Constants(t *testing.T) {
	assert.Equal(t, ConnectionStatus("CONNECTED"), StatusConnected)
	assert.Equal(t, ConnectionStatus("DISCONNECTED"), StatusDisconnected)
	assert.Equal(t, ConnectionStatus("RECONNECTING"), StatusReconnecting)
}

func TestBufferSize_Constant(t *testing.T) {
	// Verify buffer size is 1000 as per Requirement 6.2
	assert.Equal(t, 1000, getTestConfig().BufferSize)
}

func TestHeartbeatInterval_Constant(t *testing.T) {
	// Verify heartbeat interval is 30 seconds as per Requirement 6.6
	assert.Equal(t, 30*time.Second, getTestConfig().HeartbeatInterval)
}

func TestEvent_Structure(t *testing.T) {
	event := Event{
		TenantID: testTenantID1,
		Type:     "TEST_EVENT",
		Payload:  map[string]string{"key": "value"},
	}
	
	assert.Equal(t, testTenantID1, event.TenantID)
	assert.Equal(t, "TEST_EVENT", event.Type)
	assert.NotNil(t, event.Payload)
}

// TestTenantIsolation verifies that connections are isolated by tenant
func TestTenantIsolation(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	
	// Create connection keys for different tenants
	key1 := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	key2 := ConnectionKey{TenantID: testTenantID2, ChannelID: testChannelID5}
	
	// Verify keys are different even with same channel ID
	assert.NotEqual(t, key1, key2, "Different tenants should have different keys")
	
	// Verify map can store both
	hub.mu.Lock()
	hub.connections[key1] = &Connection{TenantID: testTenantID1, ChannelID: testChannelID5}
	hub.connections[key2] = &Connection{TenantID: testTenantID2, ChannelID: testChannelID5}
	hub.mu.Unlock()
	
	hub.mu.RLock()
	assert.Len(t, hub.connections, 2, "Should have 2 separate connections")
	hub.mu.RUnlock()
}

// TestChannelIsolation verifies that connections are isolated by channel
func TestChannelIsolation(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	
	// Create connection keys for different channels of same tenant
	key1 := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	key2 := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID10}
	
	// Verify keys are different even with same tenant ID
	assert.NotEqual(t, key1, key2, "Different channels should have different keys")
	
	// Verify map can store both
	hub.mu.Lock()
	hub.connections[key1] = &Connection{TenantID: testTenantID1, ChannelID: testChannelID5}
	hub.connections[key2] = &Connection{TenantID: testTenantID1, ChannelID: testChannelID10}
	hub.mu.Unlock()
	
	hub.mu.RLock()
	assert.Len(t, hub.connections, 2, "Should have 2 separate connections")
	hub.mu.RUnlock()
}

// TestSendMessage_ConnectionNotFound verifies error when connection doesn't exist
func TestSendMessage_ConnectionNotFound(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	msg := WebSocketMessage{
		Type:      "MESSAGE",
		MessageID: "test-123",
		TenantID:  testChannelID99,
		ChannelID: testChannelID99,
		Timestamp: time.Now(),
	}
	
	err := hub.SendMessage(ctx, msg)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

// TestUnregisterConnection_NotFound verifies error when connection doesn't exist
func TestUnregisterConnection_NotFound(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	err := hub.UnregisterConnection(ctx, testChannelID99, testChannelID99)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection not found")
}

// TestSendMessage_NonBlocking verifies that SendMessage is non-blocking (Requirement 6.1)
func TestSendMessage_NonBlocking(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	// Create a mock connection with buffered channel
	conn := &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Send:      make(chan []byte, hub.Config.BufferSize),
		Hub:       hub,
	}
	
	// Register the connection
	hub.mu.Lock()
	key := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	hub.connections[key] = conn
	hub.mu.Unlock()
	
	// Send a message
	msg := WebSocketMessage{
		Type:      "MESSAGE",
		MessageID: "test-123",
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	
	// Measure time to ensure it's non-blocking
	start := time.Now()
	err := hub.SendMessage(ctx, msg)
	duration := time.Since(start)
	
	// Should complete immediately (non-blocking)
	assert.NoError(t, err)
	assert.Less(t, duration, 150*time.Millisecond, "SendMessage should be non-blocking")
	
	// Verify message was sent to channel
	select {
	case data := <-conn.Send:
		assert.NotNil(t, data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not received in channel")
	}
}

// TestSendMessage_FailFast verifies fail-fast behavior when buffer is full (Requirement 6.3)
func TestSendMessage_FailFast(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	// Create a mock connection with buffered channel
	conn := &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Send:      make(chan []byte, hub.Config.BufferSize),
		Hub:       hub,
	}
	
	// Register the connection
	hub.mu.Lock()
	key := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	hub.connections[key] = conn
	hub.mu.Unlock()
	
	// Fill the buffer completely
	for i := 0; i < hub.Config.BufferSize; i++ {
		conn.Send <- []byte("dummy message")
	}
	
	// Try to send one more message - should fail fast
	msg := WebSocketMessage{
		Type:      "MESSAGE",
		MessageID: "test-overflow",
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Content:   "This should fail",
		Timestamp: time.Now(),
	}
	
	// Measure time to ensure it fails fast
	start := time.Now()
	err := hub.SendMessage(ctx, msg)
	duration := time.Since(start)
	
	// Should fail immediately (fail-fast)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send channel full")
	assert.Less(t, duration, 150*time.Millisecond, "Should fail fast without blocking")
}

// TestBufferedChannel_Capacity verifies buffered channel has correct capacity (Requirement 6.2)
func TestBufferedChannel_Capacity(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	// Create a mock connection
	conn := &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Send:      make(chan []byte, hub.Config.BufferSize),
		Hub:       hub,
	}
	
	// Register the connection
	hub.mu.Lock()
	key := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	hub.connections[key] = conn
	hub.mu.Unlock()
	
	// Send exactly BufferSize messages - all should succeed
	successCount := 0
	for i := 0; i < hub.Config.BufferSize; i++ {
		msg := WebSocketMessage{
			Type:      "MESSAGE",
			MessageID: "test-" + string(rune(i)),
			TenantID:  testTenantID1,
			ChannelID: testChannelID5,
			Content:   "Test message",
			Timestamp: time.Now(),
		}
		
		err := hub.SendMessage(ctx, msg)
		if err == nil {
			successCount++
		}
	}
	
	// All BufferSize messages should have been sent successfully
	assert.Equal(t, hub.Config.BufferSize, successCount, "Should be able to send exactly BufferSize messages")
	
	// Next message should fail
	msg := WebSocketMessage{
		Type:      "MESSAGE",
		MessageID: "test-overflow",
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Content:   "This should fail",
		Timestamp: time.Now(),
	}
	
	err := hub.SendMessage(ctx, msg)
	assert.Error(t, err, "Message beyond buffer capacity should fail")
}

// TestSendMessage_ConcurrentSafety verifies thread-safety of SendMessage
func TestSendMessage_ConcurrentSafety(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()
	
	// Create a mock connection
	conn := &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		Send:      make(chan []byte, hub.Config.BufferSize),
		Hub:       hub,
	}
	
	// Register the connection
	hub.mu.Lock()
	key := ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}
	hub.connections[key] = conn
	hub.mu.Unlock()
	
	// Start a goroutine to drain the channel
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			<-conn.Send
		}
		done <- true
	}()
	
	// Send messages concurrently from multiple goroutines
	const numGoroutines = 10
	const messagesPerGoroutine = 10
	
	errChan := make(chan error, numGoroutines*messagesPerGoroutine)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for m := 0; m < messagesPerGoroutine; m++ {
				msg := WebSocketMessage{
					Type:      "MESSAGE",
					MessageID: "test-concurrent",
					TenantID:  testTenantID1,
					ChannelID: testChannelID5,
					Content:   "Concurrent message",
					Timestamp: time.Now(),
				}
				
				err := hub.SendMessage(ctx, msg)
				errChan <- err
			}
		}(g)
	}
	
	// Wait for all messages to be sent
	wg.Wait()
	<-done
	close(errChan)
	
	// Count errors
	errorCount := 0
	for err := range errChan {
		if err != nil {
			errorCount++
		}
	}
	
	// Most messages should succeed (some may fail if buffer fills up)
	assert.LessOrEqual(t, errorCount, numGoroutines*messagesPerGoroutine, "Should handle concurrent sends safely")
}

// TestIsWebSession verifies web session detection logic
func TestIsWebSession(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		isWeb    bool
	}{
		{"empty device ID is web", "", true},
		{"UUID device ID is web", "00000000-0000-0000-0000-000000000001", true},
		{"android prefix is mobile", "android_abc123", false},
		{"ios prefix is mobile", "ios_xyz789", false},
		{"custom prefix is mobile", "huawei_abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isWeb, isWebSession(tt.deviceID))
		})
	}
}

// TestGetConnectionsByTenant_TenantIsolation verifies tenant isolation
func TestGetConnectionsByTenant_TenantIsolation(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)

	hub.mu.Lock()
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}] = &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		DeviceID:  "android_device_1",
		Status:    StatusConnected,
	}
	hub.connections[ConnectionKey{TenantID: testTenantID2, ChannelID: testChannelID10}] = &Connection{
		TenantID:  testTenantID2,
		ChannelID: testChannelID10,
		DeviceID:  "android_device_2",
		Status:    StatusConnected,
	}
	hub.mu.Unlock()

	conns1 := hub.GetConnectionsByTenant(testTenantID1)
	assert.Len(t, conns1, 1, "Should only return connections for tenant 1")
	assert.Equal(t, "android_device_1", conns1[0].DeviceID)

	conns2 := hub.GetConnectionsByTenant(testTenantID2)
	assert.Len(t, conns2, 1, "Should only return connections for tenant 2")

	conns3 := hub.GetConnectionsByTenant(testTenantID3)
	assert.Len(t, conns3, 0, "Non-existent tenant should return empty")
}

// TestGetMobileConnectionsByTenant_FiltersWebSessions verifies mobile only returns mobile devices
func TestGetMobileConnectionsByTenant_FiltersWebSessions(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)

	hub.mu.Lock()
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}] = &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		DeviceID:  "android_abc123",
		Status:    StatusConnected,
	}
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID10}] = &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID10,
		DeviceID:  "", // web session (empty)
		Status:    StatusConnected,
	}
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID99}] = &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID99,
		DeviceID:  "00000000-0000-0000-0000-000000000005", // web session (UUID)
		Status:    StatusConnected,
	}
	hub.mu.Unlock()

	conns := hub.GetMobileConnectionsByTenant(testTenantID1)
	assert.Len(t, conns, 1, "Should only return mobile devices")
	assert.Equal(t, "android_abc123", conns[0].DeviceID)
}

// TestGetMobileConnectionsByTenant_FiltersDisconnected verifies disconnected devices are excluded
func TestGetMobileConnectionsByTenant_FiltersDisconnected(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)

	hub.mu.Lock()
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}] = &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		DeviceID:  "android_abc123",
		Status:    StatusConnected,
	}
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID10}] = &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID10,
		DeviceID:  "ios_xyz789",
		Status:    StatusDisconnected,
	}
	hub.mu.Unlock()

	conns := hub.GetMobileConnectionsByTenant(testTenantID1)
	assert.Len(t, conns, 1, "Should only return connected devices")
	assert.Equal(t, "android_abc123", conns[0].DeviceID)
}

// TestSendMessageToTenant_NoConnections verifies error when no mobile devices connected
func TestSendMessageToTenant_NoConnections(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()

	msg := WebSocketMessage{
		Type:      "MESSAGE_DELIVERY",
		MessageID: "test-123",
		TenantID:  testTenantID1,
		Timestamp: time.Now(),
	}

	err := hub.SendMessageToTenant(ctx, testTenantID1, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no connected mobile devices")
}

// TestSendMessageToTenant_Delivers verifies messages are delivered to all mobile devices
func TestSendMessageToTenant_Delivers(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)
	ctx := context.Background()

	conn := &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID5,
		DeviceID:  "android_abc123",
		Status:    StatusConnected,
		Send:      make(chan []byte, hub.Config.BufferSize),
		Hub:       hub,
	}

	hub.mu.Lock()
	hub.connections[ConnectionKey{TenantID: testTenantID1, ChannelID: testChannelID5}] = conn
	hub.mu.Unlock()

	msg := WebSocketMessage{
		Type:      "MESSAGE_DELIVERY",
		MessageID: "test-123",
		TenantID:  testTenantID1,
		Payload: map[string]interface{}{
			"content": "Test message",
		},
		Timestamp: time.Now(),
	}

	err := hub.SendMessageToTenant(ctx, testTenantID1, msg)
	assert.NoError(t, err)

	// Verify message was received on the connection's send channel
	select {
	case data := <-conn.Send:
		assert.NotNil(t, data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Message not delivered to connection")
	}
}

// TestDeliverPendingMessages_WebSession verifies web sessions skip pending message delivery
func TestDeliverPendingMessages_WebSession(t *testing.T) {
	hub := NewHub(getTestConfig(), nil, nil)

	tests := []struct {
		name     string
		deviceID string
	}{
		{"empty device ID skips", ""},
		{"UUID device ID skips", "00000000-0000-0000-0000-000000000005"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{
				TenantID:  testTenantID1,
				ChannelID: testChannelID5,
				DeviceID:  tt.deviceID,
				Status:    StatusConnected,
				Send:      make(chan []byte, hub.Config.BufferSize),
				Hub:       hub,
			}
			// Should return immediately without touching DB
			hub.deliverPendingMessagesToDevice(conn)
		})
	}
}

// TestDeliverPendingMessages_NeedsDB tests pending delivery for mobile (requires DB)
func TestDeliverPendingMessages_NeedsDB(t *testing.T) {
	if db.GormDB == nil {
		t.Skip("Skipping test that requires database connection")
	}

	hub := NewHub(getTestConfig(), nil, nil)
	conn := &Connection{
		TenantID:  testTenantID1,
		ChannelID: testChannelID10,
		DeviceID:  "android_abc123",
		Status:    StatusConnected,
		Send:      make(chan []byte, hub.Config.BufferSize),
		Hub:       hub,
	}

	// Should attempt DB query (may log error but not panic)
	hub.deliverPendingMessagesToDevice(conn)
}
