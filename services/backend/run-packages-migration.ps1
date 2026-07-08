#!/usr/bin/env pwsh
# Run Packages Table Migration Script
# This script runs the packages table migration (016_packages_table.sql)

Write-Host "=== Running Packages Table Migration ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "This migration creates the packages table with:" -ForegroundColor White
Write-Host "  - Full tenant isolation (RLS policies)" -ForegroundColor White
Write-Host "  - Five default packages (Basic, Standard, Professional, Business, Enterprise)" -ForegroundColor White
Write-Host "  - Rollover configuration support" -ForegroundColor White
Write-Host ""

# Change to backend directory
Push-Location $PSScriptRoot

try {
    # Check if .env file exists
    if (-not (Test-Path ".env")) {
        Write-Host "⚠ Warning: .env file not found" -ForegroundColor Yellow
        Write-Host "  Using system environment variables" -ForegroundColor Yellow
        Write-Host ""
    }

    # Run the migration
    Write-Host "Executing migration 016_packages_table.sql..." -ForegroundColor Yellow
    go run cmd/migrate/main.go
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "  1. Insert default packages for your tenant:" -ForegroundColor White
        Write-Host "     SELECT insert_default_packages_for_tenant('<tenant-uuid>', <service-id>);" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  2. Verify the packages table:" -ForegroundColor White
        Write-Host "     SELECT * FROM packages;" -ForegroundColor Gray
        Write-Host ""
    } else {
        Write-Host ""
        Write-Host "✗ Migration failed with exit code: $LASTEXITCODE" -ForegroundColor Red
        Write-Host ""
        Write-Host "Troubleshooting:" -ForegroundColor Yellow
        Write-Host "  1. Check database connection settings in .env" -ForegroundColor White
        Write-Host "  2. Ensure PostgreSQL is running" -ForegroundColor White
        Write-Host "  3. Verify previous migrations have been applied" -ForegroundColor White
        Write-Host ""
        exit $LASTEXITCODE
    }
} finally {
    Pop-Location
}
