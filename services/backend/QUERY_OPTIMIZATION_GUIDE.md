# Query Optimization Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Query Analyzer](#query-analyzer)
3. [Prepared Statements](#prepared-statements)
4. [Optimization Patterns](#optimization-patterns)
5. [Performance Monitoring](#performance-monitoring)
6. [Best Practices](#best-practices)
7. [Troubleshooting](#troubleshooting)

## Introduction

This guide covers query optimization techniques for the tariff pricing system, including EXPLAIN ANALYZE usage, query rewriting, and prepared statement implementation.

### Goals

- Reduce query execution time by 50%+
- Improve database resource utilization
- Implement prepared statement caching
- Provide query optimization recommendations

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Application Layer                      │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Query      │  │  Prepared    │  │  Optimized   │ │
│  │  Optimizer   │  │  Statement   │  │   Queries    │ │
│  │              │  │   Cache      │  │              │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────┤
│                   Database Layer                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │  PostgreSQL with pg_stat_statements              │  │
│  │  - Query execution plans                         │  │
│  │  - Performance statistics                        │  │
│  │  - Index usage tracking                          │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Query Analyzer

### Using EXPLAIN ANALYZE

The query analyzer uses PostgreSQL's EXPLAIN ANALYZE to understand query execution:

```go
optimizer := database.NewQueryOptimizer(db)

query := `
    SELECT id, cost_per_message 
    FROM tariffs 
    WHERE tenant_id = $1 AND country_code = $2
`

plan, err := optimizer.AnalyzeQuery(ctx, query, tenantID, "YE")
if err != nil {
    return err
}

fmt.Printf("Execution time: %v\n", plan.ExecutionTime)
fmt.Printf("Rows returned: %d\n", plan.RowsReturned)
```

### Understanding Query Plans

#### Sequential Scan (Slow)

```
Seq Scan on tariffs  (cost=0.00..1234.56 rows=100 width=32)
  Filter: (country_code = 'YE')
```

**Problem:** Full table scan
**Solution:** Add index on country_code

#### Index Scan (Fast)

```
Index Scan using idx_tariffs_tenant_country on tariffs
  (cost=0.29..8.31 rows=1 width=32)
  Index Cond: ((tenant_id = '...') AND (country_code = 'YE'))
```

**Good:** Using composite index efficiently

### Automatic Optimization

```go
// Get optimization suggestions
optimized, suggestions, err := optimizer.OptimizeQuery(ctx, query, args...)

for _, suggestion := range suggestions {
    log.Printf("Optimization: %s", suggestion)
}

// Use optimized query
rows, err := db.Query(optimized, args...)
```

## Prepared Statements

### Why Use Prepared Statements?

1. **Performance**: Query is parsed and planned once
2. **Security**: Protection against SQL injection
3. **Efficiency**: Reduced network overhead

### Implementation

#### Basic Usage

```go
cache := database.NewPreparedStatementCache(db)
defer cache.Close()

// Prepare statement (cached automatically)
stmt, err := cache.GetOrPrepare(ctx, "get_tariff", `
    SELECT id, cost_per_message 
    FROM tariffs 
    WHERE tenant_id = $1 AND service_id = $2
`)

// Execute multiple times (uses cached statement)
var tariff Tariff
err = stmt.Get(&tariff, tenantID, serviceID)
```

#### Optimized Queries

```go
oq := database.NewOptimizedQueries(db)
defer oq.Close()

// All queries use prepared statements internally
tariff, err := oq.GetActiveTariff(ctx, tenantID, serviceID, "YE", "Yemen Mobile")
subscription, err := oq.GetActiveSubscription(ctx, tenantID, userID)
messages, err := oq.GetRolloverMessages(ctx, tenantID, subscriptionID)
```

### Performance Comparison

| Method | First Execution | Subsequent Executions |
|--------|----------------|----------------------|
| Raw Query | 15ms | 15ms |
| Prepared Statement | 15ms | 3ms |
| Cached Prepared | 15ms | 1ms |

## Optimization Patterns

### Pattern 1: Always Include tenant_id

**Why:** Enables partition pruning and index usage

```sql
-- Bad: Missing tenant_id
SELECT * FROM tariffs WHERE country_code = 'YE';

-- Good: Includes tenant_id first
SELECT id, cost_per_message 
FROM tariffs 
WHERE tenant_id = $1 AND country_code = $2;
```

### Pattern 2: Avoid SELECT *

**Why:** Reduces I/O and network transfer

```sql
-- Bad: Fetches all columns
SELECT * FROM subscriptions WHERE user_id = $1;

-- Good: Specific columns only
SELECT id, status, messages_remaining, end_date
FROM subscriptions 
WHERE tenant_id = $1 AND user_id = $2;
```

### Pattern 3: Use Proper Indexes

**Why:** Dramatically improves query speed

```sql
-- Create composite index starting with tenant_id
CREATE INDEX idx_tariffs_tenant_country_operator 
ON tariffs(tenant_id, country_code, network_operator);

-- Query uses index efficiently
SELECT cost_per_message 
FROM tariffs 
WHERE tenant_id = $1 
  AND country_code = $2 
  AND network_operator = $3;
```

### Pattern 4: Add Date Ranges

**Why:** Limits data scanned

```sql
-- Bad: Scans entire table
SELECT * FROM accounting_transactions WHERE tenant_id = $1;

-- Good: Limited to recent data
SELECT id, amount, created_at 
FROM accounting_transactions 
WHERE tenant_id = $1 
  AND created_at >= NOW() - INTERVAL '30 days';
```

### Pattern 5: Use LIMIT

**Why:** Prevents loading excessive data

```sql
-- Bad: May return millions of rows
SELECT * FROM balance_history WHERE tenant_id = $1;

-- Good: Limited result set
SELECT id, amount, created_at 
FROM balance_history 
WHERE tenant_id = $1 
ORDER BY created_at DESC 
LIMIT 100;
```

### Pattern 6: Cursor-Based Pagination

**Why:** More efficient than OFFSET

```sql
-- Bad: OFFSET becomes slow with large offsets
SELECT * FROM transactions 
WHERE tenant_id = $1 
ORDER BY created_at DESC 
LIMIT 100 OFFSET 10000;

-- Good: Cursor-based pagination
SELECT * FROM transactions 
WHERE tenant_id = $1 
  AND created_at < $2  -- last seen timestamp
ORDER BY created_at DESC 
LIMIT 100;
```

### Pattern 7: Replace Subqueries with JOINs

**Why:** Better query planning

```sql
-- Bad: Correlated subquery
SELECT p.*, 
  (SELECT COUNT(*) FROM subscriptions WHERE package_id = p.id)
FROM packages p;

-- Good: JOIN with aggregation
SELECT p.id, p.name, COUNT(s.id) AS subscriber_count
FROM packages p
LEFT JOIN subscriptions s ON p.id = s.package_id
WHERE p.tenant_id = $1
GROUP BY p.id, p.name;
```

### Pattern 8: Use EXISTS Instead of IN

**Why:** Better performance for large sets

```sql
-- Bad: IN with subquery
SELECT * FROM subscriptions 
WHERE package_id IN (SELECT id FROM packages WHERE price > 50);

-- Good: EXISTS
SELECT s.* FROM subscriptions s
WHERE EXISTS (
  SELECT 1 FROM packages p 
  WHERE p.id = s.package_id AND p.price > 50
);
```

## Performance Monitoring

### Enable pg_stat_statements

```sql
-- In postgresql.conf
shared_preload_libraries = 'pg_stat_statements'
pg_stat_statements.track = all
pg_stat_statements.max = 10000

-- Create extension
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

### Find Slow Queries

```sql
-- Queries with high average execution time
SELECT 
  query,
  calls,
  mean_exec_time,
  max_exec_time,
  total_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- > 100ms
ORDER BY mean_exec_time DESC
LIMIT 20;
```

### Check Index Usage

```sql
-- Find unused indexes
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 
  AND indexname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC;
```

### Monitor Cache Hit Ratio

```sql
-- Should be > 99%
SELECT 
  sum(heap_blks_read) as heap_read,
  sum(heap_blks_hit) as heap_hit,
  sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as ratio
FROM pg_statio_user_tables;
```

### Check Table Bloat

```sql
-- Tables with high dead tuple percentage
SELECT 
  schemaname,
  tablename,
  n_live_tup,
  n_dead_tup,
  ROUND(100 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_pct
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY dead_pct DESC;
```

## Best Practices

### 1. Query Design

- Always include `tenant_id` in WHERE clause
- Use specific column names, not `SELECT *`
- Add appropriate LIMIT clauses
- Use date ranges for time-series data
- Prefer JOINs over subqueries

### 2. Index Strategy

- Create composite indexes starting with `tenant_id`
- Include columns used in WHERE, JOIN, ORDER BY
- Use partial indexes for filtered queries
- Remove unused indexes
- Monitor index size and usage

### 3. Prepared Statements

- Use for frequently executed queries
- Cache prepared statements
- Close statements when done
- Monitor prepared statement count

### 4. Connection Management

- Use connection pooling
- Set appropriate pool size
- Monitor active connections
- Close connections properly

### 5. Regular Maintenance

- Run VACUUM regularly
- Run ANALYZE after bulk changes
- Monitor table bloat
- Update statistics
- Reindex when needed

## Troubleshooting

### Problem: Query is slow

**Diagnosis:**
```sql
EXPLAIN ANALYZE SELECT ...;
```

**Solutions:**
1. Check if indexes are being used
2. Add missing indexes
3. Rewrite query to use indexes
4. Add date ranges to limit data
5. Use prepared statements

### Problem: High CPU usage

**Diagnosis:**
```sql
-- Find CPU-intensive queries
SELECT query, calls, total_exec_time
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;
```

**Solutions:**
1. Optimize slow queries
2. Add indexes
3. Use query caching
4. Implement rate limiting

### Problem: High memory usage

**Diagnosis:**
```sql
-- Check work_mem usage
SELECT name, setting, unit 
FROM pg_settings 
WHERE name = 'work_mem';
```

**Solutions:**
1. Add LIMIT to queries
2. Use cursor-based pagination
3. Increase work_mem if needed
4. Optimize sort operations

### Problem: Index not being used

**Diagnosis:**
```sql
EXPLAIN SELECT ...;
-- Look for "Seq Scan" instead of "Index Scan"
```

**Solutions:**
1. Run ANALYZE on table
2. Check index definition matches query
3. Ensure statistics are up to date
4. Consider index selectivity

### Problem: Prepared statement errors

**Diagnosis:**
```go
// Check error message
if err != nil {
    log.Printf("Prepared statement error: %v", err)
}
```

**Solutions:**
1. Verify statement name is unique
2. Check connection is valid
3. Ensure statement is prepared before use
4. Close and recreate if needed

## Performance Targets

| Query Type | Target Time | Acceptable | Needs Optimization |
|-----------|-------------|------------|-------------------|
| Simple SELECT | < 5ms | < 10ms | > 10ms |
| JOIN (2 tables) | < 20ms | < 50ms | > 50ms |
| JOIN (3+ tables) | < 50ms | < 100ms | > 100ms |
| Aggregation | < 50ms | < 100ms | > 100ms |
| Complex Report | < 200ms | < 500ms | > 500ms |

## Testing

### Unit Tests

```bash
cd services/backend
go test -v ./internal/database/query_optimizer_test.go
```

### Benchmarks

```bash
go test -bench=. -benchmem ./internal/database/
```

### Load Testing

```bash
# Use k6 for load testing
k6 run performance-tests/query-performance-test.js
```

## Additional Resources

- [PostgreSQL EXPLAIN Documentation](https://www.postgresql.org/docs/current/sql-explain.html)
- [pg_stat_statements Documentation](https://www.postgresql.org/docs/current/pgstatstatements.html)
- [Query Optimization SQL Scripts](./scripts/query-optimization.sql)
- [Query Optimizer Implementation](./internal/database/query_optimizer.go)

## Summary

Query optimization is critical for system performance. Key takeaways:

1. **Always use EXPLAIN ANALYZE** to understand query execution
2. **Implement prepared statements** for frequently executed queries
3. **Follow optimization patterns** (tenant_id first, specific columns, etc.)
4. **Monitor performance** using pg_stat_statements
5. **Regular maintenance** (VACUUM, ANALYZE, reindex)

By following these guidelines, you can achieve significant performance improvements and maintain optimal database performance as the system scales.
