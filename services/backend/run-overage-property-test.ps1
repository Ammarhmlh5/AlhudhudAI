#!/usr/bin/env pwsh

# Script to run Property-Based Tests for Overage Charging
# Task 7.6: Property Test لخصم Overage

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Property-Based Tests for Overage Charging" -ForegroundColor Cyan
Write-Host "Task 7.6: Property Test لخصم Overage" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

Write-Host "Go version:" -ForegroundColor Yellow
go version
Write-Host ""

# Check if database is running
Write-Host "Checking database connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$dbCheck = psql -h localhost -U postgres -d alhudhud_test -c "SELECT 1" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Cannot connect to test database" -ForegroundColor Red
    Write-Host "Please ensure PostgreSQL is running and alhudhud_test database exists" -ForegroundColor Red
    exit 1
}
Write-Host "Database connection OK" -ForegroundColor Green
Write-Host ""

# Set working directory
Set-Location -Path "services/backend"

Write-Host "Running Property-Based Tests for Overage..." -ForegroundColor Yellow
Write-Host ""

# Run specific overage property tests
Write-Host "Test 1: Property_OverageCharging" -ForegroundColor Cyan
go test -v -run TestProperty_OverageCharging ./internal/subscription -timeout 5m
$test1Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 2: Property_MultipleOverageCharges" -ForegroundColor Cyan
go test -v -run TestProperty_MultipleOverageCharges ./internal/subscription -timeout 5m
$test2Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 3: Property_ConcurrentOverageCharges" -ForegroundColor Cyan
go test -v -run TestProperty_ConcurrentOverageCharges ./internal/subscription -timeout 5m
$test3Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 4: Property_UnlimitedPackageNoOverage" -ForegroundColor Cyan
go test -v -run TestProperty_UnlimitedPackageNoOverage ./internal/subscription -timeout 5m
$test4Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 5: Property_OverageAtomicity" -ForegroundColor Cyan
go test -v -run TestProperty_OverageAtomicity ./internal/subscription -timeout 5m
$test5Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 6: Property_OverageInvariant" -ForegroundColor Cyan
go test -v -run TestProperty_OverageInvariant ./internal/subscription -timeout 5m
$test6Result = $LASTEXITCODE
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Results Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allPassed = $true

if ($test1Result -eq 0) {
    Write-Host "[PASS] Property_OverageCharging" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Property_OverageCharging" -ForegroundColor Red
    $allPassed = $false
}

if ($test2Result -eq 0) {
    Write-Host "[PASS] Property_MultipleOverageCharges" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Property_MultipleOverageCharges" -ForegroundColor Red
    $allPassed = $false
}

if ($test3Result -eq 0) {
    Write-Host "[PASS] Property_ConcurrentOverageCharges" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Property_ConcurrentOverageCharges" -ForegroundColor Red
    $allPassed = $false
}

if ($test4Result -eq 0) {
    Write-Host "[PASS] Property_UnlimitedPackageNoOverage" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Property_UnlimitedPackageNoOverage" -ForegroundColor Red
    $allPassed = $false
}

if ($test5Result -eq 0) {
    Write-Host "[PASS] Property_OverageAtomicity" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Property_OverageAtomicity" -ForegroundColor Red
    $allPassed = $false
}

if ($test6Result -eq 0) {
    Write-Host "[PASS] Property_OverageInvariant" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Property_OverageInvariant" -ForegroundColor Red
    $allPassed = $false
}

Write-Host ""
if ($allPassed) {
    Write-Host "All property tests PASSED!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Properties Verified:" -ForegroundColor Yellow
    Write-Host "  ✓ Property 8: خصم تكلفة الرسائل الإضافية من الرصيد" -ForegroundColor Green
    Write-Host "  ✓ Overage cost calculated correctly using package overage_rate" -ForegroundColor Green
    Write-Host "  ✓ Tenant balance checked before charging" -ForegroundColor Green
    Write-Host "  ✓ Balance deducted atomically when sufficient" -ForegroundColor Green
    Write-Host "  ✓ Transaction fails when balance insufficient" -ForegroundColor Green
    Write-Host "  ✓ Subscription overage_messages and overage_cost updated" -ForegroundColor Green
    Write-Host "  ✓ Accounting transaction recorded" -ForegroundColor Green
    Write-Host "  ✓ Unlimited packages don't charge overage" -ForegroundColor Green
    Write-Host "  ✓ Multiple overage charges accumulate correctly" -ForegroundColor Green
    Write-Host "  ✓ Concurrent overage charges maintain consistency" -ForegroundColor Green
    Write-Host "  ✓ Atomicity preserved (all-or-nothing)" -ForegroundColor Green
    Write-Host "  ✓ Invariant holds: cost = rate × messages" -ForegroundColor Green
    exit 0
} else {
    Write-Host "Some property tests FAILED!" -ForegroundColor Red
    exit 1
}
