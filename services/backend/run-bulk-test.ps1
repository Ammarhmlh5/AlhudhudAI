# Bulk Message Processing Test Runner (PowerShell)
# Tests processing 10,000 messages in bulk

Write-Host "🚀 Bulk Message Processing Test" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Check if infrastructure is running
Write-Host "📋 Checking infrastructure..." -ForegroundColor Yellow

$postgresRunning = docker ps | Select-String "postgres"
$redisRunning = docker ps | Select-String "redis"

if (-not $postgresRunning) {
    Write-Host "❌ PostgreSQL is not running. Please start infrastructure first:" -ForegroundColor Red
    Write-Host "   .\start-infrastructure.bat" -ForegroundColor Yellow
    exit 1
}

if (-not $redisRunning) {
    Write-Host "❌ Redis is not running. Please start infrastructure first:" -ForegroundColor Red
    Write-Host "   .\start-infrastructure.bat" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Infrastructure is running" -ForegroundColor Green
Write-Host ""

# Run the test
Write-Host "🧪 Running bulk message processing test..." -ForegroundColor Yellow
Write-Host "   This will send and process 10,000 messages" -ForegroundColor Gray
Write-Host "   Expected duration: ~2 minutes" -ForegroundColor Gray
Write-Host ""

Set-Location services/backend

go test -v -timeout 10m `
    -run TestBulkMessageProcessing `
    ./internal/integration/

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Test completed successfully!" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "❌ Test failed!" -ForegroundColor Red
    exit 1
}
