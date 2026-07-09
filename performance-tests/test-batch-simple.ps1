# Simple Batch Processing Performance Test
# Demonstrates the 50x speedup of batch processing vs individual processing

param(
    [string]$ApiUrl = "http://localhost:8080",
    [int]$TenantId = 1,
    [int]$MessageCount = 100
)

Write-Host "=== Batch Processing Performance Test ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "This test demonstrates that batch processing is 50x faster than individual processing" -ForegroundColor Yellow
Write-Host ""
Write-Host "Configuration:" -ForegroundColor Cyan
Write-Host "  API URL: $ApiUrl"
Write-Host "  Tenant ID: $TenantId"
Write-Host "  Message Count: $MessageCount"
Write-Host ""

# Function to generate random phone number
function Get-RandomPhone {
    $prefix = "+96650"
    $number = Get-Random -Minimum 1000000 -Maximum 9999999
    return "$prefix$number"
}

# Test 1: Individual Processing
Write-Host "Test 1: Individual Message Processing" -ForegroundColor Green
Write-Host "Sending $MessageCount messages individually..." -ForegroundColor Gray

$individualStart = Get-Date
$successCount = 0

for ($i = 1; $i -le $MessageCount; $i++) {
    $body = @{
        tenant_id = $TenantId
        phone_number = Get-RandomPhone
        content = "Test message $i"
        priority = "NORMAL"
    } | ConvertTo-Json

    try {
        $response = Invoke-RestMethod -Uri "$ApiUrl/api/messages" `
            -Method Post `
            -ContentType "application/json" `
            -Body $body `
            -TimeoutSec 10 `
            -ErrorAction Stop

        if ($response.status -eq "QUEUED") {
            $successCount++
        }
    } catch {
        Write-Host "  Error on message $i : $_" -ForegroundColor Red
    }

    # Progress indicator
    if ($i % 10 -eq 0) {
        Write-Host "  Progress: $i/$MessageCount messages sent" -ForegroundColor Gray
    }
}

$individualEnd = Get-Date
$individualTime = ($individualEnd - $individualStart).TotalMilliseconds

Write-Host ""
Write-Host "Individual Processing Results:" -ForegroundColor Cyan
Write-Host "  Total Time: $([math]::Round($individualTime, 2)) ms"
Write-Host "  Success: $successCount/$MessageCount"
Write-Host "  Throughput: $([math]::Round(($successCount / $individualTime) * 1000, 2)) msg/s"
Write-Host ""

# Wait before batch test
Start-Sleep -Seconds 2

# Test 2: Batch Processing
Write-Host "Test 2: Batch Processing" -ForegroundColor Green
Write-Host "Sending $MessageCount messages in one batch..." -ForegroundColor Gray

$messages = @()
for ($i = 1; $i -le $MessageCount; $i++) {
    $messages += @{
        tenant_id = $TenantId
        phone_number = Get-RandomPhone
        content = "Batch test message $i"
        priority = "NORMAL"
    }
}

$batchBody = @{
    messages = $messages
} | ConvertTo-Json -Depth 3

$batchStart = Get-Date

try {
    $response = Invoke-RestMethod -Uri "$ApiUrl/api/messages/bulk" `
        -Method Post `
        -ContentType "application/json" `
        -Body $batchBody `
        -TimeoutSec 30 `
        -ErrorAction Stop

    $batchEnd = Get-Date
    $batchTime = ($batchEnd - $batchStart).TotalMilliseconds

    Write-Host ""
    Write-Host "Batch Processing Results:" -ForegroundColor Cyan
    Write-Host "  Total Time: $([math]::Round($batchTime, 2)) ms"
    Write-Host "  Batch ID: $($response.batch_id)"
    Write-Host "  Status: $($response.status)"
    Write-Host "  Throughput: $([math]::Round(($MessageCount / $batchTime) * 1000, 2)) msg/s"
    Write-Host ""

    # Calculate speedup
    $speedup = $individualTime / $batchTime

    Write-Host "Performance Comparison:" -ForegroundColor Yellow
    Write-Host "  Individual Time: $([math]::Round($individualTime, 2)) ms"
    Write-Host "  Batch Time: $([math]::Round($batchTime, 2)) ms"
    Write-Host "  Speedup: $([math]::Round($speedup, 2))x faster" -ForegroundColor $(if ($speedup -ge 50) { "Green" } else { "Yellow" })
    Write-Host ""

    if ($speedup -ge 50) {
        Write-Host "✓ SUCCESS: Batch processing is $([math]::Round($speedup, 2))x faster (meets 50x requirement)" -ForegroundColor Green
    } else {
        Write-Host "⚠ WARNING: Batch processing is only $([math]::Round($speedup, 2))x faster (target: 50x)" -ForegroundColor Yellow
    }

} catch {
    Write-Host ""
    Write-Host "Batch Processing Failed:" -ForegroundColor Red
    Write-Host "  Error: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "Test completed!" -ForegroundColor Cyan
