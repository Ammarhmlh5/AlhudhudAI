# Tariff System Stress Testing - Comprehensive Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Stress Testing Concepts](#stress-testing-concepts)
3. [Test Architecture](#test-architecture)
4. [Test Scenarios](#test-scenarios)
5. [Metrics and Thresholds](#metrics-and-thresholds)
6. [Running Tests](#running-tests)
7. [Analyzing Results](#analyzing-results)
8. [Optimization Strategies](#optimization-strategies)
9. [Troubleshooting](#troubleshooting)

## Introduction

### Purpose

Stress testing validates system behavior under extreme conditions by:
- Pushing the system beyond normal capacity
- Identifying breaking points and bottlenecks
- Validating graceful degradation mechanisms
- Testing resource quota enforcement
- Verifying rate limiting effectiveness
- Ensuring system recovery capabilities

### Requirements Coverage

**Requirement 27.1: Vertical/Horizontal Scaling**
- System should handle increasing load
- Should degrade gracefully when capacity exceeded
- Should recover after stress is removed

**Requirement 27.2: Resource Management**
- Resource quotas should be enforced under stress
- Rate limiting should be effective under high load
- Connection pooling should prevent resource exhaustion

### Difference from Load Testing

| Aspect | Load Testing | Stress Testing |
|--------|-------------|----------------|
| **Purpose** | Validate expected performance | Find breaking points |
| **Load Level** | Normal to high | Extreme to beyond capacity |
| **Expected Outcome** | All requests succeed | Some failures expected |
| **Duration** | Sustained load | Ramp up to failure |
| **Focus** | Performance metrics | Degradation and recovery |

## Stress Testing Concepts

### Graceful Degradation

**Definition**: System continues to operate at reduced capacity when overloaded, rather than failing completely.

**Good Graceful Degradation:**
```
Load: 100% → Response: 200 OK, 100ms
Load: 150% → Response: 200 OK, 500ms (slower but working)
Load: 200% → Response: 503 Service Unavailable (honest about capacity)
Load: 250% → Response: 503 Service Unavailable (still responding)
```

**Bad Degradation:**
```
Load: 100% → Response: 200 OK, 100ms
Load: 150% → Response: 200 OK, 100ms (seems fine)
Load: 200% → Response: Timeout (system hanging)
Load: 250% → Response: No response (system crashed)
```

### System Recovery

**Definition**: System returns to normal operation after stress is removed.

**Good Recovery:**
```
Stress Phase:    1500 VUs → 80% success, 2000ms latency
Ramp Down:       500 VUs → 90% success, 500ms latency
Recovery:        100 VUs → 99% success, 100ms latency
```

**Bad Recovery:**
```
Stress Phase:    1500 VUs → 80% success, 2000ms latency
Ramp Down:       500 VUs → 85% success, 1500ms latency (still slow)
Recovery:        100 VUs → 90% success, 800ms latency (not recovered)
```

### Resource Quotas

**Purpose**: Prevent any single tenant from consuming excessive resources.

**Quota Types:**
- **CPU Quota**: Maximum CPU cores per tenant
- **Memory Quota**: Maximum RAM per tenant
- **Bandwidth Quota**: Maximum network throughput
- **Connection Quota**: Maximum database connections
- **Rate Quota**: Maximum requests per time period

**Enforcement:**
- **Soft Limit**: Warning when approaching quota
- **Hard Limit**: Reject requests when quota exceeded

### Rate Limiting

**Purpose**: Prevent system overload by limiting request rate.

**Strategies:**
- **Token Bucket**: Allows bursts, refills over time
- **Leaky Bucket**: Smooth rate, no bursts
- **Fixed Window**: Simple but can have edge cases
- **Sliding Window**: More accurate but complex

**Response:**
```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1640000000
Retry-After: 60

{
  "error": "Rate limit exceeded",
  "limit": 1000,
  "window": "1m",
  "retry_after": 60
}
```

## Test Architecture

### Test Structure

```
tariff-stress-test.js
├── Metrics Definition
│   ├── Counters (events)
│   ├── Rates (percentages)
│   ├── Trends (latencies)
│   └── Gauges (current values)
├── Test Scenarios
│   ├── Extreme Load (20m)
│   ├── Resource Quota Stress (10m)
│   └── Rate Limiting Stress (5m)
├── Test Functions
│   ├── stressTestMixed()
│   ├── stressTestResourceQuotas()
│   └── stressTestRateLimiting()
└── Results Handler
    ├── Summary JSON
    ├── Full Results JSON
    └── HTML Report
```

### Custom Metrics

#### Counters
```javascript
const systemOverloaded = new Counter('system_overloaded');
const resourceQuotaExceeded = new Counter('resource_quota_exceeded');
const rateLimitHit = new Counter('rate_limit_hit');
const gracefulDegradation = new Counter('graceful_degradation');
const systemRecovered = new Counter('system_recovered');
```

#### Rates
```javascript
const errorRate = new Rate('error_rate');
const successRate = new Rate('success_rate');
```

#### Trends
```javascript
const responseLatency = new Trend('response_latency');
```

#### Gauges
```javascript
const activeConnections = new Gauge('active_connections');
const cpuUsage = new Gauge('cpu_usage_percent');
const memoryUsage = new Gauge('memory_usage_percent');
```

## Test Scenarios

### Scenario 1: Extreme Load Test

**Duration**: 20 minutes  
**Executor**: ramping-vus  
**Purpose**: Test system behavior beyond normal capacity

**Load Pattern:**
```javascript
stages: [
  { duration: '2m', target: 100 },   // Warm up
  { duration: '3m', target: 500 },   // High load
  { duration: '5m', target: 1000 },  // Extreme load
  { duration: '5m', target: 1500 },  // Beyond capacity
  { duration: '3m', target: 500 },   // Ramp down
  { duration: '2m', target: 0 },     // Recovery
]
```

**Operations Mix:**
- 40% Balance deductions (most common, fast)
- 30% Subscription operations (medium complexity)
- 20% Query operations (read-heavy, cacheable)
- 10% Gateway operations (resource-intensive, slow)

**Expected Behavior:**
- **0-2m (100 VUs)**: Normal operation, 99%+ success
- **2-5m (500 VUs)**: High load, 95%+ success, increased latency
- **5-10m (1000 VUs)**: Extreme load, 90%+ success, degraded latency
- **10-15m (1500 VUs)**: Beyond capacity, 80%+ success, some 503/429
- **15-18m (500 VUs)**: Ramp down, improving metrics
- **18-20m (0 VUs)**: Recovery, return to normal

**Success Criteria:**
- Success rate ≥ 80% overall
- Error rate < 20%
- P95 latency < 5 seconds
- P99 latency < 10 seconds
- Graceful degradation detected (503/429 responses)
- System recovery after stress

### Scenario 2: Resource Quota Stress Test

**Duration**: 10 minutes  
**Executor**: constant-vus (200 VUs)  
**Purpose**: Validate resource quota enforcement

**Operations:**
1. **CPU-Intensive**: Simulate heavy computation
2. **Memory-Intensive**: Allocate large data structures
3. **Bandwidth-Intensive**: Transfer large payloads

**Expected Behavior:**
- Quotas should be enforced
- Requests exceeding quotas should be rejected
- Error responses should be clear and actionable
- System should remain stable

**Success Criteria:**
- Resource quota exceeded events detected
- Appropriate error responses (429 with quota info)
- System stability maintained
- No resource exhaustion

### Scenario 3: Rate Limiting Stress Test

**Duration**: 5 minutes  
**Executor**: constant-arrival-rate (1000 req/s)  
**Purpose**: Validate rate limiting effectiveness

**Load:**
- 1000 requests per second
- Sustained for 5 minutes
- Total: ~300,000 requests

**Expected Behavior:**
- Rate limits should be enforced
- 429 responses should be returned
- Rate limit headers should be present
- System should remain responsive

**Success Criteria:**
- Rate limit hit events detected
- 429 responses with proper headers
- System stability maintained
- No request queue overflow

## Metrics and Thresholds

### Response Latency

**Thresholds:**
```javascript
'response_latency': ['p(95)<5000'],   // 95% < 5s
'response_latency': ['p(99)<10000'],  // 99% < 10s
```

**Interpretation:**
- **< 100ms**: Excellent (normal operation)
- **100-500ms**: Good (high load)
- **500-2000ms**: Acceptable (extreme load)
- **2000-5000ms**: Degraded (beyond capacity)
- **> 5000ms**: Poor (system struggling)

### Success Rate

**Threshold:**
```javascript
'success_rate': ['rate>0.80'],  // 80%+ success
```

**Interpretation:**
- **95-100%**: Excellent (normal load)
- **90-95%**: Good (high load)
- **80-90%**: Acceptable (extreme load)
- **60-80%**: Poor (beyond capacity)
- **< 60%**: Failing (system overloaded)

### Error Rate

**Threshold:**
```javascript
'error_rate': ['rate<0.20'],  // < 20% errors
```

**Interpretation:**
- **0-5%**: Excellent
- **5-10%**: Good
- **10-20%**: Acceptable under stress
- **20-40%**: High (expected at extreme load)
- **> 40%**: Failing

### HTTP Errors

**Threshold:**
```javascript
'http_req_failed': ['rate<0.30'],  // < 30% HTTP errors
```

**Acceptable Errors Under Stress:**
- **429 Too Many Requests**: Rate limit hit (expected)
- **503 Service Unavailable**: System overloaded (expected)
- **504 Gateway Timeout**: Slow response (acceptable)

**Unacceptable Errors:**
- **500 Internal Server Error**: Application bug
- **502 Bad Gateway**: Infrastructure issue
- **Connection refused**: Service down

### Graceful Degradation

**Threshold:**
```javascript
'graceful_degradation': ['count>0'],  // Should degrade gracefully
```

**Indicators:**
- 503 responses when overloaded
- 429 responses when rate limited
- Clear error messages
- System continues serving some requests

### System Recovery

**Threshold:**
```javascript
'system_recovered': ['count>0'],  // Should recover
```

**Indicators:**
- Response times return to normal
- Success rate improves
- Error rate decreases
- Resources released

## Running Tests

### Prerequisites

1. **Install k6**
   ```powershell
   choco install k6  # Windows
   brew install k6   # macOS
   ```

2. **Start Backend**
   ```powershell
   cd services/backend
   go run cmd/server/main.go
   ```

3. **Start Infrastructure**
   ```powershell
   docker-compose up -d postgres redis
   ```

### Basic Execution

```powershell
cd performance-tests
.\run-tariff-stress-test.ps1
```

### With Configuration

```powershell
.\run-tariff-stress-test.ps1 `
  -ApiUrl "http://localhost:8080" `
  -AdminToken "your-admin-token"
```

### Direct k6 Execution

```powershell
# Set environment variables
$env:API_URL = "http://localhost:8080"
$env:ADMIN_TOKEN = "your-token"

# Run test
k6 run tariff-stress-test.js
```

### Monitoring During Test

**Terminal 1: Run Test**
```powershell
.\run-tariff-stress-test.ps1
```

**Terminal 2: Monitor Resources**
```powershell
# Docker stats
docker stats

# Backend logs
cd services/backend
go run cmd/server/main.go
```

**Terminal 3: Monitor Database**
```powershell
# PostgreSQL connections
docker exec -it postgres psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# Redis memory
docker exec -it redis redis-cli info memory
```

## Analyzing Results

### Summary JSON

**Location**: `results/tariff-stress-test-summary.json`

**Key Sections:**
```json
{
  "test_1_extreme_load": {
    "total_requests": 50000,
    "success_rate": 85.5,
    "error_rate": 14.5,
    "latency_p95_ms": 3500,
    "graceful_degradation_count": 1200,
    "system_recovered_count": 800,
    "target_achieved": true
  },
  "test_2_resource_quotas": {
    "quota_exceeded_count": 150,
    "quota_enforced": true,
    "target_achieved": true
  },
  "test_3_rate_limiting": {
    "rate_limit_hit_count": 5000,
    "rate_limit_enforced": true,
    "target_achieved": true
  },
  "overall": {
    "all_targets_achieved": true,
    "graceful_degradation_working": true,
    "system_recovery_working": true
  }
}
```

### HTML Report

**Location**: `results/tariff-stress-test-report.html`

**Sections:**
- **Overview**: High-level summary
- **Metrics**: All metrics with charts
- **Checks**: Pass/fail for all checks
- **Thresholds**: Threshold results
- **Groups**: Results by test phase

### Interpreting Results

#### Excellent Results
```
✅ Success Rate: 90%+
✅ P95 Latency: < 2s
✅ Graceful Degradation: Detected
✅ System Recovery: Complete
✅ Resource Quotas: Enforced
✅ Rate Limiting: Effective
```

#### Good Results
```
✅ Success Rate: 80-90%
✅ P95 Latency: 2-5s
✅ Graceful Degradation: Detected
✅ System Recovery: Partial
✅ Resource Quotas: Enforced
✅ Rate Limiting: Effective
```

#### Poor Results
```
❌ Success Rate: < 80%
❌ P95 Latency: > 5s
❌ Graceful Degradation: Not detected
❌ System Recovery: Failed
❌ Resource Quotas: Not enforced
❌ Rate Limiting: Not effective
```

## Optimization Strategies

### If Success Rate Too Low

**Possible Causes:**
- Database connection pool too small
- Insufficient server resources
- Inefficient queries
- No caching

**Solutions:**
1. Increase database connection pool
2. Add read replicas
3. Implement caching (Redis)
4. Optimize slow queries
5. Add more server instances

### If Latency Too High

**Possible Causes:**
- Slow database queries
- No connection pooling
- Synchronous operations
- No caching

**Solutions:**
1. Add database indexes
2. Implement connection pooling
3. Use async operations
4. Add caching layer
5. Optimize critical paths

### If No Graceful Degradation

**Possible Causes:**
- No rate limiting
- No circuit breakers
- No load shedding
- No backpressure handling

**Solutions:**
1. Implement rate limiting
2. Add circuit breakers
3. Implement load shedding
4. Add request queues with limits
5. Return 503 when overloaded

### If System Doesn't Recover

**Possible Causes:**
- Resource leaks
- Connection leaks
- Memory leaks
- Goroutine leaks

**Solutions:**
1. Fix resource leaks
2. Implement proper cleanup
3. Use context with timeout
4. Monitor goroutine count
5. Implement health checks

## Troubleshooting

### Test Fails Immediately

**Check:**
```powershell
# Backend running?
curl http://localhost:8080/health

# Database accessible?
docker ps | grep postgres

# Redis accessible?
docker ps | grep redis
```

### No Rate Limiting Detected

**Check:**
```go
// Rate limiter configured?
rateLimiter := ratelimit.NewTenantRateLimiter(
    tenantID,
    &ratelimit.Config{
        APIRatePerMinute: 1000,  // Should be lower for testing
    },
)

// Middleware enabled?
router.Use(middleware.RateLimitMiddleware())
```

### No Resource Quotas Enforced

**Check:**
```go
// Quotas configured?
quotas := &models.ResourceQuota{
    CPUQuota:    2.0,  // 2 cores
    MemoryQuota: 4 * 1024 * 1024 * 1024,  // 4GB
}

// Quota checker enabled?
router.Use(middleware.ResourceQuotaMiddleware())
```

### System Crashes During Test

**This is valuable data!**

**Document:**
- Load level at crash
- Error messages
- System state
- Resource usage

**Investigate:**
```powershell
# Check logs
docker-compose logs backend

# Check resource usage
docker stats

# Check database
docker exec -it postgres psql -U postgres -c "SELECT * FROM pg_stat_activity;"
```

## Summary

Stress testing is essential for:
- ✅ Understanding system limits
- ✅ Validating graceful degradation
- ✅ Testing resource protection
- ✅ Ensuring system recovery
- ✅ Planning capacity

**Remember**: The goal is not to pass all tests perfectly, but to understand how the system behaves under extreme conditions and ensure it fails gracefully.
