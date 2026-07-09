# Load Test Runner with Resource Monitoring
# Runs k6 load test and monitors system resources simultaneously

param(
    [string]$ApiUrl = "http://localhost:8080",
    [int]$MonitorInterval = 5,
    [switch]$SkipMonitoring
)

Write-Host "╔════════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║       Load Testing - 5000+ Messages/Minute                    ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Check if k6 is installed
Write-Host "🔍 Checking prerequisites..." -ForegroundColor Yellow
try {
    $k6Version = k6 version 2>&1
    Write-Host "✅ k6 is installed: $k6Version" -ForegroundColor Green
} catch {
    Write-Host "❌ k6 is not installed. Please install k6 first." -ForegroundColor Red
    Write-Host "   Windows: choco install k6" -ForegroundColor Gray
    Write-Host "   Or download from: https://k6.io/docs/getting-started/installation/" -ForegroundColor Gray
    exit 1
}

# Check if Docker is running
try {
    docker ps | Out-Null
    Write-Host "✅ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
    exit 1
}

# Check if backend is accessible
Write-Host "🔍 Checking backend availability..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$ApiUrl/health" -Method GET -TimeoutSec 5 -ErrorAction Stop
    Write-Host "✅ Backend is accessible at $ApiUrl" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Warning: Backend may not be accessible at $ApiUrl" -ForegroundColor Yellow
    Write-Host "   Continuing anyway..." -ForegroundColor Gray
}

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "TEST CONFIGURATION" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "  API URL:              $ApiUrl" -ForegroundColor White
Write-Host "  Target Throughput:    5000+ messages/minute" -ForegroundColor White
Write-Host "  Test Duration:        ~10 minutes" -ForegroundColor White
Write-Host "  Resource Monitoring:  $(if ($SkipMonitoring) { 'Disabled' } else { 'Enabled' })" -ForegroundColor White
Write-Host "  Monitor Interval:     $MonitorInterval seconds" -ForegroundColor White
Write-Host ""

# Create results directory
$resultsDir = "results"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir -Force | Out-Null
    Write-Host "📁 Created results directory" -ForegroundColor Green
}

# Prompt user to continue
Write-Host "⚠️  This test will generate significant load on the system." -ForegroundColor Yellow
Write-Host "   Make sure you're running this on a test environment, not production!" -ForegroundColor Yellow
Write-Host ""
$continue = Read-Host "Continue? (y/n)"
if ($continue -ne 'y' -and $continue -ne 'Y') {
    Write-Host "❌ Test cancelled by user" -ForegroundColor Red
    exit 0
}

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "STARTING TEST" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host ""

# Start resource monitoring in background (if not skipped)
$monitorJob = $null
if (-not $SkipMonitoring) {
    Write-Host "📊 Starting resource monitoring..." -ForegroundColor Yellow
    $monitorScript = Join-Path $PSScriptRoot "monitor-resources.ps1"
    $monitorJob = Start-Job -FilePath $monitorScript -ArgumentList $MonitorInterval, "results/resource-monitoring.csv", 15
    Write-Host "✅ Resource monitoring started (Job ID: $($monitorJob.Id))" -ForegroundColor Green
    Start-Sleep -Seconds 2
}

Write-Host ""
Write-Host "🚀 Starting k6 load test..." -ForegroundColor Yellow
Write-Host ""

# Set environment variable for API URL
$env:API_URL = $ApiUrl

# Run k6 test
$k6Script = Join-Path $PSScriptRoot "k6-tests/load-test.js"
$testStartTime = Get-Date

try {
    k6 run $k6Script
    $testExitCode = $LASTEXITCODE
} catch {
    Write-Host "❌ Error running k6 test: $($_.Exception.Message)" -ForegroundColor Red
    $testExitCode = 1
}

$testEndTime = Get-Date
$testDuration = $testEndTime - $testStartTime

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "TEST COMPLETED" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host ""
Write-Host "⏱️  Test Duration: $($testDuration.ToString('mm\:ss'))" -ForegroundColor White
Write-Host "📊 Exit Code: $testExitCode" -ForegroundColor White
Write-Host ""

# Stop resource monitoring
if ($monitorJob) {
    Write-Host "⏹️  Stopping resource monitoring..." -ForegroundColor Yellow
    Stop-Job -Job $monitorJob
    
    # Wait a bit for the job to finish writing
    Start-Sleep -Seconds 2
    
    # Get job output
    $monitorOutput = Receive-Job -Job $monitorJob
    if ($monitorOutput) {
        Write-Host ""
        Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
        Write-Host "RESOURCE MONITORING OUTPUT" -ForegroundColor Cyan
        Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
        Write-Host $monitorOutput
    }
    
    Remove-Job -Job $monitorJob
    Write-Host "✅ Resource monitoring stopped" -ForegroundColor Green
}

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "RESULTS" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host ""

# Check if results files exist and display summary
$summaryFile = "results/load-test-summary.json"
if (Test-Path $summaryFile) {
    Write-Host "📄 Test results saved to:" -ForegroundColor Green
    Write-Host "   - results/load-test-summary.json" -ForegroundColor Gray
    Write-Host "   - results/load-test-full.json" -ForegroundColor Gray
    Write-Host "   - results/load-test-report.html" -ForegroundColor Gray
    
    if (-not $SkipMonitoring) {
        Write-Host "   - results/resource-monitoring.csv" -ForegroundColor Gray
        Write-Host "   - results/resource-monitoring-summary.json" -ForegroundColor Gray
    }
    
    Write-Host ""
    
    # Load and display key metrics
    try {
        $summary = Get-Content $summaryFile | ConvertFrom-Json
        
        Write-Host "📊 KEY METRICS:" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Throughput:" -ForegroundColor White
        Write-Host "    Messages/Minute: $([math]::Round($summary.messages_per_minute, 2)) $(if ($summary.throughput_achieved) { '✅' } else { '❌' })" -ForegroundColor $(if ($summary.throughput_achieved) { 'Green' } else { 'Red' })
        Write-Host "    Target: $($summary.target_throughput)/minute" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Success Rate:" -ForegroundColor White
        Write-Host "    Rate: $([math]::Round($summary.success_rate_percent, 2))% $(if ($summary.success_rate_achieved) { '✅' } else { '❌' })" -ForegroundColor $(if ($summary.success_rate_achieved) { 'Green' } else { 'Red' })
        Write-Host "    Target: $($summary.success_rate_target)%" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  API Response Time:" -ForegroundColor White
        Write-Host "    P95: $([math]::Round($summary.api_response_time_p95_ms, 2))ms $(if ($summary.api_response_achieved) { '✅' } else { '❌' })" -ForegroundColor $(if ($summary.api_response_achieved) { 'Green' } else { 'Red' })
        Write-Host "    Target: < $($summary.api_response_time_target_ms)ms" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  End-to-End Latency:" -ForegroundColor White
        Write-Host "    P95: $([math]::Round($summary.end_to_end_latency_p95_ms, 2))ms $(if ($summary.latency_achieved) { '✅' } else { '❌' })" -ForegroundColor $(if ($summary.latency_achieved) { 'Green' } else { 'Red' })
        Write-Host "    Target: < $($summary.end_to_end_latency_target_ms)ms" -ForegroundColor Gray
        Write-Host ""
        
        if ($summary.all_targets_achieved) {
            Write-Host "🎉 SUCCESS: All performance targets achieved!" -ForegroundColor Green
        } else {
            Write-Host "⚠️  WARNING: Some performance targets not achieved." -ForegroundColor Yellow
        }
        
    } catch {
        Write-Host "⚠️  Could not parse summary file" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠️  No summary file found. Test may have failed." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host ""

# Open HTML report if it exists
$reportFile = "results/load-test-report.html"
if (Test-Path $reportFile) {
    $openReport = Read-Host "Open HTML report in browser? (y/n)"
    if ($openReport -eq 'y' -or $openReport -eq 'Y') {
        Start-Process $reportFile
    }
}

# Exit with k6 exit code
exit $testExitCode
