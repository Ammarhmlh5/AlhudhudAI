package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Context key for user ID in rate limiting
const UserIDKey = "rate_limit_user_id"

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	// RequestsPerMinute is the maximum number of requests allowed per minute per user/IP
	RequestsPerMinute int
	// Window is the time window for rate limiting
	Window time.Duration
	// RedisClient is the Redis client for distributed rate limiting
	RedisClient *redis.Client
	// Logger for logging rate limit events
	Logger *zap.Logger
}

// SetRateLimitUserID sets the user ID in the context for rate limiting
// Call this middleware before the rate limiter to set the user ID
func SetRateLimitUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get user ID from different sources
		var userID string

		// Check if user ID is already set in context (from auth middleware)
		if existingUserID, exists := c.Get("user_id"); exists {
			userID = fmt.Sprintf("%v", existingUserID)
		}

		// Check Authorization header for Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && userID == "" {
			// For now, use the IP if no user ID is available
			// In production, you would decode the JWT to get the user ID
			userID = ""
		}

		// Store user ID in context for rate limiter
		c.Set(UserIDKey, userID)
		c.Next()
	}
}

// NewRateLimiter creates a new rate limiting middleware
//
// Configuration:
// - 100 requests per minute per user/IP (default)
// - Uses Redis for distributed rate limiting
// - Returns 429 (Too Many Requests) when limit exceeded
// - Sets rate limit headers (X-RateLimit-*)
//
// Example usage:
//   rateLimiter := middleware.NewRateLimiter(redisClient, 100, time.Minute, logger)
//   router.Use(middleware.SetRateLimitUserID(), rateLimiter)
func NewRateLimiter(redisClient *redis.Client, requestsPerMinute int, window time.Duration, logger *zap.Logger) gin.HandlerFunc {
	// If Redis is not available, use in-memory rate limiter fallback (Requirement 2.5)
	if redisClient == nil {
		if logger != nil {
			logger.Info("Rate limiter: Redis not available, using in-memory fallback",
				zap.Int("requests_per_minute", requestsPerMinute),
			)
		}
		return NewMemoryRateLimiter(requestsPerMinute, logger)
	}

	// Redis is available, use Redis-based rate limiting
	config := &RateLimiterConfig{
		RequestsPerMinute: requestsPerMinute,
		Window:            window,
		RedisClient:       redisClient,
		Logger:            logger,
	}

	return func(c *gin.Context) {
		// Get client IP
		ip := c.ClientIP()

		// Get user ID from context (set by SetRateLimitUserID middleware)
		var userID string
		if existingUserID, exists := c.Get(UserIDKey); exists {
			userID = fmt.Sprintf("%v", existingUserID)
		}

		// Generate rate limit key based on user ID and IP
		// Priority: user ID > IP
		// Format: ratelimit:<path>:user:<userId> or ratelimit:<path>:ip:<ip>
		var key string
		if userID != "" {
			key = fmt.Sprintf("ratelimit:%s:user:%s", c.Request.URL.Path, userID)
		} else {
			key = fmt.Sprintf("ratelimit:%s:ip:%s", c.Request.URL.Path, ip)
		}

		ctx := context.Background()

		// Increment request count
		count, err := config.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			if config.Logger != nil {
				config.Logger.Error("Rate limiter: Failed to increment counter",
					zap.String("ip", ip),
					zap.String("user_id", userID),
					zap.String("path", c.Request.URL.Path),
					zap.Error(err),
				)
			}
			// On error, allow request to proceed (fail open)
			c.Next()
			return
		}

		// Set expiry on first request
		if count == 1 {
			config.RedisClient.Expire(ctx, key, config.Window)
		}

		// Get TTL for remaining time calculation
		ttl, err := config.RedisClient.TTL(ctx, key).Result()
		if err != nil {
			ttl = config.Window
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, config.RequestsPerMinute-int(count))))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

		// Check if limit exceeded
		if count > int64(config.RequestsPerMinute) {
			if config.Logger != nil {
				config.Logger.Warn("Rate limit exceeded",
					zap.String("ip", ip),
					zap.String("user_id", userID),
					zap.String("path", c.Request.URL.Path),
					zap.Int64("count", count),
					zap.Int("limit", config.RequestsPerMinute),
				)
			}

			c.Header("Retry-After", strconv.Itoa(int(ttl.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests. Please try again later.",
				"message": fmt.Sprintf("Rate limit exceeded. Maximum %d requests per minute allowed.", config.RequestsPerMinute),
				"retry_after_seconds": int(ttl.Seconds()),
			})
			return
		}

		c.Next()
	}
}

// Helper function for max (Go 1.21+)
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
