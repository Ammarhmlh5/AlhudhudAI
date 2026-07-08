package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// MessageQueueItem represents a message in the Redis queue
type MessageQueueItem struct {
	MessageID   uuid.UUID `json:"message_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	PhoneNumber string    `json:"phone_number"`
	Content     string    `json:"content"`
	PartsCount  int       `json:"parts_count"`
	Cost        float64   `json:"cost"`
	DeviceID    *uuid.UUID `json:"device_id,omitempty"`
	ChannelID   *uuid.UUID `json:"channel_id,omitempty"`
	Priority    int       `json:"priority"` // 1=high, 2=normal, 3=low
	EnqueuedAt  time.Time `json:"enqueued_at"`
}

// MessageQueue provides high-performance Redis-based message queuing
type MessageQueue struct {
	client *Client
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(client *Client) *MessageQueue {
	return &MessageQueue{
		client: client,
	}
}

// Enqueue adds a message to the Redis queue (ultra-fast)
func (mq *MessageQueue) Enqueue(ctx context.Context, item *MessageQueueItem) error {
	if item.EnqueuedAt.IsZero() {
		item.EnqueuedAt = time.Now()
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Use priority-based queues
	queueKey := mq.getQueueKey(item.TenantID, item.Priority)

	// Add tenant to active set
	activeKey := fmt.Sprintf("%s:active:tenants", AppNamespace)
	if err := mq.client.Client.SAdd(ctx, activeKey, item.TenantID.String()).Err(); err != nil {
		// Log error but don't fail enqueue
		// Worst case: tenant not in active set, will be added on next enqueue
		if mq.client != nil && mq.client.Logger != nil {
			mq.client.Logger.Warn("Failed to add tenant to active set",
				zap.String("tenant_id", item.TenantID.String()),
				zap.Error(err),
			)
		}
	}

	// Use retry wrapper for enqueue operation
	return mq.client.WithRetry(ctx, func() error {
		return mq.client.Client.LPush(ctx, queueKey, data).Err()
	})
}

// EnqueueBatch adds multiple messages in a single pipeline operation
func (mq *MessageQueue) EnqueueBatch(ctx context.Context, items []*MessageQueueItem) error {
	if len(items) == 0 {
		return nil
	}

	// Use retry wrapper for batch enqueue operation
	return mq.client.WithRetry(ctx, func() error {
		pipe := mq.client.Client.Pipeline()

		// Track unique tenant IDs to add to active set
		tenantSet := make(map[uuid.UUID]bool)

		for _, item := range items {
			if item.EnqueuedAt.IsZero() {
				item.EnqueuedAt = time.Now()
			}

			data, err := json.Marshal(item)
			if err != nil {
				return fmt.Errorf("marshal message: %w", err)
			}

			queueKey := mq.getQueueKey(item.TenantID, item.Priority)
			pipe.LPush(ctx, queueKey, data)

			// Track tenant for active set
			tenantSet[item.TenantID] = true
		}

		// Add all unique tenants to active set
		activeKey := fmt.Sprintf("%s:active:tenants", AppNamespace)
		for tenantID := range tenantSet {
			pipe.SAdd(ctx, activeKey, tenantID.String())
		}

		_, err := pipe.Exec(ctx)
		return err
	})
}

// Dequeue retrieves and removes a message from the queue
func (mq *MessageQueue) Dequeue(ctx context.Context, tenantID uuid.UUID) (*MessageQueueItem, error) {
	// Try high priority first, then normal, then low
	for priority := 1; priority <= 3; priority++ {
		queueKey := mq.getQueueKey(tenantID, priority)
		
		var result string
		var err error
		
		// Use retry wrapper for dequeue operation
		retryErr := mq.client.WithRetry(ctx, func() error {
			result, err = mq.client.Client.RPop(ctx, queueKey).Result()
			return err
		})
		
		if retryErr == redis.Nil {
			continue // Try next priority
		}
		if retryErr != nil {
			return nil, retryErr
		}

		var item MessageQueueItem
		if err := json.Unmarshal([]byte(result), &item); err != nil {
			return nil, fmt.Errorf("unmarshal message: %w", err)
		}

		return &item, nil
	}

	return nil, redis.Nil // No messages in any queue
}

// DequeueBatch retrieves multiple messages at once (batch processing)
//
// STRICT PRIORITY ORDERING GUARANTEE:
// This function enforces strict priority ordering across all priority levels.
// Messages are dequeued in the following order:
//   1. High priority (priority=1) - FULLY DRAINED before moving to next level
//   2. Normal priority (priority=2) - FULLY DRAINED before moving to next level
//   3. Low priority (priority=3) - Processed last
//
// BATCH SIZE BEHAVIOR:
// The function respects the requested batch size (count parameter):
//   - If high priority queue has >= count messages, returns ONLY high priority messages
//   - If high priority has < count messages, drains high completely, then fills remaining from normal
//   - If high + normal < count messages, drains both, then fills remaining from low
//   - Never exceeds the requested count
//
// FIFO GUARANTEE WITHIN PRIORITY:
// Within each priority level, messages maintain FIFO (First-In-First-Out) ordering.
// Messages are dequeued in the exact order they were enqueued within their priority level.
//
// EXAMPLES:
//
// Example 1: High priority queue has enough messages
//   - Request: count=100
//   - Queues: high=150, normal=100, low=50
//   - Result: Returns 100 messages from high priority ONLY
//   - Remaining: high=50, normal=100, low=50
//
// Example 2: Need to combine priorities
//   - Request: count=100
//   - Queues: high=50, normal=100, low=50
//   - Result: Returns 50 high + 50 normal (strict ordering maintained)
//   - Remaining: high=0, normal=50, low=50
//
// Example 3: All priorities needed
//   - Request: count=100
//   - Queues: high=30, normal=40, low=50
//   - Result: Returns 30 high + 40 normal + 30 low (strict ordering maintained)
//   - Remaining: high=0, normal=0, low=20
//
// Example 4: Batch size larger than total queue
//   - Request: count=200
//   - Queues: high=50, normal=30, low=20
//   - Result: Returns all 100 messages (50 high + 30 normal + 20 low)
//   - Remaining: all queues empty
//
// PARAMETERS:
//   - ctx: Context for cancellation and timeout
//   - tenantID: The tenant whose messages to dequeue
//   - count: Maximum number of messages to retrieve (default: 100 if <= 0)
//
// RETURNS:
//   - []*MessageQueueItem: Slice of dequeued messages in strict priority order
//   - error: Any error encountered during dequeue operation
//
// IMPLEMENTATION DETAILS:
// The strict ordering is enforced through:
//   1. Sequential loop: for priority := 1; priority <= 3
//   2. Conditional continuation: && len(items) < count prevents lower priorities if batch is full
//   3. Remaining calculation: remaining := count - len(items) ensures exact batch size
//   4. Early exit: Loop terminates when len(items) >= count
//
// REDIS OPERATIONS:
// Uses RPopCount for atomic batch retrieval from Redis lists.
// Includes retry logic for transient Redis failures (see WithRetry wrapper).
//
// ERROR HANDLING:
// - redis.Nil: Queue is empty, continues to next priority level
// - Other errors: Returns partial results with error
// - Invalid JSON: Skips corrupted messages, continues processing
func (mq *MessageQueue) DequeueBatch(ctx context.Context, tenantID uuid.UUID, count int) ([]*MessageQueueItem, error) {
	if count <= 0 {
		count = 100
	}

	var items []*MessageQueueItem
	
	// Try high priority first, then normal, then low
	// Loop condition (len(items) < count) ensures strict priority ordering:
	// - If high priority fills the batch, loop exits before processing normal/low
	// - If high priority is exhausted, continues to normal priority
	// - If high + normal fill the batch, loop exits before processing low
	for priority := 1; priority <= 3 && len(items) < count; priority++ {
		queueKey := mq.getQueueKey(tenantID, priority)
		
		// Calculate how many more messages we need to reach the batch size
		remaining := count - len(items)
		
		var results []string
		var err error
		
		// Use retry wrapper for batch dequeue operation
		// Handles transient Redis failures with exponential backoff
		retryErr := mq.client.WithRetry(ctx, func() error {
			results, err = mq.client.Client.RPopCount(ctx, queueKey, remaining).Result()
			return err
		})
		
		// redis.Nil means queue is empty - continue to next priority level
		if retryErr == redis.Nil {
			continue
		}
		// Other errors - return partial results with error
		if retryErr != nil {
			return items, retryErr
		}

		// Unmarshal and append messages from this priority level
		for _, result := range results {
			var item MessageQueueItem
			if err := json.Unmarshal([]byte(result), &item); err != nil {
				// Skip corrupted messages - log warning in production
				continue
			}
			items = append(items, &item)
		}
	}

	return items, nil
}

// GetQueueSize returns the total number of messages in all priority queues
func (mq *MessageQueue) GetQueueSize(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var total int64
	
	for priority := 1; priority <= 3; priority++ {
		queueKey := mq.getQueueKey(tenantID, priority)
		size, err := mq.client.Client.LLen(ctx, queueKey).Result()
		if err != nil {
			return 0, err
		}
		total += size
	}

	return total, nil
}

// Peek returns messages without removing them
func (mq *MessageQueue) Peek(ctx context.Context, tenantID uuid.UUID, count int) ([]*MessageQueueItem, error) {
	if count <= 0 {
		count = 10
	}

	var items []*MessageQueueItem
	
	for priority := 1; priority <= 3 && len(items) < count; priority++ {
		queueKey := mq.getQueueKey(tenantID, priority)
		remaining := count - len(items)
		
		results, err := mq.client.Client.LRange(ctx, queueKey, -int64(remaining), -1).Result()
		if err != nil {
			continue
		}

		for _, result := range results {
			var item MessageQueueItem
			if err := json.Unmarshal([]byte(result), &item); err != nil {
				continue
			}
			items = append(items, &item)
		}
	}

	return items, nil
}

// Clear removes all messages from the queue (for testing/maintenance)
func (mq *MessageQueue) Clear(ctx context.Context, tenantID uuid.UUID) error {
	pipe := mq.client.Client.Pipeline()
	
	for priority := 1; priority <= 3; priority++ {
		queueKey := mq.getQueueKey(tenantID, priority)
		pipe.Del(ctx, queueKey)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// getQueueKey returns the Redis key for a specific tenant and priority
func (mq *MessageQueue) getQueueKey(tenantID uuid.UUID, priority int) string {
	priorityName := "normal"
	switch priority {
	case 1:
		priorityName = "high"
	case 3:
		priorityName = "low"
	}
	return fmt.Sprintf("%s:queue:messages:%s:%s", AppNamespace, priorityName, tenantID.String())
}

// GetQueueStats returns statistics about the queue
func (mq *MessageQueue) GetQueueStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	highSize, _ := mq.client.Client.LLen(ctx, mq.getQueueKey(tenantID, 1)).Result()
	normalSize, _ := mq.client.Client.LLen(ctx, mq.getQueueKey(tenantID, 2)).Result()
	lowSize, _ := mq.client.Client.LLen(ctx, mq.getQueueKey(tenantID, 3)).Result()
	
	stats["high_priority"] = highSize
	stats["normal_priority"] = normalSize
	stats["low_priority"] = lowSize
	stats["total"] = highSize + normalSize + lowSize
	
	return stats, nil
}

// GetActiveTenants retrieves list of active tenant UUIDs from Redis set
func (mq *MessageQueue) GetActiveTenants(ctx context.Context) ([]string, error) {
	activeKey := fmt.Sprintf("%s:active:tenants", AppNamespace)
	return mq.client.Client.SMembers(ctx, activeKey).Result()
}

// RemoveFromActiveSet removes a tenant from the active tenants set
func (mq *MessageQueue) RemoveFromActiveSet(ctx context.Context, tenantID uuid.UUID) error {
	activeKey := fmt.Sprintf("%s:active:tenants", AppNamespace)
	return mq.client.Client.SRem(ctx, activeKey, tenantID.String()).Err()
}
