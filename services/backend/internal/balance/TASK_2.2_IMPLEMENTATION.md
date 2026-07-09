# Task 2.2 Implementation: Property Test for Atomic Balance Operations

## Overview

This document describes the implementation of Task 2.2 from the tariff-pricing-system spec: **Property Test for Atomic Balance Operations**.

## Property 15: Atomic Balance Operations

**Validates: Requirements 16.1, 16.2**

### Property Statement (Arabic)

لأي عمليتي خصم متزامنتين على نفس الحساب، يجب أن تنجح واحدة فقط إذا كان الرصيد كافياً لواحدة فقط، ويجب ألا يصبح الرصيد سالباً أبداً

### Property Statement (English)

For any two concurrent deduction operations on the same account, exactly one should succeed if the balance is sufficient for only one, and the balance must never become negative.

## Implementation

### File: `manager_property_test.go`

The property test implementation includes 5 comprehensive test cases:

#### 1. TestProperty_AtomicBalanceOperations

**Purpose**: Main property test validating atomic balance deduction correctness

**What it tests**:
- Concurrent balance deductions on the same account
- Final balance equals initial balance minus successful deductions
- Balance never goes negative
- All successful operations are logged in balance_history

**Test parameters**:
- 10 iterations
- 50 concurrent operations per iteration
- Random amounts between 10 and 500
- Random initial balance between 10,000 and 20,000

**Properties verified**:
```
Property 1: Final Balance = Initial Balance - Sum(Successful Deductions)
Property 2: Balance >= 0 (never negative)
Property 3: History Count = Success Count
```

#### 2. TestProperty_NoRaceConditionsInBalanceOperations

**Purpose**: Detect race conditions under high concurrency

**What it tests**:
- 100 concurrent operations with the same amount
- Balance accuracy under concurrent load
- No race conditions in atomic updates

**Test parameters**:
- 5 iterations
- 100 concurrent operations per iteration
- Fixed amount: 100.0
- Large initial balance: 50,000

**Properties verified**:
```
Property: Balance = Initial - (Success Count × Amount)
```

#### 3. TestProperty_ExactlyOneSucceedsWhenBalanceSufficientForOne

**Purpose**: Critical property - exactly one operation succeeds when balance is sufficient for only one

**What it tests**:
- Balance set to exactly the amount needed for one operation
- Multiple concurrent operations attempt to deduct
- Exactly one should succeed
- Final balance should be zero
- Balance never goes negative

**Test parameters**:
- 20 iterations
- 10 concurrent operations per iteration
- Initial balance = operation amount (100.0)

**Properties verified**:
```
Property 1: Success Count = 1 (exactly one)
Property 2: Final Balance = 0
Property 3: Balance >= 0 (never negative)
```

#### 4. TestProperty_AllSuccessfulOperationsAreLogged

**Purpose**: Verify audit trail completeness

**What it tests**:
- Every successful operation is recorded in balance_history
- No operations are lost or duplicated in the log

**Test parameters**:
- 10 iterations
- 20 concurrent operations per iteration
- Random amounts between 50 and 150

**Properties verified**:
```
Property: History Count = Success Count
```

#### 5. TestProperty_BalanceNeverGoesNegative

**Purpose**: Stress test to ensure balance constraint under extreme conditions

**What it tests**:
- Small initial balance with many concurrent operations
- High pressure scenario to test constraint enforcement
- Balance never goes negative under any circumstances

**Test parameters**:
- 15 iterations
- 100 concurrent operations per iteration
- Small initial balance: 500-1000
- Random amounts: 10-100

**Properties verified**:
```
Property: Balance >= 0 (CRITICAL - never negative)
```

## Requirements Validation

### Requirement 16.1
**"THE النظام SHALL يسجل كل عملية مالية في جدول المعاملات"**

✅ Validated by:
- TestProperty_AtomicBalanceOperations (checks history count)
- TestProperty_AllSuccessfulOperationsAreLogged (dedicated test)

### Requirement 16.2
**"THE النظام SHALL يسجل نوع المعاملة (اشتراك، توريد، خصم، استرجاع)"**

✅ Validated by:
- All tests use OperationType in BalanceOperation
- History logger records operation type

## Running the Tests

### Quick Run

```bash
cd services/backend
go test -v ./internal/balance -run TestProperty_AtomicBalanceOperations
```

### Run All Property Tests for Task 2.2

```bash
cd services/backend
./run-property-test-task-2.2.ps1
```

Or manually:

```bash
go test -v ./internal/balance -run "TestProperty_AtomicBalanceOperations|TestProperty_NoRaceConditionsInBalanceOperations|TestProperty_ExactlyOneSucceedsWhenBalanceSufficientForOne|TestProperty_AllSuccessfulOperationsAreLogged|TestProperty_BalanceNeverGoesNegative"
```

### With Race Detector

```bash
go test -v -race ./internal/balance -run TestProperty
```

### Stress Test (Multiple Iterations)

```bash
go test -v ./internal/balance -run TestProperty -count=50
```

## Expected Results

### Success Output

```
=== RUN   TestProperty_AtomicBalanceOperations
=== RUN   TestProperty_AtomicBalanceOperations/Iteration_1
    manager_property_test.go:89: Iteration 1: Initial=15234.00, Final=12456.78, Deducted=2777.22, Success=45/50
=== RUN   TestProperty_AtomicBalanceOperations/Iteration_2
    manager_property_test.go:89: Iteration 2: Initial=18765.00, Final=15123.45, Deducted=3641.55, Success=48/50
...
--- PASS: TestProperty_AtomicBalanceOperations (2.34s)
PASS
```

### Failure Indicators

If any of these properties are violated, the test will fail with a clear message:

1. **Balance Mismatch**: "Property violated: Final balance should equal initial balance minus successful deductions"
2. **Negative Balance**: "Property violated: Balance should never be negative"
3. **Missing Logs**: "Property violated: History count should match successful deductions"
4. **Race Condition**: "Race condition detected: Balance mismatch"
5. **Multiple Successes**: "Property violated: Exactly one operation should succeed"

## Database Requirements

The tests require:
- PostgreSQL database named `sms_gateway_test`
- Connection: `postgres://postgres:postgres@localhost:5432/sms_gateway_test`
- Tables: `tenants`, `balance_history`
- Migrations must be applied

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Run Property Tests - Task 2.2
  run: |
    cd services/backend
    go test -v -race ./internal/balance -run TestProperty -timeout 5m
```

### GitLab CI

```yaml
test:property:task-2.2:
  script:
    - cd services/backend
    - go test -v -race ./internal/balance -run TestProperty -timeout 5m
```

## Key Implementation Details

### Atomic Operations

The tests validate that the balance manager uses:
- SERIALIZABLE transaction isolation level
- Atomic UPDATE with WHERE condition: `WHERE balance >= amount`
- Retry logic with exponential backoff for concurrent updates

### Concurrency Safety

The tests use:
- `sync.WaitGroup` for coordinating goroutines
- `sync.Mutex` for counting successes
- Random delays to increase race condition likelihood

### Test Data Isolation

Each test iteration:
- Creates a new tenant with a fresh balance
- Uses unique message IDs
- Cleans up after completion

## Troubleshooting

### Test Fails Intermittently

This may indicate a real race condition. Run with more iterations:

```bash
go test -v ./internal/balance -run TestProperty -count=100
```

### Test Times Out

Increase timeout:

```bash
go test -v ./internal/balance -run TestProperty -timeout 10m
```

### Database Connection Error

Ensure PostgreSQL is running and the test database exists:

```bash
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
```

## References

- Spec: `.kiro/specs/tariff-pricing-system/tasks.md` (Task 2.2)
- Design: `.kiro/specs/tariff-pricing-system/design.md` (Property 15)
- Requirements: `.kiro/specs/tariff-pricing-system/requirements.md` (16.1, 16.2)
- Implementation: `services/backend/internal/balance/manager.go`
- Tests: `services/backend/internal/balance/manager_property_test.go`

## Status

✅ **Task 2.2 Completed**

- [x] إنشاء ملف internal/balance/manager_property_test.go
- [x] اختبار عمليات متزامنة على نفس الحساب
- [x] التحقق من عدم وجود Race Conditions
- [x] التحقق من أن واحدة فقط تنجح عندما يكفي الرصيد لواحدة
- [x] التحقق من عدم سلبية الرصيد أبداً
- [x] التحقق من تسجيل جميع العمليات الناجحة

**Date Completed**: 2025-01-XX
**Validated Requirements**: 16.1, 16.2
**Property Validated**: Property 15 - Atomic Balance Operations
