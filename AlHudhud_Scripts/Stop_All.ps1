Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Stopping ALHudhudAI Services          " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$P = "C:\ALHudhud\ALHudhudAI"

if (-not (Test-Path $P)) {
    Write-Host "ERROR: Project not found at $P" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Set-Location $P

# Stop Docker containers
Write-Host "[1/3] Stopping Docker containers..." -ForegroundColor Yellow

# Try docker compose (new) first, then docker-compose (old)
$dockerComposeCmd = $null
if (Get-Command "docker" -ErrorAction SilentlyContinue) {
    $testCompose = docker compose version 2>&1
    if ($LASTEXITCODE -eq 0) {
        $dockerComposeCmd = "docker compose"
    }
}

if (-not $dockerComposeCmd -and (Get-Command "docker-compose" -ErrorAction SilentlyContinue)) {
    $dockerComposeCmd = "docker-compose"
}

if ($dockerComposeCmd) {
    Write-Host "Using: $dockerComposeCmd" -ForegroundColor Gray
    Invoke-Expression "$dockerComposeCmd down"
} else {
    Write-Host "WARNING: Docker Compose not found, skipping container shutdown" -ForegroundColor Yellow
}

# Stop Go processes (Backend services)
Write-Host "[2/3] Stopping Go backend processes..." -ForegroundColor Yellow
$goProcesses = Get-Process -Name go, main -ErrorAction SilentlyContinue
if ($goProcesses) {
    $goProcesses | Stop-Process -Force
    Write-Host "  Stopped $($goProcesses.Count) Go process(es)" -ForegroundColor Gray
} else {
    Write-Host "  No Go processes found" -ForegroundColor Gray
}

# Stop Node processes (Web interfaces + React Native Metro)
Write-Host "[3/3] Stopping Node.js processes (Web + Mobile)..." -ForegroundColor Yellow
$nodeProcesses = Get-Process -Name node -ErrorAction SilentlyContinue
if ($nodeProcesses) {
    $nodeProcesses | Stop-Process -Force
    Write-Host "  Stopped $($nodeProcesses.Count) Node.js process(es)" -ForegroundColor Gray
} else {
    Write-Host "  No Node.js processes found" -ForegroundColor Gray
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  All Services Stopped Successfully     " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

Read-Host "Press Enter to exit"
