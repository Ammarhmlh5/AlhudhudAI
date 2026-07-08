# Resource Monitoring Script for Load Testing
# Monitors CPU, Memory, and Database connections during load tests

param(
    [int]$IntervalSeconds = 5,
    [string]$OutputFile = "results/resource-monitoring.csv",
    [int]$DurationMinutes = 15
)

Write-Host "🔍 Starting Resource Monitoring..." -ForegroundColor Cyan
Write-Host "   Interval: $IntervalSeconds seconds" -ForegroundColor Gray
Write-Host "   Duration: $DurationMinutes minutes" -ForegroundColor Gray
Write-Host "   Output: $OutputFile" -ForegroundColor Gray
Write-Host ""

# Create results directory if it doesn't exist
$resultsDir = Split-Path -Parent $OutputFile
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir -Force | Out-Null
}

# Initialize CSV file with headers
$headers = "Timestamp,CPU_Percent,Memory_MB,Memory_Percent,Backend_CPU,Backend_Memory_MB,Redis_CPU,Redis_Memory_MB,Postgres_CPU,Postgres_Memory_MB,DB_Connections,Redis_Keys,Redis_Memory_Used_MB"
$headers | Out-File -FilePath $OutputFile -Encoding UTF8

# Calculate end time
$endTime = (Get-Date).AddMinutes($DurationMinutes)
$iteration = 0

Write-Host "📊 Monitoring started. Press Ctrl+C to stop early." -ForegroundColor Green
Write-Host ""

try {
    while ((Get-Date) -lt $endTime) {
        $iteration++
        $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        
        # System-wide metrics
        $cpu = (Get-Counter '\Processor(_Total)\% Processor Time' -ErrorAction SilentlyContinue).CounterSamples.CookedValue
        $memory = Get-WmiObject Win32_OperatingSystem
        $totalMemoryMB = [math]::Round($memory.TotalVisibleMemorySize / 1KB, 2)
        $freeMemoryMB = [math]::Round($memory.FreePhysicalMemory / 1KB, 2)
        $usedMemoryMB = $totalMemoryMB - $freeMemoryMB
        $memoryPercent = [math]::Round(($usedMemoryMB / $totalMemoryMB) * 100, 2)
        
        # Docker container metrics
        $backendCPU = 0
        $backendMemoryMB = 0
        $redisCPU = 0
        $redisMemoryMB = 0
        $postgresCPU = 0
        $postgresMemoryMB = 0
        
        try {
            # Get backend container stats
            $backendStats = docker stats backend --no-stream --format "{{.CPUPerc}},{{.MemUsage}}" 2>$null
            if ($backendStats) {
                $parts = $backendStats -split ','
                $backendCPU = [math]::Round([double]($parts[0] -replace '%', ''), 2)
                $memParts = $parts[1] -split '/'
                $backendMemoryMB = [math]::Round([double]($memParts[0] -replace 'MiB|GiB', ''), 2)
            }
            
            # Get Redis container stats
            $redisStats = docker stats redis --no-stream --format "{{.CPUPerc}},{{.MemUsage}}" 2>$null
            if ($redisStats) {
                $parts = $redisStats -split ','
                $redisCPU = [math]::Round([double]($parts[0] -replace '%', ''), 2)
                $memParts = $parts[1] -split '/'
                $redisMemoryMB = [math]::Round([double]($memParts[0] -replace 'MiB|GiB', ''), 2)
            }
            
            # Get PostgreSQL container stats
            $postgresStats = docker stats postgres --no-stream --format "{{.CPUPerc}},{{.MemUsage}}" 2>$null
            if ($postgresStats) {
                $parts = $postgresStats -split ','
                $postgresCPU = [math]::Round([double]($parts[0] -replace '%', ''), 2)
                $memParts = $parts[1] -split '/'
                $postgresMemoryMB = [math]::Round([double]($memParts[0] -replace 'MiB|GiB', ''), 2)
            }
        } catch {
            Write-Host "⚠️  Warning: Could not get Docker stats" -ForegroundColor Yellow
        }
        
        # Database connections
        $dbConnections = 0
        try {
            $dbQuery = "SELECT count(*) FROM pg_stat_activity WHERE datname = 'sms_gateway';"
            $dbResult = docker exec postgres psql -U postgres -d sms_gateway -t -c $dbQuery 2>$null
            if ($dbResult) {
                $dbConnections = [int]$dbResult.Trim()
            }
        } catch {
            # Silently continue if DB query fails
        }
        
        # Redis metrics
        $redisKeys = 0
        $redisMemoryUsedMB = 0
        try {
            $redisInfo = docker exec redis redis-cli INFO 2>$null
            if ($redisInfo) {
                # Extract keys count
                $keysLine = $redisInfo | Select-String "db0:keys=(\d+)" | ForEach-Object { $_.Matches.Groups[1].Value }
                if ($keysLine) {
                    $redisKeys = [int]$keysLine
                }
                
                # Extract memory usage
                $memLine = $redisInfo | Select-String "used_memory:(\d+)" | ForEach-Object { $_.Matches.Groups[1].Value }
                if ($memLine) {
                    $redisMemoryUsedMB = [math]::Round([int]$memLine / 1MB, 2)
                }
            }
        } catch {
            # Silently continue if Redis query fails
        }
        
        # Write to CSV
        $row = "$timestamp,$cpu,$usedMemoryMB,$memoryPercent,$backendCPU,$backendMemoryMB,$redisCPU,$redisMemoryMB,$postgresCPU,$postgresMemoryMB,$dbConnections,$redisKeys,$redisMemoryUsedMB"
        $row | Out-File -FilePath $OutputFile -Append -Encoding UTF8
        
        # Display current metrics
        Write-Host "[$timestamp] Iteration $iteration" -ForegroundColor Cyan
        Write-Host "  System:    CPU: $($cpu.ToString('0.00'))% | Memory: $usedMemoryMB MB ($memoryPercent%)" -ForegroundColor White
        Write-Host "  Backend:   CPU: $backendCPU% | Memory: $backendMemoryMB MB" -ForegroundColor Green
        Write-Host "  Redis:     CPU: $redisCPU% | Memory: $redisMemoryMB MB | Keys: $redisKeys | Used: $redisMemoryUsedMB MB" -ForegroundColor Yellow
        Write-Host "  Postgres:  CPU: $postgresCPU% | Memory: $postgresMemoryMB MB | Connections: $dbConnections" -ForegroundColor Magenta
        Write-Host ""
        
        # Sleep for the specified interval
        Start-Sleep -Seconds $IntervalSeconds
    }
} catch {
    Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
} finally {
    Write-Host "✅ Monitoring completed. Results saved to: $OutputFile" -ForegroundColor Green
    
    # Generate summary
    Write-Host ""
    Write-Host "📊 Generating summary..." -ForegroundColor Cyan
    
    $data = Import-Csv $OutputFile
    
    if ($data.Count -gt 0) {
        $avgCPU = ($data | Measure-Object -Property CPU_Percent -Average).Average
        $maxCPU = ($data | Measure-Object -Property CPU_Percent -Maximum).Maximum
        $avgMemory = ($data | Measure-Object -Property Memory_MB -Average).Average
        $maxMemory = ($data | Measure-Object -Property Memory_MB -Maximum).Maximum
        $avgBackendCPU = ($data | Measure-Object -Property Backend_CPU -Average).Average
        $maxBackendCPU = ($data | Measure-Object -Property Backend_CPU -Maximum).Maximum
        $avgBackendMemory = ($data | Measure-Object -Property Backend_Memory_MB -Average).Average
        $maxBackendMemory = ($data | Measure-Object -Property Backend_Memory_MB -Maximum).Maximum
        $avgDBConnections = ($data | Measure-Object -Property DB_Connections -Average).Average
        $maxDBConnections = ($data | Measure-Object -Property DB_Connections -Maximum).Maximum
        $avgRedisKeys = ($data | Measure-Object -Property Redis_Keys -Average).Average
        $maxRedisKeys = ($data | Measure-Object -Property Redis_Keys -Maximum).Maximum
        
        Write-Host ""
        Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
        Write-Host "RESOURCE MONITORING SUMMARY" -ForegroundColor Cyan
        Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
        Write-Host ""
        Write-Host "System CPU:" -ForegroundColor White
        Write-Host "  Average: $($avgCPU.ToString('0.00'))%" -ForegroundColor Gray
        Write-Host "  Maximum: $($maxCPU.ToString('0.00'))%" -ForegroundColor Gray
        Write-Host ""
        Write-Host "System Memory:" -ForegroundColor White
        Write-Host "  Average: $($avgMemory.ToString('0.00')) MB" -ForegroundColor Gray
        Write-Host "  Maximum: $($maxMemory.ToString('0.00')) MB" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Backend Service:" -ForegroundColor Green
        Write-Host "  CPU Average: $($avgBackendCPU.ToString('0.00'))%" -ForegroundColor Gray
        Write-Host "  CPU Maximum: $($maxBackendCPU.ToString('0.00'))%" -ForegroundColor Gray
        Write-Host "  Memory Average: $($avgBackendMemory.ToString('0.00')) MB" -ForegroundColor Gray
        Write-Host "  Memory Maximum: $($maxBackendMemory.ToString('0.00')) MB" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Database:" -ForegroundColor Magenta
        Write-Host "  Connections Average: $($avgDBConnections.ToString('0.00'))" -ForegroundColor Gray
        Write-Host "  Connections Maximum: $maxDBConnections" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Redis:" -ForegroundColor Yellow
        Write-Host "  Keys Average: $($avgRedisKeys.ToString('0.00'))" -ForegroundColor Gray
        Write-Host "  Keys Maximum: $maxRedisKeys" -ForegroundColor Gray
        Write-Host ""
        
        # Save summary to JSON
        $summary = @{
            test_date = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            duration_minutes = $DurationMinutes
            samples_collected = $data.Count
            system = @{
                cpu_avg = [math]::Round($avgCPU, 2)
                cpu_max = [math]::Round($maxCPU, 2)
                memory_avg_mb = [math]::Round($avgMemory, 2)
                memory_max_mb = [math]::Round($maxMemory, 2)
            }
            backend = @{
                cpu_avg = [math]::Round($avgBackendCPU, 2)
                cpu_max = [math]::Round($maxBackendCPU, 2)
                memory_avg_mb = [math]::Round($avgBackendMemory, 2)
                memory_max_mb = [math]::Round($maxBackendMemory, 2)
            }
            database = @{
                connections_avg = [math]::Round($avgDBConnections, 2)
                connections_max = $maxDBConnections
            }
            redis = @{
                keys_avg = [math]::Round($avgRedisKeys, 2)
                keys_max = $maxRedisKeys
            }
        }
        
        $summaryFile = "results/resource-monitoring-summary.json"
        $summary | ConvertTo-Json -Depth 10 | Out-File -FilePath $summaryFile -Encoding UTF8
        Write-Host "📄 Summary saved to: $summaryFile" -ForegroundColor Green
    }
}
