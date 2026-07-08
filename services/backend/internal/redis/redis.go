package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Global redis client instance for ID generation
var globalClient *Client

// Client wraps Redis client with logging
type Client struct {
	Client         *redis.Client
	Logger         *zap.Logger
	circuitBreaker *errors.CircuitBreaker // Circuit breaker for Redis operations
}

// NewClient creates a Redis client from just an address (for testing)
func NewClient(addr string) *Client {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	return &Client{Client: client, Logger: zap.NewNop()}
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config, logger *zap.Logger) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
		MinIdleConns: 10,
		MaxRetries:   3,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Redis connection established",
		zap.String("addr", cfg.GetRedisAddr()),
		zap.Int("db", cfg.Redis.DB),
	)

	// Initialize circuit breaker for Redis operations
	// Requirement 19.3: Circuit Breaker للحماية من فشل Redis
	// Max failures: 3, Timeout: 10 seconds (shorter timeout for cache)
	circuitBreaker := errors.NewCircuitBreaker(3, 10*time.Second)

	redisClient := &Client{
		Client:         client,
		Logger:         logger,
		circuitBreaker: circuitBreaker,
	}

	// Set as global client for ID generation
	globalClient = redisClient

	return redisClient, nil
}

// GetGlobalClient returns the global redis client instance
func GetGlobalClient() *Client {
	return globalClient
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if err := c.Client.Close(); err != nil {
		return err
	}
	c.Logger.Info("Redis connection closed")
	return nil
}

// HealthCheck performs a health check on Redis
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// GetStats returns Redis statistics
func (c *Client) GetStats() map[string]interface{} {
	stats := c.Client.PoolStats()
	
	return map[string]interface{}{
		"hits":        stats.Hits,
		"misses":      stats.Misses,
		"timeouts":    stats.Timeouts,
		"total_conns": stats.TotalConns,
		"idle_conns":  stats.IdleConns,
		"stale_conns": stats.StaleConns,
	}
}

// Note: Key generation helpers have been moved to keys.go for better organization
// and unified namespace management. Import them from the same package.

// ExecuteWithCircuitBreaker executes a Redis operation with circuit breaker protection
// Requirement 19.3: استخدام Circuit Breaker لحماية من فشل Redis
func (c *Client) ExecuteWithCircuitBreaker(ctx context.Context, operation func() error) error {
	return c.circuitBreaker.Call(ctx, operation)
}

// QueryWithCircuitBreaker executes a Redis query with circuit breaker protection
// Requirement 19.3: استخدام Circuit Breaker لحماية من فشل Redis
func (c *Client) QueryWithCircuitBreaker(ctx context.Context, query func() (interface{}, error)) (interface{}, error) {
	return c.circuitBreaker.CallWithResult(ctx, query)
}

// Get returns the value for a given key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.Client.Get(ctx, key).Result()
}

// Set sets a key to a value with expiration
func (c *Client) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.Client.Set(ctx, key, value, expiration).Err()
}

// Incr increments the value of a key
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.Client.Incr(ctx, key).Result()
}

// Expire sets a timeout on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.Client.Expire(ctx, key, expiration).Err()
}

// TTL returns the remaining time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.Client.TTL(ctx, key).Result()
}

// Delete deletes one or more keys
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.Client.Del(ctx, keys...).Err()
}

// PublishMessage publishes a message to a specific device's Redis channel
// Requirement Phase 3: Fast Path via Redis Channel
func (c *Client) PublishMessage(ctx context.Context, deviceID uuid.UUID, payload interface{}) error {
	channel := DispatchChannelKey(deviceID)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.ExecuteWithCircuitBreaker(ctx, func() error {
		return c.Client.Publish(ctx, channel, data).Err()
	})
}

// GetCircuitBreakerState returns the current state of the circuit breaker
func (c *Client) GetCircuitBreakerState() errors.CircuitState {
	return c.circuitBreaker.GetState()
}

// IsCircuitBreakerOpen checks if the circuit breaker is open
func (c *Client) IsCircuitBreakerOpen() bool {
	return c.circuitBreaker.IsOpen()
}

// ResetCircuitBreaker resets the circuit breaker to closed state
func (c *Client) ResetCircuitBreaker() {
	c.circuitBreaker.Reset()
	c.Logger.Info("Redis circuit breaker reset")
}

// WithRetry wraps a Redis operation with retry logic and exponential backoff
// Implements Bug 9 fix: Retry logic for transient Redis failures
// Max retries: 3, Backoff: 10ms, 20ms, 40ms (max 1s)
func (c *Client) WithRetry(ctx context.Context, operation func() error) error {
	maxRetries := 3
	backoff := 10 * time.Millisecond
	maxBackoff := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		if attempt < maxRetries {
			// Use select to allow context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// isRetryableError checks if an error is transient and should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"i/o timeout",
		"connection reset",
		"temporary failure",
		"network unreachable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}
