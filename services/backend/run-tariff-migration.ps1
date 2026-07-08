#!/usr/bin/env pwsh
# Run Tariff Migration Script
# This script runs the tariff table migration

Write-Host "=== Running Tariff Table Migration ===" -ForegroundColor Cyan
Write-Host ""

# Change to backend directory
Push-Location $PSScriptRoot

try {
    # Run the migration
    Write-Host "Executing migration..." -ForegroundColor Yellow
    go run cmd/migrate/main.go
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "✗ Migration failed with exit code: $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
} finally {
    Pop-Location
}
