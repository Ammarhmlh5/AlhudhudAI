# Subscriptions Table Migration

## Overview

This migration creates the `subscriptions` table for tracking customer subscriptions to monthly packages. It implements complete tenant isolation using Row Level Security (RLS) and provides comprehensive tracking of subscription lifecycle, usage, and renewal.

## Migration File

- **File**: `017_subscriptions_table.sql`
- **Version**: 1.0
- **Date**: 2026-02-20

## Table Structure

### Subscriptions Table

The `subscriptions` table tracks:
- Customer subscriptions to monthly packages
- Message usage (used, remaining)
- Overage tracking (messages, cost)
- Renewal settings (auto_renew, next_renewal_date)
- Status tracking (active, grace_period, suspended, cancelled, expired)

### Columns

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL | Primary key |
| `tenant_id` | UUID | Tenant ID for multi-tenant isolation (CRITICAL) |
| `package_id` | INTEGER | References packages(id) |
| `start_date` | DATE | Subscription start date |
| `end_date` | DATE | Subscription end date |
| `messages_used` | INTEGER | Messages used from included quota |
| `messages_remaining` | INTEGER | Messages remaining (NULL = unlimited) |
| `overage_messages` | INTEGER | Messages sent beyond quota |
| `overage_cost` | DECIMAL(10,2) | Total cost of overage messages |
| `status` | VARCHAR(20) | active, grace_period, suspended, cancelled, expired |
| `auto_renew` | BOOLEAN | Auto-renewal enabled |
| `grace_period_days` | INTEGER | Grace period after failed renewal |
| `last_renewed_at` | TIMESTAMP | Last successful renewal |
| `next_renewal_date` | DATE | Next renewal date |
| `renewal_attempts` | INTEGER | Number of renewal attempts |
| `last_renewal_attempt` | TIMESTAMP | Last renewal attempt timestamp |
| `created_at` | TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | Record update timestamp |

## Indexes

All indexes start with `tenant_id` for optimal performance with RLS:

1. `idx_subscriptions_tenant` - Basic tenant lookup
2. `idx_subscriptions_tenant_package` - Tenant + package lookup
3. `idx_subscriptions_tenant_status` - Tenant + status filtering
4. `idx_subscriptions_tenant_renewal` - Tenant + renewal date
5. `idx_subscriptions_tenant_dates` - Tenant + date range queries
6. `idx_subscriptions_tenant_auto_renew` - Tenant + auto-renewal flag
7. `idx_subscriptions_renewal_due` - Finding subscriptions needing renewal (for cron jobs)

## Row Level Security (RLS)

RLS is enabled with four policies:

1. **SELECT Policy**: Users can only see their tenant's subscriptions
2. **INSERT Policy**: Users can only create subscriptions for their tenant
3. **UPDATE Policy**: Users can only update their tenant's subscriptions
4. **DELETE Policy**: Users can only delete their tenant's subscriptions

### Setting Tenant Context

Before querying, set the tenant context:

```sql
SET app.current_tenant = '<tenant-uuid>';
```

## Helper Functions

### 1. calculate_messages_remaining

Calculates remaining messages based on package and usage:

```sql
SELECT calculate_messages_remaining(package_id, messages_used);
```

### 2. is_subscription_active

Checks if a subscription is active:

```sql
SELECT is_subscription_active(subscription_id);
```

### 3. get_active_subscription

Gets the active subscription for a tenant:

```sql
SELECT get_active_subscription('<tenant-uuid>');
```

## Usage Examples

### Creating a Subscription

```sql
-- Set tenant context
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- Create subscription
INSERT INTO subscriptions (
    tenant_id, 
    package_id, 
    start_date, 
    end_date, 
    messages_remaining, 
    next_renewal_date
)
SELECT 
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    p.id,
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '1 month',
    p.included_messages,
    CURRENT_DATE + INTERVAL '1 month'
FROM packages p
WHERE p.code = 'PKG_BASIC' 
  AND p.tenant_id = '123e4567-e89b-12d3-a456-426614174000'::uuid;
```

### Querying Subscriptions

```sql
-- Set tenant context
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- Get active subscription
SELECT 
    s.*,
    p.name AS package_name,
    p.price_usd,
    p.included_messages
FROM subscriptions s
JOIN packages p ON s.package_id = p.id
WHERE s.status = 'active'
  AND s.end_date >= CURRENT_DATE;
```

### Updating Message Usage

```sql
-- Set tenant context
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- Deduct messages
UPDATE subscriptions
SET 
    messages_used = messages_used + 1,
    messages_remaining = CASE 
        WHEN messages_remaining IS NOT NULL 
        THEN GREATEST(0, messages_remaining - 1)
        ELSE NULL
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1
  AND status = 'active';
```

### Tracking Overage

```sql
-- Set tenant context
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- Record overage usage
UPDATE subscriptions s
SET 
    overage_messages = overage_messages + 1,
    overage_cost = overage_cost + p.overage_rate,
    updated_at = CURRENT_TIMESTAMP
FROM packages p
WHERE s.id = 1
  AND s.package_id = p.id
  AND s.status = 'active';
```

### Finding Subscriptions Due for Renewal

```sql
-- Find subscriptions that need renewal (for cron job)
SELECT 
    s.id,
    s.tenant_id,
    s.package_id,
    s.next_renewal_date,
    s.renewal_attempts,
    p.price_usd
FROM subscriptions s
JOIN packages p ON s.package_id = p.id
WHERE s.next_renewal_date <= CURRENT_DATE
  AND s.status IN ('active', 'grace_period')
  AND s.auto_renew = true
ORDER BY s.next_renewal_date;
```

## Running the Migration

### Using PowerShell Script

```powershell
cd services/backend
./run-subscriptions-migration.ps1
```

### Using psql Directly

```bash
psql -h localhost -p 5432 -U postgres -d sms_gateway -f migrations/017_subscriptions_table.sql
```

## Testing the Migration

Run the test script to verify:

```bash
psql -h localhost -p 5432 -U postgres -d sms_gateway -f test_subscriptions_migration.sql
```

## Rollback

If you need to rollback the migration:

```sql
BEGIN;
DROP TRIGGER IF EXISTS trigger_subscriptions_updated_at ON subscriptions;
DROP FUNCTION IF EXISTS update_subscriptions_updated_at();
DROP FUNCTION IF EXISTS calculate_messages_remaining(INTEGER, INTEGER);
DROP FUNCTION IF EXISTS is_subscription_active(INTEGER);
DROP FUNCTION IF EXISTS get_active_subscription(UUID);
DROP TABLE IF EXISTS subscriptions CASCADE;
COMMIT;
```

## Integration with Packages Table

The subscriptions table has a foreign key relationship with the packages table:

```sql
-- Ensure package exists for the tenant before creating subscription
SELECT id FROM packages 
WHERE tenant_id = '<tenant-uuid>' 
  AND code = 'PKG_BASIC';
```

## Status Lifecycle

```
active → grace_period → suspended → cancelled
   ↓           ↓
expired    expired
```

- **active**: Subscription is active and within the subscription period
- **grace_period**: Renewal failed, but within grace period
- **suspended**: Grace period expired, subscription suspended
- **cancelled**: User cancelled the subscription
- **expired**: Subscription period ended without renewal

## Requirements Validated

This migration validates the following requirements:

- **4.3**: Record subscription start and end dates
- **4.4**: Track available messages based on package
- **4.5**: Deduct from message counter when sending
- **4.6**: Display used and remaining messages
- **4.7**: Support auto-renewal
- **4.8**: Auto-renew when period ends (if enabled)
- **4.9**: Send notification and grant grace period on renewal failure

## Next Steps

After running this migration:

1. ✅ Subscriptions table created with tenant isolation
2. ⏭️ Create rollover_messages table (Task 6.3)
3. ⏭️ Develop Subscription Manager (Task 7)
4. ⏭️ Implement auto-renewal service (Task 12)

## Related Files

- Migration: `migrations/017_subscriptions_table.sql`
- Test Script: `test_subscriptions_migration.sql`
- Run Script: `run-subscriptions-migration.ps1`
- Previous Migration: `migrations/016_packages_table.sql`
- Design Document: `.kiro/specs/tariff-pricing-system/design.md`
- Requirements: `.kiro/specs/tariff-pricing-system/requirements.md`
