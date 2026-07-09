# Overage Property Test Quick Start Guide

## Overview

This guide helps you quickly run the property-based tests for overage charging (Task 7.6).

---

## Prerequisites

1. **Go 1.21+** installed
2. **PostgreSQL** running on `localhost:5432`
3. **Test database** `alhudhud_test` created
4. **Migrations** applied to test database

---

## Quick Start

### 1. Run All Overage Property Tests

```powershell
cd services/backend
.\run-overage-property-test.ps1
```

### 2. Run Specific Test

```powershell
# Test basic overage charging
go test -v -run TestProperty_OverageCharging ./internal/subscription

# Test multiple charges
go test -v -run TestProperty_MultipleOverageCharges ./internal/subscription

# Test concurrent charges
go test -v -run TestProperty_ConcurrentOverageCharges ./internal/subscription

# Test unlimited packages
go test -v -run TestProperty_UnlimitedPackageNoOverage ./internal/subscription

# Test atomicity
go test -v -run TestProperty_OverageAtomicity ./internal/subscription

# Test invariant
go test -v -run TestProperty_OverageInvariant ./internal/subscription
```

### 3. Run with Race Detector

```powershell
go test -race -v -run TestProperty_ConcurrentOverageCharges ./internal/subscription
```

---

## What Gets Tested

### Property 8: خصم تكلفة الرسائل الإضافية من الرصيد

**Validates:** Requirements 5.3, 5.4

#### Test 1: Basic Overage Charging
- ✅ Cost calculated correctly: `cost = rate × messages`
- ✅ Balance checked before charging
- ✅ Balance deducted atomically when sufficient
- ✅ Transaction fails when balance insufficient
- ✅ Subscription updated correctly
- ✅ Accounting transaction recorded

#### Test 2: Multiple Overage Charges
- ✅ Multiple charges accumulate in `overage_messages`
- ✅ Multiple charges accumulate in `overage_cost`
- ✅ Balance decreases correctly
- ✅ Cumulative values match sum of operations

#### Test 3: Concurrent Overage Charges
- ✅ No race conditions
- ✅ Final balance matches expected
- ✅ Cumulative values match successful operations
- ✅ No lost updates or double charges

#### Test 4: Unlimited Package No Overage
- ✅ Zero cost for unlimited packages
- ✅ Balance not changed
- ✅ Subscription not updated
- ✅ No accounting transaction

#### Test 5: Overage Atomicity
- ✅ All-or-nothing updates
- ✅ On success: all changes applied
- ✅ On failure: no changes applied
- ✅ No partial state changes

#### Test 6: Overage Invariant
- ✅ Invariant holds: `cost = rate × messages`
- ✅ Holds after single operation
- ✅ Holds after multiple operations
- ✅ Rounding handled correctly

---

## Expected Output

```
========================================
Property-Based Tests for Overage Charging
Task 7.6: Property Test لخصم Overage
========================================

Test 1: Property_OverageCharging
[PASS] Property_OverageCharging

Test 2: Property_MultipleOverageCharges
[PASS] Property_MultipleOverageCharges

Test 3: Property_ConcurrentOverageCharges
[PASS] Property_ConcurrentOverageCharges

Test 4: Property_UnlimitedPackageNoOverage
[PASS] Property_UnlimitedPackageNoOverage

Test 5: Property_OverageAtomicity
[PASS] Property_OverageAtomicity

Test 6: Property_OverageInvariant
[PASS] Property_OverageInvariant

All property tests PASSED!
```

---

## Troubleshooting

### Database Connection Error

**Error:** `Cannot connect to test database`

**Solution:**
```powershell
# Check PostgreSQL is running
Get-Service postgresql*

# Create test database if not exists
psql -U postgres -c "CREATE DATABASE alhudhud_test;"

# Apply migrations
cd services/backend
go run cmd/migrate/main.go
```

### Test Timeout

**Error:** `test timed out after 5m`

**Solution:**
```powershell
# Increase timeout
go test -v -run TestProperty_OverageCharging ./internal/subscription -timeout 10m
```

### Race Condition Detected

**Error:** `WARNING: DATA RACE`

**Solution:**
- This indicates a real concurrency bug
- Review the implementation in `manager.go`
- Ensure `SERIALIZABLE` isolation level is used
- Check atomic operations

---

## Test Configuration

### Property Test Settings

```go
// Number of test cases per property
rapid.Check(t, func(t *rapid.T) {
    // Default: 100 test cases
    // Can be configured with -rapid.checks flag
})
```

### Custom Test Count

```powershell
# Run 1000 test cases per property
go test -v -run TestProperty_OverageCharging ./internal/subscription -rapid.checks=1000
```

---

## Understanding Property-Based Testing

### What is Property-Based Testing?

Instead of writing specific test cases, you define **properties** that should always hold:

**Traditional Test:**
```go
// Test specific case
balance := 100
overage := 10
rate := 0.05
expected := 0.50
```

**Property Test:**
```go
// Test property for ALL cases
∀ balance, overage, rate:
  IF balance >= (rate × overage)
  THEN charge succeeds
  AND balance_after = balance_before - (rate × overage)
```

### Benefits

1. **Automatic Edge Case Discovery**
   - Tests thousands of random scenarios
   - Finds edge cases you didn't think of

2. **Shrinking**
   - When a test fails, finds minimal failing case
   - Example: Fails with balance=5, shrinks to balance=0

3. **Confidence**
   - Verifies properties hold for ALL inputs
   - Not just specific examples

---

## Integration with Other Tests

### Task 7.4: Message Deduction Tests
```powershell
.\run-message-deduction-test.ps1
```

### Task 7.5: Overage Unit Tests
```powershell
go test -v -run TestCalculateOverageCost ./internal/subscription
```

### All Subscription Tests
```powershell
go test -v ./internal/subscription
```

---

## Next Steps

After verifying overage property tests pass:

1. **Task 7.7:** Implement `GetUsage` function
2. **Integration Testing:** Test complete message flow with overage
3. **Performance Testing:** Test overage under load

---

## Additional Resources

- **Property Test Guide:** `services/backend/internal/subscription/PROPERTY_TEST_QUICK_START.md`
- **Implementation Summary:** `.kiro/specs/tariff-pricing-system/TASK_7.6_IMPLEMENTATION_SUMMARY.md`
- **Requirements:** `.kiro/specs/tariff-pricing-system/requirements.md` (Requirement 5)
- **Design:** `.kiro/specs/tariff-pricing-system/design.md`

---

## Support

If you encounter issues:

1. Check database connection
2. Verify migrations are applied
3. Review test output for specific errors
4. Check implementation in `manager.go`
5. Consult implementation summary document

---

**Task 7.6 Status:** ✅ Completed  
**Property 8 Status:** ✅ Verified  
**Requirements 5.3, 5.4:** ✅ Validated
