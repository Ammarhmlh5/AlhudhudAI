package redis

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PoolOptimizer provides optimized Redis connection pool configuration
// Requirement 27.7: Connection Pooling optimization for Redis
type PoolOptimizer struct {
	logger *zap.Logger
}

// NewPoolOptimizer creates a new Redis pool optimizer
func NewPoolOptimizer(logger *zap.Logger) *PoolOptimizer {
	return &PoolOptimizer{
		logger: logger,
	}
}

// RedisPoolConfig holds optimized Redis pool configuration
type RedisPoolConfig struct {
	PoolSize           int
	MinIdleConns       int
	MaxIdleConns       int
	ConnMaxIdleTime    time.Duration
	ConnMaxLifetime    time.Duration
	PoolTimeout        time.Duration
	IdleCheckFrequency time.Duration
	MaxRetries         int
	MinRetryBackoff    time.Duration
	MaxRetryBackoff    time.Duration
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
}

// GetOptimizedConfig returns optimized Redis pool configuration
// Requirement 27.7: تحسين إعدادات Redis Connection Pool
func (po *PoolOptimizer) GetOptimizedConfig(env string, expectedLoad int) *RedisPoolConfig {
	numCPU := runtime.NumCPU()
	
	var config *RedisPoolConfig
	
	switch env {
	case "production":
		config = po.getProductionConfig(numCPU, expectedLoad)
	case "staging":
		config = po.getStagingConfig(numCPU, expectedLoad)
	case "development":
		config = po.getDevelopmentConfig(numCPU)
	default:
		config = po.getDevelopmentConfig(numCPU)
	}
	
	po.logger.Info("Redis pool configuration optimized",
		zap.String("environment", env),
		zap.Int("cpu_count", numCPU),
		zap.Int("expected_load", expectedLoad),
		zap.Int("pool_size", config.PoolSize),
		zap.Int("min_idle_conns", config.MinIdleConns),
		zap.Duration("conn_max_lifetime", config.ConnMaxLifetime),
		// Redundant IdleTimeout removed
	)
	
	return config
}

// getProductionConfig returns production-optimized Redis configuration
// Redis pool sizing: 10 * CPU cores is a good starting point
func (po *PoolOptimizer) getProductionConfig(numCPU, expectedLoad int) *RedisPoolConfig {
	// Base pool size: 10 * CPU cores
	basePoolSize := numCPU * 10
	
	// Adjust based on expected load
	var poolSize int
	if expectedLoad < 1000 {
		poolSize = basePoolSize
	} else if expectedLoad < 5000 {
		poolSize = basePoolSize * 2
	} else {
		poolSize = basePoolSize * 3
	}
	
	// Cap at reasonable maximum (Redis can handle many connections)
	if poolSize > 500 {
		poolSize = 500
	}
	
	// Minimum should be at least 10% of pool size
	minIdleConns := poolSize / 10
	if minIdleConns < 10 {
		minIdleConns = 10
	}
	
	// Maximum idle should be 50% of pool size
	maxIdleConns := poolSize / 2
	
	return &RedisPoolConfig{
		PoolSize:           poolSize,
		MinIdleConns:       minIdleConns,
		MaxIdleConns:       maxIdleConns,
		ConnMaxIdleTime:    10 * time.Minute, // Close idle connections after 10 minutes
		ConnMaxLifetime:    30 * time.Minute, // Rotate connections every 30 minutes
		PoolTimeout:        4 * time.Second,  // Wait up to 4 seconds for connection from pool
		IdleCheckFrequency: 1 * time.Minute,  // Check for idle connections every minute
		MaxRetries:         3,                // Retry failed operations 3 times
		MinRetryBackoff:    8 * time.Millisecond,
		MaxRetryBackoff:    512 * time.Millisecond,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
	}
}

// getStagingConfig returns staging-optimized Redis configuration
func (po *PoolOptimizer) getStagingConfig(numCPU, expectedLoad int) *RedisPoolConfig {
	// Staging uses 60% of production capacity
	prodConfig := po.getProductionConfig(numCPU, expectedLoad)
	
	poolSize := int(float64(prodConfig.PoolSize) * 0.6)
	if poolSize < 20 {
		poolSize = 20
	}
	
	minIdleConns := poolSize / 10
	if minIdleConns < 5 {
		minIdleConns = 5
	}
	
	maxIdleConns := poolSize / 2
	
	return &RedisPoolConfig{
		PoolSize:           poolSize,
		MinIdleConns:       minIdleConns,
		MaxIdleConns:       maxIdleConns,
		ConnMaxIdleTime:    8 * time.Minute,
		ConnMaxLifetime:    20 * time.Minute,
		PoolTimeout:        3 * time.Second,
		IdleCheckFrequency: 2 * time.Minute,
		MaxRetries:         3,
		MinRetryBackoff:    8 * time.Millisecond,
		MaxRetryBackoff:    512 * time.Millisecond,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
	}
}

// getDevelopmentConfig returns development-optimized Redis configuration
func (po *PoolOptimizer) getDevelopmentConfig(numCPU int) *RedisPoolConfig {
	// Development uses minimal resources
	poolSize := numCPU * 5
	if poolSize < 10 {
		poolSize = 10
	}
	if poolSize > 50 {
		poolSize = 50
	}
	
	return &RedisPoolConfig{
		PoolSize:           poolSize,
		MinIdleConns:       5,
		MaxIdleConns:       poolSize / 2,
		ConnMaxIdleTime:    5 * time.Minute,
		ConnMaxLifetime:    10 * time.Minute,
		PoolTimeout:        2 * time.Second,
		IdleCheckFrequency: 3 * time.Minute,
		MaxRetries:         2,
		MinRetryBackoff:    8 * time.Millisecond,
		MaxRetryBackoff:    256 * time.Millisecond,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
	}
}

// ApplyToRedisOptions applies optimized settings to redis.Options
func (po *PoolOptimizer) ApplyToRedisOptions(opts *redis.Options, optimized *RedisPoolConfig) {
	opts.PoolSize = optimized.PoolSize
	opts.MinIdleConns = optimized.MinIdleConns
	opts.MaxIdleConns = optimized.MaxIdleConns
	opts.ConnMaxIdleTime = optimized.ConnMaxIdleTime
	opts.ConnMaxLifetime = optimized.ConnMaxLifetime
	opts.PoolTimeout = optimized.PoolTimeout
	opts.MaxRetries = optimized.MaxRetries
	opts.MinRetryBackoff = optimized.MinRetryBackoff
	opts.MaxRetryBackoff = optimized.MaxRetryBackoff
	opts.DialTimeout = optimized.DialTimeout
	opts.ReadTimeout = optimized.ReadTimeout
	opts.WriteTimeout = optimized.WriteTimeout
}

// MonitorPoolHealth monitors Redis pool health and logs warnings
func (po *PoolOptimizer) MonitorPoolHealth(ctx context.Context, client *redis.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := client.PoolStats()
			
			// Calculate metrics
			totalConns := stats.TotalConns
			idleConns := stats.IdleConns
			staleConns := stats.StaleConns
			
			var utilizationPercent float64
			if totalConns > 0 {
				activeConns := totalConns - idleConns
				utilizationPercent = float64(activeConns) / float64(totalConns) * 100
			}
			
			var hitRate float64
			totalRequests := stats.Hits + stats.Misses
			if totalRequests > 0 {
				hitRate = float64(stats.Hits) / float64(totalRequests) * 100
			}
			
			// Log current state
			po.logger.Debug("Redis pool health check",
				zap.Uint32("total_conns", totalConns),
				zap.Uint32("idle_conns", idleConns),
				zap.Uint32("stale_conns", staleConns),
				zap.Float64("utilization_percent", utilizationPercent),
				zap.Uint32("hits", stats.Hits),
				zap.Uint32("misses", stats.Misses),
				zap.Float64("hit_rate_percent", hitRate),
				zap.Uint32("timeouts", stats.Timeouts),
			)
			
			// Warn if utilization is high
			if utilizationPercent > 85 {
				po.logger.Warn("Redis pool utilization high",
					zap.Float64("utilization_percent", utilizationPercent),
					zap.Uint32("total_conns", totalConns),
					zap.Uint32("idle_conns", idleConns),
				)
			}
			
			// Warn if many stale connections
			if staleConns > totalConns/4 {
				po.logger.Warn("High number of stale Redis connections",
					zap.Uint32("stale_conns", staleConns),
					zap.Uint32("total_conns", totalConns),
					zap.String("recommendation", "Consider decreasing ConnMaxIdleTime"),
				)
			}
			
			// Warn if many timeouts
			if stats.Timeouts > 100 {
				po.logger.Warn("Redis pool timeouts detected",
					zap.Uint32("timeouts", stats.Timeouts),
					zap.String("recommendation", "Consider increasing PoolSize or PoolTimeout"),
				)
			}
			
			// Warn if low hit rate (cache inefficiency)
			if hitRate < 70 && totalRequests > 1000 {
				po.logger.Warn("Low Redis cache hit rate",
					zap.Float64("hit_rate_percent", hitRate),
					zap.Uint32("hits", stats.Hits),
					zap.Uint32("misses", stats.Misses),
					zap.String("recommendation", "Review cache strategy and TTL settings"),
				)
			}
		}
	}
}

// GetRecommendations analyzes Redis pool stats and provides recommendations
func (po *PoolOptimizer) GetRecommendations(stats *redis.PoolStats, currentConfig *RedisPoolConfig) []string {
	recommendations := make([]string, 0)
	
	totalConns := stats.TotalConns
	idleConns := stats.IdleConns
	
	if totalConns == 0 {
		return recommendations
	}
	
	activeConns := totalConns - idleConns
	utilization := float64(activeConns) / float64(totalConns) * 100
	
	// Check utilization
	if utilization > 90 {
		recommendations = append(recommendations,
			fmt.Sprintf("Redis pool utilization is very high (%.1f%%). Consider increasing PoolSize from %d to %d",
				utilization, currentConfig.PoolSize, currentConfig.PoolSize*2))
	} else if utilization < 15 && totalConns > 20 {
		recommendations = append(recommendations,
			fmt.Sprintf("Redis pool utilization is low (%.1f%%). Consider decreasing PoolSize from %d to %d to save resources",
				utilization, currentConfig.PoolSize, currentConfig.PoolSize/2))
	}
	
	// Check stale connections
	if stats.StaleConns > totalConns/3 {
		recommendations = append(recommendations,
			fmt.Sprintf("High stale connection ratio (%d/%d). Consider decreasing ConnMaxIdleTime from %v to %v",
				stats.StaleConns, totalConns, currentConfig.ConnMaxIdleTime, currentConfig.ConnMaxIdleTime/2))
	}
	
	// Check timeouts
	if stats.Timeouts > 500 {
		recommendations = append(recommendations,
			fmt.Sprintf("High timeout count (%d). Pool is frequently exhausted. Increase PoolSize or PoolTimeout", stats.Timeouts))
	}
	
	// Check cache hit rate
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 1000 {
		hitRate := float64(stats.Hits) / float64(totalRequests) * 100
		if hitRate < 60 {
			recommendations = append(recommendations,
				fmt.Sprintf("Low cache hit rate (%.1f%%). Review cache strategy, increase TTL, or pre-warm cache", hitRate))
		}
	}
	
	// Check idle connections
	idleRatio := float64(idleConns) / float64(totalConns) * 100
	if idleRatio > 80 && totalConns > uint32(currentConfig.MinIdleConns) {
		recommendations = append(recommendations,
			fmt.Sprintf("High idle connection ratio (%.1f%%). Consider decreasing MinIdleConns from %d to %d",
				idleRatio, currentConfig.MinIdleConns, currentConfig.MinIdleConns/2))
	}
	
	return recommendations
}

// GetPoolMetrics returns detailed pool metrics for monitoring
func (po *PoolOptimizer) GetPoolMetrics(stats *redis.PoolStats) map[string]interface{} {
	totalConns := stats.TotalConns
	idleConns := stats.IdleConns
	
	var utilizationPercent float64
	if totalConns > 0 {
		activeConns := totalConns - idleConns
		utilizationPercent = float64(activeConns) / float64(totalConns) * 100
	}
	
	var hitRate float64
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		hitRate = float64(stats.Hits) / float64(totalRequests) * 100
	}
	
	return map[string]interface{}{
		"total_conns":         totalConns,
		"idle_conns":          idleConns,
		"active_conns":        totalConns - idleConns,
		"stale_conns":         stats.StaleConns,
		"utilization_percent": utilizationPercent,
		"hits":                uint64(stats.Hits),
		"misses":              uint64(stats.Misses),
		"hit_rate_percent":    hitRate,
		"timeouts":            uint64(stats.Timeouts),
	}
}
