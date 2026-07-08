package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// QueryCache provides caching for database queries
// Task 35.1: Query Caching
// Requirements: 27.9
type QueryCache struct {
	redis *redis.Client
}

// NewQueryCache creates a new query cache
func NewQueryCache(redis *redis.Client) *QueryCache {
	return &QueryCache{
		redis: redis,
	}
}

// CacheTTL defines TTL for different data types
// Requirements: 27.9 - تطبيق Caching للاستعلامات المتكررة
var CacheTTL = map[string]time.Duration{
	"tariff":       1 * time.Hour,    // التعرفات نادراً ما تتغير
	"package":      30 * time.Minute, // الباقات قد تتغير
	"subscription": 5 * time.Minute,  // الاشتراكات تتغير بشكل متكرر
	"balance":      1 * time.Minute,  // الرصيد يتغير باستمرار
	"usage":        30 * time.Second, // الاستخدام يتغير بسرعة
	"stats":        10 * time.Minute, // الإحصائيات
	"gateway":      15 * time.Minute, // البوابات المخصصة
	"rollover":     5 * time.Minute,  // الرسائل المرحّلة
}

// GetOrSet retrieves from cache or executes function and caches result
// Requirements: 27.9 - تطبيق Caching للاستعلامات المتكررة
func (qc *QueryCache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache
	val, err := qc.redis.Get(ctx, key).Result()
	if err == nil {
		var result interface{}
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result, nil
		}
	}

	// Execute query
	result, err := fn()
	if err != nil {
		return nil, err
	}

	// Cache result
	data, err := json.Marshal(result)
	if err != nil {
		return result, nil // Return result even if caching fails
	}

	// Set in cache (ignore errors)
	_ = qc.redis.Set(ctx, key, data, ttl).Err()

	return result, nil
}

// Get retrieves a value from cache
func (qc *QueryCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := qc.redis.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache
func (qc *QueryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return qc.redis.Set(ctx, key, data, ttl).Err()
}

// Delete removes a value from cache
func (qc *QueryCache) Delete(ctx context.Context, key string) error {
	return qc.redis.Del(ctx, key).Err()
}

// DeletePattern removes all keys matching a pattern using batch delete
func (qc *QueryCache) DeletePattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var allKeys []string
	
	// Scan all matching keys
	for {
		var keys []string
		var err error
		keys, cursor, err = qc.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		
		allKeys = append(allKeys, keys...)
		
		if cursor == 0 {
			break
		}
	}
	
	// Delete in batches using pipeline for better performance
	if len(allKeys) > 0 {
		const batchSize = 100
		for i := 0; i < len(allKeys); i += batchSize {
			end := i + batchSize
			if end > len(allKeys) {
				end = len(allKeys)
			}
			
			batch := allKeys[i:end]
			pipe := qc.redis.Pipeline()
			for _, key := range batch {
				pipe.Del(ctx, key)
			}
			
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("batch delete failed: %w", err)
			}
		}
	}
	
	return nil
}

// GetTTL returns TTL for a data type
// Requirements: 27.9 - تحسين TTL للمفاتيح المختلفة
func (qc *QueryCache) GetTTL(dataType string) time.Duration {
	if ttl, ok := CacheTTL[dataType]; ok {
		return ttl
	}
	return 5 * time.Minute // Default TTL
}

// GetTTLForTenant returns TTL adjusted for tenant type
// Enterprise tenants get longer TTL
func (qc *QueryCache) GetTTLForTenant(tenantType string, dataType string) time.Duration {
	baseTTL := qc.GetTTL(dataType)

	// Enterprise tenants get 2x TTL
	if tenantType == "enterprise" {
		return baseTTL * 2
	}

	return baseTTL
}

// WarmCache preloads frequently accessed data
// Requirements: 27.9 - تطبيق Cache Warming
func (qc *QueryCache) WarmCache(ctx context.Context, loader CacheLoader) error {
	// Load active tariffs
	tariffs, err := loader.LoadActiveTariffs(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tariffs: %w", err)
	}

	for _, tariff := range tariffs {
		key := fmt.Sprintf("tariff:%d", tariff.ID)
		if err := qc.Set(ctx, key, tariff, CacheTTL["tariff"]); err != nil {
			return fmt.Errorf("failed to cache tariff %d: %w", tariff.ID, err)
		}
	}

	// Load active packages
	packages, err := loader.LoadActivePackages(ctx)
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	for _, pkg := range packages {
		key := fmt.Sprintf("package:%d", pkg.ID)
		if err := qc.Set(ctx, key, pkg, CacheTTL["package"]); err != nil {
			return fmt.Errorf("failed to cache package %d: %w", pkg.ID, err)
		}
	}

	return nil
}

// CacheLoader interface for loading data to warm cache
type CacheLoader interface {
	LoadActiveTariffs(ctx context.Context) ([]CacheableTariff, error)
	LoadActivePackages(ctx context.Context) ([]CacheablePackage, error)
}

// InvalidateCache invalidates cache for a specific entity
func (qc *QueryCache) InvalidateCache(ctx context.Context, entityType string, entityID int) error {
	key := fmt.Sprintf("%s:%d", entityType, entityID)
	return qc.Delete(ctx, key)
}

// InvalidateTenantCache invalidates all cache for a tenant
func (qc *QueryCache) InvalidateTenantCache(ctx context.Context, tenantID string) error {
	pattern := fmt.Sprintf("*:tenant:%s:*", tenantID)
	return qc.DeletePattern(ctx, pattern)
}

// GetStats returns cache statistics
func (qc *QueryCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := qc.redis.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Parse info string to extract stats
	// This is a simplified version - you may want to parse more fields
	return map[string]interface{}{
		"info": info,
	}, nil
}

// Exists checks if a key exists in cache
func (qc *QueryCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := qc.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// GetMulti retrieves multiple values from cache
func (qc *QueryCache) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
	pipe := qc.redis.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	results := make(map[string]interface{})
	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == nil {
			var result interface{}
			if err := json.Unmarshal([]byte(val), &result); err == nil {
				results[keys[i]] = result
			}
		}
	}

	return results, nil
}

// SetMulti stores multiple values in cache
func (qc *QueryCache) SetMulti(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := qc.redis.Pipeline()
	var failed []string

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			failed = append(failed, key)
			continue
		}
		pipe.Set(ctx, key, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}
	
	if len(failed) > 0 {
		return fmt.Errorf("failed to marshal %d items: %v", len(failed), failed)
	}

	return nil
}

// Increment increments a counter in cache
func (qc *QueryCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return qc.redis.IncrBy(ctx, key, delta).Result()
}

// Decrement decrements a counter in cache
func (qc *QueryCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return qc.redis.DecrBy(ctx, key, delta).Result()
}

// SetWithExpiry sets a value with custom expiry
func (qc *QueryCache) SetWithExpiry(ctx context.Context, key string, value interface{}, expiry time.Time) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ttl := time.Until(expiry)
	if ttl <= 0 {
		return fmt.Errorf("expiry time is in the past")
	}

	return qc.redis.Set(ctx, key, data, ttl).Err()
}

// GetTTLRemaining returns remaining TTL for a key
func (qc *QueryCache) GetTTLRemaining(ctx context.Context, key string) (time.Duration, error) {
	return qc.redis.TTL(ctx, key).Result()
}

// Refresh refreshes TTL for a key
func (qc *QueryCache) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	return qc.redis.Expire(ctx, key, ttl).Err()
}
