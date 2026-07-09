# Concurrent Isolation Tests Guide

## Overview

This guide covers the concurrent isolation tests that verify tenant isolation under high concurrency loads. These tests ensure that:

1. **No Interference**: Operations from different tenants don't interfere with each other
2. **No Data Leakage**: Data remains isolated between tenants under concurrent load
3. **Proper Isolation**: Row Level Security (RLS) and cache isolation work correctly under stress

## Test Coverage

### 1. TestConcurrentTenantOperations

**Purpose**: Verifies that concurrent operations from multiple tenants don't interfere with each other.

**What it tests**:
- 10 tenants performing 50 concurrent operations each (500 total operations)
- Data integrity for each tenant after concurrent operations
- No cross-tenant data corruption

**Expected behavior**:
- All operations complete successfully
- Each tenant has exactly the expected number of records
- No data leakage between tenants

### 2. TestConcurrentBalanceOperations

**Purpose**: Tests concurrent balance deductions across multiple tenants.

**What it tests**:
- 5 tenants with initial balance of 10,000
- 100 concurrent deductions of 10.0 per tenant
- Atomic balance updates
- Final balance accuracy

**Expected behavior**:
- All deductions succeed
- Final balance = 10,000 - (100 × 10.0) = 9,000 for each tenant
- No race conditions or lost updates

### 3. TestConcurrentDataIsolation

**Purpose**: Verifies no data leakage between tenants under concurrent load.

**What it tests**:
- 2 tenants creating 100 records each concurrently
- Data isolation verification
- Cross-tenant data checks

**Expected behavior**:
- Each tenant has exactly 100 records
- No tenant can see the other tenant's data
- No cross-contamination of data

### 4. TestConcurrentRLSEnforcement

**Purpose**: Tests Row Level Security under concurrent query load.

**What it tests**:
- 2 tenants with 1 record each
- 200 concurrent queries per tenant
- RLS policy enforcement

**Expected behavior**:
- Each query returns exactly 1 record (tenant's own data)
- RLS policies are enforced consistently
- No tenant sees another tenant's data

### 5. TestConcurrentCacheIsolation

**Purpose**: Tests Redis cache isolation under concurrent load.

**What it tests**:
- 5 tenants performing 100 cache operations each
- Cache key isolation
- No cache pollution between tenants

**Expected behavior**:
- All cache operations succeed
- Each tenant's data is correctly isolated in cache
- No cross-tenant cache pollution

### 6. TestConcurrentTransactionIsolation

**Purpose**: Tests transaction isolation between tenants.

**What it tests**:
- 2 tenants performing 50 concurrent transactions each
- Balance updates and transaction recording
- Transaction atomicity and isolation

**Expected behavior**:
- All transactions complete successfully
- Final balances are accurate
- Transaction counts are correct
- No transaction interference between tenants

## Running the Tests

### Prerequisites

1. **Database**: PostgreSQL with test database
2. **Redis**: Redis server for cache tests
3. **Environment**: Test environment variables set

### Run All Concurrent Isolation Tests

```powershell
# Windows PowerShell
.\run-concurrent-isolation-tests.ps1
```

```bash
# Linux/Mac
./run-concurrent-isolation-tests.sh
```

### Run Individual Tests

```bash
# Test concurrent tenant operations
go test -v -run TestConcurrentTenantOperations ./internal/security

# Test concurrent balance operations
go test -v -run TestConcurrentBalanceOperations ./internal/security

# Test concurrent data isolation
go test -v -run TestConcurrentDataIsolation ./internal/security

# Test concurrent RLS enforcement
go test -v -run TestConcurrentRLSEnforcement ./internal/security

# Test concurrent cache isolation
go test -v -run TestConcurrentCacheIsolation ./internal/security

# Test concurrent transaction isolation
go test -v -run TestConcurrentTransactionIsolation ./internal/security
```

### Run with Race Detector

```bash
# Detect race conditions
go test -race -v ./internal/security
```

### Run with Timeout

```bash
# Set timeout for long-running tests
go test -timeout 5m -v ./internal/security
```

## Test Scenarios

### Scenario 1: High Concurrency Load

**Setup**:
- 10 tenants
- 50 operations per tenant
- 500 total concurrent operations

**Verification**:
- No errors during execution
- Data integrity maintained
- No cross-tenant interference

### Scenario 2: Balance Race Conditions

**Setup**:
- 5 tenants with 10,000 initial balance
- 100 concurrent deductions of 10.0 each
- 500 total balance operations

**Verification**:
- All deductions succeed
- Final balance = 9,000 per tenant
- No lost updates

### Scenario 3: Data Isolation Under Load

**Setup**:
- 2 tenants
- 100 concurrent inserts per tenant
- 200 total insert operations

**Verification**:
- Each tenant has exactly 100 records
- No data leakage
- No cross-tenant visibility

### Scenario 4: RLS Under Query Load

**Setup**:
- 2 tenants with 1 record each
- 200 concurrent queries per tenant
- 400 total queries

**Verification**:
- Each query returns exactly 1 record
- RLS policies enforced consistently
- No policy bypass

### Scenario 5: Cache Isolation

**Setup**:
- 5 tenants
- 100 cache operations per tenant
- 500 total cache operations

**Verification**:
- All cache operations succeed
- No cache key collisions
- No cross-tenant cache pollution

### Scenario 6: Transaction Isolation

**Setup**:
- 2 tenants
- 50 concurrent transactions per tenant
- 100 total transactions

**Verification**:
- All transactions commit successfully
- Balances are accurate
- No transaction interference

## Expected Results

### Success Criteria

✅ **All tests pass**: No errors during concurrent operations
✅ **Data integrity**: Each tenant's data is accurate and complete
✅ **No leakage**: No cross-tenant data visibility
✅ **Consistent isolation**: RLS and cache isolation work under load
✅ **No race conditions**: Atomic operations work correctly

### Performance Metrics

- **Throughput**: 500+ operations per second
- **Latency**: < 100ms per operation
- **Error rate**: 0%
- **Data accuracy**: 100%

## Troubleshooting

### Test Failures

**Problem**: Tests fail with "insufficient balance" errors

**Solution**:
- Check initial balance setup
- Verify atomic update logic
- Check for race conditions

**Problem**: Tests fail with "data leakage" errors

**Solution**:
- Verify RLS policies are enabled
- Check tenant_id in all queries
- Verify cache key prefixing

**Problem**: Tests timeout

**Solution**:
- Increase timeout: `go test -timeout 10m`
- Reduce number of operations
- Check database performance

### Race Conditions

**Problem**: Race detector reports data races

**Solution**:
- Review concurrent access patterns
- Add proper synchronization
- Use atomic operations where needed

### Database Deadlocks

**Problem**: Tests fail with deadlock errors

**Solution**:
- Review transaction isolation levels
- Reduce transaction scope
- Add retry logic

## Best Practices

### 1. Test Isolation

- Each test should be independent
- Clean up test data after each test
- Use unique identifiers for test data

### 2. Concurrency Control

- Use proper synchronization (WaitGroups, channels)
- Avoid shared mutable state
- Use atomic operations for counters

### 3. Error Handling

- Collect all errors from goroutines
- Report detailed error information
- Don't fail fast - collect all errors

### 4. Resource Management

- Clean up database connections
- Close Redis connections
- Release resources in defer statements

### 5. Test Data

- Use realistic data volumes
- Test edge cases (empty, full, boundary)
- Verify data integrity after tests

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Run Concurrent Isolation Tests
  run: |
    go test -v -race -timeout 10m ./internal/security
  env:
    DATABASE_URL: ${{ secrets.TEST_DATABASE_URL }}
    REDIS_URL: ${{ secrets.TEST_REDIS_URL }}
```

### Local Development

```bash
# Run before committing
make test-concurrent-isolation

# Run with coverage
go test -cover -v ./internal/security
```

## Monitoring

### Key Metrics

- **Test duration**: Should complete in < 5 minutes
- **Memory usage**: Should not exceed 500MB
- **CPU usage**: Should not exceed 80%
- **Database connections**: Should not exceed pool size

### Alerts

- Test failures
- Timeout errors
- Race condition detections
- Resource exhaustion

## Related Documentation

- [Security Tests Guide](./SECURITY_TESTS_GUIDE.md)
- [Tenant Isolation Design](../../.kiro/specs/tariff-pricing-system/design.md)
- [Requirements - Tenant Isolation](../../.kiro/specs/tariff-pricing-system/requirements.md)

## Validation Checklist

Before marking task complete:

- [ ] All 6 concurrent isolation tests pass
- [ ] No race conditions detected
- [ ] No data leakage between tenants
- [ ] Performance meets requirements
- [ ] Tests run in CI/CD pipeline
- [ ] Documentation is complete
- [ ] Code is reviewed and approved

## Requirements Validation

This test suite validates:

- **Requirement 26.5**: Multi-tenant isolation under concurrent load
- **Requirement 26.1**: Tenant data encryption and isolation
- **Requirement 26.7**: Cross-tenant access prevention
- **Requirement 26.11**: Audit logging of isolation violations

## Conclusion

These concurrent isolation tests provide comprehensive verification that the tenant isolation mechanisms work correctly under high concurrency loads. They ensure data integrity, prevent data leakage, and validate that the system can handle multiple tenants operating simultaneously without interference.
