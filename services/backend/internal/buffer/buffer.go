package buffer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/gateway"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// MessageBuffer handles buffering messages in Redis
type MessageBuffer struct {
	redis  *redis.Client
	logger *zap.Logger
}

// NewMessageBuffer creates a new message buffer
func NewMessageBuffer(redisClient *redis.Client, logger *zap.Logger) *MessageBuffer {
	return &MessageBuffer{
		redis:  redisClient,
		logger: logger,
	}
}

// Push adds a message to the buffer for a specific tenant
// Uses Redis List (LPUSH) for FIFO queue
// Optimized for fast response time (< 10ms target)
func (b *MessageBuffer) Push(ctx context.Context, tenantID uuid.UUID, msg gateway.BufferedMessage) error {
	// Serialize message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		b.logger.Error("Failed to marshal message",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", msg.MessageID),
		)
		return fmt.Errorf("marshal message: %w", err)
	}

	// Redis keys for tenant isolation
	queueKey := fmt.Sprintf("messages:pending:%s", tenantID.String())
	countKey := fmt.Sprintf("messages:count:%s", tenantID.String())
	activeTenantsKey := "tenants:active"

	// Create context with short timeout for fast response
	// We want to fail fast if Redis is slow
	// (500ms tolerates cold connection-pool dials while still failing fast under load)
	pushCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Use pipeline for atomic operations
	pipe := b.redis.Pipeline()

	// 1. Push message to tenant's queue (critical operation)
	pipe.LPush(pushCtx, queueKey, data)

	// 2. Increment message count (non-critical, for monitoring)
	pipe.Incr(pushCtx, countKey)

	// 3. Add tenant to active set (non-critical, for batch processor)
	pipe.SAdd(pushCtx, activeTenantsKey, tenantID.String())

	// Execute pipeline - this is the main blocking operation
	// Pipeline batches all commands into single network round-trip
	_, err = pipe.Exec(pushCtx)
	if err != nil {
		// Check if it's a timeout error
		if err == context.DeadlineExceeded {
			b.logger.Warn("Redis push timeout - Redis may be slow",
				zap.String("tenant_id", tenantID.String()),
				zap.String("message_id", msg.MessageID),
			)
			return fmt.Errorf("redis timeout: %w", err)
		}
		
		b.logger.Error("Failed to push message to buffer",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", msg.MessageID),
		)
		return fmt.Errorf("push to redis: %w", err)
	}

	// Use async logging to avoid blocking
	// Debug logs are typically disabled in production anyway
	if b.logger.Core().Enabled(zap.DebugLevel) {
		b.logger.Debug("Message pushed to buffer",
			zap.String("tenant_id", tenantID.String()),
			zap.String("message_id", msg.MessageID),
			zap.String("queue_key", queueKey),
		)
	}

	return nil
}

// PopBatch retrieves up to batchSize messages for a specific tenant
// Uses Redis RPOP for FIFO behavior
func (b *MessageBuffer) PopBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) ([]gateway.BufferedMessage, error) {
	queueKey := fmt.Sprintf("messages:pending:%s", tenantID.String())
	countKey := fmt.Sprintf("messages:count:%s", tenantID.String())

	messages := make([]gateway.BufferedMessage, 0, batchSize)

	// Pop messages one by one (Redis doesn't have RPOP with count in older versions)
	for i := 0; i < batchSize; i++ {
		result, err := b.redis.RPop(ctx, queueKey).Result()
		if err == redis.Nil {
			// No more messages
			break
		}
		if err != nil {
			b.logger.Error("Failed to pop message from buffer",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
			)
			return messages, fmt.Errorf("pop from redis: %w", err)
		}

		// Deserialize message
		var msg gateway.BufferedMessage
		if err := json.Unmarshal([]byte(result), &msg); err != nil {
			b.logger.Error("Failed to unmarshal message",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
				zap.String("data", result),
			)
			continue // Skip invalid message
		}

		messages = append(messages, msg)
	}

	// Update count if we popped any messages
	if len(messages) > 0 {
		b.redis.DecrBy(ctx, countKey, int64(len(messages)))
	}

	b.logger.Debug("Popped messages from buffer",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("count", len(messages)),
	)

	return messages, nil
}

// GetSize returns the current buffer size for a tenant
func (b *MessageBuffer) GetSize(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	queueKey := fmt.Sprintf("messages:pending:%s", tenantID.String())

	size, err := b.redis.LLen(ctx, queueKey).Result()
	if err != nil {
		b.logger.Error("Failed to get buffer size",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		return 0, fmt.Errorf("get list length: %w", err)
	}

	return size, nil
}

// GetAllActiveTenants returns list of tenants with pending messages
func (b *MessageBuffer) GetAllActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	activeTenantsKey := "tenants:active"

	// Get all tenant IDs from set
	results, err := b.redis.SMembers(ctx, activeTenantsKey).Result()
	if err != nil {
		b.logger.Error("Failed to get active tenants",
			zap.Error(err),
		)
		return nil, fmt.Errorf("get set members: %w", err)
	}

	tenantIDs := make([]uuid.UUID, 0, len(results))
	for _, result := range results {
		tenantID, err := uuid.Parse(result)
		if err != nil {
			b.logger.Warn("Invalid tenant ID in active set",
				zap.String("value", result),
				zap.Error(err),
			)
			continue
		}
		tenantIDs = append(tenantIDs, tenantID)
	}

	return tenantIDs, nil
}

// RemoveActiveTenant removes a tenant from active set if no messages pending
func (b *MessageBuffer) RemoveActiveTenant(ctx context.Context, tenantID uuid.UUID) error {
	// Check if tenant has any pending messages
	size, err := b.GetSize(ctx, tenantID)
	if err != nil {
		return err
	}

	if size == 0 {
		activeTenantsKey := "tenants:active"
		err := b.redis.SRem(ctx, activeTenantsKey, tenantID.String()).Err()
		if err != nil {
			b.logger.Error("Failed to remove tenant from active set",
				zap.Error(err),
				zap.String("tenant_id", tenantID.String()),
			)
			return fmt.Errorf("remove from set: %w", err)
		}

		b.logger.Debug("Removed tenant from active set",
			zap.String("tenant_id", tenantID.String()),
		)
	}

	return nil
}

// Clear removes all messages for a tenant (for testing/admin purposes)
func (b *MessageBuffer) Clear(ctx context.Context, tenantID uuid.UUID) error {
	queueKey := fmt.Sprintf("messages:pending:%s", tenantID.String())
	countKey := fmt.Sprintf("messages:count:%s", tenantID.String())
	activeTenantsKey := "tenants:active"

	pipe := b.redis.Pipeline()
	pipe.Del(ctx, queueKey)
	pipe.Del(ctx, countKey)
	pipe.SRem(ctx, activeTenantsKey, tenantID.String())

	_, err := pipe.Exec(ctx)
	if err != nil {
		b.logger.Error("Failed to clear buffer",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		return fmt.Errorf("clear buffer: %w", err)
	}

	b.logger.Info("Cleared buffer for tenant",
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}

// SetTTL sets TTL on tenant's queue to prevent memory leaks
func (b *MessageBuffer) SetTTL(ctx context.Context, tenantID uuid.UUID, ttl time.Duration) error {
	queueKey := fmt.Sprintf("messages:pending:%s", tenantID.String())

	err := b.redis.Expire(ctx, queueKey, ttl).Err()
	if err != nil {
		b.logger.Error("Failed to set TTL",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.Duration("ttl", ttl),
		)
		return fmt.Errorf("set ttl: %w", err)
	}

	return nil
}
