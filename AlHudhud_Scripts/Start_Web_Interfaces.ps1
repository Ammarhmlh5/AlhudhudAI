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

# Start Mobile Application (Expo Dev Server)
Write-Host "[4/4] Starting AlHudhud Mobile (Expo)..." -ForegroundColor Yellow
$MobileDir = "C:\ALHudhud\ALHudhud\ALHudhudMobile"
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$MobileDir'; Write-Host 'AlHudhud Mobile Expo Dev Server Starting...' -ForegroundColor Green; Write-Host '' ; Write-Host 'NEXT STEPS:' -ForegroundColor Yellow; Write-Host '  1. Install Expo Go on your phone' -ForegroundColor White; Write-Host '  2. Scan QR code shown in terminal' -ForegroundColor White; Write-Host '  3. Press ''a'' for Android emulator' -ForegroundColor Cyan; Write-Host '' ; npx expo start"

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Web Interfaces Started Successfully   " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Interfaces running:" -ForegroundColor Cyan
Write-Host "  - Admin Web:      http://localhost:5174" -ForegroundColor White
Write-Host "  - Client Web:     http://localhost:5173" -ForegroundColor White
Write-Host "  - Reseller Web:   http://localhost:5175" -ForegroundColor White
Write-Host "  - AlHudhud Mobile: Expo Dev Server (Scan QR with Expo Go)" -ForegroundColor White
Write-Host ""

Read-Host "Press Enter to exit"
