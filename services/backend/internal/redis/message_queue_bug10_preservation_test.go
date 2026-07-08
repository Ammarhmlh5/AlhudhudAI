package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Bug 10 Preservation Tests - FIFO Within Priority
// **Validates: Requirements 3.1**
//
// **Property 2: Preservation** - FIFO Within Priority
//
// These tests verify that FIFO ordering is preserved within each priority level
// on UNFIXED code. These tests should PASS on unfixed code to demonstrate that
// the current implementation correctly maintains FIFO order within priorities.

// TestPreservation_Bug10_FIFOWithinPriority verifies FIFO ordering within each priority level
func TestPreservation_Bug10_FIFOWithinPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property-based test in short mode")
	}

	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		tenantID := uuid.New()
		_ = mq.Clear(ctx, tenantID)

		// Generate random count for a single priority level
		messageCount := rapid.IntRange(10, 100).Draw(t, "messageCount")
		priority := rapid.IntRange(1, 3).Draw(t, "priority")

		// Enqueue messages with sequential content to track order
		var expectedOrder []string
		for i := 0; i < messageCount; i++ {
			content := fmt.Sprintf("Message-%d", i)
			expectedOrder = append(expectedOrder, content)
			
			item := &MessageQueueItem{
				MessageID:   uuid.New(),
				TenantID:    tenantID,
				PhoneNumber: "+966500000000",
				Content:     content,
				Priority:    priority,
				PartsCount:  1,
				Cost:        0.1,
			}
			err := mq.Enqueue(ctx, item)
			require.NoError(t, err)
			
			// Small delay to ensure ordering
			time.Sleep(1 * time.Millisecond)
		}

		// Dequeue all messages
		var actualOrder []string
		for {
			batch, err := mq.DequeueBatch(ctx, tenantID, 50)
			require.NoError(t, err)
			
			if len(batch) == 0 {
				break
			}
			
			for _, msg := range batch {
				actualOrder = append(actualOrder, msg.Content)
			}
		}

		// Verify FIFO order (first in, first out)
		assert.Equal(t, expectedOrder, actualOrder, 
			"FIFO order should be maintained within priority %d", priority)
	})
}

// TestPreservation_Bug10_FIFOAcrossAllPriorities verifies FIFO within each priority
func TestPreservation_Bug10_FIFOAcrossAllPriorities(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	_ = mq.Clear(ctx, tenantID)

	// Enqueue 20 messages for each priority with sequential IDs
	expectedHigh := []string{}
	expectedNormal := []string{}
	expectedLow := []string{}

	// High priority
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("High-%d", i)
		expectedHigh = append(expectedHigh, content)
		
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000001",
			Content:     content,
			Priority:    1,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// Normal priority
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("Normal-%d", i)
		expectedNormal = append(expectedNormal, content)
		
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000002",
			Content:     content,
			Priority:    2,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// Low priority
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf("Low-%d", i)
		expectedLow = append(expectedLow, content)
		
		item := &MessageQueueItem{
			MessageID:   uuid.New(),
			TenantID:    tenantID,
			PhoneNumber: "+966500000003",
			Content:     content,
			Priority:    3,
			PartsCount:  1,
			Cost:        0.1,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// Dequeue all
	var actualHigh, actualNormal, actualLow []string
	
	for {
		batch, err := mq.DequeueBatch(ctx, tenantID, 50)
		require.NoError(t, err)
		
		if len(batch) == 0 {
			break
		}
		
		for _, msg := range batch {
			switch msg.Priority {
			case 1:
				actualHigh = append(actualHigh, msg.Content)
			case 2:
				actualNormal = append(actualNormal, msg.Content)
			case 3:
				actualLow = append(actualLow, msg.Content)
			}
		}
	}

	// Verify FIFO within each priority
	assert.Equal(t, expectedHigh, actualHigh, "FIFO order violated in high priority")
	assert.Equal(t, expectedNormal, actualNormal, "FIFO order violated in normal priority")
	assert.Equal(t, expectedLow, actualLow, "FIFO order violated in low priority")

	t.Log("✅ FIFO ordering preserved within all priority levels")
}

// TestPreservation_Bug10_MessageIntegrity verifies message data integrity
func TestPreservation_Bug10_MessageIntegrity(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	mq := NewMessageQueue(client)
	ctx := context.Background()
	tenantID := uuid.New()

	_ = mq.Clear(ctx, tenantID)

	// Create messages with unique data
	type TestMessage struct {
		ID          uuid.UUID
		PhoneNumber string
		Content     string
		Priority    int
		Cost        float64
	}

	var sentMessages []TestMessage

	for i := 0; i < 30; i++ {
		msgID := uuid.New()
		phone := fmt.Sprintf("+96650000%04d", i)
		content := fmt.Sprintf("Test message %d", i)
		priority := (i % 3) + 1
		cost := float64(i) * 0.1

		sentMessages = append(sentMessages, TestMessage{
			ID:          msgID,
			PhoneNumber: phone,
			Content:     content,
			Priority:    priority,
			Cost:        cost,
		})

		item := &MessageQueueItem{
			MessageID:   msgID,
			TenantID:    tenantID,
			PhoneNumber: phone,
			Content:     content,
			Priority:    priority,
			PartsCount:  1,
			Cost:        cost,
		}
		err := mq.Enqueue(ctx, item)
		require.NoError(t, err)
	}

	// Dequeue all
	var receivedMessages []TestMessage
	
	for {
		batch, err := mq.DequeueBatch(ctx, tenantID, 50)
		require.NoError(t, err)
		
		if len(batch) == 0 {
			break
		}
		
		for _, msg := range batch {
			receivedMessages = append(receivedMessages, TestMessage{
				ID:          msg.MessageID,
				PhoneNumber: msg.PhoneNumber,
				Content:     msg.Content,
				Priority:    msg.Priority,
				Cost:        msg.Cost,
			})
		}
	}

	// Verify all messages received
	assert.Equal(t, len(sentMessages), len(receivedMessages), 
		"Should receive all sent messages")

	// Verify each message data integrity
	messageMap := make(map[uuid.UUID]TestMessage)
	for _, msg := range sentMessages {
		messageMap[msg.ID] = msg
	}

	for _, received := range receivedMessages {
		sent, exists := messageMap[received.ID]
		require.True(t, exists, "Received unknown message ID: %s", received.ID)
		
		assert.Equal(t, sent.PhoneNumber, received.PhoneNumber, 
			"Phone number mismatch for message %s", received.ID)
		assert.Equal(t, sent.Content, received.Content, 
			"Content mismatch for message %s", received.ID)
		assert.Equal(t, sent.Priority, received.Priority, 
			"Priority mismatch for message %s", received.ID)
		assert.Equal(t, sent.Cost, received.Cost, 
			"Cost mismatch for message %s", received.ID)
	}

	t.Log("✅ Message data integrity preserved")
}
