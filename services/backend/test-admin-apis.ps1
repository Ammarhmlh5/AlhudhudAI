# Admin Interface APIs Testing Script
# This script tests all admin API endpoints

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Admin Interface APIs Testing" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$BASE_URL = "http://localhost:8080"
$API_KEY = "test-admin-api-key"  # Replace with actual API key
$TENANT_ID = "00000000-0000-0000-0000-000000000001"  # Test tenant ID
$SERVER_ID = "00000000-0000-0000-0000-000000000002"  # Test server ID
$ALERT_ID = "1"  # Test alert ID

# Test results tracking
$totalTests = 0
$passedTests = 0
$failedTests = 0

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Headers = @{},
        [string]$Body = $null,
        [int]$ExpectedStatus = 200
    )
    
    $global:totalTests++
    
    Write-Host "Testing: $Name" -ForegroundColor Yellow
    Write-Host "  Method: $Method" -ForegroundColor Gray
    Write-Host "  Endpoint: $Endpoint" -ForegroundColor Gray
    
    try {
        $params = @{
            Uri = "$BASE_URL$Endpoint"
            Method = $Method
            Headers = $Headers
            ContentType = "application/json"
        }
        
        if ($Body) {
            $params.Body = $Body
        }
        
        $response = Invoke-WebRequest @params -ErrorAction Stop
        
        if ($response.StatusCode -eq $ExpectedStatus) {
            Write-Host "  ✓ PASSED (Status: $($response.StatusCode))" -ForegroundColor Green
            $global:passedTests++
            
            # Parse and display response
            $jsonResponse = $response.Content | ConvertFrom-Json
            if ($jsonResponse.success) {
                Write-Host "  Response: Success" -ForegroundColor Green
            }
        } else {
            Write-Host "  ✗ FAILED (Expected: $ExpectedStatus, Got: $($response.StatusCode))" -ForegroundColor Red
            $global:failedTests++
        }
    }
    catch {
        Write-Host "  ✗ FAILED (Error: $($_.Exception.Message))" -ForegroundColor Red
        $global:failedTests++
    }
    
    Write-Host ""
}

# Headers with API key
$authHeaders = @{
    "Authorization" = "Bearer $API_KEY"
    "Content-Type" = "application/json"
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "1. Company Rating APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get Company Rating" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/$TENANT_ID/rating" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get All Company Ratings" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/ratings?limit=10" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Company Ratings by Category" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/ratings?category=excellent" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Company Rating History" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/$TENANT_ID/rating/history?limit=10" `
    -Headers $authHeaders

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "2. Performance Metrics APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get Company Metrics (Hourly)" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/$TENANT_ID/metrics?period_type=hourly&limit=24" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Company Metrics (Daily)" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/$TENANT_ID/metrics?period_type=daily&limit=7" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Company Metrics (Monthly)" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/$TENANT_ID/metrics?period_type=monthly&limit=12" `
    -Headers $authHeaders

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "3. Routing Configuration APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get Routing Configuration" `
    -Method "GET" `
    -Endpoint "/api/admin/routing/$TENANT_ID" `
    -Headers $authHeaders

$routingConfigBody = @{
    routing_type = "shared"
} | ConvertTo-Json

Test-Endpoint `
    -Name "Update Routing Configuration" `
    -Method "PUT" `
    -Endpoint "/api/admin/routing/$TENANT_ID" `
    -Headers $authHeaders `
    -Body $routingConfigBody

Test-Endpoint `
    -Name "Get Routing Metrics" `
    -Method "GET" `
    -Endpoint "/api/admin/routing/$TENANT_ID/metrics" `
    -Headers $authHeaders

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "4. Distribution Strategy APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get Distribution Configuration" `
    -Method "GET" `
    -Endpoint "/api/admin/distribution/$TENANT_ID" `
    -Headers $authHeaders

$distributionConfigBody = @{
    strategy_type = "round_robin"
    strategy_config = @{}
} | ConvertTo-Json

Test-Endpoint `
    -Name "Update Distribution Configuration" `
    -Method "PUT" `
    -Endpoint "/api/admin/distribution/$TENANT_ID" `
    -Headers $authHeaders `
    -Body $distributionConfigBody

Test-Endpoint `
    -Name "Get Distribution Devices" `
    -Method "GET" `
    -Endpoint "/api/admin/distribution/$TENANT_ID/devices" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Distribution Metrics" `
    -Method "GET" `
    -Endpoint "/api/admin/distribution/$TENANT_ID/metrics" `
    -Headers $authHeaders

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "5. Server Health APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get All Servers" `
    -Method "GET" `
    -Endpoint "/api/admin/servers" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Servers by Status" `
    -Method "GET" `
    -Endpoint "/api/admin/servers?status=healthy" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Server Details" `
    -Method "GET" `
    -Endpoint "/api/admin/servers/$SERVER_ID" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Server Health History" `
    -Method "GET" `
    -Endpoint "/api/admin/servers/$SERVER_ID/health/history?limit=100" `
    -Headers $authHeaders

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "6. Alert Management APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get All Alerts" `
    -Method "GET" `
    -Endpoint "/api/admin/alerts?limit=10" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Alerts by Severity" `
    -Method "GET" `
    -Endpoint "/api/admin/alerts?severity=critical" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Alerts by Status" `
    -Method "GET" `
    -Endpoint "/api/admin/alerts?status=active" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Alert Details" `
    -Method "GET" `
    -Endpoint "/api/admin/alerts/$ALERT_ID" `
    -Headers $authHeaders

$acknowledgeBody = @{
    acknowledged_by = "00000000-0000-0000-0000-000000000003"
} | ConvertTo-Json

Test-Endpoint `
    -Name "Acknowledge Alert" `
    -Method "PUT" `
    -Endpoint "/api/admin/alerts/$ALERT_ID/acknowledge" `
    -Headers $authHeaders `
    -Body $acknowledgeBody

Test-Endpoint `
    -Name "Resolve Alert" `
    -Method "PUT" `
    -Endpoint "/api/admin/alerts/$ALERT_ID/resolve" `
    -Headers $authHeaders

$alertConfigBody = @{
    tenant_id = $TENANT_ID
    thresholds = @{
        success_rate_min = 90
        error_rate_max = 5
    }
    notification_channels = @("email", "sms")
} | ConvertTo-Json

Test-Endpoint `
    -Name "Configure Alerts" `
    -Method "POST" `
    -Endpoint "/api/admin/alerts/config" `
    -Headers $authHeaders `
    -Body $alertConfigBody

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "7. Signal/Push System APIs" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get Timing Configuration" `
    -Method "GET" `
    -Endpoint "/api/admin/signalpush/$TENANT_ID/config" `
    -Headers $authHeaders

$timingConfigBody = @{
    signal_min_gap_ms = 2000
    push_min_gap_ms = 500
    violation_rate_threshold = 50
    z_score_threshold = 3.0
    peak_ratio_threshold = 2.0
} | ConvertTo-Json

Test-Endpoint `
    -Name "Update Timing Configuration" `
    -Method "PUT" `
    -Endpoint "/api/admin/signalpush/$TENANT_ID/config" `
    -Headers $authHeaders `
    -Body $timingConfigBody

Test-Endpoint `
    -Name "Get Signal/Push Statistics" `
    -Method "GET" `
    -Endpoint "/api/admin/signalpush/$TENANT_ID/statistics" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Timing Violations" `
    -Method "GET" `
    -Endpoint "/api/admin/signalpush/$TENANT_ID/violations?limit=100" `
    -Headers $authHeaders

Test-Endpoint `
    -Name "Get Violations by Severity" `
    -Method "GET" `
    -Endpoint "/api/admin/signalpush/$TENANT_ID/violations?severity=severe" `
    -Headers $authHeaders

$forceModeBody = @{
    mode = "push"
    reason = "Testing force mode functionality"
    duration = 60
} | ConvertTo-Json

Test-Endpoint `
    -Name "Force Push Mode" `
    -Method "POST" `
    -Endpoint "/api/admin/signalpush/$TENANT_ID/force-mode" `
    -Headers $authHeaders `
    -Body $forceModeBody

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "8. Dashboard Overview API" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Get Dashboard Overview" `
    -Method "GET" `
    -Endpoint "/api/admin/dashboard/overview" `
    -Headers $authHeaders

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "9. Authentication & Authorization Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Test Without API Key (Should Fail)" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/ratings" `
    -ExpectedStatus 401

Test-Endpoint `
    -Name "Test With Invalid API Key (Should Fail)" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/ratings" `
    -Headers @{"Authorization" = "Bearer invalid-key"} `
    -ExpectedStatus 401

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "10. Input Validation Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Test-Endpoint `
    -Name "Test Invalid Tenant ID Format" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/invalid-uuid/rating" `
    -Headers $authHeaders `
    -ExpectedStatus 400

Test-Endpoint `
    -Name "Test Invalid Category Filter" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/ratings?category=invalid" `
    -Headers $authHeaders `
    -ExpectedStatus 400

Test-Endpoint `
    -Name "Test Invalid Period Type" `
    -Method "GET" `
    -Endpoint "/api/admin/companies/$TENANT_ID/metrics?period_type=invalid" `
    -Headers $authHeaders `
    -ExpectedStatus 400

Test-Endpoint `
    -Name "Test Invalid Severity Filter" `
    -Method "GET" `
    -Endpoint "/api/admin/alerts?severity=invalid" `
    -Headers $authHeaders `
    -ExpectedStatus 400

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Total Tests: $totalTests" -ForegroundColor White
Write-Host "Passed: $passedTests" -ForegroundColor Green
Write-Host "Failed: $failedTests" -ForegroundColor Red
Write-Host ""

if ($failedTests -eq 0) {
    Write-Host "✓ All tests passed!" -ForegroundColor Green
    exit 0
} else {
    $successRate = [math]::Round(($passedTests / $totalTests) * 100, 2)
    Write-Host "Success Rate: $successRate%" -ForegroundColor Yellow
    Write-Host "⚠ Some tests failed. Please review the output above." -ForegroundColor Yellow
    exit 1
}
