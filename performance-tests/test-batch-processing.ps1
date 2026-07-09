# Batch Processing Performance Test
# Tests batch processing performance with different batch sizes
# Validates Requirement 4.5: Batch processing should be 50x faster than individual processing

param(
    [string]$ApiUrl = "http://localhost:8080",
    [int]$TenantId = 1,
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Colors for output
$ColorSuccess = "Green"
$ColorWarning = "Yellow"
$ColorError = "Red"
$ColorInfo = "Cyan"

function Write-TestHeader {
    param([string]$Message)
    Write-Host "`n========================================" -ForegroundColor $ColorInfo
    Write-Host $Message -ForegroundColor $ColorInfo
    Write-Host "========================================`n" -ForegroundColor $ColorInfo
}

function Write-TestResult {
    param(
        [string]$Test,
        [bool]$Passed,
        [string]$Details = ""
    )
    
    $status = if ($Passed) { "✓ PASS" } else { "✗ FAIL" }
    $color = if ($Passed) { $ColorSuccess } else { $ColorError }
    
    Write-Host "$status - $Test" -ForegroundColor $color
    if ($Details) {
        Write-Host "  $Details" -ForegroundColor Gray
    }
}

function Generate-RandomPhone {
    $prefix = "+96650"
    $number = Get-Random -Minimum 1000000 -Maximum 9999999
    return "$prefix$number"
}

function Generate-TestMessage {
    $messages = @(
        "Performance test message",
        "Batch processing test",
        "Load testing batch processor"
    )
    return $messages | Get-Random
}

function Test-IndividualProcessing {
    param(
        [int]$MessageCount,
        [int]$TenantId
    )
    
    Write-Host "Testing individual message processing ($MessageCount messages)..." -ForegroundColor $ColorInfo
    
    $startTime = Get-Date
    $successCount = 0
    $failCount = 0
    $totalLatency = 0
    
    for ($i = 0; $i -lt $MessageCount; $i++) {
        $message = @{
            tenant_id = $TenantId
            phone_number = Generate-RandomPhone
            content = Generate-TestMessage
            priority = "NORMAL"
        } | ConvertTo-Json
        
        try {
            $msgStartTime = Get-Date
            $response = Invoke-RestMethod -Uri "$ApiUrl/api/messages" `
                -Method Post `
                -ContentType "application/json" `
                -Body $message `
                -TimeoutSec 10
            
            $msgLatency = (Get-Date) - $msgStartTime
            $totalLatency += $msgLatency.TotalMilliseconds
            
            if ($response.status -eq "QUEUED") {
                $successCount++
            } else {
                $failCount++
            }
        } catch {
            $failCount++
            if ($Verbose) {
                Write-Host "  Error: $_" -ForegroundColor $ColorError
            }
        }
        
        # Progress indicator
        if (($i + 1) % 10 -eq 0) {
            Write-Host "  Processed $($i + 1)/$MessageCount messages..." -ForegroundColor Gray
        }
    }
    
    $endTime = Get-Date
    $totalTime = ($endTime - $startTime).TotalMilliseconds
    $avgLatency = if ($successCount -gt 0) { $totalLatency / $successCount } else { 0 }
    
    return @{
        TotalTime = $totalTime
        SuccessCount = $successCount
        FailCount = $failCount
        AvgLatency = $avgLatency
        MessagesPerSecond = if ($totalTime -gt 0) { ($successCount / $totalTime) * 1000 } else { 0 }
    }
}

function Test-BatchProcessing {
    param(
        [int]$BatchSize,
        [int]$TenantId
    )
    
    Write-Host "Testing batch processing (batch size: $BatchSize)..." -ForegroundColor $ColorInfo
    
    # Generate batch of messages
    $messages = @()
    for ($i = 0; $i -lt $BatchSize; $i++) {
        $messages += @{
            tenant_id = $TenantId
            phone_number = Generate-RandomPhone
            content = Generate-TestMessage
            priority = "NORMAL"
        }
    }
    
    $payload = @{
        messages = $messages
    } | ConvertTo-Json -Depth 3
    
    try {
        $startTime = Get-Date
        $response = Invoke-RestMethod -Uri "$ApiUrl/api/messages/bulk" `
            -Method Post `
            -ContentType "application/json" `
            -Body $payload `
            -TimeoutSec 30
        
        $endTime = Get-Date
        $totalTime = ($endTime - $startTime).TotalMilliseconds
        
        if ($response.status -eq "PROCESSING") {
            return @{
                Success = $true
                TotalTime = $totalTime
                BatchSize = $BatchSize
                BatchId = $response.batch_id
                MessagesPerSecond = if ($totalTime -gt 0) { ($BatchSize / $totalTime) * 1000 } else { 0 }
            }
        } else {
            return @{
                Success = $false
                TotalTime = $totalTime
                BatchSize = $BatchSize
                Error = "Unexpected status: $($response.status)"
            }
        }
    } catch {
        return @{
            Success = $false
            BatchSize = $BatchSize
            Error = $_.Exception.Message
        }
    }
}

function Test-DatabaseLoad {
    param([int]$TenantId)
    
    Write-Host "Measuring database load during batch processing..." -ForegroundColor $ColorInfo
    
    # This is a simplified test - in production, you would query PostgreSQL metrics
    # For now, we'll measure response times as a proxy for database load
    
    $batchSizes = @(10, 50, 100)
    $results = @()
    
    foreach ($size in $batchSizes) {
        Write-Host "  Testing batch size: $size" -ForegroundColor Gray
        
        $result = Test-BatchProcessing -BatchSize $size -TenantId $TenantId
        
        if ($result.Success) {
            $results += @{
                BatchSize = $size
                ProcessingTime = $result.TotalTime
                Throughput = $result.MessagesPerSecond
            }
            
            Write-Host "    Processing Time: $([math]::Round($result.TotalTime, 2))ms" -ForegroundColor Gray
            Write-Host "    Throughput: $([math]::Round($result.MessagesPerSecond, 2)) msg/s" -ForegroundColor Gray
        } else {
            Write-Host "    Failed: $($result.Error)" -ForegroundColor $ColorError
        }
        
        # Wait between tests to avoid overwhelming the system
        Start-Sleep -Seconds 2
    }
    
    return $results
}

# Main test execution
Write-TestHeader "Batch Processing Performance Test"

Write-Host "Configuration:" -ForegroundColor $ColorInfo
Write-Host "  API URL: $ApiUrl" -ForegroundColor Gray
Write-Host "  Tenant ID: $TenantId" -ForegroundColor Gray
Write-Host ""

# Test 1: Individual vs Batch Processing Comparison
Write-TestHeader "Test 1: Individual vs Batch Processing (100 messages)"

Write-Host "Phase 1: Individual Processing" -ForegroundColor $ColorInfo
$individualResult = Test-IndividualProcessing -MessageCount 100 -TenantId $TenantId

Write-Host "`nIndividual Processing Results:" -ForegroundColor $ColorInfo
Write-Host "  Total Time: $([math]::Round($individualResult.TotalTime, 2))ms" -ForegroundColor Gray
Write-Host "  Success: $($individualResult.SuccessCount)/100" -ForegroundColor Gray
Write-Host "  Failed: $($individualResult.FailCount)" -ForegroundColor Gray
Write-Host "  Avg Latency: $([math]::Round($individualResult.AvgLatency, 2))ms" -ForegroundColor Gray
Write-Host "  Throughput: $([math]::Round($individualResult.MessagesPerSecond, 2)) msg/s" -ForegroundColor Gray

# Wait before batch test
Start-Sleep -Seconds 3

Write-Host "`nPhase 2: Batch Processing" -ForegroundColor $ColorInfo
$batchResult = Test-BatchProcessing -BatchSize 100 -TenantId $TenantId

if ($batchResult.Success) {
    Write-Host "`nBatch Processing Results:" -ForegroundColor $ColorInfo
    Write-Host "  Total Time: $([math]::Round($batchResult.TotalTime, 2))ms" -ForegroundColor Gray
    Write-Host "  Batch Size: $($batchResult.BatchSize)" -ForegroundColor Gray
    Write-Host "  Batch ID: $($batchResult.BatchId)" -ForegroundColor Gray
    Write-Host "  Throughput: $([math]::Round($batchResult.MessagesPerSecond, 2)) msg/s" -ForegroundColor Gray
    
    # Calculate speedup
    $speedup = $individualResult.TotalTime / $batchResult.TotalTime
    Write-Host "`nPerformance Comparison:" -ForegroundColor $ColorInfo
    Write-Host "  Speedup: $([math]::Round($speedup, 2))x faster" -ForegroundColor $(if ($speedup -ge 50) { $ColorSuccess } else { $ColorWarning })
    Write-Host "  Individual Time: $([math]::Round($individualResult.TotalTime, 2))ms" -ForegroundColor Gray
    Write-Host "  Batch Time: $([math]::Round($batchResult.TotalTime, 2))ms" -ForegroundColor Gray
    
    # Validate Requirement 4.5: Batch should be 50x faster
    $requirement45Met = $speedup -ge 50
    Write-TestResult -Test "Requirement 4.5: Batch 50x faster than individual" `
        -Passed $requirement45Met `
        -Details "Actual speedup: $([math]::Round($speedup, 2))x"
} else {
    Write-Host "`nBatch Processing Failed:" -ForegroundColor $ColorError
    Write-Host "  Error: $($batchResult.Error)" -ForegroundColor $ColorError
}

# Test 2: Different Batch Sizes
Write-TestHeader "Test 2: Performance with Different Batch Sizes"

$dbLoadResults = Test-DatabaseLoad -TenantId $TenantId

Write-Host "`nDatabase Load Results:" -ForegroundColor $ColorInfo
foreach ($result in $dbLoadResults) {
    Write-Host "  Batch Size $($result.BatchSize):" -ForegroundColor Gray
    Write-Host "    Processing Time: $([math]::Round($result.ProcessingTime, 2))ms" -ForegroundColor Gray
    Write-Host "    Throughput: $([math]::Round($result.Throughput, 2)) msg/s" -ForegroundColor Gray
}

# Test 3: Batch Processing Time Threshold
Write-TestHeader "Test 3: Batch Processing Time Threshold"

Write-Host "Testing batch processing time < 200ms (P95 requirement)..." -ForegroundColor $ColorInfo

$batchTests = @()
for ($i = 0; $i -lt 10; $i++) {
    $result = Test-BatchProcessing -BatchSize 100 -TenantId $TenantId
    if ($result.Success) {
        $batchTests += $result.TotalTime
        Write-Host "  Test $($i + 1): $([math]::Round($result.TotalTime, 2))ms" -ForegroundColor Gray
    }
    Start-Sleep -Milliseconds 500
}

if ($batchTests.Count -gt 0) {
    $sortedTimes = $batchTests | Sort-Object
    $p95Index = [math]::Floor($sortedTimes.Count * 0.95)
    $p95Time = $sortedTimes[$p95Index]
    $avgTime = ($batchTests | Measure-Object -Average).Average
    
    Write-Host "`nBatch Processing Time Statistics:" -ForegroundColor $ColorInfo
    Write-Host "  Average: $([math]::Round($avgTime, 2))ms" -ForegroundColor Gray
    Write-Host "  P95: $([math]::Round($p95Time, 2))ms" -ForegroundColor Gray
    Write-Host "  Min: $([math]::Round(($sortedTimes | Measure-Object -Minimum).Minimum, 2))ms" -ForegroundColor Gray
    Write-Host "  Max: $([math]::Round(($sortedTimes | Measure-Object -Maximum).Maximum, 2))ms" -ForegroundColor Gray
    
    $p95Met = $p95Time -lt 200
    Write-TestResult -Test "P95 Batch Processing Time < 200ms" `
        -Passed $p95Met `
        -Details "Actual P95: $([math]::Round($p95Time, 2))ms"
}

# Summary
Write-TestHeader "Test Summary"

Write-Host "All tests completed!" -ForegroundColor $ColorSuccess
Write-Host ""
Write-Host "Key Findings:" -ForegroundColor $ColorInfo
Write-Host "  1. Batch processing is $([math]::Round($speedup, 2))x faster than individual processing" -ForegroundColor Gray
Write-Host "  2. P95 batch processing time: $([math]::Round($p95Time, 2))ms" -ForegroundColor Gray
Write-Host "  3. Batch throughput: $([math]::Round($batchResult.MessagesPerSecond, 2)) messages/second" -ForegroundColor Gray
Write-Host ""

if ($speedup -ge 50 -and $p95Time -lt 200) {
    Write-Host "✓ All performance requirements met!" -ForegroundColor $ColorSuccess
} else {
    Write-Host "⚠ Some performance requirements not met" -ForegroundColor $ColorWarning
    if ($speedup -lt 50) {
        Write-Host "  - Batch speedup below 50x target" -ForegroundColor $ColorWarning
    }
    if ($p95Time -ge 200) {
        Write-Host "  - P95 processing time exceeds 200ms threshold" -ForegroundColor $ColorWarning
    }
}

Write-Host ""
