package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestRedisOptimizer(t *testing.T) (*RedisOptimizer, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	
	logger := zap.NewNop()
	config := DefaultOptimizerConfig()
	
	optimizer := NewRedisOptimizer(client, logger, config)
	
	return optimizer, mr
}

func TestNewRedisOptimizer(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	assert.NotNil(t, optimizer)
	assert.NotNil(t, optimizer.redis)
	assert.NotNil(t, optimizer.logger)
	assert.NotNil(t, optimizer.config)
	assert.NotNil(t, optimizer.stats)
}

func TestDefaultOptimizerConfig(t *testing.T) {
	config := DefaultOptimizerConfig()
	
	assert.True(t, config.EnableAdaptiveTTL)
	assert.True(t, config.EnableTieredCaching)
	assert.True(t, config.EnableCacheWarming)
	assert.True(t, config.EnableSmartEviction)
	assert.True(t, config.EnableMetrics)
	
	assert.Equal(t, 30*time.Second, config.MinTTL)
	assert.Equal(t, 24*time.Hour, config.MaxTTL)
	assert.Equal(t, 15*time.Minute, config.WarmingInterval)
	assert.Equal(t, 100, config.WarmingBatchSize)
	assert.Equal(t, 0.85, config.EvictionThreshold)
}

func TestGetOptimizedTTL(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	tests := []struct {
		name       string
		dataType   string
		tenantType string
		minTTL     time.Duration
		maxTTL     time.Duration
	}{
		{
			name:       "tariff for regular tenant",
			dataType:   "tariff",
			tenantType: "regular",
			minTTL:     1 * time.Hour,
			maxTTL:     3 * time.Hour,
		},
		{
			name:       "tariff for enterprise tenant",
			dataType:   "tariff",
			tenantType: "enterprise",
			minTTL:     2 * time.Hour,
			maxTTL:     5 * time.Hour,
		},
		{
			name:       "balance for regular tenant",
			dataType:   "balance",
			tenantType: "regular",
			minTTL:     1 * time.Minute,
			maxTTL:     3 * time.Minute,
		},
		{
			name:       "package for enterprise tenant",
			dataType:   "package",
			tenantType: "enterprise",
			minTTL:     45 * time.Minute,
			maxTTL:     2 * time.Hour,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := optimizer.GetOptimizedTTL(tt.dataType, tt.tenantType)
			
			assert.GreaterOrEqual(t, ttl, tt.minTTL,
				"TTL should be at least %v, got %v", tt.minTTL, ttl)
			assert.LessOrEqual(t, ttl, tt.maxTTL,
				"TTL should be at most %v, got %v", tt.maxTTL, ttl)
		})
	}
}

func TestGetOptimizedTTL_UnknownDataType(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	ttl := optimizer.GetOptimizedTTL("unknown_type", "regular")
	
	// Should return default TTL
	assert.Equal(t, 5*time.Minute, ttl)
}

func TestGetOptimizedTTL_EnforcesMinMax(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	// Set very restrictive bounds
	optimizer.config.MinTTL = 10 * time.Minute
	optimizer.config.MaxTTL = 15 * time.Minute
	
	// Test with volatile data (normally 2m)
	ttl := optimizer.GetOptimizedTTL("balance", "regular")
	assert.GreaterOrEqual(t, ttl, optimizer.config.MinTTL)
	
	// Test with static data (normally 2h)
	ttl = optimizer.GetOptimizedTTL("tariff", "regular")
	assert.LessOrEqual(t, ttl, optimizer.config.MaxTTL)
}

func TestApplyAdaptiveTTL(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	baseTTL := 10 * time.Minute
	
	tests := []struct {
		name     string
		hits     int64
		misses   int64
		expected string // "increase", "decrease", "same"
	}{
		{
			name:     "high hit rate increases TTL",
			hits:     95,
			misses:   5,
			expected: "increase",
		},
		{
			name:     "low hit rate decreases TTL",
			hits:     40,
			misses:   60,
			expected: "decrease",
		},
		{
			name:     "medium hit rate keeps TTL same",
			hits:     70,
			misses:   30,
			expected: "same",
		},
		{
			name:     "no data keeps TTL same",
			hits:     0,
			misses:   0,
			expected: "same",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset stats
			optimizer.stats.Hits = tt.hits
			optimizer.stats.Misses = tt.misses
			
			ttl := optimizer.applyAdaptiveTTL("test", baseTTL)
			
			switch tt.expected {
			case "increase":
				assert.Greater(t, ttl, baseTTL)
			case "decrease":
				assert.Less(t, ttl, baseTTL)
			case "same":
				assert.Equal(t, ttl, baseTTL)
			}
		})
	}
}

// Mock cache warmer for testing
type mockCacheWarmer struct {
	tariffs       []CacheableTariff
	packages      []CacheablePackage
	subscriptions []CacheableSubscription
	gateways      []CacheableGateway
}

func (m *mockCacheWarmer) LoadActiveTariffs(ctx context.Context) ([]CacheableTariff, error) {
	return m.tariffs, nil
}

func (m *mockCacheWarmer) LoadActivePackages(ctx context.Context) ([]CacheablePackage, error) {
	return m.packages, nil
}

func (m *mockCacheWarmer) LoadActiveSubscriptions(ctx context.Context) ([]CacheableSubscription, error) {
	return m.subscriptions, nil
}

func (m *mockCacheWarmer) LoadActiveGateways(ctx context.Context) ([]CacheableGateway, error) {
	return m.gateways, nil
}

func TestWarmCache(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	ctx := context.Background()
	
	loader := &mockCacheWarmer{
		tariffs: []CacheableTariff{
			{ID: 1},
			{ID: 2},
		},
		packages: []CacheablePackage{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		},
		subscriptions: []CacheableSubscription{
			{ID: 1},
		},
		gateways: []CacheableGateway{
			{ID: 1},
		},
	}
	
	err := optimizer.WarmCache(ctx, loader)
	require.NoError(t, err)
	
	// Verify stats updated
	stats := optimizer.GetStats()
	assert.Equal(t, int64(1), stats.WarmingRuns)
	assert.False(t, stats.LastWarmingTime.IsZero())
}

func TestWarmCache_Disabled(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	optimizer.config.EnableCacheWarming = false
	
	ctx := context.Background()
	loader := &mockCacheWarmer{}
	
	err := optimizer.WarmCache(ctx, loader)
	require.NoError(t, err)
	
	// Stats should not be updated
	stats := optimizer.GetStats()
	assert.Equal(t, int64(0), stats.WarmingRuns)
}

func TestWarmTariffs(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	ctx := context.Background()
	
	loader := &mockCacheWarmer{
		tariffs: []CacheableTariff{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		},
	}
	
	err := optimizer.warmTariffs(ctx, loader)
	require.NoError(t, err)
	
	// Verify keys exist in Redis
	exists, err := optimizer.redis.Exists(ctx, "tariff:1").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
	
	exists, err = optimizer.redis.Exists(ctx, "tariff:2").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestWarmPackages(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	ctx := context.Background()
	
	loader := &mockCacheWarmer{
		packages: []CacheablePackage{
			{ID: 1},
			{ID: 2},
		},
	}
	
	err := optimizer.warmPackages(ctx, loader)
	require.NoError(t, err)
	
	// Verify keys exist
	exists, err := optimizer.redis.Exists(ctx, "package:1").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestRecordHitAndMiss(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	// Record hits
	optimizer.RecordHit()
	optimizer.RecordHit()
	optimizer.RecordHit()
	
	// Record misses
	optimizer.RecordMiss()
	
	stats := optimizer.GetStats()
	assert.Equal(t, int64(3), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestGetHitRate(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	// No data
	hitRate := optimizer.GetHitRate()
	assert.Equal(t, 0.0, hitRate)
	
	// 75% hit rate
	optimizer.stats.Hits = 75
	optimizer.stats.Misses = 25
	
	hitRate = optimizer.GetHitRate()
	assert.Equal(t, 0.75, hitRate)
}

func TestResetStats(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	// Set some stats
	optimizer.stats.Hits = 100
	optimizer.stats.Misses = 50
	optimizer.stats.Sets = 75
	optimizer.stats.Deletes = 25
	optimizer.stats.Evictions = 10
	
	// Reset
	optimizer.ResetStats()
	
	stats := optimizer.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)
	assert.Equal(t, int64(0), stats.Deletes)
	assert.Equal(t, int64(0), stats.Evictions)
}

func TestGetPatternsForTier(t *testing.T) {
	optimizer, mr := setupTestRedisOptimizer(t)
	defer mr.Close()
	
	patterns := optimizer.getPatternsForTier(TTLTierStatic)
	
	// Should include tariff and segmentation_rules
	assert.Contains(t, patterns, "tariff:*")
	assert.Contains(t, patterns, "segmentation_rules:*")
	
	// Should not include volatile data
	assert.NotContains(t, patterns, "balance:*")
	assert.NotContains(t, patterns, "usage:*")
}

func TestOptimizedTTL_Coverage(t *testing.T) {
	// Verify all data types have TTL configuration
	requiredTypes := []string{
		"tariff",
		"package",
		"subscription",
		"balance",
		"usage",
		"stats",
		"gateway",
		"rollover",
		"segmentation_rules",
		"exchange_rates",
		"resource_quotas",
		"active_sessions",
		"rate_limit",
	}
	
	for _, dataType := range requiredTypes {
		t.Run(dataType, func(t *testing.T) {
			config, ok := OptimizedTTL[dataType]
			assert.True(t, ok, "Missing TTL config for %s", dataType)
			assert.Greater(t, config.BaseTTL, time.Duration(0))
			assert.NotEmpty(t, config.Tier)
			assert.NotEmpty(t, config.Description)
		})
	}
}

func TestTTLTiers(t *testing.T) {
	// Verify tier assignments are logical
	staticTypes := []string{"tariff", "segmentation_rules"}
	volatileTypes := []string{"balance", "usage", "rate_limit"}
	
	for _, dataType := range staticTypes {
		config := OptimizedTTL[dataType]
		assert.Equal(t, TTLTierStatic, config.Tier,
			"%s should be static tier", dataType)
		assert.GreaterOrEqual(t, config.BaseTTL, 1*time.Hour,
			"Static data should have TTL >= 1h")
	}
	
	for _, dataType := range volatileTypes {
		config := OptimizedTTL[dataType]
		assert.Equal(t, TTLTierVolatile, config.Tier,
			"%s should be volatile tier", dataType)
		assert.LessOrEqual(t, config.BaseTTL, 5*time.Minute,
			"Volatile data should have TTL <= 5m")
	}
}

func BenchmarkGetOptimizedTTL(b *testing.B) {
	mr := miniredis.RunT(&testing.T{})
	defer mr.Close()
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	
	logger := zap.NewNop()
	optimizer := NewRedisOptimizer(client, logger, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.GetOptimizedTTL("tariff", "regular")
	}
}

func BenchmarkRecordHit(b *testing.B) {
	mr := miniredis.RunT(&testing.T{})
	defer mr.Close()
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	
	logger := zap.NewNop()
	optimizer := NewRedisOptimizer(client, logger, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.RecordHit()
	}
}
