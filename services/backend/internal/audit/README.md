# Audit Trail Logging System

## Overview

The Audit Trail Logging system provides comprehensive, immutable logging for all critical operations in the SMS Gateway system. It implements Requirements 39.1-39.5 from the specification.

## Features

- **Immutable Logs**: Audit trail records cannot be modified or deleted
- **Comprehensive Logging**: Captures User ID, Action, Timestamp, IP Address, and Result
- **Flexible Querying**: Support for filtering by tenant, user, action, resource, date range, etc.
- **Automatic Integration**: Integrated with Balance Manager and Status Handler
- **7-Year Retention**: Audit logs are retained for compliance requirements

## Database Schema

The `audit_trail` table is created in the public schema (shared across all tenants):

```sql
CREATE TABLE audit_trail (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    tenant_id UUID REFERENCES tenants(id),
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    ip_address INET NOT NULL,
    user_agent TEXT,
    result VARCHAR(20) NOT NULL CHECK (result IN ('SUCCESS', 'FAILURE', 'PARTIAL')),
    error_message TEXT,
    details JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Immutability

The table has triggers that prevent UPDATE and DELETE operations:

```sql
CREATE TRIGGER prevent_audit_trail_update
    BEFORE UPDATE ON audit_trail
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_trail_modification();

CREATE TRIGGER prevent_audit_trail_delete
    BEFORE DELETE ON audit_trail
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_trail_modification();
```

## Usage

### Creating an Audit Logger

```go
import (
    "database/sql"
    "github.com/al-hudhud/backend/internal/audit"
)

db, _ := sql.Open("postgres", connStr)
auditLogger := audit.NewLogger(db)
```

### Logging Successful Operations

```go
userID := uuid.New()
tenantID := uuid.New()

err := auditLogger.LogSuccess(
    audit.ActionBalanceDeduct,
    audit.ResourceBalance,
    "msg-123",
    &userID,
    &tenantID,
    "192.168.1.1",
    map[string]interface{}{
        "amount": 10.5,
        "old_balance": 100.0,
        "new_balance": 89.5,
    },
)
```

### Logging Failed Operations

```go
err := auditLogger.LogFailure(
    audit.ActionBalanceDeduct,
    audit.ResourceBalance,
    "msg-456",
    &userID,
    &tenantID,
    "192.168.1.2",
    "Insufficient balance",
    map[string]interface{}{
        "requested_amount": 50.0,
        "current_balance": 10.0,
    },
)
```

### Querying Audit Logs

```go
filters := audit.AuditQueryFilters{
    TenantID: &tenantID,
    Action: audit.ActionBalanceDeduct,
    StartDate: &startDate,
    EndDate: &endDate,
    Limit: 100,
    Offset: 0,
}

entries, err := auditLogger.Query(filters)
for _, entry := range entries {
    fmt.Printf("Action: %s, Result: %s, Time: %s\n", 
        entry.Action, entry.Result, entry.CreatedAt)
}
```

## Action Types

### Balance Operations
- `BALANCE_DEDUCT`: Balance deduction
- `BALANCE_REFUND`: Balance refund
- `BALANCE_CHARGE`: Balance charge/top-up
- `BALANCE_ADJUSTMENT`: Manual balance adjustment

### Message Operations
- `MESSAGE_CREATE`: Message creation
- `MESSAGE_SEND`: Message sent
- `MESSAGE_FAIL`: Message failed
- `MESSAGE_CANCEL`: Message cancelled

### Status Operations
- `STATUS_UPDATE`: Status update
- `STATUS_REFUND`: Status-triggered refund

### User Operations
- `USER_LOGIN`: User login
- `USER_LOGOUT`: User logout
- `USER_CREATE`: User creation
- `USER_UPDATE`: User update
- `USER_DELETE`: User deletion
- `PASSWORD_CHANGE`: Password change
- `PASSWORD_RESET`: Password reset
- `2FA_ENABLE`: 2FA enabled
- `2FA_DISABLE`: 2FA disabled

### Tenant Operations
- `TENANT_CREATE`: Tenant creation
- `TENANT_UPDATE`: Tenant update
- `TENANT_DELETE`: Tenant deletion

### Configuration Operations
- `CONFIG_UPDATE`: Configuration update
- `RATE_LIMIT_UPDATE`: Rate limit update

### Channel Operations
- `CHANNEL_CREATE`: Channel creation
- `CHANNEL_UPDATE`: Channel update
- `CHANNEL_DELETE`: Channel deletion

### Access Operations
- `DATA_ACCESS`: Data access
- `SENSITIVE_ACCESS`: Sensitive data access

## Resource Types

- `BALANCE`: Balance resource
- `MESSAGE`: Message resource
- `USER`: User resource
- `TENANT`: Tenant resource
- `CHANNEL`: Channel resource
- `CONFIG`: Configuration resource
- `SESSION`: Session resource
- `API`: API resource

## Result Types

- `SUCCESS`: Operation succeeded
- `FAILURE`: Operation failed
- `PARTIAL`: Operation partially succeeded

## Integration

### Balance Manager Integration

The Balance Manager automatically logs:
- Successful balance deductions
- Failed balance deductions (insufficient balance)
- Balance refunds

### Status Handler Integration

The Status Handler automatically logs:
- Status updates
- Status-triggered refunds for failed messages

## Performance Considerations

### Indexes

The audit_trail table has the following indexes for optimal query performance:

- `idx_audit_trail_tenant`: (tenant_id, created_at DESC)
- `idx_audit_trail_user`: (user_id, created_at DESC)
- `idx_audit_trail_action`: (action, created_at DESC)
- `idx_audit_trail_resource`: (resource, resource_id)
- `idx_audit_trail_result`: (result, created_at DESC)
- `idx_audit_trail_created`: (created_at DESC)
- `idx_audit_trail_details`: GIN index on JSONB details

### Asynchronous Logging

Audit logging is designed to be non-blocking. If audit logging fails, it logs a warning but does not fail the main operation.

## Compliance

### Requirements Met

- **39.1**: All critical operations are logged
- **39.2**: Logs contain User ID, Action, Timestamp, IP Address, and Result
- **39.3**: All sensitive data access is logged
- **39.4**: All configuration changes are logged
- **39.5**: Audit logs are immutable (cannot be modified or deleted)

### Retention Policy

Audit logs should be retained for 7 years to meet compliance requirements. Implement a separate archival process for old logs if needed.

## Testing

Run tests with:

```bash
go test -v ./internal/audit/...
```

Note: Tests require a PostgreSQL database. Set the connection string in the test or skip tests if database is not available.

## Migration

To apply the audit trail migration:

```bash
# Run migration
psql -U postgres -d sms_gateway -f services/backend/migrations/007_audit_trail_table.sql
```

## Security Considerations

1. **Immutability**: Triggers prevent modification/deletion of audit records
2. **Access Control**: Limit access to audit_trail table to authorized personnel only
3. **Encryption**: Consider encrypting sensitive data in the details JSONB field
4. **Monitoring**: Monitor audit log growth and set up alerts for unusual patterns

## Future Enhancements

- [ ] Add audit log export functionality (CSV, JSON)
- [ ] Add audit log search API endpoint
- [ ] Add real-time audit log streaming
- [ ] Add audit log analytics dashboard
- [ ] Add automated compliance reports
