# ============================================================================
# Periodic Restore Test Script (PowerShell)
# ============================================================================
# Purpose: Periodically test backup restore procedures
# Usage: .\test-restore-periodic.ps1
# Schedule: Run monthly via Task Scheduler
# ============================================================================

param(
    [Parameter(Mandatory=$false)]
    [string]$BackupDir = "backups/tariff-data",
    
    [Parameter(Mandatory=$false)]
    [string]$TestDatabase = "sms_gateway_restore_test",
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresHost = "localhost",
    
    [Parameter(Mandatory=$false)]
    [int]$PostgresPort = 5432,
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresUser = "postgres",
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresPassword = $env:POSTGRES_PASSWORD,
    
    [Parameter(Mandatory=$false)]
    [string]$DecryptionKey = $env:BACKUP_ENCRYPTION_KEY,
    
    [Parameter(Mandatory=$false)]
    [string]$SlackWebhook = $env:SLACK_WEBHOOK_URL
)

# Colors
$ErrorColor = "Red"
$SuccessColor = "Green"
$WarningColor = "Yellow"
$InfoColor = "Cyan"

# Logging
function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "ERROR" { $ErrorColor }
        "SUCCESS" { $SuccessColor }
        "WARNING" { $WarningColor }
        default { $InfoColor }
    }
    
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
    
    $logFile = "logs/restore-test-$(Get-Date -Format 'yyyyMMdd').log"
    $logDir = Split-Path $logFile -Parent
    if (-not (Test-Path $logDir)) {
        New-Item -ItemType Directory -Path $logDir -Force | Out-Null
    }
    Add-Content -Path $logFile -Value "[$timestamp] [$Level] $Message"
}

# Send notification
function Send-Notification {
    param([string]$Message, [string]$Status = "info")
    
    if ([string]::IsNullOrEmpty($SlackWebhook)) {
        return
    }
    
    $color = switch ($Status) {
        "success" { "good" }
        "error" { "danger" }
        "warning" { "warning" }
        default { "#439FE0" }
    }
    
    $payload = @{
        attachments = @(
            @{
                color = $color
                title = "Restore Test Report"
                text = $Message
                footer = "Tariff System Backup"
                ts = [int][double]::Parse((Get-Date -UFormat %s))
            }
        )
    } | ConvertTo-Json -Depth 3
    
    try {
        Invoke-RestMethod -Uri $SlackWebhook -Method Post -Body $payload -ContentType "application/json" | Out-Null
    } catch {
        Write-Log "Failed to send notification: $_" "WARNING"
    }
}

# Get latest backup
function Get-LatestBackup {
    Write-Log "Finding latest backup..." "INFO"
    
    if (-not (Test-Path $BackupDir)) {
        Write-Log "Backup directory not found: $BackupDir" "ERROR"
        return $null
    }
    
    $backups = Get-ChildItem -Path $BackupDir -Filter "*.sql.enc" | Sort-Object LastWriteTime -Descending
    
    if ($backups.Count -eq 0) {
        Write-Log "No backups found in $BackupDir" "ERROR"
        return $null
    }
    
    $latestBackup = $backups[0]
    Write-Log "Latest backup: $($latestBackup.Name) ($(Get-Date $latestBackup.LastWriteTime -Format 'yyyy-MM-dd HH:mm:ss'))" "INFO"
    
    return $latestBackup.FullName
}

# Create test database
function New-TestDatabase {
    Write-Log "Creating test database..." "INFO"
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        # Drop if exists
        $query = "DROP DATABASE IF EXISTS $TestDatabase;"
        psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d postgres -c $query 2>&1 | Out-Null
        
        # Create new
        $query = "CREATE DATABASE $TestDatabase;"
        $result = psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d postgres -c $query 2>&1
        
        if ($LASTEXITCODE -ne 0) {
            Write-Log "Failed to create test database: $result" "ERROR"
            return $false
        }
        
        Write-Log "Test database created successfully" "SUCCESS"
        return $true
    } catch {
        Write-Log "Error creating test database: $_" "ERROR"
        return $false
    } finally {
        Remove-Item Env:\PGPASSWORD
    }
}

# Test restore
function Test-Restore {
    param([string]$BackupFile)
    
    Write-Log "Testing restore from backup..." "INFO"
    
    $startTime = Get-Date
    
    # Run restore script
    $restoreScript = Join-Path $PSScriptRoot "restore-tariff-data.ps1"
    
    if (-not (Test-Path $restoreScript)) {
        Write-Log "Restore script not found: $restoreScript" "ERROR"
        return $false
    }
    
    try {
        & $restoreScript `
            -BackupFile $BackupFile `
            -DecryptionKey $DecryptionKey `
            -PostgresHost $PostgresHost `
            -PostgresPort $PostgresPort `
            -PostgresDB $TestDatabase `
            -PostgresUser $PostgresUser `
            -PostgresPassword $PostgresPassword
        
        if ($LASTEXITCODE -ne 0) {
            Write-Log "Restore test failed" "ERROR"
            return $false
        }
        
        $duration = (Get-Date) - $startTime
        Write-Log "Restore completed in $($duration.TotalSeconds) seconds" "SUCCESS"
        
        return $true
    } catch {
        Write-Log "Error during restore test: $_" "ERROR"
        return $false
    }
}

# Verify restored data
function Test-RestoredData {
    Write-Log "Verifying restored data..." "INFO"
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        # Check table counts
        $tables = @(
            "tariff_services",
            "tariffs",
            "packages",
            "subscriptions",
            "rollover_messages",
            "dedicated_gateways"
        )
        
        $results = @{}
        
        foreach ($table in $tables) {
            $query = "SELECT COUNT(*) FROM $table;"
            $count = psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $TestDatabase -t -c $query
            $results[$table] = [int]$count.Trim()
            Write-Log "$table: $($results[$table]) rows" "INFO"
        }
        
        # Check if services exist
        if ($results["tariff_services"] -eq 0) {
            Write-Log "No services found in restored database" "ERROR"
            return $false
        }
        
        # Check data integrity
        $query = @"
SELECT 
    (SELECT COUNT(*) FROM tariffs WHERE tenant_id IS NULL) as invalid_tariffs,
    (SELECT COUNT(*) FROM packages WHERE tenant_id IS NULL) as invalid_packages,
    (SELECT COUNT(*) FROM subscriptions WHERE tenant_id IS NULL) as invalid_subscriptions;
"@
        
        $result = psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $TestDatabase -t -c $query
        $parts = $result.Trim() -split '\|'
        
        $invalidTariffs = [int]$parts[0].Trim()
        $invalidPackages = [int]$parts[1].Trim()
        $invalidSubscriptions = [int]$parts[2].Trim()
        
        if ($invalidTariffs -gt 0 -or $invalidPackages -gt 0 -or $invalidSubscriptions -gt 0) {
            Write-Log "Data integrity issues found" "WARNING"
            Write-Log "Invalid tariffs: $invalidTariffs" "WARNING"
            Write-Log "Invalid packages: $invalidPackages" "WARNING"
            Write-Log "Invalid subscriptions: $invalidSubscriptions" "WARNING"
        }
        
        Write-Log "Data verification completed" "SUCCESS"
        return $true
    } catch {
        Write-Log "Error during data verification: $_" "ERROR"
        return $false
    } finally {
        Remove-Item Env:\PGPASSWORD
    }
}

# Cleanup test database
function Remove-TestDatabase {
    Write-Log "Cleaning up test database..." "INFO"
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        $query = "DROP DATABASE IF EXISTS $TestDatabase;"
        psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d postgres -c $query 2>&1 | Out-Null
        
        Write-Log "Test database cleaned up" "SUCCESS"
    } catch {
        Write-Log "Error cleaning up test database: $_" "WARNING"
    } finally {
        Remove-Item Env:\PGPASSWORD
    }
}

# Generate report
function New-TestReport {
    param(
        [bool]$Success,
        [string]$BackupFile,
        [hashtable]$Metrics
    )
    
    $report = @"
=== Restore Test Report ===
Date: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
Backup File: $BackupFile
Status: $(if ($Success) { "SUCCESS" } else { "FAILED" })

Metrics:
- Restore Duration: $($Metrics.RestoreDuration) seconds
- Total Duration: $($Metrics.TotalDuration) seconds

Data Counts:
- Services: $($Metrics.ServiceCount)
- Tariffs: $($Metrics.TariffCount)
- Packages: $($Metrics.PackageCount)
- Subscriptions: $($Metrics.SubscriptionCount)

Next Test: $(Get-Date).AddMonths(1) -Format 'yyyy-MM-dd')
"@
    
    $reportFile = "logs/restore-test-report-$(Get-Date -Format 'yyyyMMdd').txt"
    Set-Content -Path $reportFile -Value $report
    
    Write-Log "Report saved to: $reportFile" "INFO"
    
    return $report
}

# Main execution
function Main {
    Write-Log "=== Periodic Restore Test ===" "INFO"
    
    $overallSuccess = $true
    $metrics = @{
        RestoreDuration = 0
        TotalDuration = 0
        ServiceCount = 0
        TariffCount = 0
        PackageCount = 0
        SubscriptionCount = 0
    }
    
    $startTime = Get-Date
    
    try {
        # Get latest backup
        $backupFile = Get-LatestBackup
        if ($null -eq $backupFile) {
            $overallSuccess = $false
            Send-Notification -Message "No backup files found for restore test" -Status "error"
            exit 1
        }
        
        # Create test database
        if (-not (New-TestDatabase)) {
            $overallSuccess = $false
            Send-Notification -Message "Failed to create test database" -Status "error"
            exit 1
        }
        
        # Test restore
        $restoreStart = Get-Date
        if (-not (Test-Restore -BackupFile $backupFile)) {
            $overallSuccess = $false
            Send-Notification -Message "Restore test failed" -Status "error"
        } else {
            $metrics.RestoreDuration = ((Get-Date) - $restoreStart).TotalSeconds
            
            # Verify data
            if (-not (Test-RestoredData)) {
                $overallSuccess = $false
                Send-Notification -Message "Data verification failed" -Status "warning"
            }
        }
        
    } finally {
        # Cleanup
        Remove-TestDatabase
        
        $metrics.TotalDuration = ((Get-Date) - $startTime).TotalSeconds
        
        # Generate report
        $report = New-TestReport -Success $overallSuccess -BackupFile $backupFile -Metrics $metrics
        
        # Send notification
        if ($overallSuccess) {
            Send-Notification -Message "Restore test completed successfully`n`n$report" -Status "success"
            Write-Log "=== Restore test completed successfully ===" "SUCCESS"
        } else {
            Send-Notification -Message "Restore test failed`n`n$report" -Status "error"
            Write-Log "=== Restore test failed ===" "ERROR"
            exit 1
        }
    }
}

# Run main
Main
