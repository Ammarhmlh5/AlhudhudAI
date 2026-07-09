# Load Test Quick Start Guide

## 🚀 Quick Start (5 Minutes)

### Prerequisites
- ✅ k6 installed
- ✅ Docker running
- ✅ Backend service running

### Run Load Test

```powershell
cd performance-tests
.\run-load-test.ps1
```

That's it! The script will:
1. Check prerequisites
2. Start resource monitoring
3. Run k6 load test (10 minutes)
4. Display results
5. Generate reports

## 📊 What Gets Tested

### Throughput (Requirement 13.6)
- **Target**: 5000+ messages/minute
- **Test**: Sends 50,000+ messages over 10 minutes
- **Success**: >= 5000 messages/minute

### End-to-End Latency (Requirement 21.6)
- **Target**: P95 < 2000ms
- **Test**: Measures time from API submission to delivery
- **Success**: 95% of messages delivered in < 2 seconds

### API Response Time (Requirement 3.1)
- **Target**: P95 < 10ms
- **Test**: Measures API POST /api/messages response time
- **Success**: 95% of API calls respond in < 10ms

### Success Rate
- **Target**: >= 99%
- **Test**: Tracks delivered vs failed messages
- **Success**: >= 99% messages delivered successfully

### Resource Consumption
- **CPU**: Backend < 80% average
- **Memory**: Backend < 2GB
- **Database**: < 100 connections
- **Redis**: Stable memory usage

## 📁 Results Location

After the test completes, check:
- `results/load-test-summary.json` - Key metrics
- `results/load-test-report.html` - Visual report
- `results/resource-monitoring.csv` - Resource data
- `results/resource-monitoring-summary.json` - Resource summary

## ✅ Success Criteria

The test passes if:
- ✅ Throughput >= 5000 messages/minute
- ✅ Success rate >= 99%
- ✅ P95 latency < 2000ms
- ✅ P95 API response < 10ms
- ✅ No critical errors

## 🔧 Troubleshooting

### k6 not found
```powershell
choco install k6
```

### Docker not running
```powershell
# Start Docker Desktop
```

### Backend not accessible
```powershell
# Check if backend is running
curl http://localhost:8080/health

# Start backend
cd services/backend
go run main.go
```

### Low throughput
- Increase worker pool size
- Increase database connection pool
- Optimize batch processing
- Check resource constraints

### High latency
- Check queue size
- Optimize database queries
- Check WebSocket performance
- Monitor Mobile Gateway

## 📖 Full Documentation

For detailed documentation, see:
- `.kiro/specs/message-lifecycle-rebuild/TASK_14.4_LOAD_TESTING.md`

## 🎯 Next Steps

1. Run the test
2. Review results
3. Identify bottlenecks
4. Apply optimizations
5. Re-run to verify improvements
