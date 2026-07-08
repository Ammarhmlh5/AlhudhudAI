#!/usr/bin/env pwsh

# Script to run Property Test 6: Message Deduction Tests
# Task 7.4 - Property Test لخصم من العداد

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Property Test 6: Message Deduction" -ForegroundColor Cyan
Write-Host "Task 7.4 - خصم من عداد الرسائل" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if database is running
Write-Host "Checking database connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$dbCheck = psql -h localhost -U postgres -d alhudhud_test -c "SELECT 1" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Database not available. Please start PostgreSQL and create alhudhud_test database." -ForegroundColor Red
    Write-Host ""
    Write-Host "To create the test database:" -ForegroundColor Yellow
    Write-Host "  psql -h localhost -U postgres -c 'CREATE DATABASE alhudhud_test;'" -ForegroundColor Gray
    exit 1
}
Write-Host "✓ Database connection OK" -ForegroundColor Green
Write-Host ""

# Run the property tests
Write-Host "Running Property Test 6: Message Deduction..." -ForegroundColor Yellow
Write-Host ""

# Test 1: Basic Message Deduction
Write-Host "Test 1: Basic Message Deduction Property" -ForegroundColor Cyan
go test -v -run "TestProperty_MessageDeduction$" ./internal/subscription/ -timeout 5m
$test1Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 2: Concurrent Message Deduction" -ForegroundColor Cyan
go test -v -run "TestProperty_ConcurrentMessageDeduction" ./internal/subscription/ -timeout 5m
$test2Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 3: Unlimited Package Deduction" -ForegroundColor Cyan
go test -v -run "TestProperty_UnlimitedPackageDeduction" ./internal/subscription/ -timeout 5m
$test3Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 4: Message Deduction Atomicity" -ForegroundColor Cyan
go test -v -run "TestProperty_MessageDeductionAtomicity" ./internal/subscription/ -timeout 5m
$test4Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 5: Message Deduction Invariant" -ForegroundColor Cyan
go test -v -run "TestProperty_MessageDeductionInvariant" ./internal/subscription/ -timeout 5m
$test5Result = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Results Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$allPassed = $true

if ($test1Result -eq 0) {
    Write-Host "✓ Test 1: Basic Message Deduction - PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Test 1: Basic Message Deduction - FAILED" -ForegroundColor Red
    $allPassed = $false
}

if ($test2Result -eq 0) {
    Write-Host "✓ Test 2: Concurrent Message Deduction - PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Test 2: Concurrent Message Deduction - FAILED" -ForegroundColor Red
    $allPassed = $false
}

if ($test3Result -eq 0) {
    Write-Host "✓ Test 3: Unlimited Package Deduction - PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Test 3: Unlimited Package Deduction - FAILED" -ForegroundColor Red
    $allPassed = $false
}

if ($test4Result -eq 0) {
    Write-Host "✓ Test 4: Message Deduction Atomicity - PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Test 4: Message Deduction Atomicity - FAILED" -ForegroundColor Red
    $allPassed = $false
}

if ($test5Result -eq 0) {
    Write-Host "✓ Test 5: Message Deduction Invariant - PASSED" -ForegroundColor Green
} else {
    Write-Host "✗ Test 5: Message Deduction Invariant - FAILED" -ForegroundColor Red
    $allPassed = $false
}

Write-Host ""
if ($allPassed) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "✓ All Property Tests PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Property 6 Verified:" -ForegroundColor Green
    Write-Host "  ✓ messages_remaining decreases correctly" -ForegroundColor Green
    Write-Host "  ✓ messages_used increases correctly" -ForegroundColor Green
    Write-Host "  ✓ Invariant (used + remaining = quota) holds" -ForegroundColor Green
    Write-Host "  ✓ Concurrent deductions are safe" -ForegroundColor Green
    Write-Host "  ✓ Deduction is atomic" -ForegroundColor Green
    Write-Host "  ✓ Unlimited packages work correctly" -ForegroundColor Green
    exit 0
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "✗ Some Property Tests FAILED" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    exit 1
}
