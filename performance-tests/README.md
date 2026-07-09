# Performance Tests - Admin Interface Enhancements

## Overview

This directory contains performance tests using k6 for load testing and stress testing the Admin Interface system.

## Test Files

### 1. Load Test (`admin-load-test.js`)
- **Purpose**: Test system under normal and expected load
- **Target**: 100 concurrent users
- **Duration**: 19 minutes
- **Endpoints Tested**:
  - Dashboard Overview
  - Company Ratings
  - Company Details
  - Server Health
  - Alerts
  - Distribution Config
  - Signal/Push Statistics

### 2. Stress Test (`admin-stress-test.js`)
- **Purpose**: Test system beyond normal capacity to find breaking points
- **Peak Target**: 1500 concurrent users
- **Duration**: 24 minutes
- **Stages**:
  1. Normal load: 100 users (2 min)
  2. Stress load: 500 users (5 min)
  3. High stress: 1000 users (5 min)
  4. Peak stress: 1500 users (5 min)
  5. Scale down: 1000 users (5 min)
  6. Recovery: 0 users (2 min)

## Prerequisites

### Install k6

**macOS:**
```bash
brew install k6
```

**Linux:**
```bash
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows:**
```powershell
choco install k6
```

## Running Tests

### Load Test

```bash
# Basic run
k6 run performance-tests/admin-load-test.js

# With custom base URL and API key
k6 run -e BASE_URL=https://api.example.com -e API_KEY=your-api-key performance-tests/admin-load-test.js

# With results output
k6 run --out json=load-test-results.json performance-tests/admin-load-test.js
```

### Stress Test

```bash
# Basic run
k6 run performance-tests/admin-stress-test.js

# With custom configuration
k6 run -e BASE_URL=https://api.example.com -e API_KEY=your-api-key performance-tests/admin-stress-test.js

# With results output
k6 run --out json=stress-test-results.json performance-tests/admin-stress-test.js
```

## Performance Thresholds

### Load Test Thresholds
- **Response Time (p95)**: < 200ms
- **Error Rate**: < 1%
- **Success Rate**: > 99%

### Stress Test Thresholds (Relaxed)
- **Response Time (p95)**: < 500ms
- **Error Rate**: < 5%
- **Success Rate**: > 95%

## Interpreting Results

### Key Metrics

1. **http_req_duration**: Request duration
   - avg: Average response time
   - p(95): 95th percentile (95% of requests faster than this)
   - p(99): 99th percentile

2. **http_req_failed**: Failed requests rate
   - Should be < 1% for load test
   - Should be < 5% for stress test

3. **http_reqs**: Total number of requests
   - Indicates throughput

4. **vus**: Virtual users (concurrent users)
   - Shows load level

### Example Output

```
     ✓ dashboard overview status is 200
     ✓ company ratings status is 200
     
     checks.........................: 100.00% ✓ 12000 ✗ 0
     data_received..................: 24 MB   1.3 MB/s
     data_sent......................: 2.4 MB  130 kB/s
     http_req_duration..............: avg=150ms min=50ms med=140ms max=500ms p(95)=180ms p(99)=250ms
     http_req_failed................: 0.00%   ✓ 0     ✗ 12000
     http_reqs......................: 12000   631/s
     vus............................: 100     min=0   max=100
```

## Benchmarks

### Go Benchmarks

Run Go benchmarks for specific components:

```bash
# Rating Calculator
go test -bench=. -benchmem ./services/backend/internal/admin/rating/

# Routing Engine
go test -bench=. -benchmem ./services/backend/internal/admin/routing/

# API Handlers
go test -bench=. -benchmem ./services/backend/internal/api/
```

### Expected Benchmark Results

- **Rating Calculation**: < 10ms per operation
- **Routing Decision**: < 50ms at p95
- **API Response**: < 200ms average

## Continuous Performance Testing

### CI/CD Integration

Add to your CI/CD pipeline:

```yaml
performance-test:
  script:
    - k6 run --quiet --no-color performance-tests/admin-load-test.js
  only:
    - main
    - develop
```

### Monitoring

Set up alerts for:
- Response time degradation (> 200ms average)
- Error rate increase (> 1%)
- Throughput decrease (< expected RPS)

## Troubleshooting

### High Response Times
- Check database query performance
- Review API endpoint implementations
- Check for N+1 queries
- Verify caching is working

### High Error Rates
- Check logs for error patterns
- Verify database connections
- Check rate limiting configuration
- Review authentication/authorization

### Low Throughput
- Check server resources (CPU, Memory)
- Review connection pool settings
- Check for bottlenecks in critical paths
- Verify load balancer configuration

## Notes

- Run tests against staging environment first
- Ensure test data is representative
- Monitor server resources during tests
- Clean up test data after runs
- Document any performance regressions
