# Bug 21 Performance Benchmarking Script
Write-Host "=== BUG 21 PERFORMANCE BENCHMARKING ===" -ForegroundColor Cyan
Write-Host "Testing different batch processing configurations..." -ForegroundColor Yellow
Write-Host ""

# Run the benchmark tests
go test -v -timeout 10m -run "TestBug21" ./internal/worker/

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n=== BENCHMARK COMPLETED SUCCESSFULLY ===" -ForegroundColor Green
} else {
    Write-Host "`n=== BENCHMARK FAILED ===" -ForegroundColor Red
    exit 1
}
