Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Checking ALHudhudAI Requirements      " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allGood = $true

# Check Project Path
Write-Host "[1/6] Checking Project Path..." -ForegroundColor Yellow
$P = "C:\ALHudhud\ALHudhudAI"
if (Test-Path $P) {
    Write-Host "  ✓ Project found at: $P" -ForegroundColor Green
} else {
    Write-Host "  ✗ Project NOT found at: $P" -ForegroundColor Red
    $allGood = $false
}

# Check Docker
Write-Host "[2/6] Checking Docker..." -ForegroundColor Yellow
$dockerFound = $false
$dockerComposeFound = $false

if (Get-Command "docker" -ErrorAction SilentlyContinue) {
    $dockerVersion = docker --version 2>&1
    Write-Host "  ✓ Docker found: $dockerVersion" -ForegroundColor Green
    $dockerFound = $true
    
    # Check docker compose
    $testCompose = docker compose version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ✓ Docker Compose (new) found: $testCompose" -ForegroundColor Green
        $dockerComposeFound = $true
    }
} else {
    Write-Host "  ✗ Docker NOT found" -ForegroundColor Red
    $allGood = $false
}

if (-not $dockerComposeFound -and (Get-Command "docker-compose" -ErrorAction SilentlyContinue)) {
    $composeVersion = docker-compose --version 2>&1
    Write-Host "  ✓ Docker Compose (old) found: $composeVersion" -ForegroundColor Green
    $dockerComposeFound = $true
}

if (-not $dockerComposeFound) {
    Write-Host "  ✗ Docker Compose NOT found" -ForegroundColor Red
    $allGood = $false
}

# Check Go
Write-Host "[3/6] Checking Go..." -ForegroundColor Yellow
if (Get-Command "go" -ErrorAction SilentlyContinue) {
    $goVersion = go version 2>&1
    Write-Host "  ✓ Go found: $goVersion" -ForegroundColor Green
} else {
    Write-Host "  ✗ Go NOT found" -ForegroundColor Red
    $allGood = $false
}

# Check Node.js
Write-Host "[4/6] Checking Node.js..." -ForegroundColor Yellow
if (Get-Command "node" -ErrorAction SilentlyContinue) {
    $nodeVersion = node --version 2>&1
    Write-Host "  ✓ Node.js found: $nodeVersion" -ForegroundColor Green
} else {
    Write-Host "  ✗ Node.js NOT found" -ForegroundColor Red
    $allGood = $false
}

# Check npm
Write-Host "[5/6] Checking npm..." -ForegroundColor Yellow
if (Get-Command "npm" -ErrorAction SilentlyContinue) {
    $npmVersion = npm --version 2>&1
    Write-Host "  ✓ npm found: v$npmVersion" -ForegroundColor Green
} else {
    Write-Host "  ✗ npm NOT found" -ForegroundColor Red
    $allGood = $false
}

# Check ADB (for mobile development)
Write-Host "[6/6] Checking ADB (Android Debug Bridge)..." -ForegroundColor Yellow
if (Get-Command "adb" -ErrorAction SilentlyContinue) {
    $adbVersion = adb version 2>&1 | Select-Object -First 1
    Write-Host "  ✓ ADB found: $adbVersion" -ForegroundColor Green
} else {
    Write-Host "  ⚠ ADB NOT found (optional for mobile development)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

if ($allGood) {
    Write-Host "  ✓ All Requirements Met!               " -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "You can now run:" -ForegroundColor Cyan
    Write-Host "  - 0-تشغيل_الكل.bat (Start everything)" -ForegroundColor White
    Write-Host "  - 2-تشغيل_السيرفر.bat (Start backend only)" -ForegroundColor White
    Write-Host "  - 3-تشغيل_الواجهات.bat (Start web interfaces)" -ForegroundColor White
} else {
    Write-Host "  ✗ Some Requirements Missing            " -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install missing components:" -ForegroundColor Yellow
    Write-Host "  - Docker Desktop: https://www.docker.com/products/docker-desktop" -ForegroundColor White
    Write-Host "  - Go: https://golang.org/dl/" -ForegroundColor White
    Write-Host "  - Node.js: https://nodejs.org/" -ForegroundColor White
}

Write-Host ""
Read-Host "Press Enter to exit"
