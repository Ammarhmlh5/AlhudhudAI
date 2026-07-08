package buffer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/gateway"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestRedis creates a test Redis client
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use separate DB for testing
	})

	// Test connection
	ctx := context.Background()
	err := client.Ping(ctx).Err()
	require.NoError(t, err, "Redis connection failed")

	// Clear test DB
	err = client.FlushDB(ctx).Err()
	require.NoError(t, err, "Failed to flush test DB")

	return client
}

// TestMessageBuffer_Push tests pushing messages to buffer
func TestMessageBuffer_Push(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	msg := gateway.BufferedMessage{
		MessageID:   "msg-123",
		TenantID:    tenantID,
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Priority:    "NORMAL",
		QueuedAt:    time.Now(),
	}

	// Push message
	err := buffer.Push(ctx, tenantID, msg)
	require.NoError(t, err)

	// Verify message in queue
	queueKey := fmt.Sprintf("messages:pending:%s", tenantID.String())
	size, err := client.LLen(ctx, queueKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), size)

	// Verify count
	countKey := fmt.Sprintf("messages:count:%s", tenantID.String())
	count, err := client.Get(ctx, countKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify tenant in active set
	activeKey := "tenants:active"
	isMember, err := client.SIsMember(ctx, activeKey, tenantID.String()).Result()
	require.NoError(t, err)
	assert.True(t, isMember)
}

// TestMessageBuffer_PushMultiple tests pushing multiple messages
func TestMessageBuffer_PushMultiple(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Push 5 messages
	for i := 0; i < 5; i++ {
		msg := gateway.BufferedMessage{
			MessageID:   fmt.Sprintf("msg-%d", i),
			TenantID:    tenantID,
			PhoneNumber: "+966501234567",
			Content:     "Test message",
			Priority:    "NORMAL",
			QueuedAt:    time.Now(),
		}
		err := buffer.Push(ctx, tenantID, msg)
		require.NoError(t, err)
	}

	// Verify count
	size, err := buffer.GetSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)
}

// TestMessageBuffer_PopBatch tests popping messages in batches
func TestMessageBuffer_PopBatch(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Push 10 messages
	for i := 0; i < 10; i++ {
		msg := gateway.BufferedMessage{
			MessageID:   fmt.Sprintf("msg-%d", i),
			TenantID:    tenantID,
			PhoneNumber: "+966501234567",
			Content:     "Test message",
			Priority:    "NORMAL",
			QueuedAt:    time.Now(),
		}
		err := buffer.Push(ctx, tenantID, msg)
		require.NoError(t, err)
	}

	// Pop batch of 5
	messages, err := buffer.PopBatch(ctx, tenantID, 5)
	require.NoError(t, err)
	assert.Len(t, messages, 5)

	// Verify remaining count
	size, err := buffer.GetSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)

	// Pop remaining
	messages, err = buffer.PopBatch(ctx, tenantID, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 5)

	// Verify empty
	size, err = buffer.GetSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

// TestMessageBuffer_TenantIsolation tests tenant isolation
func TestMessageBuffer_TenantIsolation(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenant1ID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tenant2ID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	// Push messages for tenant 1
	msg1 := gateway.BufferedMessage{
		MessageID:   "msg-1",
		TenantID:    tenant1ID,
		PhoneNumber: "+966501234567",
		Content:     "Tenant 1 message",
		Priority:    "NORMAL",
		QueuedAt:    time.Now(),
	}
	err := buffer.Push(ctx, tenant1ID, msg1)
	require.NoError(t, err)

	// Push messages for tenant 2
	msg2 := gateway.BufferedMessage{
		MessageID:   "msg-2",
		TenantID:    tenant2ID,
		PhoneNumber: "+966501234568",
		Content:     "Tenant 2 message",
		Priority:    "NORMAL",
		QueuedAt:    time.Now(),
	}
	err = buffer.Push(ctx, tenant2ID, msg2)
	require.NoError(t, err)

	// Verify tenant 1 has 1 message
	size1, err := buffer.GetSize(ctx, tenant1ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), size1)

	// Verify tenant 2 has 1 message
	size2, err := buffer.GetSize(ctx, tenant2ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), size2)

	// Pop from tenant 1
	messages, err := buffer.PopBatch(ctx, tenant1ID, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, tenant1ID, messages[0].TenantID)

	// Verify tenant 2 still has message
	size2, err = buffer.GetSize(ctx, tenant2ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), size2)
}

// TestMessageBuffer_GetAllActiveTenants tests getting active tenants
func TestMessageBuffer_GetAllActiveTenants(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	// Push messages for multiple tenants
	for i := 1; i <= 3; i++ {
		tenantID := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i))
		msg := gateway.BufferedMessage{
			MessageID:   fmt.Sprintf("msg-%d", i),
			TenantID:    tenantID,
			PhoneNumber: "+966501234567",
			Content:     "Test message",
			Priority:    "NORMAL",
			QueuedAt:    time.Now(),
		}
		err := buffer.Push(ctx, tenantID, msg)
		require.NoError(t, err)
	}

	// Get active tenants
	tenants, err := buffer.GetAllActiveTenants(ctx)
	require.NoError(t, err)
	assert.Len(t, tenants, 3)
	assert.Contains(t, tenants, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	assert.Contains(t, tenants, uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	assert.Contains(t, tenants, uuid.MustParse("00000000-0000-0000-0000-000000000003"))
}

// TestMessageBuffer_Clear tests clearing buffer
func TestMessageBuffer_Clear(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Push messages
	for i := 0; i < 5; i++ {
		msg := gateway.BufferedMessage{
			MessageID:   fmt.Sprintf("msg-%d", i),
			TenantID:    tenantID,
			PhoneNumber: "+966501234567",
			Content:     "Test message",
			Priority:    "NORMAL",
			QueuedAt:    time.Now(),
		}
		err := buffer.Push(ctx, tenantID, msg)
		require.NoError(t, err)
	}

	// Clear buffer
	err := buffer.Clear(ctx, tenantID)
	require.NoError(t, err)

	// Verify empty
	size, err := buffer.GetSize(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Verify not in active set
	tenants, err := buffer.GetAllActiveTenants(ctx)
	require.NoError(t, err)
	assert.NotContains(t, tenants, tenantID)
}

// TestMessageBuffer_FIFO tests FIFO ordering
func TestMessageBuffer_FIFO(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Push messages in order
	messageIDs := []string{"msg-1", "msg-2", "msg-3"}
	for _, msgID := range messageIDs {
		msg := gateway.BufferedMessage{
			MessageID:   msgID,
			TenantID:    tenantID,
			PhoneNumber: "+966501234567",
			Content:     "Test message",
			Priority:    "NORMAL",
			QueuedAt:    time.Now(),
		}
		err := buffer.Push(ctx, tenantID, msg)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Pop all messages
	messages, err := buffer.PopBatch(ctx, tenantID, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 3)

	// Verify FIFO order (first in, first out)
	assert.Equal(t, "msg-1", messages[0].MessageID)
	assert.Equal(t, "msg-2", messages[1].MessageID)
	assert.Equal(t, "msg-3", messages[2].MessageID)
}

// TestMessageBuffer_EmptyPop tests popping from empty buffer
func TestMessageBuffer_EmptyPop(t *testing.T) {
	client := setupTestRedis(t)
	defer client.Close()

	logger := zap.NewNop()
	buffer := NewMessageBuffer(client, logger)
	ctx := context.Background()

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Pop from empty buffer
	messages, err := buffer.PopBatch(ctx, tenantID, 10)
	require.NoError(t, err)
	assert.Empty(t, messages)
}
