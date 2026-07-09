#!/usr/bin/env pwsh
# Run Balance History RLS Migration Script
# This script adds Row Level Security policies to the balance_history table

Write-Host "=== Running Balance History RLS Migration ===" -ForegroundColor Cyan
Write-Host ""

# Change to backend directory
Push-Location $PSScriptRoot

try {
    # Run the migration
    Write-Host "Executing migration 014_balance_history_rls.sql..." -ForegroundColor Yellow
    go run cmd/migrate/main.go migrations
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "The following RLS policies have been added to balance_history:" -ForegroundColor Cyan
        Write-Host "  - tenant_isolation_policy (SELECT)" -ForegroundColor White
        Write-Host "  - tenant_isolation_insert_policy (INSERT)" -ForegroundColor White
        Write-Host "  - tenant_isolation_update_policy (UPDATE - DENIED)" -ForegroundColor White
        Write-Host "  - tenant_isolation_delete_policy (DELETE - DENIED)" -ForegroundColor White
        Write-Host ""
        Write-Host "Balance history is now fully isolated per tenant and immutable." -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "✗ Migration failed with exit code: $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
} finally {
    Pop-Location
}
