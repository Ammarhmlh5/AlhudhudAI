Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Starting ALHudhudAI Backend Services  " -ForegroundColor Cyan
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

# Start Docker containers
Write-Host "[1/3] Starting Docker containers..." -ForegroundColor Yellow

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
    Write-Host "  ⚠ Docker not found - SKIPPING Docker containers" -ForegroundColor Yellow
    Write-Host "  Note: Backend services require PostgreSQL and Redis" -ForegroundColor Gray
    Write-Host "  You may need to start them manually or install Docker" -ForegroundColor Gray
} else {
    Write-Host "Using: $dockerComposeCmd" -ForegroundColor Gray
    Invoke-Expression "$dockerComposeCmd up -d"

    if ($LASTEXITCODE -ne 0) {
        Write-Host "  ⚠ Failed to start Docker containers - CONTINUING anyway" -ForegroundColor Yellow
    } else {
        Write-Host "  ✓ Docker containers started" -ForegroundColor Green
    }
}

Start-Sleep -Seconds 3

Start-Sleep -Seconds 3

# Start Backend Service
Write-Host "[2/3] Starting Backend Service (services/backend)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\services\backend'; Write-Host 'Backend Service Starting...' -ForegroundColor Green; go run main.go"

Start-Sleep -Seconds 2

# Start Company API Service
Write-Host "[3/3] Starting Company API Service (services/company-api)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit","-Command","cd '$P\services\company-api'; Write-Host 'Company API Service Starting...' -ForegroundColor Green; `$env:APP_ENV='development'; .\server.exe"

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Backend Services Started Successfully " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Services running:" -ForegroundColor Cyan
Write-Host "  - Docker Containers (PostgreSQL, Redis, etc.)" -ForegroundColor White
Write-Host "  - Backend Service (Go)" -ForegroundColor White
Write-Host "  - Company API Service (Go)" -ForegroundColor White
Write-Host ""

Read-Host "Press Enter to exit"
