# Backup Retention Policy - Quick Start Guide

## Overview

The retention policy automatically manages backup lifecycle according to these rules:
- **Daily backups**: Keep for 30 days
- **Weekly backups**: Keep for 3 months (90 days)
- **Monthly backups**: Keep for 1 year (365 days)

## Quick Start

### Linux/Mac

```bash
# Run retention policy
./scripts/backup-retention-policy.sh

# With custom backup directory
BACKUP_DIR=/path/to/backups ./scripts/backup-retention-policy.sh

# Dry run (see what would be deleted without actually deleting)
# Edit script to add dry-run support or review logs first
```

### Windows (PowerShell)

```powershell
# Run retention policy
.\scripts\backup-retention-policy.ps1

# With custom backup directory
.\scripts\backup-retention-policy.ps1 -BackupBaseDir "C:\backups"

# Dry run (see what would be deleted)
.\scripts\backup-retention-policy.ps1 -DryRun

# Verbose output
.\scripts\backup-retention-policy.ps1 -Verbose

# Combination
.\scripts\backup-retention-policy.ps1 -DryRun -Verbose
```

## Automated Scheduling

### Linux/Mac (Cron)

Add to crontab:

```bash
# Run retention policy daily at 3 AM
0 3 * * * /path/to/scripts/backup-retention-policy.sh >> /var/log/retention-policy.log 2>&1
```

### Windows (Task Scheduler)

```powershell
# Create scheduled task to run daily at 3 AM
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
    -Argument "-File C:\path\to\scripts\backup-retention-policy.ps1"

$trigger = New-ScheduledTaskTrigger -Daily -At 3am

$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" `
    -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask -TaskName "BackupRetentionPolicy" `
    -Action $action -Trigger $trigger -Principal $principal `
    -Description "Apply backup retention policy for tariff system"
```

## What Gets Deleted

### Daily Backups (30 days)
- Files older than 30 days in `backups/daily/`
- Example: On Jan 31, files from Dec 31 and earlier are deleted

### Weekly Backups (90 days)
- Files older than 90 days in `backups/weekly/`
- Example: On Jan 31, files from Nov 2 and earlier are deleted

### Monthly Backups (365 days)
- Files older than 365 days in `backups/monthly/`
- Example: On Jan 31 2024, files from Jan 31 2023 and earlier are deleted

## Reports

After each run, a retention report is generated:
- Location: `backups/retention-report-YYYYMMDD-HHMMSS.txt`
- Contains: Backup counts, sizes, and file listings
- Useful for: Auditing and capacity planning

## Logs

All operations are logged to:
- Linux/Mac: `logs/retention-policy.log`
- Windows: `logs/retention-policy.log`

Log format:
```
[2024-01-15 03:00:01] [INFO] Starting Backup Retention Policy
[2024-01-15 03:00:02] [INFO] Applying Daily retention policy: 30 days
[2024-01-15 03:00:03] [SUCCESS] Deleted: tariff-backup-20231201-030000.sql.gz
[2024-01-15 03:00:05] [SUCCESS] Daily retention complete: Deleted 5 backups (250MB freed)
```

## Safety Features

1. **Verification before deletion**: Files are checked for integrity
2. **Detailed logging**: Every action is logged with timestamp
3. **Dry run mode** (PowerShell): Test without deleting
4. **Reports**: Generated after each run for audit trail
5. **Empty directory cleanup**: Removes empty folders automatically

## Monitoring

Check retention policy health:

```bash
# View recent logs
tail -n 50 logs/retention-policy.log

# Check backup counts
ls -l backups/daily/ | wc -l
ls -l backups/weekly/ | wc -l
ls -l backups/monthly/ | wc -l

# Check disk usage
du -sh backups/daily/
du -sh backups/weekly/
du -sh backups/monthly/
```

## Troubleshooting

### Issue: Too many backups being deleted

**Solution**: Adjust retention periods in the script:
```bash
# Edit retention periods
DAILY_RETENTION=30    # Increase to keep longer
WEEKLY_RETENTION=90
MONTHLY_RETENTION=365
```

### Issue: Not enough space freed

**Solution**: 
1. Check if backups are being created in correct directories
2. Verify backup rotation is working (daily → weekly → monthly)
3. Consider reducing retention periods

### Issue: Script fails with permission error

**Solution**:
```bash
# Linux/Mac: Make script executable
chmod +x scripts/backup-retention-policy.sh

# Check directory permissions
ls -la backups/
```

### Issue: Corrupted backups not detected

**Solution**: The script verifies gzip integrity. For additional checks:
```bash
# Manually test backup
gzip -t backups/daily/tariff-backup-20240115-030000.sql.gz

# Test restore
gunzip -c backups/daily/tariff-backup-20240115-030000.sql.gz | head -n 10
```

## Best Practices

1. **Run daily**: Schedule retention policy to run daily after backups complete
2. **Monitor logs**: Review logs weekly for any issues
3. **Test restores**: Periodically test restoring from old backups before they're deleted
4. **Adjust as needed**: Modify retention periods based on compliance requirements
5. **Archive important backups**: Move critical backups to long-term storage before deletion

## Integration with Backup System

The retention policy works with the backup system:

```bash
# Typical daily workflow
1. 02:00 AM - Daily backup runs (backup-tariff-data.sh)
2. 03:00 AM - Retention policy runs (backup-retention-policy.sh)
3. 04:00 AM - Backup validation runs (test-backup-validation.sh)
```

## Compliance

This retention policy satisfies:
- **Requirement 25.2**: Daily backups kept for 30 days
- **Requirement 25.3**: Weekly backups kept for 3 months
- **Requirement 25.4**: Monthly backups kept for 1 year

## Next Steps

1. ✅ Run retention policy manually to test
2. ✅ Review generated report
3. ✅ Schedule automated execution
4. ✅ Monitor logs for first week
5. ✅ Adjust retention periods if needed

## Related Documentation

- [Backup System Guide](./BACKUP_RECOVERY_QUICK_START.md)
- [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)
- [Task 37 Summary](../.kiro/specs/tariff-pricing-system/TASK_37_BACKUP_RECOVERY_SUMMARY.md)
