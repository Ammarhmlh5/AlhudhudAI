# Notification Service - Quick Start Guide

This guide will help you quickly integrate the Notification Service into your application.

## Installation

The notification service is already included in the project. No additional installation required.

## Quick Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/al-hudhud/backend/internal/notifications"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "go.uber.org/zap"
)

func main() {
    // Create logger
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // Create notification service
    notificationService := notifications.NewService(logger)
    
    ctx := context.Background()
    tenantID := uuid.New()
    
    // Example 1: Send renewal reminder
    err := notificationService.SendRenewalReminder(
        ctx,
        tenantID,
        1,                              // subscription ID
        "Standard Package",             // package name
        time.Now().AddDate(0, 0, 3),   // renewal date (3 days from now)
        decimal.NewFromFloat(20.00),   // amount
        "USD",                          // currency
    )
    if err != nil {
        logger.Error("Failed to send renewal reminder", zap.Error(err))
    }
    
    // Example 2: Send low balance warning
    err = notificationService.SendLowBalanceWarning(
        ctx,
        tenantID,
        decimal.NewFromFloat(8.50),    // current balance
        decimal.NewFromFloat(10.00),   // threshold
        "USD",                          // currency
    )
    if err != nil {
        logger.Error("Failed to send low balance warning", zap.Error(err))
    }
}
```

## Common Use Cases

### 1. Subscription Renewal Flow

```go
// In your auto renewal service
func (s *AutoRenewalService) ProcessRenewal(ctx context.Context, subscription *Subscription) error {
    // Step 1: Send reminder 3 days before
    if subscription.DaysUntilRenewal() == 3 {
        err := s.notificationService.SendRenewalReminder(
            ctx,
            subscription.TenantID,
            subscription.ID,
            subscription.PackageName,
            subscription.NextRenewalDate,
            subscription.Price,
            subscription.Currency,
        )
        if err != nil {
            s.logger.Error("Failed to send renewal reminder", zap.Error(err))
        }
    }
    
    // Step 2: Attempt renewal
    err := s.RenewSubscription(ctx, subscription)
    if err != nil {
        // Step 3a: Send failure notification
        err = s.notificationService.SendRenewalFailure(
            ctx,
            subscription.TenantID,
            subscription.ID,
            subscription.PackageName,
            "Insufficient balance",
            subscription.Price,
            subscription.CurrentBalance,
            subscription.Currency,
            3, // grace period days
            time.Now().AddDate(0, 0, 3),
        )
        return err
    }
    
    // Step 3b: Send success notification
    err = s.notificationService.SendRenewalSuccess(
        ctx,
        subscription.TenantID,
        subscription.ID,
        subscription.PackageName,
        subscription.StartDate,
        subscription.EndDate,
        subscription.Price,
        subscription.Currency,
        subscription.NewBalance,
        subscription.IncludedMessages,
    )
    
    return err
}
```

### 2. Balance Management

```go
// In your balance manager
func (m *BalanceManager) TopupBalance(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal) error {
    // Add balance
    newBalance, err := m.AddBalance(ctx, tenantID, amount)
    if err != nil {
        return err
    }
    
    // Send topup notification
    err = m.notificationService.SendBalanceTopup(
        ctx,
        tenantID,
        amount,
        "USD",
        newBalance,
        "Credit Card",
    )
    
    return err
}

func (m *BalanceManager) CheckBalance(ctx context.Context, tenantID uuid.UUID) error {
    balance, err := m.GetBalance(ctx, tenantID)
    if err != nil {
        return err
    }
    
    threshold := decimal.NewFromFloat(10.00)
    
    // Send warning if below threshold
    if balance.LessThan(threshold) {
        err = m.notificationService.SendLowBalanceWarning(
            ctx,
            tenantID,
            balance,
            threshold,
            "USD",
        )
    }
    
    return err
}
```

### 3. Message Usage Monitoring

```go
// In your subscription manager
func (m *SubscriptionManager) CheckMessageUsage(ctx context.Context, subscription *Subscription) error {
    // Calculate percentage remaining
    percentageRemaining := float64(subscription.MessagesRemaining) / float64(subscription.MessagesTotal) * 100
    
    // Send warning if below 10%
    if percentageRemaining <= 10.0 {
        err := m.notificationService.SendLowMessagesWarning(
            ctx,
            subscription.TenantID,
            subscription.ID,
            subscription.PackageName,
            subscription.MessagesRemaining,
            subscription.MessagesTotal,
            percentageRemaining,
        )
        if err != nil {
            m.logger.Error("Failed to send low messages warning", zap.Error(err))
        }
    }
    
    return nil
}
```

## Testing Your Integration

### Unit Test Example

```go
func TestNotificationIntegration(t *testing.T) {
    logger := zap.NewNop()
    notificationService := notifications.NewService(logger)
    
    ctx := context.Background()
    tenantID := uuid.New()
    
    // Test renewal reminder
    err := notificationService.SendRenewalReminder(
        ctx,
        tenantID,
        1,
        "Test Package",
        time.Now().AddDate(0, 0, 3),
        decimal.NewFromFloat(20.00),
        "USD",
    )
    
    if err != nil {
        t.Errorf("SendRenewalReminder failed: %v", err)
    }
}
```

### Running Tests

```bash
# Run notification service tests
./run-notification-tests.ps1

# Or with go test
cd services/backend
go test -v ./internal/notifications
```

## Notification Types Reference

| Function | When to Use | Priority |
|----------|-------------|----------|
| `SendRenewalReminder` | 3 days before renewal | Normal |
| `SendRenewalSuccess` | After successful renewal | Normal |
| `SendRenewalFailure` | When renewal fails | High |
| `SendBalanceTopup` | After balance added | Normal |
| `SendLowBalanceWarning` | Balance below threshold | High |
| `SendLowMessagesWarning` | Messages below 10% | Normal |

## Best Practices

1. **Always handle errors**: Log notification failures but don't block main operations
2. **Use appropriate priorities**: High priority for urgent notifications
3. **Avoid notification spam**: Check if similar notification was recently sent
4. **Include relevant data**: Provide all necessary information in notification
5. **Test thoroughly**: Verify notifications in different scenarios

## Troubleshooting

### Notifications not appearing in logs

Check that your logger is properly configured:

```go
logger, err := zap.NewProduction()
if err != nil {
    panic(err)
}
defer logger.Sync()
```

### Missing notification data

Ensure all required parameters are provided:

```go
// Bad - missing data
notificationService.SendRenewalReminder(ctx, tenantID, 0, "", time.Time{}, decimal.Zero, "")

// Good - complete data
notificationService.SendRenewalReminder(
    ctx,
    tenantID,
    subscriptionID,
    packageName,
    renewalDate,
    amount,
    currency,
)
```

## Next Steps

1. Review the [full README](README.md) for detailed documentation
2. Check [Auto Renewal Quick Start](../billing/AUTO_RENEWAL_QUICK_START.md) for integration examples
3. Explore notification preferences and customization (coming soon)
4. Implement email/SMS delivery (coming soon)

## Support

For questions or issues:
- Check the [README](README.md)
- Review test files for examples
- Contact the development team
