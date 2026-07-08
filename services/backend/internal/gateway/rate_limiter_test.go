package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRateLimiter(t *testing.T) (*RedisRateLimiter, *miniredis.Miniredis) {
	// Create mini redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create redis client
	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	// Wrap in internal redis client
	internalClient := &redis.Client{
		Client: client,
	}

	// Create rate limiter
	rl := NewRateLimiter(internalClient)

	return rl, mr
}

func TestRateLimiter_Allow_FirstRequest(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	tenantID := uuid.New()

	// First request should be allowed
	allowed, info, err := rl.Allow(ctx, tenantID)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.NotNil(t, info)
	assert.Equal(t, int64(100), info.Limit)
	assert.Equal(t, int64(99), info.Remaining)
	assert.Greater(t, info.Reset, time.Now().Unix())
}

func TestRateLimiter_Allow_MultipleRequests(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	tenantID := uuid.New()

	// Send 50 requests
	for i := 0; i < 50; i++ {
		allowed, info, err := rl.Allow(ctx, tenantID)
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, int64(100-i-1), info.Remaining)
	}
}


func TestRateLimiter_Allow_ExceedLimit(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	tenantID := uuid.New()

	// Send 100 requests (should all be allowed)
	for i := 0; i < 100; i++ {
		allowed, _, err := rl.Allow(ctx, tenantID)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 101st request should be rejected
	allowed, info, err := rl.Allow(ctx, tenantID)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.NotNil(t, info)
	assert.Equal(t, int64(100), info.Limit)
	assert.Equal(t, int64(-1), info.Remaining)
}

func TestRateLimiter_Allow_TenantIsolation(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Exhaust limit for tenant 1
	for i := 0; i < 100; i++ {
		allowed, _, err := rl.Allow(ctx, tenant1)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// Tenant 1 should be rate limited
	allowed, _, err := rl.Allow(ctx, tenant1)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// Tenant 2 should still be allowed
	allowed, info, err := rl.Allow(ctx, tenant2)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(99), info.Remaining)
}

func TestRateLimiter_Reset(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	tenantID := uuid.New()

	// Send some requests
	for i := 0; i < 50; i++ {
		_, _, err := rl.Allow(ctx, tenantID)
		assert.NoError(t, err)
	}

	// Reset counter
	err := rl.Reset(ctx, tenantID)
	assert.NoError(t, err)

	// Next request should have full limit
	allowed, info, err := rl.Allow(ctx, tenantID)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(99), info.Remaining)
}

func TestRateLimitInfo_ToHeaders(t *testing.T) {
	info := &RateLimitInfo{
		Limit:     100,
		Remaining: 50,
		Reset:     1234567890,
	}

	headers := info.ToHeaders()

	assert.Equal(t, "100", headers["X-RateLimit-Limit"])
	assert.Equal(t, "50", headers["X-RateLimit-Remaining"])
	assert.Equal(t, "1234567890", headers["X-RateLimit-Reset"])
}

func TestRateLimiter_GetLimit(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	tenantID := uuid.New()

	limit, err := rl.GetLimit(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, 100, limit)
}

// TestRateLimiter_IsolationIntegration is a comprehensive integration test
// that proves rate limit isolation between tenants.
// 
// **Validates: Requirements 3.4**
//
// This test verifies that:
// 1. Each tenant has completely separate rate limit counters
// 2. Exhausting rate limit for tenant_id=1 does NOT affect tenant_id=2
// 3. Multiple tenants can operate independently at the same time
// 4. Rate limit state is properly isolated in Redis
func TestRateLimiter_IsolationIntegration(t *testing.T) {
	rl, mr := setupTestRateLimiter(t)
	defer mr.Close()

	ctx := context.Background()
	
	// Define test tenants
	tenant1 := uuid.New()
	tenant2 := uuid.New()
	tenant3 := uuid.New()

	t.Run("Exhaust tenant 1 rate limit", func(t *testing.T) {
		// Send 100 requests for tenant 1 (exhaust limit)
		for i := 0; i < 100; i++ {
			allowed, info, err := rl.Allow(ctx, tenant1)
			require.NoError(t, err)
			assert.True(t, allowed, "Tenant 1 request %d should be allowed", i+1)
			assert.Equal(t, int64(100-i-1), info.Remaining, "Tenant 1 remaining count incorrect at request %d", i+1)
		}

		// Verify tenant 1 is now rate limited
		allowed, info, err := rl.Allow(ctx, tenant1)
		require.NoError(t, err)
		assert.False(t, allowed, "Tenant 1 should be rate limited after 100 requests")
		assert.Equal(t, int64(-1), info.Remaining, "Tenant 1 should have -1 remaining")
	})

	t.Run("Verify tenant 2 is NOT affected", func(t *testing.T) {
		// Tenant 2 should have full rate limit available
		allowed, info, err := rl.Allow(ctx, tenant2)
		require.NoError(t, err)
		assert.True(t, allowed, "Tenant 2 should NOT be affected by tenant 1's rate limit")
		assert.Equal(t, int64(99), info.Remaining, "Tenant 2 should have 99 remaining")
		assert.Equal(t, int64(100), info.Limit, "Tenant 2 should have limit of 100")

		// Send 50 more requests for tenant 2
		for i := 0; i < 50; i++ {
			allowed, info, err := rl.Allow(ctx, tenant2)
			require.NoError(t, err)
			assert.True(t, allowed, "Tenant 2 request %d should be allowed", i+2)
			assert.Equal(t, int64(99-i-1), info.Remaining, "Tenant 2 remaining count incorrect")
		}

		// Tenant 2 should still have 49 requests remaining
		allowed, info, err = rl.Allow(ctx, tenant2)
		require.NoError(t, err)
		assert.True(t, allowed, "Tenant 2 should still be allowed")
		assert.Equal(t, int64(48), info.Remaining, "Tenant 2 should have 48 remaining")
	})

	t.Run("Verify tenant 3 is also NOT affected", func(t *testing.T) {
		// Tenant 3 should also have full rate limit
		allowed, info, err := rl.Allow(ctx, tenant3)
		require.NoError(t, err)
		assert.True(t, allowed, "Tenant 3 should NOT be affected by tenant 1's rate limit")
		assert.Equal(t, int64(99), info.Remaining, "Tenant 3 should have 99 remaining")
		assert.Equal(t, int64(100), info.Limit, "Tenant 3 should have limit of 100")
	})

	t.Run("Verify tenant 1 is still rate limited", func(t *testing.T) {
		// Tenant 1 should still be rate limited
		allowed, info, err := rl.Allow(ctx, tenant1)
		require.NoError(t, err)
		assert.False(t, allowed, "Tenant 1 should still be rate limited")
		assert.Equal(t, int64(-2), info.Remaining, "Tenant 1 should have -2 remaining")
	})

	t.Run("Verify Redis key isolation", func(t *testing.T) {
		// Verify that each tenant has a separate Redis key
		client := rl.redis.Client
		
		// Check tenant 1 key exists and has high count
		key1 := "rate_limit:" + tenant1.String()
		count1, err := client.Get(ctx, key1).Int64()
		require.NoError(t, err)
		assert.Greater(t, count1, int64(100), "Tenant 1 should have count > 100")

		// Check tenant 2 key exists and has correct count
		key2 := "rate_limit:" + tenant2.String()
		count2, err := client.Get(ctx, key2).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(52), count2, "Tenant 2 should have count = 52")

		// Check tenant 3 key exists and has count of 1
		key3 := "rate_limit:" + tenant3.String()
		count3, err := client.Get(ctx, key3).Int64()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count3, "Tenant 3 should have count = 1")
	})

	t.Run("Concurrent requests from multiple tenants", func(t *testing.T) {
		// Reset all tenants
		require.NoError(t, rl.Reset(ctx, tenant1))
		require.NoError(t, rl.Reset(ctx, tenant2))
		require.NoError(t, rl.Reset(ctx, tenant3))

		// Send concurrent requests from multiple tenants
		done := make(chan bool, 3)

		// Tenant 1: send 100 requests
		go func() {
			for i := 0; i < 100; i++ {
				allowed, _, err := rl.Allow(ctx, tenant1)
				assert.NoError(t, err)
				assert.True(t, allowed)
			}
			done <- true
		}()

		// Tenant 2: send 50 requests
		go func() {
			for i := 0; i < 50; i++ {
				allowed, _, err := rl.Allow(ctx, tenant2)
				assert.NoError(t, err)
				assert.True(t, allowed)
			}
			done <- true
		}()

		// Tenant 3: send 75 requests
		go func() {
			for i := 0; i < 75; i++ {
				allowed, _, err := rl.Allow(ctx, tenant3)
				assert.NoError(t, err)
				assert.True(t, allowed)
			}
			done <- true
		}()

		// Wait for all goroutines to complete
		<-done
		<-done
		<-done

		// Verify final states
		// Tenant 1 should be at limit
		allowed1, info1, err := rl.Allow(ctx, tenant1)
		require.NoError(t, err)
		assert.False(t, allowed1, "Tenant 1 should be at limit")
		assert.Equal(t, int64(-1), info1.Remaining)

		// Tenant 2 should have 49 remaining
		allowed2, info2, err := rl.Allow(ctx, tenant2)
		require.NoError(t, err)
		assert.True(t, allowed2, "Tenant 2 should have requests remaining")
		assert.Equal(t, int64(49), info2.Remaining)

		// Tenant 3 should have 24 remaining
		allowed3, info3, err := rl.Allow(ctx, tenant3)
		require.NoError(t, err)
		assert.True(t, allowed3, "Tenant 3 should have requests remaining")
		assert.Equal(t, int64(24), info3.Remaining)
	})
}
