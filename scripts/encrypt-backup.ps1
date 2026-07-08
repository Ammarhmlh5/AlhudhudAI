# Backup Encryption Utility (PowerShell)
# Windows-compatible version of encrypt-backup.sh

param(
    [Parameter(Mandatory=$false)]
    [string]$Command,
    
    [Parameter(Mandatory=$false)]
    [string]$InputFile,
    
    [Parameter(Mandatory=$false)]
    [string]$OutputFile,
    
    [Parameter(Mandatory=$false)]
    [string]$InputDir,
    
    [Parameter(Mandatory=$false)]
    [string]$OutputDir,
    
    [Parameter(Mandatory=$false)]
    [string]$KeyFile = "C:\tariff-system\backup.key",
    
    [Parameter(Mandatory=$false)]
    [string]$Algorithm = "aes-256-cbc",
    
    [Parameter(Mandatory=$false)]
    [int]$Iterations = 100000
)

$ErrorActionPreference = "Stop"

################################################################################
# Logging Functions
################################################################################

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-ErrorMessage {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

################################################################################
# Encryption Functions
################################################################################

function Encrypt-File {
    param(
        [string]$InputPath,
        [string]$OutputPath,
        [string]$Key = $KeyFile
    )
    
    if (-not (Test-Path $InputPath)) {
        Write-ErrorMessage "Input file not found: $InputPath"
        return $false
    }
    
    if (-not (Test-Path $Key)) {
        Write-ErrorMessage "Encryption key file not found: $Key"
        return $false
    }
    
    Write-Info "Encrypting: $(Split-Path $InputPath -Leaf)"
    Write-Info "Algorithm: $Algorithm"
    Write-Info "Key derivation: pbkdf2 (iterations: $Iterations)"
    
    try {
        # Encrypt using OpenSSL
        & openssl enc -$Algorithm -salt -pbkdf2 -iter $Iterations `
            -in $InputPath -out $OutputPath -pass "file:$Key"
        
        if ($LASTEXITCODE -eq 0) {
            # Create metadata
            New-EncryptionMetadata -OriginalFile $InputPath -EncryptedFile $OutputPath -KeyFile $Key
            
            Write-Success "Successfully encrypted: $(Split-Path $OutputPath -Leaf)"
            return $true
        } else {
            Write-ErrorMessage "Failed to encrypt: $(Split-Path $InputPath -Leaf)"
            return $false
        }
    } catch {
        Write-ErrorMessage "Exception during encryption: $_"
        return $false
    }
}

function Decrypt-File {
    param(
        [string]$InputPath,
        [string]$OutputPath,
        [string]$Key = $KeyFile
    )
    
    if (-not (Test-Path $InputPath)) {
        Write-ErrorMessage "Input file not found: $InputPath"
        return $false
    }
    
    if (-not (Test-Path $Key)) {
        Write-ErrorMessage "Decryption key file not found: $Key"
        return $false
    }
    
    Write-Info "Decrypting: $(Split-Path $InputPath -Leaf)"
    
    try {
        # Decrypt using OpenSSL
        & openssl enc -d -$Algorithm -pbkdf2 -iter $Iterations `
            -in $InputPath -out $OutputPath -pass "file:$Key"
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Successfully decrypted: $(Split-Path $OutputPath -Leaf)"
            return $true
        } else {
            Write-ErrorMessage "Failed to decrypt: $(Split-Path $InputPath -Leaf)"
            return $false
        }
    } catch {
        Write-ErrorMessage "Exception during decryption: $_"
        return $false
    }
}

function New-EncryptionMetadata {
    param(
        [string]$OriginalFile,
        [string]$EncryptedFile,
        [string]$KeyFile
    )
    
    $metadataFile = "$EncryptedFile.meta"
    
    $originalSize = (Get-Item $OriginalFile).Length
    $encryptedSize = (Get-Item $EncryptedFile).Length
    $originalChecksum = (Get-FileHash -Path $OriginalFile -Algorithm SHA256).Hash
    $encryptedChecksum = (Get-FileHash -Path $EncryptedFile -Algorithm SHA256).Hash
    
    $metadata = @{
        original_file = Split-Path $OriginalFile -Leaf
        encrypted_file = Split-Path $EncryptedFile -Leaf
        encryption_algorithm = $Algorithm
        key_derivation = "pbkdf2"
        pbkdf2_iterations = $Iterations
        original_size = $originalSize
        encrypted_size = $encryptedSize
        original_checksum = $originalChecksum
        encrypted_checksum = $encryptedChecksum
        encrypted_at = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        hostname = $env:COMPUTERNAME
        key_file = Split-Path $KeyFile -Leaf
    }
    
    $metadata | ConvertTo-Json | Set-Content -Path $metadataFile
    
    Write-Info "Metadata created: $(Split-Path $metadataFile -Leaf)"
}

function Test-EncryptionIntegrity {
    param([string]$EncryptedFile)
    
    if (-not (Test-Path $EncryptedFile)) {
        Write-ErrorMessage "Encrypted file not found: $EncryptedFile"
        return $false
    }
    
    $metadataFile = "$EncryptedFile.meta"
    
    if (-not (Test-Path $metadataFile)) {
        Write-Warning "Metadata file not found: $metadataFile"
        return $false
    }
    
    Write-Info "Verifying encryption: $(Split-Path $EncryptedFile -Leaf)"
    
    try {
        $metadata = Get-Content $metadataFile | ConvertFrom-Json
        
        # Check file size
        $actualSize = (Get-Item $EncryptedFile).Length
        $expectedSize = $metadata.encrypted_size
        
        if ($actualSize -ne $expectedSize) {
            Write-ErrorMessage "File size mismatch: expected $expectedSize, got $actualSize"
            return $false
        }
        
        # Check checksum
        $actualChecksum = (Get-FileHash -Path $EncryptedFile -Algorithm SHA256).Hash
        $expectedChecksum = $metadata.encrypted_checksum
        
        if ($actualChecksum -ne $expectedChecksum) {
            Write-ErrorMessage "Checksum mismatch"
            return $false
        }
        
        Write-Success "Encryption verified successfully"
        return $true
    } catch {
        Write-ErrorMessage "Exception during verification: $_"
        return $false
    }
}

function Test-EncryptionRoundtrip {
    param(
        [string]$TestFile,
        [string]$Key = $KeyFile
    )
    
    if (-not (Test-Path $TestFile)) {
        Write-ErrorMessage "Test file not found: $TestFile"
        return $false
    }
    
    Write-Info "Testing encryption round-trip for: $(Split-Path $TestFile -Leaf)"
    
    $tempDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString()))
    $encryptedFile = Join-Path $tempDir "$(Split-Path $TestFile -Leaf).enc"
    $decryptedFile = Join-Path $tempDir "$(Split-Path $TestFile -Leaf).dec"
    
    try {
        # Calculate original checksum
        $originalChecksum = (Get-FileHash -Path $TestFile -Algorithm SHA256).Hash
        
        # Encrypt
        if (-not (Encrypt-File -InputPath $TestFile -OutputPath $encryptedFile -Key $Key)) {
            Remove-Item -Path $tempDir -Recurse -Force
            return $false
        }
        
        # Verify encryption
        if (-not (Test-EncryptionIntegrity -EncryptedFile $encryptedFile)) {
            Remove-Item -Path $tempDir -Recurse -Force
            return $false
        }
        
        # Decrypt
        if (-not (Decrypt-File -InputPath $encryptedFile -OutputPath $decryptedFile -Key $Key)) {
            Remove-Item -Path $tempDir -Recurse -Force
            return $false
        }
        
        # Calculate decrypted checksum
        $decryptedChecksum = (Get-FileHash -Path $decryptedFile -Algorithm SHA256).Hash
        
        # Compare checksums
        if ($originalChecksum -eq $decryptedChecksum) {
            Write-Success "Round-trip test passed: checksums match"
            Remove-Item -Path $tempDir -Recurse -Force
            return $true
        } else {
            Write-ErrorMessage "Round-trip test failed: checksums do not match"
            Write-ErrorMessage "Original: $originalChecksum"
            Write-ErrorMessage "Decrypted: $decryptedChecksum"
            Remove-Item -Path $tempDir -Recurse -Force
            return $false
        }
    } catch {
        Write-ErrorMessage "Exception during round-trip test: $_"
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        return $false
    }
}

function Encrypt-Directory {
    param(
        [string]$InputDirectory,
        [string]$OutputDirectory,
        [string]$Key = $KeyFile
    )
    
    if (-not (Test-Path $InputDirectory)) {
        Write-ErrorMessage "Input directory not found: $InputDirectory"
        return $false
    }
    
    if (-not (Test-Path $OutputDirectory)) {
        New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
    }
    
    Write-Info "Encrypting directory: $InputDirectory"
    
    $successCount = 0
    $failCount = 0
    
    Get-ChildItem -Path $InputDirectory -File -Recurse | ForEach-Object {
        $relativePath = $_.FullName.Substring($InputDirectory.Length + 1)
        $outputFile = Join-Path $OutputDirectory "$relativePath.enc"
        $outputSubdir = Split-Path $outputFile -Parent
        
        if (-not (Test-Path $outputSubdir)) {
            New-Item -ItemType Directory -Path $outputSubdir -Force | Out-Null
        }
        
        if (Encrypt-File -InputPath $_.FullName -OutputPath $outputFile -Key $Key) {
            $successCount++
        } else {
            $failCount++
        }
    }
    
    Write-Info "Encryption complete: $successCount succeeded, $failCount failed"
    
    return ($failCount -eq 0)
}

################################################################################
# Usage Information
################################################################################

function Show-Usage {
    Write-Host @"
Usage: .\encrypt-backup.ps1 -Command <command> [options]

Commands:
  encrypt       Encrypt a file
  decrypt       Decrypt a file
  verify        Verify encryption integrity
  test          Test encryption round-trip
  encrypt-dir   Encrypt all files in directory

Options:
  -InputFile <file>          Input file path
  -OutputFile <file>         Output file path
  -InputDir <dir>            Input directory path
  -OutputDir <dir>           Output directory path
  -KeyFile <file>            Encryption key file (default: C:\tariff-system\backup.key)
  -Algorithm <algo>          Encryption algorithm (default: aes-256-cbc)
  -Iterations <num>          PBKDF2 iterations (default: 100000)

Examples:
  # Encrypt a backup file
  .\encrypt-backup.ps1 -Command encrypt -InputFile backup.sql -OutputFile backup.sql.enc

  # Decrypt a backup file
  .\encrypt-backup.ps1 -Command decrypt -InputFile backup.sql.enc -OutputFile backup.sql

  # Verify encryption
  .\encrypt-backup.ps1 -Command verify -InputFile backup.sql.enc

  # Test encryption round-trip
  .\encrypt-backup.ps1 -Command test -InputFile backup.sql

  # Encrypt entire directory
  .\encrypt-backup.ps1 -Command encrypt-dir -InputDir C:\backups\daily -OutputDir C:\backups\daily-encrypted

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
    "encrypt" {
        if (-not $InputFile -or -not $OutputFile) {
            Write-ErrorMessage "Usage: -Command encrypt -InputFile <input> -OutputFile <output> [-KeyFile <key>]"
            exit 1
        }
        $result = Encrypt-File -InputPath $InputFile -OutputPath $OutputFile -Key $KeyFile
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "decrypt" {
        if (-not $InputFile -or -not $OutputFile) {
            Write-ErrorMessage "Usage: -Command decrypt -InputFile <input> -OutputFile <output> [-KeyFile <key>]"
            exit 1
        }
        $result = Decrypt-File -InputPath $InputFile -OutputPath $OutputFile -Key $KeyFile
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "verify" {
        if (-not $InputFile) {
            Write-ErrorMessage "Usage: -Command verify -InputFile <encrypted_file>"
            exit 1
        }
        $result = Test-EncryptionIntegrity -EncryptedFile $InputFile
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "test" {
        if (-not $InputFile) {
            Write-ErrorMessage "Usage: -Command test -InputFile <file> [-KeyFile <key>]"
            exit 1
        }
        $result = Test-EncryptionRoundtrip -TestFile $InputFile -Key $KeyFile
        exit $(if ($result) { 0 } else { 1 })
    }
    
    "encrypt-dir" {
        if (-not $InputDir -or -not $OutputDir) {
            Write-ErrorMessage "Usage: -Command encrypt-dir -InputDir <input_dir> -OutputDir <output_dir> [-KeyFile <key>]"
            exit 1
        }
        $result = Encrypt-Directory -InputDirectory $InputDir -OutputDirectory $OutputDir -Key $KeyFile
        exit $(if ($result) { 0 } else { 1 })
    }
    
    default {
        Write-ErrorMessage "Unknown command: $Command"
        Show-Usage
        exit 1
    }
}
