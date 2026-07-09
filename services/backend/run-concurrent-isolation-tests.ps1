#!/usr/bin/env pwsh

# Concurrent Isolation Tests Runner
# Tests tenant isolation under high concurrency loads

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Concurrent Isolation Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
Write-Host "Checking Go installation..." -ForegroundColor Yellow
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Go is installed" -ForegroundColor Green
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker..." -ForegroundColor Yellow
try {
    $dockerStatus = docker ps 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Docker is not running" -ForegroundColor Red
        Write-Host "Please start Docker Desktop and try again" -ForegroundColor Yellow
        exit 1
    }
    Write-Host "✅ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not available" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Check if PostgreSQL container is running
Write-Host "Checking PostgreSQL..." -ForegroundColor Yellow
$postgresRunning = docker ps --filter "name=postgres" --format "{{.Names}}" | Select-String "postgres"
if (-not $postgresRunning) {
    Write-Host "❌ PostgreSQL container is not running" -ForegroundColor Red
    Write-Host "Starting infrastructure..." -ForegroundColor Yellow
    docker-compose up -d postgres
    Start-Sleep -Seconds 5
}
Write-Host "✅ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Check if Redis container is running
Write-Host "Checking Redis..." -ForegroundColor Yellow
$redisRunning = docker ps --filter "name=redis" --format "{{.Names}}" | Select-String "redis"
if (-not $redisRunning) {
    Write-Host "❌ Redis container is not running" -ForegroundColor Red
    Write-Host "Starting infrastructure..." -ForegroundColor Yellow
    docker-compose up -d redis
    Start-Sleep -Seconds 3
}
Write-Host "✅ Redis is running" -ForegroundColor Green
Write-Host ""

# Set environment variables
Write-Host "Setting environment variables..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
$env:REDIS_URL = "localhost:6379"
$env:REDIS_PASSWORD = ""
$env:REDIS_DB = "1"
$env:GO_ENV = "test"
Write-Host "✅ Environment configured" -ForegroundColor Green
Write-Host ""

# Run tests
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Running Concurrent Isolation Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Test Suite: Concurrent Isolation" -ForegroundColor Magenta
Write-Host "Tests: 6 concurrent isolation scenarios" -ForegroundColor Gray
Write-Host "Duration: ~15 seconds" -ForegroundColor Gray
Write-Host ""

# Run with race detector and verbose output
Write-Host "Running tests with race detector..." -ForegroundColor Yellow
Write-Host ""

$testOutput = go test -race -v -timeout 10m ./internal/security -run "TestConcurrent" 2>&1

# Display output
$testOutput | ForEach-Object {
    $line = $_.ToString()
    
    if ($line -match "PASS") {
        Write-Host $line -ForegroundColor Green
    }
    elseif ($line -match "FAIL") {
        Write-Host $line -ForegroundColor Red
    }
    elseif ($line -match "RUN") {
        Write-Host $line -ForegroundColor Cyan
    }
    elseif ($line -match "WARNING: DATA RACE") {
        Write-Host $line -ForegroundColor Red
    }
    else {
        Write-Host $line
    }
}

# Check exit code
if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  ✅ All Tests Passed!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Test Results:" -ForegroundColor Cyan
    Write-Host "  • Concurrent Operations: ✅ PASS" -ForegroundColor Green
    Write-Host "  • Balance Operations: ✅ PASS" -ForegroundColor Green
    Write-Host "  • Data Isolation: ✅ PASS" -ForegroundColor Green
    Write-Host "  • RLS Enforcement: ✅ PASS" -ForegroundColor Green
    Write-Host "  • Cache Isolation: ✅ PASS" -ForegroundColor Green
    Write-Host "  • Transaction Isolation: ✅ PASS" -ForegroundColor Green
    Write-Host ""
    Write-Host "Validation:" -ForegroundColor Cyan
    Write-Host "  ✅ No race conditions detected" -ForegroundColor Green
    Write-Host "  ✅ No data leakage between tenants" -ForegroundColor Green
    Write-Host "  ✅ All operations completed successfully" -ForegroundColor Green
    Write-Host "  ✅ Data integrity maintained" -ForegroundColor Green
    Write-Host ""
    Write-Host "Requirements Validated:" -ForegroundColor Cyan
    Write-Host "  ✅ Requirement 26.5: Multi-tenant isolation" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Yellow
    Write-Host "  1. Review test output above" -ForegroundColor Gray
    Write-Host "  2. Check for any warnings" -ForegroundColor Gray
    Write-Host "  3. Mark task 34.6 as complete" -ForegroundColor Gray
    Write-Host ""
    exit 0
}
else {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "  ❌ Tests Failed" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Troubleshooting:" -ForegroundColor Yellow
    Write-Host "  1. Check test output above for errors" -ForegroundColor Gray
    Write-Host "  2. Verify database is accessible" -ForegroundColor Gray
    Write-Host "  3. Verify Redis is accessible" -ForegroundColor Gray
    Write-Host "  4. Check for race conditions" -ForegroundColor Gray
    Write-Host "  5. Review CONCURRENT_ISOLATION_TEST_GUIDE.md" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Run individual tests:" -ForegroundColor Yellow
    Write-Host "  go test -v -run TestConcurrentTenantOperations ./internal/security" -ForegroundColor Gray
    Write-Host "  go test -v -run TestConcurrentBalanceOperations ./internal/security" -ForegroundColor Gray
    Write-Host "  go test -v -run TestConcurrentDataIsolation ./internal/security" -ForegroundColor Gray
    Write-Host ""
    exit 1
}
