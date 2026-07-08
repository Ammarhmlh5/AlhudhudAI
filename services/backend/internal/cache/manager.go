package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	Increment(ctx context.Context, key string) (int64, error)
}

type Manager struct {
	client Client
}

func NewManager(client Client) *Manager {
	return &Manager{client: client}
}

func (m *Manager) Get(ctx context.Context, key string) (string, error) {
	return m.client.Get(ctx, key)
}

func (m *Manager) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return m.client.Set(ctx, key, value, expiration)
}

func (m *Manager) Delete(ctx context.Context, key string) error {
	return m.client.Delete(ctx, key)
}

func (m *Manager) DeletePattern(ctx context.Context, pattern string) error {
	return m.client.DeletePattern(ctx, pattern)
}

func (m *Manager) Exists(ctx context.Context, key string) (bool, error) {
	return m.client.Exists(ctx, key)
}

func (m *Manager) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return m.client.GetTTL(ctx, key)
}

func (m *Manager) Increment(ctx context.Context, key string) (int64, error) {
	return m.client.Increment(ctx, key)
}

func (m *Manager) SetChannelCache(ctx context.Context, tenantID string, channelData string) error {
	key := fmt.Sprintf("channel:%s:data", tenantID)
	return m.client.Set(ctx, key, channelData, RoutingTTL)
}

func (m *Manager) GetChannelCache(ctx context.Context, tenantID string) (string, error) {
	key := fmt.Sprintf("channel:%s:data", tenantID)
	return m.client.Get(ctx, key)
}

func (m *Manager) InvalidateTenantCache(ctx context.Context, tenantID string) error {
	pattern := fmt.Sprintf("tenant:%s:*", tenantID)
	return m.client.DeletePattern(ctx, pattern)
}

// redisClient adapts *redis.Client to the Client interface.
type redisClient struct {
	rdb *redis.Client
}

func NewRedisClient(addr string) Client {
	return &redisClient{
		rdb: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (c *redisClient) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *redisClient) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

func (c *redisClient) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func (c *redisClient) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var allKeys []string
	for {
		var keys []string
		var err error
		keys, cursor, err = c.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		allKeys = append(allKeys, keys...)
		if cursor == 0 {
			break
		}
	}
	if len(allKeys) > 0 {
		return c.rdb.Del(ctx, allKeys...).Err()
	}
	return nil
}

func (c *redisClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (c *redisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

func (c *redisClient) Increment(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}
