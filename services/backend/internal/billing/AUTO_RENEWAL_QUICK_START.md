# Auto Renewal Service - Quick Start Guide

## What is Auto Renewal?

The Auto Renewal Service automatically renews monthly subscriptions when they expire. It handles:
- ✅ Daily checks for expired subscriptions
- ✅ Automatic balance deduction and renewal
- ✅ Grace periods for insufficient balance
- ✅ Subscription suspension after grace period
- ✅ Automatic reactivation when balance is topped up

## Quick Setup (5 minutes)

### 1. Prerequisites

Ensure you have:
- PostgreSQL database with subscriptions table (Migration 017)
- River Queue set up and running
- Go 1.21+ installed

### 2. Initialize Service

```go
package main

import (
    "context"
    "github.com/al-hudhud/backend/internal/billing"
    "github.com/al-hudhud/backend/internal/database"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "go.uber.org/zap"
)

func main() {
    ctx := context.Background()
    
    // Setup database
    pool, _ := pgxpool.New(ctx, "postgres://...")
    defer pool.Close()
    
    // Setup logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // Setup River client
    client, _ := river.NewClient(...)
    
    // Create auto renewal service
    service := billing.NewAutoRenewalService(pool, client, logger)
    
    // Register worker
    workers := river.NewWorkers()
    river.AddWorker(workers, billing.NewDailyRenewalCheckWorker(service, logger))
    
    // Schedule daily check (once at startup)
    billing.ScheduleDailyRenewalCheck(ctx, client, logger)
    
    // Start River client
    client.Start(ctx)
}
```

### 3. Test It

```bash
# Run property tests
cd services/backend
go test -v -run TestProperty_AutoRenewal ./internal/billing/

# Or use the PowerShell script
.\run-auto-renewal-property-test.ps1
```

## Common Use Cases

### Use Case 1: Subscription Expires with Sufficient Balance

```
Scenario:
- Subscription ends on 2024-03-01
- Package price: $20
- Tenant balance: $50

Result:
✅ Balance deducted: $50 - $20 = $30
✅ Subscription renewed: 2024-03-01 to 2024-04-01
✅ Counters reset: messages_used = 0, messages_remaining = 1000
✅ Status: active
✅ Transaction recorded
✅ Invoice generated
```

### Use Case 2: Subscription Expires with Insufficient Balance

```
Scenario:
- Subscription ends on 2024-03-01
- Package price: $20
- Tenant balance: $5

Result:
⚠️ Renewal fails
⚠️ Grace period applied (3 days)
⚠️ Status: grace_period
⚠️ Notification sent (TODO: Task 13)
⏰ System will retry daily during grace period
```

### Use Case 3: Grace Period Expires

```
Scenario:
- Grace period: 3 days (2024-03-01 to 2024-03-04)
- Balance still insufficient on 2024-03-04

Result:
🚫 Subscription suspended
🚫 Status: suspended
🚫 Cannot send messages
🚫 Notification sent (TODO: Task 13)
```

### Use Case 4: Balance Topped Up

```
Scenario:
- Subscription suspended
- Tenant tops up balance: $5 + $30 = $35
- Package price: $20

Result:
✅ Auto-reactivation triggered
✅ Balance deducted: $35 - $20 = $15
✅ Subscription renewed
✅ Status: active
✅ Can send messages again
```

## Testing Scenarios

### Test 1: Successful Renewal

```sql
-- Setup: Create tenant with sufficient balance
INSERT INTO tenants (id, name, balance) 
VALUES ('123e4567-e89b-12d3-a456-426614174000', 'Test Tenant', 50.00);

-- Create package
INSERT INTO packages (tenant_id, name, price_usd, included_messages)
VALUES ('123e4567-e89b-12d3-a456-426614174000', 'Basic', 20.00, 1000);

-- Create expired subscription
INSERT INTO subscriptions (tenant_id, package_id, end_date, auto_renew)
VALUES ('123e4567-e89b-12d3-a456-426614174000', 1, CURRENT_DATE - 1, true);

-- Run renewal
-- The daily job will process this automatically
-- Or call manually: service.CheckExpiredSubscriptions(ctx)

-- Verify results
SELECT balance FROM tenants WHERE id = '123e4567-e89b-12d3-a456-426614174000';
-- Expected: 30.00

SELECT status, messages_used, messages_remaining 
FROM subscriptions WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000';
-- Expected: status='active', messages_used=0, messages_remaining=1000
```

### Test 2: Grace Period

```sql
-- Setup: Create tenant with insufficient balance
INSERT INTO tenants (id, name, balance) 
VALUES ('223e4567-e89b-12d3-a456-426614174000', 'Test Tenant 2', 5.00);

-- Create package
INSERT INTO packages (tenant_id, name, price_usd, included_messages)
VALUES ('223e4567-e89b-12d3-a456-426614174000', 'Basic', 20.00, 1000);

-- Create expired subscription
INSERT INTO subscriptions (tenant_id, package_id, end_date, auto_renew, grace_period_days)
VALUES ('223e4567-e89b-12d3-a456-426614174000', 2, CURRENT_DATE - 1, true, 3);

-- Run renewal
-- service.CheckExpiredSubscriptions(ctx)

-- Verify results
SELECT status, renewal_attempts 
FROM subscriptions WHERE tenant_id = '223e4567-e89b-12d3-a456-426614174000';
-- Expected: status='grace_period', renewal_attempts=1
```

### Test 3: Auto-Reactivation

```sql
-- Setup: Create suspended subscription
UPDATE subscriptions 
SET status = 'suspended'
WHERE tenant_id = '223e4567-e89b-12d3-a456-426614174000';

-- Top up balance
UPDATE tenants 
SET balance = 30.00
WHERE id = '223e4567-e89b-12d3-a456-426614174000';

-- Trigger auto-reactivation
-- service.AutoReactivate(ctx, tenantID)

-- Verify results
SELECT status FROM subscriptions 
WHERE tenant_id = '223e4567-e89b-12d3-a456-426614174000';
-- Expected: status='active'

SELECT balance FROM tenants 
WHERE id = '223e4567-e89b-12d3-a456-426614174000';
-- Expected: 10.00 (30 - 20)
```

## Monitoring

### Check Daily Job Status

```sql
-- Check River jobs
SELECT * FROM river_job 
WHERE kind = 'daily_renewal_check'
ORDER BY scheduled_at DESC
LIMIT 10;
```

### Monitor Renewal Success Rate

```sql
-- Count renewals by status
SELECT 
    DATE(created_at) as date,
    COUNT(*) FILTER (WHERE status = 'active') as successful,
    COUNT(*) FILTER (WHERE status = 'grace_period') as grace_period,
    COUNT(*) FILTER (WHERE status = 'suspended') as suspended
FROM subscriptions
WHERE last_renewed_at >= CURRENT_DATE - 30
GROUP BY DATE(created_at)
ORDER BY date DESC;
```

### Find Subscriptions Needing Attention

```sql
-- Subscriptions in grace period
SELECT s.id, t.name, s.end_date, s.renewal_attempts, t.balance, p.price_usd
FROM subscriptions s
JOIN tenants t ON s.tenant_id = t.id
JOIN packages p ON s.package_id = p.id
WHERE s.status = 'grace_period'
ORDER BY s.end_date;

-- Suspended subscriptions
SELECT s.id, t.name, s.end_date, t.balance, p.price_usd
FROM subscriptions s
JOIN tenants t ON s.tenant_id = t.id
JOIN packages p ON s.package_id = p.id
WHERE s.status = 'suspended'
  AND s.auto_renew = true
ORDER BY s.end_date;
```

## Troubleshooting

### Issue: Renewal Not Happening

**Check:**
1. Is auto_renew enabled?
   ```sql
   SELECT auto_renew FROM subscriptions WHERE id = <subscription_id>;
   ```

2. Is the subscription expired?
   ```sql
   SELECT end_date FROM subscriptions WHERE id = <subscription_id>;
   ```

3. Is the daily job running?
   ```sql
   SELECT * FROM river_job WHERE kind = 'daily_renewal_check' ORDER BY scheduled_at DESC LIMIT 1;
   ```

### Issue: Balance Not Deducted

**Check:**
1. Is balance sufficient?
   ```sql
   SELECT t.balance, p.price_usd
   FROM subscriptions s
   JOIN tenants t ON s.tenant_id = t.id
   JOIN packages p ON s.package_id = p.id
   WHERE s.id = <subscription_id>;
   ```

2. Check transaction logs:
   ```sql
   SELECT * FROM accounting_transactions
   WHERE reference_type = 'subscription'
     AND reference_id = <subscription_id>
   ORDER BY created_at DESC;
   ```

### Issue: Subscription Stuck in Grace Period

**Check:**
1. How many days in grace period?
   ```sql
   SELECT 
       end_date,
       grace_period_days,
       end_date + grace_period_days as grace_period_end,
       CURRENT_DATE > (end_date + grace_period_days) as should_suspend
   FROM subscriptions
   WHERE id = <subscription_id>;
   ```

2. Check renewal attempts:
   ```sql
   SELECT renewal_attempts, last_renewal_attempt
   FROM subscriptions
   WHERE id = <subscription_id>;
   ```

## Performance Tips

### 1. Index Optimization

Ensure these indexes exist:
```sql
-- For daily check query
CREATE INDEX idx_subscriptions_renewal_due 
ON subscriptions(next_renewal_date, status, auto_renew)
WHERE status IN ('active', 'grace_period') AND auto_renew = true;

-- For auto-reactivation query
CREATE INDEX idx_subscriptions_suspended_auto_renew
ON subscriptions(tenant_id, status, auto_renew)
WHERE status = 'suspended' AND auto_renew = true;
```

### 2. Batch Processing

The daily check processes subscriptions in batches:
- Processes one subscription at a time
- Commits each transaction independently
- Continues on error (logs and moves to next)

### 3. Monitoring Query Performance

```sql
-- Check slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
WHERE query LIKE '%subscriptions%'
ORDER BY mean_exec_time DESC
LIMIT 10;
```

## Next Steps

1. ✅ **Task 12.1-12.6**: Auto renewal service (DONE)
2. ⏳ **Task 13**: Notification service (TODO)
   - Email notifications
   - SMS notifications
   - In-app notifications
3. ⏳ **Task 14**: Invoice generator (TODO)
   - PDF generation
   - Email delivery
4. ⏳ **Task 15**: Admin dashboard (TODO)
   - Renewal monitoring
   - Grace period management
   - Suspension handling

## Resources

- **Full Documentation**: `internal/billing/README.md`
- **Property Tests**: `internal/billing/auto_renewal_property_test.go`
- **Requirements**: `.kiro/specs/tariff-pricing-system/requirements.md` (Requirement 14)
- **Design**: `.kiro/specs/tariff-pricing-system/design.md` (Phase 3)

## Support

For questions or issues:
1. Check the property tests for examples
2. Review the README for detailed documentation
3. Check application logs for errors
4. Refer to the requirements document for business rules
