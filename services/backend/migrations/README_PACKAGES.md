# Packages Table Migration Guide

## Overview

Migration `016_packages_table.sql` creates the packages table for the Tariff Pricing System. This table stores monthly subscription packages with different pricing tiers and message quotas.

## Features

### 1. Full Tenant Isolation
- **tenant_id column**: Every package belongs to a specific tenant
- **Row Level Security (RLS)**: Automatic filtering based on `app.current_tenant` setting
- **Composite indexes**: All indexes start with `tenant_id` for optimal performance
- **RLS Policies**: SELECT, INSERT, UPDATE, DELETE policies enforce tenant isolation

### 2. Default Packages

The migration includes five default packages:

| Package | Price | Messages | Overage Rate | Rollover |
|---------|-------|----------|--------------|----------|
| Basic | $10 | 1,000 | $0.0100 | Enabled |
| Standard | $20 | 3,000 | $0.0067 | Enabled |
| Professional | $40 | 8,000 | $0.0050 | Enabled |
| Business | $50 | 15,000 | $0.0033 | Enabled |
| Enterprise | $100 | Unlimited | $0.0000 | Disabled |

### 3. Rollover Configuration

Each package can be configured with:
- **rollover_enabled**: Whether unused messages can be rolled over
- **rollover_limit**: Maximum messages that can be rolled over (NULL = unlimited)
- **rollover_expiry_months**: How long rolled over messages remain valid (NULL = never expire)

### 4. Overage Support

- **overage_rate**: Price per additional message beyond the included quota
- Calculated automatically based on package price and included messages
- Enterprise package has 0 overage rate (unlimited messages)

## Running the Migration

### Option 1: Using the Batch Script (Windows)

```batch
cd services\backend
run-packages-migration.bat
```

### Option 2: Using PowerShell Script

```powershell
cd services/backend
./run-packages-migration.ps1
```

### Option 3: Using Go Migrate Tool Directly

```bash
cd services/backend
go run cmd/migrate/main.go
```

### Option 4: Manual SQL Execution

```bash
psql -h localhost -p 5433 -U postgres -d al_hudhud_new -f migrations/016_packages_table.sql
```

## Post-Migration Steps

### 1. Insert Default Packages for Your Tenant

```sql
-- Replace <tenant-uuid> with your actual tenant UUID
-- Replace <service-id> with the MOBILE_GATEWAY service ID (typically 2)
SELECT insert_default_packages_for_tenant('<tenant-uuid>', <service-id>);
```

Example:
```sql
SELECT insert_default_packages_for_tenant('123e4567-e89b-12d3-a456-426614174000', 2);
```

### 2. Verify the Migration

Run the test script:
```bash
psql -h localhost -p 5433 -U postgres -d al_hudhud_new -f test_packages_migration.sql
```

Or manually check:
```sql
-- Check if table exists
SELECT * FROM packages;

-- Check RLS policies
SELECT * FROM pg_policies WHERE tablename = 'packages';

-- Check indexes
SELECT * FROM pg_indexes WHERE tablename = 'packages';
```

### 3. Query Packages (with Tenant Context)

```sql
-- Set the tenant context
SET app.current_tenant = '<tenant-uuid>';

-- Query packages
SELECT 
    id,
    name,
    code,
    price_usd,
    included_messages,
    overage_rate,
    is_unlimited,
    rollover_enabled,
    is_active
FROM packages
WHERE is_active = true
ORDER BY display_order;
```

## Table Schema

```sql
CREATE TABLE packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    service_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    price_usd DECIMAL(10, 2) NOT NULL,
    included_messages INTEGER,  -- NULL = unlimited
    overage_rate DECIMAL(10, 4),
    is_unlimited BOOLEAN DEFAULT false,
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    
    -- Rollover settings
    rollover_enabled BOOLEAN DEFAULT false,
    rollover_limit INTEGER,
    rollover_expiry_months INTEGER,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Indexes

- `idx_packages_tenant_id`: Fast lookup by tenant
- `idx_packages_tenant_service`: Fast lookup by tenant and service
- `idx_packages_tenant_active`: Fast lookup of active packages per tenant
- `idx_packages_tenant_order`: Fast lookup by display order per tenant
- `idx_packages_tenant_code_unique`: Ensures code uniqueness per tenant

## RLS Policies

1. **tenant_isolation_select_policy**: Users can only SELECT their tenant's packages
2. **tenant_isolation_insert_policy**: Users can only INSERT packages for their tenant
3. **tenant_isolation_update_policy**: Users can only UPDATE their tenant's packages
4. **tenant_isolation_delete_policy**: Users can only DELETE their tenant's packages

## Helper Functions

### insert_default_packages_for_tenant(p_tenant_id UUID, p_service_id INTEGER)

Inserts the five default packages for a specific tenant.

**Parameters:**
- `p_tenant_id`: The tenant UUID
- `p_service_id`: The service ID (typically 2 for MOBILE_GATEWAY)

**Usage:**
```sql
SELECT insert_default_packages_for_tenant('123e4567-e89b-12d3-a456-426614174000', 2);
```

## Triggers

### trigger_packages_updated_at

Automatically updates the `updated_at` column whenever a package is modified.

## Constraints

1. **check_price_positive**: Ensures price_usd > 0
2. **check_messages_positive**: Ensures included_messages > 0 (if not NULL)
3. **check_billing_cycle**: Ensures billing_cycle is 'monthly' or 'yearly'

## Troubleshooting

### Migration Fails with "relation already exists"

The migration is idempotent and will skip if the table already exists. This is normal behavior.

### Cannot query packages

Make sure you've set the tenant context:
```sql
SET app.current_tenant = '<your-tenant-uuid>';
```

### No packages returned

1. Check if default packages were inserted:
   ```sql
   SELECT insert_default_packages_for_tenant('<tenant-uuid>', <service-id>);
   ```

2. Verify RLS is working:
   ```sql
   -- Disable RLS temporarily (as superuser)
   ALTER TABLE packages DISABLE ROW LEVEL SECURITY;
   SELECT * FROM packages;
   ALTER TABLE packages ENABLE ROW LEVEL SECURITY;
   ```

### Database connection error

Check your `.env` file settings:
```
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=al_hudhud_new
DB_SSLMODE=disable
```

## Requirements Validated

This migration validates the following requirements from the spec:

- **3.1**: Basic package ($10, 1,000 messages)
- **3.2**: Standard package ($20, 3,000 messages)
- **3.3**: Professional package ($40, 8,000 messages)
- **3.4**: Business package ($50, 15,000 messages)
- **3.5**: Enterprise package ($100, unlimited messages)
- **3.6**: Package creation and management
- **3.7**: Package activation/deactivation
- **3.8**: Automatic price per message calculation
- **3.9**: NULL value for unlimited messages

## Next Steps

After successfully applying this migration:

1. ✅ Verify the migration with `test_packages_migration.sql`
2. ✅ Insert default packages for your tenant(s)
3. ⏭️ Proceed to Task 6.2: Create subscriptions table
4. ⏭️ Proceed to Task 6.3: Create rollover_messages table

## Related Files

- Migration: `migrations/016_packages_table.sql`
- Test Script: `test_packages_migration.sql`
- Run Script (Batch): `run-packages-migration.bat`
- Run Script (PowerShell): `run-packages-migration.ps1`
- Spec: `.kiro/specs/tariff-pricing-system/requirements.md`
- Design: `.kiro/specs/tariff-pricing-system/design.md`
- Tasks: `.kiro/specs/tariff-pricing-system/tasks.md`
