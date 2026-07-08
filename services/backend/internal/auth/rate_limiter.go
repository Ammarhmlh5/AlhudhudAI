package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// IPRateLimiter returns a middleware that limits requests by IP address
// to prevent brute-force attacks on auth endpoints.
func IPRateLimiter(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil {
			// Skip rate limiting if Redis is not configured
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("auth_ratelimit:%s:%s", c.Request.URL.Path, ip)
		ctx := context.Background()

		// Increment request count
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		// Set expiry on first request
		if count == 1 {
			redisClient.Expire(ctx, key, window)
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}

		c.Next()
	}
}
