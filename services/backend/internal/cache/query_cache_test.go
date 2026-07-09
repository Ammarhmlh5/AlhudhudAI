package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestQueryCache_GetOrSet(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("cache miss - executes function", func(t *testing.T) {
		executed := false
		fn := func() (interface{}, error) {
			executed = true
			return map[string]string{"test": "value"}, nil
		}

		result, err := qc.GetOrSet(ctx, "test:key", 1*time.Minute, fn)
		require.NoError(t, err)
		assert.True(t, executed)
		assert.NotNil(t, result)
	})

	t.Run("cache hit - does not execute function", func(t *testing.T) {
		// Set value in cache
		err := qc.Set(ctx, "test:key2", map[string]string{"cached": "value"}, 1*time.Minute)
		require.NoError(t, err)

		executed := false
		fn := func() (interface{}, error) {
			executed = true
			return map[string]string{"new": "value"}, nil
		}

		result, err := qc.GetOrSet(ctx, "test:key2", 1*time.Minute, fn)
		require.NoError(t, err)
		assert.False(t, executed)
		assert.NotNil(t, result)
	})
}

func TestQueryCache_GetSet(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("set and get value", func(t *testing.T) {
		testData := map[string]interface{}{
			"id":   123,
			"name": "test",
		}

		err := qc.Set(ctx, "test:data", testData, 1*time.Minute)
		require.NoError(t, err)

		var result map[string]interface{}
		err = qc.Get(ctx, "test:data", &result)
		require.NoError(t, err)
		assert.Equal(t, float64(123), result["id"]) // JSON unmarshals numbers as float64
		assert.Equal(t, "test", result["name"])
	})

	t.Run("get non-existent key", func(t *testing.T) {
		var result map[string]interface{}
		err := qc.Get(ctx, "non:existent", &result)
		assert.Error(t, err)
	})
}

func TestQueryCache_Delete(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("delete existing key", func(t *testing.T) {
		err := qc.Set(ctx, "test:delete", "value", 1*time.Minute)
		require.NoError(t, err)

		err = qc.Delete(ctx, "test:delete")
		require.NoError(t, err)

		var result string
		err = qc.Get(ctx, "test:delete", &result)
		assert.Error(t, err)
	})
}

func TestQueryCache_DeletePattern(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("delete keys matching pattern", func(t *testing.T) {
		// Set multiple keys
		err := qc.Set(ctx, "tenant:123:data1", "value1", 1*time.Minute)
		require.NoError(t, err)
		err = qc.Set(ctx, "tenant:123:data2", "value2", 1*time.Minute)
		require.NoError(t, err)
		err = qc.Set(ctx, "tenant:456:data1", "value3", 1*time.Minute)
		require.NoError(t, err)

		// Delete pattern
		err = qc.DeletePattern(ctx, "tenant:123:*")
		require.NoError(t, err)

		// Verify tenant:123 keys are deleted
		var result string
		err = qc.Get(ctx, "tenant:123:data1", &result)
		assert.Error(t, err)
		err = qc.Get(ctx, "tenant:123:data2", &result)
		assert.Error(t, err)

		// Verify tenant:456 key still exists
		err = qc.Get(ctx, "tenant:456:data1", &result)
		assert.NoError(t, err)
	})
}

func TestQueryCache_GetTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)

	tests := []struct {
		name     string
		dataType string
		expected time.Duration
	}{
		{"tariff", "tariff", 1 * time.Hour},
		{"package", "package", 30 * time.Minute},
		{"subscription", "subscription", 5 * time.Minute},
		{"balance", "balance", 1 * time.Minute},
		{"unknown", "unknown", 5 * time.Minute}, // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := qc.GetTTL(tt.dataType)
			assert.Equal(t, tt.expected, ttl)
		})
	}
}

func TestQueryCache_GetTTLForTenant(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)

	t.Run("enterprise tenant gets 2x TTL", func(t *testing.T) {
		ttl := qc.GetTTLForTenant("enterprise", "tariff")
		expected := CacheTTL["tariff"] * 2
		assert.Equal(t, expected, ttl)
	})

	t.Run("regular tenant gets normal TTL", func(t *testing.T) {
		ttl := qc.GetTTLForTenant("regular", "tariff")
		expected := CacheTTL["tariff"]
		assert.Equal(t, expected, ttl)
	})
}

func TestQueryCache_Exists(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("key exists", func(t *testing.T) {
		err := qc.Set(ctx, "test:exists", "value", 1*time.Minute)
		require.NoError(t, err)

		exists, err := qc.Exists(ctx, "test:exists")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("key does not exist", func(t *testing.T) {
		exists, err := qc.Exists(ctx, "test:notexists")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestQueryCache_GetMulti(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("get multiple keys", func(t *testing.T) {
		// Set multiple keys
		err := qc.Set(ctx, "key1", "value1", 1*time.Minute)
		require.NoError(t, err)
		err = qc.Set(ctx, "key2", "value2", 1*time.Minute)
		require.NoError(t, err)

		results, err := qc.GetMulti(ctx, []string{"key1", "key2", "key3"})
		require.NoError(t, err)
		assert.Len(t, results, 2) // key3 doesn't exist
		assert.Equal(t, "value1", results["key1"])
		assert.Equal(t, "value2", results["key2"])
	})
}

func TestQueryCache_SetMulti(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("set multiple keys", func(t *testing.T) {
		items := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		err := qc.SetMulti(ctx, items, 1*time.Minute)
		require.NoError(t, err)

		// Verify all keys are set
		var result string
		err = qc.Get(ctx, "key1", &result)
		require.NoError(t, err)
		assert.Equal(t, "value1", result)

		err = qc.Get(ctx, "key2", &result)
		require.NoError(t, err)
		assert.Equal(t, "value2", result)

		err = qc.Get(ctx, "key3", &result)
		require.NoError(t, err)
		assert.Equal(t, "value3", result)
	})
}

func TestQueryCache_IncrementDecrement(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("increment counter", func(t *testing.T) {
		val, err := qc.Increment(ctx, "counter:test", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)

		val, err = qc.Increment(ctx, "counter:test", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(6), val)
	})

	t.Run("decrement counter", func(t *testing.T) {
		val, err := qc.Decrement(ctx, "counter:test", 2)
		require.NoError(t, err)
		assert.Equal(t, int64(4), val)
	})
}

func TestQueryCache_InvalidateCache(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("invalidate entity cache", func(t *testing.T) {
		err := qc.Set(ctx, "tariff:123", "value", 1*time.Minute)
		require.NoError(t, err)

		err = qc.InvalidateCache(ctx, "tariff", 123)
		require.NoError(t, err)

		var result string
		err = qc.Get(ctx, "tariff:123", &result)
		assert.Error(t, err)
	})
}

func TestQueryCache_Refresh(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	t.Run("refresh TTL", func(t *testing.T) {
		err := qc.Set(ctx, "test:refresh", "value", 1*time.Second)
		require.NoError(t, err)

		// Refresh with longer TTL
		err = qc.Refresh(ctx, "test:refresh", 1*time.Hour)
		require.NoError(t, err)

		// Get remaining TTL
		ttl, err := qc.GetTTLRemaining(ctx, "test:refresh")
		require.NoError(t, err)
		assert.True(t, ttl > 59*time.Minute) // Should be close to 1 hour
	})
}

// Benchmark tests
func BenchmarkQueryCache_Set(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = qc.Set(ctx, "bench:key", "value", 1*time.Minute)
	}
}

func BenchmarkQueryCache_Get(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	_ = qc.Set(ctx, "bench:key", "value", 1*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		_ = qc.Get(ctx, "bench:key", &result)
	}
}

func BenchmarkQueryCache_GetOrSet(b *testing.B) {
	client, mr := setupTestRedis(&testing.T{})
	defer mr.Close()

	qc := NewQueryCache(client)
	ctx := context.Background()

	fn := func() (interface{}, error) {
		return "value", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = qc.GetOrSet(ctx, "bench:key", 1*time.Minute, fn)
	}
}
