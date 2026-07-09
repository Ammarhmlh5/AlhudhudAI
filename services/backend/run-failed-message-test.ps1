#!/usr/bin/env pwsh
# سكريبت تشغيل اختبارات Failed Message Handling
# يدعم Windows, Linux, macOS

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("all", "temporary", "permanent", "channel", "network")]
    [string]$Scenario = "all",
    
    [Parameter(Mandatory=$false)]
    [switch]$Verbose,
    
    [Parameter(Mandatory=$false)]
    [switch]$Coverage,
    
    [Parameter(Mandatory=$false)]
    [switch]$Race,
    
    [Parameter(Mandatory=$false)]
    [switch]$Setup,
    
    [Parameter(Mandatory=$false)]
    [switch]$Cleanup
)

# الألوان
$ColorReset = "`e[0m"
$ColorGreen = "`e[32m"
$ColorYellow = "`e[33m"
$ColorRed = "`e[31m"
$ColorBlue = "`e[34m"
$ColorCyan = "`e[36m"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = $ColorReset
    )
    Write-Host "${Color}${Message}${ColorReset}"
}

function Write-Header {
    param([string]$Title)
    Write-Host ""
    Write-ColorOutput "═══════════════════════════════════════════════════════════" $ColorCyan
    Write-ColorOutput "  $Title" $ColorCyan
    Write-ColorOutput "═══════════════════════════════════════════════════════════" $ColorCyan
    Write-Host ""
}

function Check-Service {
    param(
        [string]$ServiceName,
        [string]$Command,
        [string]$ExpectedOutput
    )
    
    Write-ColorOutput "→ Checking $ServiceName..." $ColorYellow
    
    try {
        $output = Invoke-Expression $Command 2>&1
        if ($output -match $ExpectedOutput) {
            Write-ColorOutput "  ✓ $ServiceName is running" $ColorGreen
            return $true
        } else {
            Write-ColorOutput "  ✗ $ServiceName is not responding correctly" $ColorRed
            return $false
        }
    } catch {
        Write-ColorOutput "  ✗ $ServiceName is not running" $ColorRed
        return $false
    }
}

function Setup-Environment {
    Write-Header "Setting Up Test Environment"
    
    # Check PostgreSQL
    $pgRunning = Check-Service "PostgreSQL" "psql -U postgres -c 'SELECT 1' 2>&1" "1 row"
    
    # Check Redis
    $redisRunning = Check-Service "Redis" "redis-cli ping 2>&1" "PONG"
    
    if (-not $pgRunning -or -not $redisRunning) {
        Write-ColorOutput "`n⚠ Some services are not running. Starting with docker-compose..." $ColorYellow
        docker-compose up -d postgres redis
        Start-Sleep -Seconds 5
    }
    
    # Create test database
    Write-ColorOutput "`n→ Creating test database..." $ColorYellow
    $dbExists = psql -U postgres -lqt 2>&1 | Select-String -Pattern "sms_gateway_test"
    
    if (-not $dbExists) {
        psql -U postgres -c "CREATE DATABASE sms_gateway_test;" 2>&1 | Out-Null
        Write-ColorOutput "  ✓ Test database created" $ColorGreen
    } else {
        Write-ColorOutput "  ✓ Test database already exists" $ColorGreen
    }
    
    # Run migrations
    Write-ColorOutput "`n→ Running migrations..." $ColorYellow
    Push-Location $PSScriptRoot
    go run cmd/migrate/main.go -database sms_gateway_test
    Pop-Location
    Write-ColorOutput "  ✓ Migrations completed" $ColorGreen
    
    Write-ColorOutput "`n✅ Environment setup completed!" $ColorGreen
}

function Cleanup-TestData {
    Write-Header "Cleaning Up Test Data"
    
    Write-ColorOutput "→ Cleaning PostgreSQL test data..." $ColorYellow
    
    $cleanupSQL = @"
DELETE FROM balance_history WHERE tenant_id IS NOT NULL;
DELETE FROM accounting_transactions WHERE tenant_id IS NOT NULL;
DELETE FROM river_job WHERE id > 0;
DELETE FROM dead_letter_queue WHERE id > 0;
DELETE FROM messages WHERE tenant_id IS NOT NULL;
DELETE FROM channels WHERE tenant_id IS NOT NULL;
DELETE FROM tenants WHERE id IS NOT NULL;
"@
    
    $cleanupSQL | psql -U postgres -d sms_gateway_test 2>&1 | Out-Null
    Write-ColorOutput "  ✓ PostgreSQL cleaned" $ColorGreen
    
    Write-ColorOutput "`n→ Cleaning Redis test data..." $ColorYellow
    redis-cli -n 1 FLUSHDB 2>&1 | Out-Null
    Write-ColorOutput "  ✓ Redis cleaned" $ColorGreen
    
    Write-ColorOutput "`n✅ Cleanup completed!" $ColorGreen
}

function Run-Test {
    param(
        [string]$TestName,
        [string]$TestPattern
    )
    
    Write-Header "Running: $TestName"
    
    # Build test command
    $testCmd = "go test"
    
    if ($Verbose) {
        $testCmd += " -v"
    }
    
    if ($Coverage) {
        $testCmd += " -cover -coverprofile=coverage_${TestPattern}.out"
    }
    
    if ($Race) {
        $testCmd += " -race"
    }
    
    $testCmd += " -timeout 5m ./internal/integration -run $TestPattern"
    
    Write-ColorOutput "Command: $testCmd`n" $ColorBlue
    
    # Run test
    Push-Location $PSScriptRoot
    $startTime = Get-Date
    
    Invoke-Expression $testCmd
    $exitCode = $LASTEXITCODE
    
    $duration = (Get-Date) - $startTime
    Pop-Location
    
    Write-Host ""
    if ($exitCode -eq 0) {
        Write-ColorOutput "✅ Test passed in $($duration.TotalSeconds) seconds" $ColorGreen
    } else {
        Write-ColorOutput "❌ Test failed after $($duration.TotalSeconds) seconds" $ColorRed
    }
    
    return $exitCode
}

# Main execution
Write-Header "Failed Message Handling Tests"

# Setup if requested
if ($Setup) {
    Setup-Environment
    Write-Host ""
}

# Cleanup if requested
if ($Cleanup) {
    Cleanup-TestData
    exit 0
}

# Determine which tests to run
$testResults = @{}

switch ($Scenario) {
    "all" {
        Write-ColorOutput "Running all test scenarios...`n" $ColorCyan
        $testResults["All Scenarios"] = Run-Test "All Scenarios" "TestFailedMessage_AllScenarios"
    }
    "temporary" {
        $testResults["Temporary Failure"] = Run-Test "Temporary Failure" "TestFailedMessage_TemporaryFailure"
    }
    "permanent" {
        $testResults["Permanent Failure"] = Run-Test "Permanent Failure" "TestFailedMessage_PermanentFailure"
    }
    "channel" {
        $testResults["Channel Unavailable"] = Run-Test "Channel Unavailable" "TestFailedMessage_ChannelUnavailable"
    }
    "network" {
        $testResults["Network Error"] = Run-Test "Network Error" "TestFailedMessage_NetworkError"
    }
}

# Summary
Write-Header "Test Summary"

$totalTests = $testResults.Count
$passedTests = ($testResults.Values | Where-Object { $_ -eq 0 }).Count
$failedTests = $totalTests - $passedTests

foreach ($test in $testResults.GetEnumerator()) {
    $status = if ($test.Value -eq 0) { "✓ PASS" } else { "✗ FAIL" }
    $color = if ($test.Value -eq 0) { $ColorGreen } else { $ColorRed }
    Write-ColorOutput "  $status - $($test.Key)" $color
}

Write-Host ""
Write-ColorOutput "Total: $totalTests | Passed: $passedTests | Failed: $failedTests" $ColorCyan

if ($Coverage) {
    Write-Host ""
    Write-ColorOutput "Coverage reports generated:" $ColorYellow
    Get-ChildItem -Path $PSScriptRoot -Filter "coverage_*.out" | ForEach-Object {
        Write-ColorOutput "  → $($_.Name)" $ColorBlue
    }
    Write-ColorOutput "`nView coverage: go tool cover -html=coverage_*.out" $ColorYellow
}

Write-Host ""

# Exit with appropriate code
if ($failedTests -gt 0) {
    Write-ColorOutput "❌ Some tests failed" $ColorRed
    exit 1
} else {
    Write-ColorOutput "✅ All tests passed!" $ColorGreen
    exit 0
}
