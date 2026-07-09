# Notification Service

The Notification Service provides comprehensive notification functionality for the tariff and pricing system, supporting multiple notification channels and types.

## Features

### Notification Types

1. **Renewal Reminder** (Requirements 14.8, 17.1)
   - Sent 3 days before subscription renewal
   - Includes package details, renewal date, and required amount
   - Helps users prepare for automatic renewal

2. **Renewal Success** (Requirements 14.9, 17.2)
   - Sent after successful subscription renewal
   - Shows new subscription period, included messages, and updated balance
   - Confirms successful payment

3. **Renewal Failure** (Requirements 14.10, 17.3)
   - Sent when subscription renewal fails
   - Explains failure reason and grace period details
   - Prompts user to add balance

4. **Balance Topup** (Requirement 17.4)
   - Sent when balance is added to account
   - Shows amount added and new balance
   - Confirms successful payment

5. **Low Balance Warning** (Requirement 17.5)
   - Sent when balance falls below threshold
   - Customizable threshold per tenant
   - Prevents service interruption

6. **Low Messages Warning** (Requirement 17.6)
   - Sent when available messages reach 10% or less
   - Shows remaining messages and percentage
   - Suggests upgrading or adding balance

### Notification Channels

- **Email**: Professional email notifications
- **SMS**: Text message notifications
- **In-App**: Dashboard notifications
- **All**: Send via all available channels

### Priority Levels

- **Normal**: Regular notifications (renewal reminders, success messages)
- **High**: Important notifications (renewal failures, low balance)
- **Critical**: Urgent notifications (gateway down, critical errors)

## Usage

### Basic Usage

```go
import (
    "github.com/al-hudhud/backend/internal/notifications"
    "go.uber.org/zap"
)

// Create notification service
logger := zap.NewProduction()
notificationService := notifications.NewService(logger)

// Send renewal reminder
err := notificationService.SendRenewalReminder(
    ctx,
    tenantID,
    subscriptionID,
    "Standard Package",
    renewalDate,
    decimal.NewFromFloat(20.00),
    "USD",
)
```

### Integration with Auto Renewal

```go
// In auto renewal service
func (s *AutoRenewalService) RenewSubscription(...) error {
    // ... renewal logic ...
    
    // Send success notification
    err = s.notificationService.SendRenewalSuccess(
        ctx,
        tenantID,
        subscriptionID,
        packageName,
        startDate,
        endDate,
        amount,
        currency,
        newBalance,
        includedMessages,
    )
    
    return nil
}
```

### Integration with Balance Manager

```go
// In balance manager
func (m *Manager) TopupBalance(...) error {
    // ... topup logic ...
    
    // Send topup notification
    err = m.notificationService.SendBalanceTopup(
        ctx,
        tenantID,
        amount,
        currency,
        newBalance,
        "Credit Card",
    )
    
    return nil
}
```

## Testing

Run the notification service tests:

```bash
# Run all tests
./run-notification-tests.ps1

# Or directly with go test
cd services/backend
go test -v ./internal/notifications
```

## Implementation Status

### Completed (Task 13)

- ✅ Task 13.1: Renewal reminder notifications
- ✅ Task 13.2: Renewal success notifications
- ✅ Task 13.3: Renewal failure notifications
- ✅ Task 13.4: Balance topup notifications
- ✅ Task 13.5: Low balance warnings
- ✅ Task 13.6: Low messages warnings

### Future Enhancements

- [ ] Email delivery via SMTP/SendGrid/AWS SES
- [ ] SMS delivery via SMS gateway
- [ ] In-app notification storage in database
- [ ] Notification preferences per tenant
- [ ] Rate limiting to avoid notification spam
- [ ] Retry logic for failed deliveries
- [ ] Delivery status tracking
- [ ] Notification templates
- [ ] Multi-language support
- [ ] Webhook notifications

## Architecture

### Current Implementation

The notification service currently logs all notifications with structured logging. This provides:

- Complete audit trail of all notifications
- Easy debugging and monitoring
- Foundation for future delivery implementations

### Planned Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Notification Service                    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Email      │  │     SMS      │  │   In-App     │ │
│  │   Provider   │  │   Provider   │  │   Storage    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│         │                  │                  │         │
│         └──────────────────┼──────────────────┘         │
│                            │                            │
│                    ┌───────▼────────┐                   │
│                    │  Notification  │                   │
│                    │     Queue      │                   │
│                    └────────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

## Configuration

Future configuration options:

```yaml
notifications:
  email:
    provider: "sendgrid"  # or "smtp", "aws_ses"
    from: "noreply@example.com"
    enabled: true
  
  sms:
    provider: "twilio"  # or custom gateway
    from: "+1234567890"
    enabled: true
  
  in_app:
    enabled: true
    retention_days: 30
  
  rate_limiting:
    max_per_hour: 10
    max_per_day: 50
```

## Requirements Mapping

| Requirement | Description | Status |
|------------|-------------|--------|
| 14.8 | Send reminder 3 days before renewal | ✅ Implemented |
| 14.9 | Send success notification after renewal | ✅ Implemented |
| 14.10 | Send failure notification with grace period | ✅ Implemented |
| 17.1 | Renewal reminder notifications | ✅ Implemented |
| 17.2 | Renewal success notifications | ✅ Implemented |
| 17.3 | Renewal failure notifications | ✅ Implemented |
| 17.4 | Balance topup notifications | ✅ Implemented |
| 17.5 | Low balance warnings | ✅ Implemented |
| 17.6 | Low messages warnings | ✅ Implemented |
| 17.10 | Email notification support | 🔄 Planned |
| 17.11 | SMS notification support | 🔄 Planned |
| 17.12 | In-app notification support | 🔄 Planned |
| 17.13 | Notification preferences | 🔄 Planned |

## Related Components

- **Auto Renewal Service** (`internal/billing/auto_renewal.go`): Uses notification service for renewal events
- **Balance Manager** (`internal/balance/manager.go`): Uses notification service for balance events
- **Subscription Manager** (`internal/subscription/manager.go`): Uses notification service for subscription events

## See Also

- [Auto Renewal Quick Start](../billing/AUTO_RENEWAL_QUICK_START.md)
- [Balance Manager README](../balance/README.md)
- [Subscription Manager README](../subscription/README.md)
