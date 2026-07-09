# Connection Pooling Optimization Guide

## Overview

This guide covers the optimized connection pooling implementation for both PostgreSQL and Redis in the tariff pricing system.

**Requirement**: 27.7 - Connection Pooling optimization

## Table of Contents

1. [Database Connection Pooling](#database-connection-pooling)
2. [Redis Connection Pooling](#redis-connection-pooling)
3. [Configuration Guidelines](#configuration-guidelines)
4. [Monitoring and Tuning](#monitoring-and-tuning)
5. [Best Practices](#best-practices)

---

## Database Connection Pooling

### Architecture

The database connection pool optimizer (`internal/database/pool_optimizer.go`) provides environment-aware configuration:

```
┌─────────────────────────────────────────────────────────┐
│                  Pool Optimizer                          │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ Production   │  │   Staging    │  │ Development  │ │
│  │   Config     │  │    Config    │  │    Config    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  • CPU-aware sizing                                     │
│  • Load-based scaling                                   │
│  • Automatic tuning                                     │
└─────────────────────────────────────────────────────────┘
```

### Configuration Formula

**Production MaxConns** = `(CPU cores × 2) + effective_spindle_count`

Where:
- `CPU cores`: Number of CPU cores available
- `effective_spindle_count`: ~15 for cloud databases (AWS RDS, etc.)

**Load Adjustments**:
- Low load (< 1000 req/min): Base formula
- Medium load (1000-5000 req/min): Base × 2
- High load (> 5000 req/min): Base × 3
- Maximum cap: 200 connections

### Environment Configurations

#### Production
```go
MaxConns:          50-200 (load-dependent)
MinConns:          MaxConns / 3
MaxConnLifetime:   30 minutes
MaxConnIdleTime:   5 minutes
HealthCheckPeriod: 1 minute
ConnectTimeout:    10 seconds
AcquireTimeout:    5 seconds
```

#### Staging
```go
MaxConns:          60% of production
MinConns:          MaxConns / 4
MaxConnLifetime:   20 minutes
MaxConnIdleTime:   3 minutes
HealthCheckPeriod: 2 minutes
ConnectTimeout:    8 seconds
AcquireTimeout:    4 seconds
```

#### Development
```go
MaxConns:          5-20 (CPU-dependent)
MinConns:          2
MaxConnLifetime:   10 minutes
MaxConnIdleTime:   2 minutes
HealthCheckPeriod: 3 minutes
ConnectTimeout:    5 seconds
AcquireTimeout:    3 seconds
```

### Usage Example

```go
import (
    "github.com/al-hudhud/backend/internal/database"
    "go.uber.org/zap"
)

// Create optimizer
logger := zap.NewProduction()
optimizer := database.NewPoolOptimizer(logger)

// Get optimized configuration
// env: "production", "staging", or "development"
// expectedLoad: requests per minute
config := optimizer.GetOptimizedConfig("production", 5000)

// Apply to pgxpool.Config
poolConfig, _ := pgxpool.ParseConfig(dsn)
optimizer.ApplyToPoolConfig(poolConfig, config)

// Create pool
pool, _ := pgxpool.NewWithConfig(context.Background(), poolConfig)

// Start health monitoring
go optimizer.MonitorPoolHealth(context.Background(), pool, 1*time.Minute)
```

### Monitoring Metrics

The optimizer tracks:
- **Acquired connections**: Currently in use
- **Idle connections**: Available in pool
- **Total connections**: All connections
- **Utilization**: (Acquired / Total) × 100
- **Empty acquires**: Pool exhaustion events
- **Canceled acquires**: Timeout events

### Health Warnings

| Condition | Warning | Recommendation |
|-----------|---------|----------------|
| Utilization > 80% | High utilization | Increase MaxConns |
| Empty acquires > 100 | Pool exhaustion | Increase MaxConns or optimize queries |
| Canceled acquires > 50 | Acquire timeouts | Increase AcquireTimeout or MaxConns |
| Idle ratio > 70% | Too many idle | Decrease MaxConnIdleTime |

---

## Redis Connection Pooling

### Architecture

The Redis connection pool optimizer (`internal/redis/pool_optimizer.go`) provides similar environment-aware configuration:

```
┌─────────────────────────────────────────────────────────┐
│              Redis Pool Optimizer                        │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ Production   │  │   Staging    │  │ Development  │ │
│  │   Config     │  │    Config    │  │    Config    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  • CPU-aware sizing                                     │
│  • Load-based scaling                                   │
│  • Cache hit rate monitoring                            │
└─────────────────────────────────────────────────────────┘
```

### Configuration Formula

**Production PoolSize** = `CPU cores × 10`

**Load Adjustments**:
- Low load (< 1000 req/min): Base formula
- Medium load (1000-5000 req/min): Base × 2
- High load (> 5000 req/min): Base × 3
- Maximum cap: 500 connections

### Environment Configurations

#### Production
```go
PoolSize:           100-500 (load-dependent)
MinIdleConns:       PoolSize / 10
MaxIdleConns:       PoolSize / 2
ConnMaxLifetime:    30 minutes
IdleTimeout:        5 minutes
PoolTimeout:        4 seconds
MaxRetries:         3
DialTimeout:        5 seconds
ReadTimeout:        3 seconds
WriteTimeout:       3 seconds
```

#### Staging
```go
PoolSize:           60% of production
MinIdleConns:       PoolSize / 10
MaxIdleConns:       PoolSize / 2
ConnMaxLifetime:    20 minutes
IdleTimeout:        4 minutes
PoolTimeout:        3 seconds
MaxRetries:         3
```

#### Development
```go
PoolSize:           10-50 (CPU-dependent)
MinIdleConns:       5
MaxIdleConns:       PoolSize / 2
ConnMaxLifetime:    10 minutes
IdleTimeout:        3 minutes
PoolTimeout:        2 seconds
MaxRetries:         2
```

### Usage Example

```go
import (
    "github.com/al-hudhud/backend/internal/redis"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// Create optimizer
logger := zap.NewProduction()
optimizer := redis.NewPoolOptimizer(logger)

// Get optimized configuration
config := optimizer.GetOptimizedConfig("production", 5000)

// Create Redis options
opts := &redis.Options{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
}

// Apply optimizations
optimizer.ApplyToRedisOptions(opts, config)

// Create client
client := redis.NewClient(opts)

// Start health monitoring
go optimizer.MonitorPoolHealth(context.Background(), client, 1*time.Minute)
```

### Monitoring Metrics

The optimizer tracks:
- **Total connections**: All connections in pool
- **Idle connections**: Available connections
- **Active connections**: Currently in use
- **Stale connections**: Connections that should be closed
- **Hits/Misses**: Cache performance
- **Hit rate**: (Hits / (Hits + Misses)) × 100
- **Timeouts**: Pool exhaustion events

### Health Warnings

| Condition | Warning | Recommendation |
|-----------|---------|----------------|
| Utilization > 85% | High utilization | Increase PoolSize |
| Stale conns > 25% | Too many stale | Decrease IdleTimeout |
| Timeouts > 100 | Pool exhaustion | Increase PoolSize or PoolTimeout |
| Hit rate < 70% | Low cache efficiency | Review cache strategy and TTL |
| Idle ratio > 80% | Too many idle | Decrease MinIdleConns |

---

## Configuration Guidelines

### Choosing the Right Pool Size

#### Database Connections

**Rule of thumb**: Start with `(CPU cores × 2) + 15` for production

**Factors to consider**:
1. **Query complexity**: Complex queries need fewer connections
2. **Transaction duration**: Long transactions need more connections
3. **Concurrent users**: More users need more connections
4. **Database capacity**: Check database max_connections limit

**Example calculations**:
```
4 CPU cores, low load:
MaxConns = (4 × 2) + 15 = 23

8 CPU cores, high load:
MaxConns = ((8 × 2) + 15) × 3 = 93

16 CPU cores, high load (capped):
MaxConns = min(((16 × 2) + 15) × 3, 200) = 141
```

#### Redis Connections

**Rule of thumb**: Start with `CPU cores × 10` for production

**Factors to consider**:
1. **Operation types**: Simple GET/SET vs complex operations
2. **Pipeline usage**: Pipelining reduces connection needs
3. **Pub/Sub**: Separate connections for pub/sub
4. **Cluster mode**: More connections for cluster

**Example calculations**:
```
4 CPU cores, low load:
PoolSize = 4 × 10 = 40

8 CPU cores, high load:
PoolSize = (8 × 10) × 3 = 240

16 CPU cores, high load (capped):
PoolSize = min((16 × 10) × 3, 500) = 480
```

### Connection Lifetime Settings

#### MaxConnLifetime / ConnMaxLifetime

**Purpose**: Rotate connections to prevent stale connections

**Recommendations**:
- Production: 30 minutes
- Staging: 20 minutes
- Development: 10 minutes

**Why rotate?**:
- Prevents connection leaks
- Distributes load across database replicas
- Recovers from network issues

#### MaxConnIdleTime / IdleTimeout

**Purpose**: Close idle connections to save resources

**Recommendations**:
- Production: 5 minutes
- Staging: 3-4 minutes
- Development: 2-3 minutes

**Tuning**:
- Too short: Frequent reconnections (overhead)
- Too long: Wasted resources

### MinConns / MinIdleConns

**Purpose**: Keep minimum connections ready

**Recommendations**:
- Database: 25-33% of MaxConns
- Redis: 10% of PoolSize

**Benefits**:
- Faster response for first requests
- Reduced connection establishment overhead
- Better performance during traffic spikes

---

## Monitoring and Tuning

### Key Metrics to Monitor

#### Database Pool
```go
stats := pool.Stat()

// Monitor these metrics
acquiredConns := stats.AcquiredConns()
idleConns := stats.IdleConns()
totalConns := stats.TotalConns()
emptyAcquires := stats.EmptyAcquireCount()
canceledAcquires := stats.CanceledAcquireCount()

// Calculate utilization
utilization := float64(acquiredConns) / float64(stats.MaxConns()) * 100
```

#### Redis Pool
```go
stats := client.PoolStats()

// Monitor these metrics
totalConns := stats.TotalConns
idleConns := stats.IdleConns
staleConns := stats.StaleConns
hits := stats.Hits
misses := stats.Misses
timeouts := stats.Timeouts

// Calculate hit rate
hitRate := float64(hits) / float64(hits + misses) * 100
```

### Getting Recommendations

Both optimizers provide automatic recommendations:

```go
// Database
recommendations := optimizer.GetRecommendations(pool.Stat(), currentConfig)
for _, rec := range recommendations {
    log.Info(rec)
}

// Redis
recommendations := optimizer.GetRecommendations(client.PoolStats(), currentConfig)
for _, rec := range recommendations {
    log.Info(rec)
}
```

### Tuning Process

1. **Start with defaults**: Use optimizer's environment-based config
2. **Monitor for 24 hours**: Collect metrics across different load patterns
3. **Analyze patterns**: Look for warnings and bottlenecks
4. **Apply recommendations**: Adjust based on optimizer suggestions
5. **Test changes**: Apply in staging first
6. **Iterate**: Repeat until optimal

### Common Tuning Scenarios

#### Scenario 1: High Database Utilization

**Symptoms**:
- Utilization > 90%
- Many empty acquires
- Slow response times

**Solutions**:
1. Increase MaxConns by 50%
2. Optimize slow queries
3. Add read replicas
4. Implement connection pooling at application level

#### Scenario 2: Low Redis Hit Rate

**Symptoms**:
- Hit rate < 60%
- Many cache misses
- High database load

**Solutions**:
1. Increase cache TTL
2. Pre-warm cache for common queries
3. Review cache key strategy
4. Implement cache-aside pattern

#### Scenario 3: Connection Timeouts

**Symptoms**:
- Many canceled acquires (database)
- Many timeouts (Redis)
- Intermittent errors

**Solutions**:
1. Increase AcquireTimeout / PoolTimeout
2. Increase MaxConns / PoolSize
3. Check network latency
4. Optimize query performance

---

## Best Practices

### 1. Environment-Specific Configuration

Always use different configurations for different environments:

```go
env := os.Getenv("APP_ENV") // "production", "staging", "development"
config := optimizer.GetOptimizedConfig(env, expectedLoad)
```

### 2. Monitor Continuously

Set up continuous monitoring:

```go
// Start monitoring goroutines
go dbOptimizer.MonitorPoolHealth(ctx, dbPool, 1*time.Minute)
go redisOptimizer.MonitorPoolHealth(ctx, redisClient, 1*time.Minute)
```

### 3. Use Health Checks

Implement health check endpoints:

```go
func healthCheck(w http.ResponseWriter, r *http.Request) {
    dbStats := db.GetStats()
    redisStats := redis.GetStats()
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "database": dbStats,
        "redis":    redisStats,
    })
}
```

### 4. Set Appropriate Timeouts

Always set timeouts for operations:

```go
// Database
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Redis
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
```

### 5. Handle Pool Exhaustion Gracefully

```go
conn, err := pool.Acquire(ctx)
if err != nil {
    if errors.Is(err, pgxpool.ErrClosedPool) {
        // Pool is closed
        return ErrServiceUnavailable
    }
    if errors.Is(err, context.DeadlineExceeded) {
        // Timeout acquiring connection
        return ErrPoolExhausted
    }
    return err
}
defer conn.Release()
```

### 6. Use Connection Pooling at All Layers

```
┌─────────────────────────────────────────┐
│         Application Layer               │
│  (HTTP handlers, business logic)        │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│      Connection Pool Layer              │
│  (Database pool, Redis pool)            │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│         Database / Redis                │
└─────────────────────────────────────────┘
```

### 7. Implement Circuit Breakers

Both database and Redis clients include circuit breakers:

```go
// Database with circuit breaker
err := db.ExecuteWithCircuitBreaker(ctx, func() error {
    return db.Pool.QueryRow(ctx, query).Scan(&result)
})

// Redis with circuit breaker
err := redis.ExecuteWithCircuitBreaker(ctx, func() error {
    return redis.Client.Set(ctx, key, value, ttl).Err()
})
```

### 8. Log Pool Statistics

Regularly log pool statistics for analysis:

```go
ticker := time.NewTicker(5 * time.Minute)
for range ticker.C {
    logger.Info("Database pool stats", zap.Any("stats", db.GetStats()))
    logger.Info("Redis pool stats", zap.Any("stats", redis.GetStats()))
}
```

---

## Testing

### Running Tests

```bash
# Database pool optimizer tests
cd services/backend
go test -v ./internal/database -run TestPoolOptimizer

# Redis pool optimizer tests
go test -v ./internal/redis -run TestRedisPoolOptimizer

# Benchmarks
go test -bench=. ./internal/database
go test -bench=. ./internal/redis
```

### Load Testing

Use the provided load testing scripts to validate pool configuration:

```bash
# Run load test
cd performance-tests
./run-load-test.ps1

# Monitor pool metrics during test
# Check logs for warnings and recommendations
```

---

## Troubleshooting

### Problem: "too many connections" Error

**Cause**: MaxConns exceeds database limit

**Solution**:
1. Check database max_connections: `SHOW max_connections;`
2. Reduce MaxConns in application
3. Increase database max_connections if possible

### Problem: Slow Query Performance

**Cause**: Pool exhaustion or slow queries

**Solution**:
1. Check pool utilization
2. Analyze slow query log
3. Add indexes
4. Optimize queries
5. Increase MaxConns if needed

### Problem: High Memory Usage

**Cause**: Too many idle connections

**Solution**:
1. Decrease MaxConnIdleTime / IdleTimeout
2. Decrease MinConns / MinIdleConns
3. Decrease MaxConns / PoolSize

### Problem: Connection Timeouts

**Cause**: Network issues or pool exhaustion

**Solution**:
1. Check network latency
2. Increase timeouts
3. Increase pool size
4. Check for connection leaks

---

## References

- [PostgreSQL Connection Pooling Best Practices](https://wiki.postgresql.org/wiki/Number_Of_Database_Connections)
- [pgx Pool Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool)
- [Redis Connection Pooling](https://redis.io/docs/manual/patterns/connection-pooling/)
- [go-redis Documentation](https://redis.uptrace.dev/)

---

## Summary

Connection pooling optimization is critical for:
- **Performance**: Reduce connection overhead
- **Scalability**: Handle more concurrent requests
- **Reliability**: Prevent resource exhaustion
- **Cost**: Optimize resource usage

Use the provided optimizers to automatically configure pools based on your environment and load patterns. Monitor continuously and tune based on recommendations.
