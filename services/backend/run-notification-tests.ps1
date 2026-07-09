#!/usr/bin/env pwsh

Write-Host "=== Notification Service Tests ===" -ForegroundColor Cyan
Write-Host ""

# Navigate to backend directory
Push-Location services/backend

try {
    Write-Host "Running Notification Service Tests..." -ForegroundColor Yellow
    Write-Host ""
    
    # Run tests
    go test -v ./internal/notifications
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "All notification tests passed!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Implemented notifications:" -ForegroundColor Cyan
        Write-Host "  - Renewal reminder 3 days before" -ForegroundColor Gray
        Write-Host "  - Renewal success" -ForegroundColor Gray
        Write-Host "  - Renewal failure" -ForegroundColor Gray
        Write-Host "  - Balance topup" -ForegroundColor Gray
        Write-Host "  - Low balance warning" -ForegroundColor Gray
        Write-Host "  - Low messages warning" -ForegroundColor Gray
    } else {
        Write-Host ""
        Write-Host "Some notification tests failed!" -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}
