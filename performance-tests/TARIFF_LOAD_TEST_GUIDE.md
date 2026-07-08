# Tariff System Load Testing Guide

## Overview

This guide covers load testing for the tariff pricing system, validating the system's ability to handle high concurrent loads across three critical operations.

## Test Coverage

### Test 1: 1000 Concurrent Subscriptions (Requirement 27.1)
- **Target**: 1000 concurrent subscription creations
- **Success Criteria**: 95%+ success rate (≥950 successful)
- **Latency Target**: P95 < 500ms
- **VUs**: 100 virtual users
- **Duration**: ~5 minutes

Tests the subscription manager's ability to handle concurrent package subscriptions with atomic balance deductions.

### Test 2: 10000 Concurrent Balance Deductions (Requirement 27.2)
- **Target**: 10000 concurrent balance deduction operations
- **Success Criteria**: 95%+ success rate (≥9500 successful)
- **Latency Target**: P95 < 100ms (atomic operations)
- **VUs**: 200 virtual users
- **Duration**: ~10 minutes

Tests the balance manager's atomic deduction operations under extreme concurrent load, validating race condition prevention.

### Test 3: 100 Concurrent Dedicated Gateways (Requirement 27.3)
- **Target**: 100 concurrent dedicated gateway creations
- **Success Criteria**: 95%+ success rate (≥95 successful)
- **Latency Target**: P95 < 1000ms
- **VUs**: 50 virtual users
- **Duration**: ~5 minutes

Tests the gateway manager's ability to create dedicated gateways with resource quotas under concurrent load.

## Prerequisites

### 1. Install k6

**Windows (Chocolatey):**
```powershell
choco install k6
```

**Windows (Scoop):**
```powershell
scoop install k6
```

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
# Debian/Ubuntu
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### 2. Ensure Backend is Running

The backend server must be running and accessible:

```powershell
# Start the backend
cd services/backend
go run main.go
```

### 3. Prepare Test Data

Ensure the database has:
- Multiple tenant accounts with sufficient balance
- Package definitions (Basic, Standard, Professional, Business, Enterprise)
- Admin authentication token

## Running the Tests

### Quick Start

```powershell
# Run all tariff load tests
cd performance-tests
.\run-tariff-load-test.ps1
```

### Custom Configuration

```powershell
# Specify API URL and admin token
.\run-tariff-load-test.ps1 -ApiUrl "http://localhost:8080" -AdminToken "your-admin-token"

# Skip pre-flight checks
.\run-tariff-load-test.ps1 -SkipPreCheck
```

### Direct k6 Execution

```bash
# Set environment variables
export API_URL=http://localhost:8080
export ADMIN_TOKEN=admin-test-token

# Run the test
k6 run tariff-load-test.js
```

## Test Execution Flow

The tests run sequentially to avoid resource contention:

```
Timeline:
├── 0:00 - 5:00   Test 1: Concurrent Subscriptions (1000 iterations)
├── 6:00 - 16:00  Test 2: Concurrent Balance Deductions (10000 iterations)
└── 17:00 - 22:00 Test 3: Concurrent Dedicated Gateways (100 iterations)

Total Duration: ~22 minutes
```

## Understanding Results

### Console Output

The test provides real-time metrics:

```
✓ Subscription: status is 200 or 201
✓ Subscription: has subscription_id
✓ Subscription: response time < 500ms

✓ Deduction: status is 200
✓ Deduction: balance updated
✓ Deduction: response time < 100ms
✓ Deduction: atomic operation

✓ Gateway: status is 200 or 201
✓ Gateway: has gateway_id
✓ Gateway: response time < 1000ms
✓ Gateway: resource quotas created
```

### Result Files

Three files are generated in `results/`:

1. **tariff-load-test-summary.json** - Key metrics and pass/fail status
2. **tariff-load-test-full.json** - Complete k6 metrics data
3. **tariff-load-test-report.html** - Visual HTML report

### Key Metrics

#### Test 1: Subscriptions
- **Created**: Number of successful subscriptions
- **Success Rate**: Percentage of successful operations
- **Latency P95**: 95th percentile response time
- **Target Achieved**: Whether ≥950 subscriptions were created

#### Test 2: Balance Deductions
- **Completed**: Number of successful deductions
- **Success Rate**: Percentage of successful operations
- **Latency P95**: 95th percentile response time (should be <100ms)
- **Target Achieved**: Whether ≥9500 deductions were completed

#### Test 3: Dedicated Gateways
- **Created**: Number of successful gateway creations
- **Success Rate**: Percentage of successful operations
- **Latency P95**: 95th percentile response time
- **Target Achieved**: Whether ≥95 gateways were created

## Interpreting Results

### Success Criteria

All three tests must meet their targets:

✅ **PASSED** - All targets achieved:
- Test 1: ≥950 subscriptions created (95%+)
- Test 2: ≥9500 deductions completed (95%+)
- Test 3: ≥95 gateways created (95%+)
- All latency targets met

❌ **FAILED** - One or more targets not met

### Common Issues

#### Low Success Rate (<95%)

**Possible Causes:**
- Database connection pool exhausted
- Insufficient tenant balance
- Race conditions in concurrent operations
- Network timeouts

**Solutions:**
- Increase database connection pool size
- Ensure test tenants have sufficient balance
- Review atomic operation implementations
- Check network stability

#### High Latency

**Possible Causes:**
- Database query performance
- Lock contention
- Insufficient server resources
- Network latency

**Solutions:**
- Add database indexes
- Optimize queries
- Scale server resources
- Use connection pooling

#### HTTP Errors (>5%)

**Possible Causes:**
- API rate limiting
- Server overload
- Authentication issues
- Database errors

**Solutions:**
- Adjust rate limits
- Scale server capacity
- Verify admin token
- Check database logs

## Performance Targets

### Requirement 27.1: Concurrent Subscriptions
- **Throughput**: 1000 concurrent operations
- **Success Rate**: ≥95%
- **Latency P95**: <500ms
- **Validates**: Atomic balance deduction, subscription creation

### Requirement 27.2: Concurrent Balance Deductions
- **Throughput**: 10000 concurrent operations
- **Success Rate**: ≥95%
- **Latency P95**: <100ms
- **Validates**: Race condition prevention, atomic updates

### Requirement 27.3: Concurrent Dedicated Gateways
- **Throughput**: 100 concurrent operations
- **Success Rate**: ≥95%
- **Latency P95**: <1000ms
- **Validates**: Gateway creation, resource quota allocation

## Monitoring During Tests

### System Resources

Monitor these metrics during test execution:

```powershell
# CPU and Memory
Get-Process -Name "backend" | Select-Object CPU, WorkingSet

# Database connections
# Check PostgreSQL pg_stat_activity

# Redis memory
redis-cli INFO memory
```

### Database Performance

```sql
-- Active queries
SELECT pid, query, state, query_start
FROM pg_stat_activity
WHERE state = 'active';

-- Lock contention
SELECT * FROM pg_locks WHERE NOT granted;

-- Connection count
SELECT count(*) FROM pg_stat_activity;
```

### Application Logs

Monitor backend logs for errors:

```powershell
# Follow logs
tail -f services/backend/logs/app.log
```

## Troubleshooting

### Test Fails to Start

**Error**: "k6 is not installed"
```powershell
# Install k6
choco install k6
```

**Error**: "API is not responding"
```powershell
# Check if backend is running
curl http://localhost:8080/health

# Start backend if needed
cd services/backend
go run main.go
```

### Test Hangs or Times Out

**Issue**: Test appears stuck

**Solutions:**
1. Check backend logs for errors
2. Verify database is responsive
3. Check Redis connectivity
4. Restart backend and retry

### Low Performance

**Issue**: Tests complete but don't meet targets

**Solutions:**
1. Increase database connection pool
2. Add database indexes
3. Optimize slow queries
4. Scale server resources
5. Enable Redis caching

## Best Practices

### Before Running Tests

1. ✅ Ensure backend is running and healthy
2. ✅ Verify database has test data
3. ✅ Check sufficient disk space for results
4. ✅ Close unnecessary applications
5. ✅ Use a stable network connection

### During Tests

1. ✅ Monitor system resources
2. ✅ Watch for error patterns in logs
3. ✅ Don't interrupt the test
4. ✅ Note any anomalies

### After Tests

1. ✅ Review all three result files
2. ✅ Check for error patterns
3. ✅ Compare with previous runs
4. ✅ Document any issues found
5. ✅ Clean up test data if needed

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Tariff Load Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Start Backend
        run: |
          cd services/backend
          go run main.go &
          sleep 10
      
      - name: Run Load Tests
        run: |
          cd performance-tests
          ./run-tariff-load-test.ps1
      
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: performance-tests/results/
```

## Related Documentation

- [Performance Testing Guide](./README.md)
- [Integration Tests Guide](../services/backend/internal/integration/TARIFF_SYSTEM_TEST_GUIDE.md)
- [Property Tests Guide](../services/backend/internal/PROPERTY_TESTS_GUIDE.md)
- [Task 34 Testing Guide](../.kiro/specs/tariff-pricing-system/guides/TASK_34_COMPREHENSIVE_TESTING_GUIDE.md)

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review backend logs
3. Consult the comprehensive testing guide
4. Contact the development team
