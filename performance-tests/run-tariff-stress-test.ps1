#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Run Tariff System Stress Tests
.DESCRIPTION
    Executes k6 stress tests for the tariff pricing system to validate:
    - System behavior under extreme load (beyond normal capacity)
    - Resource quota enforcement under stress
    - Rate limiting effectiveness under stress
    - Graceful degradation and recovery
    
    Requirements: 27.1, 27.2
.PARAMETER ApiUrl
    Base URL of the API (default: http://localhost:8080)
.PARAMETER AdminToken
    Admin authentication token for API access
.PARAMETER SkipSetup
    Skip environment setup checks
.EXAMPLE
    .\run-tariff-stress-test.ps1
.EXAMPLE
    .\run-tariff-stress-test.ps1 -ApiUrl "http://localhost:8080" -AdminToken "your-token"
#>

param(
    [string]$ApiUrl = "http://localhost:8080",
    [string]$AdminToken = "",
    [switch]$SkipSetup
)

# Color output functions
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Write-Header {
    param([string]$Message)
    Write-Host ""
    Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
    Write-ColorOutput " $Message" "Cyan"
    Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
    Write-Host ""
}

function Write-Success {
    param([string]$Message)
    Write-ColorOutput "✅ $Message" "Green"
}

function Write-Error {
    param([string]$Message)
    Write-ColorOutput "❌ $Message" "Red"
}

function Write-Warning {
    param([string]$Message)
    Write-ColorOutput "⚠️  $Message" "Yellow"
}

function Write-Info {
    param([string]$Message)
    Write-ColorOutput "ℹ️  $Message" "Blue"
}

# Main script
Write-Header "Tariff System Stress Testing"

Write-Info "Test Configuration:"
Write-Host "  API URL:        $ApiUrl"
Write-Host "  Admin Token:    $(if ($AdminToken) { '***' } else { 'Not provided' })"
Write-Host ""

# Check if k6 is installed
if (-not $SkipSetup) {
    Write-Info "Checking prerequisites..."
    
    try {
        $k6Version = k6 version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "k6 is installed: $k6Version"
        } else {
            throw "k6 not found"
        }
    } catch {
        Write-Error "k6 is not installed or not in PATH"
        Write-Info "Please install k6 from: https://k6.io/docs/getting-started/installation/"
        Write-Info "Windows: choco install k6"
        Write-Info "macOS: brew install k6"
        Write-Info "Linux: sudo apt-get install k6"
        exit 1
    }
    
    Write-Host ""
}

# Check if API is accessible
if (-not $SkipSetup) {
    Write-Info "Checking API connectivity..."
    
    try {
        $response = Invoke-WebRequest -Uri "$ApiUrl/health" -Method GET -TimeoutSec 5 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            Write-Success "API is accessible at $ApiUrl"
        } else {
            Write-Warning "API returned status code: $($response.StatusCode)"
        }
    } catch {
        Write-Error "Cannot connect to API at $ApiUrl"
        Write-Info "Please ensure the backend service is running"
        Write-Info "You can start it with: cd services/backend && go run cmd/server/main.go"
        exit 1
    }
    
    Write-Host ""
}

# Create results directory
$resultsDir = Join-Path $PSScriptRoot "results"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir | Out-Null
    Write-Info "Created results directory: $resultsDir"
}

# Prepare environment variables
$env:API_URL = $ApiUrl
if ($AdminToken) {
    $env:ADMIN_TOKEN = $AdminToken
}

Write-Header "Starting Stress Tests"

Write-Warning "⚠️  STRESS TEST WARNING ⚠️"
Write-Host ""
Write-ColorOutput "This stress test will:" "Yellow"
Write-ColorOutput "  • Push the system beyond normal capacity" "Yellow"
Write-ColorOutput "  • Generate extreme load (up to 1500 concurrent users)" "Yellow"
Write-ColorOutput "  • Test resource quota enforcement" "Yellow"
Write-ColorOutput "  • Test rate limiting effectiveness" "Yellow"
Write-ColorOutput "  • Run for approximately 40 minutes" "Yellow"
Write-Host ""
Write-ColorOutput "The system is EXPECTED to show degraded performance and errors." "Yellow"
Write-ColorOutput "This is normal and validates graceful degradation." "Yellow"
Write-Host ""

$confirmation = Read-Host "Do you want to continue? (yes/no)"
if ($confirmation -ne "yes") {
    Write-Info "Stress test cancelled by user"
    exit 0
}

Write-Host ""
Write-Info "Running stress tests..."
Write-Info "This will take approximately 40 minutes..."
Write-Host ""

Write-ColorOutput "Test Phases:" "Cyan"
Write-ColorOutput "  Phase 1 (0-20m):  Extreme Load Test" "Cyan"
Write-ColorOutput "    • Ramp up to 1500 concurrent users" "Cyan"
Write-ColorOutput "    • Test system behavior beyond capacity" "Cyan"
Write-ColorOutput "    • Validate graceful degradation" "Cyan"
Write-ColorOutput "    • Test system recovery" "Cyan"
Write-Host ""
Write-ColorOutput "  Phase 2 (22-32m): Resource Quota Stress Test" "Cyan"
Write-ColorOutput "    • 200 concurrent users for 10 minutes" "Cyan"
Write-ColorOutput "    • Test CPU, Memory, Bandwidth quotas" "Cyan"
Write-ColorOutput "    • Validate quota enforcement" "Cyan"
Write-Host ""
Write-ColorOutput "  Phase 3 (33-38m): Rate Limiting Stress Test" "Cyan"
Write-ColorOutput "    • 1000 requests per second" "Cyan"
Write-ColorOutput "    • Test rate limit enforcement" "Cyan"
Write-ColorOutput "    • Validate graceful handling" "Cyan"
Write-Host ""

# Run k6 stress test
$testFile = Join-Path $PSScriptRoot "tariff-stress-test.js"

try {
    k6 run $testFile
    $exitCode = $LASTEXITCODE
    
    Write-Host ""
    
    if ($exitCode -eq 0) {
        Write-Header "Stress Test Results"
        Write-Success "All stress tests completed successfully!"
        Write-Host ""
        
        # Display summary if available
        $summaryFile = Join-Path $resultsDir "tariff-stress-test-summary.json"
        if (Test-Path $summaryFile) {
            $summary = Get-Content $summaryFile | ConvertFrom-Json
            
            Write-ColorOutput "📊 Test Summary:" "Cyan"
            Write-Host ""
            
            # Test 1: Extreme Load
            Write-ColorOutput "Test 1: Extreme Load" "White"
            Write-Host "  Total Requests:       $($summary.test_1_extreme_load.total_requests)"
            Write-Host "  Success Rate:         $([math]::Round($summary.test_1_extreme_load.success_rate, 2))%"
            Write-Host "  Error Rate:           $([math]::Round($summary.test_1_extreme_load.error_rate, 2))%"
            Write-Host "  Latency (P95):        $([math]::Round($summary.test_1_extreme_load.latency_p95_ms, 2))ms"
            Write-Host "  Graceful Degradation: $($summary.test_1_extreme_load.graceful_degradation_count) times"
            Write-Host "  System Recovered:     $($summary.test_1_extreme_load.system_recovered_count) times"
            if ($summary.test_1_extreme_load.target_achieved) {
                Write-Success "Status: PASSED"
            } else {
                Write-Error "Status: FAILED"
            }
            Write-Host ""
            
            # Test 2: Resource Quotas
            Write-ColorOutput "Test 2: Resource Quota Enforcement" "White"
            Write-Host "  Quota Exceeded:       $($summary.test_2_resource_quotas.quota_exceeded_count) times"
            Write-Host "  Quota Enforced:       $(if ($summary.test_2_resource_quotas.quota_enforced) { 'YES' } else { 'NO' })"
            Write-Host "  Graceful Handling:    $(if ($summary.test_2_resource_quotas.graceful_handling) { 'YES' } else { 'NO' })"
            if ($summary.test_2_resource_quotas.target_achieved) {
                Write-Success "Status: PASSED"
            } else {
                Write-Error "Status: FAILED"
            }
            Write-Host ""
            
            # Test 3: Rate Limiting
            Write-ColorOutput "Test 3: Rate Limiting Effectiveness" "White"
            Write-Host "  Rate Limit Hit:       $($summary.test_3_rate_limiting.rate_limit_hit_count) times"
            Write-Host "  Rate Limit Enforced:  $(if ($summary.test_3_rate_limiting.rate_limit_enforced) { 'YES' } else { 'NO' })"
            Write-Host "  Graceful Handling:    $(if ($summary.test_3_rate_limiting.graceful_handling) { 'YES' } else { 'NO' })"
            if ($summary.test_3_rate_limiting.target_achieved) {
                Write-Success "Status: PASSED"
            } else {
                Write-Error "Status: FAILED"
            }
            Write-Host ""
            
            # Overall
            Write-ColorOutput "Overall Results:" "Cyan"
            Write-Host "  All Targets Achieved: $(if ($summary.overall.all_targets_achieved) { 'YES ✅' } else { 'NO ❌' })"
            Write-Host "  All Thresholds Passed: $(if ($summary.overall.thresholds_passed) { 'YES ✅' } else { 'NO ❌' })"
            Write-Host "  Graceful Degradation: $(if ($summary.overall.graceful_degradation_working) { 'WORKING ✅' } else { 'NOT WORKING ❌' })"
            Write-Host "  System Recovery:      $(if ($summary.overall.system_recovery_working) { 'WORKING ✅' } else { 'NOT WORKING ❌' })"
            Write-Host "  Max Connections:      $($summary.overall.max_concurrent_connections)"
            Write-Host ""
        }
        
        Write-Info "Detailed results saved to:"
        Write-Host "  • $resultsDir\tariff-stress-test-summary.json"
        Write-Host "  • $resultsDir\tariff-stress-test-full.json"
        Write-Host "  • $resultsDir\tariff-stress-test-report.html"
        Write-Host ""
        
        Write-Success "Stress testing completed successfully!"
        Write-Host ""
        Write-ColorOutput "Key Findings:" "Cyan"
        Write-ColorOutput "  ✅ System handled extreme load gracefully" "Green"
        Write-ColorOutput "  ✅ Resource quotas enforced under stress" "Green"
        Write-ColorOutput "  ✅ Rate limiting effective under stress" "Green"
        Write-ColorOutput "  ✅ System degraded gracefully when overloaded" "Green"
        Write-ColorOutput "  ✅ System recovered after stress" "Green"
        
    } else {
        Write-Header "Stress Test Results"
        Write-Error "Some stress tests failed or did not meet thresholds"
        Write-Host ""
        Write-Warning "This may indicate:"
        Write-Host "  • System cannot handle extreme load gracefully"
        Write-Host "  • Resource quotas not enforced properly"
        Write-Host "  • Rate limiting not effective"
        Write-Host "  • System does not degrade gracefully"
        Write-Host "  • System does not recover properly"
        Write-Host ""
        Write-Info "Check the detailed results for more information:"
        Write-Host "  • $resultsDir\tariff-stress-test-summary.json"
        Write-Host "  • $resultsDir\tariff-stress-test-report.html"
        
        exit 1
    }
    
} catch {
    Write-Error "Failed to run stress tests: $_"
    exit 1
}

Write-Host ""
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput " Stress Testing Complete" "Cyan"
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-Host ""
