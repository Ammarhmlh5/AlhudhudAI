package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Collector collects metrics for dedicated gateways
type Collector struct {
	db              *sql.DB
	redis           *redis.Client
	lastNetStats    map[string]net.IOCountersStat
	lastNetStatsTime time.Time
}

// NewCollector creates a new metrics collector
func NewCollector(db *sql.DB, redis *redis.Client) *Collector {
	return &Collector{
		db:              db,
		redis:           redis,
		lastNetStats:    make(map[string]net.IOCountersStat),
		lastNetStatsTime: time.Now(),
	}
}

// CollectAll collects all metrics for all active dedicated gateways
func (c *Collector) CollectAll(ctx context.Context) ([]GatewayMetrics, error) {
	// Get all active dedicated gateways
	gateways, err := c.getActiveDedicatedGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active gateways: %w", err)
	}

	var allMetrics []GatewayMetrics

	for _, gateway := range gateways {
		metrics, err := c.CollectForGateway(ctx, gateway.ID, gateway.TenantID)
		if err != nil {
			// Log error but continue with other gateways
			fmt.Printf("Error collecting metrics for gateway %d: %v\n", gateway.ID, err)
			continue
		}
		allMetrics = append(allMetrics, *metrics)
	}

	return allMetrics, nil
}

// CollectForGateway collects metrics for a specific gateway
func (c *Collector) CollectForGateway(ctx context.Context, gatewayID int, tenantID uuid.UUID) (*GatewayMetrics, error) {
	metrics := &GatewayMetrics{
		GatewayID: gatewayID,
		TenantID:  tenantID,
		Timestamp: time.Now(),
	}

	// Collect system metrics
	systemMetrics, err := c.collectSystemMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect system metrics: %w", err)
	}
	metrics.SystemMetrics = *systemMetrics

	// Collect database metrics
	dbMetrics, err := c.collectDatabaseMetrics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to collect database metrics: %w", err)
	}
	metrics.DatabaseMetrics = *dbMetrics

	// Collect Redis metrics
	redisMetrics, err := c.collectRedisMetrics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Redis metrics: %w", err)
	}
	metrics.RedisMetrics = *redisMetrics

	return metrics, nil
}

// collectSystemMetrics collects CPU, Memory, and Bandwidth metrics
func (c *Collector) collectSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// Collect CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}
	if len(cpuPercent) > 0 {
		metrics.CPUUsagePercent = cpuPercent[0]
	}

	// Collect Memory usage
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage: %w", err)
	}
	metrics.MemoryUsageMB = float64(vmStat.Used) / 1024 / 1024
	metrics.MemoryUsagePercent = vmStat.UsedPercent

	// Collect Bandwidth usage
	bandwidth, err := c.calculateBandwidthUsage(ctx)
	if err != nil {
		// Log error but don't fail - bandwidth is not critical
		fmt.Printf("Warning: failed to calculate bandwidth: %v\n", err)
		metrics.BandwidthUsageKBps = 0
	} else {
		metrics.BandwidthUsageKBps = bandwidth
	}

	return metrics, nil
}

// calculateBandwidthUsage calculates bandwidth usage in KB/s
func (c *Collector) calculateBandwidthUsage(ctx context.Context) (float64, error) {
	netStats, err := net.IOCounters(true)
	if err != nil {
		return 0, fmt.Errorf("failed to get network stats: %w", err)
	}

	now := time.Now()
	timeDiff := now.Sub(c.lastNetStatsTime).Seconds()

	if timeDiff == 0 || len(c.lastNetStats) == 0 {
		// First run, store stats and return 0
		for _, stat := range netStats {
			c.lastNetStats[stat.Name] = stat
		}
		c.lastNetStatsTime = now
		return 0, nil
	}

	var totalBytesSent, totalBytesRecv uint64

	for _, stat := range netStats {
		if lastStat, exists := c.lastNetStats[stat.Name]; exists {
			totalBytesSent += stat.BytesSent - lastStat.BytesSent
			totalBytesRecv += stat.BytesRecv - lastStat.BytesRecv
		}
		c.lastNetStats[stat.Name] = stat
	}

	c.lastNetStatsTime = now

	// Calculate KB/s
	totalBytes := totalBytesSent + totalBytesRecv
	kbps := float64(totalBytes) / 1024 / timeDiff

	return kbps, nil
}

// collectDatabaseMetrics collects database connection and query metrics
func (c *Collector) collectDatabaseMetrics(ctx context.Context, tenantID uuid.UUID) (*DatabaseMetrics, error) {
	metrics := &DatabaseMetrics{}

	// Get database connection pool stats
	stats := c.db.Stats()
	metrics.ActiveConnections = stats.InUse
	metrics.IdleConnections = stats.Idle
	metrics.MaxConnections = stats.MaxOpenConnections

	// Calculate queries per second
	// This is a simplified calculation - in production, you'd track this more accurately
	qps, err := c.calculateQueriesPerSecond(ctx, tenantID)
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to calculate QPS: %v\n", err)
		metrics.QueriesPerSecond = 0
	} else {
		metrics.QueriesPerSecond = qps
	}

	// Calculate average query time
	avgQueryTime, err := c.calculateAvgQueryTime(ctx, tenantID)
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to calculate avg query time: %v\n", err)
		metrics.AvgQueryTime = 0
	} else {
		metrics.AvgQueryTime = avgQueryTime
	}

	return metrics, nil
}

// calculateQueriesPerSecond calculates the number of queries per second
func (c *Collector) calculateQueriesPerSecond(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	// Query PostgreSQL statistics
	query := `
		SELECT 
			COALESCE(SUM(calls), 0) as total_calls
		FROM pg_stat_statements
		WHERE query NOT LIKE '%pg_stat_statements%'
		LIMIT 1
	`

	var totalCalls int64
	err := c.db.QueryRowContext(ctx, query).Scan(&totalCalls)
	if err != nil {
		// pg_stat_statements might not be enabled, return 0
		return 0, nil
	}

	// This is a simplified calculation
	// In production, you'd track the difference over time
	return float64(totalCalls) / 60.0, nil // Approximate QPS over last minute
}

// calculateAvgQueryTime calculates average query execution time
func (c *Collector) calculateAvgQueryTime(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	// Query PostgreSQL statistics
	query := `
		SELECT 
			COALESCE(AVG(mean_exec_time), 0) as avg_time
		FROM pg_stat_statements
		WHERE query NOT LIKE '%pg_stat_statements%'
	`

	var avgTime float64
	err := c.db.QueryRowContext(ctx, query).Scan(&avgTime)
	if err != nil {
		// pg_stat_statements might not be enabled, return 0
		return 0, nil
	}

	return avgTime, nil
}

// collectRedisMetrics collects Redis memory and cache metrics
func (c *Collector) collectRedisMetrics(ctx context.Context, tenantID uuid.UUID) (*RedisMetrics, error) {
	metrics := &RedisMetrics{}

	// Get Redis INFO
	info, err := c.redis.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Parse memory usage
	memoryUsage, err := c.parseRedisMemory(info)
	if err != nil {
		fmt.Printf("Warning: failed to parse Redis memory: %v\n", err)
	} else {
		metrics.MemoryUsed = int64(memoryUsage * 1024 * 1024)
	}

	// Get total keys for this tenant
	pattern := fmt.Sprintf("tenant:%s:*", tenantID.String())
	keys, err := c.redis.Keys(ctx, pattern).Result()
	if err != nil {
		fmt.Printf("Warning: failed to count Redis keys: %v\n", err)
	} else {
		metrics.KeysCount = int64(len(keys))
	}

	// Calculate cache hit rate
	hitRate, err := c.calculateCacheHitRate(ctx, tenantID)
	if err != nil {
		fmt.Printf("Warning: failed to calculate cache hit rate: %v\n", err)
		metrics.HitRate = 0
	} else {
		metrics.HitRate = hitRate
	}

	// Get commands per second
	cps, err := c.calculateCommandsPerSecond(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to calculate commands per second: %v\n", err)
		metrics.OpsPerSec = 0
	} else {
		metrics.OpsPerSec = int64(cps)
	}

	return metrics, nil
}

// parseRedisMemory parses Redis memory usage from INFO output
func (c *Collector) parseRedisMemory(info string) (float64, error) {
	// Simple parsing - in production, use a proper parser
	var usedMemory int64
	_, err := fmt.Sscanf(info, "used_memory:%d", &usedMemory)
	if err != nil {
		return 0, err
	}
	return float64(usedMemory) / 1024 / 1024, nil
}

// calculateCacheHitRate calculates Redis cache hit rate
func (c *Collector) calculateCacheHitRate(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	// Get Redis stats
	info, err := c.redis.Info(ctx, "stats").Result()
	if err != nil {
		return 0, err
	}

	var hits, misses int64
	fmt.Sscanf(info, "keyspace_hits:%d", &hits)
	fmt.Sscanf(info, "keyspace_misses:%d", &misses)

	total := hits + misses
	if total == 0 {
		return 0, nil
	}

	return float64(hits) / float64(total) * 100, nil
}

// calculateCommandsPerSecond calculates Redis commands per second
func (c *Collector) calculateCommandsPerSecond(ctx context.Context) (float64, error) {
	// Get Redis stats
	info, err := c.redis.Info(ctx, "stats").Result()
	if err != nil {
		return 0, err
	}

	var totalCommands int64
	fmt.Sscanf(info, "total_commands_processed:%d", &totalCommands)

	// This is a simplified calculation
	// In production, you'd track the difference over time
	return float64(totalCommands) / 60.0, nil // Approximate CPS over last minute
}

// getActiveDedicatedGateways retrieves all active dedicated gateways
func (c *Collector) getActiveDedicatedGateways(ctx context.Context) ([]struct {
	ID       int
	TenantID uuid.UUID
}, error) {
	query := `
		SELECT id, tenant_id
		FROM dedicated_gateways
		WHERE status = 'active'
		  AND deleted_at IS NULL
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gateways []struct {
		ID       int
		TenantID uuid.UUID
	}

	for rows.Next() {
		var gateway struct {
			ID       int
			TenantID uuid.UUID
		}
		if err := rows.Scan(&gateway.ID, &gateway.TenantID); err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}

	return gateways, rows.Err()
}

// GetSystemInfo returns general system information
func (c *Collector) GetSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"num_cpu":      runtime.NumCPU(),
		"go_version":   runtime.Version(),
		"num_goroutine": runtime.NumGoroutine(),
	}
}
