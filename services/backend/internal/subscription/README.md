# Subscription Manager

## Overview

The Subscription Manager handles monthly subscription packages for the SMS gateway service. It provides functionality for:

- Subscribing to packages
- Message quota management
- Overage tracking and billing
- Usage reporting

## Features

### 1. Package Subscription (Task 7.1)

Allows tenants to subscribe to monthly packages with atomic balance deduction.

**Requirements Implemented:**
- 4.1: Package subscription with balance check
- 4.2: Atomic balance deduction
- 4.3: Subscription period management
- 4.4: Accounting transaction recording

**Key Features:**
- Atomic balance deduction using SERIALIZABLE transactions
- Insufficient balance detection
- Automatic subscription period calculation (monthly/yearly)
- Accounting transaction recording
- Tenant isolation via RLS

### 2. Message Deduction (Task 7.3)

Deducts messages from subscription quota atomically.

**Requirements Implemented:**
- 4.5: Message deduction from quota
- 4.6: Usage tracking

**Key Features:**
- Atomic UPDATE for messages_used and messages_remaining
- Handles unlimited packages (NULL remaining)
- Detects overage condition (when messages_remaining reaches 0)
- Uses SERIALIZABLE transaction isolation level
- Respects Row Level Security (RLS)
- Supports multi-segment messages (long messages)

**Status:** ✅ Implemented

### 3. Overage Management (Task 7.5)

Calculates and charges for messages beyond the included quota.

**Requirements Implemented:**
- 5.1: Overage detection
- 5.2: Overage cost calculation
- 5.3: Balance deduction for overage
- 5.4: Overage tracking
- 5.5: Overage reporting

**Status:** Not yet implemented (TODO in task 7.5)

### 4. Usage Reporting (Task 7.7)

Provides detailed usage information for subscriptions.

**Requirements Implemented:**
- 4.6: Usage display
- 5.7: Overage display

**Status:** Not yet implemented (TODO in task 7.7)

## Usage

### Subscribe to a Package

```go
import (
    "context"
    "github.com/al-hudhud/backend/internal/subscription"
    "github.com/google/uuid"
)

// Create manager
manager := subscription.NewManager(pool, logger)

// Subscribe to package
result, err := manager.Subscribe(ctx, subscription.SubscribeRequest{
    TenantID:  uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
    PackageID: 1, // Basic package
    AutoRenew: true,
})

if err != nil {
    log.Fatal(err)
}

if !result.Success {
    log.Printf("Subscription failed: %s", result.Message)
    return
}

log.Printf("Subscription created: ID=%d", result.SubscriptionID)
log.Printf("Balance: %s -> %s", result.OldBalance, result.NewBalance)
```

### Get Active Subscription

```go
sub, err := manager.GetActiveSubscription(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Active subscription: %+v", sub)
```

### Get Package Details

```go
pkg, err := manager.GetPackage(ctx, packageID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Package: %s - $%s/month", pkg.Name, pkg.PriceUSD)
log.Printf("Included messages: %d", *pkg.IncludedMessages)
```

### Deduct Messages from Subscription

```go
// Deduct messages when sending (single message)
result, err := manager.DeductMessages(ctx, subscription.DeductMessagesRequest{
    SubscriptionID: 123,
    MessageCount:   1,
    MessageID:      "msg_abc123",
})

if err != nil {
    log.Fatal(err)
}

if !result.Success {
    log.Printf("Deduction failed: %v", result.Error)
    if result.OverageMessages > 0 {
        log.Printf("Overage detected: %d messages", result.OverageMessages)
        // Handle overage scenario (will be implemented in task 7.5)
    }
    return
}

log.Printf("Messages deducted: %d", result.MessagesDeducted)
if result.MessagesRemaining != nil {
    log.Printf("Messages remaining: %d", *result.MessagesRemaining)
} else {
    log.Printf("Messages remaining: unlimited")
}
```

```go
// Deduct multiple segments for long messages
result, err := manager.DeductMessages(ctx, subscription.DeductMessagesRequest{
    SubscriptionID: 123,
    MessageCount:   3, // Long message split into 3 segments
    MessageID:      "msg_xyz789",
})

if err != nil {
    log.Fatal(err)
}

if result.Success {
    log.Printf("Deducted %d message segments", result.MessagesDeducted)
}
```

```go
// Handle unlimited packages
result, err := manager.DeductMessages(ctx, subscription.DeductMessagesRequest{
    SubscriptionID: 456, // Enterprise unlimited package
    MessageCount:   1,
    MessageID:      "msg_unlimited",
})

if err != nil {
    log.Fatal(err)
}

if result.Success {
    // MessagesRemaining will be nil for unlimited packages
    log.Printf("Message sent (unlimited package)")
}
```

## Database Schema

### Packages Table

```sql
CREATE TABLE packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    service_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    price_usd DECIMAL(10, 2) NOT NULL,
    included_messages INTEGER, -- NULL = unlimited
    overage_rate DECIMAL(10, 4),
    is_unlimited BOOLEAN DEFAULT false,
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    rollover_enabled BOOLEAN DEFAULT false,
    rollover_limit INTEGER,
    rollover_expiry_months INTEGER,
    is_active BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Subscriptions Table

```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    package_id INTEGER NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,
    status VARCHAR(20) DEFAULT 'active',
    auto_renew BOOLEAN DEFAULT true,
    grace_period_days INTEGER DEFAULT 3,
    last_renewed_at TIMESTAMP,
    next_renewal_date DATE,
    renewal_attempts INTEGER DEFAULT 0,
    last_renewal_attempt TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Available Packages

1. **Basic** - $10/month - 1,000 messages
2. **Standard** - $20/month - 3,000 messages
3. **Professional** - $40/month - 8,000 messages
4. **Business** - $50/month - 15,000 messages
5. **Enterprise** - $100/month - Unlimited messages

## Error Handling

The manager uses typed errors for better error handling:

```go
result, err := manager.Subscribe(ctx, req)
if err != nil {
    if subErr, ok := err.(*subscription.SubscriptionError); ok {
        switch subErr.Type {
        case subscription.ErrorInsufficientBalance:
            // Handle insufficient balance
        case subscription.ErrorPackageNotFound:
            // Handle package not found
        case subscription.ErrorPackageInactive:
            // Handle inactive package
        default:
            // Handle other errors
        }
    }
}
```

## Testing

### Unit Tests

Run unit tests:
```bash
cd services/backend
go test ./internal/subscription/...
```

### Property-Based Tests

Property-based tests will be added in task 7.2 to verify:
- Atomic subscription operations
- Balance consistency
- Idempotency

## Implementation Status

- [x] Task 7.1: Subscribe function (COMPLETED)
- [x] Task 7.2: Property test for subscription (COMPLETED)
- [x] Task 7.3: DeductMessages function (COMPLETED)
- [ ] Task 7.4: Property test for message deduction (TODO)
- [ ] Task 7.5: CalculateOverageCost function (TODO)
- [ ] Task 7.6: Property test for overage (TODO)
- [ ] Task 7.7: GetUsage function (TODO)

## Next Steps

1. Implement property-based tests for Subscribe (Task 7.2)
2. Implement DeductMessages function (Task 7.3)
3. Implement CalculateOverageCost function (Task 7.5)
4. Implement GetUsage function (Task 7.7)

## Related Components

- **Balance Manager**: Handles balance operations
- **Tariff Engine**: Calculates message costs
- **Rollover Manager**: Manages message rollover (Task 8)
- **Auto Renewal Service**: Handles automatic renewals (Task 12)

## References

- Requirements: `.kiro/specs/tariff-pricing-system/requirements.md`
- Design: `.kiro/specs/tariff-pricing-system/design.md`
- Tasks: `.kiro/specs/tariff-pricing-system/tasks.md`
