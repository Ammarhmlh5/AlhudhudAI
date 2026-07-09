# Gateway Subscription Billing - Quick Start Guide

## Overview

This module implements automatic subscription billing for dedicated gateways (Task 19.6).

**نموذج العمل المالي (Business Model):**
- الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل
- المنصة تتقاضى فقط:
  1. رسوم الاشتراك (subscription_fee) - شهري أو سنوي
  2. رسوم إيجار السيرفر (server_rental_fee) - شهري
- **لا يتم خصم أي تكلفة متعلقة بالرسائل**

## Requirements Covered

- ✅ **Requirement 8.10**: حفظ نوع الاشتراك ورسوم الاشتراك
- ✅ **Requirement 8.11**: حفظ رسوم إيجار السيرفر الشهرية
- ✅ **Requirement 8.15**: عدم معالجة أي دفعات متعلقة بتكلفة الرسائل

## Files Created

1. `services/backend/internal/billing/gateway_billing.go` - Main billing service
2. `services/backend/internal/billing/gateway_billing_test.go` - Unit tests
3. `services/backend/internal/billing/gateway_billing_cron.go` - Cron job for automatic billing

## Key Features

### 1. Automatic Billing
- Daily check for gateways with due billing dates
- Charges subscription_fee + server_rental_fee
- Updates next_billing_date automatically
- Records transaction in accounting_transactions
- Generates invoice for enterprise company

### 2. Billing Cycles
- **Monthly**: Charges every month
- **Yearly**: Charges every year

### 3. Transaction Recording
- All billing operations are recorded in `accounting_transactions`
- Transaction type: `GATEWAY_SUBSCRIPTION`
- Description clearly states subscription and rental fees
- **Does NOT include message costs**

### 4. Invoice Generation
- Automatic invoice creation for each billing
- Invoice type: `gateway_subscription`
- Includes breakdown of subscription and rental fees

## Usage

### Initialize the Service

```go
import (
    "github.com/al-hudhud/backend/internal/billing"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
)

// Create service
pool := // your pgxpool.Pool
logger := // your zap.Logger
service := billing.NewGatewayBillingService(pool, logger)
```

### Manual Billing

```go
// Get gateway billing info
info, err := service.GetGatewayBillingInfo(ctx, tenantID, gatewayID)
if err != nil {
    log.Fatal(err)
}

// Bill the gateway
err = service.BillGatewaySubscription(ctx, info)
if err != nil {
    log.Fatal(err)
}
```

### Automatic Billing (Cron Job)

```go
// Create cron job (checks daily)
cron := billing.NewGatewayBillingCron(pool, logger, 24*time.Hour)

// Start in background
go cron.Start(context.Background())
```

### Check Upcoming Billings

```go
// Get gateways with billing due in next 7 days
billings, err := service.GetUpcomingBillings(ctx, 7)
if err != nil {
    log.Fatal(err)
}

for _, billing := range billings {
    fmt.Printf("Gateway: %s, Due: %s, Amount: %s\n",
        billing.GatewayName,
        billing.NextBillingDate,
        billing.SubscriptionFee.Add(billing.ServerRentalFee),
    )
}
```

## Integration with main.go

Add to `services/backend/main.go`:

```go
import (
    "github.com/al-hudhud/backend/internal/billing"
)

func main() {
    // ... existing code ...

    // Initialize Gateway Billing Cron Job
    gatewayBillingCron := billing.NewGatewayBillingCron(
        db.GetDB(),
        logger.GetLogger(),
        24*time.Hour, // Check daily
    )
    go gatewayBillingCron.Start(context.Background())

    // ... rest of code ...
}
```

## Testing

### Run Unit Tests

```bash
cd services/backend
go test -v ./internal/billing -run TestBillGatewaySubscription
```

### Run All Gateway Billing Tests

```bash
go test -v ./internal/billing -run Gateway
```

### Test Coverage

```bash
go test -cover ./internal/billing
```

## Database Schema

The billing service uses the following tables:

1. **dedicated_gateways**: Gateway configuration with subscription fees
2. **accounting_transactions**: Records all billing transactions
3. **billing_invoices**: Stores generated invoices
4. **tenants**: Tenant balance management

## Important Notes

### ⚠️ Critical: No Message Costs

**The system SHALL NOT charge for message costs.**

- Subscription billing ONLY charges:
  - `subscription_fee` (monthly or yearly)
  - `server_rental_fee` (monthly)
- Message costs are paid directly by the enterprise company to the telecom provider
- The platform only monitors the telecom balance and sends alerts

### Transaction Descriptions

All transaction descriptions clearly state:
- "Gateway subscription billing for [gateway_code]"
- "Subscription: [amount], Server Rental: [amount]"
- **NO mention of message costs**

### Invoice Descriptions

All invoice descriptions include:
- Gateway name
- Subscription fee amount
- Server rental fee amount
- Billing cycle (monthly/yearly)
- **NO mention of message costs**

## Error Handling

### Insufficient Balance

If the tenant doesn't have sufficient balance:
- Billing fails with error: "insufficient balance: required X, available Y"
- No transaction is recorded
- No invoice is generated
- Balance remains unchanged (atomic operation)

### Transaction Atomicity

All billing operations are atomic:
- Balance deduction
- Gateway next_billing_date update
- Transaction recording
- Invoice generation

If any step fails, the entire operation is rolled back.

## Monitoring

### Check Billing Status

```go
// Get upcoming billings for next 30 days
billings, err := service.GetUpcomingBillings(ctx, 30)

// Send reminders 3 days before billing
cron.SendBillingReminders(ctx, 3)
```

### Manual Trigger

```go
// Run billing check manually
err := cron.RunOnce(ctx)
```

## Next Steps

1. ✅ Implement gateway billing service
2. ✅ Create unit tests
3. ✅ Add cron job for automatic billing
4. ⏳ Integrate with notification service (Task 13)
5. ⏳ Add API endpoints for billing management
6. ⏳ Create admin dashboard for billing overview

## Related Tasks

- Task 19.5: Balance Monitor (monitors telecom balance)
- Task 14: Invoice Generator (generates PDF invoices)
- Task 13: Notification Service (sends billing notifications)

## Support

For questions or issues, refer to:
- `.kiro/specs/tariff-pricing-system/requirements.md` (Requirements 8.10, 8.11, 8.15)
- `.kiro/specs/tariff-pricing-system/design.md` (Design documentation)
- `services/backend/migrations/020_dedicated_gateways_table.sql` (Database schema)
