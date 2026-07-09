################################################################################
# Backup Retention Policy Script (PowerShell)
# 
# This script implements the retention policy for tariff system backups:
# - Daily backups: Keep for 30 days
# - Weekly backups: Keep for 3 months (90 days)
# - Monthly backups: Keep for 1 year (365 days)
#
# Requirements: 25.2, 25.3, 25.4
################################################################################

param(
    [string]$BackupBaseDir = "./backups",
    [string]$LogDir = "./logs",
    [switch]$DryRun = $false,
    [switch]$Verbose = $false
)

# Configuration
$DailyDir = Join-Path $BackupBaseDir "daily"
$WeeklyDir = Join-Path $BackupBaseDir "weekly"
$MonthlyDir = Join-Path $BackupBaseDir "monthly"
$LogFile = Join-Path $LogDir "retention-policy.log"

# Retention periods (in days)
$DailyRetention = 30
$WeeklyRetention = 90
$MonthlyRetention = 365

################################################################################
# Logging Functions
################################################################################

function Write-Log {
    param(
        [string]$Level,
        [string]$Message
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    
    # Ensure log directory exists
    $logDir = Split-Path $LogFile -Parent
    if (-not (Test-Path $logDir)) {
        New-Item -ItemType Directory -Path $logDir -Force | Out-Null
    }
    
    Add-Content -Path $LogFile -Value $logMessage
    
    # Console output with colors
    switch ($Level) {
        "INFO" { Write-Host "[INFO] $Message" -ForegroundColor Blue }
        "SUCCESS" { Write-Host "[SUCCESS] $Message" -ForegroundColor Green }
        "WARNING" { Write-Host "[WARNING] $Message" -ForegroundColor Yellow }
        "ERROR" { Write-Host "[ERROR] $Message" -ForegroundColor Red }
    }
}

function Write-LogInfo {
    param([string]$Message)
    Write-Log -Level "INFO" -Message $Message
}

function Write-LogSuccess {
    param([string]$Message)
    Write-Log -Level "SUCCESS" -Message $Message
}

function Write-LogWarning {
    param([string]$Message)
    Write-Log -Level "WARNING" -Message $Message
}

function Write-LogError {
    param([string]$Message)
    Write-Log -Level "ERROR" -Message $Message
}

################################################################################
# Retention Policy Functions
################################################################################

function Apply-RetentionPolicy {
    param(
        [string]$BackupDir,
        [int]$RetentionDays,
        [string]$BackupType
    )
    
    Write-LogInfo "Applying $BackupType retention policy: $RetentionDays days"
    
    if (-not (Test-Path $BackupDir)) {
        Write-LogWarning "Directory does not exist: $BackupDir"
        return @{
            DeletedCount = 0
            KeptCount = 0
            SizeDeleted = 0
        }
    }
    
    $deletedCount = 0
    $keptCount = 0
    $totalSizeDeleted = 0
    
    # Get all backup files
    $backupFiles = Get-ChildItem -Path $BackupDir -File -Recurse | 
        Where-Object { $_.Extension -in @('.gz', '.sql', '.tar') }
    
    $cutoffDate = (Get-Date).AddDays(-$RetentionDays)
    
    foreach ($file in $backupFiles) {
        $fileAge = (Get-Date) - $file.LastWriteTime
        $fileAgeDays = [math]::Floor($fileAge.TotalDays)
        
        if ($file.LastWriteTime -lt $cutoffDate) {
            $fileSizeMB = [math]::Round($file.Length / 1MB, 2)
            
            if ($Verbose) {
                Write-LogInfo "File: $($file.Name), Age: $fileAgeDays days, Size: ${fileSizeMB}MB"
            }
            
            if ($DryRun) {
                Write-LogInfo "[DRY RUN] Would delete: $($file.Name) (${fileAgeDays} days old, ${fileSizeMB}MB)"
            } else {
                Write-LogInfo "Deleting old backup (${fileAgeDays} days old): $($file.Name)"
                
                try {
                    Remove-Item -Path $file.FullName -Force
                    $deletedCount++
                    $totalSizeDeleted += $file.Length
                    Write-LogSuccess "Deleted: $($file.Name)"
                } catch {
                    Write-LogError "Failed to delete: $($file.Name) - $_"
                }
            }
        } else {
            $keptCount++
            if ($Verbose) {
                Write-LogInfo "Keeping: $($file.Name) (${fileAgeDays} days old)"
            }
        }
    }
    
    $sizeMB = [math]::Round($totalSizeDeleted / 1MB, 2)
    
    Write-LogSuccess "$BackupType retention complete:"
    Write-LogSuccess "  - Deleted: $deletedCount backups (${sizeMB}MB freed)"
    Write-LogSuccess "  - Kept: $keptCount backups"
    Write-Host ""
    
    return @{
        DeletedCount = $deletedCount
        KeptCount = $keptCount
        SizeDeleted = $totalSizeDeleted
    }
}

function Remove-EmptyDirectories {
    Write-LogInfo "Cleaning up empty directories..."
    
    $directories = @($DailyDir, $WeeklyDir, $MonthlyDir)
    
    foreach ($dir in $directories) {
        if (Test-Path $dir) {
            Get-ChildItem -Path $dir -Directory -Recurse | 
                Where-Object { (Get-ChildItem $_.FullName).Count -eq 0 } |
                ForEach-Object {
                    if ($DryRun) {
                        Write-LogInfo "[DRY RUN] Would remove empty directory: $($_.FullName)"
                    } else {
                        Remove-Item -Path $_.FullName -Force
                        Write-LogInfo "Removed empty directory: $($_.FullName)"
                    }
                }
        }
    }
    
    Write-LogSuccess "Empty directories cleaned up"
}

function Get-BackupStatistics {
    param([string]$BackupDir)
    
    if (-not (Test-Path $BackupDir)) {
        return @{
            Count = 0
            TotalSize = 0
            Files = @()
        }
    }
    
    $files = Get-ChildItem -Path $BackupDir -File -Recurse | 
        Where-Object { $_.Extension -in @('.gz', '.sql', '.tar') }
    
    $totalSize = ($files | Measure-Object -Property Length -Sum).Sum
    
    return @{
        Count = $files.Count
        TotalSize = $totalSize
        Files = $files
    }
}

function New-RetentionReport {
    Write-LogInfo "Generating retention report..."
    
    $reportFile = Join-Path $BackupBaseDir "retention-report-$(Get-Date -Format 'yyyyMMdd-HHmmss').txt"
    
    $report = @"
==========================================
Backup Retention Policy Report
Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
==========================================

Retention Policies:
  - Daily backups: $DailyRetention days
  - Weekly backups: $WeeklyRetention days
  - Monthly backups: $MonthlyRetention days

"@
    
    foreach ($backupType in @("daily", "weekly", "monthly")) {
        $dir = Join-Path $BackupBaseDir $backupType
        $stats = Get-BackupStatistics -BackupDir $dir
        
        $report += @"
----------------------------------------
$($backupType.ToUpper()) Backups:
----------------------------------------
  Location: $dir
  Total backups: $($stats.Count)
  Total size: $([math]::Round($stats.TotalSize / 1MB, 2))MB

"@
        
        if ($stats.Count -gt 0) {
            $report += "  Backup files:`n"
            foreach ($file in $stats.Files | Sort-Object LastWriteTime -Descending) {
                $sizeMB = [math]::Round($file.Length / 1MB, 2)
                $age = [math]::Floor(((Get-Date) - $file.LastWriteTime).TotalDays)
                $report += "    - $($file.Name) (${sizeMB}MB, ${age} days old, $($file.LastWriteTime.ToString('yyyy-MM-dd')))`n"
            }
        } else {
            $report += "  No backups found`n"
        }
        
        $report += "`n"
    }
    
    $report += @"
==========================================
End of Report
==========================================
"@
    
    $report | Out-File -FilePath $reportFile -Encoding UTF8
    
    Write-LogSuccess "Report generated: $reportFile"
    Write-Host ""
    Write-Host $report
    
    return $reportFile
}

function Test-BackupIntegrity {
    param([string]$BackupFile)
    
    # Check if file exists and is readable
    if (-not (Test-Path $BackupFile)) {
        Write-LogError "Backup file not found: $BackupFile"
        return $false
    }
    
    # Check if file is not empty
    $fileInfo = Get-Item $BackupFile
    if ($fileInfo.Length -eq 0) {
        Write-LogError "Backup file is empty: $BackupFile"
        return $false
    }
    
    # For .gz files, verify integrity
    if ($BackupFile -match '\.gz$') {
        try {
            # Test gzip integrity using 7-Zip if available
            if (Get-Command 7z -ErrorAction SilentlyContinue) {
                $result = & 7z t $BackupFile 2>&1
                if ($LASTEXITCODE -ne 0) {
                    Write-LogError "Backup file is corrupted: $BackupFile"
                    return $false
                }
            }
        } catch {
            Write-LogWarning "Could not verify gzip integrity: $_"
        }
    }
    
    return $true
}

################################################################################
# Main Execution
################################################################################

function Main {
    Write-LogInfo "=========================================="
    Write-LogInfo "Starting Backup Retention Policy"
    if ($DryRun) {
        Write-LogInfo "MODE: DRY RUN (no files will be deleted)"
    }
    Write-LogInfo "=========================================="
    Write-LogInfo "Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    Write-Host ""
    
    # Verify backup directories exist
    foreach ($dir in @($DailyDir, $WeeklyDir, $MonthlyDir)) {
        if (-not (Test-Path $dir)) {
            Write-LogWarning "Creating missing directory: $dir"
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }
    
    # Apply retention policies
    $dailyResults = Apply-RetentionPolicy -BackupDir $DailyDir -RetentionDays $DailyRetention -BackupType "Daily"
    $weeklyResults = Apply-RetentionPolicy -BackupDir $WeeklyDir -RetentionDays $WeeklyRetention -BackupType "Weekly"
    $monthlyResults = Apply-RetentionPolicy -BackupDir $MonthlyDir -RetentionDays $MonthlyRetention -BackupType "Monthly"
    
    # Clean up empty directories
    Remove-EmptyDirectories
    
    # Generate report
    $reportFile = New-RetentionReport
    
    # Summary
    $totalDeleted = $dailyResults.DeletedCount + $weeklyResults.DeletedCount + $monthlyResults.DeletedCount
    $totalKept = $dailyResults.KeptCount + $weeklyResults.KeptCount + $monthlyResults.KeptCount
    $totalSizeFreed = [math]::Round(($dailyResults.SizeDeleted + $weeklyResults.SizeDeleted + $monthlyResults.SizeDeleted) / 1MB, 2)
    
    Write-LogInfo "=========================================="
    Write-LogSuccess "Backup Retention Policy Completed"
    Write-LogInfo "=========================================="
    Write-LogSuccess "Total backups deleted: $totalDeleted"
    Write-LogSuccess "Total backups kept: $totalKept"
    Write-LogSuccess "Total space freed: ${totalSizeFreed}MB"
    Write-LogSuccess "Report saved to: $reportFile"
    Write-LogInfo "=========================================="
}

# Run main function
try {
    Main
} catch {
    Write-LogError "An error occurred: $_"
    Write-LogError $_.ScriptStackTrace
    exit 1
}
