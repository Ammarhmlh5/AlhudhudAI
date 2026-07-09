# Database Migrations Guide

This document provides information about database migrations, including history, rollback procedures, and data preservation notes.

## Migration History

### Phase 1: Critical Security Fixes

| Migration | Description | Status |
|-----------|-------------|--------|
| 050 | Create RBAC tables (role_permissions, user_roles) | ✅ Applied |
| 051 | Add NOT NULL constraints to critical columns | ✅ Applied |
| 052 | Unify financial precision to DECIMAL(12,4) | ✅ Applied |
| 053 | Enable Row Level Security (RLS) on 6 tables | ✅ Applied |
| 054 | Create tenant_id composite indexes | ✅ Applied |
| 055 | Add CHECK constraints for financial columns | ✅ Applied |
| 056 | Create updated_at auto-update triggers | ✅ Applied |

### Phase 2: Application Updates

| Migration | Description | Status |
|-----------|-------------|--------|
| 057 | Unify UUID generation | ⏸ Pending |
| 058 | Add user ID tracking for rate limiting | ⏸ Pending |

## Running Migrations

### Using the Migration Runner

```bash
# Run all pending migrations
go run cmd/run-migration/main.go

# Run specific migration
go run cmd/run-migration/main.go --migration=050
```

### Manual Execution

```bash
# Connect to database
psql -h localhost -U alhudhud_user -d alhudhud

# Run migration file
\i services/backend/migrations/050_create_rbac_tables.sql
```

## Rollback Procedures

### Rollback Migration 050 (RBAC Tables)

```sql
-- Drop foreign key constraints first
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS fk_role_permissions_role;
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS fk_role_permissions_permission;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_user;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_role;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS fk_user_roles_tenant;

-- Drop tables
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
```

### Rollback Migration 051 (NOT NULL Constraints)

```sql
ALTER TABLE messages ALTER COLUMN channel_id DROP NOT NULL;
ALTER TABLE messages ALTER COLUMN device_id DROP NOT NULL;
ALTER TABLE dispatch_channels ALTER COLUMN name DROP NOT NULL;
ALTER TABLE dispatch_channels ALTER COLUMN type DROP NOT NULL;
ALTER TABLE accounting_transactions ALTER COLUMN type DROP NOT NULL;
ALTER TABLE accounting_transactions ALTER COLUMN status DROP NOT NULL;
```

### Rollback Migration 052 (Financial Precision)

```sql
-- Note: This may cause data loss if values exceed DECIMAL(12,2)
ALTER TABLE messages ALTER COLUMN cost TYPE DECIMAL(12,2);
```

### Rollback Migration 053 (RLS)

```sql
ALTER TABLE messages DISABLE ROW LEVEL SECURITY;
ALTER TABLE balance_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE devices DISABLE ROW LEVEL SECURITY;
ALTER TABLE channels DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE audit_trail DISABLE ROW LEVEL SECURITY;

-- Drop policies
DROP POLICY IF EXISTS messages_tenant_policy ON messages;
DROP POLICY IF EXISTS balance_history_tenant_policy ON balance_history;
DROP POLICY IF EXISTS devices_tenant_policy ON devices;
DROP POLICY IF EXISTS channels_tenant_policy ON channels;
DROP POLICY IF EXISTS users_tenant_policy ON users;
DROP POLICY IF EXISTS audit_trail_tenant_policy ON audit_trail;
```

### Rollback Migration 054 (Indexes)

```sql
DROP INDEX IF EXISTS idx_messages_tenant_status_created;
DROP INDEX IF EXISTS idx_balance_history_tenant_type_created;
DROP INDEX IF EXISTS idx_accounting_transactions_tenant_status;
DROP INDEX IF EXISTS idx_subscriptions_tenant_expires;
```

### Rollback Migration 055 (CHECK Constraints)

```sql
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_cost_check;
ALTER TABLE balance_history DROP CONSTRAINT IF EXISTS balance_history_amount_check;
ALTER TABLE accounting_transactions DROP CONSTRAINT IF EXISTS accounting_transactions_amount_check;
```

### Rollback Migration 056 (Triggers)

```sql
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_channels_updated_at ON channels;
DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;
DROP TRIGGER IF EXISTS update_messages_updated_at ON messages;

DROP FUNCTION IF EXISTS update_updated_at_column();
```

## Data Preservation Notes

### Before Running Migrations

1. **Backup your database**
   ```bash
   pg_dump -h localhost -U alhudhud_user -d alhudhud > backup_$(date +%Y%m%d).sql
   ```

2. **Test migrations on staging**
   - Always test migrations on a staging environment first
   - Verify data integrity after migration
   - Check application functionality

3. **Check disk space**
   - Migrations may require additional disk space
   - Ensure at least 2x current database size free

### During Migration Execution

- **Migration 050**: No data loss, creates new tables
- **Migration 051**: May fail if existing data contains NULL values
- **Migration 052**: May truncate decimal values beyond 2 places
- **Migration 053**: No data loss, enables access control
- **Migration 054**: No data loss, improves performance
- **Migration 055**: May fail if existing data violates constraints
- **Migration 056**: No data loss, adds automatic timestamps

### After Migration

1. Verify all constraints are applied
2. Check RLS policies are working
3. Test application functionality
4. Monitor application logs for errors
5. Run database health checks

## Troubleshooting

### Migration Fails with Constraint Violation

```sql
-- Check existing NULL values
SELECT COUNT(*) FROM messages WHERE channel_id IS NULL;

-- Update NULL values before migration
UPDATE messages SET channel_id = '00000000-0000-0000-0000-000000000000' WHERE channel_id IS NULL;
```

### RLS Blocks Access After Migration

```sql
-- Check RLS status
SELECT relname, relrowsecurity FROM pg_class WHERE relname IN ('messages', 'balance_history');

-- Verify policy exists
SELECT * FROM pg_policies WHERE tablename = 'messages';

-- Test with superuser
SET ROLE postgres;
-- Test access
```

### Performance Issues After Index Creation

```sql
-- Check if indexes were created
SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'messages';

-- Analyze tables to update statistics
ANALYZE messages;
ANALYZE balance_history;
```

## Best Practices

1. **Always backup before migrations**
2. **Test migrations on staging first**
3. **Run migrations during low-traffic periods**
4. **Monitor database performance after migrations**
5. **Document any manual steps required**
6. **Keep rollback scripts up to date**
7. **Verify data integrity after migrations**
8. **Test application functionality end-to-end