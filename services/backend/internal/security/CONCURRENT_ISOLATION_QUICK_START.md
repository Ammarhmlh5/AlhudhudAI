# Concurrent Isolation Tests - Quick Start

## What Are These Tests?

Concurrent isolation tests verify that multiple tenants can operate simultaneously without:
- Interfering with each other's operations
- Seeing each other's data
- Causing race conditions or data corruption

## Quick Run

### Windows PowerShell

```powershell
cd services/backend
.\run-concurrent-isolation-tests.ps1
```

### Linux/Mac

```bash
cd services/backend
./run-concurrent-isolation-tests.sh
```

## What Gets Tested

### 1. Concurrent Operations (500 operations)
- 10 tenants × 50 operations each
- Verifies no interference between tenants

### 2. Balance Operations (500 deductions)
- 5 tenants × 100 deductions each
- Verifies atomic balance updates

### 3. Data Isolation (200 inserts)
- 2 tenants × 100 inserts each
- Verifies no data leakage

### 4. RLS Enforcement (400 queries)
- 2 tenants × 200 queries each
- Verifies Row Level Security works under load

### 5. Cache Isolation (500 operations)
- 5 tenants × 100 cache operations each
- Verifies Redis isolation

### 6. Transaction Isolation (100 transactions)
- 2 tenants × 50 transactions each
- Verifies transaction atomicity

## Expected Output

```
=== RUN   TestConcurrentTenantOperations
--- PASS: TestConcurrentTenantOperations (2.34s)
=== RUN   TestConcurrentBalanceOperations
--- PASS: TestConcurrentBalanceOperations (1.89s)
=== RUN   TestConcurrentDataIsolation
--- PASS: TestConcurrentDataIsolation (1.56s)
=== RUN   TestConcurrentRLSEnforcement
--- PASS: TestConcurrentRLSEnforcement (2.12s)
=== RUN   TestConcurrentCacheIsolation
--- PASS: TestConcurrentCacheIsolation (1.78s)
=== RUN   TestConcurrentTransactionIsolation
--- PASS: TestConcurrentTransactionIsolation (1.45s)
PASS
ok      github.com/yourusername/sms-gateway/internal/security   11.14s
```

## Success Criteria

✅ All 6 tests pass
✅ No race conditions detected
✅ No data leakage between tenants
✅ All operations complete successfully
✅ Data integrity maintained

## Common Issues

### Issue: Tests timeout

**Solution**: Increase timeout
```bash
go test -timeout 10m -v ./internal/security
```

### Issue: Database connection errors

**Solution**: Check database is running
```bash
docker ps | grep postgres
```

### Issue: Redis connection errors

**Solution**: Check Redis is running
```bash
docker ps | grep redis
```

## Run Individual Tests

```bash
# Test 1: Concurrent operations
go test -v -run TestConcurrentTenantOperations ./internal/security

# Test 2: Balance operations
go test -v -run TestConcurrentBalanceOperations ./internal/security

# Test 3: Data isolation
go test -v -run TestConcurrentDataIsolation ./internal/security

# Test 4: RLS enforcement
go test -v -run TestConcurrentRLSEnforcement ./internal/security

# Test 5: Cache isolation
go test -v -run TestConcurrentCacheIsolation ./internal/security

# Test 6: Transaction isolation
go test -v -run TestConcurrentTransactionIsolation ./internal/security
```

## Run with Race Detector

```bash
go test -race -v ./internal/security
```

## Performance Expectations

- **Duration**: < 15 seconds total
- **Throughput**: 500+ operations/second
- **Memory**: < 500MB
- **CPU**: < 80%

## Next Steps

After tests pass:
1. Review test output for any warnings
2. Check performance metrics
3. Verify no race conditions
4. Mark task 34.6 as complete

## Need Help?

See [CONCURRENT_ISOLATION_TEST_GUIDE.md](./CONCURRENT_ISOLATION_TEST_GUIDE.md) for detailed documentation.
