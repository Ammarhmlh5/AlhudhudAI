# ALHudhud Backend Watchdog
# Monitors the backend health endpoint and restarts backend_new.exe when it
# stops responding. Run it detached (e.g. via Task Scheduler at logon) so it
# keeps the backend alive:
#   powershell -ExecutionPolicy Bypass -File watchdog.ps1

param(
    [string]$BackendExe = "backend_new.exe",
    [string]$WorkingDir  = $PSScriptRoot,
    [int]$Port           = 8085,
    [int]$CheckInterval  = 15,
    [int]$MaxFailedChecks = 4
)

$HealthUrl    = "http://localhost:$Port/api/v1/health"
$LogDir       = Join-Path $WorkingDir "logs"
$WatchdogLog  = Join-Path $LogDir "watchdog.log"

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

$script:failedCount = 0
$script:generation  = 0

function Write-WatchLog {
    param([string]$Message)
    $stamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
    Write-Host "[watchdog] $Message"
    Add-Content -Path $WatchdogLog -Value "[$stamp] $Message"
}

function Get-BackendPidOnPort {
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($conn) { return $conn.OwningProcess }
    return $null
}

function Start-Backend {
    $script:generation++
    $stdout = Join-Path $LogDir "backend_$($script:generation).log"
    $stderr = Join-Path $LogDir "backend_error_$($script:generation).log"
    $exePath = Join-Path $WorkingDir $BackendExe

    try {
        $proc = Start-Process -FilePath $exePath `
            -WorkingDirectory $WorkingDir `
            -RedirectStandardOutput $stdout `
            -RedirectStandardError $stderr `
            -WindowStyle Hidden `
            -PassThru
        Write-WatchLog "started $exePath (PID $($proc.Id), gen $($script:generation)) -> $stdout / $stderr"
    } catch {
        Write-WatchLog "FAILED to start ${exePath}: $($_.Exception.Message)"
    }
    $script:failedCount = 0
}

function Wait-ForBackendHealthy {
    param([int]$TimeoutSeconds = 60)
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if (Test-BackendHealthy) { return $true }
        Start-Sleep -Seconds 3
    }
    return $false
}

function Test-BackendHealthy {
    try {
        $resp = Invoke-WebRequest -Uri $HealthUrl -TimeoutSec 10 -UseBasicParsing -ErrorAction Stop
        return ($resp.StatusCode -eq 200)
    } catch {
        return $false
    }
}

Write-WatchLog "monitoring $HealthUrl every ${CheckInterval}s (restart after $MaxFailedChecks failures)"

# Bootstrap: start the backend right away if nothing is listening.
if (-not (Get-BackendPidOnPort)) {
    Write-WatchLog "no backend running - starting now"
    Start-Backend
    if (Wait-ForBackendHealthy -TimeoutSeconds 60) {
        Write-WatchLog "backend healthy after initial start"
    } else {
        Write-WatchLog "backend did not become healthy within 60s after initial start"
    }
}

while ($true) {
    try {
        $listenerPid = Get-BackendPidOnPort

        if ($listenerPid) {
            if (Test-BackendHealthy) {
                if ($script:failedCount -gt 0) { Write-WatchLog "recovered (healthy again)" }
                $script:failedCount = 0
            } else {
                $script:failedCount++
                Write-WatchLog "listener PID $listenerPid present but unhealthy ($($script:failedCount)/$MaxFailedChecks)"
            }
        } else {
            $script:failedCount++
            Write-WatchLog "no listener on port $Port ($($script:failedCount)/$MaxFailedChecks)"
        }

        if ($script:failedCount -ge $MaxFailedChecks) {
            Write-WatchLog "backend unresponsive - restarting"
            $listenerPid = Get-BackendPidOnPort
            if ($listenerPid) {
                Stop-Process -Id $listenerPid -Force -ErrorAction SilentlyContinue
                Start-Sleep -Seconds 3
            }
            Start-Backend
            if (-not (Wait-ForBackendHealthy -TimeoutSeconds 60)) {
                Write-WatchLog "backend did not become healthy within 60s"
            } else {
                Write-WatchLog "backend healthy after restart"
            }
        }
    } catch {
        Write-WatchLog "watchdog error: $($_.Exception.Message)"
    }

    Start-Sleep -Seconds $CheckInterval
}
