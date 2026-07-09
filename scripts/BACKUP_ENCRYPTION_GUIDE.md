# Backup Encryption Guide

## Overview

This guide provides comprehensive documentation for the backup encryption system used in the Tariff Pricing System. All backups are encrypted using AES-256-CBC encryption with PBKDF2 key derivation to ensure data security and compliance with requirement 25.6.

## Table of Contents

1. [Encryption Algorithm Details](#encryption-algorithm-details)
2. [Key Generation Procedures](#key-generation-procedures)
3. [Key Rotation Best Practices](#key-rotation-best-practices)
4. [Decryption Procedures for Restore](#decryption-procedures-for-restore)
5. [Security Considerations](#security-considerations)
6. [Troubleshooting](#troubleshooting)

---

## Encryption Algorithm Details

### Algorithm: AES-256-CBC

**Advanced Encryption Standard (AES)** with 256-bit key in **Cipher Block Chaining (CBC)** mode.

**Key Features:**
- **Symmetric encryption**: Same key for encryption and decryption
- **Block cipher**: Operates on 128-bit blocks
- **Key size**: 256 bits (32 bytes)
- **Security level**: Military-grade encryption
- **Performance**: Fast encryption/decryption on modern hardware

### Key Derivation: PBKDF2

**Password-Based Key Derivation Function 2 (PBKDF2)** with SHA-256.

**Parameters:**
- **Hash function**: SHA-256
- **Iterations**: 100,000 (configurable)
- **Salt**: Randomly generated for each encryption
- **Output**: 256-bit encryption key

**Why PBKDF2?**
- Resistant to brute-force attacks
- Computationally expensive for attackers
- Industry-standard key derivation
- NIST recommended

### Encryption Process

```
1. Generate random salt (OpenSSL automatic)
2. Derive encryption key from master key using PBKDF2
3. Encrypt data using AES-256-CBC
4. Prepend salt to encrypted data
5. Create metadata file with checksums
```

### File Format

**Encrypted File Structure:**
```
[Salt (8 bytes)][Encrypted Data (variable)]
```

**Metadata File (.meta):**
```json
{
  "original_file": "backup.sql",
  "encrypted_file": "backup.sql.enc",
  "encryption_algorithm": "aes-256-cbc",
  "key_derivation": "pbkdf2",
  "pbkdf2_iterations": 100000,
  "original_size": 1048576,
  "encrypted_size": 1048592,
  "original_checksum": "sha256_hash",
  "encrypted_checksum": "sha256_hash",
  "encrypted_at": "2024-01-15T10:30:00Z",
  "hostname": "backup-server",
  "key_file": "backup.key"
}
```

---

## Key Generation Procedures

### Initial Setup

**Linux/macOS:**
```bash
# Initialize key management system
sudo ./scripts/key-management.sh init

# This creates:
# - /etc/tariff-system/keys/backup.key (active key)
# - /etc/tariff-system/keys/backups/ (backup directory)
```

**Windows:**
```powershell
# Initialize key management system
.\scripts\key-management.ps1 -Command init

# This creates:
# - C:\tariff-system\keys\backup.key (active key)
# - C:\tariff-system\keys\backups\ (backup directory)
```

### Manual Key Generation

**Linux/macOS:**
```bash
# Generate a new key
sudo ./scripts/key-management.sh generate /path/to/new.key

# The key is generated using:
openssl rand -base64 32 > /path/to/new.key

# Permissions are automatically set to 400 (read-only for owner)
```

**Windows:**
```powershell
# Generate a new key
.\scripts\key-management.ps1 -Command generate -KeyFile C:\path\to\new.key

# The key is generated using OpenSSL
# Permissions are set to allow only current user
```

### Key Storage Locations

**Recommended Locations:**

**Linux/macOS:**
- **Production**: `/etc/tariff-system/keys/backup.key`
- **Development**: `~/.tariff-system/keys/backup.key`
- **Backup**: `/secure/offsite/location/backup.key`

**Windows:**
- **Production**: `C:\tariff-system\keys\backup.key`
- **Development**: `%USERPROFILE%\.tariff-system\keys\backup.key`
- **Backup**: Secure network location or encrypted USB drive

**Security Requirements:**
- Keys must NEVER be stored in version control
- Keys must NEVER be stored in plain text in application code
- Keys must be stored on encrypted filesystems
- Keys must have restrictive permissions (400 on Linux, restricted ACL on Windows)
- Keys should be backed up to secure offsite location

---

## Key Rotation Best Practices

### Why Rotate Keys?

- **Limit exposure**: Reduces impact if key is compromised
- **Compliance**: Many regulations require periodic key rotation
- **Best practice**: Industry standard is 90-day rotation
- **Defense in depth**: Additional security layer

### Rotation Schedule

**Recommended:**
- **Production**: Every 90 days
- **Development**: Every 180 days
- **Testing**: As needed

### Rotation Process

**Linux/macOS:**
```bash
# Check if rotation is needed
sudo ./scripts/key-management.sh check-rotation

# Perform rotation
sudo ./scripts/key-management.sh rotate

# Verify new key
sudo ./scripts/key-management.sh verify
```

**Windows:**
```powershell
# Check if rotation is needed
.\scripts\key-management.ps1 -Command check-rotation

# Perform rotation
.\scripts\key-management.ps1 -Command rotate

# Verify new key
.\scripts\key-management.ps1 -Command verify
```

### What Happens During Rotation?

1. **Backup current key**: Old key is backed up with version and timestamp
2. **Generate new key**: New key is generated with incremented version
3. **Replace active key**: New key becomes the active encryption key
4. **Preserve old keys**: Old keys are kept for decrypting existing backups

### Managing Multiple Key Versions

**Key Versioning:**
```
backup.key.v1.20240115_103000  (January 15, 2024)
backup.key.v2.20240415_103000  (April 15, 2024)
backup.key.v3.20240715_103000  (July 15, 2024)
```

**Decryption Strategy:**
- Always try active key first
- If decryption fails, try previous versions in reverse chronological order
- Keep old keys for at least the retention period of backups encrypted with them

**Example:**
```bash
# List all key backups
sudo ./scripts/key-management.sh list-backups

# Decrypt with specific key version
sudo ./scripts/encrypt-backup.sh decrypt \
  backup.sql.enc \
  backup.sql \
  /etc/tariff-system/keys/backups/backup.key.v2.20240415_103000
```

---

## Decryption Procedures for Restore

### Standard Decryption

**Linux/macOS:**
```bash
# Decrypt a single file
sudo ./scripts/encrypt-backup.sh decrypt \
  /backups/daily/backup.sql.enc \
  /restore/backup.sql

# Decrypt with specific key
sudo ./scripts/encrypt-backup.sh decrypt \
  /backups/daily/backup.sql.enc \
  /restore/backup.sql \
  /path/to/specific.key
```

**Windows:**
```powershell
# Decrypt a single file
.\scripts\encrypt-backup.ps1 -Command decrypt `
  -InputFile C:\backups\daily\backup.sql.enc `
  -OutputFile C:\restore\backup.sql

# Decrypt with specific key
.\scripts\encrypt-backup.ps1 -Command decrypt `
  -InputFile C:\backups\daily\backup.sql.enc `
  -OutputFile C:\restore\backup.sql `
  -KeyFile C:\path\to\specific.key
```

### Verify Before Restore

**Always verify encryption integrity before attempting restore:**

**Linux/macOS:**
```bash
# Verify encrypted backup
sudo ./scripts/encrypt-backup.sh verify /backups/daily/backup.sql.enc

# Output:
# [INFO] Verifying encryption: backup.sql.enc
# [SUCCESS] File size OK: 1048592 bytes
# [SUCCESS] Checksum verified
# [SUCCESS] Encryption verified successfully
```

**Windows:**
```powershell
# Verify encrypted backup
.\scripts\encrypt-backup.ps1 -Command verify `
  -InputFile C:\backups\daily\backup.sql.enc
```

### Restore from Encrypted Backup

**Complete Restore Process:**

**Linux/macOS:**
```bash
#!/bin/bash
# restore-from-encrypted.sh

ENCRYPTED_BACKUP="/backups/daily/tariff_backup_daily_20240115.tar.gz.enc"
TEMP_DIR="/tmp/restore_$$"
DB_NAME="sms_gateway"

# Create temp directory
mkdir -p "$TEMP_DIR"

# Decrypt backup
echo "Decrypting backup..."
./scripts/encrypt-backup.sh decrypt \
  "$ENCRYPTED_BACKUP" \
  "$TEMP_DIR/backup.tar.gz"

# Extract archive
echo "Extracting archive..."
tar -xzf "$TEMP_DIR/backup.tar.gz" -C "$TEMP_DIR"

# Restore each table
for dump_file in "$TEMP_DIR"/*.dump.enc; do
    table_name=$(basename "$dump_file" .dump.enc)
    
    echo "Decrypting $table_name..."
    ./scripts/encrypt-backup.sh decrypt \
        "$dump_file" \
        "$TEMP_DIR/$table_name.dump"
    
    echo "Restoring $table_name..."
    pg_restore -h localhost -U postgres -d "$DB_NAME" \
        -t "$table_name" --clean --if-exists \
        "$TEMP_DIR/$table_name.dump"
done

# Cleanup
rm -rf "$TEMP_DIR"

echo "Restore completed successfully"
```

**Windows:**
```powershell
# restore-from-encrypted.ps1

$EncryptedBackup = "C:\backups\daily\tariff_backup_daily_20240115.zip.enc"
$TempDir = "C:\temp\restore_$(Get-Date -Format 'yyyyMMddHHmmss')"
$DBName = "sms_gateway"

# Create temp directory
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

# Decrypt backup
Write-Host "Decrypting backup..."
.\scripts\encrypt-backup.ps1 -Command decrypt `
    -InputFile $EncryptedBackup `
    -OutputFile "$TempDir\backup.zip"

# Extract archive
Write-Host "Extracting archive..."
Expand-Archive -Path "$TempDir\backup.zip" -DestinationPath $TempDir

# Restore each table
Get-ChildItem -Path $TempDir -Filter "*.dump.enc" | ForEach-Object {
    $tableName = $_.BaseName -replace '\.dump$', ''
    
    Write-Host "Decrypting $tableName..."
    .\scripts\encrypt-backup.ps1 -Command decrypt `
        -InputFile $_.FullName `
        -OutputFile "$TempDir\$tableName.dump"
    
    Write-Host "Restoring $tableName..."
    & pg_restore -h localhost -U postgres -d $DBName `
        -t $tableName --clean --if-exists `
        "$TempDir\$tableName.dump"
}

# Cleanup
Remove-Item -Path $TempDir -Recurse -Force

Write-Host "Restore completed successfully"
```

---

## Security Considerations

### Key Security

**DO:**
- ✅ Store keys on encrypted filesystems
- ✅ Use restrictive file permissions (400 on Linux)
- ✅ Backup keys to secure offsite location
- ✅ Rotate keys regularly (every 90 days)
- ✅ Use hardware security modules (HSM) for production
- ✅ Audit key access regularly
- ✅ Document key recovery procedures

**DON'T:**
- ❌ Store keys in version control (Git, SVN, etc.)
- ❌ Store keys in application code
- ❌ Email keys or send via insecure channels
- ❌ Store keys on shared network drives without encryption
- ❌ Use the same key for multiple environments
- ❌ Share keys between team members unnecessarily

### Encryption Best Practices

**1. Use Strong Keys:**
- Minimum 256-bit keys (32 bytes)
- Generated using cryptographically secure random number generator
- Never use predictable or weak keys

**2. Protect Keys in Transit:**
- Use secure channels (SSH, HTTPS, VPN)
- Encrypt keys when transferring
- Use key exchange protocols (Diffie-Hellman)

**3. Protect Keys at Rest:**
- Store on encrypted filesystems
- Use hardware security modules (HSM) for production
- Consider key management services (AWS KMS, Azure Key Vault)

**4. Monitor and Audit:**
- Log all key access
- Monitor for unauthorized access attempts
- Regular security audits
- Incident response procedures

### Compliance Considerations

**GDPR (General Data Protection Regulation):**
- Encryption is required for personal data
- Keys must be managed securely
- Data breach notification within 72 hours

**PCI DSS (Payment Card Industry Data Security Standard):**
- Strong cryptography required
- Key rotation every 90 days
- Separate keys for different environments

**HIPAA (Health Insurance Portability and Accountability Act):**
- Encryption required for protected health information
- Key management procedures documented
- Regular security assessments

### Disaster Recovery

**Key Recovery Procedures:**

1. **Identify the issue:**
   - Key lost or corrupted
   - Key compromised
   - System failure

2. **Restore from backup:**
   ```bash
   # Linux/macOS
   sudo ./scripts/key-management.sh restore /secure/backup/backup.key
   
   # Windows
   .\scripts\key-management.ps1 -Command restore -BackupFile C:\secure\backup\backup.key
   ```

3. **Verify restored key:**
   ```bash
   # Linux/macOS
   sudo ./scripts/key-management.sh verify
   
   # Windows
   .\scripts\key-management.ps1 -Command verify
   ```

4. **Test decryption:**
   ```bash
   # Test with a known encrypted backup
   ./scripts/encrypt-backup.sh test /path/to/test/backup.sql
   ```

5. **If key is compromised:**
   - Immediately rotate to new key
   - Re-encrypt all backups with new key
   - Audit access logs
   - Notify security team

---

## Troubleshooting

### Common Issues

#### 1. "Encryption key file not found"

**Cause:** Key file doesn't exist or path is incorrect

**Solution:**
```bash
# Check if key exists
ls -la /etc/tariff-system/keys/backup.key

# If missing, initialize key management
sudo ./scripts/key-management.sh init

# Or specify correct path
export ENCRYPTION_KEY_FILE=/path/to/correct/key
```

#### 2. "Failed to decrypt: bad decrypt"

**Cause:** Wrong key or corrupted file

**Solution:**
```bash
# Try with previous key versions
sudo ./scripts/key-management.sh list-backups

# Decrypt with specific version
sudo ./scripts/encrypt-backup.sh decrypt \
  backup.sql.enc \
  backup.sql \
  /etc/tariff-system/keys/backups/backup.key.v1.20240115_103000

# Verify file integrity
sudo ./scripts/encrypt-backup.sh verify backup.sql.enc
```

#### 3. "Checksum mismatch"

**Cause:** File corrupted during transfer or storage

**Solution:**
```bash
# Check file integrity
sha256sum backup.sql.enc

# Compare with metadata
cat backup.sql.enc.meta | grep encrypted_checksum

# If corrupted, restore from another backup location
# (S3, offsite backup, etc.)
```

#### 4. "Permission denied"

**Cause:** Insufficient permissions to access key file

**Solution:**
```bash
# Check current permissions
ls -la /etc/tariff-system/keys/backup.key

# Should be: -r-------- (400)

# Fix permissions if needed
sudo chmod 400 /etc/tariff-system/keys/backup.key
sudo chown root:root /etc/tariff-system/keys/backup.key

# Run with sudo
sudo ./scripts/encrypt-backup.sh decrypt ...
```

#### 5. "OpenSSL not found"

**Cause:** OpenSSL not installed or not in PATH

**Solution:**
```bash
# Linux (Ubuntu/Debian)
sudo apt-get install openssl

# Linux (CentOS/RHEL)
sudo yum install openssl

# macOS
brew install openssl

# Windows
# Download from: https://slproweb.com/products/Win32OpenSSL.html
# Or use: choco install openssl
```

### Testing Encryption

**Round-trip Test:**
```bash
# Create test file
echo "Test data" > test.txt

# Test encryption/decryption
./scripts/encrypt-backup.sh test test.txt

# Output:
# [INFO] Testing encryption round-trip for: test.txt
# [INFO] Encrypting: test.txt
# [SUCCESS] Successfully encrypted: test.txt.enc
# [INFO] Verifying encryption: test.txt.enc
# [SUCCESS] Encryption verified successfully
# [INFO] Decrypting: test.txt.enc
# [SUCCESS] Successfully decrypted: test.txt.dec
# [SUCCESS] Round-trip test passed: checksums match
```

### Performance Tuning

**Adjust PBKDF2 Iterations:**

Higher iterations = More secure but slower
Lower iterations = Faster but less secure

```bash
# Default: 100,000 iterations
export PBKDF2_ITERATIONS=100000

# For faster encryption (less secure):
export PBKDF2_ITERATIONS=50000

# For maximum security (slower):
export PBKDF2_ITERATIONS=200000
```

**Benchmark:**
```bash
# Time encryption
time ./scripts/encrypt-backup.sh encrypt large_file.sql large_file.sql.enc

# Typical results:
# 100,000 iterations: ~2 seconds for 100MB file
# 200,000 iterations: ~4 seconds for 100MB file
```

---

## Additional Resources

**OpenSSL Documentation:**
- https://www.openssl.org/docs/

**NIST Guidelines:**
- https://csrc.nist.gov/publications/detail/sp/800-132/final

**Key Management Best Practices:**
- https://www.nist.gov/publications/recommendation-key-management

**Encryption Standards:**
- AES: https://nvlpubs.nist.gov/nistpubs/FIPS/NIST.FIPS.197.pdf
- PBKDF2: https://tools.ietf.org/html/rfc2898

---

## Support

For issues or questions:
- Check troubleshooting section above
- Review logs: `/var/log/key-management.log`
- Contact system administrator
- Refer to disaster recovery procedures

---

**Document Version:** 1.0  
**Last Updated:** 2024-01-15  
**Requirements:** 25.6
