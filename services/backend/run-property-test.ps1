#!/usr/bin/env pwsh
# Property Test Runner for Balance Manager
# تشغيل Property-Based Tests للـ Balance Manager

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Property-Based Tests - Balance Manager" -ForegroundColor Cyan
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

# عرض الخيارات
Write-Host "Available test modes:" -ForegroundColor Cyan
Write-Host "  1. Quick test (default)" -ForegroundColor White
Write-Host "  2. With race detector" -ForegroundColor White
Write-Host "  3. Multiple iterations (count=20)" -ForegroundColor White
Write-Host "  4. Stress test (count=50)" -ForegroundColor White
Write-Host "  5. Specific test" -ForegroundColor White
Write-Host ""

# قراءة الاختيار
$choice = Read-Host "Select mode (1-5, default=1)"
if ([string]::IsNullOrWhiteSpace($choice)) {
    $choice = "1"
}

Write-Host ""
Write-Host "🧪 Running Property Tests..." -ForegroundColor Cyan
Write-Host ""

switch ($choice) {
    "1" {
        Write-Host "Mode: Quick test" -ForegroundColor Yellow
        go test -v ./internal/balance -run TestProperty
    }
    "2" {
        Write-Host "Mode: With race detector" -ForegroundColor Yellow
        go test -v -race ./internal/balance -run TestProperty
    }
    "3" {
        Write-Host "Mode: Multiple iterations (20x)" -ForegroundColor Yellow
        go test -v ./internal/balance -run TestProperty -count=20
    }
    "4" {
        Write-Host "Mode: Stress test (50x)" -ForegroundColor Yellow
        go test -v -race ./internal/balance -run TestProperty -count=50
    }
    "5" {
        Write-Host "Available tests:" -ForegroundColor Yellow
        Write-Host "  a. TestProperty_AtomicBalanceDeductionCorrectness" -ForegroundColor White
        Write-Host "  b. TestProperty_NoRaceConditionsInBalanceOperations" -ForegroundColor White
        Write-Host "  c. TestProperty_BalanceOperationIdempotency" -ForegroundColor White
        Write-Host "  d. TestProperty_BalanceMonotonicity" -ForegroundColor White
        Write-Host ""
        $testChoice = Read-Host "Select test (a-d)"
        
        $testName = switch ($testChoice) {
            "a" { "TestProperty_AtomicBalanceDeductionCorrectness" }
            "b" { "TestProperty_NoRaceConditionsInBalanceOperations" }
            "c" { "TestProperty_BalanceOperationIdempotency" }
            "d" { "TestProperty_BalanceMonotonicity" }
            default { "TestProperty" }
        }
        
        Write-Host "Running: $testName" -ForegroundColor Yellow
        go test -v ./internal/balance -run $testName
    }
    default {
        Write-Host "Invalid choice, running quick test" -ForegroundColor Yellow
        go test -v ./internal/balance -run TestProperty
    }
}

$exitCode = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($exitCode -eq 0) {
    Write-Host "✅ All Property Tests PASSED" -ForegroundColor Green
} else {
    Write-Host "❌ Some Property Tests FAILED" -ForegroundColor Red
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "📚 For more information, see:" -ForegroundColor Cyan
Write-Host "  - internal/balance/PROPERTY_TEST_QUICK_START.md" -ForegroundColor White
Write-Host "  - internal/balance/PROPERTY_TEST_GUIDE.md" -ForegroundColor White
Write-Host ""

exit $exitCode
