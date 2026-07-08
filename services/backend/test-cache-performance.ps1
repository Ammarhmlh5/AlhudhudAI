#!/usr/bin/env pwsh
# Cache Performance Test Runner
# Tests Redis Cache performance in Routing Engine

param(
    [string]$Test = "all",
    [switch]$Short,
    [switch]$Verbose
)

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Cache Performance Test Runner" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Check if we're in the right directory
if (-not (Test-Path "go.mod")) {
    Write-Host "Error: Must be run from services/backend directory" -ForegroundColor Red
    exit 1
}

# Check PostgreSQL
Write-Host "Checking PostgreSQL..." -ForegroundColor Yellow
try {
    $pgResult = docker ps --filter "name=postgres" --format "{{.Status}}"
    if ($pgResult -match "Up") {
        Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
    } else {
        Write-Host "✗ PostgreSQL is not running" -ForegroundColor Red
        Write-Host "  Run: docker-compose up -d postgres" -ForegroundColor Yellow
        exit 1
    }
} catch {
    Write-Host "✗ Could not check PostgreSQL status" -ForegroundColor Red
    exit 1
}

# Check Redis
Write-Host "Checking Redis..." -ForegroundColor Yellow
try {
    $redisResult = docker ps --filter "name=redis" --format "{{.Status}}"
    if ($redisResult -match "Up") {
        Write-Host "✓ Redis is running" -ForegroundColor Green
    } else {
        Write-Host "✗ Redis is not running" -ForegroundColor Red
        Write-Host "  Run: docker-compose up -d redis" -ForegroundColor Yellow
        exit 1
    }
} catch {
    Write-Host "✗ Could not check Redis status" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Build test command
$testCmd = "go test"

if ($Verbose) {
    $testCmd += " -v"
}

if ($Short) {
    $testCmd += " -short"
    Write-Host "Running in SHORT mode (skipping performance tests)" -ForegroundColor Yellow
} else {
    # Select test to run
    switch ($Test) {
        "all" {
            $testCmd += " -run TestCachePerformance"
            Write-Host "Running ALL cache performance tests..." -ForegroundColor Cyan
        }
        "hitrate" {
            $testCmd += " -run TestCachePerformance_HitRate"
            Write-Host "Running Cache Hit Rate test..." -ForegroundColor Cyan
        }
        "latency" {
            $testCmd += " -run TestCachePerformance_LatencyComparison"
            Write-Host "Running Latency Comparison test..." -ForegroundColor Cyan
        }
        "dbload" {
            $testCmd += " -run TestCachePerformance_DatabaseLoadReduction"
            Write-Host "Running Database Load Reduction test..." -ForegroundColor Cyan
        }
        "concurrent" {
            $testCmd += " -run TestCachePerformance_ConcurrentAccess"
            Write-Host "Running Concurrent Access test..." -ForegroundColor Cyan
        }
        "ttl" {
            $testCmd += " -run TestCachePerformance_TTLExpiration"
            Write-Host "Running TTL Expiration test..." -ForegroundColor Cyan
        }
        "multitenant" {
            $testCmd += " -run TestCachePerformance_MultiTenantIsolation"
            Write-Host "Running Multi-Tenant Isolation test..." -ForegroundColor Cyan
        }
        default {
            Write-Host "Unknown test: $Test" -ForegroundColor Red
            Write-Host ""
            Write-Host "Available tests:" -ForegroundColor Yellow
            Write-Host "  all          - Run all cache performance tests" -ForegroundColor White
            Write-Host "  hitrate      - Cache Hit Rate test (Requirement 5.7)" -ForegroundColor White
            Write-Host "  latency      - Latency Comparison test" -ForegroundColor White
            Write-Host "  dbload       - Database Load Reduction test (Requirement 5.7)" -ForegroundColor White
            Write-Host "  concurrent   - Concurrent Access test" -ForegroundColor White
            Write-Host "  ttl          - TTL Expiration test" -ForegroundColor White
            Write-Host "  multitenant  - Multi-Tenant Isolation test" -ForegroundColor White
            exit 1
        }
    }
}

$testCmd += " -timeout 60s ./internal/routing/"

Write-Host ""
Write-Host "Command: $testCmd" -ForegroundColor Gray
Write-Host ""

# Run tests
$startTime = Get-Date
try {
    Invoke-Expression $testCmd
    $exitCode = $LASTEXITCODE
} catch {
    Write-Host ""
    Write-Host "Error running tests: $_" -ForegroundColor Red
    exit 1
}

$endTime = Get-Date
$duration = $endTime - $startTime

Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Test Results" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Duration: $($duration.TotalSeconds) seconds" -ForegroundColor White

if ($exitCode -eq 0) {
    Write-Host "Status: PASSED ✓" -ForegroundColor Green
} else {
    Write-Host "Status: FAILED ✗" -ForegroundColor Red
}

Write-Host ""

# Show next steps
if ($exitCode -eq 0) {
    Write-Host "Next Steps:" -ForegroundColor Yellow
    Write-Host "  1. Review test output above" -ForegroundColor White
    Write-Host "  2. Check that Cache Hit Rate > 95%" -ForegroundColor White
    Write-Host "  3. Check that Database Query Reduction > 99%" -ForegroundColor White
    Write-Host "  4. Review TASK_14.7_EXECUTION_SUMMARY.md for details" -ForegroundColor White
} else {
    Write-Host "Troubleshooting:" -ForegroundColor Yellow
    Write-Host "  1. Check PostgreSQL and Redis are running" -ForegroundColor White
    Write-Host "  2. Run migrations: go run cmd/migrate/main.go up" -ForegroundColor White
    Write-Host "  3. Check logs for errors" -ForegroundColor White
    Write-Host "  4. Review CACHE_PERFORMANCE_TEST_GUIDE.md" -ForegroundColor White
}

Write-Host ""

exit $exitCode
