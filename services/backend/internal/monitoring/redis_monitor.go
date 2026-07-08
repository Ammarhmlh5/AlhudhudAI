package monitoring

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisMonitor monitors Redis performance and health
// Task 35.2: Redis Optimization - Monitoring
// Requirements: 29.8
type RedisMonitor struct {
	client *redis.Client
	logger *zap.Logger
}

// RedisMetrics contains Redis performance metrics
type RedisMetrics struct {
	// Memory metrics
	MemoryUsed        int64   `json:"memory_used"`
	MemoryUsedHuman   string  `json:"memory_used_human"`
	MemoryPeak        int64   `json:"memory_peak"`
	MemoryPeakHuman   string  `json:"memory_peak_human"`
	MemoryFragRatio   float64 `json:"memory_frag_ratio"`
	
	// Key metrics
	KeysCount         int64   `json:"keys_count"`
	ExpiringKeys      int64   `json:"expiring_keys"`
	EvictedKeys       int64   `json:"evicted_keys"`
	ExpiredKeys       int64   `json:"expired_keys"`
	
	// Performance metrics
	HitRate           float64 `json:"hit_rate"`
	Hits              int64   `json:"hits"`
	Misses            int64   `json:"misses"`
	OpsPerSec         int64   `json:"ops_per_sec"`
	
	// Connection metrics
	ConnectedClients  int64   `json:"connected_clients"`
	BlockedClients    int64   `json:"blocked_clients"`
	MaxClients        int64   `json:"max_clients"`
	
	// Persistence metrics
	LastSaveTime      int64   `json:"last_save_time"`
	ChangesSinceLastSave int64 `json:"changes_since_last_save"`
	
	// Replication metrics
	Role              string  `json:"role"`
	ConnectedSlaves   int64   `json:"connected_slaves"`
	
	// CPU metrics
	UsedCPUSys        float64 `json:"used_cpu_sys"`
	UsedCPUUser       float64 `json:"used_cpu_user"`
	
	// Network metrics
	TotalNetInput     int64   `json:"total_net_input"`
	TotalNetOutput    int64   `json:"total_net_output"`
	
	// Timestamp
	Timestamp         time.Time `json:"timestamp"`
}

// NewRedisMonitor creates a new Redis monitor
func NewRedisMonitor(client *redis.Client, logger *zap.Logger) *RedisMonitor {
	return &RedisMonitor{
		client: client,
		logger: logger,
	}
}

// CollectMetrics collects Redis metrics
// Requirements: 29.8 - يراقب استخدام Redis (Memory، Keys، Hit Rate)
func (rm *RedisMonitor) CollectMetrics(ctx context.Context) (*RedisMetrics, error) {
	// Get INFO stats
	info, err := rm.client.Info(ctx, "all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	metrics := &RedisMetrics{
		Timestamp: time.Now(),
	}

	// Parse INFO output
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		// Memory metrics
		case "used_memory":
			metrics.MemoryUsed, _ = strconv.ParseInt(value, 10, 64)
		case "used_memory_human":
			metrics.MemoryUsedHuman = value
		case "used_memory_peak":
			metrics.MemoryPeak, _ = strconv.ParseInt(value, 10, 64)
		case "used_memory_peak_human":
			metrics.MemoryPeakHuman = value
		case "mem_fragmentation_ratio":
			metrics.MemoryFragRatio, _ = strconv.ParseFloat(value, 64)

		// Key metrics
		case "evicted_keys":
			metrics.EvictedKeys, _ = strconv.ParseInt(value, 10, 64)
		case "expired_keys":
			metrics.ExpiredKeys, _ = strconv.ParseInt(value, 10, 64)

		// Performance metrics
		case "keyspace_hits":
			metrics.Hits, _ = strconv.ParseInt(value, 10, 64)
		case "keyspace_misses":
			metrics.Misses, _ = strconv.ParseInt(value, 10, 64)
		case "instantaneous_ops_per_sec":
			metrics.OpsPerSec, _ = strconv.ParseInt(value, 10, 64)

		// Connection metrics
		case "connected_clients":
			metrics.ConnectedClients, _ = strconv.ParseInt(value, 10, 64)
		case "blocked_clients":
			metrics.BlockedClients, _ = strconv.ParseInt(value, 10, 64)
		case "maxclients":
			metrics.MaxClients, _ = strconv.ParseInt(value, 10, 64)

		// Persistence metrics
		case "rdb_last_save_time":
			metrics.LastSaveTime, _ = strconv.ParseInt(value, 10, 64)
		case "rdb_changes_since_last_save":
			metrics.ChangesSinceLastSave, _ = strconv.ParseInt(value, 10, 64)

		// Replication metrics
		case "role":
			metrics.Role = value
		case "connected_slaves":
			metrics.ConnectedSlaves, _ = strconv.ParseInt(value, 10, 64)

		// CPU metrics
		case "used_cpu_sys":
			metrics.UsedCPUSys, _ = strconv.ParseFloat(value, 64)
		case "used_cpu_user":
			metrics.UsedCPUUser, _ = strconv.ParseFloat(value, 64)

		// Network metrics
		case "total_net_input_bytes":
			metrics.TotalNetInput, _ = strconv.ParseInt(value, 10, 64)
		case "total_net_output_bytes":
			metrics.TotalNetOutput, _ = strconv.ParseInt(value, 10, 64)
		}
	}

	// Calculate hit rate
	if metrics.Hits+metrics.Misses > 0 {
		metrics.HitRate = float64(metrics.Hits) / float64(metrics.Hits+metrics.Misses) * 100
	}

	// Get keys count
	metrics.KeysCount, err = rm.getKeysCount(ctx)
	if err != nil {
		rm.logger.Warn("Failed to get keys count", zap.Error(err))
	}

	// Get expiring keys count
	metrics.ExpiringKeys, err = rm.getExpiringKeysCount(ctx)
	if err != nil {
		rm.logger.Warn("Failed to get expiring keys count", zap.Error(err))
	}

	return metrics, nil
}

// getKeysCount returns total number of keys
func (rm *RedisMonitor) getKeysCount(ctx context.Context) (int64, error) {
	info, err := rm.client.Info(ctx, "keyspace").Result()
	if err != nil {
		return 0, err
	}

	var total int64
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "db") {
			// Format: db0:keys=123,expires=45,avg_ttl=67890
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				keyValue := strings.Split(parts[1], ",")
				for _, kv := range keyValue {
					if strings.HasPrefix(kv, "keys=") {
						count, _ := strconv.ParseInt(strings.TrimPrefix(kv, "keys="), 10, 64)
						total += count
					}
				}
			}
		}
	}

	return total, nil
}

// getExpiringKeysCount returns number of keys with expiration
func (rm *RedisMonitor) getExpiringKeysCount(ctx context.Context) (int64, error) {
	info, err := rm.client.Info(ctx, "keyspace").Result()
	if err != nil {
		return 0, err
	}

	var total int64
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "db") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				keyValue := strings.Split(parts[1], ",")
				for _, kv := range keyValue {
					if strings.HasPrefix(kv, "expires=") {
						count, _ := strconv.ParseInt(strings.TrimPrefix(kv, "expires="), 10, 64)
						total += count
					}
				}
			}
		}
	}

	return total, nil
}

// CheckHealth checks Redis health
func (rm *RedisMonitor) CheckHealth(ctx context.Context) error {
	return rm.client.Ping(ctx).Err()
}

// GetSlowLog returns slow log entries
func (rm *RedisMonitor) GetSlowLog(ctx context.Context, count int64) ([]redis.SlowLog, error) {
	return rm.client.SlowLogGet(ctx, count).Result()
}

// GetMemoryStats returns detailed memory statistics
func (rm *RedisMonitor) GetMemoryStats(ctx context.Context) (map[string]string, error) {
	info, err := rm.client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]string)
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			stats[parts[0]] = parts[1]
		}
	}
	return stats, nil
}

// GetClientList returns list of connected clients
func (rm *RedisMonitor) GetClientList(ctx context.Context) (string, error) {
	return rm.client.ClientList(ctx).Result()
}

// GetConfig returns Redis configuration
func (rm *RedisMonitor) GetConfig(ctx context.Context, parameter string) (map[string]string, error) {
	result, err := rm.client.ConfigGet(ctx, parameter).Result()
	if err != nil {
		return nil, err
	}

	config := make(map[string]string)
	for k, v := range result {
		config[k] = v
	}

	return config, nil
}

// MonitorContinuously monitors Redis continuously
func (rm *RedisMonitor) MonitorContinuously(ctx context.Context, interval time.Duration, callback func(*RedisMetrics)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := rm.CollectMetrics(ctx)
			if err != nil {
				rm.logger.Error("Failed to collect Redis metrics", zap.Error(err))
				continue
			}

			callback(metrics)

			// Log warnings
			rm.checkThresholds(metrics)
		}
	}
}

// checkThresholds checks if metrics exceed thresholds
func (rm *RedisMonitor) checkThresholds(metrics *RedisMetrics) {
	// Check hit rate
	if metrics.HitRate < 80 {
		rm.logger.Warn("Low Redis hit rate",
			zap.Float64("hit_rate", metrics.HitRate),
			zap.Int64("hits", metrics.Hits),
			zap.Int64("misses", metrics.Misses),
		)
	}

	// Check memory fragmentation
	if metrics.MemoryFragRatio > 1.5 {
		rm.logger.Warn("High memory fragmentation",
			zap.Float64("fragmentation_ratio", metrics.MemoryFragRatio),
		)
	}

	// Check evicted keys
	if metrics.EvictedKeys > 0 {
		rm.logger.Warn("Keys being evicted",
			zap.Int64("evicted_keys", metrics.EvictedKeys),
		)
	}

	// Check connected clients
	if metrics.MaxClients > 0 && metrics.ConnectedClients > metrics.MaxClients*80/100 {
		rm.logger.Warn("High number of connected clients",
			zap.Int64("connected_clients", metrics.ConnectedClients),
			zap.Int64("max_clients", metrics.MaxClients),
		)
	}
}

// FlushStats resets Redis statistics
func (rm *RedisMonitor) FlushStats(ctx context.Context) error {
	return rm.client.ConfigResetStat(ctx).Err()
}

// GetLatencyHistory returns latency history
func (rm *RedisMonitor) GetLatencyHistory(ctx context.Context, event string) (interface{}, error) {
	return nil, fmt.Errorf("LatencyHistory not fully supported in go-redis v9 direct api")
}
