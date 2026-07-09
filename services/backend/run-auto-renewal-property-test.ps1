#!/usr/bin/env pwsh

# Auto Renewal Property Test Runner
# This script runs the property-based tests for the auto renewal service

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Auto Renewal Property Test Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL connection..." -ForegroundColor Yellow
$pgCheck = psql -h localhost -U postgres -d sms_gateway -c "SELECT 1;" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Cannot connect to PostgreSQL database" -ForegroundColor Red
    Write-Host "Please ensure PostgreSQL is running and the database exists" -ForegroundColor Red
    exit 1
}
Write-Host "PostgreSQL connection OK" -ForegroundColor Green
Write-Host ""

# Set environment variables
$env:GOPTER_MIN_SUCCESSFUL_TESTS = "100"

# Run the property tests
Write-Host "Running Auto Renewal Property Tests..." -ForegroundColor Yellow
Write-Host ""

Set-Location services/backend

# Run all property tests in the billing package
go test -v -run "TestProperty_" ./internal/billing/... -timeout 5m

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "All Property Tests PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "Some Property Tests FAILED!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    exit 1
}
