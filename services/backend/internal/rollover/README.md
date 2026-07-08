# Rollover Manager

## Overview

The Rollover Manager handles the rollover of unused messages from one billing period to the next for subscription-based packages. It implements a FIFO (First-In-First-Out) system for using rolled-over messages.

## Features

### 1. Calculate Remaining Messages
**Requirement 6.2**

Calculates the number of messages remaining at the end of a billing period that are eligible for rollover.

```go
remaining, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
```

**Behavior:**
- Returns 0 if rollover is disabled for the package
- Returns 0 if package is unlimited
- Returns 0 if no messages remaining
- Returns exact `messages_remaining` when rollover is enabled

### 2. Rollover Messages
**Requirements 6.3, 6.4, 6.5, 6.6, 6.7**

Rolls over remaining messages to the next billing period with proper limits and expiry dates.

```go
err := manager.RolloverMessages(ctx, subscriptionID, remainingMessages)
```

**Behavior:**
- Respects `rollover_limit` from package (Requirement 6.5)
- Records in `rollover_messages` table (Requirement 6.3)
- Sets `expiry_date` based on `rollover_expiry_months` (Requirements 6.6, 6.7)
- Only rolls over if `rollover_enabled` is true (Requirement 6.4)

**Example:**
```go
// Package has rollover_limit = 500
// Subscription has 800 messages remaining
// Only 500 will be rolled over

err := manager.RolloverMessages(ctx, subscriptionID, 800)
// Creates rollover record with messages_count = 500
```

### 3. Use Rollover Messages (FIFO)
**Requirement 6.8**

Uses rolled-over messages in FIFO order (oldest first).

```go
used, err := manager.UseRolloverMessages(ctx, subscriptionID, messageCount)
```

**Behavior:**
- Uses oldest rollover messages first (FIFO)
- Updates `messages_count` in rollover records
- Deletes records when fully consumed
- Returns number of messages actually used

**Example:**
```go
// Rollover records:
// Record 1 (oldest): 100 messages
// Record 2: 200 messages
// Record 3 (newest): 150 messages

// Use 250 messages
used, err := manager.UseRolloverMessages(ctx, subscriptionID, 250)
// Result: used = 250
// Record 1: deleted (fully consumed)
// Record 2: 50 messages remaining
// Record 3: 150 messages (untouched)
```

### 4. Get Rollover Messages
**Requirement 6.9**

Retrieves all active rollover messages for a subscription.

```go
messages, err := manager.GetRolloverMessages(ctx, subscriptionID)
```

**Returns:**
- List of rollover messages ordered by `created_at` (oldest first)
- Only includes non-expired messages (`expiry_date >= CURRENT_DATE`)

### 5. Cleanup Expired Rollover
**Requirement 6.7**

Deletes expired rollover messages. Should be run as a daily cron job.

```go
err := manager.CleanupExpiredRollover(ctx)
```

**Behavior:**
- Deletes all rollover messages where `expiry_date < CURRENT_DATE`
- Logs deleted records for audit trail
- Returns total count of deleted records and messages

## Database Schema

### rollover_messages Table

```sql
CREATE TABLE rollover_messages (
    id BIGSERIAL PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES subscriptions(id),
    tenant_id UUID NOT NULL,
    messages_count BIGINT NOT NULL,
    source_month TIMESTAMP NOT NULL,
    expiry_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT rollover_messages_tenant_id_fkey 
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Indexes
CREATE INDEX idx_rollover_tenant_subscription 
    ON rollover_messages(tenant_id, subscription_id);
    
CREATE INDEX idx_rollover_tenant_expiry 
    ON rollover_messages(tenant_id, expiry_date);
```

## Usage Examples

### Monthly Renewal Flow

```go
// At end of billing period
remaining, err := manager.CalculateRemainingMessages(ctx, subscriptionID)
if err != nil {
    return err
}

if remaining > 0 {
    // Rollover remaining messages
    err = manager.RolloverMessages(ctx, subscriptionID, remaining)
    if err != nil {
        return err
    }
}

// Reset subscription counters
err = subscriptionManager.ResetCounters(ctx, subscriptionID)
```

### Sending Messages with Rollover

```go
// Try to use rollover messages first
usedFromRollover, err := rolloverManager.UseRolloverMessages(ctx, subscriptionID, messageCount)
if err != nil {
    return err
}

remainingToDeduct := messageCount - usedFromRollover

if remainingToDeduct > 0 {
    // Use from current subscription
    err = subscriptionManager.DeductMessages(ctx, subscriptionID, remainingToDeduct)
    if err != nil {
        return err
    }
}
```

### Daily Cleanup Job

```go
// Run daily at midnight
func cleanupExpiredRolloverJob(ctx context.Context) error {
    err := rolloverManager.CleanupExpiredRollover(ctx)
    if err != nil {
        logger.Error("Failed to cleanup expired rollover", zap.Error(err))
        return err
    }
    return nil
}
```

## Property-Based Tests

The Rollover Manager includes comprehensive property-based tests:

### Property 9: Calculate Remaining Messages
**Validates:** Requirement 6.2

Tests that `CalculateRemainingMessages`:
- Returns 0 when rollover is disabled
- Returns 0 when package is unlimited
- Returns 0 when messages_remaining <= 0
- Returns correct messages_remaining when rollover enabled
- Is idempotent (multiple calls return same result)

### Property 10: Respect Rollover Limit
**Validates:** Requirement 6.5

Tests that `RolloverMessages`:
- Never rolls over more than `rollover_limit`
- Rolls over all messages if remaining < limit
- Rolls over exactly limit if remaining >= limit
- Handles nil rollover_limit (unlimited rollover)

### Property 11: FIFO Order
**Validates:** Requirement 6.8

Tests that `UseRolloverMessages`:
- Uses oldest rollover messages first
- Correctly updates or deletes rollover records
- Returns correct number of messages used
- Handles partial consumption correctly

## Running Tests

```bash
# Run all tests
cd services/backend/internal/rollover
go test -v

# Run property tests only
go test -v -run TestProperty

# Run with more iterations
go test -v -run TestProperty -rapid.checks=1000

# Run specific property test
go test -v -run TestProperty_UseRolloverMessages_FIFO
```

Or use the PowerShell script:

```powershell
./services/backend/run-rollover-property-test.ps1
```

## Configuration

### Package Configuration

```sql
-- Enable rollover with 500 message limit, 3 month expiry
UPDATE packages SET
    rollover_enabled = true,
    rollover_limit = 500,
    rollover_expiry_months = 3
WHERE id = 1;

-- Unlimited rollover (no limit)
UPDATE packages SET
    rollover_enabled = true,
    rollover_limit = NULL,
    rollover_expiry_months = 3
WHERE id = 2;

-- Disable rollover
UPDATE packages SET
    rollover_enabled = false
WHERE id = 3;
```

## Multi-Tenancy Isolation

The Rollover Manager enforces strict tenant isolation:

1. **Row Level Security (RLS)**: All queries automatically filter by `tenant_id`
2. **Tenant Context**: `tenant_id` is extracted from JWT/API Key
3. **Cross-Tenant Prevention**: Middleware prevents access to other tenants' data

## Error Handling

```go
// Invalid subscription ID
_, err := manager.CalculateRemainingMessages(ctx, 0)
// Returns: "subscription_id is required"

// Subscription not found
_, err := manager.CalculateRemainingMessages(ctx, 99999)
// Returns: "failed to get subscription: no rows"

// Transaction failure
err := manager.RolloverMessages(ctx, subscriptionID, 100)
// Returns: "failed to commit transaction: ..."
```

## Performance Considerations

1. **Indexes**: Ensure indexes on `tenant_id`, `subscription_id`, and `expiry_date`
2. **Batch Cleanup**: Run cleanup job during off-peak hours
3. **FIFO Query**: Uses `ORDER BY created_at ASC` with `FOR UPDATE` lock
4. **Transaction Isolation**: Uses transactions for atomic operations

## Monitoring

Key metrics to monitor:

1. **Rollover Usage Rate**: Percentage of subscriptions using rollover
2. **Average Rollover Amount**: Average messages rolled over per subscription
3. **Expiry Rate**: Percentage of rolled-over messages that expire unused
4. **Cleanup Job Duration**: Time taken for daily cleanup
5. **FIFO Performance**: Query performance for `UseRolloverMessages`

## Cron Job Setup

### Using River Queue

```go
// Schedule daily cleanup at midnight
type CleanupExpiredRolloverArgs struct{}

func (CleanupExpiredRolloverArgs) Kind() string {
    return "cleanup_expired_rollover"
}

type CleanupExpiredRolloverWorker struct {
    river.WorkerDefaults[CleanupExpiredRolloverArgs]
    manager rollover.Manager
}

func (w *CleanupExpiredRolloverWorker) Work(ctx context.Context, job *river.Job[CleanupExpiredRolloverArgs]) error {
    return w.manager.CleanupExpiredRollover(ctx)
}

// Schedule job
_, err := riverClient.InsertMany(ctx, []river.InsertManyParams{
    {
        Args: CleanupExpiredRolloverArgs{},
        InsertOpts: &river.InsertOpts{
            ScheduledAt: time.Now().Add(24 * time.Hour),
        },
    },
})
```

## Troubleshooting

### Rollover not working
1. Check `rollover_enabled` is true in package
2. Check `is_unlimited` is false in package
3. Check `messages_remaining` > 0 in subscription
4. Check RLS policies are applied

### FIFO order incorrect
1. Verify `created_at` timestamps are different
2. Check query uses `ORDER BY created_at ASC`
3. Verify no manual updates to `created_at`

### Expired messages not cleaned up
1. Check cron job is running
2. Verify `expiry_date` is set correctly
3. Check cleanup job logs for errors

## References

- Requirements: 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9
- Design Document: `.kiro/specs/tariff-pricing-system/design.md`
- Property Tests: `manager_property_test.go`
- Migration: `migrations/018_rollover_messages_table.sql`
