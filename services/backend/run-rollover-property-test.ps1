#!/usr/bin/env pwsh

Write-Host "=== Rollover Manager Property Tests ===" -ForegroundColor Cyan
Write-Host ""

# Set test database URL
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"

# Navigate to rollover directory
Push-Location services/backend/internal/rollover

try {
    Write-Host "Running Property Tests..." -ForegroundColor Yellow
    Write-Host ""
    
    # Run property tests with 100 iterations
    go test -v -run TestProperty -rapid.checks=100
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✓ All property tests passed!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Property 9 validated:" -ForegroundColor Cyan
        Write-Host "  - Returns 0 when rollover is disabled" -ForegroundColor Gray
        Write-Host "  - Returns 0 when package is unlimited" -ForegroundColor Gray
        Write-Host "  - Returns 0 when messages_remaining <= 0" -ForegroundColor Gray
        Write-Host "  - Returns correct messages_remaining when rollover enabled" -ForegroundColor Gray
        Write-Host "  - Idempotent (multiple calls return same result)" -ForegroundColor Gray
        Write-Host "  - Handles boundary values correctly" -ForegroundColor Gray
    } else {
        Write-Host ""
        Write-Host "✗ Property tests failed!" -ForegroundColor Red
        Write-Host ""
        Write-Host "Troubleshooting:" -ForegroundColor Yellow
        Write-Host "  1. Check PostgreSQL is running: pg_isready" -ForegroundColor Gray
        Write-Host "  2. Check test database exists: psql -U postgres -l | grep sms_gateway_test" -ForegroundColor Gray
        Write-Host "  3. Run migrations: cd services/backend && go run cmd/migrate/main.go" -ForegroundColor Gray
        exit 1
    }
} finally {
    Pop-Location
}
