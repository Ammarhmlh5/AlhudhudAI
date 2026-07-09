# Recovery Procedures - Tariff Pricing System

## Document Information

- **Version**: 1.0
- **Requirements**: 25.8, 25.9, 25.11
- **Last Updated**: 2024
- **Review Cycle**: Monthly

## Table of Contents

1. [Overview](#overview)
2. [Recovery Scenarios](#recovery-scenarios)
3. [Step-by-Step Procedures](#step-by-step-procedures)
4. [Validation and Testing](#validation-and-testing)
5. [Periodic Testing Schedule](#periodic-testing-schedule)

---

## Overview

### Purpose

This document provides detailed recovery procedures for the Tariff Pricing System.
It covers various failure scenarios and step-by-step instructions for data recovery.

### Recovery Objectives

- **RTO (Recovery Time Objective)**: 1 hour for critical data
- **RPO (Recovery Point Objective)**: 24 hours maximum data loss
- **Success Rate**: 99% of recovery operations successful

### Prerequisites

- Access to backup files (local or S3)
- Database credentials
- Encryption key file
- PostgreSQL client tools
- Sufficient disk space for restore operations

---

## Recovery Scenarios

### Scenario 1: Single Table Corruption

**When to Use**: One or more tables are corrupted but database is accessible

**Impact**: Partial data loss, specific functionality affected

**Recovery Time**: 15-30 minutes


**Procedure**:

```bash
# 1. Identify corrupted tables
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\dt"

# 2. List available backups
bash scripts/restore-tariff-data.sh --list

# 3. Restore specific tables only
bash scripts/restore-tariff-data.sh \
    --tables tariffs,packages \
    --drop \
    /var/backups/tariff-system/daily/latest_backup.tar.gz

# 4. Verify restoration
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM tariffs;"
```

---

### Scenario 2: Complete Database Loss

**When to Use**: Entire database is lost or inaccessible

**Impact**: Complete system outage

**Recovery Time**: 45-60 minutes

**Procedure**:

```bash
# 1. Stop application services
systemctl stop tariff-api

# 2. Recreate database if needed
psql -h $DB_HOST -U postgres -c "DROP DATABASE IF EXISTS sms_gateway;"
psql -h $DB_HOST -U postgres -c "CREATE DATABASE sms_gateway;"

# 3. Restore from latest backup
bash scripts/restore-tariff-data.sh \
    --drop \
    /var/backups/tariff-system/daily/latest_backup.tar.gz

# 4. Verify all tables
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f scripts/verify-data-integrity.sql

# 5. Restart application
systemctl start tariff-api
```

---

### Scenario 3: Point-in-Time Recovery

**When to Use**: Need to restore to a specific point in time

**Impact**: Data loss from specific time forward

**Recovery Time**: 30-45 minutes


**Procedure**:

```bash
# 1. Identify the target restore point
# List backups by date
ls -lh /var/backups/tariff-system/daily/ | grep "20240115"

# 2. Stop application to prevent new writes
systemctl stop tariff-api

# 3. Restore from specific backup
bash scripts/restore-tariff-data.sh \
    --drop \
    /var/backups/tariff-system/daily/tariff_backup_daily_20240115_140000.tar.gz

# 4. Verify data timestamp
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c \
    "SELECT MAX(created_at) FROM accounting_transactions;"

# 5. Resume operations
systemctl start tariff-api
```

---

### Scenario 4: Disaster Recovery (Off-site Restore)

**When to Use**: Primary site is unavailable, restore from S3

**Impact**: Complete system outage

**Recovery Time**: 60-90 minutes

**Procedure**:

```bash
# 1. Configure AWS credentials
aws configure

# 2. List available S3 backups
aws s3 ls s3://your-backup-bucket/backups/daily/

# 3. Restore from S3
bash scripts/restore-tariff-data.sh \
    --s3 backups/daily/2024-01-15/tariff_backup_daily_20240115_140000.tar.gz \
    --drop

# 4. Verify restoration
bash scripts/test-backup-validation.sh

# 5. Update DNS/load balancer to new location
# (Manual step - update infrastructure)
```

---

## Step-by-Step Procedures

### Procedure A: Pre-Recovery Checklist

**Before starting any recovery operation**:


1. **Assess the Situation**
   - [ ] Identify what failed
   - [ ] Determine scope of data loss
   - [ ] Estimate recovery time
   - [ ] Notify stakeholders

2. **Verify Prerequisites**
   - [ ] Database credentials available
   - [ ] Encryption key accessible
   - [ ] Sufficient disk space
   - [ ] Backup files accessible

3. **Create Backup of Current State**
   ```bash
   # Even if corrupted, backup current state
   pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME \
       -F c -f /tmp/pre_recovery_backup_$(date +%s).dump
   ```

4. **Stop Application Traffic**
   ```bash
   # Prevent new writes during recovery
   systemctl stop tariff-api
   
   # Or set database to read-only
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
       -c "ALTER DATABASE sms_gateway SET default_transaction_read_only = on;"
   ```

5. **Document Recovery Start**
   ```bash
   echo "Recovery started at $(date)" >> /var/log/recovery.log
   echo "Backup file: $BACKUP_FILE" >> /var/log/recovery.log
   ```

---

### Procedure B: Restore Execution

**Execute the restore operation**:

1. **Verify Backup Integrity**
   ```bash
   # Test backup before restoring
   bash scripts/restore-tariff-data.sh --verify-only $BACKUP_FILE
   ```

2. **Execute Restore**
   ```bash
   # Full restore with drop
   bash scripts/restore-tariff-data.sh --drop $BACKUP_FILE
   
   # Or partial restore
   bash scripts/restore-tariff-data.sh --tables tariffs,packages $BACKUP_FILE
   ```

3. **Monitor Progress**
   ```bash
   # Watch restore log in real-time
   tail -f /var/backups/tariff-system/restore_*.log
   ```


---

### Procedure C: Post-Recovery Validation

**Verify the restore was successful**:

1. **Check Table Counts**
   ```bash
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<EOF
   SELECT 'tariffs' as table_name, COUNT(*) as row_count FROM tariffs
   UNION ALL
   SELECT 'packages', COUNT(*) FROM packages
   UNION ALL
   SELECT 'subscriptions', COUNT(*) FROM subscriptions
   UNION ALL
   SELECT 'dedicated_gateways', COUNT(*) FROM dedicated_gateways;
   EOF
   ```

2. **Verify Referential Integrity**
   ```bash
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<EOF
   -- Check for orphaned subscriptions
   SELECT COUNT(*) as orphaned_subscriptions
   FROM subscriptions s
   LEFT JOIN packages p ON s.package_id = p.id
   WHERE p.id IS NULL;
   
   -- Check for orphaned tariffs
   SELECT COUNT(*) as orphaned_tariffs
   FROM tariffs t
   LEFT JOIN services s ON t.service_id = s.id
   WHERE s.id IS NULL;
   EOF
   ```

3. **Check Data Consistency**
   ```bash
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<EOF
   -- Check for invalid dates
   SELECT COUNT(*) as invalid_subscriptions
   FROM subscriptions
   WHERE end_date < start_date;
   
   -- Check for negative costs
   SELECT COUNT(*) as invalid_tariffs
   FROM tariffs
   WHERE cost_per_message < 0;
   EOF
   ```

4. **Test Critical Queries**
   ```bash
   # Test tariff lookup
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c \
       "SELECT * FROM tariffs WHERE is_active = true LIMIT 5;"
   
   # Test subscription lookup
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c \
       "SELECT * FROM subscriptions WHERE status = 'active' LIMIT 5;"
   ```


5. **Review Restore Report**
   ```bash
   # Check the generated restore report
   cat /var/backups/tariff-system/restore_report_*.txt
   ```

---

### Procedure D: Resume Operations

**Bring the system back online**:

1. **Re-enable Database Writes**
   ```bash
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
       -c "ALTER DATABASE sms_gateway SET default_transaction_read_only = off;"
   ```

2. **Start Application Services**
   ```bash
   systemctl start tariff-api
   
   # Wait for health check
   sleep 10
   curl http://localhost:8080/health
   ```

3. **Verify Application Connectivity**
   ```bash
   # Test API endpoints
   curl http://localhost:8080/api/v1/tariffs
   curl http://localhost:8080/api/v1/packages
   ```

4. **Monitor System Health**
   ```bash
   # Check logs for errors
   journalctl -u tariff-api -n 50 --no-pager
   
   # Monitor metrics
   curl http://localhost:8080/metrics | grep -E "(http_requests|database_queries)"
   ```

5. **Notify Stakeholders**
   ```bash
   # Send recovery notification
   echo "System recovered at $(date)" | \
       mail -s "Tariff System Recovery Complete" admin@example.com
   ```

---

## Validation and Testing

### Automated Validation Script

Create a validation script to run after every restore:


```bash
#!/bin/bash
# Save as: scripts/validate-restore.sh

# Run validation checks
bash scripts/validate-restore.sh

# Check exit code
if [ $? -eq 0 ]; then
    echo "Validation passed"
else
    echo "Validation failed - review errors"
fi
```

### Manual Validation Checklist

After any restore operation, verify:

- [ ] All required tables exist
- [ ] Row counts match expectations
- [ ] No orphaned records (referential integrity)
- [ ] No data inconsistencies (negative values, invalid dates)
- [ ] Indexes are present
- [ ] Constraints are intact
- [ ] Latest timestamps are reasonable
- [ ] Critical queries execute successfully
- [ ] Application can connect to database
- [ ] No errors in application logs

---

## Periodic Testing Schedule

### Monthly Restore Test

**Frequency**: First Sunday of each month at 3:00 AM

**Purpose**: Verify backup integrity and recovery procedures

**Procedure**:

```bash
# Run automated periodic test
bash scripts/test-restore-periodic.sh

# Review test report
cat /var/log/restore-tests/restore_test_*.txt
```

**Setup Automation** (Linux/macOS):

```bash
# Add to crontab
crontab -e

# Add this line for first Sunday of month at 3 AM
0 3 1-7 * 0 /path/to/scripts/test-restore-periodic.sh
```

**Setup Automation** (Windows):

```powershell
# Create scheduled task
$action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
    -Argument "-File C:\path\to\scripts\test-restore-periodic.ps1"

$trigger = New-ScheduledTaskTrigger -Weekly -DaysOfWeek Sunday -At 3am

Register-ScheduledTask -TaskName "MonthlyRestoreTest" `
    -Action $action -Trigger $trigger
```


---

### Quarterly Full DR Drill

**Frequency**: Quarterly (January, April, July, October)

**Purpose**: Test complete disaster recovery procedures

**Duration**: 4 hours

**Participants**: All technical teams

**Procedure**:

1. **Pre-Drill Preparation** (1 week before)
   - Schedule maintenance window
   - Notify stakeholders
   - Prepare test environment
   - Review DR plan

2. **Drill Execution** (Day of drill)
   - Simulate complete outage
   - Execute full recovery procedure
   - Measure RTO/RPO achievement
   - Document all steps and times

3. **Post-Drill Review** (Within 1 week)
   - Analyze results
   - Identify improvements
   - Update procedures
   - Train team on changes

**Drill Checklist**:

- [ ] Test environment prepared
- [ ] Stakeholders notified
- [ ] Backup files accessible
- [ ] Recovery team assembled
- [ ] Communication channels tested
- [ ] Execute database restore
- [ ] Execute application deployment
- [ ] Verify system functionality
- [ ] Measure recovery times
- [ ] Document lessons learned
- [ ] Update DR plan

---

### Weekly Backup Validation

**Frequency**: Every Sunday at 3:00 AM

**Purpose**: Verify backup files are valid and restorable

**Procedure**:

```bash
# Run backup validation
bash scripts/test-backup-validation.sh

# Check results
cat /var/backups/tariff-system/validation_*.log
```

**Setup Automation**:

```bash
# Add to crontab
0 3 * * 0 /path/to/scripts/test-backup-validation.sh
```

---

## Recovery Time Tracking

### Measuring RTO

Track actual recovery times for each restore operation:

```bash
# Start timer
START_TIME=$(date +%s)

# Perform restore
bash scripts/restore-tariff-data.sh --drop $BACKUP_FILE

# Calculate duration
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "Recovery completed in $DURATION seconds"
```

### RTO Targets

| Component | Target RTO | Actual RTO | Status |
|-----------|------------|------------|--------|
| Database | 60 minutes | _measured_ | ✓/✗ |
| Application | 30 minutes | _measured_ | ✓/✗ |
| Full System | 90 minutes | _measured_ | ✓/✗ |


---

## Troubleshooting Common Issues

### Issue 1: Restore Fails with "Insufficient Disk Space"

**Symptoms**:
- Restore operation fails midway
- Error message about disk space

**Solution**:
```bash
# Check available disk space
df -h

# Clean up old backups
find /var/backups/tariff-system/daily -mtime +30 -delete

# Or move backups to S3
aws s3 sync /var/backups/tariff-system/ s3://backup-bucket/

# Retry restore
bash scripts/restore-tariff-data.sh --drop $BACKUP_FILE
```

---

### Issue 2: Decryption Fails

**Symptoms**:
- "Failed to decrypt" error
- Restore cannot proceed

**Solution**:
```bash
# Verify encryption key exists
ls -l /etc/tariff-system/backup.key

# Test decryption manually
openssl enc -aes-256-cbc -d -pbkdf2 \
    -in test.dump.enc \
    -out test.dump \
    -pass file:/etc/tariff-system/backup.key

# If key is wrong, restore from backup key
cp /etc/tariff-system/backup.key.backup /etc/tariff-system/backup.key
```

---

### Issue 3: Database Connection Fails

**Symptoms**:
- Cannot connect to database
- "Connection refused" errors

**Solution**:
```bash
# Check PostgreSQL service
systemctl status postgresql

# Start if stopped
systemctl start postgresql

# Check connectivity
psql -h $DB_HOST -U $DB_USER -d postgres -c "SELECT version();"

# Check firewall
sudo ufw status
sudo ufw allow 5432/tcp
```

---

### Issue 4: Validation Fails After Restore

**Symptoms**:
- Restore completes but validation fails
- Data inconsistencies detected

**Solution**:
```bash
# Review validation errors
cat /var/backups/tariff-system/restore_report_*.txt

# Check for specific issues
psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<EOF
-- Find orphaned records
SELECT 'subscriptions' as table_name, COUNT(*) 
FROM subscriptions s 
LEFT JOIN packages p ON s.package_id = p.id 
WHERE p.id IS NULL;

-- Find invalid data
SELECT 'tariffs' as table_name, COUNT(*) 
FROM tariffs 
WHERE cost_per_message < 0;
EOF

# If issues are minor, fix manually
# If issues are major, restore from different backup
```

---

## Best Practices

1. **Always Test Backups**
   - Run monthly restore tests
   - Verify backup integrity weekly
   - Test different backup types (daily, weekly, monthly)

2. **Document Everything**
   - Record all restore operations
   - Track recovery times
   - Note any issues encountered

3. **Keep Multiple Backup Copies**
   - Local backups for fast recovery
   - S3 backups for disaster recovery
   - Off-site backups for catastrophic failures

4. **Automate Where Possible**
   - Automated backup validation
   - Automated periodic testing
   - Automated alerting

5. **Maintain Recovery Procedures**
   - Review procedures quarterly
   - Update after each DR test
   - Train team on changes

6. **Monitor Backup Health**
   - Check backup sizes
   - Verify backup completion
   - Alert on failures

7. **Secure Encryption Keys**
   - Store keys separately from backups
   - Maintain key backups
   - Rotate keys annually

8. **Practice Recovery**
   - Regular DR drills
   - Team training sessions
   - Tabletop exercises

---

## Related Documentation

- [Disaster Recovery Plan](../docs/DISASTER_RECOVERY_PLAN.md)
- [Backup & Recovery Quick Start](./BACKUP_RECOVERY_QUICK_START.md)
- [Database Schema](../docs/DATABASE_SCHEMA_TARIFF.md)
- [Operations Manual](../docs/OPERATIONS_MANUAL.md)

---

## Support and Escalation

### Level 1: On-Call Engineer
- Initial assessment
- Execute standard recovery procedures
- Contact: oncall@example.com

### Level 2: Database Administrator
- Complex database issues
- Data integrity problems
- Contact: dba@example.com

### Level 3: Engineering Manager
- Major incidents
- Decision making
- Contact: eng-manager@example.com

### Emergency Contact
- 24/7 Hotline: +1-XXX-XXX-XXXX
- Slack: #incident-response
- Email: emergency@example.com

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2024 | Operations Team | Initial version |

---

## Approval

This document has been reviewed and approved by:

- [ ] Database Administrator
- [ ] DevOps Lead
- [ ] Engineering Manager
- [ ] Security Officer

**Last Review Date**: _______________

**Next Review Date**: _______________
