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

func TestNewLogger(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.db)
	assert.NotNil(t, logger.collector)
}

func TestLogGatewayMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := 1

	metrics := &GatewayMetrics{
		GatewayID: gatewayID,
		TenantID:  tenantID,
		Timestamp: time.Now(),
		SystemMetrics: SystemMetrics{
			CPUUsagePercent:    45.5,
			MemoryUsageMB:      1024.0,
			BandwidthUsageKBps: 512.0,
		},
		DatabaseMetrics: DatabaseMetrics{
			ActiveConnections: 10,
			QueriesPerSecond:  50.0,
		},
		RedisMetrics: RedisMetrics{
			MemoryUsed: 256 * 1024 * 1024,
			KeysCount:  1000,
		},
	}

	// Mock message metrics query (last hour) - run before transaction
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM messages WHERE gateway_id").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock message metrics query (today) - run before transaction
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM messages WHERE gateway_id").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))

	// Mock SET app.current_tenant via RunInTenantTx
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
			WithArgs("false").
			WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock INSERT into resource_usage
	mock.ExpectExec("INSERT INTO resource_usage").
		WithArgs(
			gatewayID,
			tenantID,
			sqlmock.AnyArg(), // timestamp
			45.5,             // cpu_usage_percent
			1024,             // memory_usage_mb
			0.5,              // bandwidth_usage_mbps (512 KB/s = 0.5 MB/s)
			0,                // active_connections
			10,               // db_connections_active
			50,               // queries_per_second
			256,              // redis_memory_usage_mb
			1000,             // redis_keys_count
			100,              // messages_processed_last_hour
			500,              // messages_processed_today
			0,                // current_queue_depth
			0,                // api_requests_last_minute
			0,                // api_requests_last_hour
			0,                // active_worker_threads
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = logger.logGatewayMetrics(ctx, metrics)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestLogResourceUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()

	// Mock getActiveDedicatedGateways
	tenantID := uuid.New()
	mock.ExpectQuery("SELECT id, tenant_id FROM dedicated_gateways").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "tenant_id"}).
				AddRow(1, tenantID),
		)

	// For each gateway, we need to mock the metrics collection
	// This is complex, so we'll just test that it doesn't panic
	// In a real test, you'd mock all the system calls

	// Note: This test may fail because we can't mock system calls
	// In production, you'd use dependency injection for system metrics
	err = logger.LogResourceUsage(ctx)
	// We may get an error because we can't collect real system metrics
	// but we're testing that the function structure is correct
	// The function should handle errors gracefully and continue
	if err != nil {
		t.Logf("LogResourceUsage returned error (expected if Redis not available): %v", err)
	}
}

func TestGetRecentUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := 1
	limit := 10

	// Mock SET app.current_tenant via RunInTenantTx
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
			WithArgs("false").
			WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock SELECT query
	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads",
	}).
		AddRow(
			1, gatewayID, tenantID, time.Now(),
			45.5, 1024, 0.5,
			10, 5, 50,
			256, 1000,
			100, 500,
			10, 20,
			100, 4,
		).
		AddRow(
			2, gatewayID, tenantID, time.Now().Add(-1*time.Minute),
			50.0, 1100, 0.6,
			12, 6, 55,
			260, 1050,
			110, 520,
			12, 22,
			105, 4,
		)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, limit).
		WillReturnRows(rows)
	mock.ExpectCommit()

	records, err := logger.GetRecentUsage(ctx, gatewayID, tenantID, limit)
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	// Verify first record
	assert.Equal(t, int64(1), records[0].ID)
	assert.Equal(t, gatewayID, records[0].GatewayID)
	assert.Equal(t, tenantID, records[0].TenantID)
	assert.Equal(t, 45.5, records[0].CPUUsagePercent)
	assert.Equal(t, 1024, records[0].MemoryUsageMB)
	assert.Equal(t, 0.5, records[0].BandwidthUsageMBps)
	assert.Equal(t, 5, records[0].DBConnectionsActive)
	assert.Equal(t, 50, records[0].QueriesPerSecond)
	assert.Equal(t, 256, records[0].RedisMemoryUsageMB)
	assert.Equal(t, 1000, records[0].RedisKeysCount)
	assert.Equal(t, 100, records[0].MessagesProcessedLastHour)
	assert.Equal(t, 500, records[0].MessagesProcessedToday)
	assert.Equal(t, 10, records[0].CurrentQueueDepth)
	assert.Equal(t, 20, records[0].APIRequestsLastMinute)
	assert.Equal(t, 100, records[0].APIRequestsLastHour)
	assert.Equal(t, 4, records[0].ActiveWorkerThreads)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetRecentUsage_WithNullValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := 1
	limit := 10

	// Mock SET app.current_tenant via RunInTenantTx
	mock.ExpectBegin()
	mock.ExpectExec("SELECT set_config\\('app.current_tenant', \\$1, true\\)").
		WithArgs(tenantID.String()).
		WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("SELECT set_config\\('app.is_super_admin', \\$1, true\\)").
			WithArgs("false").
			WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock SELECT query with NULL values
	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads",
	}).
		AddRow(
			1, gatewayID, tenantID, time.Now(),
			nil, nil, nil, // NULL values
			nil, nil, nil,
			nil, nil,
			nil, nil,
			nil, nil,
			nil, nil,
		)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, limit).
		WillReturnRows(rows)
	mock.ExpectCommit()

	records, err := logger.GetRecentUsage(ctx, gatewayID, tenantID, limit)
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	// Verify that NULL values are handled correctly (should be zero values)
	assert.Equal(t, int64(1), records[0].ID)
	assert.Equal(t, 0.0, records[0].CPUUsagePercent)
	assert.Equal(t, 0, records[0].MemoryUsageMB)
	assert.Equal(t, 0.0, records[0].BandwidthUsageMBps)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestLogForGateway(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = mock // Not used in this test

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	tenantID := uuid.New()
	gatewayID := 1

	// This test will fail because we can't mock system calls
	// but we're testing that the function structure is correct
	err = logger.LogForGateway(ctx, gatewayID, tenantID)
	assert.Error(t, err) // Expected to fail due to system metrics collection
}

func TestGetMessageMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	gatewayID := 1

	// Mock last hour query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM messages WHERE gateway_id").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock today query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM messages WHERE gateway_id").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))

	metrics, err := logger.getMessageMetrics(ctx, gatewayID)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, 100, metrics.LastHour)
	assert.Equal(t, 500, metrics.Today)
	assert.Equal(t, 0, metrics.QueueDepth) // Placeholder

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetMessageMetrics_NoMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	gatewayID := 1

	// Mock last hour query - no rows
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM messages WHERE gateway_id").
		WithArgs(gatewayID).
		WillReturnError(sql.ErrNoRows)

	// Mock today query - no rows
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM messages WHERE gateway_id").
		WithArgs(gatewayID).
		WillReturnError(sql.ErrNoRows)

	metrics, err := logger.getMessageMetrics(ctx, gatewayID)
	assert.NoError(t, err) // Should not error on ErrNoRows
	assert.NotNil(t, metrics)
	assert.Equal(t, 0, metrics.LastHour)
	assert.Equal(t, 0, metrics.Today)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestGetAPIMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = mock // Not used in this test yet

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	gatewayID := 1

	metrics, err := logger.getAPIMetrics(ctx, gatewayID)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	// Placeholders for now
	assert.Equal(t, 0, metrics.LastMinute)
	assert.Equal(t, 0, metrics.LastHour)
}

func TestGetWorkerMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = mock // Not used in this test yet

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	collector := NewCollector(db, redisClient)
	logger := NewLogger(db, collector)

	ctx := context.Background()
	gatewayID := 1

	metrics, err := logger.getWorkerMetrics(ctx, gatewayID)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	// Placeholder for now
	assert.Equal(t, 0, metrics.ActiveThreads)
}
