# Run Bug 14 Preservation Tests
# These tests verify that channel cache behavior for active channels is preserved

Write-Host "Running Bug 14 Preservation Tests..." -ForegroundColor Cyan
Write-Host "These tests verify that active channel cache behavior is maintained" -ForegroundColor Yellow
Write-Host ""

# Run the tests
go test -v -count=1 -run "TestProperty_Bug14_Preservation|TestBug14_Preservation" ./internal/redis/

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ All Bug 14 preservation tests PASSED" -ForegroundColor Green
    Write-Host "Active channel cache behavior is preserved on UNFIXED code" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "❌ Some Bug 14 preservation tests FAILED" -ForegroundColor Red
    Write-Host "This indicates an issue with the test setup or baseline behavior" -ForegroundColor Red
    exit 1
}
