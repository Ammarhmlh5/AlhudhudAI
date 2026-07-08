# Stress Test Quick Start Guide

## 🎯 Purpose

This stress test pushes the system **beyond normal limits** to:
- Find maximum capacity (breaking point)
- Identify bottlenecks
- Understand failure modes
- Guide optimization efforts

**Target**: 10,000+ messages/minute (up to 18,000/min at peak)

⚠️ **WARNING**: This test is designed to stress the system until it fails. Failures are EXPECTED and help us understand system limits.

## 🚀 Quick Start (5 Minutes)

### 1. Install k6

```powershell
# Using Chocolatey (Windows)
choco install k6

# Or download from: https://k6.io/docs/getting-started/installation/
```

### 2. Start Infrastructure

```powershell
# From project root
docker-compose up -d

# Verify backend is running
curl http://localhost:8080/health
```

### 3. Run Stress Test

```powershell
cd performance-tests
.\run-stress-test.ps1
```

The script will:
- ✅ Check prerequisites
- ⚠️ Display warning about stress testing
- ❓ Ask for confirmation
- 🚀 Run the test (8 minutes)
- 📊 Display results
- 💡 Provide recommendations

## 📊 What to Expect

### Test Stages (8 minutes total)

```
1 min  @ 50 VUs   (~3,000/min)   - Warm up
2 min  @ 100 VUs  (~6,000/min)   - Normal load
2 min  @ 200 VUs  (~12,000/min)  - High load
2 min  @ 300 VUs  (~18,000/min)  - EXTREME stress
1 min  @ 0 VUs                   - Cool down
```

### Expected Behavior

✅ **Good Signs**:
- System handles 10,000+/min initially
- Errors increase gradually (not suddenly)
- System recovers during cool down
- Clear bottleneck identified

⚠️ **Expected Issues** (these are NORMAL in stress testing):
- Error rate increases at high load
- Response time degrades
- Some requests fail (5-20% at peak)
- Resources approach limits

❌ **Concerning Signs** (need investigation):
- System crashes
- Sudden failure (0% to 100% errors)
- No recovery during cool down
- Data corruption

## 📈 Sample Results

### Successful Stress Test

```
╔════════════════════════════════════════════════════════╗
║     Stress Test - 10,000 Messages/Minute Summary      ║
╚════════════════════════════════════════════════════════╝

📊 Request Statistics:
   Total Requests:      144,000
   Successful:          120,000 (83.33%)
   Failed:              24,000 (16.67%)

🚀 Throughput:
   Requests/Minute:     18,000.00
   Target (10,000/min): ✓ ACHIEVED

⏱️  Response Time:
   P95:                 150.20ms
   Max:                 2,500.00ms

🎯 Overall Result:
   ⚠ PARTIAL - System reached limits (EXPECTED)

🔍 Breaking Points:
   • Maximum sustainable: ~12,000/min
   • Database connections saturated at peak
   • Error rate increased above 200 VUs
```

## 🔍 Analysis Checklist

After the test, analyze:

### 1. Maximum Capacity
- [ ] What's the highest throughput with <5% errors?
- [ ] At what load does system start failing?
- [ ] Is this sufficient for production needs?

### 2. Bottlenecks
- [ ] Which resource hit 100% first?
  - [ ] CPU
  - [ ] Memory
  - [ ] Database connections
  - [ ] Redis memory
  - [ ] Worker pool
- [ ] Is the bottleneck expected?
- [ ] Can it be optimized?

### 3. Failure Modes
- [ ] What types of errors occurred?
  - [ ] 429 (Rate limit)
  - [ ] 500 (Server error)
  - [ ] 503 (Service unavailable)
- [ ] Did errors appear gradually?
- [ ] Did system recover during cool down?

### 4. Resource Usage
- [ ] CPU usage pattern
- [ ] Memory growth (any leaks?)
- [ ] Database connection usage
- [ ] Redis memory usage

### 5. Data Integrity
- [ ] Any race conditions?
- [ ] Balance management still atomic?
- [ ] Messages lost or duplicated?

## 💡 Common Optimizations

Based on bottleneck identified:

### Database Connection Pool Saturated
```go
// Increase max connections
MaxOpenConns: 150  // was 100
MaxIdleConns: 20   // was 10
```

### Worker Pool Too Small
```go
// Increase worker count
WorkerCount: 100  // was 50
```

### CPU Saturated
- Profile hot paths
- Optimize JSON marshaling
- Consider horizontal scaling

### Memory Growing
- Check for goroutine leaks
- Verify Redis TTL settings
- Review object pooling

### Redis Memory High
- Review TTL settings
- Implement eviction policy
- Consider Redis cluster

## 📋 Manual Execution

If you prefer to run k6 directly:

```powershell
cd performance-tests

# Set API URL
$env:API_URL = "http://localhost:8080"

# Run test
k6 run `
  --summary-export=results/stress-test-summary.json `
  --out json=results/stress-test-full.json `
  k6-tests/stress-test.js
```

## 📊 View Results

### Summary JSON
```powershell
Get-Content results/stress-test-summary.json | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

### Resource Monitoring
```powershell
# View CSV data
Import-Csv results/resource-monitoring.csv | Out-GridView

# View summary
Get-Content results/resource-monitoring-summary.json | ConvertFrom-Json
```

## 🎓 Understanding Results

### Throughput Achievement

| Result | Meaning | Action |
|--------|---------|--------|
| 18,000+/min | Excellent | System exceeds requirements significantly |
| 12,000-18,000/min | Good | System handles stress well, minor optimizations |
| 10,000-12,000/min | Acceptable | Meets requirements, consider optimizations |
| <10,000/min | Needs Work | Investigate bottlenecks, optimize |

### Error Rate

| Error Rate | Meaning | Action |
|------------|---------|--------|
| <5% | Excellent | System handles stress gracefully |
| 5-15% | Good | Expected under extreme stress |
| 15-30% | Acceptable | System reaching limits |
| >30% | Concerning | Investigate failure modes |

### Response Time (P95)

| P95 Latency | Meaning | Action |
|-------------|---------|--------|
| <50ms | Excellent | System responsive under stress |
| 50-100ms | Good | Acceptable degradation |
| 100-200ms | Acceptable | Expected under extreme load |
| >200ms | Concerning | Investigate bottlenecks |

## ⚠️ Safety Guidelines

### DO NOT:
- ❌ Run on production systems
- ❌ Run during business hours
- ❌ Run without monitoring
- ❌ Run without backup plan

### DO:
- ✅ Run on test/staging environment
- ✅ Monitor system resources
- ✅ Have rollback plan ready
- ✅ Document findings
- ✅ Schedule during maintenance windows

## 🔄 Iterative Optimization

1. **Run stress test** → Identify bottleneck
2. **Apply optimization** → Fix bottleneck
3. **Re-run stress test** → Verify improvement
4. **Repeat** → Until acceptable performance

### Example Iteration

**Iteration 1**:
- Result: 8,000/min, DB connections saturated
- Action: Increase connection pool to 150
- Expected: Higher throughput

**Iteration 2**:
- Result: 12,000/min, CPU at 95%
- Action: Optimize hot paths
- Expected: Better CPU utilization

**Iteration 3**:
- Result: 15,000/min, acceptable performance
- Action: Document capacity limits
- Status: Complete

## 📚 Additional Resources

- **Full Documentation**: `.kiro/specs/message-lifecycle-rebuild/TASK_14.5_EXECUTION_SUMMARY.md`
- **Load Test Guide**: `LOAD_TEST_QUICK_START.md`
- **k6 Documentation**: https://k6.io/docs/
- **Performance Testing Best Practices**: https://k6.io/docs/testing-guides/

## 🎯 Success Criteria

Stress test is successful when:
- ✅ System handles 10,000+/min initially
- ✅ Breaking point identified
- ✅ Bottlenecks discovered
- ✅ Failure modes understood
- ✅ Optimization plan created
- ✅ Capacity limits documented

## 📝 Next Steps

After completing stress test:

1. **Document findings**:
   - Maximum sustainable throughput
   - Identified bottlenecks
   - Failure modes
   - Resource limits

2. **Create optimization plan**:
   - Prioritize improvements
   - Estimate impact
   - Plan implementation

3. **Update capacity planning**:
   - Set production limits (80% of max)
   - Plan scaling strategy
   - Define monitoring alerts

4. **Schedule regular testing**:
   - Monthly stress tests
   - After major changes
   - Before scaling events

## 🆘 Troubleshooting

### k6 not found
```powershell
choco install k6
```

### Backend not accessible
```powershell
docker-compose up -d
curl http://localhost:8080/health
```

### Test fails immediately
- Check Docker containers are running
- Verify database is accessible
- Check Redis is running
- Review backend logs

### High error rate from start
- System may already be under load
- Check for pending messages in queue
- Restart infrastructure
- Verify system health

### System crashes during test
- This indicates a critical issue
- Review crash logs
- Check for memory leaks
- Verify database stability
- Consider reducing load

## 📞 Support

For issues or questions:
1. Check full documentation in `TASK_14.5_EXECUTION_SUMMARY.md`
2. Review backend logs: `docker-compose logs backend`
3. Check database logs: `docker-compose logs postgres`
4. Review Redis logs: `docker-compose logs redis`

---

**Remember**: Stress testing is about finding limits, not proving perfection. Failures are valuable learning opportunities! 🎓

