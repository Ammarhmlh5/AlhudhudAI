# Simple API Gateway Performance Test
# Tests response time for 1000 requests and measures throughput
# Requirements: 3.1 - Response time < 10ms

param(
    [string]$ApiUrl = "http://localhost:8083/api/messages",
    [int]$TotalRequests = 1000,
    [int]$ConcurrentRequests = 10,
    [int]$TenantId = 1
)

Write-Host "=== API Gateway Performance Test ===" -ForegroundColor Yellow
Write-Host ""
Write-Host "Configuration:"
Write-Host "  API URL: $ApiUrl"
Write-Host "  Total Requests: $TotalRequests"
Write-Host "  Concurrent Requests: $ConcurrentRequests"
Write-Host "  Tenant ID: $TenantId"
Write-Host ""

# Check if backend is running
Write-Host "Checking backend availability..." -ForegroundColor Cyan
try {
    $healthCheck = Invoke-WebRequest -Uri "http://localhost:8083/health" -Method GET -TimeoutSec 5 -ErrorAction Stop
    Write-Host "✓ Backend is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Backend is not running or not accessible" -ForegroundColor Red
    Write-Host "Please start the backend server first:" -ForegroundColor Yellow
    Write-Host "  cd services/backend" -ForegroundColor Yellow
    Write-Host "  go run main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "Starting performance test..." -ForegroundColor Cyan
Write-Host ""

# Arrays to store results
$responseTimes = @()
$successCount = 0
$failureCount = 0
$startTime = Get-Date

# Function to send a single request
function Send-TestRequest {
    param([int]$RequestId)
    
    $payload = @{
        tenant_id = $TenantId
        phone_number = "+96650" + (Get-Random -Minimum 1000000 -Maximum 9999999)
        content = "Performance test message #$RequestId"
        priority = "NORMAL"
    } | ConvertTo-Json

    $headers = @{
        "Content-Type" = "application/json"
    }

    try {
        $requestStart = Get-Date
        $response = Invoke-WebRequest -Uri $ApiUrl -Method POST -Body $payload -Headers $headers -TimeoutSec 5 -ErrorAction Stop
        $requestEnd = Get-Date
        $duration = ($requestEnd - $requestStart).TotalMilliseconds

        $result = @{
            Success = $true
            Duration = $duration
            StatusCode = $response.StatusCode
        }

        return $result
    } catch {
        $requestEnd = Get-Date
        $duration = ($requestEnd - $requestStart).TotalMilliseconds

        $result = @{
            Success = $false
            Duration = $duration
            Error = $_.Exception.Message
        }

        return $result
    }
}

# Run requests in batches for concurrency
$batchSize = $ConcurrentRequests
$totalBatches = [Math]::Ceiling($TotalRequests / $batchSize)

for ($batch = 0; $batch -lt $totalBatches; $batch++) {
    $batchStart = $batch * $batchSize
    $batchEnd = [Math]::Min(($batch + 1) * $batchSize, $TotalRequests)
    $currentBatchSize = $batchEnd - $batchStart

    # Create jobs for concurrent requests
    $jobs = @()
    for ($i = 0; $i -lt $currentBatchSize; $i++) {
        $requestId = $batchStart + $i + 1
        $job = Start-Job -ScriptBlock {
            param($Url, $TenantId, $RequestId)
            
            $payload = @{
                tenant_id = $TenantId
                phone_number = "+96650" + (Get-Random -Minimum 1000000 -Maximum 9999999)
                content = "Performance test message #$RequestId"
                priority = "NORMAL"
            } | ConvertTo-Json

            $headers = @{
                "Content-Type" = "application/json"
            }

            try {
                $requestStart = Get-Date
                $response = Invoke-WebRequest -Uri $Url -Method POST -Body $payload -Headers $headers -TimeoutSec 5 -ErrorAction Stop
                $requestEnd = Get-Date
                $duration = ($requestEnd - $requestStart).TotalMilliseconds

                return @{
                    Success = $true
                    Duration = $duration
                    StatusCode = $response.StatusCode
                }
            } catch {
                $requestEnd = Get-Date
                $duration = ($requestEnd - $requestStart).TotalMilliseconds

                return @{
                    Success = $false
                    Duration = $duration
                    Error = $_.Exception.Message
                }
            }
        } -ArgumentList $ApiUrl, $TenantId, $requestId

        $jobs += $job
    }

    # Wait for all jobs in this batch to complete
    $results = $jobs | Wait-Job | Receive-Job
    $jobs | Remove-Job

    # Process results
    foreach ($result in $results) {
        $responseTimes += $result.Duration
        if ($result.Success) {
            $successCount++
        } else {
            $failureCount++
        }
    }

    # Progress indicator
    $completed = $batchEnd
    $percentComplete = [Math]::Round(($completed / $TotalRequests) * 100, 1)
    Write-Progress -Activity "Running performance test" -Status "$completed / $TotalRequests requests ($percentComplete%)" -PercentComplete $percentComplete
}

Write-Progress -Activity "Running performance test" -Completed

$endTime = Get-Date
$totalDuration = ($endTime - $startTime).TotalSeconds

Write-Host ""
Write-Host "=== Test Results ===" -ForegroundColor Yellow
Write-Host ""

# Calculate statistics
$avgResponseTime = ($responseTimes | Measure-Object -Average).Average
$minResponseTime = ($responseTimes | Measure-Object -Minimum).Minimum
$maxResponseTime = ($responseTimes | Measure-Object -Maximum).Maximum
$sortedTimes = $responseTimes | Sort-Object
$p95Index = [Math]::Floor($sortedTimes.Count * 0.95)
$p99Index = [Math]::Floor($sortedTimes.Count * 0.99)
$p95ResponseTime = $sortedTimes[$p95Index]
$p99ResponseTime = $sortedTimes[$p99Index]
$throughput = $TotalRequests / $totalDuration
$successRate = ($successCount / $TotalRequests) * 100

Write-Host "Total Requests: $TotalRequests" -ForegroundColor White
Write-Host "Successful: $successCount" -ForegroundColor Green
Write-Host "Failed: $failureCount" -ForegroundColor $(if ($failureCount -gt 0) { "Red" } else { "Green" })
Write-Host "Success Rate: $([Math]::Round($successRate, 2))%" -ForegroundColor $(if ($successRate -ge 99) { "Green" } else { "Yellow" })
Write-Host ""

Write-Host "Response Time Statistics:" -ForegroundColor Cyan
Write-Host "  Average: $([Math]::Round($avgResponseTime, 2)) ms" -ForegroundColor White
Write-Host "  Minimum: $([Math]::Round($minResponseTime, 2)) ms" -ForegroundColor White
Write-Host "  Maximum: $([Math]::Round($maxResponseTime, 2)) ms" -ForegroundColor White
Write-Host "  P95: $([Math]::Round($p95ResponseTime, 2)) ms" -ForegroundColor $(if ($p95ResponseTime -lt 10) { "Green" } else { "Red" })
Write-Host "  P99: $([Math]::Round($p99ResponseTime, 2)) ms" -ForegroundColor White
Write-Host ""

Write-Host "Throughput:" -ForegroundColor Cyan
Write-Host "  Total Duration: $([Math]::Round($totalDuration, 2)) seconds" -ForegroundColor White
Write-Host "  Requests/Second: $([Math]::Round($throughput, 2))" -ForegroundColor $(if ($throughput -ge 100) { "Green" } else { "Yellow" })
Write-Host "  Requests/Minute: $([Math]::Round($throughput * 60, 2))" -ForegroundColor White
Write-Host ""

# Check against requirements
Write-Host "=== Requirements Validation ===" -ForegroundColor Yellow
Write-Host ""

$requirement31Pass = $p95ResponseTime -lt 10
Write-Host "Requirement 3.1 (Response Time < 10ms):" -ForegroundColor Cyan
Write-Host "  P95 Response Time: $([Math]::Round($p95ResponseTime, 2)) ms" -ForegroundColor White
Write-Host "  Status: $(if ($requirement31Pass) { '✓ PASS' } else { '✗ FAIL' })" -ForegroundColor $(if ($requirement31Pass) { "Green" } else { "Red" })
Write-Host ""

# Save results to file
$resultsDir = "performance-tests/results"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir -Force | Out-Null
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$resultsFile = "$resultsDir/api-gateway-test-$timestamp.json"

$results = @{
    timestamp = $timestamp
    configuration = @{
        api_url = $ApiUrl
        total_requests = $TotalRequests
        concurrent_requests = $ConcurrentRequests
        tenant_id = $TenantId
    }
    results = @{
        total_requests = $TotalRequests
        successful_requests = $successCount
        failed_requests = $failureCount
        success_rate_percent = [Math]::Round($successRate, 2)
        total_duration_seconds = [Math]::Round($totalDuration, 2)
        throughput_per_second = [Math]::Round($throughput, 2)
        throughput_per_minute = [Math]::Round($throughput * 60, 2)
    }
    response_times = @{
        average_ms = [Math]::Round($avgResponseTime, 2)
        minimum_ms = [Math]::Round($minResponseTime, 2)
        maximum_ms = [Math]::Round($maxResponseTime, 2)
        p95_ms = [Math]::Round($p95ResponseTime, 2)
        p99_ms = [Math]::Round($p99ResponseTime, 2)
    }
    requirements_validation = @{
        requirement_3_1 = @{
            description = "Response Time < 10ms (P95)"
            p95_response_time_ms = [Math]::Round($p95ResponseTime, 2)
            threshold_ms = 10
            passed = $requirement31Pass
        }
    }
}

$results | ConvertTo-Json -Depth 10 | Out-File -FilePath $resultsFile -Encoding UTF8

Write-Host "Results saved to: $resultsFile" -ForegroundColor Green
Write-Host ""

# Exit with appropriate code
if ($requirement31Pass -and $successRate -ge 99) {
    Write-Host "✓ All tests passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ Some tests failed. Please review the results." -ForegroundColor Red
    exit 1
}
