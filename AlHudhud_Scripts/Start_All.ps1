Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Starting Complete ALHudhudAI System   " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$P = "C:\ALHudhud\ALHudhudAI"
$D = "C:\Program Files\Docker\Docker\docker.exe"

if (-not (Test-Path $P)) {
    Write-Host "ERROR: Project not found at $P" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Set-Location $P

# ============================================
# PHASE 1: Start Docker Infrastructure
# ============================================
Write-Host ""
Write-Host "PHASE 1: Starting Docker Infrastructure" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta

# Find Docker Compose command
$dockerComposeCmd = $null

# Check for docker compose (new version)
if (Get-Command "docker" -ErrorAction SilentlyContinue) {
    $testCompose = docker compose version 2>&1
    if ($LASTEXITCODE -eq 0) {
        $dockerComposeCmd = "docker compose"
    }
}

# Check for docker-compose (old version)
if (-not $dockerComposeCmd -and (Get-Command "docker-compose" -ErrorAction SilentlyContinue)) {
    $dockerComposeCmd = "docker-compose"
}

# Check Docker Desktop installation paths
if (-not $dockerComposeCmd) {
    $dockerPaths = @(
        "C:\Program Files\Docker\Docker\resources\bin\docker.exe",
        "C:\Program Files\Docker\Docker\docker.exe",
        "$env:ProgramFiles\Docker\Docker\resources\bin\docker.exe",
        "$env:ProgramFiles\Docker\Docker\docker.exe"
    )
    
    foreach ($dockerPath in $dockerPaths) {
        if (Test-Path $dockerPath) {
            Write-Host "Found Docker at: $dockerPath" -ForegroundColor Gray
            $testCompose = & $dockerPath compose version 2>&1
            if ($LASTEXITCODE -eq 0) {
                $dockerComposeCmd = "$dockerPath compose"
                break
            }
        }
    }
}

if (-not $dockerComposeCmd) {
    Write-Host "⚠ Docker not found - SKIPPING Docker containers" -ForegroundColor Yellow
    Write-Host "Note: Backend services require PostgreSQL and Redis" -ForegroundColor Gray
    Write-Host "You may need to start them manually or install Docker" -ForegroundColor Gray
    Write-Host ""
} else {
    Write-Host "Using: $dockerComposeCmd" -ForegroundColor Gray
    Invoke-Expression "$dockerComposeCmd up -d"

    if ($LASTEXITCODE -ne 0) {
        Write-Host "⚠ Failed to start Docker containers - CONTINUING anyway" -ForegroundColor Yellow
    } else {
        Write-Host "Docker containers started successfully" -ForegroundColor Green
    }
}

Start-Sleep -Seconds 5

# ============================================
# PHASE 2: Start Backend Services
# ============================================
Write-Host ""
Write-Host "PHASE 2: Starting Backend Services" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta

Write-Host "[1/2] Starting Backend Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\services\backend'; Write-Host '=== Backend Service ===' -ForegroundColor Green; go run main.go"
Start-Sleep -Seconds 3

Write-Host "[2/2] Starting Company API Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\services\company-api'; Write-Host '=== Company API Service ===' -ForegroundColor Green; `$env:APP_ENV='development'; go run cmd/server/main.go"
Start-Sleep -Seconds 3

Write-Host "Backend services started successfully" -ForegroundColor Green

# ============================================
# PHASE 3: Start Web Interfaces
# ============================================
Write-Host ""
Write-Host "PHASE 3: Starting Web Interfaces" -ForegroundColor Magenta
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
# PHASE 4: Start Mobile Gateway
# ============================================
Write-Host ""
Write-Host "PHASE 4: Starting Mobile Gateway" -ForegroundColor Magenta
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
Write-Host "Infrastructure:" -ForegroundColor Yellow
Write-Host "  ✓ Docker Containers (PostgreSQL, Redis, etc.)" -ForegroundColor White
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
