# Backup Retention Policy - Complete Guide

## Table of Contents

1. [Overview](#overview)
2. [Retention Rules](#retention-rules)
3. [Implementation Details](#implementation-details)
4. [Usage](#usage)
5. [Automation](#automation)
6. [Monitoring](#monitoring)
7. [Compliance](#compliance)
8. [Troubleshooting](#troubleshooting)

## Overview

The backup retention policy automatically manages the lifecycle of tariff system backups according to predefined rules. This ensures:

- **Compliance**: Meet regulatory requirements for data retention
- **Cost optimization**: Free up storage space by removing old backups
- **Risk management**: Maintain sufficient backup history for recovery
- **Automation**: Reduce manual intervention and human error

### Key Features

- ✅ Automatic deletion of old backups
- ✅ Separate retention periods for daily, weekly, and monthly backups
- ✅ Detailed logging and reporting
- ✅ Dry-run mode for testing
- ✅ Integrity verification before deletion
- ✅ Empty directory cleanup
- ✅ Cross-platform support (Linux/Mac/Windows)

## Retention Rules

### Daily Backups (30 Days)

**Purpose**: Short-term recovery for recent issues

- **Retention period**: 30 days
- **Storage location**: `backups/daily/`
- **Typical size**: ~500MB per backup
- **Expected count**: ~30 backups
- **Use case**: Recover from issues within the last month

**Example Timeline**:
```
Today: Jan 31, 2024
├── Keep: Jan 1-31, 2024 (31 backups)
└── Delete: Dec 31, 2023 and earlier
```

### Weekly Backups (90 Days / 3 Months)

**Purpose**: Medium-term recovery and trend analysis

- **Retention period**: 90 days (3 months)
- **Storage location**: `backups/weekly/`
- **Typical size**: ~500MB per backup
- **Expected count**: ~12-13 backups
- **Use case**: Recover from issues discovered after monthly cycles

**Example Timeline**:
```
Today: Jan 31, 2024
├── Keep: Nov 2, 2023 - Jan 31, 2024 (~13 backups)
└── Delete: Nov 1, 2023 and earlier
```

### Monthly Backups (365 Days / 1 Year)

**Purpose**: Long-term recovery and compliance

- **Retention period**: 365 days (1 year)
- **Storage location**: `backups/monthly/`
- **Typical size**: ~500MB per backup
- **Expected count**: ~12 backups
- **Use case**: Annual audits, compliance, long-term recovery

**Example Timeline**:
```
Today: Jan 31, 2024
├── Keep: Feb 1, 2023 - Jan 31, 2024 (12 backups)
└── Delete: Jan 31, 2023 and earlier
```

## Implementation Details

### Algorithm

The retention policy uses a simple age-based algorithm:

```
For each backup file:
  1. Calculate file age = Current Date - File Last Modified Date
  2. If age > retention period:
     a. Verify file integrity
     b. Log deletion intent
     c. Delete file
     d. Update statistics
  3. Else:
     Keep file
```

### File Selection

Files are selected for deletion based on:

1. **Location**: Must be in the correct directory (daily/weekly/monthly)
2. **Extension**: Must be `.sql.gz`, `.tar.gz`, or similar backup format
3. **Age**: Must be older than the retention period
4. **Integrity**: Must pass integrity checks (if applicable)

### Safety Mechanisms

1. **Integrity Verification**: Files are checked before deletion
   - Gzip files: Tested with `gzip -t` (Linux) or 7-Zip (Windows)
   - Empty files: Rejected
   - Corrupted files: Logged and skipped

2. **Detailed Logging**: Every action is logged
   - Timestamp for each operation
   - File name, age, and size
   - Success/failure status
   - Error messages

3. **Dry Run Mode** (PowerShell only): Test without deleting
   - Shows what would be deleted
   - Useful for validation before actual execution

4. **Reports**: Generated after each run
   - Backup counts and sizes
   - Files kept and deleted
   - Storage space freed

## Usage

### Linux/Mac

#### Basic Usage

```bash
# Run retention policy
./scripts/backup-retention-policy.sh

# With custom backup directory
BACKUP_DIR=/path/to/backups ./scripts/backup-retention-policy.sh

# With custom log directory
LOG_DIR=/var/log ./scripts/backup-retention-policy.sh
```

#### Environment Variables

```bash
# Set backup directory
export BACKUP_DIR=/mnt/backups

# Set log directory
export LOG_DIR=/var/log/backups

# Run with custom settings
./scripts/backup-retention-policy.sh
```

### Windows (PowerShell)

#### Basic Usage

```powershell
# Run retention policy
.\scripts\backup-retention-policy.ps1

# With custom backup directory
.\scripts\backup-retention-policy.ps1 -BackupBaseDir "C:\backups"

# With custom log directory
.\scripts\backup-retention-policy.ps1 -LogDir "C:\logs"
```

#### Advanced Options

```powershell
# Dry run (see what would be deleted)
.\scripts\backup-retention-policy.ps1 -DryRun

# Verbose output
.\scripts\backup-retention-policy.ps1 -Verbose

# Combination
.\scripts\backup-retention-policy.ps1 -BackupBaseDir "C:\backups" -DryRun -Verbose
```

### Testing

Before running in production, test with sample data:

```powershell
# Create test backups
.\scripts\test-retention-policy.ps1

# Run retention policy in dry-run mode
.\scripts\backup-retention-policy.ps1 -BackupBaseDir "./test-backups" -DryRun

# Verify results
.\scripts\backup-retention-policy.ps1 -BackupBaseDir "./test-backups"

# Cleanup test data
.\scripts\test-retention-policy.ps1 -CleanupAfter
```

## Automation

### Linux/Mac (Cron)

#### Daily Execution

Add to crontab (`crontab -e`):

```bash
# Run retention policy daily at 3 AM
0 3 * * * /path/to/scripts/backup-retention-policy.sh >> /var/log/retention-policy.log 2>&1

# With email notifications on error
0 3 * * * /path/to/scripts/backup-retention-policy.sh || mail -s "Retention Policy Failed" admin@example.com
```

#### Weekly Execution

```bash
# Run retention policy weekly on Sunday at 3 AM
0 3 * * 0 /path/to/scripts/backup-retention-policy.sh >> /var/log/retention-policy.log 2>&1
```

### Windows (Task Scheduler)

#### Create Scheduled Task

```powershell
# Define task action
$action = New-ScheduledTaskAction `
    -Execute "PowerShell.exe" `
    -Argument "-ExecutionPolicy Bypass -File C:\path\to\scripts\backup-retention-policy.ps1"

# Define trigger (daily at 3 AM)
$trigger = New-ScheduledTaskTrigger -Daily -At 3am

# Define principal (run as SYSTEM)
$principal = New-ScheduledTaskPrincipal `
    -UserId "SYSTEM" `
    -LogonType ServiceAccount `
    -RunLevel Highest

# Register task
Register-ScheduledTask `
    -TaskName "BackupRetentionPolicy" `
    -Action $action `
    -Trigger $trigger `
    -Principal $principal `
    -Description "Apply backup retention policy for tariff system"
```

#### Verify Task

```powershell
# List scheduled tasks
Get-ScheduledTask -TaskName "BackupRetentionPolicy"

# View task details
Get-ScheduledTask -TaskName "BackupRetentionPolicy" | Get-ScheduledTaskInfo

# Run task manually
Start-ScheduledTask -TaskName "BackupRetentionPolicy"
```

## Monitoring

### Log Files

Logs are stored in `logs/retention-policy.log`:

```
[2024-01-15 03:00:01] [INFO] Starting Backup Retention Policy
[2024-01-15 03:00:02] [INFO] Applying Daily retention policy: 30 days
[2024-01-15 03:00:03] [INFO] Deleting old backup (35 days old): tariff-backup-20231201-030000.sql.gz
[2024-01-15 03:00:04] [SUCCESS] Deleted: tariff-backup-20231201-030000.sql.gz
[2024-01-15 03:00:05] [SUCCESS] Daily retention complete: Deleted 5 backups (250MB freed)
```

### Monitoring Commands

```bash
# View recent logs
tail -n 50 logs/retention-policy.log

# Search for errors
grep ERROR logs/retention-policy.log

# Count deletions today
grep "$(date +%Y-%m-%d)" logs/retention-policy.log | grep "Deleted:" | wc -l

# Check backup counts
find backups/daily -type f | wc -l
find backups/weekly -type f | wc -l
find backups/monthly -type f | wc -l

# Check disk usage
du -sh backups/daily/
du -sh backups/weekly/
du -sh backups/monthly/
du -sh backups/
```

### Reports

Reports are generated after each run:

```
backups/retention-report-20240115-030000.txt
```

Report contents:
- Retention policy settings
- Backup counts and sizes
- File listings with ages
- Summary statistics

### Alerts

Set up alerts for:

1. **Retention policy failures**: Script exits with error
2. **Excessive deletions**: More than expected files deleted
3. **Low backup counts**: Fewer backups than expected
4. **Disk space issues**: Not enough space freed

Example alert script:

```bash
#!/bin/bash
# Check if retention policy ran successfully

LOG_FILE="logs/retention-policy.log"
ALERT_EMAIL="admin@example.com"

# Check if log file was updated today
if [ ! -f "$LOG_FILE" ]; then
    echo "Retention policy log not found" | mail -s "Alert: Retention Policy" $ALERT_EMAIL
    exit 1
fi

# Check for errors
if grep -q "ERROR" "$LOG_FILE"; then
    echo "Retention policy encountered errors" | mail -s "Alert: Retention Policy" $ALERT_EMAIL
    exit 1
fi

# Check if policy ran today
if ! grep -q "$(date +%Y-%m-%d)" "$LOG_FILE"; then
    echo "Retention policy did not run today" | mail -s "Alert: Retention Policy" $ALERT_EMAIL
    exit 1
fi

echo "Retention policy check passed"
```

## Compliance

### Regulatory Requirements

The retention policy satisfies:

- **Requirement 25.2**: Daily backups retained for 30 days
- **Requirement 25.3**: Weekly backups retained for 3 months
- **Requirement 25.4**: Monthly backups retained for 1 year

### Audit Trail

All retention operations are logged:

1. **What was deleted**: File name, age, size
2. **When it was deleted**: Timestamp
3. **Why it was deleted**: Age exceeded retention period
4. **Who deleted it**: System (automated) or user (manual)

### Compliance Reports

Generate compliance reports:

```bash
# Extract deletion summary for audit
grep "retention complete" logs/retention-policy.log > compliance-report.txt

# List all backups with ages
find backups/ -type f -printf "%p %TY-%Tm-%Td\n" > backup-inventory.txt
```

## Troubleshooting

### Issue: Too Many Backups Deleted

**Symptoms**: More backups deleted than expected

**Diagnosis**:
```bash
# Check retention periods in script
grep "RETENTION=" scripts/backup-retention-policy.sh

# Verify backup ages
find backups/daily -type f -printf "%p %TY-%Tm-%Td\n"
```

**Solution**:
1. Increase retention periods in script
2. Verify backup rotation is working correctly
3. Check if backups are being created in correct directories

### Issue: Not Enough Space Freed

**Symptoms**: Disk usage still high after retention policy runs

**Diagnosis**:
```bash
# Check backup sizes
du -sh backups/*/

# Count backups
find backups/ -type f | wc -l

# Check for large files
find backups/ -type f -size +100M
```

**Solution**:
1. Verify backups are being rotated (daily → weekly → monthly)
2. Check if old backups exist outside retention directories
3. Consider reducing retention periods
4. Compress backups more aggressively

### Issue: Script Fails with Permission Error

**Symptoms**: Script exits with permission denied error

**Diagnosis**:
```bash
# Check script permissions
ls -la scripts/backup-retention-policy.sh

# Check directory permissions
ls -la backups/
```

**Solution**:
```bash
# Make script executable
chmod +x scripts/backup-retention-policy.sh

# Fix directory permissions
chmod 755 backups/
chmod 755 backups/daily/ backups/weekly/ backups/monthly/
```

### Issue: Corrupted Backups Not Detected

**Symptoms**: Corrupted backups remain after retention policy runs

**Diagnosis**:
```bash
# Manually test backups
for file in backups/daily/*.gz; do
    echo "Testing: $file"
    gzip -t "$file" || echo "CORRUPTED: $file"
done
```

**Solution**:
1. Enhance integrity checking in script
2. Run separate validation script before retention policy
3. Manually remove corrupted backups

### Issue: Retention Policy Not Running Automatically

**Symptoms**: Scheduled task not executing

**Diagnosis**:
```bash
# Linux: Check cron logs
grep CRON /var/log/syslog

# Windows: Check task scheduler
Get-ScheduledTask -TaskName "BackupRetentionPolicy" | Get-ScheduledTaskInfo
```

**Solution**:
```bash
# Linux: Verify crontab entry
crontab -l

# Windows: Re-register task
Unregister-ScheduledTask -TaskName "BackupRetentionPolicy" -Confirm:$false
# Then re-create task
```

## Best Practices

1. **Test before production**: Use dry-run mode and test data
2. **Monitor regularly**: Review logs weekly
3. **Verify backups**: Test restores before old backups are deleted
4. **Adjust as needed**: Modify retention periods based on requirements
5. **Archive critical backups**: Move important backups to long-term storage
6. **Document changes**: Log any modifications to retention periods
7. **Automate monitoring**: Set up alerts for failures
8. **Regular audits**: Review retention reports monthly

## Related Documentation

- [Backup System Quick Start](./BACKUP_RECOVERY_QUICK_START.md)
- [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)
- [Task 37 Summary](../.kiro/specs/tariff-pricing-system/TASK_37_BACKUP_RECOVERY_SUMMARY.md)
- [Database Schema](../docs/DATABASE_SCHEMA_TARIFF.md)
