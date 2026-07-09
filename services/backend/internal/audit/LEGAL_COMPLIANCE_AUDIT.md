# Legal Compliance Audit Logging

## Overview

Enhanced audit logging system for legal compliance with GDPR, CCPA, and other regulations. Provides comprehensive tracking of all sensitive operations with long-term retention.

## Legal Compliance Events

### Consent Management
- **consent_granted**: User grants consent
- **consent_revoked**: User revokes consent

### Data Rights
- **data_exported**: User exports their data (GDPR Article 20)
- **data_deleted**: User data is deleted (GDPR Article 17)

### Security Events
- **fraud_detected**: Fraud detection alert triggered
- **account_suspended**: Account suspended due to suspicious activity
- **account_deleted**: Account permanently deleted

### Financial Events
- **payment_processed**: Payment successfully processed
- **payment_failed**: Payment failed
- **subscription_changed**: Subscription plan changed

## Usage

### Log Consent Events

```go
import "your-project/internal/audit"

logger := audit.NewLogger(db)

// Log consent granted
err := logger.LogConsentGranted(
    ctx,
    &userID,
    &tenantID,
    "privacy_policy",
    "1.0",
    "192.168.1.1",
)

// Log consent revoked
err := logger.LogConsentRevoked(
    ctx,
    &userID,
    &tenantID,
    "marketing",
    "192.168.1.1",
)
```

### Log Data Rights Events

```go
// Log data export
err := logger.LogDataExported(
    ctx,
    &userID,
    &tenantID,
    "export_123",
    "192.168.1.1",
)

// Log data deletion
err := logger.LogDataDeleted(
    ctx,
    &userID,
    &tenantID,
    "deletion_request_456",
    "192.168.1.1",
)
```

### Log Security Events

```go
// Log fraud detection
err := logger.LogFraudDetected(
    ctx,
    &userID,
    &tenantID,
    alertID,
    "high",
    "192.168.1.1",
)

// Log account suspension
err := logger.LogAccountSuspended(
    ctx,
    &userID,
    &tenantID,
    "Multiple failed payment attempts",
    "192.168.1.1",
)
```

### Custom Legal Events

```go
event := &audit.LegalEvent{
    UserID:     &userID,
    TenantID:   &tenantID,
    EventType:  "custom_legal_event",
    ResourceID: "resource_123",
    IPAddress:  "192.168.1.1",
    UserAgent:  "Mozilla/5.0...",
    Details: map[string]interface{}{
        "custom_field": "value",
    },
}

err := logger.LogLegalEvent(ctx, event)
```

## Retention Policy

### Legal Events
**Retention**: 7 years (legal requirement)
**Reason**: Compliance with financial regulations and GDPR

### Financial Events
**Retention**: 7 years
**Reason**: Tax compliance and audit requirements

### User Activity
**Retention**: 1 year
**Reason**: Security monitoring and troubleshooting

### System Logs
**Retention**: 90 days
**Reason**: Operational monitoring

## Compliance Requirements

### GDPR Compliance

#### Article 5: Principles
✅ **Lawfulness, fairness and transparency**: All data processing is logged
✅ **Purpose limitation**: Logs specify purpose of each operation
✅ **Data minimisation**: Only necessary data is logged
✅ **Accuracy**: Logs are immutable and timestamped
✅ **Storage limitation**: Retention policies implemented
✅ **Integrity and confidentiality**: Logs are encrypted

#### Article 17: Right to Erasure
✅ Deletion requests are logged
✅ Deletion process is audited
✅ Logs retained for legal compliance

#### Article 20: Right to Data Portability
✅ Export requests are logged
✅ Export process is audited

#### Article 30: Records of Processing Activities
✅ All processing activities are logged
✅ Logs include purpose, categories, and recipients

### CCPA Compliance

✅ **Right to Know**: All data access is logged
✅ **Right to Delete**: Deletion requests are logged
✅ **Right to Opt-Out**: Consent changes are logged
✅ **Non-Discrimination**: All actions are logged equally

## Database Schema

### audit_trail Table

```sql
CREATE TABLE audit_trail (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID,
    tenant_id UUID,
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    ip_address INET NOT NULL,
    user_agent TEXT,
    result VARCHAR(20) NOT NULL,
    error_message TEXT,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Indexes for performance
CREATE INDEX idx_audit_trail_user ON audit_trail(user_id, created_at DESC);
CREATE INDEX idx_audit_trail_tenant ON audit_trail(tenant_id, created_at DESC);
CREATE INDEX idx_audit_trail_action ON audit_trail(action, created_at DESC);
CREATE INDEX idx_audit_trail_resource ON audit_trail(resource, created_at DESC);
CREATE INDEX idx_audit_trail_created_at ON audit_trail(created_at DESC);

-- Partitioning by month for performance
CREATE TABLE audit_trail_2024_01 PARTITION OF audit_trail
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## Query Examples

### Get All Consent Events for User

```go
filters := audit.AuditQueryFilters{
    UserID:    &userID,
    Action:    "consent_granted",
    StartDate: &startDate,
    EndDate:   &endDate,
    Limit:     100,
}

entries, err := logger.Query(filters)
```

### Get All Data Deletion Events

```go
filters := audit.AuditQueryFilters{
    Action:   "data_deleted",
    Resource: "legal_compliance",
    Limit:    100,
}

entries, err := logger.Query(filters)
```

### Get All Fraud Alerts for Tenant

```go
filters := audit.AuditQueryFilters{
    TenantID: &tenantID,
    Action:   "fraud_detected",
    Limit:    100,
}

entries, err := logger.Query(filters)
```

## Best Practices

### 1. Always Log Legal Events
```go
// ❌ Bad: No logging
func DeleteUserData(userID int64) error {
    return deleteData(userID)
}

// ✅ Good: Log the operation
func DeleteUserData(ctx context.Context, userID int64) error {
    err := deleteData(userID)
    if err != nil {
        return err
    }
    
    return auditLogger.LogDataDeleted(ctx, &userID, &tenantID, requestID, ipAddress)
}
```

### 2. Include Context
```go
// ✅ Always include IP address and user agent
event := &audit.LegalEvent{
    UserID:    &userID,
    TenantID:  &tenantID,
    EventType: audit.EventConsentGranted,
    IPAddress: getIPAddress(r),
    UserAgent: r.UserAgent(),
    Details: map[string]interface{}{
        "consent_type": "privacy_policy",
        "version":      "1.0",
    },
}
```

### 3. Handle Errors
```go
// ✅ Log audit failures but don't block operation
err := auditLogger.LogLegalEvent(ctx, event)
if err != nil {
    log.Error("Failed to log legal event", zap.Error(err))
    // Continue with operation
}
```

### 4. Use Transactions
```go
// ✅ Log within same transaction as operation
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Perform operation
err = deleteUserData(tx, userID)
if err != nil {
    return err
}

// Log operation
err = auditLogger.LogDataDeleted(ctx, &userID, &tenantID, requestID, ipAddress)
if err != nil {
    return err
}

return tx.Commit()
```

## Monitoring and Alerts

### Critical Events to Monitor

1. **High Volume of Deletions**: Alert if >10 deletion requests per hour
2. **Failed Exports**: Alert if export failure rate >5%
3. **Consent Revocations**: Monitor for patterns
4. **Fraud Alerts**: Immediate notification for critical severity

### Metrics to Track

- Number of consent grants/revokes per day
- Number of data exports per day
- Number of deletion requests per day
- Average time to process deletion
- Fraud alert rate

## Archival and Backup

### Daily Backup
```bash
# Backup audit logs
pg_dump -t audit_trail -f audit_trail_$(date +%Y%m%d).sql

# Encrypt backup
gpg --encrypt --recipient admin@company.com audit_trail_$(date +%Y%m%d).sql

# Upload to S3
aws s3 cp audit_trail_$(date +%Y%m%d).sql.gpg s3://backups/audit/
```

### Monthly Archival
```sql
-- Archive old logs to separate table
INSERT INTO audit_trail_archive
SELECT * FROM audit_trail
WHERE created_at < NOW() - INTERVAL '1 year'
AND action NOT IN ('consent_granted', 'consent_revoked', 'data_deleted', 'data_exported');

-- Delete archived logs (keep legal events)
DELETE FROM audit_trail
WHERE created_at < NOW() - INTERVAL '1 year'
AND action NOT IN ('consent_granted', 'consent_revoked', 'data_deleted', 'data_exported');
```

## Requirements

- Requirements: 36.10, 36.11
- GDPR Compliance
- CCPA Compliance
- 7-year retention for legal events

## Testing

Run tests:
```bash
go test ./internal/audit/...
```

## Status

✅ Enhanced for Legal Compliance
✅ 7-year Retention Policy
✅ Documented
✅ Tested
