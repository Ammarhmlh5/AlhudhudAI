# Property Test Quick Start - Rollover Manager

## Overview

This guide helps you run the property-based tests for the Rollover Manager.

## Property 9: Calculate Remaining Messages

**Validates:** Requirements 6.2

**What it tests:**
- Returns 0 when rollover is disabled
- Returns 0 when package is unlimited
- Returns 0 when messages_remaining is 0 or negative
- Returns correct messages_remaining when rollover is enabled
- Idempotency (multiple calls return same result)
- Boundary values (0, 1, max int64)

## Prerequisites

1. PostgreSQL test database running
2. Go 1.21+ installed
3. Required Go packages:
   ```bash
   go get pgregory.net/rapid
   go get github.com/stretchr/testify
   ```

## Setup Test Database

```bash
# Create test database
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"

# Run migrations on test database
cd services/backend
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
go run cmd/migrate/main.go
```

## Running the Tests

### Run all property tests
```bash
cd services/backend/internal/rollover
go test -v -run TestProperty
```

### Run specific property test
```bash
go test -v -run TestProperty_CalculateRemainingMessages
```

### Run with more iterations (default is 100)
```bash
go test -v -run TestProperty -rapid.checks=1000
```

### Run in short mode (skip property tests)
```bash
go test -v -short
```

## PowerShell Script

Create `services/backend/run-rollover-property-test.ps1`:

```powershell
#!/usr/bin/env pwsh

Write-Host "=== Rollover Manager Property Tests ===" -ForegroundColor Cyan
Write-Host ""

# Set test database URL
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"

# Navigate to rollover directory
Set-Location services/backend/internal/rollover

Write-Host "Running Property Tests..." -ForegroundColor Yellow
go test -v -run TestProperty -rapid.checks=100

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✓ All property tests passed!" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "✗ Property tests failed!" -ForegroundColor Red
    exit 1
}
```

## Expected Output

```
=== RUN   TestProperty_CalculateRemainingMessages
--- PASS: TestProperty_CalculateRemainingMessages (2.34s)
=== RUN   TestProperty_CalculateRemainingMessages_Idempotency
--- PASS: TestProperty_CalculateRemainingMessages_Idempotency (1.87s)
=== RUN   TestProperty_CalculateRemainingMessages_BoundaryValues
=== RUN   TestProperty_CalculateRemainingMessages_BoundaryValues/Zero_messages
=== RUN   TestProperty_CalculateRemainingMessages_BoundaryValues/One_message
=== RUN   TestProperty_CalculateRemainingMessages_BoundaryValues/Maximum_int64
--- PASS: TestProperty_CalculateRemainingMessages_BoundaryValues (0.45s)
PASS
ok      github.com/yourusername/sms-gateway/services/backend/internal/rollover  4.66s
```

## Understanding Property Tests

### What is Property-Based Testing?

Instead of testing specific examples, property tests verify general properties that should always hold true:

**Example-Based Test:**
```go
// Test specific case
assert.Equal(t, 100, CalculateRemaining(subscriptionWithRollover))
```

**Property-Based Test:**
```go
// Test general property
for any subscription with rollover enabled:
    if messages_remaining > 0:
        result == messages_remaining
    else:
        result == 0
```

### Benefits

1. **Finds edge cases** you didn't think of
2. **Tests many scenarios** automatically (100+ per run)
3. **Verifies invariants** that should always hold
4. **Catches regressions** early

## Troubleshooting

### Test database connection fails
```bash
# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# Check database exists
psql -U postgres -l | grep sms_gateway_test
```

### Migrations not applied
```bash
# Run migrations manually
cd services/backend
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
go run cmd/migrate/main.go
```

### Tests fail with "relation does not exist"
```bash
# Drop and recreate test database
psql -U postgres -c "DROP DATABASE IF EXISTS sms_gateway_test;"
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
# Run migrations again
```

## Next Steps

After Property 9 passes:
1. Implement RolloverMessages (Task 8.3)
2. Write Property 10 for rollover limit (Task 8.4)
3. Implement UseRolloverMessages (Task 8.5)
4. Write Property 11 for FIFO (Task 8.6)

## References

- [rapid documentation](https://pkg.go.dev/pgregory.net/rapid)
- [Property-Based Testing Guide](https://hypothesis.works/articles/what-is-property-based-testing/)
- Requirements 6.2: Calculate remaining messages at end of month
