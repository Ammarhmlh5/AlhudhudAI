# Task 14.2: API Gateway Performance Test

## Overview

This task tests the API Gateway performance to verify it meets the requirement of responding within 10ms (Requirement 3.1).

## Test Objectives

1. **Response Time**: Measure response time for 1000 requests
2. **Verification**: Verify P95 response time < 10ms
3. **Throughput**: Measure throughput (requests/second)
4. **Success Rate**: Verify success rate >= 99%

## Requirements Validated

- **Requirement 3.1**: API Gateway SHALL respond within 10ms maximum

## Test Implementation

### Test Script

A PowerShell-based performance test script has been created:
- **Location**: `performance-tests/test-api-gateway-simple.ps1`
- **No external dependencies**: Works on Windows without k6 or Apache Bench
- **Concurrent requests**: Supports configurable concurrency level

### Test Configuration

```powershell
# Default configuration
API URL: http://localhost:8083/api/messages
Total Requests: 1000
Concurrent Requests: 10
Tenant ID: 1
```

## How to Run the Test

### Prerequisites

1. **Backend must be running**:
   ```powershell
   cd services/backend
   go run main.go
   ```

2. **Infrastructure must be running** (PostgreSQL, Redis):
   ```powershell
   # Start infrastructure
   .\start-infrastructure.bat
   
   # Run migrations
   .\run-migrations.bat
   ```

### Execute the Test

```powershell
# Run with default settings
.\performance-tests\test-api-gateway-simple.ps1

# Run with custom settings
.\performance-tests\test-api-gateway-simple.ps1 -TotalRequests 2000 -ConcurrentRequests 20

# Run with custom API URL
.\performance-tests\test-api-gateway-simple.ps1 -ApiUrl "http://localhost:8080/api/messages"
```

### Alternative: Using k6 (if installed)

If k6 is installed, you can use the more comprehensive test:

```bash
# Install k6 first
choco install k6

# Run the test
k6 run performance-tests/k6-tests/api-gateway-test.js
```

## Test Metrics

### Response Time Metrics

- **Average Response Time**: Mean of all response times
- **Minimum Response Time**: Fastest response
- **Maximum Response Time**: Slowest response
- **P95 Response Time**: 95% of requests completed within this time (CRITICAL)
- **P99 Response Time**: 99% of requests completed within this time

### Throughput Metrics

- **Total Duration**: Total time to complete all requests
- **Requests/Second**: Number of requests processed per second
- **Requests/Minute**: Number of requests processed per minute

### Success Metrics

- **Total Requests**: Total number of requests sent
- **Successful Requests**: Number of successful responses (HTTP 200)
- **Failed Requests**: Number of failed requests
- **Success Rate**: Percentage of successful requests

## Success Criteria

### ✅ PASS Conditions

1. **P95 Response Time < 10ms** (Requirement 3.1)
2. **Success Rate >= 99%**
3. **No critical errors**

### ❌ FAIL Conditions

1. P95 Response Time >= 10ms
2. Success Rate < 99%
3. Backend not accessible

## Expected Results

Based on the design specifications:

```
Response Time Statistics:
  Average: ~5-8 ms
  P95: < 10 ms ✓
  P99: < 15 ms

Throughput:
  Requests/Second: >= 100
  Success Rate: >= 99%
```

## Results Storage

Test results are automatically saved to:
- **Location**: `performance-tests/results/api-gateway-test-YYYYMMDD-HHMMSS.json`
- **Format**: JSON with detailed metrics
- **Includes**: All response times, statistics, and validation results

### Sample Results File

```json
{
  "timestamp": "20240225-143022",
  "configuration": {
    "api_url": "http://localhost:8083/api/messages",
    "total_requests": 1000,
    "concurrent_requests": 10,
    "tenant_id": 1
  },
  "results": {
    "total_requests": 1000,
    "successful_requests": 998,
    "failed_requests": 2,
    "success_rate_percent": 99.8,
    "total_duration_seconds": 12.5,
    "throughput_per_second": 80.0,
    "throughput_per_minute": 4800.0
  },
  "response_times": {
    "average_ms": 7.2,
    "minimum_ms": 4.5,
    "maximum_ms": 15.3,
    "p95_ms": 9.8,
    "p99_ms": 12.1
  },
  "requirements_validation": {
    "requirement_3_1": {
      "description": "Response Time < 10ms (P95)",
      "p95_response_time_ms": 9.8,
      "threshold_ms": 10,
      "passed": true
    }
  }
}
```

## Troubleshooting

### Issue: Backend not running

**Error**: `Backend is not running or not accessible`

**Solution**:
```powershell
cd services/backend
go run main.go
```

### Issue: High response times (> 10ms)

**Possible causes**:
1. Database connection pool exhausted
2. Redis not responding quickly
3. High system load
4. Network latency

**Solutions**:
- Check database connections: `DB_MAX_OPEN_CONNS=100`
- Check Redis performance: `redis-cli --latency`
- Monitor system resources: CPU, Memory
- Reduce concurrent requests temporarily

### Issue: Low success rate (< 99%)

**Possible causes**:
1. Rate limiting triggered
2. Database errors
3. Redis connection issues
4. Validation errors

**Solutions**:
- Check backend logs for errors
- Verify rate limiting settings: `RATE_LIMIT_REQUESTS_PER_MINUTE=100`
- Check database connectivity
- Verify Redis is running

### Issue: Connection timeout

**Error**: `The operation has timed out`

**Solutions**:
- Increase timeout in script (default: 5 seconds)
- Check network connectivity
- Verify backend is listening on correct port
- Check firewall settings

## Performance Optimization Tips

If the test fails to meet the < 10ms requirement:

1. **Redis Optimization**:
   - Ensure Redis is running locally (not remote)
   - Check Redis memory usage
   - Verify Redis persistence settings

2. **Database Optimization**:
   - Increase connection pool size
   - Add appropriate indexes
   - Check query performance

3. **Backend Optimization**:
   - Enable Go profiling to identify bottlenecks
   - Optimize JSON serialization
   - Review middleware overhead

4. **System Optimization**:
   - Close unnecessary applications
   - Ensure adequate RAM available
   - Check disk I/O performance

## Next Steps

After completing this test:

1. **Document Results**: Save the results file for reference
2. **Analyze Performance**: Review metrics and identify bottlenecks
3. **Proceed to Task 14.3**: Test batch processing performance
4. **Update Spec**: Mark task 14.2 as complete

## Related Tasks

- **Task 14.1**: Performance testing setup (completed)
- **Task 14.3**: Batch processing performance test (next)
- **Task 14.4**: Load testing (5000+ messages/minute)
- **Task 14.5**: Stress testing (10,000 messages/minute)

## References

- **Requirements**: `.kiro/specs/message-lifecycle-rebuild/requirements.md` (Requirement 3.1)
- **Design**: `.kiro/specs/message-lifecycle-rebuild/design.md` (API Gateway Component)
- **Implementation**: `services/backend/internal/gateway/handler.go`

---

**Status**: Ready to execute
**Created**: 2024-02-25
**Last Updated**: 2024-02-25
