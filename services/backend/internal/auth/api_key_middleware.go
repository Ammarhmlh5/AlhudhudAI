package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// TenantLookup defines the interface for verifying API keys (with cache)
type TenantLookup interface {
	GetTenantByAPIKey(ctx context.Context, key string, isAccounting bool) (*models.Tenant, error)
}

// APIKeyMiddleware validates the accounting or phone API keys and enforces rate limits
func APIKeyMiddleware(lookup TenantLookup, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		accKey := c.GetHeader("X-Accounting-Key")
		if accKey == "" {
			accKey = c.GetHeader("X-API-Key") // Accept X-API-Key as an alias for accounting key
		}
		phoneKey := c.GetHeader("X-Phone-Key")
		webKey := c.GetHeader("X-Web-Key")

		if accKey == "" && phoneKey == "" && webKey == "" {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		var tenant *models.Tenant
		var err error

		if accKey != "" {
			tenant, err = lookup.GetTenantByAPIKey(ctx, accKey, true)
			c.Set("request_source", "ACCOUNTING")
		} else if phoneKey != "" {
			tenant, err = lookup.GetTenantByAPIKey(ctx, phoneKey, false)
			c.Set("request_source", "MOBILE")
		} else if webKey != "" {
			// GetTenantByAPIKey now handles WEB prefix internally via key lookup
			tenant, err = lookup.GetTenantByAPIKey(ctx, webKey, false)
			c.Set("request_source", "WEB_API")
		}

		if err != nil || tenant == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
			c.Abort()
			return
		}

		// Phase 10: Per-Tenant Rate Limiting
		if redisClient != nil {
			if !checkRateLimit(c, redisClient, tenant.ID) {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Please try again later."})
				c.Abort()
				return
			}
		}

		// Set tenant context
		c.Set("tenant_id", tenant.ID.String())
		c.Set("user_type", string(models.TenantType))

		c.Next()
	}
}

func checkRateLimit(c *gin.Context, client *redis.Client, tenantID uuid.UUID) bool {
	// Simple sliding window rate limit: 100 requests per 10 seconds per tenant
	// In production, these limits could be dynamic based on the tenant's plan
	key := fmt.Sprintf("ratelimit:tenant:%s", tenantID)
	limit := 100
	window := 10 * time.Second

	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	// Clear old requests
	client.Client.ZRemRangeByScore(c, key, "0", fmt.Sprintf("%d", windowStart))

	// Count current requests
	count, err := client.Client.ZCard(c, key).Result()
	if err != nil {
		return true // Fallback to allow if Redis is down
	}

	if count >= int64(limit) {
		return false
	}

	// Add current request
	client.Client.ZAdd(c, key, goredis.Z{
		Score:  float64(now),
		Member: now,
	})
	client.Client.Expire(c, key, window)

	return true
}
