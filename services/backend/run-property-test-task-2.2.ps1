#!/usr/bin/env pwsh
# Property Test Runner for Task 2.2 - Atomic Balance Operations
# تشغيل Property Test للمهمة 2.2

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Task 2.2: Property Test - Atomic Balance Operations" -ForegroundColor Cyan
Write-Host "Property 15: Atomic Balance Operations" -ForegroundColor Cyan
Write-Host "Validates: Requirements 16.1, 16.2" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# التحقق من وجود Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

Write-Host "✓ Go found: $(go version)" -ForegroundColor Green
Write-Host ""

# الانتقال إلى مجلد backend
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptPath

Write-Host "📂 Working directory: $(Get-Location)" -ForegroundColor Yellow
Write-Host ""

Write-Host "🧪 Running Property Tests for Task 2.2..." -ForegroundColor Cyan
Write-Host ""
Write-Host "Tests to run:" -ForegroundColor Yellow
Write-Host "  1. TestProperty_AtomicBalanceOperations" -ForegroundColor White
Write-Host "  2. TestProperty_NoRaceConditionsInBalanceOperations" -ForegroundColor White
Write-Host "  3. TestProperty_ExactlyOneSucceedsWhenBalanceSufficientForOne" -ForegroundColor White
Write-Host "  4. TestProperty_AllSuccessfulOperationsAreLogged" -ForegroundColor White
Write-Host "  5. TestProperty_BalanceNeverGoesNegative" -ForegroundColor White
Write-Host ""

# Run the property tests
Write-Host "Running tests..." -ForegroundColor Cyan
go test -v ./internal/balance -run "TestProperty_AtomicBalanceOperations|TestProperty_NoRaceConditionsInBalanceOperations|TestProperty_ExactlyOneSucceedsWhenBalanceSufficientForOne|TestProperty_AllSuccessfulOperationsAreLogged|TestProperty_BalanceNeverGoesNegative" -timeout 5m

$exitCode = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($exitCode -eq 0) {
    Write-Host "✅ All Property Tests PASSED" -ForegroundColor Green
    Write-Host ""
    Write-Host "Task 2.2 completed successfully!" -ForegroundColor Green
    Write-Host "Property 15 validated: Atomic Balance Operations work correctly" -ForegroundColor Green
} else {
    Write-Host "❌ Some Property Tests FAILED" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please review the test output above for details" -ForegroundColor Yellow
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

exit $exitCode
