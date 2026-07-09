#!/usr/bin/env pwsh
# PowerShell script to run WebSocket Reconnection Integration Test
# اختبار إعادة اتصال WebSocket

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "WebSocket Reconnection Integration Test" -ForegroundColor Cyan
Write-Host "اختبار إعادة اتصال WebSocket" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker status..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
    Write-Host "   Run: docker-compose up -d" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ Docker is running" -ForegroundColor Green
Write-Host ""

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL..." -ForegroundColor Yellow
$pgRunning = docker ps --filter "name=postgres" --format "{{.Names}}" 2>&1
if (-not $pgRunning) {
    Write-Host "❌ PostgreSQL container is not running." -ForegroundColor Red
    Write-Host "   Run: docker-compose up -d postgres" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Check if Redis is running
Write-Host "Checking Redis..." -ForegroundColor Yellow
$redisRunning = docker ps --filter "name=redis" --format "{{.Names}}" 2>&1
if (-not $redisRunning) {
    Write-Host "❌ Redis container is not running." -ForegroundColor Red
    Write-Host "   Run: docker-compose up -d redis" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ Redis is running" -ForegroundColor Green
Write-Host ""

# Wait for services to be ready
Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 2
Write-Host "✓ Services are ready" -ForegroundColor Green
Write-Host ""

# Run the test
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running WebSocket Reconnection Test..." -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Set-Location -Path "services/backend"

# Run the test with verbose output
go test -v -run TestWebSocketReconnection ./internal/integration/

$testResult = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($testResult -eq 0) {
    Write-Host "✓ WebSocket Reconnection Test PASSED" -ForegroundColor Green
    Write-Host "✓ اختبار إعادة اتصال WebSocket نجح" -ForegroundColor Green
} else {
    Write-Host "❌ WebSocket Reconnection Test FAILED" -ForegroundColor Red
    Write-Host "❌ اختبار إعادة اتصال WebSocket فشل" -ForegroundColor Red
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Return to original directory
Set-Location -Path "../.."

exit $testResult
