package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// Collector collects metrics from various sources
type Collector struct {
	db    *sql.DB
	redis *redis.Client
}

// NewCollector creates a new metrics collector
func NewCollector(db *sql.DB, redis *redis.Client) *Collector {
	return &Collector{
		db:    db,
		redis: redis,
	}
}

// Start starts the metrics collection loop
func (c *Collector) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect initial metrics
	c.collectAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collectAll(ctx)
		}
	}
}

// collectAll collects all metrics
func (c *Collector) collectAll(ctx context.Context) {
	c.collectDatabaseMetrics(ctx)
	c.collectRedisMetrics(ctx)
	c.collectQueueMetrics(ctx)
	c.collectSuccessRates(ctx)
	c.collectSystemMetrics(ctx)
}

// collectDatabaseMetrics collects database connection pool metrics
func (c *Collector) collectDatabaseMetrics(ctx context.Context) {
	if c.db == nil {
		return
	}

	stats := c.db.Stats()
	
	SetDatabaseConnections("idle", float64(stats.Idle))
	SetDatabaseConnections("in_use", float64(stats.InUse))
	SetDatabaseConnections("max_open", float64(stats.MaxOpenConnections))
	SetDatabaseConnections("open", float64(stats.OpenConnections))
	SetDatabaseConnections("wait_count", float64(stats.WaitCount))
}

// collectRedisMetrics collects Redis metrics
func (c *Collector) collectRedisMetrics(ctx context.Context) {
	if c.redis == nil {
		return
	}

	// Get Redis info
	info, err := c.redis.Info(ctx, "stats").Result()
	if err != nil {
		return
	}

	// Parse and set metrics (simplified)
	_ = info // TODO: Parse Redis info and set relevant metrics
}

// collectQueueMetrics collects queue size metrics for all tenants
func (c *Collector) collectQueueMetrics(ctx context.Context) {
	if c.redis == nil {
		return
	}

	// Get all tenant keys
	keys, err := c.redis.Keys(ctx, "messages:pending:*").Result()
	if err != nil {
		return
	}

	for _, key := range keys {
		// Extract tenant_id from key
		// Format: messages:pending:{tenant_id}
		var tenantID string
		fmt.Sscanf(key, "messages:pending:%s", &tenantID)

		// Get queue size
		size, err := c.redis.LLen(ctx, key).Result()
		if err != nil {
			continue
		}

		SetQueueSize(tenantID, "pending", float64(size))
	}
}

// collectSuccessRates collects success rates for all tenants
func (c *Collector) collectSuccessRates(ctx context.Context) {
	if c.db == nil {
		return
	}

	query := `
		SELECT 
			tenant_id,
			COUNT(*) FILTER (WHERE status IN ('SENT', 'DELIVERED')) as success_count,
			COUNT(*) as total_count
		FROM messages
		WHERE created_at > NOW() - INTERVAL '1 hour'
		  AND deleted_at IS NULL
		GROUP BY tenant_id
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tenantID int64
		var successCount, totalCount int64

		if err := rows.Scan(&tenantID, &successCount, &totalCount); err != nil {
			continue
		}

		if totalCount > 0 {
			rate := float64(successCount) / float64(totalCount) * 100
			SetSuccessRate(fmt.Sprintf("%d", tenantID), rate)
		}
	}
}

// CollectTenantBalances collects current balances for all tenants
func (c *Collector) CollectTenantBalances(ctx context.Context) {
	if c.db == nil {
		return
	}

	query := `
		SELECT id, balance
		FROM tenants
		WHERE deleted_at IS NULL
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tenantID int64
		var balance float64

		if err := rows.Scan(&tenantID, &balance); err != nil {
			continue
		}

		SetTenantBalance(fmt.Sprintf("%d", tenantID), balance)
	}
}

// collectSystemMetrics collects CPU and Memory usage metrics
func (c *Collector) collectSystemMetrics(ctx context.Context) {
	// CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		ServerLoadCPU.Set(cpuPercent[0])
	}

	// Memory usage
	vMem, err := mem.VirtualMemory()
	if err == nil {
		ServerLoadMemory.Set(vMem.UsedPercent)
	}
}

// CollectCacheHitRates calculates cache hit rates
func (c *Collector) CollectCacheHitRates(ctx context.Context) {
	// This would typically be calculated from the counter metrics
	// For now, we'll leave it as a placeholder
	// In production, you'd track hits and misses separately and calculate the rate
}
