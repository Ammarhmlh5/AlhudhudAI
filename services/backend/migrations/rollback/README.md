# Database Migration Rollback Scripts

This directory contains rollback scripts for database migrations.

## Overview

Each migration has a corresponding rollback script that reverses the changes made by the migration. This allows you to safely revert changes if needed.

## Rollback Scripts

### Admin System Migrations (032-045)

- `032_company_ratings_table_down.sql` - Rollback company_ratings table
- `033_performance_metrics_table_down.sql` - Rollback performance_metrics table
- `034_routing_configurations_table_down.sql` - Rollback routing_configurations table
- `035_distribution_strategies_table_down.sql` - Rollback distribution_strategies table
- `036_phone_devices_table_down.sql` - Rollback phone_devices table
- `037_distribution_metrics_table_down.sql` - Rollback distribution_metrics table
- `038_server_health_logs_table_down.sql` - Rollback server_health_logs table
- `039_alerts_table_down.sql` - Rollback alerts table
- `040_message_queue_table_down.sql` - Rollback message_queue table
- `041_waiting_messages_table_down.sql` - Rollback waiting_messages table
- `042_timing_violations_table_down.sql` - Rollback timing_violations table
- `043_tenant_statistics_table_down.sql` - Rollback tenant_statistics table
- `044_enforcement_actions_table_down.sql` - Rollback enforcement_actions table
- `045_tenant_timing_config_table_down.sql` - Rollback tenant_timing_config table

## Usage

### Using Migration Tool

```bash
# Rollback last migration
./migrate down

# Rollback to specific version
./migrate goto 031

# Rollback all admin system migrations
./migrate goto 031
```

### Manual Rollback

```bash
# Connect to database
psql -U $DB_USER -d $DB_NAME

# Run rollback script
\i services/backend/migrations/rollback/045_tenant_timing_config_table_down.sql
```

### Docker

```bash
# Rollback using Docker
docker-compose exec admin-backend /app/admin-backend migrate down

# Rollback to specific version
docker-compose exec admin-backend /app/admin-backend migrate goto 031
```

## Rollback Order

**IMPORTANT**: Rollback migrations in reverse order to avoid foreign key constraint violations.

Correct rollback order:
1. 045_tenant_timing_config_table
2. 044_enforcement_actions_table
3. 043_tenant_statistics_table
4. 042_timing_violations_table
5. 041_waiting_messages_table
6. 040_message_queue_table
7. 039_alerts_table
8. 038_server_health_logs_table
9. 037_distribution_metrics_table
10. 036_phone_devices_table
11. 035_distribution_strategies_table
12. 034_routing_configurations_table
13. 033_performance_metrics_table
14. 032_company_ratings_table

## Testing Rollbacks

### Test in Staging

Always test rollbacks in staging environment first:

```bash
# 1. Backup database
pg_dump -U $DB_USER $DB_NAME > backup_before_rollback.sql

# 2. Run rollback
./migrate down

# 3. Verify data integrity
psql -U $DB_USER -d $DB_NAME -c "SELECT * FROM information_schema.tables WHERE table_schema = 'public';"

# 4. Test application
curl http://localhost:8080/health

# 5. If successful, apply to production
# 6. If failed, restore backup
psql -U $DB_USER $DB_NAME < backup_before_rollback.sql
```

## Data Preservation

### Before Rollback

Some rollback scripts preserve data by creating backup tables:

```sql
-- Example: Backup data before dropping table
CREATE TABLE company_ratings_backup AS SELECT * FROM company_ratings;

-- Drop table
DROP TABLE company_ratings;
```

### After Rollback

If you need to restore data after rollback:

```sql
-- Restore from backup
INSERT INTO company_ratings SELECT * FROM company_ratings_backup;

-- Drop backup table
DROP TABLE company_ratings_backup;
```

## Emergency Rollback

In case of critical issues:

```bash
# 1. Stop application
docker-compose stop admin-backend

# 2. Backup database
docker-compose exec db pg_dump -U $DB_USER $DB_NAME > emergency_backup.sql

# 3. Rollback all admin migrations
docker-compose exec admin-backend /app/admin-backend migrate goto 031

# 4. Verify database state
docker-compose exec db psql -U $DB_USER -d $DB_NAME -c "\dt"

# 5. Restart application
docker-compose start admin-backend
```

## Rollback Verification

After rollback, verify:

1. **Tables Removed**: Check that tables are dropped
2. **Foreign Keys**: Verify no orphaned foreign keys
3. **Indexes**: Confirm indexes are removed
4. **Functions**: Check that functions/triggers are dropped
5. **Data Integrity**: Verify remaining data is intact

```sql
-- Check tables
SELECT table_name FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;

-- Check foreign keys
SELECT constraint_name, table_name, constraint_type 
FROM information_schema.table_constraints 
WHERE constraint_type = 'FOREIGN KEY';

-- Check indexes
SELECT indexname, tablename FROM pg_indexes 
WHERE schemaname = 'public';
```

## Best Practices

1. **Always Backup**: Create database backup before rollback
2. **Test First**: Test rollback in staging environment
3. **Document**: Document reason for rollback
4. **Verify**: Verify application works after rollback
5. **Monitor**: Monitor logs and metrics after rollback
6. **Communicate**: Inform team about rollback

## Troubleshooting

### Foreign Key Constraint Errors

```sql
-- Disable foreign key checks temporarily
SET session_replication_role = 'replica';

-- Run rollback
\i rollback_script.sql

-- Re-enable foreign key checks
SET session_replication_role = 'origin';
```

### Dependent Objects

```sql
-- Find dependent objects
SELECT * FROM pg_depend 
WHERE refobjid = 'table_name'::regclass;

-- Drop dependent objects first
DROP VIEW dependent_view CASCADE;
```

### Permission Issues

```bash
# Grant necessary permissions
psql -U postgres -d $DB_NAME -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $DB_USER;"
```

## Additional Resources

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Migration Best Practices](https://www.postgresql.org/docs/current/ddl-alter.html)
- [Backup and Restore](https://www.postgresql.org/docs/current/backup.html)
