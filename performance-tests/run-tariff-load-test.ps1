#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Run Tariff System Load Tests
.DESCRIPTION
    Executes load tests for the tariff pricing system:
    - Test 1: 1000 concurrent subscriptions (Requirement 27.1)
    - Test 2: 10000 concurrent balance deductions (Requirement 27.2)
    - Test 3: 100 concurrent dedicated gateways (Requirement 27.3)
.PARAMETER ApiUrl
    Base URL for the API (default: http://localhost:8080)
.PARAMETER AdminToken
    Admin authentication token for API calls
.PARAMETER SkipPreCheck
    Skip pre-flight checks
.EXAMPLE
    .\run-tariff-load-test.ps1
.EXAMPLE
    .\run-tariff-load-test.ps1 -ApiUrl "http://localhost:8080" -AdminToken "your-token"
#>

param(
    [string]$ApiUrl = "http://localhost:8080",
    [string]$AdminToken = "admin-test-token",
    [switch]$SkipPreCheck
)

$ErrorActionPreference = "Stop"

# Colors
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
    Write-ColorOutput "═══════════════════════════════════════════════════════════════" $ColorCyan
    Write-ColorOutput "  $Title" $ColorCyan
    Write-ColorOutput "═══════════════════════════════════════════════════════════════" $ColorCyan
    Write-Host ""
}

function Test-Command {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

function Test-ApiHealth {
    param([string]$Url)
    try {
        $response = Invoke-WebRequest -Uri "$Url/health" -Method GET -TimeoutSec 5 -ErrorAction SilentlyContinue
        return $response.StatusCode -eq 200
    } catch {
        return $false
    }
}

# Main execution
Write-Header "Tariff System Load Testing"

Write-ColorOutput "📋 Test Configuration:" $ColorBlue
Write-Host "   API URL:      $ApiUrl"
Write-Host "   Admin Token:  $($AdminToken.Substring(0, [Math]::Min(10, $AdminToken.Length)))..."
Write-Host ""

# Pre-flight checks
if (-not $SkipPreCheck) {
    Write-Header "Pre-flight Checks"
    
    # Check k6
    Write-ColorOutput "🔍 Checking k6 installation..." $ColorYellow
    if (-not (Test-Command "k6")) {
        Write-ColorOutput "❌ k6 is not installed!" $ColorRed
        Write-Host ""
        Write-Host "Please install k6:"
        Write-Host "  Windows (Chocolatey): choco install k6"
        Write-Host "  Windows (Scoop):      scoop install k6"
        Write-Host "  macOS:                brew install k6"
        Write-Host "  Linux:                See https://k6.io/docs/getting-started/installation/"
        exit 1
    }
    Write-ColorOutput "✅ k6 is installed" $ColorGreen
    
    # Check API health
    Write-ColorOutput "🔍 Checking API health..." $ColorYellow
    if (-not (Test-ApiHealth -Url $ApiUrl)) {
        Write-ColorOutput "❌ API is not responding at $ApiUrl" $ColorRed
        Write-Host ""
        Write-Host "Please ensure:"
        Write-Host "  1. The backend server is running"
        Write-Host "  2. The API URL is correct"
        Write-Host "  3. The server is accessible"
        Write-Host ""
        Write-Host "To skip this check, use: -SkipPreCheck"
        exit 1
    }
    Write-ColorOutput "✅ API is healthy" $ColorGreen
    
    Write-Host ""
}

# Create results directory
$resultsDir = Join-Path $PSScriptRoot "results"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir | Out-Null
    Write-ColorOutput "📁 Created results directory" $ColorGreen
}

# Run the load test
Write-Header "Running Tariff Load Tests"

Write-ColorOutput "🚀 Starting load tests..." $ColorYellow
Write-Host ""
Write-Host "This will run three sequential tests:"
Write-Host "  1. 1000 concurrent subscriptions (0-5 min)"
Write-Host "  2. 10000 concurrent balance deductions (6-16 min)"
Write-Host "  3. 100 concurrent dedicated gateways (17-22 min)"
Write-Host ""
Write-Host "Total estimated duration: ~22 minutes"
Write-Host ""

$testFile = Join-Path $PSScriptRoot "tariff-load-test.js"

if (-not (Test-Path $testFile)) {
    Write-ColorOutput "❌ Test file not found: $testFile" $ColorRed
    exit 1
}

$env:API_URL = $ApiUrl
$env:ADMIN_TOKEN = $AdminToken

try {
    $startTime = Get-Date
    
    k6 run $testFile
    
    $exitCode = $LASTEXITCODE
    $endTime = Get-Date
    $duration = $endTime - $startTime
    
    Write-Host ""
    Write-Header "Test Results"
    
    if ($exitCode -eq 0) {
        Write-ColorOutput "✅ All load tests completed successfully!" $ColorGreen
    } else {
        Write-ColorOutput "⚠️  Some tests did not meet thresholds (exit code: $exitCode)" $ColorYellow
    }
    
    Write-Host ""
    Write-Host "Duration: $($duration.ToString('mm\:ss'))"
    Write-Host ""
    
    # Display results files
    Write-ColorOutput "📊 Results saved to:" $ColorBlue
    
    $summaryFile = Join-Path $resultsDir "tariff-load-test-summary.json"
    $fullFile = Join-Path $resultsDir "tariff-load-test-full.json"
    $reportFile = Join-Path $resultsDir "tariff-load-test-report.html"
    
    if (Test-Path $summaryFile) {
        Write-Host "   ✓ Summary:  $summaryFile"
        
        # Display key metrics
        try {
            $summary = Get-Content $summaryFile | ConvertFrom-Json
            
            Write-Host ""
            Write-ColorOutput "📈 Key Metrics:" $ColorCyan
            Write-Host ""
            
            # Test 1
            $test1 = $summary.test_1_concurrent_subscriptions
            Write-Host "   Test 1 (Subscriptions):"
            Write-Host "     Created:      $($test1.created)/$($test1.target)"
            Write-Host "     Success Rate: $($test1.success_rate.ToString('F2'))%"
            Write-Host "     Latency P95:  $($test1.latency_p95_ms.ToString('F2'))ms"
            Write-Host "     Status:       $(if ($test1.target_achieved) { '✅ PASSED' } else { '❌ FAILED' })"
            Write-Host ""
            
            # Test 2
            $test2 = $summary.test_2_concurrent_deductions
            Write-Host "   Test 2 (Balance Deductions):"
            Write-Host "     Completed:    $($test2.completed)/$($test2.target)"
            Write-Host "     Success Rate: $($test2.success_rate.ToString('F2'))%"
            Write-Host "     Latency P95:  $($test2.latency_p95_ms.ToString('F2'))ms"
            Write-Host "     Status:       $(if ($test2.target_achieved) { '✅ PASSED' } else { '❌ FAILED' })"
            Write-Host ""
            
            # Test 3
            $test3 = $summary.test_3_concurrent_gateways
            Write-Host "   Test 3 (Dedicated Gateways):"
            Write-Host "     Created:      $($test3.created)/$($test3.target)"
            Write-Host "     Success Rate: $($test3.success_rate.ToString('F2'))%"
            Write-Host "     Latency P95:  $($test3.latency_p95_ms.ToString('F2'))ms"
            Write-Host "     Status:       $(if ($test3.target_achieved) { '✅ PASSED' } else { '❌ FAILED' })"
            Write-Host ""
            
            # Overall
            Write-Host "   Overall:"
            Write-Host "     All Targets:  $(if ($summary.overall.all_targets_achieved) { '✅ ACHIEVED' } else { '❌ NOT ACHIEVED' })"
            Write-Host "     Thresholds:   $(if ($summary.overall.thresholds_passed) { '✅ PASSED' } else { '❌ FAILED' })"
            Write-Host ""
            
        } catch {
            Write-ColorOutput "⚠️  Could not parse summary file" $ColorYellow
        }
    }
    
    if (Test-Path $fullFile) {
        Write-Host "   ✓ Full:     $fullFile"
    }
    
    if (Test-Path $reportFile) {
        Write-Host "   ✓ Report:   $reportFile"
        Write-Host ""
        Write-ColorOutput "💡 Open the HTML report in your browser for detailed visualizations" $ColorCyan
    }
    
    Write-Host ""
    
    exit $exitCode
    
} catch {
    Write-ColorOutput "❌ Error running load tests: $_" $ColorRed
    exit 1
}
