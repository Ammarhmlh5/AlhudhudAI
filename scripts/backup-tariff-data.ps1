# Tariff System Backup Script (PowerShell)
# Windows-compatible version of backup-tariff-data.sh

param(
    [string]$BackupDir = "C:\backups\tariff-system",
    [string]$EncryptionKeyFile = "C:\tariff-system\backup.key",
    [string]$DBHost = "localhost",
    [string]$DBPort = "5432",
    [string]$DBName = "sms_gateway",
    [string]$DBUser = "postgres",
    [string]$DBPassword = "",
    [string]$S3Bucket = "",
    [string]$AlertEmail = "admin@example.com"
)

$ErrorActionPreference = "Stop"

# Set environment variable for PostgreSQL password
$env:PGPASSWORD = $DBPassword

# Timestamp
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$Date = Get-Date -Format "yyyy-MM-dd"
$DayOfWeek = (Get-Date).DayOfWeek
$DayOfMonth = (Get-Date).Day

# Determine backup type
$BackupType = "daily"
if ($DayOfWeek -eq "Sunday") {
    $BackupType = "weekly"
}
if ($DayOfMonth -eq 1) {
    $BackupType = "monthly"
}

# Create backup directory structure
$Directories = @("daily", "weekly", "monthly", "temp")
foreach ($dir in $Directories) {
    $path = Join-Path $BackupDir $dir
    if (-not (Test-Path $path)) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }
}

# Log file
$LogFile = Join-Path $BackupDir "backup_$Timestamp.log"

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] $Message"
    Write-Host $logMessage
    Add-Content -Path $LogFile -Value $logMessage
}

function Write-LogError {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] ERROR: $Message"
    Write-Host $logMessage -ForegroundColor Red
    Add-Content -Path $LogFile -Value $logMessage
}

function Send-Alert {
    param(
        [string]$Subject,
        [string]$Message
    )
    
    Write-LogError $Message
    
    # Send email if configured
    if ($AlertEmail) {
        try {
            # Requires Send-MailMessage or similar
            # Send-MailMessage -To $AlertEmail -Subject $Subject -Body $Message
        } catch {
            Write-LogError "Failed to send alert email: $_"
        }
    }
}

function Backup-Table {
    param(
        [string]$TableName,
        [string]$OutputFile
    )
    
    Write-Log "Backing up table: $TableName"
    
    try {
        & pg_dump -h $DBHost -p $DBPort -U $DBUser -d $DBName `
            -t $TableName --no-owner --no-acl -F c -f $OutputFile
        
        if ($LASTEXITCODE -eq 0) {
            Write-Log "Successfully backed up $TableName"
            return $true
        } else {
            Write-LogError "Failed to backup $TableName"
            return $false
        }
    } catch {
        Write-LogError "Exception backing up $TableName : $_"
        return $false
    }
}

function Encrypt-File {
    param(
        [string]$InputFile,
        [string]$OutputFile
    )
    
    if (-not (Test-Path $EncryptionKeyFile)) {
        Write-LogError "Encryption key file not found: $EncryptionKeyFile"
        return $false
    }
    
    Write-Log "Encrypting: $(Split-Path $InputFile -Leaf)"
    
    try {
        & openssl enc -aes-256-cbc -salt -pbkdf2 -in $InputFile `
            -out $OutputFile -pass "file:$EncryptionKeyFile"
        
        if ($LASTEXITCODE -eq 0) {
            Write-Log "Successfully encrypted $(Split-Path $InputFile -Leaf)"
            Remove-Item $InputFile -Force
            return $true
        } else {
            Write-LogError "Failed to encrypt $(Split-Path $InputFile -Leaf)"
            return $false
        }
    } catch {
        Write-LogError "Exception encrypting file: $_"
        return $false
    }
}

function Test-Backup {
    param([string]$BackupFile)
    
    if (-not (Test-Path $BackupFile)) {
        Write-LogError "Backup file not found: $BackupFile"
        return $false
    }
    
    $fileSize = (Get-Item $BackupFile).Length
    
    if ($fileSize -lt 100) {
        Write-LogError "Backup file too small ($fileSize bytes): $BackupFile"
        return $false
    }
    
    Write-Log "Backup verified: $(Split-Path $BackupFile -Leaf) ($fileSize bytes)"
    return $true
}

function Upload-ToS3 {
    param(
        [string]$LocalFile,
        [string]$S3Path
    )
    
    if (-not $S3Bucket) {
        Write-Log "S3 upload skipped (no bucket configured)"
        return $true
    }
    
    Write-Log "Uploading to S3: $S3Path"
    
    try {
        & aws s3 cp $LocalFile "s3://$S3Bucket/$S3Path" `
            --storage-class STANDARD_IA --server-side-encryption AES256
        
        if ($LASTEXITCODE -eq 0) {
            Write-Log "Successfully uploaded to S3"
            return $true
        } else {
            Write-LogError "Failed to upload to S3"
            return $false
        }
    } catch {
        Write-LogError "Exception uploading to S3: $_"
        return $false
    }
}

# Main backup process
Write-Log "=========================================="
Write-Log "Starting Tariff System Backup"
Write-Log "Backup Type: $BackupType"
Write-Log "Timestamp: $Timestamp"
Write-Log "=========================================="

# Temporary directory
$TempDir = Join-Path $BackupDir "temp\$Timestamp"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

# Tables to backup
$Tables = @(
    "services",
    "tariffs",
    "packages",
    "subscriptions",
    "rollover_messages",
    "dedicated_gateways",
    "resource_quotas",
    "resource_usage",
    "gateway_users",
    "gateway_api_keys",
    "gateway_statistics",
    "accounting_transactions",
    "balance_history",
    "segmentation_rules",
    "currencies",
    "exchange_rates",
    "billing_invoices"
)

# Backup each table
$BackupSuccess = $true
foreach ($table in $Tables) {
    $backupFile = Join-Path $TempDir "$table.dump"
    $encryptedFile = Join-Path $TempDir "$table.dump.enc"
    
    if (Backup-Table -TableName $table -OutputFile $backupFile) {
        if (Encrypt-File -InputFile $backupFile -OutputFile $encryptedFile) {
            if (-not (Test-Backup -BackupFile $encryptedFile)) {
                $BackupSuccess = $false
            }
        } else {
            $BackupSuccess = $false
        }
    } else {
        $BackupSuccess = $false
    }
}

# Create archive
if ($BackupSuccess) {
    Write-Log "Creating backup archive"
    
    $ArchiveName = "tariff_backup_${BackupType}_${Timestamp}.zip"
    $ArchivePath = Join-Path $BackupDir "$BackupType\$ArchiveName"
    
    try {
        Compress-Archive -Path "$TempDir\*" -DestinationPath $ArchivePath -Force
        Write-Log "Successfully created archive: $ArchiveName"
        
        if (Test-Backup -BackupFile $ArchivePath) {
            # Upload to S3
            $S3Path = "backups/$BackupType/$Date/$ArchiveName"
            Upload-ToS3 -LocalFile $ArchivePath -S3Path $S3Path
            
            Write-Log "Backup completed successfully"
        } else {
            $BackupSuccess = $false
        }
    } catch {
        Write-LogError "Failed to create archive: $_"
        $BackupSuccess = $false
    }
}

# Cleanup temp directory
Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue

# Apply retention policy
Write-Log "Applying retention policy"

$RetentionDays = @{
    "daily" = 30
    "weekly" = 84
    "monthly" = 365
}

foreach ($type in $RetentionDays.Keys) {
    $path = Join-Path $BackupDir $type
    $cutoffDate = (Get-Date).AddDays(-$RetentionDays[$type])
    
    Get-ChildItem -Path $path -Filter "*.zip" | 
        Where-Object { $_.LastWriteTime -lt $cutoffDate } |
        Remove-Item -Force
    
    Write-Log "Cleaned up $type backups older than $($RetentionDays[$type]) days"
}

# Final status
if ($BackupSuccess) {
    Write-Log "=========================================="
    Write-Log "Backup completed successfully"
    Write-Log "=========================================="
    exit 0
} else {
    Send-Alert -Subject "Tariff System Backup Failed" `
        -Message "Backup failed on $env:COMPUTERNAME at $(Get-Date). Check log: $LogFile"
    Write-Log "=========================================="
    Write-Log "Backup completed with errors"
    Write-Log "=========================================="
    exit 1
}
