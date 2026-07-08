# Query Optimization Quick Start Guide

## Overview

This guide provides quick instructions for using the query optimization tools in the tariff pricing system.

## Quick Start

### 1. Analyze a Query

```go
import "your-project/internal/database"

// Create optimizer
optimizer := database.NewQueryOptimizer(db)

// Analyze query
query := "SELECT * FROM tariffs WHERE tenant_id = $1"
plan, err := optimizer.AnalyzeQuery(ctx, query, tenantID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Execution time: %v\n", plan.ExecutionTime)
fmt.Printf("Total cost: %.2f\n", plan.TotalCost)
```

### 2. Get Optimization Suggestions

```go
// Get optimization suggestions
optimized, suggestions, err := optimizer.OptimizeQuery(ctx, query, tenantID)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Optimized query:", optimized)
for _, suggestion := range suggestions {
    fmt.Println("- ", suggestion)
}
```

### 3. Use Prepared Statements

```go
// Create optimized queries instance
oq := database.NewOptimizedQueries(db)
defer oq.Close()

// Use prepared statement (automatically cached)
tariff, err := oq.GetActiveTariff(ctx, tenantID, serviceID, countryCode, operator)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Tariff cost: %.2f %s\n", tariff.CostPerMessage, tariff.Currency)
```

### 4. Run SQL Analysis Scripts

```bash
# Run query optimization analysis
psql -U postgres -d sms_gateway -f services/backend/scripts/query-optimization.sql
```

## Common Optimizations

### 1. Always Include tenant_id

❌ **Bad:**
```sql
SELECT * FROM tariffs WHERE country_code = 'YE'
```

✅ **Good:**
```sql
SELECT id, cost_per_message FROM tariffs 
WHERE tenant_id = $1 AND country_code = $2
```

### 2. Avoid SELECT *

❌ **Bad:**
```sql
SELECT * FROM subscriptions WHERE user_id = $1
```

✅ **Good:**
```sql
SELECT id, status, messages_remaining, end_date 
FROM subscriptions WHERE tenant_id = $1 AND user_id = $2
```

### 3. Use Prepared Statements

❌ **Bad:**
```go
// Query executed every time
db.Query("SELECT * FROM tariffs WHERE id = ?", id)
```

✅ **Good:**
```go
// Prepared once, executed many times
stmt, _ := cache.GetOrPrepare(ctx, "get_tariff", query)
stmt.Query(id)
```

### 4. Add Date Ranges

❌ **Bad:**
```sql
SELECT * FROM accounting_transactions WHERE tenant_id = $1
```

✅ **Good:**
```sql
SELECT id, amount, created_at FROM accounting_transactions 
WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '30 days'
```

## Performance Targets

- **Simple queries**: < 10ms
- **Join queries**: < 50ms
- **Aggregation queries**: < 100ms
- **Complex reports**: < 500ms

## Testing

```bash
# Run query optimizer tests
cd services/backend
go test -v ./internal/database/query_optimizer_test.go

# Run benchmarks
go test -bench=. -benchmem ./internal/database/
```

## Monitoring

### Check Slow Queries

```sql
-- Find queries taking > 100ms
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
WHERE mean_exec_time > 100
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### Check Index Usage

```sql
-- Find unused indexes
SELECT indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND indexname NOT LIKE '%_pkey';
```

## Common Issues

### Issue: Query is slow

**Solution:**
1. Run EXPLAIN ANALYZE
2. Check if indexes are being used
3. Add tenant_id to WHERE clause
4. Use specific columns instead of SELECT *

### Issue: High memory usage

**Solution:**
1. Add LIMIT to queries
2. Use cursor-based pagination
3. Avoid loading large result sets

### Issue: Prepared statement not found

**Solution:**
1. Ensure statement is prepared before use
2. Check statement name matches
3. Verify connection is still valid

## Next Steps

- Read [QUERY_OPTIMIZATION_GUIDE.md](./QUERY_OPTIMIZATION_GUIDE.md) for detailed information
- Review [query-optimization.sql](./scripts/query-optimization.sql) for examples
- Check [query_optimizer.go](./internal/database/query_optimizer.go) for implementation details

## Support

For issues or questions:
1. Check the troubleshooting section in the full guide
2. Review test cases for usage examples
3. Consult the SQL optimization scripts
