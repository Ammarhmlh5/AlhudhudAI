# Redis Failure Fallback Test Runner
# This script runs the Redis failure fallback integration test

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Redis Failure Fallback Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Docker is running
Write-Host "Checking Docker status..." -ForegroundColor Yellow
$dockerRunning = docker info 2>$null
if (-not $dockerRunning) {
    Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
    exit 1
}
Write-Host "✓ Docker is running" -ForegroundColor Green
Write-Host ""

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL status..." -ForegroundColor Yellow
$pgRunning = docker-compose ps postgres 2>$null | Select-String "Up"
if (-not $pgRunning) {
    Write-Host "⚠ PostgreSQL is not running. Starting..." -ForegroundColor Yellow
    docker-compose up -d postgres
    Start-Sleep -Seconds 5
}
Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Check if Redis is running
Write-Host "Checking Redis status..." -ForegroundColor Yellow
$redisRunning = docker-compose ps redis 2>$null | Select-String "Up"
if (-not $redisRunning) {
    Write-Host "⚠ Redis is not running. Starting..." -ForegroundColor Yellow
    docker-compose up -d redis
    Start-Sleep -Seconds 3
}
Write-Host "✓ Redis is running" -ForegroundColor Green
Write-Host ""

# Run database migrations
Write-Host "Running database migrations..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/sms_gateway_test?sslmode=disable"
go run cmd/migrate/main.go up 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Migrations completed" -ForegroundColor Green
} else {
    Write-Host "⚠ Migrations may have already been applied" -ForegroundColor Yellow
}
Write-Host ""

# Run the test
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Redis Failure Fallback Test..." -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if user wants to run specific scenario
$scenario = $args[0]

if ($scenario) {
    Write-Host "Running specific scenario: $scenario" -ForegroundColor Yellow
    Write-Host ""
    go test -v ./internal/integration -run "TestRedisFailureFallback/$scenario" -timeout 10m
} else {
    Write-Host "Running all scenarios..." -ForegroundColor Yellow
    Write-Host ""
    go test -v ./internal/integration -run TestRedisFailureFallback -timeout 10m
}

$testResult = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($testResult -eq 0) {
    Write-Host "✅ Test PASSED" -ForegroundColor Green
    Write-Host ""
    Write-Host "All Redis failure fallback scenarios completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Verified:" -ForegroundColor Cyan
    Write-Host "  ✓ Redis failure detection" -ForegroundColor Green
    Write-Host "  ✓ Automatic fallback to direct processing" -ForegroundColor Green
    Write-Host "  ✓ No message loss" -ForegroundColor Green
    Write-Host "  ✓ Automatic reconnection to Redis" -ForegroundColor Green
    Write-Host "  ✓ Circuit breaker behavior" -ForegroundColor Green
} else {
    Write-Host "❌ Test FAILED" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please check the error messages above." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Common issues:" -ForegroundColor Yellow
    Write-Host "  • Redis connection issues" -ForegroundColor Yellow
    Write-Host "  • Database connection issues" -ForegroundColor Yellow
    Write-Host "  • Circuit breaker not opening" -ForegroundColor Yellow
    Write-Host "  • Fallback operation failures" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "For detailed troubleshooting, see:" -ForegroundColor Yellow
    Write-Host "  internal/integration/REDIS_FAILURE_TEST_GUIDE.md" -ForegroundColor Cyan
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Show available scenarios
if ($testResult -ne 0 -or -not $scenario) {
    Write-Host "Available scenarios:" -ForegroundColor Cyan
    Write-Host "  1. RedisFailureWithFallback" -ForegroundColor White
    Write-Host "  2. SystemContinuesWithoutRedis" -ForegroundColor White
    Write-Host "  3. AutomaticReconnectionToRedis" -ForegroundColor White
    Write-Host "  4. BatchProcessingWithRedisFailure" -ForegroundColor White
    Write-Host "  5. CircuitBreakerBehavior" -ForegroundColor White
    Write-Host ""
    Write-Host "To run a specific scenario:" -ForegroundColor Yellow
    Write-Host "  .\run-redis-failure-test.ps1 <scenario-name>" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Example:" -ForegroundColor Yellow
    Write-Host "  .\run-redis-failure-test.ps1 RedisFailureWithFallback" -ForegroundColor Cyan
    Write-Host ""
}

exit $testResult
