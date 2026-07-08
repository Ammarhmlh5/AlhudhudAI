# Encryption Key Management Script (PowerShell)
# Windows-compatible version of key-management.sh

param(
    [Parameter(Mandatory=$false)]
    [string]$Command,
    
    [Parameter(Mandatory=$false)]
    [string]$KeyFile,
    
    [Parameter(Mandatory=$false)]
    [string]$Destination,
    
    [Parameter(Mandatory=$false)]
    [string]$BackupFile,
    
    [Parameter(Mandatory=$false)]
    [string]$KeyDir = "C:\tariff-system\keys",
    
    [Parameter(Mandatory=$false)]
    [int]$RotationDays = 90,
    
    [Parameter(Mandatory=$false)]
    [int]$KeySize = 32
)

$ErrorActionPreference = "Stop"

# Configuration
$ActiveKeyFile = Join-Path $KeyDir "backup.key"
$KeyBackupDir = Join-Path $KeyDir "backups"
$LogFile = "C:\tariff-system\logs\key-management.log"

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
    
    # Create log directory if it doesn't exist
    $logDir = Split-Path $LogFile -Parent
    if (-not (Test-Path $logDir)) {
        New-Item -ItemType Directory -Path $logDir -Force | Out-Null
    }
    
    Add-Content -Path $LogFile -Value $logMessage
}

function Write-Info {
    param([string]$Message)
    Write-Log -Level "INFO" -Message $Message
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Log -Level "SUCCESS" -Message $Message
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Log -Level "WARNING" -Message $Message
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-ErrorMessage {
    param([string]$Message)
    Write-Log -Level "ERROR" -Message $Message
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

################################################################################
# Key Generation Functions
################################################################################

function New-EncryptionKey {
    param(
        [string]$Path,
        [string]$Version = "v1"
    )
    
    Write-Info "Generating new encryption key: $Version"
    
    # Create key directory if it doesn't exist
    $keyDir = Split-Path $Path -Parent
    if (-not (Test-Path $keyDir)) {
        New-Item -ItemType Directory -Path $keyDir -Force | Out-Null
    }
    
    try {
        # Generate random key using OpenSSL
        & openssl rand -base64 $KeySize | Set-Content -Path $Path -NoNewline
        
        if ($LASTEXITCODE -eq 0) {
            # Set secure permissions (remove inheritance, grant only to current user)
            $acl = Get-Acl $Path
            $acl.SetAccessRuleProtection($true, $false)
            $acl.Access | ForEach-Object { $acl.RemoveAccessRule($_) | Out-Null }
            
            $currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
            $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                $currentUser, "Read", "Allow"
            )
            $acl.AddAccessRule($accessRule)
            Set-Acl -Path $Path -AclObject $acl
            
            Write-Success "Key generated successfully: $Path"
            
            # Create metadata
            New-KeyMetadata -KeyFile $Path -Version $Version
            
            return $true
        } else {
            Write-ErrorMessage "Failed to generate key"
            return $false
        }
    } catch {
        Write-ErrorMessage "Exception generating key: $_"
        return $false
    }
}

function New-KeyMetadata {
    param(
        [string]$KeyFile,
        [string]$Version
    )
    
    $metadataFile = "$KeyFile.meta"
    $keyChecksum = (Get-FileHash -Path $KeyFile -Algorithm SHA256).Hash
    $nextRotation = (Get-Date).AddDays($RotationDays).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    
    $metadata = @{
        key_version = $Version
        key_file = Split-Path $KeyFile -Leaf
        key_size = $KeySize
        key_checksum = $keyChecksum
        created_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        created_by = $env:USERNAME
        hostname = $env:COMPUTERNAME
        rotation_days = $RotationDays
        next_rotation = $nextRotation
    }
    
    $metadata | ConvertTo-Json | Set-Content -Path $metadataFile
    
    # Set secure permissions on metadata
    $acl = Get-Acl $KeyFile
    Set-Acl -Path $metadataFile -AclObject $acl
    
    Write-Info "Key metadata created: $metadataFile"
}

function Initialize-KeyManagement {
    Write-Info "Initializing key management system"
    
    # Create directories
    if (-not (Test-Path $KeyDir)) {
        New-Item -ItemType Directory -Path $KeyDir -Force | Out-Null
    }
    
    if (-not (Test-Path $KeyBackupDir)) {
        New-Item -ItemType Directory -Path $KeyBackupDir -Force | Out-Null
    }
    
    # Generate initial key if it doesn't exist
    if (-not (Test-Path $ActiveKeyFile)) {
        Write-Info "No active key found, generating initial key"
        New-EncryptionKey -Path $ActiveKeyFile -Version "v1"
    } else {
        Write-Warning "Active key already exists: $ActiveKeyFile"
    }
    
    Write-Success "Key management system initialized"
}

################################################################################
# Key Rotation Functions
################################################################################

function Invoke-KeyRotation {
    Write-Info "Starting key rotation"
    
    if (-not (Test-Path $ActiveKeyFile)) {
        Write-ErrorMessage "No active key found to rotate"
        return $false
    }
    
    # Determine next version number
    $currentVersion = Get-KeyVersion -KeyFile $ActiveKeyFile
    $nextVersion = Get-NextVersion -Version $currentVersion
    
    Write-Info "Current version: $currentVersion"
    Write-Info "Next version: $nextVersion"
    
    # Backup current key
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupFile = Join-Path $KeyBackupDir "backup.key.$currentVersion.$timestamp"
    
    Write-Info "Backing up current key to: $backupFile"
    Copy-Item -Path $ActiveKeyFile -Destination $backupFile
    Copy-Item -Path "$ActiveKeyFile.meta" -Destination "$backupFile.meta"
    
    # Generate new key
    $tempKey = Join-Path $KeyDir "backup.key.new"
    if (New-EncryptionKey -Path $tempKey -Version $nextVersion) {
        # Replace active key
        Move-Item -Path $tempKey -Destination $ActiveKeyFile -Force
        Move-Item -Path "$tempKey.meta" -Destination "$ActiveKeyFile.meta" -Force
        
        Write-Success "Key rotation completed successfully"
        Write-Info "Old key backed up to: $backupFile"
        Write-Info "New key version: $nextVersion"
        
        return $true
    } else {
        Write-ErrorMessage "Failed to generate new key"
        return $false
    }
}

function Test-RotationNeeded {
    if (-not (Test-Path $ActiveKeyFile)) {
        Write-ErrorMessage "No active key found"
        return $false
    }
    
    $metadataFile = "$ActiveKeyFile.meta"
    
    if (-not (Test-Path $metadataFile)) {
        Write-Warning "No metadata found for active key"
        return $true  # Assume rotation needed
    }
    
    try {
        $metadata = Get-Content $metadataFile | ConvertFrom-Json
        $createdAt = [DateTime]::Parse($metadata.created_at)
        $ageDays = ((Get-Date) - $createdAt).Days
        
        Write-Info "Key age: $ageDays days (rotation threshold: $RotationDays days)"
        
        if ($ageDays -ge $RotationDays) {
            Write-Warning "Key rotation needed (age: $ageDays days)"
            return $true
        } else {
            Write-Info "Key rotation not needed yet"
            return $false
        }
    } catch {
        Write-Warning "Failed to parse metadata: $_"
        return $true
    }
}

function Get-KeyVersion {
    param([string]$KeyFile)
    
    $metadataFile = "$KeyFile.meta"
    
    if (Test-Path $metadataFile) {
        try {
            $metadata = Get-Content $metadataFile | ConvertFrom-Json
            return $metadata.key_version
        } catch {
            return "v1"
        }
    } else {
        return "v1"
    }
}

function Get-NextVersion {
    param([string]$Version)
    
    $number = [int]($Version -replace 'v', '')
    $nextNumber = $number + 1
    return "v$nextNumber"
}

################################################################################
# Key Backup and Recovery Functions
################################################################################

function Backup-EncryptionKey {
    param([string]$DestinationPath)
    
    if (-not (Test-Path $ActiveKeyFile)) {
        Write-ErrorMessage "No active key found to backup"
        return $false
    }
    
    Write-Info "Backing up key to: $DestinationPath"
    
    # Create destination directory
    $destDir = Split-Path $DestinationPath -Parent
    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }
    
    try {
        # Copy key and metadata
        Copy-Item -Path $ActiveKeyFile -Destination $DestinationPath
        Copy-Item -Path "$ActiveKeyFile.meta" -Destination "$DestinationPath.meta"
        
        Write-Success "Key backed up successfully"
        return $true
    } catch {
        Write-ErrorMessage "Failed to backup key: $_"
        return $false
    }
}

function Restore-EncryptionKey {
    param([string]$BackupPath)
    
    if (-not (Test-Path $BackupPath)) {
        Write-ErrorMessage "Backup file not found: $BackupPath"
        return $false
    }
    
    Write-Warning "Restoring key from backup: $BackupPath"
    Write-Warning "This will replace the current active key!"
    
    $confirm = Read-Host "Are you sure you want to continue? (yes/no)"
    
    if ($confirm -ne "yes") {
        Write-Info "Restore cancelled"
        return $false
    }
    
    try {
        # Backup current key before restoring
        if (Test-Path $ActiveKeyFile) {
            $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
            $safetyBackup = Join-Path $KeyBackupDir "backup.key.before_restore.$timestamp"
            Write-Info "Creating safety backup: $safetyBackup"
            Copy-Item -Path $ActiveKeyFile -Destination $safetyBackup
            Copy-Item -Path "$ActiveKeyFile.meta" -Destination "$safetyBackup.meta"
        }
        
        # Restore key
        Copy-Item -Path $BackupPath -Destination $ActiveKeyFile -Force
        if (Test-Path "$BackupPath.meta") {
            Copy-Item -Path "$BackupPath.meta" -Destination "$ActiveKeyFile.meta" -Force
        }
        
        Write-Success "Key restored successfully"
        return $true
    } catch {
        Write-ErrorMessage "Failed to restore key: $_"
        return $false
    }
}

function Get-KeyBackups {
    Write-Info "Available key backups:"
    
    if (-not (Test-Path $KeyBackupDir)) {
        Write-Warning "No backup directory found"
        return
    }
    
    $backups = Get-ChildItem -Path $KeyBackupDir -Filter "backup.key.*" -File | Sort-Object LastWriteTime -Descending
    
    if ($backups.Count -eq 0) {
        Write-Info "No backups found"
        return
    }
    
    $count = 0
    foreach ($backup in $backups) {
        $count++
        Write-Host "  $count. $($backup.Name)"
        Write-Host "     Date: $($backup.LastWriteTime.ToString('yyyy-MM-dd HH:mm:ss'))"
        Write-Host "     Size: $($backup.Length) bytes"
        
        $metaFile = "$($backup.FullName).meta"
        if (Test-Path $metaFile) {
            try {
                $metadata = Get-Content $metaFile | ConvertFrom-Json
                Write-Host "     Version: $($metadata.key_version)"
            } catch {}
        }
        
        Write-Host ""
    }
    
    Write-Info "Total backups: $count"
}

################################################################################
# Key Verification Functions
################################################################################

function Test-KeyIntegrity {
    param([string]$Path = $ActiveKeyFile)
    
    if (-not (Test-Path $Path)) {
        Write-ErrorMessage "Key file not found: $Path"
        return $false
    }
    
    Write-Info "Verifying key: $Path"
    
    # Check file size
    $size = (Get-Item $Path).Length
    if ($size -lt 32) {
        Write-ErrorMessage "Key file too small: $size bytes"
        return $false
    } else {
        Write-Success "File size OK: $size bytes"
    }
    
    # Verify checksum if metadata exists
    $metadataFile = "$Path.meta"
    if (Test-Path $metadataFile) {
        try {
            $metadata = Get-Content $metadataFile | ConvertFrom-Json
            $expectedChecksum = $metadata.key_checksum
            $actualChecksum = (Get-FileHash -Path $Path -Algorithm SHA256).Hash
            
            if ($expectedChecksum -eq $actualChecksum) {
                Write-Success "Checksum verified"
            } else {
                Write-ErrorMessage "Checksum mismatch!"
                Write-ErrorMessage "Expected: $expectedChecksum"
                Write-ErrorMessage "Actual: $actualChecksum"
                return $false
            }
        } catch {
            Write-Warning "Failed to verify checksum: $_"
        }
    } else {
        Write-Warning "No metadata found for checksum verification"
    }
    
    Write-Success "Key verification passed"
    return $true
}

################################################################################
# Usage Information
################################################################################

function Show-Usage {
    Write-Host @"
Usage: .\key-management.ps1 -Command <command> [options]

Commands:
  init                       Initialize key management system
  generate                   Generate a new encryption key
  rotate                     Rotate the active encryption key
  check-rotation             Check if key rotation is needed
  backup                     Backup active key to destination
  restore                    Restore key from backup
  list-backups               List all key backups
  verify                     Verify key integrity

Options:
  -KeyFile <file>            Key file path (for generate command)
  -Destination <path>        Destination path (for backup command)
  -BackupFile <file>         Backup file path (for restore command)
  -KeyDir <dir>              Key directory (default: C:\tariff-system\keys)
  -RotationDays <num>        Key rotation period in days (default: 90)
  -KeySize <num>             Key size in bytes (default: 32)

Examples:
  # Initialize key management
  .\key-management.ps1 -Command init

  # Generate a new key
  .\key-management.ps1 -Command generate -KeyFile C:\path\to\new.key

  # Rotate the active key
  .\key-management.ps1 -Command rotate

  # Check if rotation is needed
  .\key-management.ps1 -Command check-rotation

  # Backup active key
  .\key-management.ps1 -Command backup -Destination C:\secure\backup.key

  # Restore from backup
  .\key-management.ps1 -Command restore -BackupFile C:\secure\backup.key

  # List all backups
  .\key-management.ps1 -Command list-backups

  # Verify key integrity
  .\key-management.ps1 -Command verify

"@
}

################################################################################
# Main Execution
################################################################################

if (-not $Command) {
    Show-Usage
    exit 1
}

switch ($Command.ToLower()) {
    "init" {
        Initialize-KeyManagement
    }
    
    "generate" {
        if (-not $KeyFile) {
            Write-ErrorMessage "Usage: -Command generate -KeyFile <key_file>"
            exit 1
        }
        $result = New-EncryptionKey -Path $KeyFile -Version "v1"
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "rotate" {
        $result = Invoke-KeyRotation
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "check-rotation" {
        $result = Test-RotationNeeded
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "backup" {
        if (-not $Destination) {
            Write-ErrorMessage "Usage: -Command backup -Destination <destination>"
            exit 1
        }
        $result = Backup-EncryptionKey -DestinationPath $Destination
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "restore" {
        if (-not $BackupFile) {
            Write-ErrorMessage "Usage: -Command restore -BackupFile <backup_file>"
            exit 1
        }
        $result = Restore-EncryptionKey -BackupPath $BackupFile
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "list-backups" {
        Get-KeyBackups
    }
    
    "verify" {
        $path = if ($KeyFile) { $KeyFile } else { $ActiveKeyFile }
        $result = Test-KeyIntegrity -Path $path
        exit $(if ($result) { 0 } else { 1 })
    }
    
    default {
        Write-ErrorMessage "Unknown command: $Command"
        Show-Usage
        exit 1
    }
}
