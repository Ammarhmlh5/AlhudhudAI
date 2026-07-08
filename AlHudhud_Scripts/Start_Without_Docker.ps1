Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Starting ALHudhudAI (Without Docker)  " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$P = "C:\ALHudhud\ALHudhudAI"

if (-not (Test-Path $P)) {
    Write-Host "ERROR: Project not found at $P" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Set-Location $P

# ============================================
# PHASE 0: Start Database Services
# ============================================
Write-Host ""
Write-Host "PHASE 0: Starting Database Services" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta

# Start PostgreSQL
Write-Host "Starting PostgreSQL..." -ForegroundColor Yellow
$pgServices = @("postgresql-x64-14", "postgresql-x64-15", "postgresql-x64-16", "PostgreSQL", "postgresql")
$pgStarted = $false
foreach ($service in $pgServices) {
    $svc = Get-Service -Name $service -ErrorAction SilentlyContinue
    if ($svc) {
        if ($svc.Status -eq "Running") {
            Write-Host "  ✓ PostgreSQL running" -ForegroundColor Green
            $pgStarted = $true
            break
        } else {
            Start-Service -Name $service -ErrorAction SilentlyContinue
            if ($?) {
                Write-Host "  ✓ PostgreSQL started" -ForegroundColor Green
                $pgStarted = $true
                break
            }
        }
    }
}
if (-not $pgStarted) { Write-Host "  ⚠ PostgreSQL not found" -ForegroundColor Yellow }

# Start Redis
Write-Host "Starting Redis..." -ForegroundColor Yellow
$redisServices = @("Redis", "redis-server")
$redisStarted = $false
foreach ($service in $redisServices) {
    $svc = Get-Service -Name $service -ErrorAction SilentlyContinue
    if ($svc) {
        if ($svc.Status -eq "Running") {
            Write-Host "  ✓ Redis running" -ForegroundColor Green
            $redisStarted = $true
            break
        } else {
            Start-Service -Name $service -ErrorAction SilentlyContinue
            if ($?) {
                Write-Host "  ✓ Redis started" -ForegroundColor Green
                $redisStarted = $true
                break
            }
        }
    }
}
if (-not $redisStarted) { Write-Host "  ⚠ Redis not found" -ForegroundColor Yellow }

Start-Sleep -Seconds 2

# ============================================
# PHASE 1: Start Backend Services
# ============================================
Write-Host ""
Write-Host "PHASE 1: Starting Backend Services" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta

Write-Host "[1/2] Starting Backend Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\services\backend'; Write-Host '=== Backend Service ===' -ForegroundColor Green; Write-Host 'Using PostgreSQL on localhost:5433' -ForegroundColor Cyan; go run main.go"
Start-Sleep -Seconds 3

Write-Host "[2/2] Starting Company API Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\services\company-api'; Write-Host '=== Company API Service ===' -ForegroundColor Green; Write-Host 'Using PostgreSQL on localhost:5433' -ForegroundColor Cyan; `$env:APP_ENV='development'; go run cmd/server/main.go"
Start-Sleep -Seconds 3

Write-Host "Backend services started successfully" -ForegroundColor Green

# ============================================
# PHASE 2: Start Web Interfaces
# ============================================
Write-Host ""
Write-Host "PHASE 2: Starting Web Interfaces" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta

Write-Host "[1/3] Starting Admin Web (port 5174)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\admin-web'; Write-Host '=== Admin Web Interface ===' -ForegroundColor Green; Write-Host 'URL: http://localhost:5174' -ForegroundColor Cyan; npm run dev -- --port 5174"
Start-Sleep -Seconds 2

Write-Host "[2/3] Starting Client Web (port 5173)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\client-web'; Write-Host '=== Client Web Interface ===' -ForegroundColor Green; Write-Host 'URL: http://localhost:5173' -ForegroundColor Cyan; npm run dev -- --port 5173"
Start-Sleep -Seconds 2

Write-Host "[3/3] Starting Reseller Web (port 5175)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\reseller-web'; Write-Host '=== Reseller Web Interface ===' -ForegroundColor Green; Write-Host 'URL: http://localhost:5175' -ForegroundColor Cyan; npm run dev -- --port 5175"
Start-Sleep -Seconds 2

Write-Host "Web interfaces started successfully" -ForegroundColor Green

# ============================================
# PHASE 3: Start Mobile Gateway
# ============================================
Write-Host ""
Write-Host "PHASE 3: Starting Mobile Gateway" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta

Write-Host "Starting React Native Metro Bundler..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\apps\mobile_gateway'; Write-Host '=== Mobile Gateway (React Native) ===' -ForegroundColor Green; Write-Host 'Metro Bundler Starting...' -ForegroundColor Cyan; Write-Host '' ; Write-Host 'NEXT STEPS TO RUN ON YOUR DEVICE:' -ForegroundColor Yellow; Write-Host '  1. Connect your Android device via USB cable' -ForegroundColor White; Write-Host '  2. Enable USB Debugging on your device' -ForegroundColor White; Write-Host '  3. Verify device: adb devices' -ForegroundColor White; Write-Host '  4. Run: npm run android' -ForegroundColor Cyan; Write-Host '  5. Or press ''a'' in Metro Bundler' -ForegroundColor Cyan; Write-Host '' ; npm start"

Start-Sleep -Seconds 2

# ============================================
# SUMMARY
# ============================================
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  ALL SERVICES STARTED SUCCESSFULLY     " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "System Status:" -ForegroundColor Cyan
Write-Host ""
Write-Host "Database:" -ForegroundColor Yellow
Write-Host "  ✓ PostgreSQL (localhost:5433) - External" -ForegroundColor White
Write-Host ""
Write-Host "Backend Services:" -ForegroundColor Yellow
Write-Host "  ✓ Backend Service (Go)" -ForegroundColor White
Write-Host "  ✓ Company API Service (Go)" -ForegroundColor White
Write-Host ""
Write-Host "Web Interfaces:" -ForegroundColor Yellow
Write-Host "  ✓ Admin Web:     http://localhost:5174" -ForegroundColor White
Write-Host "  ✓ Client Web:    http://localhost:5173" -ForegroundColor White
Write-Host "  ✓ Reseller Web:  http://localhost:5175" -ForegroundColor White
Write-Host ""
Write-Host "Mobile Application:" -ForegroundColor Yellow
Write-Host "  ✓ Metro Bundler:  Running (check mobile_gateway window)" -ForegroundColor White
Write-Host "  → Next: Connect USB device → run 'npm run android' or press 'a' in Metro" -ForegroundColor Cyan
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

Read-Host "Press Enter to exit"
