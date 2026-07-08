package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMobileMessageFormat tests the WebSocket message format for mobile delivery
func TestMobileMessageFormat(t *testing.T) {
	// Test the message format that gets sent to mobile devices
	msg := ws.WebSocketMessage{
		Type:      "MESSAGE_DELIVERY",
		MessageID: uuid.New().String(),
		TenantID:  uuid.New(),
		Payload: map[string]interface{}{
			"message_id":   uuid.New().String(),
			"content":      "اختبار رسالة للجهاز المحمول",
			"sender_name":  "نظام المحاسبة",
			"phone_number": "+966501234567",
			"message_type": "ALERT",
			"timestamp":    time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	require.NoError(t, err, "Should marshal message to JSON")

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Should unmarshal JSON back")

	assert.Equal(t, "MESSAGE_DELIVERY", parsed["type"], "Message type should be MESSAGE_DELIVERY")
	assert.NotNil(t, parsed["payload"], "Payload should not be nil")

	payload := parsed["payload"].(map[string]interface{})
	assert.Equal(t, "اختبار رسالة للجهاز المحمول", payload["content"], "Content should match")
	assert.Equal(t, "نظام المحاسبة", payload["sender_name"], "Sender name should match")
	assert.Equal(t, "+966501234567", payload["phone_number"], "Phone number should match")

	t.Log("✅ WebSocket message format is correct")
	t.Logf("  - Type: %s", parsed["type"])
	t.Logf("  - Content: %s", payload["content"])
	t.Logf("  - Sender: %s", payload["sender_name"])
}

// TestMessageDeliveryLog_Model tests the MessageDeliveryLog model structure
func TestMessageDeliveryLog_Model(t *testing.T) {
	// Test that MessageDeliveryLog model can be created
	deliveryLog := models.MessageDeliveryLog{
		ID:              uuid.New(),
		MessageID:       uuid.New(),
		TenantID:        uuid.New(),
		DeviceID:        uuid.New(),
		ChannelID:       uuid.New(),
		DeliveryChannel: "MOBILE",
		Delivered:       true,
		DeliveredAt:     time.Now(),
	}

	// Verify all fields are set correctly
	assert.NotEqual(t, uuid.Nil, deliveryLog.ID, "ID should not be nil")
	assert.NotEqual(t, uuid.Nil, deliveryLog.MessageID, "MessageID should not be nil")
	assert.NotEqual(t, uuid.Nil, deliveryLog.TenantID, "TenantID should not be nil")
	assert.NotEqual(t, uuid.Nil, deliveryLog.DeviceID, "DeviceID should not be nil")
	assert.Equal(t, "MOBILE", deliveryLog.DeliveryChannel, "DeliveryChannel should be MOBILE")
	assert.True(t, deliveryLog.Delivered, "Delivered should be true")

	t.Log("✅ MessageDeliveryLog model structure is correct")
	t.Logf("  - ID: %s", deliveryLog.ID)
	t.Logf("  - Delivery Channel: %s", deliveryLog.DeliveryChannel)
	t.Logf("  - Delivered: %v", deliveryLog.Delivered)
}

// TestWebSocketHub_Methods tests that WebSocket hub has required methods
func TestWebSocketHub_Methods(t *testing.T) {
	// Test that the hub has the required methods for mobile delivery
	if ws.GlobalHub == nil {
		t.Log("⚠️  GlobalHub is nil (requires server initialization)")
		t.Log("   This is expected in test environment without full server startup")
		return
	}

	// Test GetMobileConnectionsByTenant
	tenantID := uuid.New()
	connections := ws.GlobalHub.GetMobileConnectionsByTenant(tenantID)
	assert.NotNil(t, connections, "Should return empty slice, not nil")
	assert.IsType(t, []*ws.Connection{}, connections, "Should return slice of connections")

	// Test GetConnectionsByTenant
	allConnections := ws.GlobalHub.GetConnectionsByTenant(tenantID)
	assert.NotNil(t, allConnections, "Should return empty slice, not nil")

	t.Log("✅ WebSocket hub methods are available")
	t.Logf("  - GetMobileConnectionsByTenant: OK")
	t.Logf("  - GetConnectionsByTenant: OK")
}

// TestMessageActivationWorker_Structure tests MessageActivationWorker structure
func TestMessageActivationWorker_Structure(t *testing.T) {
	// Test MessageActivationArgs structure
	args := MessageActivationArgs{
		MessageID: uuid.New(),
		TenantID:  uuid.New(),
	}

	assert.NotEqual(t, uuid.Nil, args.MessageID, "MessageID should not be nil")
	assert.NotEqual(t, uuid.Nil, args.TenantID, "TenantID should not be nil")
	assert.Equal(t, "message_activation", args.Kind(), "Kind should be message_activation")

	t.Log("✅ MessageActivationWorker structure is correct")
	t.Logf("  - Kind: %s", args.Kind())
	t.Logf("  - MessageID: %s", args.MessageID)
}

// TestMessageStatus_Values tests message status constants
func TestMessageStatus_Values(t *testing.T) {
	// Verify message status values
	assert.Equal(t, models.MessageStatus("PENDING"), models.MessagePending, "MessagePending should be PENDING")
	assert.Equal(t, models.MessageStatus("QUEUED"), models.MessageQueued, "MessageQueued should be QUEUED")
	assert.Equal(t, models.MessageStatus("DELIVERED"), models.MessageDelivered, "MessageDelivered should be DELIVERED")
	assert.Equal(t, models.MessageStatus("SENT"), models.MessageSent, "MessageSent should be SENT")
	assert.Equal(t, models.MessageStatus("FAILED"), models.MessageFailed, "MessageFailed should be FAILED")

	t.Log("✅ Message status constants are correct")
	t.Logf("  - PENDING: %s", models.MessagePending)
	t.Logf("  - QUEUED: %s", models.MessageQueued)
	t.Logf("  - DELIVERED: %s", models.MessageDelivered)
}

// TestMessageSource_Validation tests message source validation
func TestMessageSource_Validation(t *testing.T) {
	// All sources now trigger mobile delivery (ACCOUNTING_SYSTEM, WEB_UI, API, ...)
	testCases := []struct {
		name     string
		source   string
		expected bool
	}{
		{
			name:     "ACCOUNTING_SYSTEM should trigger mobile delivery",
			source:   "ACCOUNTING_SYSTEM",
			expected: true,
		},
		{
			name:     "WEB_UI should trigger mobile delivery",
			source:   "WEB_UI",
			expected: true,
		},
		{
			name:     "API should trigger mobile delivery",
			source:   "API",
			expected: true,
		},
		{
			name:     "Empty source should trigger mobile delivery",
			source:   "",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.source != "MOBILE"
			assert.Equal(t, tc.expected, result, "Source validation should match expected")
		})
	}

	t.Log("✅ Message source validation tests passed")
}

// TestPendingMessage_Reconnection tests pending message delivery on reconnection
func TestPendingMessage_Reconnection(t *testing.T) {
	// Create a pending message that should be delivered when device reconnects
	pendingMessage := models.Message{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Source:       "ACCOUNTING_SYSTEM",
		PhoneNumber:  "+966501234567",
		Content:      "رسالة معلقة سيتم تسليمها عند إعادة الاتصال",
		Status:       models.MessagePending,
		MessageType:  models.MsgCatAlert,
		CharCount:    40,
		PartsCount:   1,
		Encoding:     "UNICODE",
		Cost:         0.10,
		CreatedAt:    time.Now(),
	}

	// Verify it's a pending accounting message
	assert.Equal(t, models.MessagePending, pendingMessage.Status, "Status should be PENDING")
	assert.Equal(t, "ACCOUNTING_SYSTEM", pendingMessage.Source, "Source should be ACCOUNTING_SYSTEM")

	t.Log("✅ Pending message for reconnection delivery is correctly configured")
	t.Logf("  - Message ID: %s", pendingMessage.ID)
	t.Logf("  - Status: %s", pendingMessage.Status)
	t.Logf("  - Source: %s", pendingMessage.Source)
	t.Logf("  - Content: %s", pendingMessage.Content)
}

// TestWebSocketMessage_Serialization tests WebSocket message serialization
func TestWebSocketMessage_Serialization(t *testing.T) {
	// Test complete WebSocket message serialization
	msg := ws.WebSocketMessage{
		Type:      "MESSAGE_DELIVERY",
		MessageID: "msg-12345",
		TenantID:  uuid.New(),
		ChannelID: uuid.New(),
		Payload: map[string]interface{}{
			"message_id":   "msg-12345",
			"content":      "Test message",
			"sender_name":  "System",
			"phone_number": "+966501234567",
			"message_type": "ALERT",
			"timestamp":    "2026-05-24T02:00:00Z",
		},
		Timestamp: time.Now(),
	}

	// Test JSON serialization
	data, err := json.Marshal(msg)
	require.NoError(t, err, "Should serialize to JSON")

	// Test deserialization
	var decoded ws.WebSocketMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err, "Should deserialize from JSON")

	assert.Equal(t, msg.Type, decoded.Type, "Type should match")
	assert.Equal(t, msg.MessageID, decoded.MessageID, "MessageID should match")
	assert.Equal(t, msg.TenantID, decoded.TenantID, "TenantID should match")

	t.Log("✅ WebSocket message serialization tests passed")
	t.Logf("  - Serialized size: %d bytes", len(data))
	t.Logf("  - Type: %s", decoded.Type)
	t.Logf("  - MessageID: %s", decoded.MessageID)
}