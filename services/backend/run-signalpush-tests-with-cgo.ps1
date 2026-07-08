# Enable CGO and run Signal/Push tests
Write-Host "Enabling CGO and running Signal/Push tests..." -ForegroundColor Cyan

$env:CGO_ENABLED = "1"

Write-Host "CGO_ENABLED = $env:CGO_ENABLED" -ForegroundColor Yellow
Write-Host "Running tests..." -ForegroundColor Yellow

go test ./internal/admin/signalpush/... -v

if ($LASTEXITCODE -eq 0) {
    Write-Host "`nAll tests passed!" -ForegroundColor Green
} else {
    Write-Host "`nSome tests failed!" -ForegroundColor Red
}
