# Signal/Push System Checkpoint Test Script
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Signal/Push System Checkpoint Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

$testsPassed = 0
$testsFailed = 0

# Test implementations
Write-Host "`nChecking implementations..." -ForegroundColor Yellow
$files = @(
    "internal/admin/signalpush/timing.go",
    "internal/admin/signalpush/analyzer.go",
    "internal/admin/signalpush/enforcement.go",
    "internal/admin/signalpush/mode.go",
    "internal/admin/signalpush/sender.go",
    "internal/admin/signalpush/load.go",
    "internal/admin/signalpush/queue.go",
    "internal/admin/signalpush/delivery.go",
    "internal/admin/signalpush/workers.go",
    "internal/admin/signalpush/notify.go"
)

foreach ($file in $files) {
    if (Test-Path $file) {
        Write-Host "  OK $file" -ForegroundColor Green
        $testsPassed++
    } else {
        Write-Host "  FAIL $file" -ForegroundColor Red
        $testsFailed++
    }
}

# Test migrations
Write-Host "`nChecking database migrations..." -ForegroundColor Yellow
$migrations = @(
    "migrations/040_message_queue_table.sql",
    "migrations/041_waiting_messages_table.sql",
    "migrations/042_timing_violations_table.sql",
    "migrations/043_tenant_statistics_table.sql",
    "migrations/044_enforcement_actions_table.sql",
    "migrations/045_tenant_timing_config_table.sql"
)

foreach ($migration in $migrations) {
    if (Test-Path $migration) {
        Write-Host "  OK $migration" -ForegroundColor Green
        $testsPassed++
    } else {
        Write-Host "  FAIL $migration" -ForegroundColor Red
        $testsFailed++
    }
}

# Run unit tests
Write-Host "`nRunning unit tests..." -ForegroundColor Yellow
go test ./internal/admin/signalpush/... -v
if ($LASTEXITCODE -eq 0) {
    Write-Host "  OK Unit tests passed" -ForegroundColor Green
    $testsPassed++
} else {
    Write-Host "  FAIL Unit tests failed" -ForegroundColor Red
    $testsFailed++
}

# Summary
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "Passed: $testsPassed" -ForegroundColor Green
Write-Host "Failed: $testsFailed" -ForegroundColor Red

if ($testsFailed -eq 0) {
    Write-Host "`nAll tests passed!" -ForegroundColor Green
} else {
    Write-Host "`nSome tests failed!" -ForegroundColor Red
}
