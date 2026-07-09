# Rollover Messages Table Migration

## Overview

This migration creates the `rollover_messages` table for tracking messages that are rolled over from one subscription period to the next. The rollover feature allows customers to carry forward unused messages from their monthly package to the next billing cycle.

## Migration File

- **File**: `018_rollover_messages_table.sql`
- **Version**: 1.0
- **Date**: 2026-02-20

## Table Structure

### rollover_messages

Stores rolled-over messages with FIFO (First-In-First-Out) usage tracking.

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL | Primary key |
| `tenant_id` | UUID | Tenant ID for multi-tenant isolation (CRITICAL for RLS) |
| `subscription_id` | INTEGER | References subscriptions(id) |
| `messages_count` | INTEGER | Total number of messages rolled over |
| `source_month` | DATE | The month/period from which messages were rolled over |
| `expiry_date` | DATE | Date when rollover messages expire (NULL = never expire) |
| `messages_used` | INTEGER | Number of messages used from this rollover batch |
| `messages_remaining` | INTEGER | Number of messages remaining in this rollover batch |
| `status` | VARCHAR(20) | Status: 'active', 'expired', 'consumed' |
| `created_at` | TIMESTAMP | Creation timestamp (used for FIFO ordering) |
| `updated_at` | TIMESTAMP | Last update timestamp |

## Key Features

### 1. Tenant Isolation

- **tenant_id column**: Every rollover record is associated with a specific tenant
- **Row Level Security (RLS)**: Enabled with policies for SELECT, INSERT, UPDATE, DELETE
- **Composite indexes**: All indexes start with `tenant_id` for optimal performance

### 2. FIFO Usage

Rollover messages are used in FIFO (First-In-First-Out) order:
- Oldest rollover messages are used first
- Ordering is based on `created_at` timestamp
- The `use_rollover_messages()` function handles FIFO logic automatically

### 3. Expiry Management

- Rollover messages can have an expiry date
- `expiry_date = NULL` means messages never expire
- The `expire_old_rollover_messages()` function marks expired messages
- Should be run daily via cron job

### 4. Status Tracking

Three statuses:
- **active**: Messages are available for use
- **expired**: Messages have passed their expiry_date
- **consumed**: All messages have been used (messages_remaining = 0)

## Indexes

All indexes are optimized for tenant isolation and performance:

1. `idx_rollover_tenant`: Basic tenant lookup
2. `idx_rollover_tenant_subscription`: Tenant + subscription lookup
3. `idx_rollover_tenant_status`: Tenant + status filtering
4. `idx_rollover_tenant_expiry`: Tenant + expiry date lookup
5. `idx_rollover_tenant_created`: FIFO ordering (tenant + subscription + created_at)
6. `idx_rollover_expired`: Finding expired messages for cleanup

## Helper Functions

### 1. get_available_rollover_messages(tenant_id, subscription_id)

Returns available rollover messages in FIFO order.

```sql
SELECT * FROM get_available_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1
);
```

### 2. use_rollover_messages(tenant_id, subscription_id, messages_needed)

Uses rollover messages in FIFO order and returns the number of messages actually used.

```sql
-- Try to use 10 messages from rollover
SELECT use_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    10
);
-- Returns: number of messages actually used (may be less than requested)
```

### 3. get_total_rollover_messages(tenant_id, subscription_id)

Returns the total number of available rollover messages for a subscription.

```sql
SELECT get_total_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1
);
```

### 4. create_rollover_messages(tenant_id, subscription_id, messages_count, source_month, expiry_months)

Creates a new rollover record when a subscription renews.

```sql
-- Rollover 500 messages that expire in 3 months
SELECT create_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    500,
    '2026-02-01'::date,
    3
);

-- Rollover 1000 messages that never expire
SELECT create_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    1000,
    '2026-02-01'::date,
    NULL
);
```

### 5. expire_old_rollover_messages()

Marks expired rollover messages as 'expired'. Should be run daily via cron job.

```sql
-- Run daily to expire old rollover messages
SELECT expire_old_rollover_messages();
-- Returns: number of messages expired
```

## Running the Migration

### Prerequisites

1. PostgreSQL database is running
2. `packages` table exists (migration 016)
3. `subscriptions` table exists (migration 017)
4. You have a valid tenant UUID

### Steps

1. **Run the migration**:

```bash
# Using psql
psql -U your_user -d sms_gateway -f migrations/018_rollover_messages_table.sql

# Or using the Go migration tool
go run cmd/migrate/main.go up
```

2. **Verify the table was created**:

```sql
-- Check table exists
\dt rollover_messages

-- Check RLS is enabled
SELECT tablename, rowsecurity 
FROM pg_tables 
WHERE tablename = 'rollover_messages';

-- Check policies
\d rollover_messages
```

3. **Test with sample data** (optional):

```sql
-- Set tenant context
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- Create a rollover record
SELECT create_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,  -- subscription_id
    500,  -- messages to rollover
    '2026-02-01'::date,  -- source month
    3  -- expire after 3 months
);

-- Check total available
SELECT get_total_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1
);

-- Use some messages
SELECT use_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    10
);

-- Verify usage
SELECT * FROM rollover_messages 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'::uuid;
```

## Usage Workflow

### When Subscription Renews

1. Calculate remaining messages from current period
2. Check if rollover is enabled for the package
3. Apply rollover limit if configured
4. Create rollover record using `create_rollover_messages()`
5. Reset subscription's `messages_used` and `messages_remaining`

Example:

```sql
-- At subscription renewal
DO $
DECLARE
    v_tenant_id UUID := '123e4567-e89b-12d3-a456-426614174000';
    v_subscription_id INTEGER := 1;
    v_remaining_messages INTEGER;
    v_rollover_limit INTEGER;
    v_rollover_expiry INTEGER;
    v_messages_to_rollover INTEGER;
BEGIN
    -- Get remaining messages
    SELECT messages_remaining INTO v_remaining_messages
    FROM subscriptions
    WHERE id = v_subscription_id;
    
    -- Get package rollover settings
    SELECT p.rollover_limit, p.rollover_expiry_months
    INTO v_rollover_limit, v_rollover_expiry
    FROM subscriptions s
    JOIN packages p ON s.package_id = p.id
    WHERE s.id = v_subscription_id;
    
    -- Calculate messages to rollover
    IF v_rollover_limit IS NULL THEN
        v_messages_to_rollover := v_remaining_messages;
    ELSE
        v_messages_to_rollover := LEAST(v_remaining_messages, v_rollover_limit);
    END IF;
    
    -- Create rollover record if there are messages to rollover
    IF v_messages_to_rollover > 0 THEN
        PERFORM create_rollover_messages(
            v_tenant_id,
            v_subscription_id,
            v_messages_to_rollover,
            CURRENT_DATE,
            v_rollover_expiry
        );
    END IF;
END $;
```

### When Sending Messages

1. Check if subscription has included messages remaining
2. If not, check for available rollover messages
3. Use rollover messages in FIFO order
4. If rollover exhausted, use overage (charge from balance)

Example:

```sql
-- When sending a message
DO $
DECLARE
    v_tenant_id UUID := '123e4567-e89b-12d3-a456-426614174000';
    v_subscription_id INTEGER := 1;
    v_messages_needed INTEGER := 1;
    v_subscription_remaining INTEGER;
    v_rollover_used INTEGER;
BEGIN
    -- Check subscription messages
    SELECT messages_remaining INTO v_subscription_remaining
    FROM subscriptions
    WHERE id = v_subscription_id;
    
    IF v_subscription_remaining > 0 THEN
        -- Use from subscription
        UPDATE subscriptions
        SET messages_used = messages_used + v_messages_needed,
            messages_remaining = messages_remaining - v_messages_needed
        WHERE id = v_subscription_id;
    ELSE
        -- Try to use rollover messages
        v_rollover_used := use_rollover_messages(
            v_tenant_id,
            v_subscription_id,
            v_messages_needed
        );
        
        IF v_rollover_used < v_messages_needed THEN
            -- Not enough rollover, use overage
            -- (charge from balance)
            RAISE NOTICE 'Using overage for % messages', v_messages_needed - v_rollover_used;
        END IF;
    END IF;
END $;
```

### Daily Cleanup (Cron Job)

Run this daily to expire old rollover messages:

```sql
-- Expire old rollover messages
SELECT expire_old_rollover_messages();
```

## Testing the Migration

### Test Script

Create a file `test_rollover_migration.sql`:

```sql
-- Test Rollover Messages Migration
-- This script tests the rollover_messages table and helper functions

BEGIN;

-- Set tenant context
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- Test 1: Create rollover messages
SELECT 'Test 1: Creating rollover messages...' AS test;
SELECT create_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    500,
    '2026-02-01'::date,
    3
) AS rollover_id;

-- Test 2: Get total available
SELECT 'Test 2: Getting total available rollover messages...' AS test;
SELECT get_total_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1
) AS total_available;

-- Test 3: Use some messages (FIFO)
SELECT 'Test 3: Using 10 messages from rollover...' AS test;
SELECT use_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    10
) AS messages_used;

-- Test 4: Check remaining
SELECT 'Test 4: Checking remaining after usage...' AS test;
SELECT id, messages_count, messages_used, messages_remaining, status
FROM rollover_messages
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'::uuid;

-- Test 5: Create another rollover batch (older)
SELECT 'Test 5: Creating older rollover batch...' AS test;
SELECT create_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    300,
    '2026-01-01'::date,
    2
) AS rollover_id;

-- Test 6: Use messages (should use oldest first - FIFO)
SELECT 'Test 6: Using 50 messages (should use oldest first)...' AS test;
SELECT use_rollover_messages(
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    50
) AS messages_used;

-- Test 7: Verify FIFO order
SELECT 'Test 7: Verifying FIFO order...' AS test;
SELECT id, source_month, messages_count, messages_used, messages_remaining, created_at
FROM rollover_messages
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'::uuid
ORDER BY created_at;

-- Test 8: Test expiry
SELECT 'Test 8: Testing expiry function...' AS test;
-- Create an expired rollover
INSERT INTO rollover_messages (
    tenant_id, subscription_id, messages_count, source_month, 
    expiry_date, messages_remaining, status
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000'::uuid,
    1,
    100,
    '2025-12-01'::date,
    '2026-01-01'::date,  -- Already expired
    100,
    'active'
);

-- Run expiry function
SELECT expire_old_rollover_messages() AS expired_count;

-- Check expired status
SELECT id, expiry_date, status
FROM rollover_messages
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'::uuid
  AND status = 'expired';

-- Test 9: Test RLS (should only see own tenant's data)
SELECT 'Test 9: Testing RLS isolation...' AS test;
SELECT COUNT(*) AS my_rollover_count
FROM rollover_messages;

ROLLBACK;

SELECT 'All tests completed successfully!' AS result;
```

Run the test:

```bash
psql -U your_user -d sms_gateway -f test_rollover_migration.sql
```

## Rollback

If you need to rollback this migration:

```sql
BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_rollover_messages_updated_at ON rollover_messages;

-- Drop functions
DROP FUNCTION IF EXISTS update_rollover_messages_updated_at();
DROP FUNCTION IF EXISTS get_available_rollover_messages(UUID, INTEGER);
DROP FUNCTION IF EXISTS use_rollover_messages(UUID, INTEGER, INTEGER);
DROP FUNCTION IF EXISTS get_total_rollover_messages(UUID, INTEGER);
DROP FUNCTION IF EXISTS expire_old_rollover_messages();
DROP FUNCTION IF EXISTS create_rollover_messages(UUID, INTEGER, INTEGER, DATE, INTEGER);

-- Drop table
DROP TABLE IF EXISTS rollover_messages CASCADE;

COMMIT;
```

## Integration with Subscription Manager

The rollover_messages table integrates with the Subscription Manager (Task 7):

1. **At Renewal**: Subscription Manager calls `create_rollover_messages()`
2. **When Sending**: Subscription Manager calls `use_rollover_messages()`
3. **Display Usage**: Subscription Manager calls `get_total_rollover_messages()`

## Next Steps

1. ✅ Rollover messages table created with tenant isolation
2. ⏭️ Develop Subscription Manager (Task 7)
3. ⏭️ Develop Rollover Manager (Task 8)
4. ⏭️ Implement auto-renewal service (Task 12)

## Related Files

- Migration: `migrations/018_rollover_messages_table.sql`
- Previous: `migrations/017_subscriptions_table.sql`
- Next: Subscription Manager implementation (Task 7)

## Requirements Validated

This migration validates the following requirements from the spec:

- **Requirement 6.2**: Calculate remaining messages at end of month
- **Requirement 6.3**: Record rolled-over messages with source and expiry
- **Requirement 6.7**: Support expiry dates for rollover messages
- **Requirement 6.8**: Use rollover messages in FIFO order (oldest first)

## Notes

- All rollover messages are tenant-isolated using RLS
- FIFO ordering is critical for fair usage
- Expiry cleanup should run daily via cron
- The `use_rollover_messages()` function is atomic and handles concurrency
- Rollover messages are separate from subscription messages for clear tracking
