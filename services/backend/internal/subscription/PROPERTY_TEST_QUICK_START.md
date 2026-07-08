# Property Test Quick Start - Subscription Manager

## Overview

This guide helps you run property-based tests for the Subscription Manager to verify correctness under various scenarios.

## Property 5: Safe Subscription

**Validates Requirements:** 4.1, 4.2

**What it tests:**
- Subscription succeeds only when balance is sufficient
- Balance is deducted atomically
- Subscription is created correctly
- Accounting transaction is recorded
- No race conditions during concurrent subscriptions

## Prerequisites

1. **Test Database**: You need a test PostgreSQL database
2. **Go 1.21+**: Property testing requires Go 1.21 or higher
3. **Dependencies**: Install required packages

```bash
cd services/backend
go get pgregory.net/rapid
go get github.com/stretchr/testify
```

## Setup Test Database

### Option 1: Use Docker

```bash
# Start test database
docker run -d \
  --name alhudhud-test-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=alhudhud_test \
  -p 5433:5432 \
  postgres:15

# Wait for database to be ready
sleep 5

# Run migrations
export DATABASE_URL="postgres://postgres:postgres@localhost:5433/alhudhud_test?sslmode=disable"
cd services/backend
go run cmd/migrate/main.go up
```

### Option 2: Use Existing Database

```bash
# Create test database
createdb alhudhud_test

# Run migrations
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alhudhud_test?sslmode=disable"
cd services/backend
go run cmd/migrate/main.go up
```

## Running Property Tests

### Run All Property Tests

```bash
cd services/backend/internal/subscription
go test -v -run TestProperty
```

### Run Specific Property Test

```bash
# Test safe subscription
go test -v -run TestProperty_SafeSubscription

# Test atomic subscription
go test -v -run TestProperty_AtomicSubscription

# Test concurrent subscriptions
go test -v -run TestProperty_ConcurrentSubscriptions

# Test idempotency
go test -v -run TestProperty_SubscriptionIdempotency
```

### Run with More Iterations

```bash
# Run with 1000 iterations (default is 100)
go test -v -run TestProperty -rapid.checks=1000
```

### Run with Specific Seed

```bash
# Run with specific seed for reproducibility
go test -v -run TestProperty -rapid.seed=12345
```

## Understanding Test Output

### Successful Test

```
=== RUN   TestProperty_SafeSubscription
--- PASS: TestProperty_SafeSubscription (2.34s)
    rapid: 100 checks passed
```

### Failed Test

```
=== RUN   TestProperty_SafeSubscription
--- FAIL: TestProperty_SafeSubscription (1.23s)
    rapid: failed after 42 checks
    rapid: seed=1234567890
    rapid: counterexample:
        initial_balance=50
        package_price=60
    rapid: error: Balance should not change on failed subscription
```

If a test fails, you can reproduce it using the seed:

```bash
go test -v -run TestProperty_SafeSubscription -rapid.seed=1234567890
```

## Property Tests Explained

### 1. TestProperty_SafeSubscription

Tests the core subscription logic:

```go
// Generates random test cases:
// - initial_balance: 0-200
// - package_price: 10-100
//
// Verifies:
// - If balance >= price: subscription succeeds
// - If balance < price: subscription fails gracefully
// - Balance is deducted correctly
// - Subscription is created
```

**Example scenarios tested:**
- Balance=100, Price=50 → Success, Balance=50
- Balance=30, Price=50 → Fail, Balance=30 (unchanged)
- Balance=100, Price=100 → Success, Balance=0

### 2. TestProperty_AtomicSubscription

Tests atomicity of subscription:

```go
// Verifies:
// - Balance deduction is atomic
// - Subscription creation is atomic
// - Accounting transaction is atomic
// - All operations succeed or all fail
```

**What it checks:**
- Balance is exactly initial - price (no partial deductions)
- Subscription exists if and only if balance was deducted
- Accounting transaction exists if and only if subscription exists

### 3. TestProperty_ConcurrentSubscriptions

Tests concurrent subscription attempts:

```go
// Generates:
// - concurrency: 2-5 concurrent subscriptions
//
// Verifies:
// - No race conditions
// - Successful subscriptions <= balance allows
// - Final balance is correct
```

**Example scenario:**
- Initial balance: 100
- Package price: 10
- Concurrent attempts: 5
- Expected: Max 10 successful subscriptions
- Final balance: 0 (if all 10 succeed)

### 4. TestProperty_SubscriptionIdempotency

Tests multiple subscription attempts:

```go
// Verifies:
// - Each subscription gets unique ID
// - Balance is deducted for each subscription
// - Multiple subscriptions are allowed (not idempotent by design)
```

**Note:** Current implementation allows multiple subscriptions. This is by design - tenants can have multiple subscriptions (though typically only one is active).

## Troubleshooting

### Test Database Connection Failed

```
Test database not available
```

**Solution:**
1. Check database is running: `psql -h localhost -p 5433 -U postgres -d alhudhud_test`
2. Check connection string in test file
3. Run migrations: `go run cmd/migrate/main.go up`

### RLS Policy Errors

```
ERROR: permission denied for table packages
```

**Solution:**
1. Ensure RLS policies are created (run migrations)
2. Check tenant context is set correctly
3. Verify tenant exists in database

### Concurrent Test Failures

```
Balance should match expected after concurrent subscriptions
```

**Solution:**
1. This might indicate a race condition
2. Check SERIALIZABLE isolation is working
3. Verify atomic balance deduction logic
4. Run test multiple times to confirm

### Cleanup Errors

```
Cleanup warning: foreign key constraint violation
```

**Solution:**
- These are warnings, not failures
- Cleanup is best-effort
- Database will be cleaned up eventually

## Performance Benchmarks

Run benchmarks to measure performance:

```bash
cd services/backend/internal/subscription
go test -bench=. -benchmem
```

Expected performance:
- Subscribe: ~5-10ms per operation
- Concurrent subscriptions: ~20-50ms for 5 concurrent
- Property test (100 checks): ~2-5 seconds

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run Property Tests
  run: |
    cd services/backend/internal/subscription
    go test -v -run TestProperty -rapid.checks=1000
```

### Skip in Short Mode

Property tests are skipped in short mode:

```bash
# Skip property tests (for quick checks)
go test -short

# Run all tests including property tests
go test
```

## Next Steps

After property tests pass:

1. ✅ Task 7.2 complete
2. → Task 7.3: Implement DeductMessages
3. → Task 7.4: Property test for DeductMessages
4. → Task 7.5: Implement CalculateOverageCost
5. → Task 7.6: Property test for overage
6. → Task 7.7: Implement GetUsage

## References

- **Property Testing Library**: https://pkg.go.dev/pgregory.net/rapid
- **Requirements**: `.kiro/specs/tariff-pricing-system/requirements.md`
- **Design**: `.kiro/specs/tariff-pricing-system/design.md`
- **Implementation**: `manager.go`

## Support

If you encounter issues:

1. Check test database is running
2. Verify migrations are applied
3. Check RLS policies are enabled
4. Review test output for specific errors
5. Use `-rapid.seed` to reproduce failures

---

**Last Updated**: 2026-02-27  
**Property**: 5 - Safe Subscription  
**Requirements**: 4.1, 4.2


---

## Property 6: Message Deduction

**Validates Requirements:** 4.5

**What it tests:**

- `messages_remaining` decreases correctly when messages are deducted
- `messages_used` increases correctly when messages are deducted
- The sum of `messages_used + messages_remaining` equals the original quota (for limited packages)
- Concurrent deductions don't cause race conditions
- Deduction is atomic (either succeeds completely or fails completely)
- Unlimited packages handle deductions correctly (NULL remaining)

## Running Property 6 Tests

### Run All Message Deduction Tests

```bash
cd services/backend/internal/subscription

# Run all Property 6 tests
go test -v -run TestProperty_MessageDeduction
go test -v -run TestProperty_ConcurrentMessageDeduction
go test -v -run TestProperty_UnlimitedPackageDeduction
go test -v -run TestProperty_MessageDeductionAtomicity
go test -v -run TestProperty_MessageDeductionInvariant
```

### Run with PowerShell Script

```powershell
cd services/backend
.\run-message-deduction-test.ps1
```

This script runs all Property 6 tests and provides a nice summary.

## Property 6 Tests Explained

### 1. TestProperty_MessageDeduction

Tests basic message deduction properties:

```go
// Generates random test cases:
// - included_messages: 100-1000
// - message_count: 1-50
//
// Verifies:
// - If remaining >= count: deduction succeeds
// - If remaining < count: deduction fails (overage)
// - messages_remaining decreases by count
// - messages_used increases by count
// - Invariant: used + remaining = quota
```

**Example scenarios tested:**

- Quota=100, Used=0, Deduct=10 → Success, Used=10, Remaining=90
- Quota=100, Used=95, Deduct=10 → Fail (overage), Used=95, Remaining=5
- Quota=100, Used=50, Deduct=50 → Success, Used=100, Remaining=0

### 2. TestProperty_ConcurrentMessageDeduction

Tests concurrent message deductions:

```go
// Generates:
// - included_messages: 100
// - message_count: 5
// - concurrency: 5-15 concurrent deductions
//
// Verifies:
// - No race conditions
// - Successful deductions <= quota allows
// - Final messages_used is correct
// - Invariant holds after concurrent operations
```

**Example scenario:**

- Initial quota: 100
- Message count per deduction: 5
- Concurrent attempts: 15
- Expected: Max 20 successful deductions (100/5)
- Final state: Used=100, Remaining=0

### 3. TestProperty_UnlimitedPackageDeduction

Tests message deduction for unlimited packages:

```go
// Generates:
// - message_count: 1-100
//
// Verifies:
// - Deduction always succeeds for unlimited packages
// - messages_used increases correctly
// - messages_remaining stays NULL
```

**Example scenario:**

- Package: Unlimited (messages_remaining = NULL)
- Deduct: 50 messages
- Result: Success, Used increases by 50, Remaining stays NULL

### 4. TestProperty_MessageDeductionAtomicity

Tests atomicity of message deduction:

```go
// Verifies:
// - Both messages_used and messages_remaining update together
// - On success: used increases by count, remaining decreases by count
// - On failure: neither field changes
// - Increase in used equals decrease in remaining
```

**What it checks:**

- Atomic update: both fields change in same transaction
- No partial updates
- All-or-nothing behavior

### 5. TestProperty_MessageDeductionInvariant

Tests that the invariant always holds:

```go
// Generates:
// - Multiple deduction operations (3-10)
// - Random message counts for each
//
// Verifies:
// - Invariant holds after every single operation
// - used + remaining = quota (always)
```

**Example scenario:**

- Initial: Used=0, Remaining=100, Quota=100
- Deduct 10: Used=10, Remaining=90, Invariant: 10+90=100 ✓
- Deduct 20: Used=30, Remaining=70, Invariant: 30+70=100 ✓
- Deduct 15: Used=45, Remaining=55, Invariant: 45+55=100 ✓

## Understanding Property 6 Test Output

### Successful Test

```text
=== RUN   TestProperty_MessageDeduction
--- PASS: TestProperty_MessageDeduction (2.45s)
    rapid: 100 checks passed
```

### Failed Test Example

```text
=== RUN   TestProperty_MessageDeduction
--- FAIL: TestProperty_MessageDeduction (1.34s)
    rapid: failed after 56 checks
    rapid: seed=9876543210
    rapid: counterexample:
        included_messages=150
        message_count=25
        messages_used=130
        messages_remaining=20
    rapid: error: Invariant violated: 130 + 20 != 150
```

To reproduce:

```bash
go test -v -run TestProperty_MessageDeduction -rapid.seed=9876543210
```

## Key Properties Verified

### Property 1: Correct Deduction

```text
messages_remaining_after = messages_remaining_before - message_count
messages_used_after = messages_used_before + message_count
```

### Property 2: Invariant Preservation

```text
messages_used + messages_remaining = included_messages (always)
```

### Property 3: Atomicity

```text
If deduction succeeds:
  - Both fields update together
  - Increase in used = Decrease in remaining
  
If deduction fails:
  - Neither field changes
```

### Property 4: Concurrent Safety

```text
Concurrent deductions:
  - No race conditions
  - Total deducted <= quota
  - Invariant holds
```

### Property 5: Unlimited Handling

```text
For unlimited packages:
  - messages_remaining = NULL (always)
  - messages_used increases
  - Deduction always succeeds
```

## Troubleshooting Property 6 Tests

### Invariant Violation

```text
Invariant violated: used + remaining != quota
```

**Possible causes:**

1. Race condition in concurrent deductions
2. Non-atomic update of used/remaining
3. Missing WHERE clause in UPDATE
4. Transaction isolation level too low

**Solution:**

- Check SERIALIZABLE isolation level
- Verify atomic UPDATE statement
- Review concurrent test results

### Concurrent Test Failures

```text
Successful deductions exceed quota
```

**Possible causes:**

1. Race condition in balance check
2. Missing row-level locking
3. Incorrect WHERE condition

**Solution:**

- Use `WHERE messages_remaining >= $1` in UPDATE
- Verify SERIALIZABLE isolation
- Check for proper error handling

### Unlimited Package Issues

```text
messages_remaining should be NULL but got value
```

**Possible causes:**

1. Package setup incorrect
2. Wrong UPDATE query for unlimited
3. Migration issue

**Solution:**

- Verify package has `is_unlimited = true`
- Check `included_messages IS NULL`
- Review DeductMessages logic for unlimited

## Performance Expectations

Expected performance for Property 6 tests:

- DeductMessages: ~3-8ms per operation
- Concurrent deductions (10 concurrent): ~30-80ms
- Property test (100 checks): ~3-7 seconds
- All Property 6 tests: ~15-30 seconds

## Integration with Task 7.3

Property 6 tests verify the implementation from Task 7.3:

```go
// Task 7.3: DeductMessages implementation
func (m *manager) DeductMessages(ctx context.Context, req DeductMessagesRequest) (*DeductMessagesResult, error) {
    // 1. Start SERIALIZABLE transaction
    // 2. Get current subscription state
    // 3. Check if subscription is active
    // 4. Handle unlimited packages (NULL remaining)
    // 5. Handle limited packages with atomic UPDATE
    // 6. Detect overage condition
    // 7. Commit transaction
}
```

The property tests verify:

- ✅ Atomicity (SERIALIZABLE transaction)
- ✅ Correct state updates (used + remaining)
- ✅ Invariant preservation (used + remaining = quota)
- ✅ Concurrent safety (no race conditions)
- ✅ Unlimited package handling (NULL remaining)

## Next Steps After Property 6

After Property 6 tests pass:

1. ✅ Task 7.3 complete (DeductMessages implementation)
2. ✅ Task 7.4 complete (Property Test for DeductMessages)
3. → Task 7.5: Implement CalculateOverageCost
4. → Task 7.6: Property test for overage
5. → Task 7.7: Implement GetUsage

## References for Property 6

- **Requirements**: Requirement 4.5 - Message deduction from quota
- **Design**: Message deduction with atomic updates
- **Implementation**: `manager.go` - `DeductMessages` function
- **Tests**: `manager_property_test.go` - Property 6 tests

---

**Last Updated**: 2026-02-27  
**Properties**: 5 (Safe Subscription), 6 (Message Deduction)  
**Requirements**: 4.1, 4.2, 4.5
