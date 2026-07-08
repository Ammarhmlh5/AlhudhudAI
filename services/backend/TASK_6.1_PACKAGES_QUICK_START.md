# Task 6.1: Packages Table - Quick Start Guide

## What Was Implemented

✅ **Migration File**: `migrations/016_packages_table.sql`
- Creates packages table with full tenant isolation
- Implements Row Level Security (RLS) policies
- Creates composite indexes starting with tenant_id
- Includes helper function to insert default packages
- Includes automatic updated_at trigger

✅ **Default Packages**:
1. **Basic**: $10/month - 1,000 messages
2. **Standard**: $20/month - 3,000 messages
3. **Professional**: $40/month - 8,000 messages
4. **Business**: $50/month - 15,000 messages
5. **Enterprise**: $100/month - Unlimited messages

✅ **Features**:
- Tenant isolation with RLS
- Rollover configuration support
- Overage rate calculation
- Billing cycle support (monthly/yearly)
- Active/inactive status management

## Quick Start

### Step 1: Run the Migration

**Option A - Using Batch Script (Recommended for Windows):**
```batch
cd services\backend
run-packages-migration.bat
```

**Option B - Using Go Migrate:**
```bash
cd services/backend
go run cmd/migrate/main.go
```

**Option C - Using psql:**
```bash
psql -h localhost -p 5433 -U postgres -d al_hudhud_new -f migrations/016_packages_table.sql
```

### Step 2: Insert Default Packages

Connect to your database and run:

```sql
-- Replace with your actual tenant UUID and service ID
SELECT insert_default_packages_for_tenant('your-tenant-uuid-here', 2);
```

**Example:**
```sql
-- For tenant: 123e4567-e89b-12d3-a456-426614174000
-- Service ID: 2 (MOBILE_GATEWAY)
SELECT insert_default_packages_for_tenant('123e4567-e89b-12d3-a456-426614174000', 2);
```

### Step 3: Verify the Migration

Run the test script:
```bash
psql -h localhost -p 5433 -U postgres -d al_hudhud_new -f test_packages_migration.sql
```

Or manually verify:
```sql
-- Set tenant context
SET app.current_tenant = 'your-tenant-uuid-here';

-- Query packages
SELECT 
    id,
    name,
    code,
    price_usd,
    included_messages,
    overage_rate,
    rollover_enabled,
    is_active
FROM packages
ORDER BY display_order;
```

Expected output:
```
 id |     name      |       code        | price_usd | included_messages | overage_rate | rollover_enabled | is_active
----+---------------+-------------------+-----------+-------------------+--------------+------------------+-----------
  1 | Basic         | PKG_BASIC         |     10.00 |              1000 |       0.0100 | t                | t
  2 | Standard      | PKG_STANDARD      |     20.00 |              3000 |       0.0067 | t                | t
  3 | Professional  | PKG_PROFESSIONAL  |     40.00 |              8000 |       0.0050 | t                | t
  4 | Business      | PKG_BUSINESS      |     50.00 |             15000 |       0.0033 | t                | t
  5 | Enterprise    | PKG_ENTERPRISE    |    100.00 |              NULL |       0.0000 | f                | t
```

## Database Configuration

Make sure your `.env` file has the correct database settings:

```env
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=al_hudhud_new
DB_SSLMODE=disable
```

## Testing RLS (Row Level Security)

### Test 1: Verify Tenant Isolation

```sql
-- Set tenant context to tenant A
SET app.current_tenant = 'tenant-a-uuid';
SELECT * FROM packages;  -- Should only see tenant A's packages

-- Set tenant context to tenant B
SET app.current_tenant = 'tenant-b-uuid';
SELECT * FROM packages;  -- Should only see tenant B's packages
```

### Test 2: Verify Insert Policy

```sql
-- Set tenant context
SET app.current_tenant = 'your-tenant-uuid';

-- Try to insert a package for your tenant (should succeed)
INSERT INTO packages (tenant_id, service_id, name, code, price_usd, included_messages)
VALUES (current_setting('app.current_tenant')::uuid, 2, 'Custom', 'PKG_CUSTOM', 30.00, 5000);

-- Try to insert a package for another tenant (should fail)
INSERT INTO packages (tenant_id, service_id, name, code, price_usd, included_messages)
VALUES ('different-tenant-uuid', 2, 'Hacked', 'PKG_HACKED', 1.00, 1000000);
-- Error: new row violates row-level security policy
```

### Test 3: Verify Update Policy

```sql
-- Set tenant context
SET app.current_tenant = 'your-tenant-uuid';

-- Update your own package (should succeed)
UPDATE packages SET price_usd = 12.00 WHERE code = 'PKG_BASIC';

-- Try to update another tenant's package (should fail - no rows affected)
SET app.current_tenant = 'different-tenant-uuid';
UPDATE packages SET price_usd = 1.00 WHERE code = 'PKG_BASIC';
-- 0 rows affected (RLS prevents seeing other tenant's data)
```

## Common Issues & Solutions

### Issue 1: "relation already exists"

**Solution**: This is normal. The migration is idempotent and will skip if already applied.

### Issue 2: "no rows returned" when querying packages

**Solution**: Make sure you've:
1. Set the tenant context: `SET app.current_tenant = 'your-uuid';`
2. Inserted default packages: `SELECT insert_default_packages_for_tenant(...);`

### Issue 3: "function insert_default_packages_for_tenant does not exist"

**Solution**: The migration didn't complete successfully. Re-run the migration.

### Issue 4: Database connection error

**Solution**: Check your `.env` file and ensure PostgreSQL is running:
```bash
# Check if PostgreSQL is running
psql -h localhost -p 5433 -U postgres -d al_hudhud_new -c "SELECT 1;"
```

## Files Created

1. **Migration**: `migrations/016_packages_table.sql`
   - Main migration file with table definition, RLS policies, indexes

2. **Test Script**: `test_packages_migration.sql`
   - Comprehensive verification queries

3. **Run Scripts**:
   - `run-packages-migration.bat` (Windows batch)
   - `run-packages-migration.ps1` (PowerShell)

4. **Documentation**:
   - `migrations/README_PACKAGES.md` (Detailed guide)
   - `TASK_6.1_PACKAGES_QUICK_START.md` (This file)

## Requirements Validated

✅ **Requirement 3.1**: Basic package ($10, 1,000 messages)
✅ **Requirement 3.2**: Standard package ($20, 3,000 messages)
✅ **Requirement 3.3**: Professional package ($40, 8,000 messages)
✅ **Requirement 3.4**: Business package ($50, 15,000 messages)
✅ **Requirement 3.5**: Enterprise package ($100, unlimited)
✅ **Requirement 3.6**: Package creation with name, price, messages, billing cycle
✅ **Requirement 3.7**: Package activation/deactivation support
✅ **Requirement 3.8**: Automatic price per message calculation (overage_rate)
✅ **Requirement 3.9**: NULL value for unlimited messages (Enterprise)

## Next Steps

After completing this task:

1. ✅ Mark Task 6.1 as complete
2. ⏭️ Proceed to **Task 6.2**: Create subscriptions table
3. ⏭️ Proceed to **Task 6.3**: Create rollover_messages table
4. ⏭️ Continue with Phase 2 implementation

## Need Help?

- **Detailed Documentation**: See `migrations/README_PACKAGES.md`
- **Spec Requirements**: See `.kiro/specs/tariff-pricing-system/requirements.md`
- **Design Document**: See `.kiro/specs/tariff-pricing-system/design.md`
- **Task List**: See `.kiro/specs/tariff-pricing-system/tasks.md`

## Summary

Task 6.1 is complete! The packages table has been created with:
- ✅ Full tenant isolation (RLS)
- ✅ Five default packages
- ✅ Rollover configuration support
- ✅ Overage rate calculation
- ✅ Composite indexes for performance
- ✅ Helper functions for easy setup
- ✅ Comprehensive documentation

The migration is ready to be applied to your database. Follow the Quick Start steps above to get started!
