# Backup Encryption Test Script (PowerShell)
# Windows-compatible version of test-backup-encryption.sh

param(
    [string]$TestDir = ".\test-encryption"
)

$ErrorActionPreference = "Continue"

# Configuration
$KeyDir = Join-Path $TestDir "keys"
$BackupDir = Join-Path $TestDir "backups"
$RestoreDir = Join-Path $TestDir "restore"
$LogFile = Join-Path $TestDir "test-results.log"

# Test counters
$script:TestsRun = 0
$script:TestsPassed = 0
$script:TestsFailed = 0

################################################################################
# Logging Functions
################################################################################

function Write-TestLog {
    param([string]$Message)
    Write-Host "[TEST] $Message" -ForegroundColor Blue
    Add-Content -Path $LogFile -Value "[TEST] $Message"
}

function Write-Pass {
    param([string]$Message)
    Write-Host "[PASS] $Message" -ForegroundColor Green
    Add-Content -Path $LogFile -Value "[PASS] $Message"
    $script:TestsPassed++
}

function Write-Fail {
    param([string]$Message)
    Write-Host "[FAIL] $Message" -ForegroundColor Red
    Add-Content -Path $LogFile -Value "[FAIL] $Message"
    $script:TestsFailed++
}

function Write-TestInfo {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
    Add-Content -Path $LogFile -Value "[INFO] $Message"
}

################################################################################
# Setup and Cleanup
################################################################################

function Initialize-TestEnvironment {
    Write-TestInfo "Setting up test environment"
    
    # Create directories
    New-Item -ItemType Directory -Path $TestDir -Force | Out-Null
    New-Item -ItemType Directory -Path $KeyDir -Force | Out-Null
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
    New-Item -ItemType Directory -Path $RestoreDir -Force | Out-Null
    
    # Set environment variables
    $env:KEY_DIR = $KeyDir
    $env:ENCRYPTION_KEY_FILE = Join-Path $KeyDir "backup.key"
    $env:BACKUP_DIR = $BackupDir
    
    # Create test data
    "Test data for encryption" | Set-Content -Path (Join-Path $TestDir "test-file.txt")
    "SELECT * FROM tariffs;" | Set-Content -Path (Join-Path $TestDir "test-backup.sql")
    
    # Create larger test file (10MB)
    $largeFile = Join-Path $TestDir "large-file.bin"
    $bytes = New-Object byte[] (10 * 1024 * 1024)
    $rng = New-Object System.Security.Cryptography.RNGCryptoServiceProvider
    $rng.GetBytes($bytes)
    [System.IO.File]::WriteAllBytes($largeFile, $bytes)
    
    Write-TestInfo "Test environment ready"
}

function Remove-TestEnvironment {
    Write-TestInfo "Cleaning up test environment"
    Remove-Item -Path $TestDir -Recurse -Force -ErrorAction SilentlyContinue
}

################################################################################
# Test Functions
################################################################################

function Test-KeyInitialization {
    $script:TestsRun++
    Write-TestLog "Test 1: Key Initialization"
    
    try {
        .\scripts\key-management.ps1 -Command init -KeyDir $KeyDir
        
        $keyFile = Join-Path $KeyDir "backup.key"
        if (Test-Path $keyFile) {
            Write-Pass "Key initialized successfully"
            return $true
        } else {
            Write-Fail "Key file not created"
            return $false
        }
    } catch {
        Write-Fail "Key initialization failed: $_"
        return $false
    }
}

function Test-EncryptFile {
    $script:TestsRun++
    Write-TestLog "Test 2: File Encryption"
    
    $inputFile = Join-Path $TestDir "test-file.txt"
    $outputFile = Join-Path $BackupDir "test-file.txt.enc"
    
    try {
        .\scripts\encrypt-backup.ps1 -Command encrypt `
            -InputFile $inputFile `
            -OutputFile $outputFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        if ((Test-Path $outputFile) -and (Test-Path "$outputFile.meta")) {
            Write-Pass "File encrypted successfully"
            return $true
        } else {
            Write-Fail "Encrypted file or metadata not created"
            return $false
        }
    } catch {
        Write-Fail "Encryption failed: $_"
        return $false
    }
}

function Test-DecryptFile {
    $script:TestsRun++
    Write-TestLog "Test 3: File Decryption"
    
    $inputFile = Join-Path $BackupDir "test-file.txt.enc"
    $outputFile = Join-Path $RestoreDir "test-file.txt"
    
    try {
        .\scripts\encrypt-backup.ps1 -Command decrypt `
            -InputFile $inputFile `
            -OutputFile $outputFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        if (Test-Path $outputFile) {
            Write-Pass "File decrypted successfully"
            return $true
        } else {
            Write-Fail "Decrypted file not created"
            return $false
        }
    } catch {
        Write-Fail "Decryption failed: $_"
        return $false
    }
}

function Test-RoundtripIntegrity {
    $script:TestsRun++
    Write-TestLog "Test 4: Round-trip Integrity"
    
    $originalFile = Join-Path $TestDir "test-file.txt"
    $decryptedFile = Join-Path $RestoreDir "test-file.txt"
    
    $originalHash = (Get-FileHash -Path $originalFile -Algorithm SHA256).Hash
    $decryptedHash = (Get-FileHash -Path $decryptedFile -Algorithm SHA256).Hash
    
    if ($originalHash -eq $decryptedHash) {
        Write-Pass "Round-trip integrity verified (checksums match)"
        return $true
    } else {
        Write-Fail "Round-trip integrity failed (checksums differ)"
        Write-Fail "Original: $originalHash"
        Write-Fail "Decrypted: $decryptedHash"
        return $false
    }
}

function Test-EncryptionVerification {
    $script:TestsRun++
    Write-TestLog "Test 5: Encryption Verification"
    
    $encryptedFile = Join-Path $BackupDir "test-file.txt.enc"
    
    try {
        .\scripts\encrypt-backup.ps1 -Command verify `
            -InputFile $encryptedFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        Write-Pass "Encryption verification passed"
        return $true
    } catch {
        Write-Fail "Encryption verification failed: $_"
        return $false
    }
}

function Test-LargeFileEncryption {
    $script:TestsRun++
    Write-TestLog "Test 6: Large File Encryption"
    
    $inputFile = Join-Path $TestDir "large-file.bin"
    $encryptedFile = Join-Path $BackupDir "large-file.bin.enc"
    $decryptedFile = Join-Path $RestoreDir "large-file.bin"
    
    try {
        Write-TestInfo "Encrypting 10MB file..."
        .\scripts\encrypt-backup.ps1 -Command encrypt `
            -InputFile $inputFile `
            -OutputFile $encryptedFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        Write-TestInfo "Decrypting 10MB file..."
        .\scripts\encrypt-backup.ps1 -Command decrypt `
            -InputFile $encryptedFile `
            -OutputFile $decryptedFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        $originalHash = (Get-FileHash -Path $inputFile -Algorithm SHA256).Hash
        $decryptedHash = (Get-FileHash -Path $decryptedFile -Algorithm SHA256).Hash
        
        if ($originalHash -eq $decryptedHash) {
            Write-Pass "Large file encryption/decryption successful"
            return $true
        } else {
            Write-Fail "Large file integrity check failed"
            return $false
        }
    } catch {
        Write-Fail "Large file encryption failed: $_"
        return $false
    }
}

function Test-KeyRotation {
    $script:TestsRun++
    Write-TestLog "Test 7: Key Rotation"
    
    $keyFile = Join-Path $KeyDir "backup.key"
    $oldHash = (Get-FileHash -Path $keyFile -Algorithm SHA256).Hash
    
    try {
        .\scripts\key-management.ps1 -Command rotate -KeyDir $KeyDir
        
        $newHash = (Get-FileHash -Path $keyFile -Algorithm SHA256).Hash
        
        if ($oldHash -ne $newHash) {
            Write-Pass "Key rotated successfully"
            return $true
        } else {
            Write-Fail "Key rotation did not change the key"
            return $false
        }
    } catch {
        Write-Fail "Key rotation failed: $_"
        return $false
    }
}

function Test-KeyBackup {
    $script:TestsRun++
    Write-TestLog "Test 8: Key Backup"
    
    $backupFile = Join-Path $TestDir "backup-key-test.key"
    
    try {
        .\scripts\key-management.ps1 -Command backup `
            -Destination $backupFile `
            -KeyDir $KeyDir
        
        if ((Test-Path $backupFile) -and (Test-Path "$backupFile.meta")) {
            Write-Pass "Key backed up successfully"
            return $true
        } else {
            Write-Fail "Backup file not created"
            return $false
        }
    } catch {
        Write-Fail "Key backup failed: $_"
        return $false
    }
}

function Test-KeyVerification {
    $script:TestsRun++
    Write-TestLog "Test 9: Key Verification"
    
    try {
        .\scripts\key-management.ps1 -Command verify -KeyDir $KeyDir
        
        Write-Pass "Key verification passed"
        return $true
    } catch {
        Write-Fail "Key verification failed: $_"
        return $false
    }
}

function Test-MetadataCreation {
    $script:TestsRun++
    Write-TestLog "Test 10: Metadata Creation"
    
    $metadataFile = Join-Path $BackupDir "test-file.txt.enc.meta"
    
    if (Test-Path $metadataFile) {
        $metadata = Get-Content $metadataFile | ConvertFrom-Json
        
        if ($metadata.encryption_algorithm -and $metadata.key_derivation -and 
            $metadata.original_checksum -and $metadata.encrypted_checksum) {
            Write-Pass "Metadata contains all required fields"
            return $true
        } else {
            Write-Fail "Metadata missing required fields"
            return $false
        }
    } else {
        Write-Fail "Metadata file not found"
        return $false
    }
}

function Test-SQLBackupEncryption {
    $script:TestsRun++
    Write-TestLog "Test 11: SQL Backup Encryption"
    
    $inputFile = Join-Path $TestDir "test-backup.sql"
    $encryptedFile = Join-Path $BackupDir "test-backup.sql.enc"
    $decryptedFile = Join-Path $RestoreDir "test-backup.sql"
    
    try {
        .\scripts\encrypt-backup.ps1 -Command encrypt `
            -InputFile $inputFile `
            -OutputFile $encryptedFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        .\scripts\encrypt-backup.ps1 -Command decrypt `
            -InputFile $encryptedFile `
            -OutputFile $decryptedFile `
            -KeyFile (Join-Path $KeyDir "backup.key")
        
        $originalContent = Get-Content $inputFile -Raw
        $decryptedContent = Get-Content $decryptedFile -Raw
        
        if ($originalContent -eq $decryptedContent) {
            Write-Pass "SQL backup encryption/decryption successful"
            return $true
        } else {
            Write-Fail "SQL backup content differs after round-trip"
            return $false
        }
    } catch {
        Write-Fail "SQL backup encryption failed: $_"
        return $false
    }
}

function Test-WrongKeyDecryption {
    $script:TestsRun++
    Write-TestLog "Test 12: Wrong Key Decryption (Should Fail)"
    
    $encryptedFile = Join-Path $BackupDir "test-file.txt.enc"
    $wrongKey = Join-Path $TestDir "wrong.key"
    $outputFile = Join-Path $RestoreDir "test-wrong-key.txt"
    
    # Generate wrong key
    & openssl rand -base64 32 | Set-Content -Path $wrongKey -NoNewline
    
    try {
        .\scripts\encrypt-backup.ps1 -Command decrypt `
            -InputFile $encryptedFile `
            -OutputFile $outputFile `
            -KeyFile $wrongKey `
            -ErrorAction Stop
        
        Write-Fail "Decryption with wrong key should have failed"
        return $false
    } catch {
        Write-Pass "Decryption correctly failed with wrong key"
        return $true
    }
}

################################################################################
# Main Test Execution
################################################################################

Write-Host "=========================================="
Write-Host "Backup Encryption Test Suite"
Write-Host "=========================================="
Write-Host ""

# Setup
Initialize-TestEnvironment

# Run tests
Test-KeyInitialization
Test-EncryptFile
Test-DecryptFile
Test-RoundtripIntegrity
Test-EncryptionVerification
Test-LargeFileEncryption
Test-KeyRotation
Test-KeyBackup
Test-KeyVerification
Test-MetadataCreation
Test-SQLBackupEncryption
Test-WrongKeyDecryption

# Cleanup
Remove-TestEnvironment

# Results
Write-Host ""
Write-Host "=========================================="
Write-Host "Test Results"
Write-Host "=========================================="
Write-Host "Tests Run:    $TestsRun"
Write-Host "Tests Passed: $TestsPassed" -ForegroundColor Green
Write-Host "Tests Failed: $TestsFailed" -ForegroundColor Red
Write-Host "=========================================="

if ($TestsFailed -eq 0) {
    Write-Host "All tests passed!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "Some tests failed!" -ForegroundColor Red
    exit 1
}
