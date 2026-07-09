# Task 6.3: Rollover Messages Table - Quick Start Guide

## Overview

This guide helps you quickly implement Task 6.3: Create the `rollover_messages` table for tracking messages rolled over from one subscription period to the next.

## What Was Created

### 1. Migration File
- **File**: `migrations/018_rollover_messages_table.sql`
- **Purpose**: Creates rollover_messages table with tenant isolation and FIFO tracking

### 2. Documentation
- **File**: `migrations/README_ROLLOVER.md`
- **Purpose**: Comprehensive documentation with usage examples

### 3. Test Script
- **File**: `test_rollover_migration.sql`
- **Purpose**: Tests all functionality including FIFO, expiry, and RLS

### 4. PowerShell Runner
- **File**: `run-rollover-migration.ps1`
- **Purpose**: Easy migration execution with verification

## Quick Start (5 Minutes)

### Step 1: Run the Migration

```powershell
# Navigate to backend directory
cd services/backend

# Run the migration script
.\run-rollover-migration.ps1
```

The script will:
1. Prompt for database password
2. Run the migration
3. Verify table creation
4. Check RLS and helper functions
5. Optionally run tests

### Step 2: Verify Installation

```sql
-- Connect to database
psql -U postgres -d sms_gateway

-- Check table exists
\dt rollover_messages

-- Check RLS is enabled
SELECT tablename, rowsecurity 
FROM pg_tables 
WHERE tablename = 'rollover_messages';

-- List helper functions
\df *rollover*
```

### Step 3: Test Basic Functionality

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

-- Verify
SELECT * FROM rollover_messages 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'::uuid;
```

## Key Features

### 1. FIFO (First-In-First-Out) Usage

Rollover messages are automatically used in FIFO order:

```sql
-- Oldest rollover messages are used first
SELECT use_rollover_messages(tenant_id, subscription_id, messages_needed);
```

### 2. Expiry Management

Rollover messages can expire:

```sql
-- Create with expiry (3 months)
SELECT create_rollover_messages(
    tenant_id, subscription_id, 500, '2026-02-01'::date, 3
);

-- Create without expiry (never expires)
SELECT create_rollover_messages(
    tenant_id, subscription_id, 500, '2026-02-01'::date, NULL
);

-- Run daily to expire old messages
SELECT expire_old_rollover_messages();
```

### 3. Tenant Isolation

All data is isolated by tenant using RLS:

```sql
-- Set tenant context (required)
SET app.current_tenant = '<your-tenant-uuid>';

-- All queries automatically filtered by tenant_id
SELECT * FROM rollover_messages;
```

## Helper Functions Reference

### create_rollover_messages()

Creates a new rollover record when subscription renews.

```sql
SELECT create_rollover_messages(
    tenant_id UUID,
    subscription_id INTEGER,
    messages_count INTEGER,
    source_month DATE,
    expiry_months INTEGER  -- NULL = never expire
);
```

### use_rollover_messages()

Uses rollover messages in FIFO order.

```sql
SELECT use_rollover_messages(
    tenant_id UUID,
    subscription_id INTEGER,
    messages_needed INTEGER
);
-- Returns: number of messages actually used
```

### get_total_rollover_messages()

Gets total available rollover messages.

```sql
SELECT get_total_rollover_messages(
    tenant_id UUID,
    subscription_id INTEGER
);
-- Returns: total available messages
```

### get_available_rollover_messages()

Gets detailed list of available rollover batches (FIFO order).

```sql
SELECT * FROM get_available_rollover_messages(
    tenant_id UUID,
    subscription_id INTEGER
);
-- Returns: id, messages_remaining, created_at (ordered oldest first)
```

### expire_old_rollover_messages()

Marks expired rollover messages as 'expired'.

```sql
SELECT expire_old_rollover_messages();
-- Returns: number of messages expired
-- Run daily via cron job
```

## Integration Workflow

### When Subscription Renews

```sql
-- 1. Get remaining messages from current period
SELECT messages_remaining FROM subscriptions WHERE id = ?;

-- 2. Check package rollover settings
SELECT rollover_enabled, rollover_limit, rollover_expiry_months 
FROM packages WHERE id = ?;

-- 3. Create rollover record
SELECT create_rollover_messages(
    tenant_id,
    subscription_id,
    messages_to_rollover,
    CURRENT_DATE,
    rollover_expiry_months
);

-- 4. Reset subscription counters
UPDATE subscriptions 
SET messages_used = 0,
    messages_remaining = included_messages
WHERE id = ?;
```

### When Sending Messages

```sql
-- 1. Try to use subscription messages first
IF subscription.messages_remaining > 0 THEN
    -- Use from subscription
    UPDATE subscriptions SET messages_used = messages_used + 1;
ELSE
    -- 2. Try to use rollover messages
    messages_used := use_rollover_messages(tenant_id, subscription_id, 1);
    
    IF messages_used = 0 THEN
        -- 3. Use overage (charge from balance)
        -- Handle overage logic
    END IF;
END IF;
```

## Table Structure

```sql
CREATE TABLE rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    subscription_id INTEGER NOT NULL,
    messages_count INTEGER NOT NULL,
    source_month DATE NOT NULL,
    expiry_date DATE,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Status Values

- **active**: Messages are available for use
- **expired**: Messages have passed their expiry_date
- **consumed**: All messages have been used (messages_remaining = 0)

## Indexes

All indexes are optimized for tenant isolation:

1. `idx_rollover_tenant`: Basic tenant lookup
2. `idx_rollover_tenant_subscription`: Tenant + subscription
3. `idx_rollover_tenant_status`: Tenant + status filtering
4. `idx_rollover_tenant_expiry`: Tenant + expiry date
5. `idx_rollover_tenant_created`: FIFO ordering (tenant + subscription + created_at)
6. `idx_rollover_expired`: Finding expired messages for cleanup

## Testing

### Run Full Test Suite

```powershell
# Run all tests
psql -U postgres -d sms_gateway -f test_rollover_migration.sql
```

### Manual Testing

```sql
-- Test FIFO ordering
BEGIN;
SET app.current_tenant = '<tenant-uuid>';

-- Create two rollover batches
SELECT create_rollover_messages('<tenant-uuid>'::uuid, 1, 100, '2026-01-01'::date, 3);
SELECT pg_sleep(1);  -- Wait 1 second
SELECT create_rollover_messages('<tenant-uuid>'::uuid, 1, 200, '2026-02-01'::date, 3);

-- Use 50 messages (should use from oldest batch first)
SELECT use_rollover_messages('<tenant-uuid>'::uuid, 1, 50);

-- Verify FIFO
SELECT id, messages_count, messages_used, messages_remaining, created_at
FROM rollover_messages
WHERE tenant_id = '<tenant-uuid>'::uuid
ORDER BY created_at;

ROLLBACK;
```

## Common Issues & Solutions

### Issue 1: RLS Policy Error

**Error**: `new row violates row-level security policy`

**Solution**: Set tenant context before operations:
```sql
SET app.current_tenant = '<your-tenant-uuid>';
```

### Issue 2: No Messages Used

**Problem**: `use_rollover_messages()` returns 0

**Causes**:
1. No active rollover messages
2. All rollover messages expired
3. Wrong tenant_id or subscription_id

**Solution**: Check available messages:
```sql
SELECT * FROM get_available_rollover_messages(tenant_id, subscription_id);
```

### Issue 3: FIFO Not Working

**Problem**: Newer messages used before older ones

**Cause**: `created_at` timestamps are the same

**Solution**: Ensure proper timing when creating rollover records, or use explicit timestamps:
```sql
INSERT INTO rollover_messages (..., created_at) 
VALUES (..., CURRENT_TIMESTAMP - INTERVAL '1 day');
```

## Performance Considerations

### Indexes

All indexes start with `tenant_id` for optimal performance in multi-tenant environment.

### Cleanup

Run `expire_old_rollover_messages()` daily to keep table size manageable:

```sql
-- Add to cron job
SELECT expire_old_rollover_messages();
```

### Partitioning (Future)

For high-volume systems, consider partitioning by `created_at`:

```sql
-- Example: Monthly partitions
CREATE TABLE rollover_messages_2026_02 
PARTITION OF rollover_messages
FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
```

## Requirements Validated

This implementation validates:

- ✅ **Requirement 6.2**: Calculate remaining messages at end of month
- ✅ **Requirement 6.3**: Record rolled-over messages with source and expiry
- ✅ **Requirement 6.7**: Support expiry dates for rollover messages
- ✅ **Requirement 6.8**: Use rollover messages in FIFO order (oldest first)

## Next Steps

1. ✅ **Task 6.1**: packages table created
2. ✅ **Task 6.2**: subscriptions table created
3. ✅ **Task 6.3**: rollover_messages table created ← **YOU ARE HERE**
4. ⏭️ **Task 7**: Develop Subscription Manager
5. ⏭️ **Task 8**: Develop Rollover Manager

## Related Files

- Migration: `migrations/018_rollover_messages_table.sql`
- Documentation: `migrations/README_ROLLOVER.md`
- Test Script: `test_rollover_migration.sql`
- Runner Script: `run-rollover-migration.ps1`
- Previous: `migrations/017_subscriptions_table.sql`

## Support

For detailed documentation, see:
- `README_ROLLOVER.md` - Complete usage guide
- Design document: `.kiro/specs/tariff-pricing-system/design.md`
- Requirements: `.kiro/specs/tariff-pricing-system/requirements.md`

## Summary

You now have:
- ✅ Rollover messages table with tenant isolation
- ✅ FIFO usage tracking
- ✅ Expiry management
- ✅ Helper functions for all operations
- ✅ Comprehensive tests
- ✅ Easy-to-use migration scripts

The rollover system is ready for integration with the Subscription Manager (Task 7)!
