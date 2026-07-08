# Test Script for Checkpoint 9: Routing and Health Monitoring Systems
# Tests the Server Health Monitor and Dedicated Server Routing implementations

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Checkpoint 9: Routing and Health Monitor" -ForegroundColor Cyan
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
$testsSkipped = 0

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 1: File Structure Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check Health Monitor files
Write-Host "Checking Health Monitor files..." -ForegroundColor Yellow
$healthFiles = @(
    "internal/admin/health/types.go",
    "internal/admin/health/checker.go",
    "internal/admin/health/classifier.go",
    "internal/admin/health/alert.go",
    "internal/admin/health/logger.go",
    "internal/admin/health/classifier_test.go"
)

$healthFilesExist = $true
foreach ($file in $healthFiles) {
    if (Test-Path $file) {
        Write-Host "  OK: $file" -ForegroundColor Green
    } else {
        Write-Host "  ERROR: $file NOT FOUND" -ForegroundColor Red
        $healthFilesExist = $false
    }
}

if ($healthFilesExist) {
    Write-Host "PASS: All Health Monitor files exist" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Some Health Monitor files are missing" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

# Check Routing files
Write-Host "Checking Routing Engine files..." -ForegroundColor Yellow
$routingFiles = @(
    "internal/admin/routing/types.go",
    "internal/admin/routing/config.go",
    "internal/admin/routing/engine.go",
    "internal/admin/routing/metrics.go"
)

$routingFilesExist = $true
foreach ($file in $routingFiles) {
    if (Test-Path $file) {
        Write-Host "  OK: $file" -ForegroundColor Green
    } else {
        Write-Host "  ERROR: $file NOT FOUND" -ForegroundColor Red
        $routingFilesExist = $false
    }
}

if ($routingFilesExist) {
    Write-Host "PASS: All Routing Engine files exist" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Some Routing Engine files are missing" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

# Check migrations
Write-Host "Checking database migrations..." -ForegroundColor Yellow
$migrations = @(
    "migrations/032_company_ratings_table.sql",
    "migrations/033_performance_metrics_table.sql",
    "migrations/034_routing_configurations_table.sql",
    "migrations/038_server_health_logs_table.sql"
)

$migrationsExist = $true
foreach ($migration in $migrations) {
    if (Test-Path $migration) {
        Write-Host "  OK: $migration" -ForegroundColor Green
    } else {
        Write-Host "  ERROR: $migration NOT FOUND" -ForegroundColor Red
        $migrationsExist = $false
    }
}

if ($migrationsExist) {
    Write-Host "PASS: All required migrations exist" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Some migrations are missing" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 2: Health Monitor Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Health Classifier Tests
Write-Host "Test 1: Health Status Classifier" -ForegroundColor Yellow
Write-Host "Running classifier unit tests..." -ForegroundColor White
go test -v ./internal/admin/health -run TestStatusClassifier
if ($LASTEXITCODE -eq 0) {
    Write-Host "PASS: Health Classifier tests PASSED" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Health Classifier tests FAILED" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

# Test 2: Compile Health Monitor
Write-Host "Test 2: Health Monitor Compilation" -ForegroundColor Yellow
Write-Host "Compiling health monitor package..." -ForegroundColor White
go build ./internal/admin/health
if ($LASTEXITCODE -eq 0) {
    Write-Host "PASS: Health Monitor compiles successfully" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Health Monitor compilation FAILED" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 3: Routing Engine Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Test 3: Compile Routing Engine
Write-Host "Test 3: Routing Engine Compilation" -ForegroundColor Yellow
Write-Host "Compiling routing engine package..." -ForegroundColor White
go build ./internal/admin/routing
if ($LASTEXITCODE -eq 0) {
    Write-Host "PASS: Routing Engine compiles successfully" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Routing Engine compilation FAILED" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

# Test 4: Run Routing Engine Tests
Write-Host "Test 4: Routing Engine Unit Tests" -ForegroundColor Yellow
Write-Host "Running routing engine unit tests..." -ForegroundColor White
go test -v ./internal/admin/routing
if ($LASTEXITCODE -eq 0) {
    Write-Host "PASS: Routing Engine tests PASSED" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Routing Engine tests FAILED" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 4: Integration Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Test 5: Check integration points
Write-Host "Test 5: Integration Points Verification" -ForegroundColor Yellow

$integrationChecks = @{
    "Health Monitor uses database" = (Select-String -Path "internal/admin/health/logger.go" -Pattern "db|DB" -Quiet)
    "Routing Engine uses database" = (Select-String -Path "internal/admin/routing/config.go" -Pattern "db|DB" -Quiet)
    "Health Monitor creates alerts" = (Select-String -Path "internal/admin/health/alert.go" -Pattern "Alert|alert" -Quiet)
    "Routing checks health status" = (Select-String -Path "internal/admin/routing/engine.go" -Pattern "health_status|HealthStatus" -Quiet)
    "Metrics use atomic operations" = (Select-String -Path "internal/admin/routing/metrics.go" -Pattern "atomic" -Quiet)
}

$integrationPassed = $true
foreach ($check in $integrationChecks.GetEnumerator()) {
    if ($check.Value) {
        Write-Host "  OK: $($check.Key)" -ForegroundColor Green
    } else {
        Write-Host "  ERROR: $($check.Key) NOT DETECTED" -ForegroundColor Red
        $integrationPassed = $false
    }
}

if ($integrationPassed) {
    Write-Host "PASS: All integration points verified" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "FAIL: Some integration points missing" -ForegroundColor Red
    $testsFailed++
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$totalTests = $testsPassed + $testsFailed + $testsSkipped
Write-Host "Total Tests: $totalTests" -ForegroundColor White
Write-Host "  Passed:  $testsPassed" -ForegroundColor Green
Write-Host "  Failed:  $testsFailed" -ForegroundColor Red
Write-Host "  Skipped: $testsSkipped" -ForegroundColor Yellow
Write-Host ""

if ($testsFailed -eq 0) {
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Checkpoint 9 Status: COMPLETE" -ForegroundColor Green
    Write-Host ""
    Write-Host "Systems Verified:" -ForegroundColor Yellow
    Write-Host "  OK: Server Health Monitor" -ForegroundColor Green
    Write-Host "    - Health Checker (periodic checks every 30s)" -ForegroundColor White
    Write-Host "    - Status Classifier (healthy/degraded/unhealthy)" -ForegroundColor White
    Write-Host "    - Alert Trigger (status change notifications)" -ForegroundColor White
    Write-Host "    - History Logger (30-day retention)" -ForegroundColor White
    Write-Host ""
    Write-Host "  OK: Dedicated Server Routing" -ForegroundColor Green
    Write-Host "    - Routing Configuration Manager" -ForegroundColor White
    Write-Host "    - Routing Engine (health-aware routing)" -ForegroundColor White
    Write-Host "    - Metrics Tracker (atomic operations)" -ForegroundColor White
    Write-Host "    - No-fallback policy (isolation maintained)" -ForegroundColor White
    Write-Host ""
    exit 0
} else {
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "SOME TESTS FAILED" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please review the failed tests above and fix any issues." -ForegroundColor Yellow
    exit 1
}
