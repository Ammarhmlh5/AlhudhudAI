package gateway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/al-hudhud/backend/internal/redis"
	"github.com/google/uuid"
)

// RedisRateLimiter manages rate limiting per tenant using Redis
type RedisRateLimiter struct {
	redis        *redis.Client
	defaultLimit int           // Default: 100 requests per minute
	window       time.Duration // Default: 1 minute
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		redis:        redisClient,
		defaultLimit: 100,
		window:       time.Minute,
	}
}

// Allow checks if request is allowed for tenant
// Returns true if allowed, false if rate limit exceeded
func (rl *RedisRateLimiter) Allow(ctx context.Context, tenantID uuid.UUID) (bool, *RateLimitInfo, error) {
	key := fmt.Sprintf("rate_limit:%s", tenantID.String())
	
	// Increment counter
	count, err := rl.redis.Client.Incr(ctx, key).Result()
	if err != nil {
		return false, nil, fmt.Errorf("increment rate limit counter: %w", err)
	}
	
	// Set TTL on first request
	if count == 1 {
		err = rl.redis.Client.Expire(ctx, key, rl.window).Err()
		if err != nil {
			return false, nil, fmt.Errorf("set rate limit TTL: %w", err)
		}
	}
	
	// Get TTL for reset time
	ttl, err := rl.redis.Client.TTL(ctx, key).Result()
	if err != nil {
		return false, nil, fmt.Errorf("get rate limit TTL: %w", err)
	}
	
	// Calculate reset time
	resetAt := time.Now().Add(ttl)
	
	// Build rate limit info
	info := &RateLimitInfo{
		Limit:     int64(rl.defaultLimit),
		Remaining: int64(rl.defaultLimit) - count,
		Reset:     resetAt.Unix(),
	}
	
	// Check if limit exceeded
	if count > int64(rl.defaultLimit) {
		return false, info, nil
	}
	
	return true, info, nil
}

// GetLimit returns current limit for tenant
func (rl *RedisRateLimiter) GetLimit(ctx context.Context, tenantID uuid.UUID) (int, error) {
	// For now, return default limit
	// In future, this could be customized per tenant from database
	return rl.defaultLimit, nil
}

// Reset resets rate limit counter for tenant (for testing)
func (rl *RedisRateLimiter) Reset(ctx context.Context, tenantID uuid.UUID) error {
	key := redis.RateLimitKey(tenantID)
	return rl.redis.Client.Del(ctx, key).Err()
}

// RateLimitInfo contains rate limit information
type RateLimitInfo struct {
	Limit     int64 // Maximum requests allowed
	Remaining int64 // Remaining requests in current window
	Reset     int64 // Unix timestamp when limit resets
}

// ToHeaders converts rate limit info to HTTP headers
func (info *RateLimitInfo) ToHeaders() map[string]string {
	return map[string]string{
		"X-RateLimit-Limit":     strconv.FormatInt(info.Limit, 10),
		"X-RateLimit-Remaining": strconv.FormatInt(info.Remaining, 10),
		"X-RateLimit-Reset":     strconv.FormatInt(info.Reset, 10),
	}
}
