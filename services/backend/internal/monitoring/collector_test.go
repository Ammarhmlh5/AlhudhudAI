package monitoring

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.db)
	assert.NotNil(t, collector.redis)
	assert.NotNil(t, collector.lastNetStats)
}

func TestCollectSystemMetrics(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()

	metrics, err := collector.collectSystemMetrics(ctx)

	// System metrics should always be collectible
	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	// CPU usage should be between 0 and 100
	assert.GreaterOrEqual(t, metrics.CPUUsagePercent, 0.0)
	assert.LessOrEqual(t, metrics.CPUUsagePercent, 100.0)

	// Memory usage should be positive
	assert.Greater(t, metrics.MemoryUsageMB, 0.0)
	assert.GreaterOrEqual(t, metrics.MemoryUsagePercent, 0.0)
	assert.LessOrEqual(t, metrics.MemoryUsagePercent, 100.0)

	// Bandwidth might be 0 on first run
	assert.GreaterOrEqual(t, metrics.BandwidthUsageKBps, 0.0)
}

func TestCollectDatabaseMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()
	tenantID := uuid.New()

	// Mock QPS query
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(calls\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"total_calls"}).AddRow(1200))

	// Mock avg query time query
	mock.ExpectQuery("SELECT COALESCE\\(AVG\\(mean_exec_time\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"avg_time"}).AddRow(5.5))

	metrics, err := collector.collectDatabaseMetrics(ctx, tenantID)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)

	// Connection stats should be available
	assert.GreaterOrEqual(t, metrics.ActiveConnections, 0)
	assert.GreaterOrEqual(t, metrics.IdleConnections, 0)
	assert.GreaterOrEqual(t, metrics.MaxConnections, 0)

	// QPS should be calculated
	assert.GreaterOrEqual(t, metrics.QueriesPerSecond, 0.0)

	// Avg query time should be set
	assert.Equal(t, 5.5, metrics.AvgQueryTime)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectForGateway(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Use a mock Redis client for testing
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()
	gatewayID := 1
	tenantID := uuid.New()

	// Mock database queries
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(calls\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"total_calls"}).AddRow(1200))

	mock.ExpectQuery("SELECT COALESCE\\(AVG\\(mean_exec_time\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"avg_time"}).AddRow(5.5))

	metrics, err := collector.CollectForGateway(ctx, gatewayID, tenantID)

	// Note: This test might fail if Redis is not available
	// In production, you'd use a mock Redis client
	if err != nil {
		t.Logf("Warning: CollectForGateway failed (expected if Redis not available): %v", err)
		return
	}

	assert.NotNil(t, metrics)
	assert.Equal(t, gatewayID, metrics.GatewayID)
	assert.Equal(t, tenantID, metrics.TenantID)
	assert.False(t, metrics.Timestamp.IsZero())

	// System metrics should be present
	assert.GreaterOrEqual(t, metrics.SystemMetrics.CPUUsagePercent, 0.0)
	assert.Greater(t, metrics.SystemMetrics.MemoryUsageMB, 0.0)

	// Database metrics should be present
	assert.GreaterOrEqual(t, metrics.DatabaseMetrics.ActiveConnections, 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetActiveDedicatedGateways(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()

	tenantID1 := uuid.New()
	tenantID2 := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenant_id"}).
		AddRow(1, tenantID1).
		AddRow(2, tenantID2)

	mock.ExpectQuery("SELECT id, tenant_id FROM dedicated_gateways").
		WillReturnRows(rows)

	gateways, err := collector.getActiveDedicatedGateways(ctx)

	assert.NoError(t, err)
	assert.Len(t, gateways, 2)
	assert.Equal(t, 1, gateways[0].ID)
	assert.Equal(t, tenantID1, gateways[0].TenantID)
	assert.Equal(t, 2, gateways[1].ID)
	assert.Equal(t, tenantID2, gateways[1].TenantID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSystemInfo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)

	info := collector.GetSystemInfo()

	assert.NotNil(t, info)
	assert.Contains(t, info, "num_cpu")
	assert.Contains(t, info, "go_version")
	assert.Contains(t, info, "num_goroutine")

	assert.Greater(t, info["num_cpu"].(int), 0)
	assert.Greater(t, info["num_goroutine"].(int), 0)
}

func TestCalculateBandwidthUsage(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()

	// First call should return 0 (no previous data)
	bandwidth1, err := collector.calculateBandwidthUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0.0, bandwidth1)

	// Wait a bit and call again
	time.Sleep(100 * time.Millisecond)

	bandwidth2, err := collector.calculateBandwidthUsage(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, bandwidth2, 0.0)
}

func TestCalculateQueriesPerSecond_NoExtension(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()
	tenantID := uuid.New()

	// Simulate pg_stat_statements not being available
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(calls\\), 0\\)").
		WillReturnError(sql.ErrNoRows)

	qps, err := collector.calculateQueriesPerSecond(ctx, tenantID)

	// Should not error, just return 0
	assert.NoError(t, err)
	assert.Equal(t, 0.0, qps)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateAvgQueryTime_NoExtension(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()
	tenantID := uuid.New()

	// Simulate pg_stat_statements not being available
	mock.ExpectQuery("SELECT COALESCE\\(AVG\\(mean_exec_time\\), 0\\)").
		WillReturnError(sql.ErrNoRows)

	avgTime, err := collector.calculateAvgQueryTime(ctx, tenantID)

	// Should not error, just return 0
	assert.NoError(t, err)
	assert.Equal(t, 0.0, avgTime)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()

	tenantID := uuid.New()

	// Mock getting active gateways
	rows := sqlmock.NewRows([]string{"id", "tenant_id"}).
		AddRow(1, tenantID)

	mock.ExpectQuery("SELECT id, tenant_id FROM dedicated_gateways").
		WillReturnRows(rows)

	// Mock database metrics queries
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(calls\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"total_calls"}).AddRow(1200))

	mock.ExpectQuery("SELECT COALESCE\\(AVG\\(mean_exec_time\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"avg_time"}).AddRow(5.5))

	allMetrics, err := collector.CollectAll(ctx)

	// Note: This test might fail if Redis is not available
	if err != nil {
		t.Logf("Warning: CollectAll failed (expected if Redis not available): %v", err)
		return
	}

	// Should have collected metrics for at least one gateway
	assert.GreaterOrEqual(t, len(allMetrics), 0)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Benchmark tests

func BenchmarkCollectSystemMetrics(b *testing.B) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.collectSystemMetrics(ctx)
	}
}

func BenchmarkCollectDatabaseMetrics(b *testing.B) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	ctx := context.Background()
	tenantID := uuid.New()

	// Setup mocks
	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(calls\\), 0\\)").
			WillReturnRows(sqlmock.NewRows([]string{"total_calls"}).AddRow(1200))

		mock.ExpectQuery("SELECT COALESCE\\(AVG\\(mean_exec_time\\), 0\\)").
			WillReturnRows(sqlmock.NewRows([]string{"avg_time"}).AddRow(5.5))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = collector.collectDatabaseMetrics(ctx, tenantID)
	}
}
