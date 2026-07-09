package resources

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

func TestNewResourceLimiter(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()

	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	assert.NotNil(t, limiter)
	assert.Equal(t, gatewayID, limiter.gatewayID)
	assert.Equal(t, tenantID, limiter.tenantID)
}

func TestGetResourceQuota_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuCores := 4.0
	memoryMB := 8192
	bandwidthMbps := 200

	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, cpuCores, memoryMB, bandwidthMbps,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(rows)

	ctx := context.Background()
	quota, err := limiter.GetResourceQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, quota)
	assert.Equal(t, gatewayID, quota.GatewayID)
	assert.Equal(t, cpuCores, *quota.CPUCores)
	assert.Equal(t, memoryMB, *quota.MemoryMB)
	assert.Equal(t, bandwidthMbps, *quota.BandwidthMbps)
	assert.True(t, quota.IsActive)
	assert.Equal(t, "soft", quota.EnforcementMode)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetResourceQuota_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnError(sql.ErrNoRows)

	ctx := context.Background()
	quota, err := limiter.GetResourceQuota(ctx)

	assert.Error(t, err)
	assert.Nil(t, quota)
	assert.Contains(t, err.Error(), "no active resource quota found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLatestResourceUsage_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuUsage := 75.5
	memoryUsage := 6144
	bandwidthUsage := 150.0

	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		cpuUsage, memoryUsage, bandwidthUsage,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(rows)

	ctx := context.Background()
	usage, err := limiter.GetLatestResourceUsage(ctx)

	require.NoError(t, err)
	assert.NotNil(t, usage)
	assert.Equal(t, gatewayID, usage.GatewayID)
	assert.Equal(t, tenantID, usage.TenantID)
	assert.Equal(t, cpuUsage, *usage.CPUUsagePercent)
	assert.Equal(t, memoryUsage, *usage.MemoryUsageMB)
	assert.Equal(t, bandwidthUsage, *usage.BandwidthUsageMbps)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCPUQuota_NotExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuCores := 4.0
	cpuUsage := 75.0 // 75% usage, not exceeded

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, cpuCores, 8192, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		cpuUsage, 6144, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckCPUQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Exceeded)
	assert.Equal(t, "CPU", result.ResourceType)
	assert.Equal(t, cpuUsage, result.CurrentUsage)
	assert.Equal(t, cpuCores, result.Quota)
	assert.Equal(t, cpuUsage, result.UsagePercent)
	assert.Equal(t, "soft", result.EnforcementMode)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCPUQuota_Exceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuCores := 4.0
	cpuUsage := 100.0 // 100% usage, exceeded

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, cpuCores, 8192, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "hard",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		cpuUsage, 6144, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckCPUQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Exceeded)
	assert.Equal(t, "CPU", result.ResourceType)
	assert.Equal(t, cpuUsage, result.CurrentUsage)
	assert.Equal(t, "hard", result.EnforcementMode)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckMemoryQuota_NotExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	memoryMB := 8192
	memoryUsage := 6144 // 75% usage

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, memoryMB, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, memoryUsage, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckMemoryQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Exceeded)
	assert.Equal(t, "Memory", result.ResourceType)
	assert.Equal(t, memoryUsage, result.CurrentUsage)
	assert.Equal(t, memoryMB, result.Quota)
	assert.InDelta(t, 75.0, result.UsagePercent, 0.1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckMemoryQuota_Exceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	memoryMB := 8192
	memoryUsage := 8200 // Exceeded

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, memoryMB, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "hard",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, memoryUsage, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckMemoryQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Exceeded)
	assert.Equal(t, "Memory", result.ResourceType)
	assert.Equal(t, memoryUsage, result.CurrentUsage)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckBandwidthQuota_NotExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	bandwidthMbps := 200
	bandwidthUsage := 150.0 // 75% usage

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, 8192, bandwidthMbps,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, 6144, bandwidthUsage,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckBandwidthQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Exceeded)
	assert.Equal(t, "Bandwidth", result.ResourceType)
	assert.Equal(t, bandwidthUsage, result.CurrentUsage)
	assert.Equal(t, bandwidthMbps, result.Quota)
	assert.InDelta(t, 75.0, result.UsagePercent, 0.1)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckBandwidthQuota_Exceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	bandwidthMbps := 200
	bandwidthUsage := 210.0 // Exceeded

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, 8192, bandwidthMbps,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "hard",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, 6144, bandwidthUsage,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckBandwidthQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Exceeded)
	assert.Equal(t, "Bandwidth", result.ResourceType)
	assert.Equal(t, bandwidthUsage, result.CurrentUsage)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCPUQuota_NoQuotaSet(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query with NULL cpu_cores
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, nil, 8192, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query (still needed even if quota is NULL)
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, 6144, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.CheckCPUQuota(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Exceeded)
	assert.Equal(t, "CPU", result.ResourceType)
	assert.Nil(t, result.CurrentUsage)
	assert.Nil(t, result.Quota)
	assert.Equal(t, 0.0, result.UsagePercent)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckAllQuotas(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times for CPU, Memory, Bandwidth)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "soft",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		// Mock usage query
		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			75.0, 6144, 150.0,
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	results, err := limiter.CheckAllQuotas(ctx)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 3)
	assert.Contains(t, results, "cpu")
	assert.Contains(t, results, "memory")
	assert.Contains(t, results, "bandwidth")

	// All should not be exceeded
	assert.False(t, results["cpu"].Exceeded)
	assert.False(t, results["memory"].Exceeded)
	assert.False(t, results["bandwidth"].Exceeded)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsQuotaExceeded_NoExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "soft",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			75.0, 6144, 150.0,
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	exceeded, resources, err := limiter.IsQuotaExceeded(ctx)

	require.NoError(t, err)
	assert.False(t, exceeded)
	assert.Empty(t, resources)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsQuotaExceeded_SomeExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "hard",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		// CPU exceeded, Memory exceeded, Bandwidth OK
		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			100.0, 8200, 150.0, // CPU and Memory exceeded
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	exceeded, resources, err := limiter.IsQuotaExceeded(ctx)

	require.NoError(t, err)
	assert.True(t, exceeded)
	assert.Len(t, resources, 2)
	assert.Contains(t, resources, "cpu")
	assert.Contains(t, resources, "memory")

	assert.NoError(t, mock.ExpectationsWereMet())
}


// ============================================================================
// TenantRateLimiter Tests
// ============================================================================

func TestNewTenantRateLimiter(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  50000,
		QueriesPerSecond: 200,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)

	assert.NotNil(t, limiter)
	assert.Equal(t, tenantID, limiter.tenantID)
	assert.Equal(t, gatewayID, limiter.gatewayID)
	assert.Len(t, limiter.limiters, 3)
	assert.Contains(t, limiter.limiters, "api")
	assert.Contains(t, limiter.limiters, "messages")
	assert.Contains(t, limiter.limiters, "queries")
}

func TestTenantRateLimiter_AllowAPI_Success(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 60, // 1 request per second
		MessagesPerHour:  3600,
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// First request should succeed
	err := limiter.AllowAPI(ctx)
	assert.NoError(t, err)
}

func TestTenantRateLimiter_AllowAPI_RateLimitExceeded(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 2, // Very low limit for testing
		MessagesPerHour:  3600,
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// Consume all tokens
	err := limiter.AllowAPI(ctx)
	assert.NoError(t, err)
	err = limiter.AllowAPI(ctx)
	assert.NoError(t, err)

	// Next request should fail
	err = limiter.AllowAPI(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API rate limit exceeded")
	assert.Contains(t, err.Error(), tenantID.String())
}

func TestTenantRateLimiter_AllowMessage_Success(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  3600, // 1 message per second
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// First message should succeed
	err := limiter.AllowMessage(ctx)
	assert.NoError(t, err)
}

func TestTenantRateLimiter_AllowMessage_RateLimitExceeded(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  10, // Very low limit for testing (burst = 1)
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// Consume the burst token
	err := limiter.AllowMessage(ctx)
	assert.NoError(t, err)

	// Next message should fail
	err = limiter.AllowMessage(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message rate limit exceeded")
	assert.Contains(t, err.Error(), tenantID.String())
}

func TestTenantRateLimiter_AllowQuery_Success(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  3600,
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// First query should succeed
	err := limiter.AllowQuery(ctx)
	assert.NoError(t, err)
}

func TestTenantRateLimiter_AllowQuery_RateLimitExceeded(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  3600,
		QueriesPerSecond: 1, // Very low limit for testing (burst = 2)
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// Consume all burst tokens
	err := limiter.AllowQuery(ctx)
	assert.NoError(t, err)
	err = limiter.AllowQuery(ctx)
	assert.NoError(t, err)

	// Next query should fail
	err = limiter.AllowQuery(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query rate limit exceeded")
	assert.Contains(t, err.Error(), tenantID.String())
}

func TestTenantRateLimiter_AllowAPIWithWait_Success(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 60, // 1 request per second
		MessagesPerHour:  3600,
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// First request should succeed immediately
	err := limiter.AllowAPIWithWait(ctx)
	assert.NoError(t, err)
}

func TestTenantRateLimiter_AllowAPIWithWait_ContextCancelled(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 2, // Very low limit
		MessagesPerHour:  3600,
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	
	// Consume all tokens
	ctx := context.Background()
	limiter.AllowAPI(ctx)
	limiter.AllowAPI(ctx)

	// Create a context that's already cancelled
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should fail immediately due to cancelled context
	err := limiter.AllowAPIWithWait(cancelledCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API rate limit wait failed")
}

func TestTenantRateLimiter_GetRateLimitStatus(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  50000,
		QueriesPerSecond: 200,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	
	status := limiter.GetRateLimitStatus()
	
	assert.NotNil(t, status)
	assert.Len(t, status, 3)
	assert.Contains(t, status, "api")
	assert.Contains(t, status, "messages")
	assert.Contains(t, status, "queries")

	// Check API limiter status
	apiStatus := status["api"].(map[string]interface{})
	assert.NotNil(t, apiStatus["limit"])
	assert.NotNil(t, apiStatus["burst"])
	assert.NotNil(t, apiStatus["available"])
}

func TestTenantRateLimiter_UpdateRateLimits(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	initialConfig := &RateLimitConfig{
		APIRatePerMinute: 300,
		MessagesPerHour:  50000,
		QueriesPerSecond: 200,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, initialConfig)
	
	// Update with new config
	newConfig := &RateLimitConfig{
		APIRatePerMinute: 600,
		MessagesPerHour:  100000,
		QueriesPerSecond: 400,
	}
	
	limiter.UpdateRateLimits(newConfig)
	
	// Verify the limits were updated
	status := limiter.GetRateLimitStatus()
	apiStatus := status["api"].(map[string]interface{})
	
	// The limit should be updated (600/60 = 10 per second)
	limitValue := float64(apiStatus["limit"].(rate.Limit))
	assert.InDelta(t, 10.0, limitValue, 0.01)
	assert.Equal(t, 600, apiStatus["burst"].(int))
}

func TestTenantRateLimiter_TokenBucketBehavior(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 60, // 1 request per second, burst of 60
		MessagesPerHour:  3600,
		QueriesPerSecond: 100,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// Test burst capability - should be able to make 60 requests immediately
	successCount := 0
	for i := 0; i < 70; i++ {
		err := limiter.AllowAPI(ctx)
		if err == nil {
			successCount++
		}
	}

	// Should have succeeded for burst amount (60)
	assert.Equal(t, 60, successCount)

	// Wait for tokens to refill (1 second = 1 token)
	time.Sleep(2 * time.Second)

	// Should be able to make 2 more requests
	err := limiter.AllowAPI(ctx)
	assert.NoError(t, err)
	err = limiter.AllowAPI(ctx)
	assert.NoError(t, err)
}

func TestTenantRateLimiter_IndependentLimiters(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 2,
		MessagesPerHour:  10,
		QueriesPerSecond: 1,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// Exhaust API limiter
	limiter.AllowAPI(ctx)
	limiter.AllowAPI(ctx)
	err := limiter.AllowAPI(ctx)
	assert.Error(t, err)

	// Message limiter should still work
	err = limiter.AllowMessage(ctx)
	assert.NoError(t, err)

	// Query limiter should still work
	err = limiter.AllowQuery(ctx)
	assert.NoError(t, err)
}

func TestResourceLimiter_LoadRateLimitConfigFromDB_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	apiRate := 500
	messagesPerHour := 100000
	queriesPerSecond := 300

	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, 8192, 200,
		100, 100, 50, queriesPerSecond, 1024, 500000,
		messagesPerHour, 500000, 5000, apiRate, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(rows)

	ctx := context.Background()
	config, err := limiter.LoadRateLimitConfigFromDB(ctx)

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, apiRate, config.APIRatePerMinute)
	assert.Equal(t, messagesPerHour, config.MessagesPerHour)
	assert.Equal(t, queriesPerSecond, config.QueriesPerSecond)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResourceLimiter_LoadRateLimitConfigFromDB_DefaultValues(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Return NULL values for rate limits
	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, 8192, 200,
		100, 100, 50, nil, 1024, 500000,
		nil, 500000, 5000, nil, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(rows)

	ctx := context.Background()
	config, err := limiter.LoadRateLimitConfigFromDB(ctx)

	require.NoError(t, err)
	assert.NotNil(t, config)
	// Should use default values
	assert.Equal(t, 300, config.APIRatePerMinute)
	assert.Equal(t, 50000, config.MessagesPerHour)
	assert.Equal(t, 200, config.QueriesPerSecond)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResourceLimiter_CreateTenantRateLimiter_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	apiRate := 500
	messagesPerHour := 100000
	queriesPerSecond := 300

	rows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, 8192, 200,
		100, 100, 50, queriesPerSecond, 1024, 500000,
		messagesPerHour, 500000, 5000, apiRate, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(rows)

	ctx := context.Background()
	rateLimiter, err := limiter.CreateTenantRateLimiter(ctx)

	require.NoError(t, err)
	assert.NotNil(t, rateLimiter)
	assert.Equal(t, tenantID, rateLimiter.tenantID)
	assert.Equal(t, gatewayID, rateLimiter.gatewayID)
	assert.Len(t, rateLimiter.limiters, 3)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRateLimiter_ConcurrentAccess(t *testing.T) {
	tenantID := uuid.New()
	gatewayID := 1
	config := &RateLimitConfig{
		APIRatePerMinute: 1000,
		MessagesPerHour:  100000,
		QueriesPerSecond: 500,
	}

	limiter := NewTenantRateLimiter(tenantID, gatewayID, config)
	ctx := context.Background()

	// Test concurrent access to rate limiter
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.AllowAPI(ctx)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should have succeeded for some requests (up to burst limit)
	assert.Greater(t, successCount, 0)
	assert.LessOrEqual(t, successCount, 1000) // Burst limit
}

// ============================================================================
// Enforcement Policy Tests
// Requirement 9.11: تطبيق Soft Limit (تحذير فقط)
// Requirement 9.12: تطبيق Hard Limit (رفض الطلبات)
// ============================================================================

func TestEnforceQuota_SoftLimit_NotExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuCores := 4.0
	cpuUsage := 75.0 // 75% usage, not exceeded

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, cpuCores, 8192, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		cpuUsage, 6144, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.EnforceQuota(ctx, "cpu")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "Request should be allowed when quota is not exceeded")
	assert.Equal(t, EnforcementSoft, result.Policy)
	assert.Equal(t, "CPU", result.ResourceType)
	assert.Empty(t, result.RejectionReason)
	assert.Empty(t, result.WarningMessage, "No warning when usage is below 90%")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnforceQuota_SoftLimit_ApproachingLimit(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuCores := 4.0
	cpuUsage := 92.0 // 92% usage, approaching limit

	// Mock quota query
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, cpuCores, 8192, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		cpuUsage, 6144, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.EnforceQuota(ctx, "cpu")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "Request should be allowed with soft limit")
	assert.Equal(t, EnforcementSoft, result.Policy)
	assert.NotEmpty(t, result.WarningMessage, "Should have warning when approaching limit (>=90%)")
	assert.Contains(t, result.WarningMessage, "approaching limit")
	assert.Empty(t, result.RejectionReason)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnforceQuota_SoftLimit_Exceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	cpuCores := 4.0
	cpuUsage := 105.0 // 105% usage, exceeded

	// Mock quota query with soft enforcement
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, cpuCores, 8192, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "soft",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		cpuUsage, 6144, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.EnforceQuota(ctx, "cpu")

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Requirement 9.11: Soft Limit allows request but sends warning
	assert.True(t, result.Allowed, "Request should be allowed with soft limit even when exceeded")
	assert.Equal(t, EnforcementSoft, result.Policy)
	assert.NotEmpty(t, result.WarningMessage, "Should have warning when quota is exceeded")
	assert.Contains(t, result.WarningMessage, "WARNING")
	assert.Contains(t, result.WarningMessage, "quota exceeded")
	assert.Empty(t, result.RejectionReason)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnforceQuota_HardLimit_NotExceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	memoryMB := 8192
	memoryUsage := 6144 // 75% usage

	// Mock quota query with hard enforcement
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, memoryMB, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "hard",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, memoryUsage, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.EnforceQuota(ctx, "memory")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "Request should be allowed when quota is not exceeded")
	assert.Equal(t, EnforcementHard, result.Policy)
	assert.Equal(t, "Memory", result.ResourceType)
	assert.Empty(t, result.RejectionReason)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnforceQuota_HardLimit_Exceeded(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	memoryMB := 8192
	memoryUsage := 8300 // Exceeded

	// Mock quota query with hard enforcement
	quotaRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
		"max_concurrent_connections", "storage_gb", "max_db_connections",
		"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
		"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
		"api_rate_limit_per_minute", "api_rate_limit_per_hour",
		"max_worker_threads", "is_active", "enforcement_mode",
		"created_at", "updated_at",
	}).AddRow(
		1, gatewayID, 4.0, memoryMB, 200,
		100, 100, 50, 200, 1024, 500000,
		50000, 500000, 5000, 300, 15000,
		20, true, "hard",
		time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
		WithArgs(gatewayID).
		WillReturnRows(quotaRows)

	// Mock usage query
	usageRows := sqlmock.NewRows([]string{
		"id", "gateway_id", "tenant_id", "timestamp",
		"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
		"active_connections", "db_connections_active", "queries_per_second",
		"redis_memory_usage_mb", "redis_keys_count",
		"messages_processed_last_hour", "messages_processed_today",
		"current_queue_depth", "api_requests_last_minute",
		"api_requests_last_hour", "active_worker_threads", "created_at",
	}).AddRow(
		1, gatewayID, tenantID, time.Now(),
		75.0, memoryUsage, 150.0,
		50, 30, 150, 512, 100000,
		5000, 50000, 100, 50, 500, 15, time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM resource_usage").
		WithArgs(gatewayID, tenantID).
		WillReturnRows(usageRows)

	ctx := context.Background()
	result, err := limiter.EnforceQuota(ctx, "memory")

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Requirement 9.12: Hard Limit rejects requests when quota is exceeded
	assert.False(t, result.Allowed, "Request should be rejected with hard limit when exceeded")
	assert.Equal(t, EnforcementHard, result.Policy)
	assert.NotEmpty(t, result.RejectionReason, "Should have rejection reason")
	assert.Contains(t, result.RejectionReason, "quota exceeded")
	assert.Contains(t, result.RejectionReason, "Request rejected")
	assert.Empty(t, result.WarningMessage)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnforceQuota_InvalidResourceType(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	ctx := context.Background()
	result, err := limiter.EnforceQuota(ctx, "invalid_resource")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestEnforceAllQuotas_AllAllowed(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times for CPU, Memory, Bandwidth)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "soft",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		// All resources within limits
		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			75.0, 6144, 150.0,
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	result, err := limiter.EnforceAllQuotas(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Allowed, "All quotas within limits, request should be allowed")
	assert.Equal(t, EnforcementSoft, result.Policy)
	assert.Equal(t, "all", result.ResourceType)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnforceAllQuotas_SomeExceeded_HardLimit(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "hard",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		// CPU and Memory exceeded
		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			105.0, 8300, 150.0, // CPU and Memory exceeded
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	result, err := limiter.EnforceAllQuotas(ctx)

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Requirement 9.12: Hard Limit rejects requests when any quota is exceeded
	assert.False(t, result.Allowed, "Request should be rejected when any hard limit is exceeded")
	assert.Equal(t, EnforcementHard, result.Policy)
	assert.Equal(t, "multiple", result.ResourceType)
	assert.NotEmpty(t, result.RejectionReason)
	assert.Contains(t, result.RejectionReason, "Multiple quotas exceeded")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldAllowRequest_Allowed(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "soft",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			75.0, 6144, 150.0,
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	allowed, message, err := limiter.ShouldAllowRequest(ctx)

	require.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed")
	assert.NotEmpty(t, message) // May contain warnings

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldAllowRequest_Rejected(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	gatewayID := 1
	tenantID := uuid.New()
	limiter := NewResourceLimiter(db, gatewayID, tenantID)

	// Mock quota query (called 3 times)
	for i := 0; i < 3; i++ {
		quotaRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "cpu_cores", "memory_mb", "bandwidth_mbps",
			"max_concurrent_connections", "storage_gb", "max_db_connections",
			"max_queries_per_second", "redis_memory_mb", "redis_max_keys",
			"max_messages_per_hour", "max_messages_per_day", "max_queue_depth",
			"api_rate_limit_per_minute", "api_rate_limit_per_hour",
			"max_worker_threads", "is_active", "enforcement_mode",
			"created_at", "updated_at",
		}).AddRow(
			1, gatewayID, 4.0, 8192, 200,
			100, 100, 50, 200, 1024, 500000,
			50000, 500000, 5000, 300, 15000,
			20, true, "hard",
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_quotas").
			WithArgs(gatewayID).
			WillReturnRows(quotaRows)

		// CPU exceeded
		usageRows := sqlmock.NewRows([]string{
			"id", "gateway_id", "tenant_id", "timestamp",
			"cpu_usage_percent", "memory_usage_mb", "bandwidth_usage_mbps",
			"active_connections", "db_connections_active", "queries_per_second",
			"redis_memory_usage_mb", "redis_keys_count",
			"messages_processed_last_hour", "messages_processed_today",
			"current_queue_depth", "api_requests_last_minute",
			"api_requests_last_hour", "active_worker_threads", "created_at",
		}).AddRow(
			1, gatewayID, tenantID, time.Now(),
			105.0, 6144, 150.0, // CPU exceeded
			50, 30, 150, 512, 100000,
			5000, 50000, 100, 50, 500, 15, time.Now(),
		)

		mock.ExpectQuery("SELECT (.+) FROM resource_usage").
			WithArgs(gatewayID, tenantID).
			WillReturnRows(usageRows)
	}

	ctx := context.Background()
	allowed, message, err := limiter.ShouldAllowRequest(ctx)

	require.NoError(t, err)
	assert.False(t, allowed, "Request should be rejected")
	assert.NotEmpty(t, message, "Should have rejection reason")
	assert.Contains(t, message, "exceeded")

	assert.NoError(t, mock.ExpectationsWereMet())
}
