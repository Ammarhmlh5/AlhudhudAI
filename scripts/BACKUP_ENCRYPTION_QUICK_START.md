# Backup Encryption Quick Start Guide

Quick reference for encrypting and decrypting backups in the Tariff Pricing System.

## Prerequisites

- OpenSSL installed
- Appropriate permissions (sudo/admin)
- Encryption key initialized

---

## Initial Setup

### Linux/macOS

```bash
# Initialize key management (first time only)
sudo ./scripts/key-management.sh init

# Verify key was created
ls -la /etc/tariff-system/keys/backup.key
```

### Windows

```powershell
# Initialize key management (first time only)
.\scripts\key-management.ps1 -Command init

# Verify key was created
Get-Item C:\tariff-system\keys\backup.key
```

---

## Common Operations

### Encrypt a File

**Linux/macOS:**
```bash
sudo ./scripts/encrypt-backup.sh encrypt \
  /backups/backup.sql \
  /backups/backup.sql.enc
```

**Windows:**
```powershell
.\scripts\encrypt-backup.ps1 -Command encrypt `
  -InputFile C:\backups\backup.sql `
  -OutputFile C:\backups\backup.sql.enc
```

### Decrypt a File

**Linux/macOS:**
```bash
sudo ./scripts/encrypt-backup.sh decrypt \
  /backups/backup.sql.enc \
  /restore/backup.sql
```

**Windows:**
```powershell
.\scripts\encrypt-backup.ps1 -Command decrypt `
  -InputFile C:\backups\backup.sql.enc `
  -OutputFile C:\restore\backup.sql
```

### Verify Encryption

**Linux/macOS:**
```bash
sudo ./scripts/encrypt-backup.sh verify /backups/backup.sql.enc
```

**Windows:**
```powershell
.\scripts\encrypt-backup.ps1 -Command verify `
  -InputFile C:\backups\backup.sql.enc
```

### Test Round-trip

**Linux/macOS:**
```bash
sudo ./scripts/encrypt-backup.sh test /backups/backup.sql
```

**Windows:**
```powershell
.\scripts\encrypt-backup.ps1 -Command test `
  -InputFile C:\backups\backup.sql
```

---

## Key Management

### Rotate Key (Every 90 Days)

**Linux/macOS:**
```bash
# Check if rotation needed
sudo ./scripts/key-management.sh check-rotation

# Perform rotation
sudo ./scripts/key-management.sh rotate

# Verify new key
sudo ./scripts/key-management.sh verify
```

**Windows:**
```powershell
# Check if rotation needed
.\scripts\key-management.ps1 -Command check-rotation

# Perform rotation
.\scripts\key-management.ps1 -Command rotate

# Verify new key
.\scripts\key-management.ps1 -Command verify
```

### Backup Key

**Linux/macOS:**
```bash
sudo ./scripts/key-management.sh backup /secure/location/backup.key
```

**Windows:**
```powershell
.\scripts\key-management.ps1 -Command backup `
  -Destination C:\secure\location\backup.key
```

### List Key Backups

**Linux/macOS:**
```bash
sudo ./scripts/key-management.sh list-backups
```

**Windows:**
```powershell
.\scripts\key-management.ps1 -Command list-backups
```

### Restore Key

**Linux/macOS:**
```bash
sudo ./scripts/key-management.sh restore /secure/location/backup.key
```

**Windows:**
```powershell
.\scripts\key-management.ps1 -Command restore `
  -BackupFile C:\secure\location\backup.key
```

---

## Integration with Backup Scripts

The backup scripts (`backup-tariff-data.sh` and `backup-tariff-data.ps1`) automatically encrypt backups using these utilities.

### Environment Variables

```bash
# Linux/macOS
export ENCRYPTION_KEY_FILE="/etc/tariff-system/keys/backup.key"
export ENCRYPTION_ALGORITHM="aes-256-cbc"
export PBKDF2_ITERATIONS=100000

# Windows (PowerShell)
$env:ENCRYPTION_KEY_FILE = "C:\tariff-system\keys\backup.key"
```

### Manual Backup with Encryption

**Linux/macOS:**
```bash
# Run backup script (encryption is automatic)
sudo ./scripts/backup-tariff-data.sh

# Backups are stored encrypted in:
# /var/backups/tariff-system/daily/
# /var/backups/tariff-system/weekly/
# /var/backups/tariff-system/monthly/
```

**Windows:**
```powershell
# Run backup script (encryption is automatic)
.\scripts\backup-tariff-data.ps1

# Backups are stored encrypted in:
# C:\backups\tariff-system\daily\
# C:\backups\tariff-system\weekly\
# C:\backups\tariff-system\monthly\
```

---

## Troubleshooting

### Key Not Found

```bash
# Linux/macOS
sudo ./scripts/key-management.sh init

# Windows
.\scripts\key-management.ps1 -Command init
```

### Decryption Failed

```bash
# Try with previous key version
sudo ./scripts/key-management.sh list-backups

# Decrypt with specific key
sudo ./scripts/encrypt-backup.sh decrypt \
  backup.sql.enc \
  backup.sql \
  /etc/tariff-system/keys/backups/backup.key.v1.20240115_103000
```

### Permission Denied

```bash
# Linux/macOS
sudo chmod 400 /etc/tariff-system/keys/backup.key
sudo chown root:root /etc/tariff-system/keys/backup.key

# Run with sudo
sudo ./scripts/encrypt-backup.sh ...
```

---

## Security Checklist

- [ ] Encryption key initialized
- [ ] Key permissions set to 400 (Linux) or restricted ACL (Windows)
- [ ] Key backed up to secure offsite location
- [ ] Key rotation scheduled (every 90 days)
- [ ] Test restore procedure documented
- [ ] Disaster recovery plan in place

---

## Quick Reference

| Operation | Linux/macOS | Windows |
|-----------|-------------|---------|
| Initialize | `./key-management.sh init` | `.\key-management.ps1 -Command init` |
| Encrypt | `./encrypt-backup.sh encrypt <in> <out>` | `.\encrypt-backup.ps1 -Command encrypt -InputFile <in> -OutputFile <out>` |
| Decrypt | `./encrypt-backup.sh decrypt <in> <out>` | `.\encrypt-backup.ps1 -Command decrypt -InputFile <in> -OutputFile <out>` |
| Verify | `./encrypt-backup.sh verify <file>` | `.\encrypt-backup.ps1 -Command verify -InputFile <file>` |
| Rotate Key | `./key-management.sh rotate` | `.\key-management.ps1 -Command rotate` |
| Backup Key | `./key-management.sh backup <dest>` | `.\key-management.ps1 -Command backup -Destination <dest>` |

---

## Encryption Details

- **Algorithm:** AES-256-CBC
- **Key Derivation:** PBKDF2 with SHA-256
- **Iterations:** 100,000
- **Key Size:** 256 bits (32 bytes)
- **Salt:** Randomly generated per encryption

---

## Support

For detailed documentation, see: `BACKUP_ENCRYPTION_GUIDE.md`

**Requirements:** 25.6
