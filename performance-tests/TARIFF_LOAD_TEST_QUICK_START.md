# Tariff Load Tests - Quick Start

## What This Tests

Three critical load scenarios for the tariff pricing system:

1. **1000 Concurrent Subscriptions** (Requirement 27.1)
2. **10000 Concurrent Balance Deductions** (Requirement 27.2)
3. **100 Concurrent Dedicated Gateways** (Requirement 27.3)

## Prerequisites

```powershell
# Install k6
choco install k6

# Ensure backend is running
cd services/backend
go run main.go
```

## Run Tests

```powershell
cd performance-tests
.\run-tariff-load-test.ps1
```

## Expected Duration

~22 minutes total:
- Test 1: 5 minutes (subscriptions)
- Test 2: 10 minutes (balance deductions)
- Test 3: 5 minutes (gateways)

## Success Criteria

✅ **All tests must achieve:**
- 95%+ success rate
- Meet latency targets
- No critical errors

## Results Location

```
performance-tests/results/
├── tariff-load-test-summary.json    # Key metrics
├── tariff-load-test-full.json       # Complete data
└── tariff-load-test-report.html     # Visual report
```

## Quick Interpretation

### Console Output

Look for these indicators:

```
✅ Test 1: 950+ subscriptions created
✅ Test 2: 9500+ deductions completed
✅ Test 3: 95+ gateways created
✅ All Targets: ACHIEVED
```

### Common Issues

**Low success rate?**
- Check database connection pool
- Verify tenant balances
- Review backend logs

**High latency?**
- Add database indexes
- Optimize queries
- Scale resources

**HTTP errors?**
- Verify admin token
- Check rate limits
- Review API logs

## Next Steps

1. Review HTML report for detailed visualizations
2. Compare with previous test runs
3. Document any performance issues
4. Optimize based on findings

## Full Documentation

See [TARIFF_LOAD_TEST_GUIDE.md](./TARIFF_LOAD_TEST_GUIDE.md) for complete details.
