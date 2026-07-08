#!/usr/bin/env pwsh
# PowerShell script to run End-to-End integration tests
# Usage: .\run-e2e-test.ps1

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  End-to-End Integration Test Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker status..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
    Write-Host ""
    Write-Host "To start infrastructure:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d postgres redis" -ForegroundColor White
    exit 1
}
Write-Host "✓ Docker is running" -ForegroundColor Green
Write-Host ""

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL..." -ForegroundColor Yellow
$pgRunning = docker ps --filter "name=postgres" --filter "status=running" -q
if (-not $pgRunning) {
    Write-Host "❌ PostgreSQL is not running." -ForegroundColor Red
    Write-Host ""
    Write-Host "To start PostgreSQL:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d postgres" -ForegroundColor White
    exit 1
}
Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Check if Redis is running
Write-Host "Checking Redis..." -ForegroundColor Yellow
$redisRunning = docker ps --filter "name=redis" --filter "status=running" -q
if (-not $redisRunning) {
    Write-Host "❌ Redis is not running." -ForegroundColor Red
    Write-Host ""
    Write-Host "To start Redis:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d redis" -ForegroundColor White
    exit 1
}
Write-Host "✓ Redis is running" -ForegroundColor Green
Write-Host ""

# Wait for services to be ready
Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 2
Write-Host "✓ Services ready" -ForegroundColor Green
Write-Host ""

# Run migrations if needed
Write-Host "Checking database migrations..." -ForegroundColor Yellow
Write-Host "(Skipping - assuming migrations are already applied)" -ForegroundColor Gray
Write-Host ""

# Set test environment variables
$env:TEST_DB_HOST = "localhost"
$env:TEST_DB_PORT = "5432"
$env:TEST_DB_USER = "postgres"
$env:TEST_DB_PASSWORD = "postgres"
$env:TEST_DB_NAME = "sms_gateway_test"
$env:TEST_REDIS_HOST = "localhost"
$env:TEST_REDIS_PORT = "6379"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Running End-to-End Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Run the integration tests
Write-Host "Executing: go test -v ./internal/integration/..." -ForegroundColor Yellow
Write-Host ""

go test -v -timeout 5m ./internal/integration/...

$exitCode = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($exitCode -eq 0) {
    Write-Host "✅ All tests passed!" -ForegroundColor Green
} else {
    Write-Host "❌ Some tests failed!" -ForegroundColor Red
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

exit $exitCode
