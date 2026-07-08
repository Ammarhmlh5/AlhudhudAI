package database

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PoolOptimizer provides optimized connection pool configuration
// Requirement 27.7: Connection Pooling optimization
type PoolOptimizer struct {
	logger *zap.Logger
}

// NewPoolOptimizer creates a new pool optimizer
func NewPoolOptimizer(logger *zap.Logger) *PoolOptimizer {
	return &PoolOptimizer{
		logger: logger,
	}
}

// PoolConfig holds optimized pool configuration
type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout  time.Duration
	AcquireTimeout  time.Duration
}

// GetOptimizedConfig returns optimized pool configuration based on environment
// Requirement 27.7: تحسين إعدادات Database Connection Pool
func (po *PoolOptimizer) GetOptimizedConfig(env string, expectedLoad int) *PoolConfig {
	numCPU := runtime.NumCPU()
	
	var config *PoolConfig
	
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
	
	po.logger.Info("Database pool configuration optimized",
		zap.String("environment", env),
		zap.Int("cpu_count", numCPU),
		zap.Int("expected_load", expectedLoad),
		zap.Int32("max_conns", config.MaxConns),
		zap.Int32("min_conns", config.MinConns),
		zap.Duration("max_conn_lifetime", config.MaxConnLifetime),
		zap.Duration("max_conn_idle_time", config.MaxConnIdleTime),
	)
	
	return config
}

// getProductionConfig returns production-optimized configuration
// Formula: MaxConns = (CPU cores * 2) + effective_spindle_count
// For cloud databases, effective_spindle_count ≈ 10-20
func (po *PoolOptimizer) getProductionConfig(numCPU, expectedLoad int) *PoolConfig {
	// Base calculation: (CPU * 2) + spindle_count
	baseMaxConns := (numCPU * 2) + 15
	
	// Adjust based on expected load
	// Low load: 0-1000 req/min
	// Medium load: 1000-5000 req/min
	// High load: 5000+ req/min
	var maxConns int32
	if expectedLoad < 1000 {
		maxConns = int32(baseMaxConns)
	} else if expectedLoad < 5000 {
		maxConns = int32(baseMaxConns * 2)
	} else {
		maxConns = int32(baseMaxConns * 3)
	}
	
	// Cap at reasonable maximum
	if maxConns > 200 {
		maxConns = 200
	}
	
	// MinConns should be 25-50% of MaxConns for production
	minConns := maxConns / 3
	if minConns < 5 {
		minConns = 5
	}
	
	return &PoolConfig{
		MaxConns:          maxConns,
		MinConns:          minConns,
		MaxConnLifetime:   30 * time.Minute, // Rotate connections every 30 minutes
		MaxConnIdleTime:   5 * time.Minute,  // Close idle connections after 5 minutes
		HealthCheckPeriod: 1 * time.Minute,  // Check health every minute
		ConnectTimeout:    10 * time.Second, // Timeout for new connections
		AcquireTimeout:    5 * time.Second,  // Timeout for acquiring from pool
	}
}

// getStagingConfig returns staging-optimized configuration
func (po *PoolOptimizer) getStagingConfig(numCPU, expectedLoad int) *PoolConfig {
	// Staging uses 60% of production capacity
	prodConfig := po.getProductionConfig(numCPU, expectedLoad)
	
	maxConns := int32(float64(prodConfig.MaxConns) * 0.6)
	if maxConns < 10 {
		maxConns = 10
	}
	
	minConns := maxConns / 4
	if minConns < 3 {
		minConns = 3
	}
	
	return &PoolConfig{
		MaxConns:          maxConns,
		MinConns:          minConns,
		MaxConnLifetime:   20 * time.Minute,
		MaxConnIdleTime:   3 * time.Minute,
		HealthCheckPeriod: 2 * time.Minute,
		ConnectTimeout:    8 * time.Second,
		AcquireTimeout:    4 * time.Second,
	}
}

// getDevelopmentConfig returns development-optimized configuration
func (po *PoolOptimizer) getDevelopmentConfig(numCPU int) *PoolConfig {
	// Development uses minimal resources
	maxConns := int32(numCPU * 2)
	if maxConns < 5 {
		maxConns = 5
	}
	if maxConns > 20 {
		maxConns = 20
	}
	
	return &PoolConfig{
		MaxConns:          maxConns,
		MinConns:          2,
		MaxConnLifetime:   10 * time.Minute,
		MaxConnIdleTime:   2 * time.Minute,
		HealthCheckPeriod: 3 * time.Minute,
		ConnectTimeout:    5 * time.Second,
		AcquireTimeout:    3 * time.Second,
	}
}

// ApplyToPoolConfig applies optimized settings to pgxpool.Config
func (po *PoolOptimizer) ApplyToPoolConfig(poolConfig *pgxpool.Config, optimized *PoolConfig) {
	poolConfig.MaxConns = optimized.MaxConns
	poolConfig.MinConns = optimized.MinConns
	poolConfig.MaxConnLifetime = optimized.MaxConnLifetime
	poolConfig.MaxConnIdleTime = optimized.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = optimized.HealthCheckPeriod
	
	// Set connection timeouts
	if poolConfig.ConnConfig != nil {
		poolConfig.ConnConfig.ConnectTimeout = optimized.ConnectTimeout
	}
}

// MonitorPoolHealth monitors pool health and logs warnings
func (po *PoolOptimizer) MonitorPoolHealth(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := pool.Stat()
			
			// Calculate utilization
			utilization := float64(stats.AcquiredConns()) / float64(stats.MaxConns()) * 100
			
			// Log current state
			po.logger.Debug("Database pool health check",
				zap.Int32("acquired_conns", stats.AcquiredConns()),
				zap.Int32("idle_conns", stats.IdleConns()),
				zap.Int32("total_conns", stats.TotalConns()),
				zap.Int32("max_conns", stats.MaxConns()),
				zap.Float64("utilization_percent", utilization),
				zap.Int64("empty_acquire", stats.EmptyAcquireCount()),
				zap.Int64("canceled_acquire", stats.CanceledAcquireCount()),
			)
			
			// Warn if utilization is high
			if utilization > 80 {
				po.logger.Warn("Database pool utilization high",
					zap.Float64("utilization_percent", utilization),
					zap.Int32("acquired_conns", stats.AcquiredConns()),
					zap.Int32("max_conns", stats.MaxConns()),
				)
			}
			
			// Warn if many empty acquires (pool exhaustion)
			if stats.EmptyAcquireCount() > 100 {
				po.logger.Warn("Database pool exhaustion detected",
					zap.Int64("empty_acquire_count", stats.EmptyAcquireCount()),
					zap.String("recommendation", "Consider increasing MaxConns"),
				)
			}
			
			// Warn if many canceled acquires (timeout issues)
			if stats.CanceledAcquireCount() > 50 {
				po.logger.Warn("Database pool acquire timeouts detected",
					zap.Int64("canceled_acquire_count", stats.CanceledAcquireCount()),
					zap.String("recommendation", "Consider increasing AcquireTimeout or MaxConns"),
				)
			}
		}
	}
}

// PoolStats defines the interface for database pool statistics to allow mocking in tests
type PoolStats interface {
	AcquiredConns() int32
	MaxConns() int32
	IdleConns() int32
	TotalConns() int32
	EmptyAcquireCount() int64
	CanceledAcquireCount() int64
}

// GetRecommendations analyzes pool stats and provides recommendations
func (po *PoolOptimizer) GetRecommendations(stats PoolStats, currentConfig *PoolConfig) []string {
	recommendations := make([]string, 0)
	
	if stats.MaxConns() == 0 {
		return recommendations
	}
	
	utilization := float64(stats.AcquiredConns()) / float64(stats.MaxConns()) * 100
	
	// Check utilization
	if utilization > 90 {
		recommendations = append(recommendations,
			fmt.Sprintf("Pool utilization is very high (%.1f%%). Consider increasing MaxConns from %d to %d",
				utilization, currentConfig.MaxConns, currentConfig.MaxConns*2))
	} else if utilization < 20 && stats.MaxConns() > 10 {
		recommendations = append(recommendations,
			fmt.Sprintf("Pool utilization is low (%.1f%%). Consider decreasing MaxConns from %d to %d to save resources",
				utilization, currentConfig.MaxConns, currentConfig.MaxConns/2))
	}
	
	// Check idle connections
	idleRatio := float64(stats.IdleConns()) / float64(stats.TotalConns()) * 100
	if idleRatio > 70 && stats.TotalConns() > int32(currentConfig.MinConns) {
		recommendations = append(recommendations,
			fmt.Sprintf("High idle connection ratio (%.1f%%). Consider decreasing MaxConnIdleTime from %v to %v",
				idleRatio, currentConfig.MaxConnIdleTime, currentConfig.MaxConnIdleTime/2))
	}
	
	// Check empty acquires
	if stats.EmptyAcquireCount() > 1000 {
		recommendations = append(recommendations,
			"High empty acquire count detected. Pool is frequently exhausted. Increase MaxConns or optimize query performance")
	}
	
	// Check canceled acquires
	if stats.CanceledAcquireCount() > 500 {
		recommendations = append(recommendations,
			"High canceled acquire count detected. Connections are timing out. Increase AcquireTimeout or MaxConns")
	}
	
	return recommendations
}
