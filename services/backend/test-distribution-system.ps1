# Test Script for Checkpoint 11: Message Distribution System
# Verifies the distribution strategy configurations, types, selectors, and device exclusion logic.

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Checkpoint 11: Message Distribution" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
Write-Host "Checking Go installation..." -ForegroundColor Yellow
$goVersion = go version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}
Write-Host "PASS: Go is installed: $goVersion" -ForegroundColor Green
Write-Host ""

# Set working directory
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptPath

$testsPassed = 0
$testsFailed = 0

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 1: File Structure Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check Distribution Core files
Write-Host "Checking Distribution System files..." -ForegroundColor Yellow
$distFiles = @(
    "internal/admin/distribution/types.go",
    "internal/admin/distribution/round_robin.go",
    "internal/admin/distribution/load_balancing.go",
    "internal/admin/distribution/priority.go",
    "internal/admin/distribution/custom.go",
    "internal/admin/distribution/failure.go",
    "internal/admin/distribution/pool.go",
    "internal/admin/distribution/selector.go"
)

$distFilesExist = $true
foreach ($file in $distFiles) {
    if (Test-Path $file) {
        Write-Host "  OK: $file" -ForegroundColor Green
    } else {
        Write-Host "  ERROR: $file NOT FOUND" -ForegroundColor Red
        $distFilesExist = $false
    }
}

if ($distFilesExist) {
    Write-Host "PASS: All Distribution files exist" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Some Distribution files are missing" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 2: Code Compilation Check" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Compile distribution package
Write-Host "Compiling distribution package..." -ForegroundColor Yellow
go build ./internal/admin/distribution
if ($LASTEXITCODE -eq 0) {
    Write-Host "PASS: Distribution package compiles successfully" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Distribution package compilation FAILED" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 3: Execution of Distribution Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Compile and run the standalone Go test program
Write-Host "Running distribution test suite..." -ForegroundColor Yellow
go run cmd/test-distribution/main.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "PASS: Distribution test suite execution successful" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Distribution test suite FAILED" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$totalTests = $testsPassed + $testsFailed
Write-Host "Total Tests: $totalTests" -ForegroundColor White
Write-Host "  Passed: $testsPassed" -ForegroundColor Green
Write-Host "  Failed: $testsFailed" -ForegroundColor Red
Write-Host ""

if ($testsFailed -eq 0) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "ALL MESSAGE DISTRIBUTION TESTS PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Checkpoint 11 Status: COMPLETE" -ForegroundColor Green
    exit 0
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "SOME TESTS FAILED" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    exit 1
}
