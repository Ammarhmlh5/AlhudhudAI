# Connection Pooling Optimization - Quick Start

## Overview

Quick guide to implement optimized connection pooling for Database and Redis.

**Task**: 35.3 Connection Pooling  
**Requirement**: 27.7

---

## Quick Implementation

### 1. Database Connection Pool

```go
package main

import (
    "context"
    "github.com/al-hudhud/backend/internal/config"
    "github.com/al-hudhud/backend/internal/database"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
)

func main() {
    // Load config
    cfg, _ := config.Load()
    logger, _ := zap.NewProduction()
    
    // Create pool optimizer
    optimizer := database.NewPoolOptimizer(logger)
    
    // Get optimized config (auto-detects CPU, adjusts for load)
    poolConfig := optimizer.GetOptimizedConfig(
        cfg.Server.AppEnv,  // "production", "staging", "development"
        5000,                // expected requests per minute
    )
    
    // Parse DSN and apply optimizations
    pgxConfig, _ := pgxpool.ParseConfig(cfg.GetDatabaseDSN())
    optimizer.ApplyToPoolConfig(pgxConfig, poolConfig)
    
    // Create pool
    pool, _ := pgxpool.NewWithConfig(context.Background(), pgxConfig)
    defer pool.Close()
    
    // Start health monitoring (optional but recommended)
    ctx := context.Background()
    go optimizer.MonitorPoolHealth(ctx, pool, 1*time.Minute)
    
    // Use pool normally
    conn, _ := pool.Acquire(ctx)
    defer conn.Release()
}
```

### 2. Redis Connection Pool

```go
package main

import (
    "context"
    "github.com/al-hudhud/backend/internal/config"
    "github.com/al-hudhud/backend/internal/redis"
    goredis "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

func main() {
    // Load config
    cfg, _ := config.Load()
    logger, _ := zap.NewProduction()
    
    // Create pool optimizer
    optimizer := redis.NewPoolOptimizer(logger)
    
    // Get optimized config
    poolConfig := optimizer.GetOptimizedConfig(
        cfg.Server.AppEnv,  // "production", "staging", "development"
        5000,                // expected requests per minute
    )
    
    // Create Redis options and apply optimizations
    opts := &goredis.Options{
        Addr:     cfg.GetRedisAddr(),
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    }
    optimizer.ApplyToRedisOptions(opts, poolConfig)
    
    // Create client
    client := goredis.NewClient(opts)
    defer client.Close()
    
    // Start health monitoring (optional but recommended)
    ctx := context.Background()
    go optimizer.MonitorPoolHealth(ctx, client, 1*time.Minute)
    
    // Use client normally
    client.Set(ctx, "key", "value", 0)
}
```

---

## Configuration by Environment

### Production (High Performance)
- **Database**: 50-200 connections (load-dependent)
- **Redis**: 100-500 connections (load-dependent)
- **Lifetime**: 30 minutes
- **Idle timeout**: 5 minutes

### Staging (Medium Performance)
- **Database**: 60% of production
- **Redis**: 60% of production
- **Lifetime**: 20 minutes
- **Idle timeout**: 3-4 minutes

### Development (Minimal Resources)
- **Database**: 5-20 connections
- **Redis**: 10-50 connections
- **Lifetime**: 10 minutes
- **Idle timeout**: 2-3 minutes

---

## Monitoring

### Get Current Stats

```go
// Database
stats := pool.Stat()
logger.Info("Database pool stats",
    zap.Int32("acquired", stats.AcquiredConns()),
    zap.Int32("idle", stats.IdleConns()),
    zap.Int32("total", stats.TotalConns()),
)

// Redis
stats := client.PoolStats()
logger.Info("Redis pool stats",
    zap.Uint32("total", stats.TotalConns),
    zap.Uint32("idle", stats.IdleConns),
    zap.Uint64("hits", stats.Hits),
    zap.Uint64("misses", stats.Misses),
)
```

### Get Recommendations

```go
// Database
recommendations := optimizer.GetRecommendations(pool.Stat(), currentConfig)
for _, rec := range recommendations {
    logger.Warn("Database pool recommendation", zap.String("recommendation", rec))
}

// Redis
recommendations := optimizer.GetRecommendations(client.PoolStats(), currentConfig)
for _, rec := range recommendations {
    logger.Warn("Redis pool recommendation", zap.String("recommendation", rec))
}
```

---

## Testing

### Run Unit Tests

```bash
# Database pool optimizer tests
go test -v ./internal/database -run TestPoolOptimizer

# Redis pool optimizer tests
go test -v ./internal/redis -run TestRedisPoolOptimizer
```

### Run Benchmarks

```bash
# Database benchmarks
go test -bench=BenchmarkPoolOptimizer ./internal/database

# Redis benchmarks
go test -bench=BenchmarkRedisPoolOptimizer ./internal/redis
```

### Load Testing

```bash
# Use provided load test script
cd performance-tests
./run-load-test.ps1

# Monitor pool metrics during test
# Check logs for warnings
```

---

## Common Issues

### Issue: Pool Exhaustion

**Symptoms**: "connection pool exhausted" errors

**Solution**:
```go
// Increase MaxConns
config := optimizer.GetOptimizedConfig(env, expectedLoad * 2)
```

### Issue: High Memory Usage

**Symptoms**: High memory consumption

**Solution**:
```go
// Decrease idle timeout
config.MaxConnIdleTime = 2 * time.Minute  // Database
config.IdleTimeout = 2 * time.Minute      // Redis
```

### Issue: Slow Queries

**Symptoms**: Slow response times

**Solution**:
1. Check pool utilization
2. Optimize queries
3. Add indexes
4. Increase pool size if needed

---

## Environment Variables

Add these to your `.env` file:

```bash
# Database Pool Settings
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME=1800  # 30 minutes in seconds
DB_CONN_MAX_IDLE_TIME=300   # 5 minutes in seconds

# Redis Pool Settings (handled by optimizer)
# No additional env vars needed - optimizer auto-configures
```

---

## Next Steps

1. ✅ Implement optimized pooling in your application
2. ✅ Start health monitoring
3. ✅ Run load tests
4. ✅ Monitor metrics for 24 hours
5. ✅ Apply recommendations
6. ✅ Tune based on actual load patterns

---

## Full Documentation

See [CONNECTION_POOLING_GUIDE.md](./CONNECTION_POOLING_GUIDE.md) for:
- Detailed configuration guidelines
- Tuning strategies
- Troubleshooting guide
- Best practices
- Advanced scenarios

---

## Support

For issues or questions:
1. Check the full guide: `CONNECTION_POOLING_GUIDE.md`
2. Review test files for examples
3. Check logs for optimizer recommendations
