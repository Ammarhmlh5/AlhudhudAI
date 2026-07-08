package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CachedChannel represents a cached channel selection
type CachedChannel struct {
	ChannelID  uuid.UUID  `json:"channel_id"`
	GatewayID  *uuid.UUID `json:"gateway_id,omitempty"`
	DeviceID   *uuid.UUID `json:"device_id,omitempty"`
	Type       string     `json:"type"`
	Priority   int        `json:"priority"`
	IsOnline   bool       `json:"is_online"`
	CachedAt   time.Time  `json:"cached_at"`
}

// ChannelCache provides fast channel selection caching
type ChannelCache struct {
	client *Client
	ttl    time.Duration
}

// NewChannelCache creates a new channel cache
func NewChannelCache(client *Client) *ChannelCache {
	return &ChannelCache{
		client: client,
		ttl:    1 * time.Minute, // Cache for 1 minute - reduced from 5 minutes to minimize stale data (Bug 23)
	}
}

// Get retrieves a cached channel for a tenant
func (cc *ChannelCache) Get(ctx context.Context, tenantID uuid.UUID) (*CachedChannel, error) {
	key := cc.getCacheKey(tenantID)
	
	var data string
	var err error
	
	// Use retry wrapper for cache get operation
	retryErr := cc.client.WithRetry(ctx, func() error {
		data, err = cc.client.Client.Get(ctx, key).Result()
		return err
	})
	
	if retryErr != nil {
		return nil, retryErr
	}

	var channel CachedChannel
	if err := json.Unmarshal([]byte(data), &channel); err != nil {
		return nil, fmt.Errorf("unmarshal channel: %w", err)
	}

	return &channel, nil
}

// Set stores a channel in cache
func (cc *ChannelCache) Set(ctx context.Context, tenantID uuid.UUID, channel *CachedChannel) error {
	if channel.CachedAt.IsZero() {
		channel.CachedAt = time.Now()
	}

	data, err := json.Marshal(channel)
	if err != nil {
		return fmt.Errorf("marshal channel: %w", err)
	}

	key := cc.getCacheKey(tenantID)
	
	// Use retry wrapper for cache set operation
	return cc.client.WithRetry(ctx, func() error {
		return cc.client.Client.Set(ctx, key, data, cc.ttl).Err()
	})
}

// Invalidate removes a cached channel
func (cc *ChannelCache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	key := cc.getCacheKey(tenantID)
	
	// Use retry wrapper for cache invalidation
	return cc.client.WithRetry(ctx, func() error {
		return cc.client.Client.Del(ctx, key).Err()
	})
}

// SetWithTTL stores a channel with custom TTL
func (cc *ChannelCache) SetWithTTL(ctx context.Context, tenantID uuid.UUID, channel *CachedChannel, ttl time.Duration) error {
	if channel.CachedAt.IsZero() {
		channel.CachedAt = time.Now()
	}

	data, err := json.Marshal(channel)
	if err != nil {
		return fmt.Errorf("marshal channel: %w", err)
	}

	key := cc.getCacheKey(tenantID)
	
	// Use retry wrapper for cache set with TTL
	return cc.client.WithRetry(ctx, func() error {
		return cc.client.Client.Set(ctx, key, data, ttl).Err()
	})
}

// getCacheKey returns the Redis key for channel cache
func (cc *ChannelCache) getCacheKey(tenantID uuid.UUID) string {
	return fmt.Sprintf("%s:cache:channel:preferred:%s", AppNamespace, tenantID.String())
}

// GetMultiple retrieves cached channels for multiple tenants
func (cc *ChannelCache) GetMultiple(ctx context.Context, tenantIDs []uuid.UUID) (map[uuid.UUID]*CachedChannel, error) {
	if len(tenantIDs) == 0 {
		return make(map[uuid.UUID]*CachedChannel), nil
	}

	keys := make([]string, len(tenantIDs))
	for i, tenantID := range tenantIDs {
		keys[i] = cc.getCacheKey(tenantID)
	}

	var results []interface{}
	var err error
	
	// Use retry wrapper for batch get operation
	retryErr := cc.client.WithRetry(ctx, func() error {
		results, err = cc.client.Client.MGet(ctx, keys...).Result()
		return err
	})
	
	if retryErr != nil {
		return nil, retryErr
	}

	channels := make(map[uuid.UUID]*CachedChannel)
	for i, result := range results {
		if result == nil {
			continue
		}

		data, ok := result.(string)
		if !ok {
			continue
		}

		var channel CachedChannel
		if err := json.Unmarshal([]byte(data), &channel); err != nil {
			continue
		}

		channels[tenantIDs[i]] = &channel
	}

	return channels, nil
}

// InvalidateChannel removes a specific channel from cache for a tenant
// This is used for active invalidation when a channel goes offline (Bug 23 - Task 23.2.1)
func (cc *ChannelCache) InvalidateChannel(ctx context.Context, tenantID uuid.UUID) error {
	return cc.Invalidate(ctx, tenantID)
}

// VerifyChannelOnline checks if a cached channel is actually online in the database
// This is used for health check verification (Bug 23 - Task 23.2.2)
// Returns true if channel is online, false otherwise
func (cc *ChannelCache) VerifyChannelOnline(ctx context.Context, channelID uuid.UUID) (bool, error) {
	// Note: This requires database access which should be injected
	// For now, we return an error indicating this needs to be called from a higher level
	// that has database access (e.g., from the routing engine or fast processor)
	return false, fmt.Errorf("VerifyChannelOnline should be called from a component with database access")
}

// InvalidateChannelOnOffline immediately removes cache entry when a channel goes offline
// This is used for active invalidation to prevent failed message sends (Bug 13 - Task 15.1)
// Target: < 100ms invalidation time
func (cc *ChannelCache) InvalidateChannelOnOffline(ctx context.Context, tenantID uuid.UUID) error {
	key := cc.getCacheKey(tenantID)
	
	// Use retry wrapper for immediate cache deletion
	return cc.client.WithRetry(ctx, func() error {
		err := cc.client.Client.Del(ctx, key).Err()
		if err != nil {
			return fmt.Errorf("failed to invalidate channel cache for tenant %s: %w", tenantID.String(), err)
		}
		return nil
	})
}

