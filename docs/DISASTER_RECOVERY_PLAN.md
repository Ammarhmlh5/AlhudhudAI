# Disaster Recovery Plan - Tariff Pricing System

## Document Information

- **Version**: 1.0
- **Last Updated**: 2024
- **Owner**: Platform Operations Team
- **Review Cycle**: Quarterly

## Table of Contents

1. [Overview](#overview)
2. [Recovery Objectives](#recovery-objectives)
3. [Disaster Scenarios](#disaster-scenarios)
4. [Recovery Procedures](#recovery-procedures)
5. [Roles and Responsibilities](#roles-and-responsibilities)
6. [Communication Plan](#communication-plan)
7. [Testing and Maintenance](#testing-and-maintenance)

---

## Overview

### Purpose

This Disaster Recovery Plan (DRP) provides comprehensive procedures for recovering the Tariff Pricing System in the event of a disaster or major system failure. The plan ensures business continuity and minimizes data loss.

### Scope

This plan covers:
- Database recovery
- Application server recovery
- Cache layer recovery
- Network and infrastructure recovery
- Data integrity verification

### Requirements

- **Requirement 25.11**: Disaster Recovery Plan documentation
- **Requirement 28.8**: Disaster Recovery Plan implementation
- **Requirement 28.9**: Regular DR testing

---

## Recovery Objectives

### Recovery Time Objective (RTO)

The maximum acceptable time to restore system functionality:

| Component | RTO | Priority |
|-----------|-----|----------|
| Database (Critical Tables) | 1 hour | P0 |
| Application Servers | 2 hours | P0 |
| Redis Cache | 30 minutes | P1 |
| Monitoring Systems | 4 hours | P2 |
| Reporting Systems | 8 hours | P3 |

### Recovery Point Objective (RPO)

The maximum acceptable data loss:

| Data Type | RPO | Backup Frequency |
|-----------|-----|------------------|
| Financial Transactions | 0 minutes | Real-time replication |
| Tariffs & Packages | 1 hour | Hourly snapshots |
| User Data | 24 hours | Daily backups |
| Logs & Metrics | 1 hour | Continuous streaming |

### Service Level Objectives (SLO)

- **Availability**: 99.9% uptime
- **Data Durability**: 99.999999999% (11 nines)
- **Recovery Success Rate**: 99% of DR tests successful

---

## Disaster Scenarios

### Scenario 1: Database Failure

**Description**: Primary database becomes unavailable or corrupted.

**Impact**: 
- All financial transactions halted
- No access to tariff or subscription data
- Complete system outage

**Detection**:
- Database connection failures
- Prometheus alert: `database_down`
- Application health checks failing

**Recovery Priority**: P0 (Critical)

---

### Scenario 2: Application Server Failure

**Description**: One or more application servers crash or become unresponsive.

**Impact**:
- Reduced capacity
- Potential service degradation
- Load balancer redirects traffic

**Detection**:
- Health check failures
- Prometheus alert: `app_server_down`
- Increased error rates

**Recovery Priority**: P0 (Critical)

---

### Scenario 3: Redis Cache Failure

**Description**: Redis cluster becomes unavailable.

**Impact**:
- Increased database load
- Slower response times
- No immediate data loss (cache only)

**Detection**:
- Redis connection failures
- Prometheus alert: `redis_down`
- Increased database query latency

**Recovery Priority**: P1 (High)

---

### Scenario 4: Complete Data Center Outage

**Description**: Entire data center or availability zone fails.

**Impact**:
- Complete system outage
- All services unavailable
- Requires failover to backup region

**Detection**:
- All monitoring alerts
- External monitoring services
- Customer reports

**Recovery Priority**: P0 (Critical)

---

### Scenario 5: Data Corruption

**Description**: Database corruption due to hardware failure or software bug.

**Impact**:
- Inconsistent data
- Potential financial discrepancies
- Requires point-in-time recovery

**Detection**:
- Data integrity checks failing
- Checksum mismatches
- Application errors

**Recovery Priority**: P0 (Critical)

---

### Scenario 6: Ransomware Attack

**Description**: Malicious encryption of system data.

**Impact**:
- Data inaccessible
- System compromised
- Requires clean restore from backups

**Detection**:
- Unusual file modifications
- Security alerts
- System behavior anomalies

**Recovery Priority**: P0 (Critical)

---

## Recovery Procedures

### Procedure 1: Database Recovery from Backup

**Scenario**: Database failure or corruption

**Prerequisites**:
- Valid backup available
- Encryption key accessible
- Database credentials available

**Steps**:

1. **Assess the Situation** (5 minutes)
   ```bash
   # Check database status
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT version();"
   
   # Check replication lag
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT * FROM pg_stat_replication;"
   ```

2. **Stop Application Traffic** (5 minutes)
   ```bash
   # Update load balancer to maintenance mode
   aws elbv2 modify-target-group --target-group-arn $TG_ARN --health-check-enabled false
   
   # Stop application servers
   systemctl stop tariff-api
   ```

3. **Create Restore Point** (2 minutes)
   ```bash
   # If database is still accessible
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
     -c "SELECT pg_create_restore_point('before_recovery_$(date +%s)');"
   ```

4. **Identify Latest Valid Backup** (5 minutes)
   ```bash
   # List available backups
   bash scripts/restore-tariff-data.sh --list
   
   # Verify backup integrity
   bash scripts/test-backup-validation.sh
   ```

5. **Restore Database** (30-45 minutes)
   ```bash
   # Full restore
   bash scripts/restore-tariff-data.sh \
     --drop \
     /var/backups/tariff-system/daily/tariff_backup_daily_YYYYMMDD_HHMMSS.tar.gz
   
   # Or restore from S3
   bash scripts/restore-tariff-data.sh \
     --s3 backups/daily/YYYY-MM-DD/tariff_backup_daily_YYYYMMDD_HHMMSS.tar.gz \
     --drop
   ```

6. **Verify Data Integrity** (10 minutes)
   ```bash
   # Run integrity checks
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f scripts/verify-data-integrity.sql
   
   # Check row counts
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME <<EOF
   SELECT 'tariffs' as table_name, COUNT(*) FROM tariffs
   UNION ALL
   SELECT 'packages', COUNT(*) FROM packages
   UNION ALL
   SELECT 'subscriptions', COUNT(*) FROM subscriptions;
   EOF
   ```

7. **Restore Application Servers** (10 minutes)
   ```bash
   # Start application servers
   systemctl start tariff-api
   
   # Verify health
   curl http://localhost:8080/health
   ```

8. **Resume Traffic** (5 minutes)
   ```bash
   # Re-enable health checks
   aws elbv2 modify-target-group --target-group-arn $TG_ARN --health-check-enabled true
   
   # Monitor traffic
   watch -n 1 'curl -s http://localhost:8080/metrics | grep http_requests_total'
   ```

9. **Post-Recovery Validation** (15 minutes)
   - Run smoke tests
   - Verify critical transactions
   - Check monitoring dashboards
   - Review error logs

**Total Estimated Time**: 60-90 minutes

**Rollback Plan**: If recovery fails, restore from previous backup or failover to read replica.

---

### Procedure 2: Application Server Recovery

**Scenario**: Application server failure

**Prerequisites**:
- Healthy load balancer
- Container images available
- Configuration files accessible

**Steps**:

1. **Identify Failed Servers** (2 minutes)
   ```bash
   # Check server status
   aws ec2 describe-instance-status --instance-ids $INSTANCE_IDS
   
   # Check application logs
   journalctl -u tariff-api -n 100 --no-pager
   ```

2. **Remove from Load Balancer** (2 minutes)
   ```bash
   # Deregister unhealthy targets
   aws elbv2 deregister-targets \
     --target-group-arn $TG_ARN \
     --targets Id=$INSTANCE_ID
   ```

3. **Diagnose Issue** (10 minutes)
   - Check system resources (CPU, memory, disk)
   - Review application logs
   - Check database connectivity
   - Verify Redis connectivity

4. **Restart Application** (5 minutes)
   ```bash
   # Restart service
   systemctl restart tariff-api
   
   # Wait for health check
   sleep 30
   curl http://localhost:8080/health
   ```

5. **If Restart Fails, Redeploy** (15 minutes)
   ```bash
   # Pull latest image
   docker pull $ECR_REPO/tariff-api:latest
   
   # Stop and remove old container
   docker stop tariff-api
   docker rm tariff-api
   
   # Start new container
   docker run -d \
     --name tariff-api \
     --env-file /etc/tariff-system/.env \
     -p 8080:8080 \
     $ECR_REPO/tariff-api:latest
   ```

6. **Re-register with Load Balancer** (2 minutes)
   ```bash
   # Register target
   aws elbv2 register-targets \
     --target-group-arn $TG_ARN \
     --targets Id=$INSTANCE_ID
   
   # Wait for healthy status
   aws elbv2 describe-target-health \
     --target-group-arn $TG_ARN \
     --targets Id=$INSTANCE_ID
   ```

7. **Monitor Performance** (10 minutes)
   - Check response times
   - Monitor error rates
   - Verify metrics collection

**Total Estimated Time**: 30-45 minutes

---

### Procedure 3: Redis Cache Recovery

**Scenario**: Redis cluster failure

**Prerequisites**:
- Redis configuration files
- Backup snapshots (if available)

**Steps**:

1. **Assess Redis Status** (5 minutes)
   ```bash
   # Check Redis connectivity
   redis-cli -h $REDIS_HOST -p $REDIS_PORT ping
   
   # Check cluster status
   redis-cli -h $REDIS_HOST -p $REDIS_PORT cluster info
   
   # Check memory usage
   redis-cli -h $REDIS_HOST -p $REDIS_PORT info memory
   ```

2. **Restart Redis** (5 minutes)
   ```bash
   # Restart Redis service
   systemctl restart redis
   
   # Or restart container
   docker restart redis
   
   # Verify startup
   redis-cli ping
   ```

3. **If Restart Fails, Rebuild Cluster** (20 minutes)
   ```bash
   # Stop all Redis nodes
   systemctl stop redis
   
   # Clear data directories (if corrupted)
   rm -rf /var/lib/redis/*
   
   # Start Redis nodes
   systemctl start redis
   
   # Recreate cluster
   redis-cli --cluster create \
     $NODE1:6379 $NODE2:6379 $NODE3:6379 \
     --cluster-replicas 1
   ```

4. **Warm Up Cache** (10 minutes)
   ```bash
   # Run cache warming script
   bash scripts/warm-cache.sh
   
   # Monitor cache hit rate
   redis-cli info stats | grep keyspace_hits
   ```

5. **Verify Application Connectivity** (5 minutes)
   ```bash
   # Test from application
   curl http://localhost:8080/health/redis
   
   # Check metrics
   curl http://localhost:8080/metrics | grep redis_
   ```

**Total Estimated Time**: 30-45 minutes

**Note**: Application continues to function without Redis, but with degraded performance.

---

### Procedure 4: Complete System Recovery

**Scenario**: Complete data center outage

**Prerequisites**:
- Backup region configured
- DNS failover capability
- Valid backups in remote location

**Steps**:

1. **Declare Disaster** (5 minutes)
   - Notify stakeholders
   - Activate DR team
   - Document incident start time

2. **Provision Infrastructure in Backup Region** (30 minutes)
   ```bash
   # Launch EC2 instances
   aws ec2 run-instances \
     --image-id $AMI_ID \
     --instance-type t3.large \
     --count 3 \
     --region us-west-2
   
   # Create RDS instance
   aws rds create-db-instance \
     --db-instance-identifier tariff-db-dr \
     --db-instance-class db.t3.large \
     --engine postgres \
     --region us-west-2
   
   # Create ElastiCache cluster
   aws elasticache create-cache-cluster \
     --cache-cluster-id tariff-redis-dr \
     --cache-node-type cache.t3.medium \
     --engine redis \
     --region us-west-2
   ```

3. **Restore Database from S3** (45 minutes)
   ```bash
   # Download latest backup from S3
   aws s3 cp \
     s3://$BACKUP_BUCKET/backups/daily/latest/ \
     /tmp/backup/ \
     --recursive \
     --region us-west-2
   
   # Restore database
   bash scripts/restore-tariff-data.sh \
     --drop \
     /tmp/backup/tariff_backup_daily_*.tar.gz
   ```

4. **Deploy Application** (20 minutes)
   ```bash
   # Deploy application containers
   for instance in $INSTANCE_IDS; do
     ssh $instance "docker pull $ECR_REPO/tariff-api:latest"
     ssh $instance "docker run -d --env-file /etc/tariff-system/.env -p 8080:8080 $ECR_REPO/tariff-api:latest"
   done
   ```

5. **Configure Load Balancer** (10 minutes)
   ```bash
   # Create target group
   aws elbv2 create-target-group \
     --name tariff-api-dr \
     --protocol HTTP \
     --port 8080 \
     --vpc-id $VPC_ID \
     --region us-west-2
   
   # Register targets
   aws elbv2 register-targets \
     --target-group-arn $TG_ARN \
     --targets Id=$INSTANCE1 Id=$INSTANCE2 Id=$INSTANCE3
   ```

6. **Update DNS** (10 minutes)
   ```bash
   # Update Route53 to point to DR region
   aws route53 change-resource-record-sets \
     --hosted-zone-id $ZONE_ID \
     --change-batch file://dns-failover.json
   
   # Wait for DNS propagation
   watch -n 5 'dig api.tariff-system.com +short'
   ```

7. **Verify System Functionality** (20 minutes)
   - Run smoke tests
   - Verify critical APIs
   - Check monitoring dashboards
   - Test user workflows

8. **Notify Stakeholders** (10 minutes)
   - Send recovery notification
   - Update status page
   - Document recovery time

**Total Estimated Time**: 2-3 hours

---

### Procedure 5: Data Corruption Recovery

**Scenario**: Data corruption detected

**Prerequisites**:
- Point-in-time recovery capability
- Transaction logs available
- Backup before corruption

**Steps**:

1. **Identify Corruption Scope** (15 minutes)
   ```sql
   -- Check for inconsistencies
   SELECT * FROM tariffs WHERE cost_per_message < 0;
   SELECT * FROM subscriptions WHERE end_date < start_date;
   SELECT * FROM accounting_transactions WHERE amount IS NULL;
   
   -- Check referential integrity
   SELECT COUNT(*) FROM subscriptions s
   LEFT JOIN packages p ON s.package_id = p.id
   WHERE p.id IS NULL;
   ```

2. **Determine Corruption Time** (10 minutes)
   - Review audit logs
   - Check transaction timestamps
   - Identify last known good state

3. **Stop Write Operations** (5 minutes)
   ```bash
   # Set database to read-only
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
     -c "ALTER DATABASE $DB_NAME SET default_transaction_read_only = on;"
   
   # Stop application writes
   systemctl stop tariff-api
   ```

4. **Restore to Point-in-Time** (45 minutes)
   ```bash
   # Restore from backup before corruption
   bash scripts/restore-tariff-data.sh \
     /var/backups/tariff-system/daily/tariff_backup_daily_YYYYMMDD_HHMMSS.tar.gz
   
   # Apply transaction logs up to corruption point
   pg_restore -h $DB_HOST -U $DB_USER -d $DB_NAME \
     --use-list=transactions_before_corruption.list \
     backup.dump
   ```

5. **Verify Data Integrity** (20 minutes)
   ```bash
   # Run comprehensive integrity checks
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f scripts/verify-data-integrity.sql
   
   # Compare row counts with audit logs
   bash scripts/compare-data-counts.sh
   ```

6. **Resume Operations** (10 minutes)
   ```bash
   # Re-enable writes
   psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
     -c "ALTER DATABASE $DB_NAME SET default_transaction_read_only = off;"
   
   # Start application
   systemctl start tariff-api
   ```

7. **Root Cause Analysis** (Post-recovery)
   - Identify corruption source
   - Implement preventive measures
   - Update monitoring alerts

**Total Estimated Time**: 90-120 minutes

---

## Roles and Responsibilities

### Incident Commander

**Responsibilities**:
- Declare disaster
- Coordinate recovery efforts
- Make critical decisions
- Communicate with stakeholders

**Contact**: ops-lead@example.com

### Database Administrator

**Responsibilities**:
- Database recovery
- Data integrity verification
- Backup management
- Performance tuning

**Contact**: dba@example.com

### DevOps Engineer

**Responsibilities**:
- Infrastructure provisioning
- Application deployment
- Network configuration
- Monitoring setup

**Contact**: devops@example.com

### Application Developer

**Responsibilities**:
- Application troubleshooting
- Code deployment
- Integration testing
- Bug fixes

**Contact**: dev-team@example.com

### Security Officer

**Responsibilities**:
- Security assessment
- Access control
- Incident investigation
- Compliance verification

**Contact**: security@example.com

---

## Communication Plan

### Internal Communication

**During Incident**:
- Slack channel: #incident-response
- Conference bridge: +1-XXX-XXX-XXXX
- Status updates: Every 30 minutes

**Escalation Path**:
1. On-call engineer
2. Team lead
3. Engineering manager
4. CTO

### External Communication

**Customer Notification**:
- Status page: status.tariff-system.com
- Email notifications
- In-app banners

**Stakeholder Updates**:
- Executive summary every 2 hours
- Post-incident report within 24 hours

---

## Testing and Maintenance

### DR Testing Schedule

| Test Type | Frequency | Duration | Participants |
|-----------|-----------|----------|--------------|
| Backup Validation | Daily | 15 min | Automated |
| Partial Recovery | Monthly | 2 hours | DBA, DevOps |
| Full DR Drill | Quarterly | 4 hours | All teams |
| Tabletop Exercise | Bi-annually | 2 hours | Leadership |

### Test Procedures

**Monthly Partial Recovery Test**:
1. Select random backup from previous week
2. Restore to test environment
3. Verify data integrity
4. Document results
5. Update procedures if needed

**Quarterly Full DR Drill**:
1. Schedule maintenance window
2. Simulate complete outage
3. Execute full recovery procedure
4. Measure RTO/RPO achievement
5. Conduct post-drill review
6. Update plan based on lessons learned

### Plan Maintenance

**Review Triggers**:
- After each DR test
- After actual disaster recovery
- Quarterly scheduled review
- After major system changes
- After infrastructure updates

**Update Process**:
1. Identify needed changes
2. Draft updates
3. Review with team
4. Approve changes
5. Distribute updated plan
6. Train team on changes

---

## Appendices

### Appendix A: Contact List

| Role | Name | Phone | Email |
|------|------|-------|-------|
| Incident Commander | TBD | TBD | TBD |
| DBA Lead | TBD | TBD | TBD |
| DevOps Lead | TBD | TBD | TBD |
| Security Lead | TBD | TBD | TBD |

### Appendix B: System Credentials

**Location**: Secure password manager
**Access**: Incident response team only

### Appendix C: Vendor Contacts

| Vendor | Service | Support Phone | Support Email |
|--------|---------|---------------|---------------|
| AWS | Infrastructure | +1-XXX-XXX-XXXX | aws-support@amazon.com |
| DataDog | Monitoring | +1-XXX-XXX-XXXX | support@datadoghq.com |

### Appendix D: Recovery Checklists

**Database Recovery Checklist**:
- [ ] Assess situation
- [ ] Stop application traffic
- [ ] Create restore point
- [ ] Identify backup
- [ ] Restore database
- [ ] Verify integrity
- [ ] Restore applications
- [ ] Resume traffic
- [ ] Post-recovery validation

**Application Recovery Checklist**:
- [ ] Identify failed servers
- [ ] Remove from load balancer
- [ ] Diagnose issue
- [ ] Restart or redeploy
- [ ] Re-register with LB
- [ ] Monitor performance

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2024 | Operations Team | Initial version |

---

## Approval

| Role | Name | Signature | Date |
|------|------|-----------|------|
| CTO | TBD | | |
| VP Engineering | TBD | | |
| Security Officer | TBD | | |
