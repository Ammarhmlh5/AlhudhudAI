#!/usr/bin/env pwsh
# Script to run Tariff Engine Property Tests
# This script runs all property-based tests for the tariff engine

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Tariff Engine Property Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL connection..." -ForegroundColor Yellow
$pgCheck = docker ps --filter "name=postgres" --filter "status=running" --format "{{.Names}}"
if (-not $pgCheck) {
    Write-Host "Warning: PostgreSQL container may not be running" -ForegroundColor Yellow
    Write-Host "Starting infrastructure..." -ForegroundColor Yellow
    docker-compose up -d postgres redis
    Start-Sleep -Seconds 5
}

# Check if Redis is running
Write-Host "Checking Redis connection..." -ForegroundColor Yellow
$redisCheck = docker ps --filter "name=redis" --filter "status=running" --format "{{.Names}}"
if (-not $redisCheck) {
    Write-Host "Warning: Redis container may not be running" -ForegroundColor Yellow
    Write-Host "Starting infrastructure..." -ForegroundColor Yellow
    docker-compose up -d redis
    Start-Sleep -Seconds 3
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Property Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Set working directory
Set-Location -Path "services/backend"

# Run Property Tests
Write-Host "Test 1: Active Tariff Selection Property Tests" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_UseActiveTariffOnly -count=1
$test1Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 2: Tariff Effective Date Boundaries" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_TariffEffectiveDateBoundaries -count=1
$test2Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 3: Most Specific Tariff Selection" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_MostSpecificTariffSelection -count=1
$test3Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 4: Inactive Tariffs Never Used" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_InactiveTariffsNeverUsed -count=1
$test4Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 5: Tariff Cache Consistency" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_TariffCacheConsistency -count=1
$test5Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 6: Correct Segment Calculation" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_CorrectSegmentCalculation -count=1
$test6Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 7: Segments Never Less Than One" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_SegmentsNeverLessThanOne -count=1
$test7Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 8: Segments Increase Monotonically" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_SegmentsIncreaseMonotonically -count=1
$test8Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 9: GSM-7 Segmentation Rules" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_GSM7SegmentationRules -count=1
$test9Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 10: Unicode Segmentation Rules" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_UnicodeSegmentationRules -count=1
$test10Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 11: Random Message Lengths" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_RandomMessageLengths -count=1
$test11Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 12: Default Encoding Is GSM-7" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_DefaultEncodingIsGSM7 -count=1
$test12Result = $LASTEXITCODE

Write-Host ""
Write-Host "Test 13: Invalid Message Length" -ForegroundColor Green
Write-Host "-----------------------------------------------" -ForegroundColor Gray
go test -v ./internal/tariff -run TestProperty_InvalidMessageLength -count=1
$test13Result = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Results Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$totalTests = 13
$passedTests = 0

$results = @(
    @{Name="Active Tariff Selection"; Result=$test1Result},
    @{Name="Tariff Effective Date Boundaries"; Result=$test2Result},
    @{Name="Most Specific Tariff Selection"; Result=$test3Result},
    @{Name="Inactive Tariffs Never Used"; Result=$test4Result},
    @{Name="Tariff Cache Consistency"; Result=$test5Result},
    @{Name="Correct Segment Calculation"; Result=$test6Result},
    @{Name="Segments Never Less Than One"; Result=$test7Result},
    @{Name="Segments Increase Monotonically"; Result=$test8Result},
    @{Name="GSM-7 Segmentation Rules"; Result=$test9Result},
    @{Name="Unicode Segmentation Rules"; Result=$test10Result},
    @{Name="Random Message Lengths"; Result=$test11Result},
    @{Name="Default Encoding Is GSM-7"; Result=$test12Result},
    @{Name="Invalid Message Length"; Result=$test13Result}
)

foreach ($result in $results) {
    if ($result.Result -eq 0) {
        Write-Host "✓ $($result.Name)" -ForegroundColor Green
        $passedTests++
    } else {
        Write-Host "✗ $($result.Name)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Total: $passedTests/$totalTests tests passed" -ForegroundColor $(if ($passedTests -eq $totalTests) { "Green" } else { "Yellow" })
Write-Host ""

if ($passedTests -eq $totalTests) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "All Property Tests PASSED! ✓" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Properties Validated:" -ForegroundColor Cyan
    Write-Host "  • Property 3: استخدام التعرفة النشطة الصحيحة" -ForegroundColor White
    Write-Host "  • Property 12: حساب أجزاء الرسالة بدقة" -ForegroundColor White
    Write-Host ""
    Write-Host "Requirements Covered:" -ForegroundColor Cyan
    Write-Host "  • Requirement 2.5: Active tariff selection" -ForegroundColor White
    Write-Host "  • Requirement 7.1: GSM-7 encoding support" -ForegroundColor White
    Write-Host "  • Requirement 7.2: Unicode encoding support" -ForegroundColor White
    Write-Host "  • Requirement 7.3: Segmentation rules" -ForegroundColor White
    Write-Host "  • Requirement 7.6: Accurate segment calculation" -ForegroundColor White
    Write-Host ""
    exit 0
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "Some Property Tests FAILED!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please review the test output above for details." -ForegroundColor Yellow
    Write-Host ""
    exit 1
}
