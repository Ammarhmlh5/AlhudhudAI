Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Starting Database Services            " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Start PostgreSQL
Write-Host "[1/2] Starting PostgreSQL..." -ForegroundColor Yellow

$pgServices = @(
    "postgresql-x64-14",
    "postgresql-x64-15",
    "postgresql-x64-16",
    "PostgreSQL",
    "postgresql"
)

$pgStarted = $false
foreach ($service in $pgServices) {
    $svc = Get-Service -Name $service -ErrorAction SilentlyContinue
    if ($svc) {
        if ($svc.Status -eq "Running") {
            Write-Host "  ✓ PostgreSQL already running ($service)" -ForegroundColor Green
            $pgStarted = $true
            break
        } else {
            Write-Host "  Starting $service..." -ForegroundColor Gray
            Start-Service -Name $service -ErrorAction SilentlyContinue
            if ($?) {
                Write-Host "  ✓ PostgreSQL started ($service)" -ForegroundColor Green
                $pgStarted = $true
                break
            }
        }
    }
}

if (-not $pgStarted) {
    Write-Host "  ⚠ PostgreSQL service not found or failed to start" -ForegroundColor Yellow
    Write-Host "  Please start PostgreSQL manually" -ForegroundColor Gray
}

# Start Redis
Write-Host "[2/2] Starting Redis..." -ForegroundColor Yellow

$redisServices = @(
    "Redis",
    "redis-server"
)

$redisStarted = $false
foreach ($service in $redisServices) {
    $svc = Get-Service -Name $service -ErrorAction SilentlyContinue
    if ($svc) {
        if ($svc.Status -eq "Running") {
            Write-Host "  ✓ Redis already running ($service)" -ForegroundColor Green
            $redisStarted = $true
            break
        } else {
            Write-Host "  Starting $service..." -ForegroundColor Gray
            Start-Service -Name $service -ErrorAction SilentlyContinue
            if ($?) {
                Write-Host "  ✓ Redis started ($service)" -ForegroundColor Green
                $redisStarted = $true
                break
            }
        }
    }
}

if (-not $redisStarted) {
    Write-Host "  ⚠ Redis service not found or failed to start" -ForegroundColor Yellow
    Write-Host "  Please start Redis manually" -ForegroundColor Gray
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Database Services Check Complete      " -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

Read-Host "Press Enter to exit"
