# Backup & Recovery Quick Start Guide

## Overview

This guide provides quick instructions for backing up and restoring the Tariff Pricing System data.

## Prerequisites

### Linux/macOS
- PostgreSQL client tools (`pg_dump`, `pg_restore`, `psql`)
- OpenSSL
- AWS CLI (optional, for S3 storage)
- Bash shell

### Windows
- PostgreSQL client tools
- OpenSSL for Windows
- AWS CLI (optional)
- PowerShell 5.1 or later

## Quick Start

### 1. Setup Encryption Key

**Linux/macOS:**
```bash
# Create encryption key directory
sudo mkdir -p /etc/tariff-system

# Generate encryption key
openssl rand -base64 32 | sudo tee /etc/tariff-system/backup.key

# Secure the key
sudo chmod 600 /etc/tariff-system/backup.key
```

**Windows:**
```powershell
# Create encryption key directory
New-Item -ItemType Directory -Path "C:\tariff-system" -Force

# Generate encryption key
openssl rand -base64 32 | Out-File -FilePath "C:\tariff-system\backup.key" -Encoding ASCII

# Secure the key (restrict access)
$acl = Get-Acl "C:\tariff-system\backup.key"
$acl.SetAccessRuleProtection($true, $false)
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
    $env:USERNAME, "FullControl", "Allow"
)
$acl.AddAccessRule($rule)
Set-Acl "C:\tariff-system\backup.key" $acl
```

### 2. Configure Environment Variables

**Linux/macOS:**
```bash
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_NAME="sms_gateway"
export DB_USER="postgres"
export DB_PASSWORD="your_password"
export BACKUP_DIR="/var/backups/tariff-system"
export S3_BUCKET="your-backup-bucket"  # Optional
```

**Windows:**
```powershell
$env:DB_HOST = "localhost"
$env:DB_PORT = "5432"
$env:DB_NAME = "sms_gateway"
$env:DB_USER = "postgres"
$env:DB_PASSWORD = "your_password"
$env:BACKUP_DIR = "C:\backups\tariff-system"
$env:S3_BUCKET = "your-backup-bucket"  # Optional
```

### 3. Run Backup

**Linux/macOS:**
```bash
# Make script executable
chmod +x scripts/backup-tariff-data.sh

# Run backup
bash scripts/backup-tariff-data.sh
```

**Windows:**
```powershell
# Run backup
.\scripts\backup-tariff-data.ps1
```

### 4. Verify Backup

**Linux/macOS:**
```bash
# Make script executable
chmod +x scripts/test-backup-validation.sh

# Run validation
bash scripts/test-backup-validation.sh
```

**Windows:**
```powershell
# List backups
Get-ChildItem -Path "$env:BACKUP_DIR\daily" -Filter "*.zip" | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 5
```

### 5. Restore from Backup

**Linux/macOS:**
```bash
# List available backups
bash scripts/restore-tariff-data.sh --list

# Restore from local backup
bash scripts/restore-tariff-data.sh \
    /var/backups/tariff-system/daily/tariff_backup_daily_20240101_120000.tar.gz

# Restore from S3
bash scripts/restore-tariff-data.sh \
    --s3 backups/daily/2024-01-01/tariff_backup_daily_20240101_120000.tar.gz

# Restore specific tables only
bash scripts/restore-tariff-data.sh \
    --tables tariffs,packages \
    /var/backups/tariff-system/daily/backup.tar.gz

# Drop existing tables before restore
bash scripts/restore-tariff-data.sh \
    --drop \
    /var/backups/tariff-system/daily/backup.tar.gz
```

**Windows:**
```powershell
# List available backups
Get-ChildItem -Path "$env:BACKUP_DIR\daily" -Filter "*.zip"

# Restore (manual process on Windows)
# 1. Extract the backup archive
Expand-Archive -Path "C:\backups\tariff-system\daily\backup.zip" -DestinationPath "C:\temp\restore"

# 2. Decrypt each file
Get-ChildItem "C:\temp\restore\*.enc" | ForEach-Object {
    $outputFile = $_.FullName -replace '\.enc$', ''
    openssl enc -aes-256-cbc -d -pbkdf2 -in $_.FullName -out $outputFile -pass "file:C:\tariff-system\backup.key"
}

# 3. Restore each table
Get-ChildItem "C:\temp\restore\*.dump" | ForEach-Object {
    pg_restore -h localhost -U postgres -d sms_gateway --no-owner --no-acl $_.FullName
}
```

## Automated Backups

### Linux/macOS (Cron)

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /path/to/scripts/backup-tariff-data.sh >> /var/log/tariff-backup.log 2>&1

# Add weekly validation on Sundays at 3 AM
0 3 * * 0 /path/to/scripts/test-backup-validation.sh >> /var/log/tariff-validation.log 2>&1
```

### Windows (Task Scheduler)

```powershell
# Create scheduled task for daily backup
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
    -Argument "-File C:\path\to\scripts\backup-tariff-data.ps1"

$trigger = New-ScheduledTaskTrigger -Daily -At 2am

$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount

Register-ScheduledTask -TaskName "TariffSystemBackup" `
    -Action $action `
    -Trigger $trigger `
    -Principal $principal `
    -Description "Daily backup of Tariff System data"
```

## Backup Types and Retention

| Type | Frequency | Retention | Storage Location |
|------|-----------|-----------|------------------|
| Daily | Every day at 2 AM | 30 days | `$BACKUP_DIR/daily/` |
| Weekly | Every Sunday | 12 weeks (84 days) | `$BACKUP_DIR/weekly/` |
| Monthly | 1st of each month | 12 months (365 days) | `$BACKUP_DIR/monthly/` |

## S3 Storage (Optional)

### Configure AWS CLI

```bash
# Install AWS CLI
# Linux: sudo apt-get install awscli
# macOS: brew install awscli
# Windows: Download from https://aws.amazon.com/cli/

# Configure credentials
aws configure
```

### Enable S3 Backup

```bash
# Set S3 bucket
export S3_BUCKET="your-backup-bucket"

# Run backup (will automatically upload to S3)
bash scripts/backup-tariff-data.sh
```

### S3 Lifecycle Policy

Create an S3 lifecycle policy to automatically transition old backups to cheaper storage:

```json
{
  "Rules": [
    {
      "Id": "TransitionOldBackups",
      "Status": "Enabled",
      "Transitions": [
        {
          "Days": 30,
          "StorageClass": "STANDARD_IA"
        },
        {
          "Days": 90,
          "StorageClass": "GLACIER"
        }
      ],
      "Expiration": {
        "Days": 365
      }
    }
  ]
}
```

## Monitoring and Alerts

### Check Backup Status

**Linux/macOS:**
```bash
# Check last backup
ls -lh /var/backups/tariff-system/daily/ | tail -n 5

# Check backup log
tail -f /var/backups/tariff-system/backup_*.log
```

**Windows:**
```powershell
# Check last backup
Get-ChildItem -Path "C:\backups\tariff-system\daily" | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 5

# Check backup log
Get-Content "C:\backups\tariff-system\backup_*.log" -Tail 50
```

### Email Alerts

Configure email alerts for backup failures:

**Linux/macOS:**
```bash
# Install mail utility
sudo apt-get install mailutils  # Ubuntu/Debian
sudo yum install mailx          # CentOS/RHEL

# Set alert email
export ALERT_EMAIL="admin@example.com"
```

**Windows:**
```powershell
# Configure SMTP settings in backup script
# Or use Send-MailMessage cmdlet
```

## Disaster Recovery

For complete disaster recovery procedures, see:
- [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)

### Quick Recovery Steps

1. **Assess the situation**
   - Identify what failed
   - Determine recovery scope
   - Notify stakeholders

2. **Stop application traffic**
   ```bash
   systemctl stop tariff-api
   ```

3. **Restore database**
   ```bash
   bash scripts/restore-tariff-data.sh --drop /path/to/backup.tar.gz
   ```

4. **Verify data integrity**
   ```bash
   psql -h localhost -U postgres -d sms_gateway -f scripts/verify-data-integrity.sql
   ```

5. **Resume application**
   ```bash
   systemctl start tariff-api
   ```

## Troubleshooting

### Backup Fails

**Problem**: `pg_dump: error: connection to database failed`

**Solution**:
```bash
# Check database connectivity
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT version();"

# Check PostgreSQL service
systemctl status postgresql
```

### Encryption Fails

**Problem**: `Encryption key file not found`

**Solution**:
```bash
# Verify key exists
ls -l /etc/tariff-system/backup.key

# Regenerate if missing
openssl rand -base64 32 | sudo tee /etc/tariff-system/backup.key
sudo chmod 600 /etc/tariff-system/backup.key
```

### Restore Fails

**Problem**: `Failed to decrypt backup file`

**Solution**:
```bash
# Verify encryption key is correct
# Try manual decryption
openssl enc -aes-256-cbc -d -pbkdf2 \
    -in backup.dump.enc \
    -out backup.dump \
    -pass file:/etc/tariff-system/backup.key
```

### Disk Space Issues

**Problem**: `No space left on device`

**Solution**:
```bash
# Check disk usage
df -h

# Clean up old backups manually
find /var/backups/tariff-system/daily -name "*.tar.gz" -mtime +30 -delete

# Move backups to S3
aws s3 sync /var/backups/tariff-system/ s3://your-bucket/backups/
```

## Best Practices

1. **Test restores regularly** - Run monthly restore tests to verify backups work
2. **Monitor backup size** - Sudden changes may indicate issues
3. **Secure encryption keys** - Store keys separately from backups
4. **Use S3 versioning** - Enable versioning on S3 bucket for extra protection
5. **Document procedures** - Keep recovery procedures up to date
6. **Automate validation** - Run automated backup validation daily
7. **Off-site storage** - Always maintain off-site backups (S3)
8. **Encrypt in transit** - Use SSL/TLS for S3 uploads
9. **Access control** - Restrict backup access to authorized personnel only
10. **Audit logs** - Review backup logs regularly

## Support

For issues or questions:
- Check logs: `$BACKUP_DIR/backup_*.log`
- Review [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)
- Contact: ops-team@example.com

## Related Documentation

- [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)
- [Database Schema](../docs/DATABASE_SCHEMA_TARIFF.md)
- [Operations Manual](../docs/OPERATIONS_MANUAL.md)
