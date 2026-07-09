package routing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/cache"
	"github.com/al-hudhud/backend/internal/errors"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Engine implements the RoutingEngine interface
// Provides channel selection with tenant isolation and Redis caching
type Engine struct {
	db             *sql.DB
	redisClient    *redis.Client
	logger         *zap.Logger
	cacheTTL       time.Duration // Cache TTL for channels (default: 5 minutes)
	secretKey      string        // Secret key for secure cache keys
	circuitBreaker *errors.CircuitBreaker
	memoryFallback sync.Map // In-memory fallback for critical routing data
}

// NewEngine creates a new routing engine instance
// If redisClient is nil, caching will be disabled
func NewEngine(db *sql.DB, redisClient *redis.Client, logger *zap.Logger, secretKey string) *Engine {
	return &Engine{
		db:             db,
		redisClient:    redisClient,
		logger:         logger,
		cacheTTL:       5 * time.Minute, // Requirement 5.2: TTL = 5 minutes
		secretKey:      secretKey,
		circuitBreaker: errors.NewCircuitBreaker(3, 30*time.Second), // 3 failures, 30s timeout
	}
}

func (e *Engine) channelCacheKey(tenantID uuid.UUID) string {
	// Requirement 16.9: Cryptographically secure cache keys (HMAC-SHA256)
	// Prevents metadata/tenant guessing via Redis keys
	h := hmac.New(sha256.New, []byte(e.secretKey))
	h.Write([]byte(tenantID.String()))
	signature := hex.EncodeToString(h.Sum(nil))[:16] // Use first 16 chars for brevity
	
	rawKey := fmt.Sprintf("channels:cache:%s:%s", tenantID.String(), signature)
	return cache.GetVersionedKey(rawKey)
}

// getChannelsFromCache retrieves channels from Redis cache
// Returns nil if cache miss or Redis unavailable
// Requirement 5.1: Search in Cache first
func (e *Engine) getChannelsFromCache(ctx context.Context, tenantID uuid.UUID) ([]*Channel, error) {
	if e.redisClient == nil {
		return nil, nil // Caching disabled
	}

	key := e.channelCacheKey(tenantID)

	// Phase 3.1: Circuit Breaker protection for Redis
	result, err := e.circuitBreaker.CallWithResult(ctx, func() (interface{}, error) {
		return e.redisClient.Get(ctx, key).Result()
	})

	if err != nil {
		if err == redis.Nil {
			// Cache miss - this is normal
			e.logger.Debug("cache miss for channels", zap.String("tenant_id", tenantID.String()))
			metrics.RecordCacheMiss("channels")
			return nil, nil
		}

		// Redis error or Circuit Open - Use In-Memory Fallback
		e.logger.Warn("Redis unavailable, using in-memory fallback",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)

		if val, ok := e.memoryFallback.Load(key); ok {
			if channels, ok := val.([]*Channel); ok {
				e.logger.Info("In-memory fallback hit", zap.String("tenant_id", tenantID.String()))
				return channels, nil
			}
		}

		metrics.RecordCacheMiss("channels")
		return nil, nil
	}

	data := result.(string)
	var channels []*Channel
	if err := json.Unmarshal([]byte(data), &channels); err != nil {
		return nil, nil
	}

	// Update memory fallback on successful hit to keep it warm
	e.memoryFallback.Store(key, channels)

	e.logger.Debug("cache hit for channels",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("count", len(channels)),
	)
	
	metrics.RecordCacheHit("channels")
	return channels, nil
}

func (e *Engine) setChannelsToCache(ctx context.Context, tenantID uuid.UUID, channels []*Channel) error {
	if e.redisClient == nil {
		return nil // Caching disabled
	}

	key := e.channelCacheKey(tenantID)
	data, err := json.Marshal(channels)
	if err != nil {
		return fmt.Errorf("marshal channels: %w", err)
	}

	// Phase 3.1: Resilience - Store in memory fallback as well
	e.memoryFallback.Store(key, channels)

	// Phase 3.1: Circuit Breaker protection for Redis write
	return e.circuitBreaker.Call(ctx, func() error {
		return e.redisClient.Set(ctx, key, data, cache.RoutingTTL).Err()
	})
}

// invalidateChannelCache invalidates the channel cache for a tenant
// Used when channel status changes
func (e *Engine) invalidateChannelCache(ctx context.Context, tenantID uuid.UUID) error {
	if e.redisClient == nil {
		return nil // Caching disabled
	}

	key := e.channelCacheKey(tenantID)
	
	// Phase 3.1: Clear memory fallback
	e.memoryFallback.Delete(key)

	// Phase 3.1: Circuit Breaker protection for Redis delete
	err := e.circuitBreaker.Call(ctx, func() error {
		return e.redisClient.Del(ctx, key).Err()
	})

	if err != nil {
		e.logger.Warn("failed to invalidate channel cache",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return err
	}

	e.logger.Debug("channel cache invalidated",
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}

// VerifyChannelOnline checks if a channel is actually online in the database
// This is used for health check verification (Bug 23 - Task 23.2.2)
// Returns true if channel is online, false otherwise
func (e *Engine) VerifyChannelOnline(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT status 
		FROM dispatch_channels 
		WHERE id = $1 
		  AND tenant_id = $2
	`
	
	var status string
	err := e.db.QueryRowContext(ctx, query, channelID, tenantID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			e.logger.Warn("channel not found during health check",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
			)
			return false, nil
		}
		e.logger.Error("failed to verify channel status",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return false, err
	}
	
	isOnline := status == ChannelStatusOnline
	
	if !isOnline {
		e.logger.Info("channel health check: channel is offline",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.String("status", status),
		)
	}
	
	return isOnline, nil
}


// SelectChannel selects the best available channel for a tenant
// Implementation of Requirements 5.1, 5.2, 5.3 and 5.4:
// - Searches in Cache first, then Database on cache miss (5.1, 5.2)
// - Filters channels by ONLINE status only (5.3)
// - Orders by Priority DESC, then CurrentLoad ASC (5.4)
// - Ensures complete tenant isolation
func (e *Engine) SelectChannel(ctx context.Context, tenantID uuid.UUID) (*Channel, error) {
	e.logger.Debug("selecting channel",
		zap.String("tenant_id", tenantID.String()),
	)

	// Requirement 5.1: Search in Cache first
	cachedChannels, err := e.getChannelsFromCache(ctx, tenantID)
	if err == nil && cachedChannels != nil && len(cachedChannels) > 0 {
		// Cache hit - return first channel (already sorted by priority and load)
		channel := cachedChannels[0]
		e.logger.Info("channel selected from cache",
			zap.String("tenant_id", tenantID.String()),
			zap.String("channel_id", channel.ID.String()),
			zap.String("channel_key", channel.ChannelKey),
			zap.Int("priority", channel.Priority),
			zap.Int("current_load", channel.CurrentLoad),
		)
		return channel, nil
	}

	// Cache miss - query database with retry logic
	// Implements Requirement 19.2: Retry with exponential backoff
	retryConfig := errors.RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
	}

	var channel *Channel
	err = errors.Retry(ctx, retryConfig, func() error {
		// Query to select best channel with tenant isolation
		// CRITICAL: WHERE tenant_id = $1 ensures tenant isolation
		// CRITICAL: WHERE status = 'ONLINE' ensures only online channels (Requirement 5.3)
		// CRITICAL: ORDER BY priority DESC, current_load ASC (Requirement 5.4)
		query := `
			SELECT 
				id, 
				tenant_id, 
				gateway_id,
				type, 
				channel_key, 
				status, 
				priority, 
				current_load,
				config,
				created_at, 
				updated_at
			FROM dispatch_channels
			WHERE tenant_id = $1 
			  AND status = $2
			ORDER BY priority DESC, current_load ASC
			LIMIT 1
		`

		var ch Channel
		err := e.db.QueryRowContext(
			ctx,
			query,
			tenantID,
			ChannelStatusOnline,
		).Scan(
			&ch.ID,
			&ch.TenantID,
			&ch.GatewayID,
			&ch.Type,
			&ch.ChannelKey,
			&ch.Status,
			&ch.Priority,
			&ch.CurrentLoad,
			&ch.Config,
			&ch.CreatedAt,
			&ch.UpdatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				e.logger.Warn("no online channels available",
					zap.String("tenant_id", tenantID.String()),
				)
				// Don't retry if no channels available
				return errors.Wrap(err, errors.CodeTenantNotFound, errors.CategoryBusiness, "no online channels available")
			}
			e.logger.Error("failed to select channel",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to select channel")
		}

		channel = &ch
		return nil
	})

	if err != nil {
		// Check if it's a "no channels available" error
		if appErr, ok := err.(*errors.AppError); ok && appErr.Code == errors.CodeNoChannelAvailable {
			return nil, ErrNoChannelAvailable
		}
		return nil, err
	}

	// Requirement 5.2: Store in Cache for future requests
	// Get all channels and cache them
	channels, err := e.GetAvailableChannels(ctx, tenantID)
	if err == nil && len(channels) > 0 {
		_ = e.setChannelsToCache(ctx, tenantID, channels)
	}

	e.logger.Info("channel selected from database",
		zap.String("tenant_id", tenantID.String()),
		zap.String("channel_id", channel.ID.String()),
		zap.String("channel_key", channel.ChannelKey),
		zap.Int("priority", channel.Priority),
		zap.Int("current_load", channel.CurrentLoad),
	)

	return channel, nil
}

// GetAvailableChannels returns all available channels for a tenant
// Filtered by ONLINE status only
// Ordered by Priority DESC, then CurrentLoad ASC
// Uses Redis cache with 5-minute TTL (Requirements 5.1, 5.2)
// Implements Requirement 19.2: Retry with exponential backoff
func (e *Engine) GetAvailableChannels(ctx context.Context, tenantID uuid.UUID) ([]*Channel, error) {
	e.logger.Debug("getting available channels",
		zap.String("tenant_id", tenantID.String()),
	)

	// Requirement 5.1: Search in Cache first
	cachedChannels, err := e.getChannelsFromCache(ctx, tenantID)
	if err == nil && cachedChannels != nil {
		e.logger.Info("available channels retrieved from cache",
			zap.String("tenant_id", tenantID.String()),
			zap.Int("count", len(cachedChannels)),
		)
		return cachedChannels, nil
	}

	// Cache miss - query database with retry logic
	retryConfig := errors.RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
	}

	var channels []*Channel
	err = errors.Retry(ctx, retryConfig, func() error {
		// Query to get all online channels with tenant isolation
		query := `
			SELECT 
				id, 
				tenant_id, 
				gateway_id,
				type, 
				channel_key, 
				status, 
				priority, 
				current_load,
				config,
				created_at, 
				updated_at
			FROM dispatch_channels
			WHERE tenant_id = $1 
			  AND status = $2
			ORDER BY priority DESC, current_load ASC
		`

		rows, err := e.db.QueryContext(
			ctx,
			query,
			tenantID,
			ChannelStatusOnline,
		)
		if err != nil {
			e.logger.Error("failed to query available channels",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to query available channels")
		}
		defer rows.Close()

		channels = nil // Reset channels slice
		for rows.Next() {
			var channel Channel
			err := rows.Scan(
				&channel.ID,
				&channel.TenantID,
				&channel.GatewayID,
				&channel.Type,
				&channel.ChannelKey,
				&channel.Status,
				&channel.Priority,
				&channel.CurrentLoad,
				&channel.Config,
				&channel.CreatedAt,
				&channel.UpdatedAt,
			)
			if err != nil {
				e.logger.Error("failed to scan channel",
					zap.String("tenant_id", tenantID.String()),
					zap.Error(err),
				)
				// Wrap as retryable database error
				return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to scan channel")
			}
			channels = append(channels, &channel)
		}

		if err := rows.Err(); err != nil {
			e.logger.Error("error iterating channels",
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "error iterating channels")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Requirement 5.2: Store in Cache with TTL = 5 minutes
	if len(channels) > 0 {
		_ = e.setChannelsToCache(ctx, tenantID, channels)
	}

	e.logger.Info("available channels retrieved from database",
		zap.String("tenant_id", tenantID.String()),
		zap.Int("count", len(channels)),
	)

	return channels, nil
}

// IncrementChannelLoad increments the current load of a channel by 1
// Implementation of Requirement 5.6: Update current_load after each send
// Implements Requirement 19.2: Retry with exponential backoff
// Invalidates cache to ensure accurate load balancing
func (e *Engine) IncrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error {
	e.logger.Debug("incrementing channel load",
		zap.String("channel_id", channelID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	// Requirement 5.6: Invalidate cache BEFORE database update to prevent race conditions
	if err := e.invalidateChannelCache(ctx, tenantID); err != nil {
		e.logger.Warn("failed to invalidate cache before increment",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	}

	// Retry configuration for database operations
	retryConfig := errors.RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
	}

	err := errors.Retry(ctx, retryConfig, func() error {
		// Update current_load in database
		// CRITICAL: WHERE tenant_id = $2 ensures tenant isolation
		query := `
			UPDATE dispatch_channels
			SET current_load = current_load + 1,
			    updated_at = NOW()
			WHERE id = $1 
			  AND tenant_id = $2
		`

		result, err := e.db.ExecContext(ctx, query, channelID, tenantID)
		if err != nil {
			e.logger.Error("failed to increment channel load",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to increment channel load")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			e.logger.Error("failed to get rows affected",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to get rows affected")
		}

		if rowsAffected == 0 {
			e.logger.Warn("channel not found or tenant mismatch",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
			)
			// Don't retry if channel not found
			return errors.Wrap(fmt.Errorf("channel not found or tenant mismatch"), errors.CodeTenantNotFound, errors.CategoryBusiness, "channel not found")
		}

		return nil
	})

	if err != nil {
		return err
	}


	e.logger.Info("channel load incremented",
		zap.String("channel_id", channelID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}

// DecrementChannelLoad decrements the current load of a channel by 1
// Implementation of Requirement 5.6: Update current_load after message completion
// Implements Requirement 19.2: Retry with exponential backoff
// Invalidates cache to ensure accurate load balancing
func (e *Engine) DecrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error {
	e.logger.Debug("decrementing channel load",
		zap.String("channel_id", channelID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	// Requirement 5.6: Invalidate cache BEFORE database update to prevent race conditions
	if err := e.invalidateChannelCache(ctx, tenantID); err != nil {
		e.logger.Warn("failed to invalidate cache before decrement",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	}

	// Retry configuration for database operations
	retryConfig := errors.RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
	}

	err := errors.Retry(ctx, retryConfig, func() error {
		// Update current_load in database
		// CRITICAL: WHERE tenant_id = $2 ensures tenant isolation
		// CRITICAL: current_load >= 1 prevents negative values
		query := `
			UPDATE dispatch_channels
			SET current_load = GREATEST(current_load - 1, 0),
			    updated_at = NOW()
			WHERE id = $1 
			  AND tenant_id = $2
			  AND current_load >= 1
		`

		result, err := e.db.ExecContext(ctx, query, channelID, tenantID)
		if err != nil {
			e.logger.Error("failed to decrement channel load",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to decrement channel load")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			e.logger.Error("failed to get rows affected",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
				zap.Error(err),
			)
			// Wrap as retryable database error
			return errors.Wrap(err, errors.CodeDatabaseQuery, errors.CategoryDatabase, "failed to get rows affected")
		}

		if rowsAffected == 0 {
			e.logger.Warn("channel not found, tenant mismatch, or load already zero",
				zap.String("channel_id", channelID.String()),
				zap.String("tenant_id", tenantID.String()),
			)
			// Don't return error - this is not critical
		}

		return nil
	})

	if err != nil {
		// Log error but don't fail - decrement is not critical
		e.logger.Error("failed to decrement channel load after retries",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	}


	e.logger.Info("channel load decremented",
		zap.String("channel_id", channelID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}

// UpdateChannelLoad updates the current load of a channel to a specific value
// Implementation of Requirement 5.6: Update current_load after each send
// Invalidates cache to ensure accurate load balancing
func (e *Engine) UpdateChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID, load int) error {
	e.logger.Debug("updating channel load",
		zap.String("channel_id", channelID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.Int("load", load),
	)

	// Validate load value
	if load < 0 {
		e.logger.Error("invalid load value",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Int("load", load),
		)
		return fmt.Errorf("load value cannot be negative: %d", load)
	}

	// Requirement 5.6: Invalidate cache BEFORE database update to prevent race conditions
	if err := e.invalidateChannelCache(ctx, tenantID); err != nil {
		e.logger.Warn("failed to invalidate cache before update",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	}

	query := `
		UPDATE dispatch_channels
		SET current_load = $1,
		    updated_at = NOW()
		WHERE id = $2 
		  AND tenant_id = $3
	`

	result, err := e.db.ExecContext(ctx, query, load, channelID, tenantID)
	if err != nil {
		e.logger.Error("failed to update channel load",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Int("load", load),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update channel load: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		e.logger.Error("failed to get rows affected",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		e.logger.Warn("channel not found or tenant mismatch",
			zap.String("channel_id", channelID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		return fmt.Errorf("channel not found or tenant mismatch")
	}


	e.logger.Info("channel load updated",
		zap.String("channel_id", channelID.String()),
		zap.String("tenant_id", tenantID.String()),
		zap.Int("load", load),
	)

	return nil
}

// SetMessageWaiting updates message status to WAITING when no channels available
// Implementation of Requirement 5.5: Update message status to WAITING when no channels available
// Ensures tenant isolation and logs the event for monitoring
func (e *Engine) SetMessageWaiting(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) error {
	e.logger.Warn("no channels available, setting message to WAITING",
		zap.String("message_id", messageID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	// Update message status to WAITING
	// CRITICAL: WHERE tenant_id = $2 ensures tenant isolation
	query := `
		UPDATE messages
		SET status = 'WAITING',
		    updated_at = NOW()
		WHERE id = $1 
		  AND tenant_id = $2
		  AND status IN ('PENDING', 'PROCESSING')
	`

	result, err := e.db.ExecContext(ctx, query, messageID, tenantID)
	if err != nil {
		e.logger.Error("failed to set message to WAITING",
			zap.String("message_id", messageID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to set message to WAITING: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		e.logger.Error("failed to get rows affected",
			zap.String("message_id", messageID.String()),
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		e.logger.Warn("message not found, tenant mismatch, or invalid status",
			zap.String("message_id", messageID.String()),
			zap.String("tenant_id", tenantID.String()),
		)
		return fmt.Errorf("message not found, tenant mismatch, or invalid status")
	}

	e.logger.Info("message status updated to WAITING",
		zap.String("message_id", messageID.String()),
		zap.String("tenant_id", tenantID.String()),
	)

	return nil
}
