# Script to run concurrent operations tests
# Usage: .\run-concurrent-test.ps1

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Concurrent Operations Test Runner" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Check if database is running
Write-Host "Checking database connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$dbCheck = psql -h localhost -U postgres -d alhudhud_test -c "SELECT 1" 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Database 'alhudhud_test' not found or not accessible" -ForegroundColor Red
    Write-Host ""
    Write-Host "Creating test database..." -ForegroundColor Yellow
    psql -h localhost -U postgres -c "CREATE DATABASE alhudhud_test" 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Test database created successfully" -ForegroundColor Green
    } else {
        Write-Host "❌ Failed to create test database" -ForegroundColor Red
        Write-Host "Please ensure PostgreSQL is running and accessible" -ForegroundColor Yellow
        exit 1
    }
}

Write-Host "✅ Database connection OK" -ForegroundColor Green
Write-Host ""

# Run migrations on test database
Write-Host "Running migrations on test database..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/alhudhud_test?sslmode=disable"

# Apply migrations
Get-ChildItem -Path "migrations" -Filter "*.sql" | Sort-Object Name | ForEach-Object {
    Write-Host "  Applying: $($_.Name)" -ForegroundColor Gray
    Get-Content $_.FullName | psql -h localhost -U postgres -d alhudhud_test 2>&1 | Out-Null
}

Write-Host "✅ Migrations applied" -ForegroundColor Green
Write-Host ""

# Run the concurrent tests
Write-Host "Running concurrent operations tests..." -ForegroundColor Yellow
Write-Host ""

# Test 1: Basic concurrent deductions
Write-Host "Test 1: 100 Concurrent Balance Deductions" -ForegroundColor Cyan
go test -v ./internal/balance -run TestConcurrentBalanceDeductions -timeout 5m

Write-Host ""
Write-Host "Test 2: Mixed Concurrent Operations" -ForegroundColor Cyan
go test -v ./internal/balance -run TestConcurrentMixedOperations -timeout 5m

Write-Host ""
Write-Host "Test 3: High Contention Scenario" -ForegroundColor Cyan
go test -v ./internal/balance -run TestHighContentionScenario -timeout 5m

Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "All tests completed!" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Cyan
