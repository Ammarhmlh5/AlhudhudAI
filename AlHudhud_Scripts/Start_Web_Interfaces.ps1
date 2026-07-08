Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Starting ALHudhudAI Web Interfaces    " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$P = "C:\ALHudhud\ALHudhudAI"

if (-not (Test-Path $P)) {
    Write-Host "ERROR: Project not found at $P" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Set-Location $P

# Start Admin Web Interface
Write-Host "[1/4] Starting Admin Web (port 5174)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\admin-web'; Write-Host 'Admin Web Starting on http://localhost:5174' -ForegroundColor Green; npm run dev -- --port 5174"

Start-Sleep -Seconds 2

# Start Client Web Interface
Write-Host "[2/4] Starting Client Web (port 5173)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\client-web'; Write-Host 'Client Web Starting on http://localhost:5173' -ForegroundColor Green; npm run dev -- --port 5173"

Start-Sleep -Seconds 2

# Start Reseller Web Interface
Write-Host "[3/4] Starting Reseller Web (port 5175)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\reseller-web'; Write-Host 'Reseller Web Starting on http://localhost:5175' -ForegroundColor Green; npm run dev -- --port 5175"

Start-Sleep -Seconds 2

# Start Mobile Gateway (React Native Metro Bundler)
Write-Host "[4/4] Starting Mobile Gateway (React Native Metro)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\mobile_gateway'; Write-Host 'Mobile Gateway Metro Bundler Starting on port 8085...' -ForegroundColor Green; Write-Host '' ; Write-Host 'NEXT STEPS:' -ForegroundColor Yellow; Write-Host '  1. Connect your Android device via USB' -ForegroundColor White; Write-Host '  2. Enable USB Debugging on your device' -ForegroundColor White; Write-Host '  3. Run: npx react-native run-android --port 8085' -ForegroundColor Cyan; Write-Host '  4. Or press ''a'' in Metro Bundler to run on Android' -ForegroundColor Cyan; Write-Host '' ; npx react-native start --port 8085"

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Web Interfaces Started Successfully   " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Interfaces running:" -ForegroundColor Cyan
Write-Host "  - Admin Web:     http://localhost:5174" -ForegroundColor White
Write-Host "  - Client Web:    http://localhost:5173" -ForegroundColor White
Write-Host "  - Reseller Web:  http://localhost:5175" -ForegroundColor White
Write-Host "  - Mobile Gateway: Metro Bundler (Connect USB device → run 'npm run android' or press 'a')" -ForegroundColor White
Write-Host ""

Read-Host "Press Enter to exit"
