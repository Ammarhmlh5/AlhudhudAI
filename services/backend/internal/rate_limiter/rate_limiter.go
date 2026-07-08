package rate_limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RateLimiter struct {
	client RedisClient
	limit  int
	window time.Duration
}

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Delete(ctx context.Context, keys ...string) error
}

func NewRateLimiter(client RedisClient, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (r *RateLimiter) key(tenantID uuid.UUID) string {
	return fmt.Sprintf("rate_limit:%s", tenantID.String())
}

func (r *RateLimiter) AllowRequest(ctx context.Context, tenantID uuid.UUID) (bool, int, error) {
	key := r.key(tenantID)

	count, err := r.client.Get(ctx, key)
	if err != nil {
		count = "0"
	}

	current := 0
	fmt.Sscanf(count, "%d", &current)

	if current >= r.limit {
		return false, 0, nil
	}

	newCount, err := r.client.Incr(ctx, key)
	if err != nil {
		return false, 0, err
	}

	if newCount == 1 {
		if err := r.client.Expire(ctx, key, r.window); err != nil {
			return false, 0, err
		}
	}

	remaining := r.limit - int(newCount)
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, nil
}

func (r *RateLimiter) Reset(ctx context.Context, tenantID uuid.UUID) error {
	return r.client.Delete(ctx, r.key(tenantID))
}

func (r *RateLimiter) GetRemainingRequests(ctx context.Context, tenantID uuid.UUID) (int, error) {
	key := r.key(tenantID)

	count, err := r.client.Get(ctx, key)
	if err != nil {
		return r.limit, nil
	}

	current := 0
	fmt.Sscanf(count, "%d", &current)

	remaining := r.limit - current
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}
