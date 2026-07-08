# Tariff System Stress Testing - Quick Start Guide

## Overview

This guide helps you run stress tests for the tariff pricing system to validate system behavior under extreme conditions.

**Requirements Validated:**
- **27.1**: System behavior under high load (beyond normal capacity)
- **27.2**: Resource quota enforcement and rate limiting under stress

## What Are Stress Tests?

Stress tests push the system **beyond its normal capacity** to:
- Identify breaking points
- Validate graceful degradation
- Test resource quota enforcement
- Verify rate limiting effectiveness
- Ensure system recovery after stress

**Key Difference from Load Tests:**
- **Load Tests**: Validate performance under expected load
- **Stress Tests**: Push system beyond capacity to test limits and recovery

## Prerequisites

1. **k6 installed** (load testing tool)
   ```powershell
   # Windows
   choco install k6
   
   # macOS
   brew install k6
   
   # Linux
   sudo apt-get install k6
   ```

2. **Backend service running**
   ```powershell
   cd services/backend
   go run cmd/server/main.go
   ```

3. **Database and Redis running**
   ```powershell
   docker-compose up -d postgres redis
   ```

## Quick Start

### Run All Stress Tests

```powershell
cd performance-tests
.\run-tariff-stress-test.ps1
```

### With Custom Configuration

```powershell
.\run-tariff-stress-test.ps1 -ApiUrl "http://localhost:8080" -AdminToken "your-token"
```

### Skip Setup Checks

```powershell
.\run-tariff-stress-test.ps1 -SkipSetup
```

## Test Phases

### Phase 1: Extreme Load (0-20 minutes)
- **Duration**: 20 minutes
- **Load**: Ramps up to 1500 concurrent users
- **Purpose**: Test system behavior beyond normal capacity
- **Expected**: System should degrade gracefully, not crash

**Load Pattern:**
```
0-2m:   0 → 100 VUs   (warm up)
2-5m:   100 → 500 VUs (high load)
5-10m:  500 → 1000 VUs (extreme load)
10-15m: 1000 → 1500 VUs (beyond capacity)
15-18m: 1500 → 500 VUs (ramp down)
18-20m: 500 → 0 VUs (recovery test)
```

**Operations Mix:**
- 40% Balance deductions (most common)
- 30% Subscription operations
- 20% Query operations (read-heavy)
- 10% Gateway operations (resource-intensive)

### Phase 2: Resource Quota Stress (22-32 minutes)
- **Duration**: 10 minutes
- **Load**: 200 concurrent users
- **Purpose**: Test resource quota enforcement under stress
- **Expected**: Quotas should be enforced, requests rejected gracefully

**Tests:**
- CPU-intensive operations
- Memory-intensive operations
- Bandwidth-intensive operations

### Phase 3: Rate Limiting Stress (33-38 minutes)
- **Duration**: 5 minutes
- **Rate**: 1000 requests per second
- **Purpose**: Test rate limiting effectiveness
- **Expected**: Rate limits enforced, 429 responses returned

## Success Criteria

### Test 1: Extreme Load
- ✅ Success rate ≥ 80% (some failures expected under extreme stress)
- ✅ Error rate < 20%
- ✅ P95 latency < 5 seconds (degraded but acceptable)
- ✅ P99 latency < 10 seconds
- ✅ Graceful degradation detected (503/429 responses)
- ✅ System recovery after stress

### Test 2: Resource Quotas
- ✅ Resource quotas exceeded detected
- ✅ Quota violations rejected gracefully
- ✅ Appropriate error responses (429 with quota info)

### Test 3: Rate Limiting
- ✅ Rate limits hit detected
- ✅ Rate limiting enforced (429 responses)
- ✅ Rate limit headers present (X-RateLimit-*)

## Understanding Results

### Graceful Degradation

**Good Signs:**
- System returns 503 (Service Unavailable) when overloaded
- System returns 429 (Too Many Requests) when rate limited
- Error messages are clear and actionable
- System continues serving some requests

**Bad Signs:**
- System crashes or becomes unresponsive
- Database connections exhausted
- Memory leaks
- No error responses, just timeouts

### System Recovery

**Good Signs:**
- Response times return to normal after stress
- Success rate improves after ramp down
- No lingering errors or degradation
- Resources released properly

**Bad Signs:**
- System remains slow after stress
- High error rates persist
- Resource usage stays high
- Requires restart to recover

## Interpreting Metrics

### Response Latency
- **Under 1s**: Normal operation
- **1-5s**: Degraded but acceptable under stress
- **5-10s**: Severe degradation
- **Over 10s**: System struggling

### Success Rate
- **95-100%**: Excellent (normal load)
- **80-95%**: Good (under stress)
- **60-80%**: Acceptable (extreme stress)
- **Below 60%**: Poor (system failing)

### Error Rate
- **0-5%**: Excellent
- **5-20%**: Acceptable under stress
- **20-40%**: High but may be expected at extreme load
- **Over 40%**: System failing

## Common Issues

### Issue: All requests failing

**Possible Causes:**
- Backend not running
- Database connection issues
- Authentication token invalid

**Solution:**
```powershell
# Check backend
curl http://localhost:8080/health

# Check database
docker ps | grep postgres

# Check logs
cd services/backend
go run cmd/server/main.go
```

### Issue: No rate limiting detected

**Possible Causes:**
- Rate limiter not configured
- Rate limits too high
- Rate limiter disabled

**Solution:**
- Check rate limiter configuration in backend
- Verify rate limits are set appropriately
- Check middleware is enabled

### Issue: No resource quota enforcement

**Possible Causes:**
- Resource quotas not configured
- Quota checker disabled
- Quotas set too high

**Solution:**
- Check resource quota configuration
- Verify quota enforcement middleware
- Review quota thresholds

## Results Files

After running stress tests, you'll find:

### 1. Summary JSON
**File**: `results/tariff-stress-test-summary.json`

Contains high-level metrics:
- Success/failure counts
- Latency percentiles
- Graceful degradation events
- System recovery events

### 2. Full Results JSON
**File**: `results/tariff-stress-test-full.json`

Contains complete k6 output:
- All metrics
- All checks
- Threshold results
- Detailed timings

### 3. HTML Report
**File**: `results/tariff-stress-test-report.html`

Visual report with:
- Charts and graphs
- Metric trends over time
- Pass/fail indicators
- Detailed breakdowns

## Best Practices

### 1. Run During Off-Peak Hours
Stress tests generate extreme load. Run them when:
- No production traffic
- No other tests running
- System resources available

### 2. Monitor System Resources
Watch during stress tests:
```powershell
# CPU and Memory
docker stats

# Database connections
# Check PostgreSQL connection count

# Redis memory
redis-cli info memory
```

### 3. Start Small
First time running stress tests:
1. Run with lower VUs (e.g., 500 max instead of 1500)
2. Observe system behavior
3. Gradually increase load
4. Find actual breaking point

### 4. Baseline First
Before stress testing:
1. Run load tests to establish baseline
2. Document normal performance
3. Compare stress test results to baseline

## Troubleshooting

### Test Takes Too Long

**Reduce duration:**
```javascript
// Edit tariff-stress-test.js
stages: [
  { duration: '1m', target: 100 },   // Reduced from 2m
  { duration: '2m', target: 500 },   // Reduced from 3m
  // ... etc
]
```

### System Crashes During Test

**This is valuable data!** Document:
- When crash occurred (which phase)
- Load level at crash
- Error messages
- System state before crash

Then:
1. Identify bottleneck
2. Fix issue
3. Re-run stress test
4. Verify fix

### Results Not Saved

**Check:**
```powershell
# Verify results directory exists
Test-Path performance-tests/results

# Create if missing
New-Item -ItemType Directory -Path performance-tests/results

# Check permissions
Get-Acl performance-tests/results
```

## Next Steps

After stress testing:

1. **Analyze Results**
   - Review all three result files
   - Identify bottlenecks
   - Document breaking points

2. **Optimize System**
   - Address identified issues
   - Tune resource quotas
   - Adjust rate limits

3. **Re-test**
   - Run stress tests again
   - Verify improvements
   - Document new limits

4. **Document Capacity**
   - Maximum concurrent users
   - Maximum requests per second
   - Resource requirements
   - Scaling recommendations

## Requirements Validation

### Requirement 27.1: Vertical/Horizontal Scaling
**Validated by:**
- System handles increasing load up to breaking point
- Graceful degradation when capacity exceeded
- System recovery after stress

**Evidence:**
- Success rate under extreme load
- Latency degradation pattern
- Recovery metrics

### Requirement 27.2: Resource Management
**Validated by:**
- Resource quota enforcement
- Rate limiting effectiveness
- Connection pooling behavior

**Evidence:**
- Quota exceeded events
- Rate limit hit events
- Resource usage metrics

## Support

If you encounter issues:

1. **Check Logs**
   ```powershell
   # Backend logs
   cd services/backend
   go run cmd/server/main.go
   
   # Docker logs
   docker-compose logs -f
   ```

2. **Review Documentation**
   - Load test guide: `TARIFF_LOAD_TEST_GUIDE.md`
   - Integration test guide: `services/backend/internal/integration/TARIFF_SYSTEM_TEST_GUIDE.md`

3. **Ask for Help**
   - Include stress test results
   - Provide system logs
   - Describe observed behavior

## Summary

Stress testing validates that your system:
- ✅ Handles extreme load gracefully
- ✅ Enforces resource quotas under stress
- ✅ Applies rate limiting effectively
- ✅ Degrades gracefully when overloaded
- ✅ Recovers properly after stress

**Remember**: Stress tests are supposed to push the system to failure. The goal is to understand how it fails and ensure it fails gracefully.
