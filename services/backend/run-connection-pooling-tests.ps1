#!/usr/bin/env pwsh

# Connection Pooling Tests Runner
# Task 35.3: Connection Pooling optimization tests

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Connection Pooling Optimization Tests" -ForegroundColor Cyan
Write-Host "Task 35.3 - Requirement 27.7" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Change to backend directory
Set-Location -Path $PSScriptRoot

# Test counter
$testsPassed = 0
$testsFailed = 0

# Function to run tests
function Run-Test {
    param(
        [string]$Name,
        [string]$Command
    )
    
    Write-Host "Running: $Name" -ForegroundColor Yellow
    Write-Host "Command: $Command" -ForegroundColor Gray
    Write-Host ""
    
    Invoke-Expression $Command
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $Name PASSED" -ForegroundColor Green
        $script:testsPassed++
    } else {
        Write-Host "✗ $Name FAILED" -ForegroundColor Red
        $script:testsFailed++
    }
    Write-Host ""
}

# 1. Database Pool Optimizer Tests
Write-Host "1. Database Pool Optimizer Tests" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Run-Test "Database Pool Optimizer Unit Tests" "go test -v ./internal/database -run TestPoolOptimizer"

# 2. Redis Pool Optimizer Tests
Write-Host "2. Redis Pool Optimizer Tests" -ForegroundColor Cyan
Write-Host "=============================" -ForegroundColor Cyan
Run-Test "Redis Pool Optimizer Unit Tests" "go test -v ./internal/redis -run TestRedisPoolOptimizer"

# 3. Database Pool Optimizer Benchmarks
Write-Host "3. Database Pool Optimizer Benchmarks" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Run-Test "Database Pool Optimizer Benchmarks" "go test -bench=BenchmarkPoolOptimizer ./internal/database -benchmem"

# 4. Redis Pool Optimizer Benchmarks
Write-Host "4. Redis Pool Optimizer Benchmarks" -ForegroundColor Cyan
Write-Host "===================================" -ForegroundColor Cyan
Run-Test "Redis Pool Optimizer Benchmarks" "go test -bench=BenchmarkRedisPoolOptimizer ./internal/redis -benchmem"

# 5. Configuration Validation Tests
Write-Host "5. Configuration Validation Tests" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Run-Test "Pool Config Validation" "go test -v ./internal/database -run TestPoolConfig_Validation"
Run-Test "Redis Pool Config Validation" "go test -v ./internal/redis -run TestRedisPoolConfig_Validation"

# 6. Environment Scaling Tests
Write-Host "6. Environment Scaling Tests" -ForegroundColor Cyan
Write-Host "=============================" -ForegroundColor Cyan
Run-Test "Database Environment Scaling" "go test -v ./internal/database -run TestPoolOptimizer_EnvironmentScaling"
Run-Test "Redis Environment Scaling" "go test -v ./internal/redis -run TestRedisPoolOptimizer_EnvironmentScaling"

# 7. Load Scaling Tests
Write-Host "7. Load Scaling Tests" -ForegroundColor Cyan
Write-Host "=====================" -ForegroundColor Cyan
Run-Test "Redis Load Scaling" "go test -v ./internal/redis -run TestRedisPoolOptimizer_LoadScaling"

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Tests Passed: $testsPassed" -ForegroundColor Green
Write-Host "Tests Failed: $testsFailed" -ForegroundColor $(if ($testsFailed -eq 0) { "Green" } else { "Red" })
Write-Host ""

if ($testsFailed -eq 0) {
    Write-Host "✓ All connection pooling tests passed!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Cyan
    Write-Host "1. Review CONNECTION_POOLING_GUIDE.md for detailed documentation" -ForegroundColor White
    Write-Host "2. Check CONNECTION_POOLING_QUICK_START.md for implementation examples" -ForegroundColor White
    Write-Host "3. Update database.go and redis.go to use optimizers" -ForegroundColor White
    Write-Host "4. Run load tests to validate configuration" -ForegroundColor White
    Write-Host "5. Monitor pool metrics in production" -ForegroundColor White
    exit 0
} else {
    Write-Host "✗ Some tests failed. Please review the output above." -ForegroundColor Red
    exit 1
}
