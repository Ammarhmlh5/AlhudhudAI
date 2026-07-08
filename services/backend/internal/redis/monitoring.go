package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// MonitoringMetrics holds Redis monitoring metrics
type MonitoringMetrics struct {
	UsedMemory        int64
	MaxMemory         int64
	MemoryUsagePercent float64
	ConnectedClients  int64
	TotalCommands     int64
	EvictedKeys       int64
	ExpiredKeys       int64
	KeyspaceHits      int64
	KeyspaceMisses    int64
	HitRate           float64
	SlowLogCount      int64
}

// Monitor provides Redis monitoring capabilities
type Monitor struct {
	client *Client
	logger *zap.Logger
}

// NewMonitor creates a new Redis monitor
func NewMonitor(client *Client, logger *zap.Logger) *Monitor {
	return &Monitor{
		client: client,
		logger: logger,
	}
}

// GetMetrics retrieves current Redis metrics
func (m *Monitor) GetMetrics(ctx context.Context) (*MonitoringMetrics, error) {
	info, err := m.client.Client.Info(ctx, "memory", "stats", "clients").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	metrics := &MonitoringMetrics{}
	lines := strings.Split(info, "\r\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "used_memory":
			metrics.UsedMemory, _ = strconv.ParseInt(value, 10, 64)
		case "maxmemory":
			metrics.MaxMemory, _ = strconv.ParseInt(value, 10, 64)
		case "connected_clients":
			metrics.ConnectedClients, _ = strconv.ParseInt(value, 10, 64)
		case "total_commands_processed":
			metrics.TotalCommands, _ = strconv.ParseInt(value, 10, 64)
		case "evicted_keys":
			metrics.EvictedKeys, _ = strconv.ParseInt(value, 10, 64)
		case "expired_keys":
			metrics.ExpiredKeys, _ = strconv.ParseInt(value, 10, 64)
		case "keyspace_hits":
			metrics.KeyspaceHits, _ = strconv.ParseInt(value, 10, 64)
		case "keyspace_misses":
			metrics.KeyspaceMisses, _ = strconv.ParseInt(value, 10, 64)
		}
	}

	// Calculate derived metrics
	if metrics.MaxMemory > 0 {
		metrics.MemoryUsagePercent = float64(metrics.UsedMemory) / float64(metrics.MaxMemory) * 100
	}

	totalKeyspaceOps := metrics.KeyspaceHits + metrics.KeyspaceMisses
	if totalKeyspaceOps > 0 {
		metrics.HitRate = float64(metrics.KeyspaceHits) / float64(totalKeyspaceOps) * 100
	}

	// Get slow log count
	slowLog, err := m.client.Client.SlowLogGet(ctx, 1).Result()
	if err == nil {
		metrics.SlowLogCount = int64(len(slowLog))
	}

	return metrics, nil
}

// CheckHealth performs health checks and returns warnings
func (m *Monitor) CheckHealth(ctx context.Context) []string {
	var warnings []string

	metrics, err := m.GetMetrics(ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to get metrics: %v", err))
		return warnings
	}

	// Check memory usage
	if metrics.MemoryUsagePercent > 90 {
		warnings = append(warnings, fmt.Sprintf("CRITICAL: Redis memory usage is %.2f%% (used: %d MB / max: %d MB)",
			metrics.MemoryUsagePercent,
			metrics.UsedMemory/1024/1024,
			metrics.MaxMemory/1024/1024))
	} else if metrics.MemoryUsagePercent > 80 {
		warnings = append(warnings, fmt.Sprintf("WARNING: Redis memory usage is %.2f%%", metrics.MemoryUsagePercent))
	}

	// Check eviction rate
	if metrics.EvictedKeys > 1000 {
		warnings = append(warnings, fmt.Sprintf("WARNING: High eviction rate detected (%d keys evicted)", metrics.EvictedKeys))
	}

	// Check hit rate
	if metrics.HitRate < 80 && (metrics.KeyspaceHits+metrics.KeyspaceMisses) > 1000 {
		warnings = append(warnings, fmt.Sprintf("WARNING: Low cache hit rate (%.2f%%)", metrics.HitRate))
	}

	// Check connection count
	if metrics.ConnectedClients > 1000 {
		warnings = append(warnings, fmt.Sprintf("WARNING: High number of connected clients (%d)", metrics.ConnectedClients))
	}

	return warnings
}

// StartMonitoring starts periodic monitoring
func (m *Monitor) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Redis monitoring stopped")
			return
		case <-ticker.C:
			warnings := m.CheckHealth(ctx)
			if len(warnings) > 0 {
				for _, warning := range warnings {
					m.logger.Warn("Redis health check warning", zap.String("warning", warning))
				}
			}

			// Log metrics
			metrics, err := m.GetMetrics(ctx)
			if err == nil {
				m.logger.Info("Redis metrics",
					zap.Float64("memory_usage_percent", metrics.MemoryUsagePercent),
					zap.Int64("connected_clients", metrics.ConnectedClients),
					zap.Float64("hit_rate", metrics.HitRate),
					zap.Int64("evicted_keys", metrics.EvictedKeys))
			}
		}
	}
}
