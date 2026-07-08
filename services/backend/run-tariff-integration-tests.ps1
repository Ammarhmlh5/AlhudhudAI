#!/usr/bin/env pwsh
# Script to run Tariff System Integration Tests
# **Validates: All Requirements (1-42)**

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Tariff System Integration Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker status..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Docker is not running. Please start Docker first." -ForegroundColor Red
    exit 1
}
Write-Host "Docker is running." -ForegroundColor Green
Write-Host ""

# Check if test database exists
Write-Host "Checking test database..." -ForegroundColor Yellow
$dbExists = docker exec postgres psql -U postgres -lqt | Select-String -Pattern "sms_gateway_test"
if (-not $dbExists) {
    Write-Host "Creating test database..." -ForegroundColor Yellow
    docker exec postgres psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Failed to create test database." -ForegroundColor Red
        exit 1
    }
    Write-Host "Test database created." -ForegroundColor Green
} else {
    Write-Host "Test database exists." -ForegroundColor Green
}
Write-Host ""

# Run migrations on test database
Write-Host "Running migrations on test database..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
go run cmd/migrate/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Warning: Migrations may have failed. Continuing anyway..." -ForegroundColor Yellow
}
Write-Host ""

# Set test environment variables
$env:TEST_DB_HOST = "localhost"
$env:TEST_DB_PORT = "5432"
$env:TEST_DB_USER = "postgres"
$env:TEST_DB_PASSWORD = "postgres"
$env:TEST_DB_NAME = "sms_gateway_test"
$env:TEST_REDIS_HOST = "localhost"
$env:TEST_REDIS_PORT = "6379"
$env:TEST_REDIS_DB = "2"

Write-Host "Running Tariff System Integration Tests..." -ForegroundColor Yellow
Write-Host ""

# Run the tests with verbose output
go test -v -timeout 10m ./internal/integration -run TestTariffSystemSuite

$exitCode = $LASTEXITCODE

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "All Tests Passed!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "Some Tests Failed!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
}

exit $exitCode
