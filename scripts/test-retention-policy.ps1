################################################################################
# Test Retention Policy Script
# 
# This script creates test backup files with various ages to verify
# the retention policy works correctly
################################################################################

param(
    [string]$TestBackupDir = "./test-backups",
    [switch]$CleanupAfter = $false
)

Write-Host "=========================================="  -ForegroundColor Cyan
Write-Host "Retention Policy Test Setup" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Create test directory structure
$dailyDir = Join-Path $TestBackupDir "daily"
$weeklyDir = Join-Path $TestBackupDir "weekly"
$monthlyDir = Join-Path $TestBackupDir "monthly"

foreach ($dir in @($dailyDir, $weeklyDir, $monthlyDir)) {
    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Write-Host "[✓] Created directory: $dir" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "Creating test backup files..." -ForegroundColor Yellow
Write-Host ""

# Function to create test backup file
function New-TestBackup {
    param(
        [string]$Directory,
        [int]$DaysOld,
        [string]$Type
    )
    
    $date = (Get-Date).AddDays(-$DaysOld)
    $filename = "tariff-backup-$($date.ToString('yyyyMMdd'))-030000.sql.gz"
    $filepath = Join-Path $Directory $filename
    
    # Create dummy gzip file
    $content = "-- Test backup created on $($date.ToString('yyyy-MM-dd'))`n"
    $content += "-- This is a test backup file for retention policy testing`n"
    $content += "-- Age: $DaysOld days`n"
    $content += "-- Type: $Type`n"
    
    # Write content and compress
    $tempFile = "$filepath.tmp"
    $content | Out-File -FilePath $tempFile -Encoding UTF8
    
    # Compress using .NET (simple compression)
    $bytes = [System.IO.File]::ReadAllBytes($tempFile)
    $stream = [System.IO.File]::Create($filepath)
    $gzipStream = New-Object System.IO.Compression.GZipStream($stream, [System.IO.Compression.CompressionMode]::Compress)
    $gzipStream.Write($bytes, 0, $bytes.Length)
    $gzipStream.Close()
    $stream.Close()
    
    Remove-Item $tempFile
    
    # Set file timestamp
    (Get-Item $filepath).LastWriteTime = $date
    
    $sizeMB = [math]::Round((Get-Item $filepath).Length / 1MB, 2)
    
    return @{
        Path = $filepath
        Age = $DaysOld
        Date = $date
        Size = $sizeMB
    }
}

# Create daily backups (should keep 30 days)
Write-Host "Daily Backups (30-day retention):" -ForegroundColor Cyan
$dailyBackups = @()

# Should be kept (within 30 days)
foreach ($age in @(1, 5, 10, 15, 20, 25, 29)) {
    $backup = New-TestBackup -Directory $dailyDir -DaysOld $age -Type "Daily"
    $dailyBackups += $backup
    Write-Host "  [KEEP] Created: $(Split-Path $backup.Path -Leaf) - $($age) days old" -ForegroundColor Green
}

# Should be deleted (older than 30 days)
foreach ($age in @(31, 35, 40, 45, 50, 60)) {
    $backup = New-TestBackup -Directory $dailyDir -DaysOld $age -Type "Daily"
    $dailyBackups += $backup
    Write-Host "  [DELETE] Created: $(Split-Path $backup.Path -Leaf) - $($age) days old" -ForegroundColor Red
}

Write-Host ""

# Create weekly backups (should keep 90 days)
Write-Host "Weekly Backups (90-day retention):" -ForegroundColor Cyan
$weeklyBackups = @()

# Should be kept (within 90 days)
foreach ($age in @(7, 14, 21, 28, 35, 60, 85)) {
    $backup = New-TestBackup -Directory $weeklyDir -DaysOld $age -Type "Weekly"
    $weeklyBackups += $backup
    Write-Host "  [KEEP] Created: $(Split-Path $backup.Path -Leaf) - $($age) days old" -ForegroundColor Green
}

# Should be deleted (older than 90 days)
foreach ($age in @(91, 100, 120, 150)) {
    $backup = New-TestBackup -Directory $weeklyDir -DaysOld $age -Type "Weekly"
    $weeklyBackups += $backup
    Write-Host "  [DELETE] Created: $(Split-Path $backup.Path -Leaf) - $($age) days old" -ForegroundColor Red
}

Write-Host ""

# Create monthly backups (should keep 365 days)
Write-Host "Monthly Backups (365-day retention):" -ForegroundColor Cyan
$monthlyBackups = @()

# Should be kept (within 365 days)
foreach ($age in @(30, 60, 90, 180, 270, 360)) {
    $backup = New-TestBackup -Directory $monthlyDir -DaysOld $age -Type "Monthly"
    $monthlyBackups += $backup
    Write-Host "  [KEEP] Created: $(Split-Path $backup.Path -Leaf) - $($age) days old" -ForegroundColor Green
}

# Should be deleted (older than 365 days)
foreach ($age in @(366, 400, 500, 600)) {
    $backup = New-TestBackup -Directory $monthlyDir -DaysOld $age -Type "Monthly"
    $monthlyBackups += $backup
    Write-Host "  [DELETE] Created: $(Split-Path $backup.Path -Leaf) - $($age) days old" -ForegroundColor Red
}

Write-Host ""
Write-Host "=========================================="  -ForegroundColor Cyan
Write-Host "Test Setup Complete" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Summary
$totalBackups = $dailyBackups.Count + $weeklyBackups.Count + $monthlyBackups.Count
$shouldKeep = 7 + 7 + 6  # Daily + Weekly + Monthly that should be kept
$shouldDelete = 6 + 4 + 4  # Daily + Weekly + Monthly that should be deleted

Write-Host "Summary:" -ForegroundColor Yellow
Write-Host "  Total test backups created: $totalBackups" -ForegroundColor White
Write-Host "  Expected to keep: $shouldKeep backups" -ForegroundColor Green
Write-Host "  Expected to delete: $shouldDelete backups" -ForegroundColor Red
Write-Host ""

Write-Host "Test backup location: $TestBackupDir" -ForegroundColor Yellow
Write-Host ""

# Instructions
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "  1. Run the retention policy with test directory:" -ForegroundColor White
Write-Host "     .\scripts\backup-retention-policy.ps1 -BackupBaseDir '$TestBackupDir' -DryRun" -ForegroundColor Gray
Write-Host ""
Write-Host "  2. Verify the dry run output shows correct files to delete" -ForegroundColor White
Write-Host ""
Write-Host "  3. Run without -DryRun to actually delete:" -ForegroundColor White
Write-Host "     .\scripts\backup-retention-policy.ps1 -BackupBaseDir '$TestBackupDir'" -ForegroundColor Gray
Write-Host ""
Write-Host "  4. Verify results:" -ForegroundColor White
Write-Host "     - Daily: Should have 7 files remaining" -ForegroundColor Gray
Write-Host "     - Weekly: Should have 7 files remaining" -ForegroundColor Gray
Write-Host "     - Monthly: Should have 6 files remaining" -ForegroundColor Gray
Write-Host ""

if ($CleanupAfter) {
    Write-Host "Cleaning up test backups..." -ForegroundColor Yellow
    Remove-Item -Path $TestBackupDir -Recurse -Force
    Write-Host "[✓] Test backups cleaned up" -ForegroundColor Green
}

Write-Host "=========================================="  -ForegroundColor Cyan
