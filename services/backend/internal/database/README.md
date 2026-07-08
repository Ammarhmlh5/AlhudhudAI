# Database Package

## Overview

This package provides database connection management with optimized connection pooling for PostgreSQL.

## Features

- **Dual Connection Support**: Both `pgxpool` (modern) and `sql.DB` (legacy compatibility)
- **Optimized Connection Pool**: Configured for high performance (5000+ messages/minute)
- **Health Monitoring**: Built-in health checks and statistics
- **Production Ready**: Follows PostgreSQL best practices

## Connection Pool Configuration

### Default Settings (Requirements 12.1-12.4)

| Setting | Value | Description |
|---------|-------|-------------|
| MaxOpenConns | 100 | Maximum number of open connections |
| MaxIdleConns | 10 | Number of idle connections to maintain |
| ConnMaxLifetime | 1 hour | Maximum lifetime of a connection |
| ConnMaxIdleTime | 10 minutes | Maximum idle time before closing |
| HealthCheckPeriod | 1 minute | Frequency of health checks |

### Environment Variables

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=hudhud_admin
DB_PASSWORD=hudhud_pass
DB_NAME=al_hudhud
DB_SSLMODE=disable

# Connection Pool Settings
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=3600    # seconds (1 hour)
DB_CONN_MAX_IDLE_TIME=600    # seconds (10 minutes)
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/al-hudhud/backend/internal/config"
    "github.com/al-hudhud/backend/internal/database"
    "go.uber.org/zap"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // Create database connection
    db, err := database.NewDatabase(cfg, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Use pgxpool for modern operations
    var count int
    err = db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM messages").Scan(&count)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use sql.DB for legacy compatibility
    rows, err := db.SqlDB.Query("SELECT id, content FROM messages LIMIT 10")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
}
```

### Health Check

```go
ctx := context.Background()
if err := db.HealthCheck(ctx); err != nil {
    log.Printf("Database health check failed: %v", err)
}
```

### Monitoring Statistics

```go
stats := db.GetStats()

// pgxpool statistics
pgxStats := stats["pgxpool"].(map[string]interface{})
fmt.Printf("Active connections: %d\n", pgxStats["acquired_conns"])
fmt.Printf("Idle connections: %d\n", pgxStats["idle_conns"])
fmt.Printf("Total connections: %d\n", pgxStats["total_conns"])

// sql.DB statistics
sqlStats := stats["sql_db"].(map[string]interface{})
fmt.Printf("Open connections: %d\n", sqlStats["open_conns"])
fmt.Printf("In use: %d\n", sqlStats["in_use"])
fmt.Printf("Wait count: %d\n", sqlStats["wait_count"])
```

## Connection Pool Behavior

### Connection Lifecycle

1. **Acquisition**: When a query is executed, a connection is acquired from the pool
2. **Usage**: The connection is used for the query
3. **Release**: After the query completes, the connection is returned to the pool
4. **Idle**: If not used, the connection remains idle for up to 10 minutes
5. **Recycling**: After 1 hour, the connection is closed and a new one is created
6. **Closure**: Idle connections are closed after 10 minutes of inactivity

### Under Load

- **Normal Load** (< 100 concurrent queries): All queries get connections immediately
- **High Load** (= 100 concurrent queries): New queries wait for available connections
- **Overload** (> 100 concurrent queries): Queries timeout after 5 seconds (configurable via context)

### Example: Handling Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

var result int
err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Query timeout: connection pool exhausted")
    } else {
        log.Printf("Query error: %v", err)
    }
}
```

## Testing

### Run All Tests

```bash
cd services/backend
go test -v ./internal/database
```

### Run Specific Test

```bash
go test -v ./internal/database -run TestConnectionPoolConfiguration
```

### Run Load Tests

```bash
go test -v ./internal/database -run TestConnectionPoolUnderLoad
```

### Skip Long Tests

```bash
go test -v -short ./internal/database
```

## Performance Considerations

### Why 100 Max Connections?

- PostgreSQL can handle 100-200 connections efficiently
- Allows 50 workers × 2 connections (read + write)
- Leaves headroom for admin connections and monitoring

### Why 10 Idle Connections?

- Keeps connections warm for immediate use
- Reduces connection establishment overhead
- Balances resource usage vs. performance

### Why 1 Hour Max Lifetime?

- Prevents stale connections
- Allows PostgreSQL to reclaim resources
- Handles network issues gracefully

### Why 10 Minutes Idle Timeout?

- Closes unused connections quickly
- Reduces resource consumption during low traffic
- Maintains pool efficiency

## Troubleshooting

### Connection Pool Exhausted

**Symptom**: Queries timeout with "context deadline exceeded"

**Causes**:
- Too many concurrent queries (> 100)
- Long-running queries holding connections
- Connection leaks (not closing connections)

**Solutions**:
1. Increase `DB_MAX_OPEN_CONNS` (not recommended beyond 200)
2. Optimize slow queries
3. Use connection pooling at application level
4. Check for connection leaks in code

### High Wait Count

**Symptom**: `wait_count` in statistics is high

**Causes**:
- Frequent connection pool exhaustion
- Queries waiting for available connections

**Solutions**:
1. Increase `DB_MAX_OPEN_CONNS`
2. Reduce query execution time
3. Implement query caching
4. Use read replicas for read-heavy workloads

### Too Many Idle Connections

**Symptom**: `idle_conns` is always at max

**Causes**:
- Low traffic
- `DB_MAX_IDLE_CONNS` too high

**Solutions**:
1. Reduce `DB_MAX_IDLE_CONNS`
2. Reduce `DB_CONN_MAX_IDLE_TIME`
3. This is usually not a problem unless resources are constrained

## Best Practices

### 1. Always Use Context

```go
// Good
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)

// Bad
err := db.Pool.QueryRow(context.Background(), "SELECT 1").Scan(&result)
```

### 2. Close Rows

```go
// Good
rows, err := db.Pool.Query(ctx, "SELECT * FROM messages")
if err != nil {
    return err
}
defer rows.Close()

// Bad (connection leak)
rows, err := db.Pool.Query(ctx, "SELECT * FROM messages")
// Forgot to close rows
```

### 3. Use Transactions for Multiple Operations

```go
tx, err := db.Pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// Multiple operations
_, err = tx.Exec(ctx, "UPDATE tenants SET balance = balance - $1 WHERE id = $2", cost, tenantID)
if err != nil {
    return err
}

_, err = tx.Exec(ctx, "INSERT INTO balance_history (...) VALUES (...)")
if err != nil {
    return err
}

return tx.Commit(ctx)
```

### 4. Monitor Connection Pool

```go
// Log statistics periodically
ticker := time.NewTicker(1 * time.Minute)
go func() {
    for range ticker.C {
        stats := db.GetStats()
        logger.Info("Connection pool stats", zap.Any("stats", stats))
    }
}()
```

## Architecture

```
┌─────────────────────────────────────────┐
│         Application Layer               │
│  (Handlers, Services, Workers)          │
└─────────────────┬───────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│         Database Package                │
│  ┌─────────────┐    ┌─────────────┐    │
│  │   pgxpool   │    │   sql.DB    │    │
│  │  (Modern)   │    │  (Legacy)   │    │
│  └──────┬──────┘    └──────┬──────┘    │
│         │                  │            │
│         └──────────┬───────┘            │
│                    │                    │
│         Connection Pool (100 max)       │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────┐
│         PostgreSQL Database             │
│  (al_hudhud)                            │
└─────────────────────────────────────────┘
```

## Related Documentation

- [Configuration Package](../config/README.md)
- [Balance Manager](../balance/README.md)
- [Message Lifecycle Spec](.kiro/specs/message-lifecycle-rebuild/)

## License

Copyright © 2026 Al-Hudhud AI
