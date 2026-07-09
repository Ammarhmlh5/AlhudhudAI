# PowerShell script to run Multi-Tenant Scenarios integration test
# المهمة 15.4: اختبار Multi-Tenant Scenarios

Write-Host "🏢 Multi-Tenant Scenarios Integration Test" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "🔍 Checking Docker status..." -ForegroundColor Yellow
$dockerRunning = docker ps 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
    Write-Host ""
    Write-Host "To start infrastructure:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d" -ForegroundColor White
    exit 1
}

Write-Host "✅ Docker is running" -ForegroundColor Green
Write-Host ""

# Check if PostgreSQL is running
Write-Host "🔍 Checking PostgreSQL..." -ForegroundColor Yellow
$pgRunning = docker ps --filter "name=postgres" --filter "status=running" -q
if (-not $pgRunning) {
    Write-Host "❌ PostgreSQL is not running" -ForegroundColor Red
    Write-Host ""
    Write-Host "To start infrastructure:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d postgres" -ForegroundColor White
    exit 1
}

Write-Host "✅ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Check if Redis is running
Write-Host "🔍 Checking Redis..." -ForegroundColor Yellow
$redisRunning = docker ps --filter "name=redis" --filter "status=running" -q
if (-not $redisRunning) {
    Write-Host "❌ Redis is not running" -ForegroundColor Red
    Write-Host ""
    Write-Host "To start infrastructure:" -ForegroundColor Yellow
    Write-Host "  docker-compose up -d redis" -ForegroundColor White
    exit 1
}

Write-Host "✅ Redis is running" -ForegroundColor Green
Write-Host ""

# Run migrations
Write-Host "🔄 Running database migrations..." -ForegroundColor Yellow
Push-Location services/backend
go run cmd/migrate/main.go up
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Migration failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location

Write-Host "✅ Migrations completed" -ForegroundColor Green
Write-Host ""

# Run the test
Write-Host "🧪 Running Multi-Tenant Scenarios test..." -ForegroundColor Yellow
Write-Host ""
Write-Host "Test objectives:" -ForegroundColor Cyan
Write-Host "  1. Send messages from 3+ tenants concurrently" -ForegroundColor White
Write-Host "  2. Verify complete data isolation between tenants" -ForegroundColor White
Write-Host "  3. Verify tenant operations don't affect other tenants" -ForegroundColor White
Write-Host "  4. Verify rate limiting is isolated per tenant" -ForegroundColor White
Write-Host "  5. Verify channels and messages are properly isolated" -ForegroundColor White
Write-Host "  6. Verify no data leakage between tenants" -ForegroundColor White
Write-Host ""

Push-Location services/backend/internal/integration

# Run with verbose output
go test -v -run TestMultiTenant -timeout 5m

$testResult = $LASTEXITCODE

Pop-Location

Write-Host ""
if ($testResult -eq 0) {
    Write-Host "✅ Multi-Tenant Scenarios test PASSED!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Verified:" -ForegroundColor Green
    Write-Host "  ✓ Complete data isolation between tenants" -ForegroundColor White
    Write-Host "  ✓ No cross-tenant data leakage" -ForegroundColor White
    Write-Host "  ✓ Independent rate limiting per tenant" -ForegroundColor White
    Write-Host "  ✓ Isolated balance operations" -ForegroundColor White
    Write-Host "  ✓ Isolated channel management" -ForegroundColor White
    Write-Host ""
    Write-Host "📋 Requirements validated: 1.1, 3.4, 5.1" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "  1. ✅ Task 15.4 completed" -ForegroundColor Green
    Write-Host "  2. ➡️  Continue to Task 15.5 (WebSocket Reconnection)" -ForegroundColor White
    Write-Host "  3. ➡️  Continue to Task 15.6 (Database Failure Recovery)" -ForegroundColor White
} else {
    Write-Host "❌ Multi-Tenant Scenarios test FAILED!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please check the test output above for details." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Common issues:" -ForegroundColor Yellow
    Write-Host "  - Database connection issues" -ForegroundColor White
    Write-Host "  - Redis connection issues" -ForegroundColor White
    Write-Host "  - Missing migrations" -ForegroundColor White
    Write-Host "  - Port conflicts" -ForegroundColor White
    exit 1
}
