#!/usr/bin/env pwsh
# Database Failure Recovery Test Runner
# This script runs the database failure recovery integration test

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Database Failure Recovery Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
    exit 1
}
Write-Host "✓ Docker is running" -ForegroundColor Green
Write-Host ""

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL..." -ForegroundColor Yellow
$postgresRunning = docker ps --filter "name=postgres" --filter "status=running" --format "{{.Names}}"
if (-not $postgresRunning) {
    Write-Host "❌ PostgreSQL is not running. Starting PostgreSQL..." -ForegroundColor Yellow
    docker-compose up -d postgres
    Write-Host "Waiting for PostgreSQL to be ready..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
}
Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Check if database exists
Write-Host "Checking database..." -ForegroundColor Yellow
$dbExists = docker exec postgres psql -U postgres -lqt | Select-String -Pattern "sms_gateway_test"
if (-not $dbExists) {
    Write-Host "Creating test database..." -ForegroundColor Yellow
    docker exec postgres psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
}
Write-Host "✓ Database exists" -ForegroundColor Green
Write-Host ""

# Run migrations
Write-Host "Running migrations..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
if (Test-Path "migrations") {
    # Check if migrate tool is available
    $migrateExists = Get-Command migrate -ErrorAction SilentlyContinue
    if ($migrateExists) {
        migrate -path migrations -database $env:DATABASE_URL up
        Write-Host "✓ Migrations applied" -ForegroundColor Green
    } else {
        Write-Host "⚠ migrate tool not found. Skipping migrations." -ForegroundColor Yellow
        Write-Host "  Install: https://github.com/golang-migrate/migrate" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠ migrations folder not found. Skipping migrations." -ForegroundColor Yellow
}
Write-Host ""

# Run the test
Write-Host "Running Database Failure Recovery Test..." -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

go test -v ./internal/integration -run TestDatabaseFailureRecovery -timeout 5m

$testResult = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($testResult -eq 0) {
    Write-Host "✅ Test PASSED" -ForegroundColor Green
    Write-Host ""
    Write-Host "All database failure recovery scenarios passed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Validated:" -ForegroundColor Cyan
    Write-Host "  ✓ Primary database failure detection" -ForegroundColor Green
    Write-Host "  ✓ Automatic failover to backup database" -ForegroundColor Green
    Write-Host "  ✓ System continues operation without data loss" -ForegroundColor Green
    Write-Host "  ✓ Automatic reconnection to primary when available" -ForegroundColor Green
    Write-Host "  ✓ Connection pool behavior during failure" -ForegroundColor Green
} else {
    Write-Host "❌ Test FAILED" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please check the error messages above." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Common issues:" -ForegroundColor Yellow
    Write-Host "  1. PostgreSQL not running: docker-compose up -d postgres" -ForegroundColor Yellow
    Write-Host "  2. Database not created: docker exec postgres psql -U postgres -c 'CREATE DATABASE sms_gateway_test;'" -ForegroundColor Yellow
    Write-Host "  3. Migrations not applied: Run migrations first" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "For more help, see:" -ForegroundColor Yellow
    Write-Host "  - internal/integration/DATABASE_FAILURE_TEST_QUICK_START.md" -ForegroundColor Cyan
    Write-Host "  - internal/integration/DATABASE_FAILURE_TEST_GUIDE.md" -ForegroundColor Cyan
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

exit $testResult
