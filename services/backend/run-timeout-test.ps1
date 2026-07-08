#!/usr/bin/env pwsh

Write-Host "Running Balance Manager Timeout Tests..." -ForegroundColor Cyan

# Navigate to backend directory
Set-Location -Path $PSScriptRoot

# Run timeout tests
Write-Host "`nRunning timeout tests..." -ForegroundColor Yellow
go test -v ./internal/balance -run "TestDeductBalanceWithTimeout"

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✓ All timeout tests passed!" -ForegroundColor Green
} else {
    Write-Host "`n✗ Some timeout tests failed!" -ForegroundColor Red
    exit 1
}
