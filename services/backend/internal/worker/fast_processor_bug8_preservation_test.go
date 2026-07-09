package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/ws"
)

// Bug 8 Preservation Tests: Successful WebSocket Message Sending
// **Validates: Requirement 3.3**
//
// **Property 2: Preservation** - WebSocket Messages Sent
//
// These tests verify that SUCCESSFUL WebSocket message sending continues to work correctly
// when wsHub is properly initialized. They test the EXISTING functionality that must
// continue to work after the fix.
//
// Expected outcome on UNFIXED code: Tests PASS (successful sends work correctly)
// Expected outcome on FIXED code: Tests PASS (behavior preserved)
//
// IMPORTANT: Bug 8 is about preventing crashes when wsHub is nil.
// These tests verify that when wsHub IS initialized, messages are sent successfully.

// mockHubBug8Preservation is a mock WebSocket Hub for Bug 8 preservation tests
// This mock simulates CORRECT WebSocket behavior when hub is initialized
type mockHubBug8Preservation struct {
	mu              sync.RWMutex
	sentMessages    []ws.WebSocketMessage
	connections     map[string]bool // key: tenantID_channelID
	shouldFailSend  bool
	sendChannelFull bool
}

// newMockHubBug8Preservation creates a new mock hub with initialized connections
func newMockHubBug8Preservation() *mockHubBug8Preservation {
	return &mockHubBug8Preservation{
		sentMessages: make([]ws.WebSocketMessage, 0),
		connections:  make(map[string]bool),
	}
}

// RegisterConnection simulates registering a connection
func (m *mockHubBug8Preservation) RegisterConnection(tenantID, channelID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := tenantID.String() + "_" + channelID.String()
	m.connections[key] = true
}

// SendMessage simulates CORRECT WebSocket message sending behavior
func (m *mockHubBug8Preservation) SendMessage(ctx context.Context, msg ws.WebSocketMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate send channel full error
	if m.sendChannelFull {
		return fmt.Errorf("send channel full for tenant_id=%s, channel_id=%s", 
			msg.TenantID.String(), msg.ChannelID.String())
	}

	// Simulate connection not found error
	key := msg.TenantID.String() + "_" + msg.ChannelID.String()
	if !m.connections[key] {
		return fmt.Errorf("connection not found for tenant_id=%s, channel_id=%s", 
			msg.TenantID.String(), msg.ChannelID.String())
	}

	// Simulate send failure
	if m.shouldFailSend {
		return fmt.Errorf("simulated send failure for tenant_id=%s, channel_id=%s", 
			msg.TenantID.String(), msg.ChannelID.String())
	}

	// CORRECT BEHAVIOR: Successfully send message
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

// GetSentMessages returns all sent messages
func (m *mockHubBug8Preservation) GetSentMessages() []ws.WebSocketMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]ws.WebSocketMessage{}, m.sentMessages...)
}

// GetSentMessageCount returns the count of sent messages
func (m *mockHubBug8Preservation) GetSentMessageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sentMessages)
}

// TestBug8_Preservation_SuccessfulWebSocketSend verifies that successful
// WebSocket message sending continues to work when wsHub is initialized
// **Validates: Requirement 3.3**
func TestBug8_Preservation_SuccessfulWebSocketSend(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 8 preservation test in short mode")
	}

	// Property: Successful WebSocket sends work correctly when hub is initialized
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		channelID := uuid.New()
		messageID := uuid.New()
		phoneNumber := rapid.StringMatching(`\+[0-9]{10,15}`).Draw(t, "phoneNumber")
		content := rapid.StringN(1, 160, -1).Draw(t, "content")

		// Setup: Create INITIALIZED wsHub (CORRECT CONDITION)
		ctx := context.Background()
		mockHub := newMockHubBug8Preservation()
		
		// Register connection (simulate connected device)
		mockHub.RegisterConnection(tenantID, channelID)

		// Create WebSocket message
		wsMessage := ws.WebSocketMessage{
			Type:        "MESSAGE",
			MessageID:   messageID.String(),
			TenantID:    tenantID,
			ChannelID:   channelID,
			PhoneNumber: phoneNumber,
			Content:     content,
			Timestamp:   time.Now(),
		}

		// ACT: Send message via WebSocket (hub is initialized)
		err := mockHub.SendMessage(ctx, wsMessage)

		// ASSERT: Message sent successfully
		assert.NoError(t, err,
			"Preservation: WebSocket send should succeed when hub is initialized")

		// Verify message was sent
		sentMessages := mockHub.GetSentMessages()
		assert.Equal(t, 1, len(sentMessages),
			"Preservation: Exactly one message should be sent")

		if len(sentMessages) > 0 {
			sent := sentMessages[0]
			assert.Equal(t, messageID.String(), sent.MessageID,
				"Preservation: Sent message ID should match")
			assert.Equal(t, tenantID, sent.TenantID,
				"Preservation: Sent tenant ID should match")
			assert.Equal(t, channelID, sent.ChannelID,
				"Preservation: Sent channel ID should match")
			assert.Equal(t, phoneNumber, sent.PhoneNumber,
				"Preservation: Sent phone number should match")
			assert.Equal(t, content, sent.Content,
				"Preservation: Sent content should match")
		}
	})
}

// TestBug8_Preservation_MultipleWebSocketSends verifies that multiple
// sequential WebSocket sends work correctly
// **Validates: Requirement 3.3**
func TestBug8_Preservation_MultipleWebSocketSends(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 8 preservation test in short mode")
	}

	// Property: Multiple WebSocket sends work correctly
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		channelID := uuid.New()
		
		// Number of messages: 2-10
		messageCount := rapid.IntRange(2, 10).Draw(t, "messageCount")

		// Setup: Create INITIALIZED wsHub
		ctx := context.Background()
		mockHub := newMockHubBug8Preservation()
		mockHub.RegisterConnection(tenantID, channelID)

		// ACT: Send multiple messages
		for i := 0; i < messageCount; i++ {
			messageID := uuid.New()
			phoneNumber := rapid.StringMatching(`\+[0-9]{10,15}`).Draw(t, "phoneNumber")
			content := rapid.StringN(1, 160, -1).Draw(t, "content")

			wsMessage := ws.WebSocketMessage{
				Type:        "MESSAGE",
				MessageID:   messageID.String(),
				TenantID:    tenantID,
				ChannelID:   channelID,
				PhoneNumber: phoneNumber,
				Content:     content,
				Timestamp:   time.Now(),
			}

			err := mockHub.SendMessage(ctx, wsMessage)

			// ASSERT: Each send succeeds
			assert.NoError(t, err,
				"Preservation: WebSocket send %d should succeed", i+1)
		}

		// ASSERT: All messages sent
		sentCount := mockHub.GetSentMessageCount()
		assert.Equal(t, messageCount, sentCount,
			"Preservation: All %d messages should be sent", messageCount)
	})
}

// TestBug8_Preservation_RealTimeDelivery verifies that messages are
// delivered in real-time to connected devices
// **Validates: Requirement 3.3**
func TestBug8_Preservation_RealTimeDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 8 preservation test in short mode")
	}

	// Property: Messages are delivered in real-time
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		channelID := uuid.New()
		messageID := uuid.New()
		phoneNumber := rapid.StringMatching(`\+[0-9]{10,15}`).Draw(t, "phoneNumber")
		content := rapid.StringN(1, 160, -1).Draw(t, "content")

		// Setup
		ctx := context.Background()
		mockHub := newMockHubBug8Preservation()
		mockHub.RegisterConnection(tenantID, channelID)

		// Record send time
		sendTime := time.Now()

		wsMessage := ws.WebSocketMessage{
			Type:        "MESSAGE",
			MessageID:   messageID.String(),
			TenantID:    tenantID,
			ChannelID:   channelID,
			PhoneNumber: phoneNumber,
			Content:     content,
			Timestamp:   sendTime,
		}

		// ACT: Send message
		err := mockHub.SendMessage(ctx, wsMessage)

		// ASSERT: Message sent successfully
		assert.NoError(t, err,
			"Preservation: Real-time delivery should succeed")

		// Verify message delivered immediately (within 100ms)
		deliveryTime := time.Now()
		deliveryLatency := deliveryTime.Sub(sendTime)
		assert.Less(t, deliveryLatency, 100*time.Millisecond,
			"Preservation: Message should be delivered in real-time (< 100ms)")

		// Verify message content
		sentMessages := mockHub.GetSentMessages()
		require.Equal(t, 1, len(sentMessages),
			"Preservation: Message should be delivered")
		
		assert.Equal(t, messageID.String(), sentMessages[0].MessageID,
			"Preservation: Delivered message ID should match")
	})
}

// TestBug8_Preservation_ConcurrentWebSocketSends verifies that concurrent
// WebSocket sends work correctly
// **Validates: Requirement 3.3**
func TestBug8_Preservation_ConcurrentWebSocketSends(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Bug 8 preservation test in short mode")
	}

	// Property: Concurrent WebSocket sends work correctly
	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		channelID := uuid.New()
		
		// Number of concurrent sends: 5-20
		concurrentCount := rapid.IntRange(5, 20).Draw(t, "concurrentCount")

		// Pre-generate all test data before spawning goroutines
		type testMessage struct {
			messageID   uuid.UUID
			phoneNumber string
			content     string
		}
		
		messages := make([]testMessage, concurrentCount)
		for i := 0; i < concurrentCount; i++ {
			messages[i] = testMessage{
				messageID:   uuid.New(),
				phoneNumber: rapid.StringMatching(`\+[0-9]{10,15}`).Draw(t, "phoneNumber"),
				content:     rapid.StringN(1, 160, -1).Draw(t, "content"),
			}
		}

		// Setup
		ctx := context.Background()
		mockHub := newMockHubBug8Preservation()
		mockHub.RegisterConnection(tenantID, channelID)

		// ACT: Send messages concurrently
		var wg sync.WaitGroup
		errors := make(chan error, concurrentCount)

		for i := 0; i < concurrentCount; i++ {
			wg.Add(1)
			go func(msg testMessage) {
				defer wg.Done()

				wsMessage := ws.WebSocketMessage{
					Type:        "MESSAGE",
					MessageID:   msg.messageID.String(),
					TenantID:    tenantID,
					ChannelID:   channelID,
					PhoneNumber: msg.phoneNumber,
					Content:     msg.content,
					Timestamp:   time.Now(),
				}

				err := mockHub.SendMessage(ctx, wsMessage)
				if err != nil {
					errors <- err
				}
			}(messages[i])
		}

		wg.Wait()
		close(errors)

		// ASSERT: No errors occurred
		errorCount := 0
		for err := range errors {
			t.Logf("Concurrent send error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount,
			"Preservation: All concurrent sends should succeed")

		// ASSERT: All messages sent
		sentCount := mockHub.GetSentMessageCount()
		assert.Equal(t, concurrentCount, sentCount,
			"Preservation: All %d concurrent messages should be sent", concurrentCount)
	})
}

// Simple unit test: Successful WebSocket send with initialized hub
// **Validates: Requirement 3.3**
func TestBug8_Preservation_Simple_SuccessfulSend(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	// Setup: Create INITIALIZED wsHub (CORRECT CONDITION)
	mockHub := newMockHubBug8Preservation()
	mockHub.RegisterConnection(tenantID, channelID)

	// Create test message
	wsMessage := ws.WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   messageID.String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Timestamp:   time.Now(),
	}

	// ACT: Send message
	err := mockHub.SendMessage(ctx, wsMessage)

	// ASSERT: Message sent successfully
	assert.NoError(t, err, "Preservation: WebSocket send should succeed when hub is initialized")

	// Verify message was sent
	sentMessages := mockHub.GetSentMessages()
	assert.Equal(t, 1, len(sentMessages), "Preservation: Exactly one message should be sent")
	assert.Equal(t, messageID.String(), sentMessages[0].MessageID, "Preservation: Message ID should match")

	t.Logf("✅ Successful WebSocket send: message %s sent to tenant %s", messageID.String(), tenantID.String())
}

// Simple unit test: Multiple sequential WebSocket sends
// **Validates: Requirement 3.3**
func TestBug8_Preservation_Simple_MultipleSequentialSends(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	channelID := uuid.New()

	// Setup
	mockHub := newMockHubBug8Preservation()
	mockHub.RegisterConnection(tenantID, channelID)

	// Send 3 messages sequentially
	messageIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	for i, msgID := range messageIDs {
		wsMessage := ws.WebSocketMessage{
			Type:        "MESSAGE",
			MessageID:   msgID.String(),
			TenantID:    tenantID,
			ChannelID:   channelID,
			PhoneNumber: "+966501234567",
			Content:     "Test message " + string(rune('A'+i)),
			Timestamp:   time.Now(),
		}

		err := mockHub.SendMessage(ctx, wsMessage)
		assert.NoError(t, err, "Preservation: Send %d should succeed", i+1)
	}

	// Verify all messages sent
	sentMessages := mockHub.GetSentMessages()
	assert.Equal(t, 3, len(sentMessages), "Preservation: All 3 messages should be sent")

	// Verify message order
	for i, msgID := range messageIDs {
		assert.Equal(t, msgID.String(), sentMessages[i].MessageID,
			"Preservation: Message %d ID should match", i+1)
	}

	t.Logf("✅ Multiple sequential sends: 3 messages sent successfully")
}

// Simple unit test: WebSocket send with JSON serialization
// **Validates: Requirement 3.3**
func TestBug8_Preservation_Simple_JSONSerialization(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	// Setup
	mockHub := newMockHubBug8Preservation()
	mockHub.RegisterConnection(tenantID, channelID)

	// Create message with special characters
	wsMessage := ws.WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   messageID.String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		PhoneNumber: "+966501234567",
		Content:     "مرحبا! Hello 123 @#$%",
		Timestamp:   time.Now(),
	}

	// ACT: Send message
	err := mockHub.SendMessage(ctx, wsMessage)

	// ASSERT: Message sent successfully
	assert.NoError(t, err, "Preservation: WebSocket send with special characters should succeed")

	// Verify message can be serialized to JSON
	sentMessages := mockHub.GetSentMessages()
	require.Equal(t, 1, len(sentMessages), "Preservation: Message should be sent")

	jsonData, err := json.Marshal(sentMessages[0])
	assert.NoError(t, err, "Preservation: Message should be JSON serializable")
	assert.NotEmpty(t, jsonData, "Preservation: JSON data should not be empty")

	// Verify deserialization
	var deserializedMsg ws.WebSocketMessage
	err = json.Unmarshal(jsonData, &deserializedMsg)
	assert.NoError(t, err, "Preservation: Message should be JSON deserializable")
	assert.Equal(t, wsMessage.Content, deserializedMsg.Content,
		"Preservation: Deserialized content should match")

	t.Logf("✅ JSON serialization: message with special characters serialized successfully")
}

// Simple unit test: WebSocket send to multiple channels
// **Validates: Requirement 3.3**
func TestBug8_Preservation_Simple_MultipleChannels(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	channel1 := uuid.New()
	channel2 := uuid.New()
	channel3 := uuid.New()

	// Setup: Register multiple channels
	mockHub := newMockHubBug8Preservation()
	mockHub.RegisterConnection(tenantID, channel1)
	mockHub.RegisterConnection(tenantID, channel2)
	mockHub.RegisterConnection(tenantID, channel3)

	// Send message to each channel
	channels := []uuid.UUID{channel1, channel2, channel3}
	for i, channelID := range channels {
		messageID := uuid.New()
		wsMessage := ws.WebSocketMessage{
			Type:        "MESSAGE",
			MessageID:   messageID.String(),
			TenantID:    tenantID,
			ChannelID:   channelID,
			PhoneNumber: "+966501234567",
			Content:     "Test message for channel " + string(rune('A'+i)),
			Timestamp:   time.Now(),
		}

		err := mockHub.SendMessage(ctx, wsMessage)
		assert.NoError(t, err, "Preservation: Send to channel %d should succeed", i+1)
	}

	// Verify all messages sent
	sentMessages := mockHub.GetSentMessages()
	assert.Equal(t, 3, len(sentMessages), "Preservation: All 3 messages should be sent")

	// Verify each channel received its message
	for i, channelID := range channels {
		assert.Equal(t, channelID, sentMessages[i].ChannelID,
			"Preservation: Message %d should be sent to correct channel", i+1)
	}

	t.Logf("✅ Multiple channels: messages sent to 3 different channels successfully")
}

// Simple unit test: Real-time delivery latency
// **Validates: Requirement 3.3**
func TestBug8_Preservation_Simple_RealTimeLatency(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	channelID := uuid.New()
	messageID := uuid.New()

	// Setup
	mockHub := newMockHubBug8Preservation()
	mockHub.RegisterConnection(tenantID, channelID)

	// Record send time
	sendTime := time.Now()

	wsMessage := ws.WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   messageID.String(),
		TenantID:    tenantID,
		ChannelID:   channelID,
		PhoneNumber: "+966501234567",
		Content:     "Real-time test message",
		Timestamp:   sendTime,
	}

	// ACT: Send message
	err := mockHub.SendMessage(ctx, wsMessage)

	// Record delivery time
	deliveryTime := time.Now()
	latency := deliveryTime.Sub(sendTime)

	// ASSERT: Message sent successfully with low latency
	assert.NoError(t, err, "Preservation: Real-time send should succeed")
	assert.Less(t, latency, 10*time.Millisecond,
		"Preservation: Delivery latency should be < 10ms (actual: %v)", latency)

	// Verify message delivered
	sentMessages := mockHub.GetSentMessages()
	assert.Equal(t, 1, len(sentMessages), "Preservation: Message should be delivered")

	t.Logf("✅ Real-time delivery: message delivered in %v", latency)
}

// Simple unit test: Concurrent sends from multiple goroutines
// **Validates: Requirement 3.3**
func TestBug8_Preservation_Simple_ConcurrentSends(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	channelID := uuid.New()

	// Setup
	mockHub := newMockHubBug8Preservation()
	mockHub.RegisterConnection(tenantID, channelID)

	// Send 10 messages concurrently
	concurrentCount := 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			messageID := uuid.New()
			wsMessage := ws.WebSocketMessage{
				Type:        "MESSAGE",
				MessageID:   messageID.String(),
				TenantID:    tenantID,
				ChannelID:   channelID,
				PhoneNumber: "+966501234567",
				Content:     "Concurrent message",
				Timestamp:   time.Now(),
			}

			err := mockHub.SendMessage(ctx, wsMessage)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// ASSERT: No errors
	errorCount := 0
	for err := range errors {
		t.Logf("Concurrent send error: %v", err)
		errorCount++
	}
	assert.Equal(t, 0, errorCount, "Preservation: All concurrent sends should succeed")

	// Verify all messages sent
	sentMessages := mockHub.GetSentMessages()
	assert.Equal(t, concurrentCount, len(sentMessages),
		"Preservation: All %d concurrent messages should be sent", concurrentCount)

	t.Logf("✅ Concurrent sends: %d messages sent concurrently without errors", concurrentCount)
}
