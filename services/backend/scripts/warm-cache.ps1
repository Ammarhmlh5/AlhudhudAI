# Cache Warming Script (PowerShell)
# Task 35.2: Redis Optimization - Cache Warming
# Requirements: 27.9

Write-Host "================================" -ForegroundColor Cyan
Write-Host "Cache Warming Script" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match "^([^=]+)=(.*)$") {
            [System.Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
}

# Configuration
$API_URL = $env:API_URL
if (-not $API_URL) { $API_URL = "http://localhost:8080" }

$ADMIN_TOKEN = $env:ADMIN_TOKEN

if (-not $ADMIN_TOKEN) {
    Write-Host "Error: ADMIN_TOKEN not set" -ForegroundColor Red
    Write-Host "Please set ADMIN_TOKEN environment variable" -ForegroundColor Red
    exit 1
}

Write-Host "API URL: $API_URL" -ForegroundColor Yellow
Write-Host ""

# Function to call API
function Call-API {
    param (
        [string]$Endpoint,
        [string]$Description
    )
    
    Write-Host "Warming: $Description" -ForegroundColor Green
    
    try {
        $headers = @{
            "Authorization" = "Bearer $ADMIN_TOKEN"
            "Content-Type" = "application/json"
        }
        
        $response = Invoke-RestMethod -Uri "$API_URL$Endpoint" -Method Post -Headers $headers -ErrorAction Stop
        
        Write-Host "✓ $Description warmed successfully" -ForegroundColor Green
    }
    catch {
        Write-Host "✗ Failed to warm $Description" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    }
    
    Write-Host ""
}

# Warm cache for different data types
Write-Host "Starting cache warming..." -ForegroundColor Cyan
Write-Host ""

# 1. Warm tariffs cache
Call-API "/admin/cache/warm/tariffs" "Active Tariffs"

# 2. Warm packages cache
Call-API "/admin/cache/warm/packages" "Active Packages"

# 3. Warm services cache
Call-API "/admin/cache/warm/services" "Active Services"

# 4. Warm segmentation rules cache
Call-API "/admin/cache/warm/segmentation-rules" "Segmentation Rules"

# 5. Warm exchange rates cache
Call-API "/admin/cache/warm/exchange-rates" "Exchange Rates"

# 6. Warm all caches
Call-API "/admin/cache/warm/all" "All Caches"

Write-Host "================================" -ForegroundColor Cyan
Write-Host "Cache Warming Complete" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
