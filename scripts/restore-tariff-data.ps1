# ============================================================================
# Tariff System Data Restore Script (PowerShell)
# ============================================================================
# Purpose: Restore tariff system data from encrypted backups
# Usage: .\restore-tariff-data.ps1 -BackupFile <path> [-DecryptionKey <key>]
# ============================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$BackupFile,
    
    [Parameter(Mandatory=$false)]
    [string]$DecryptionKey = $env:BACKUP_ENCRYPTION_KEY,
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresHost = "localhost",
    
    [Parameter(Mandatory=$false)]
    [int]$PostgresPort = 5432,
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresDB = "sms_gateway",
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresUser = "postgres",
    
    [Parameter(Mandatory=$false)]
    [string]$PostgresPassword = $env:POSTGRES_PASSWORD,
    
    [Parameter(Mandatory=$false)]
    [switch]$SkipValidation,
    
    [Parameter(Mandatory=$false)]
    [switch]$DryRun
)

# Colors for output
$ErrorColor = "Red"
$SuccessColor = "Green"
$WarningColor = "Yellow"
$InfoColor = "Cyan"

# Logging function
function Write-Log {
    param(
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "ERROR" { $ErrorColor }
        "SUCCESS" { $SuccessColor }
        "WARNING" { $WarningColor }
        default { $InfoColor }
    }
    
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
    
    # Also log to file
    $logFile = "logs/restore-$(Get-Date -Format 'yyyyMMdd').log"
    $logDir = Split-Path $logFile -Parent
    if (-not (Test-Path $logDir)) {
        New-Item -ItemType Directory -Path $logDir -Force | Out-Null
    }
    Add-Content -Path $logFile -Value "[$timestamp] [$Level] $Message"
}

# Check prerequisites
function Test-Prerequisites {
    Write-Log "Checking prerequisites..." "INFO"
    
    # Check if backup file exists
    if (-not (Test-Path $BackupFile)) {
        Write-Log "Backup file not found: $BackupFile" "ERROR"
        return $false
    }
    
    # Check if decryption key is provided
    if ([string]::IsNullOrEmpty($DecryptionKey)) {
        Write-Log "Decryption key not provided. Set BACKUP_ENCRYPTION_KEY environment variable or use -DecryptionKey parameter" "ERROR"
        return $false
    }
    
    # Check if PostgreSQL password is provided
    if ([string]::IsNullOrEmpty($PostgresPassword)) {
        Write-Log "PostgreSQL password not provided. Set POSTGRES_PASSWORD environment variable or use -PostgresPassword parameter" "ERROR"
        return $false
    }
    
    # Check if psql is available
    try {
        $null = Get-Command psql -ErrorAction Stop
    } catch {
        Write-Log "psql command not found. Please install PostgreSQL client tools" "ERROR"
        return $false
    }
    
    # Check if openssl is available
    try {
        $null = Get-Command openssl -ErrorAction Stop
    } catch {
        Write-Log "openssl command not found. Please install OpenSSL" "ERROR"
        return $false
    }
    
    Write-Log "All prerequisites met" "SUCCESS"
    return $true
}

# Decrypt backup file
function Invoke-DecryptBackup {
    param([string]$EncryptedFile, [string]$Key)
    
    Write-Log "Decrypting backup file..." "INFO"
    
    $decryptedFile = [System.IO.Path]::GetTempFileName()
    
    try {
        $keyFile = [System.IO.Path]::GetTempFileName()
        Set-Content -Path $keyFile -Value $Key -NoNewline
        
        $process = Start-Process -FilePath "openssl" `
            -ArgumentList "enc", "-d", "-aes-256-cbc", "-pbkdf2", `
                         "-in", $EncryptedFile, `
                         "-out", $decryptedFile, `
                         "-pass", "file:$keyFile" `
            -NoNewWindow -Wait -PassThru
        
        Remove-Item $keyFile -Force
        
        if ($process.ExitCode -ne 0) {
            Write-Log "Failed to decrypt backup file" "ERROR"
            return $null
        }
        
        Write-Log "Backup file decrypted successfully" "SUCCESS"
        return $decryptedFile
    } catch {
        Write-Log "Error during decryption: $_" "ERROR"
        return $null
    }
}

# Validate backup file
function Test-BackupFile {
    param([string]$BackupFile)
    
    Write-Log "Validating backup file..." "INFO"
    
    # Check if file is a valid SQL dump
    $firstLine = Get-Content $BackupFile -First 1
    if ($firstLine -notmatch "^--.*PostgreSQL") {
        Write-Log "Invalid backup file format" "ERROR"
        return $false
    }
    
    # Check if file contains tariff system tables
    $content = Get-Content $BackupFile -Raw
    $requiredTables = @(
        "tariff_services",
        "tariffs",
        "packages",
        "subscriptions",
        "rollover_messages",
        "dedicated_gateways"
    )
    
    foreach ($table in $requiredTables) {
        if ($content -notmatch $table) {
            Write-Log "Backup file missing table: $table" "WARNING"
        }
    }
    
    Write-Log "Backup file validation passed" "SUCCESS"
    return $true
}

# Create database backup before restore
function Backup-CurrentDatabase {
    Write-Log "Creating backup of current database..." "INFO"
    
    $backupDir = "backups/pre-restore"
    if (-not (Test-Path $backupDir)) {
        New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    }
    
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupFile = "$backupDir/pre-restore-$timestamp.sql"
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        $process = Start-Process -FilePath "pg_dump" `
            -ArgumentList "-h", $PostgresHost, `
                         "-p", $PostgresPort, `
                         "-U", $PostgresUser, `
                         "-d", $PostgresDB, `
                         "-f", $backupFile, `
                         "--no-owner", "--no-acl" `
            -NoNewWindow -Wait -PassThru
        
        if ($process.ExitCode -ne 0) {
            Write-Log "Failed to create pre-restore backup" "ERROR"
            return $false
        }
        
        Write-Log "Pre-restore backup created: $backupFile" "SUCCESS"
        return $true
    } catch {
        Write-Log "Error creating pre-restore backup: $_" "ERROR"
        return $false
    } finally {
        Remove-Item Env:\PGPASSWORD
    }
}

# Restore database from backup
function Restore-Database {
    param([string]$BackupFile)
    
    Write-Log "Restoring database from backup..." "INFO"
    
    if ($DryRun) {
        Write-Log "DRY RUN: Would restore from $BackupFile" "WARNING"
        return $true
    }
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        # Start transaction
        Write-Log "Starting restore transaction..." "INFO"
        
        $process = Start-Process -FilePath "psql" `
            -ArgumentList "-h", $PostgresHost, `
                         "-p", $PostgresPort, `
                         "-U", $PostgresUser, `
                         "-d", $PostgresDB, `
                         "-f", $BackupFile `
            -NoNewWindow -Wait -PassThru -RedirectStandardError "restore-errors.log"
        
        if ($process.ExitCode -ne 0) {
            Write-Log "Failed to restore database. Check restore-errors.log for details" "ERROR"
            return $false
        }
        
        Write-Log "Database restored successfully" "SUCCESS"
        return $true
    } catch {
        Write-Log "Error during restore: $_" "ERROR"
        return $false
    } finally {
        Remove-Item Env:\PGPASSWORD
    }
}

# Verify restore
function Test-RestoreIntegrity {
    Write-Log "Verifying restore integrity..." "INFO"
    
    $env:PGPASSWORD = $PostgresPassword
    
    try {
        # Check if tables exist
        $query = @"
SELECT COUNT(*) as table_count 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN (
    'tariff_services', 'tariffs', 'packages', 
    'subscriptions', 'rollover_messages', 'dedicated_gateways'
);
"@
        
        $result = psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $PostgresDB -t -c $query
        $tableCount = [int]$result.Trim()
        
        if ($tableCount -lt 6) {
            Write-Log "Some tables are missing after restore" "ERROR"
            return $false
        }
        
        # Check if data exists
        $query = "SELECT COUNT(*) FROM tariff_services;"
        $result = psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $PostgresDB -t -c $query
        $serviceCount = [int]$result.Trim()
        
        if ($serviceCount -eq 0) {
            Write-Log "No data found in tariff_services table" "WARNING"
        }
        
        Write-Log "Restore integrity verification passed" "SUCCESS"
        return $true
    } catch {
        Write-Log "Error during integrity verification: $_" "ERROR"
        return $false
    } finally {
        Remove-Item Env:\PGPASSWORD
    }
}

# Main execution
function Main {
    Write-Log "=== Tariff System Data Restore ===" "INFO"
    Write-Log "Backup file: $BackupFile" "INFO"
    Write-Log "Target database: $PostgresDB@$PostgresHost:$PostgresPort" "INFO"
    
    if ($DryRun) {
        Write-Log "DRY RUN MODE - No changes will be made" "WARNING"
    }
    
    # Check prerequisites
    if (-not (Test-Prerequisites)) {
        Write-Log "Prerequisites check failed. Aborting restore" "ERROR"
        exit 1
    }
    
    # Decrypt backup if encrypted
    $decryptedFile = $BackupFile
    if ($BackupFile -match "\.enc$") {
        $decryptedFile = Invoke-DecryptBackup -EncryptedFile $BackupFile -Key $DecryptionKey
        if ($null -eq $decryptedFile) {
            Write-Log "Failed to decrypt backup file. Aborting restore" "ERROR"
            exit 1
        }
    }
    
    # Validate backup file
    if (-not $SkipValidation) {
        if (-not (Test-BackupFile -BackupFile $decryptedFile)) {
            Write-Log "Backup file validation failed. Aborting restore" "ERROR"
            if ($decryptedFile -ne $BackupFile) {
                Remove-Item $decryptedFile -Force
            }
            exit 1
        }
    }
    
    # Create pre-restore backup
    if (-not $DryRun) {
        if (-not (Backup-CurrentDatabase)) {
            Write-Log "Failed to create pre-restore backup. Aborting restore" "ERROR"
            if ($decryptedFile -ne $BackupFile) {
                Remove-Item $decryptedFile -Force
            }
            exit 1
        }
    }
    
    # Restore database
    if (-not (Restore-Database -BackupFile $decryptedFile)) {
        Write-Log "Database restore failed" "ERROR"
        if ($decryptedFile -ne $BackupFile) {
            Remove-Item $decryptedFile -Force
        }
        exit 1
    }
    
    # Verify restore
    if (-not $SkipValidation -and -not $DryRun) {
        if (-not (Test-RestoreIntegrity)) {
            Write-Log "Restore integrity verification failed" "ERROR"
            if ($decryptedFile -ne $BackupFile) {
                Remove-Item $decryptedFile -Force
            }
            exit 1
        }
    }
    
    # Cleanup
    if ($decryptedFile -ne $BackupFile) {
        Remove-Item $decryptedFile -Force
    }
    
    Write-Log "=== Restore completed successfully ===" "SUCCESS"
    Write-Log "Please verify the restored data before resuming operations" "WARNING"
}

# Run main function
Main
