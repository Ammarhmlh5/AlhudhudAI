# Run Bug 13 Verification Test
# This test verifies that the cache invalidation fix works correctly

Write-Host "Running Bug 13 Verification Test..." -ForegroundColor Cyan
Write-Host "This test verifies that InvalidateChannelOnOffline() works correctly" -ForegroundColor Yellow
Write-Host ""

# Run the test
go test -v -count=1 -run "TestBug13_CacheNotInvalidatedOnChannelOffline_Exploration" ./internal/redis/

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Bug 13 verification test PASSED" -ForegroundColor Green
    Write-Host "Cache invalidation is working correctly - bug is FIXED" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "❌ Bug 13 verification test FAILED" -ForegroundColor Red
    Write-Host "Cache invalidation is not working as expected" -ForegroundColor Red
    exit 1
}
