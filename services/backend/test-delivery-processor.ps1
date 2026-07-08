#!/usr/bin/env pwsh
# Test script for Delivery Report Processor

Write-Host "Testing Delivery Report Processor..." -ForegroundColor Cyan

# Set CGO_ENABLED for SQLite support
$env:CGO_ENABLED = "1"

# Run tests
Write-Host "`nRunning unit tests..." -ForegroundColor Yellow
go test -v ./internal/admin/signalpush -run "TestBatchProcessor|TestDeliveryReportProcessor"

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✓ All tests passed!" -ForegroundColor Green
} else {
    Write-Host "`n✗ Tests failed!" -ForegroundColor Red
    exit 1
}
