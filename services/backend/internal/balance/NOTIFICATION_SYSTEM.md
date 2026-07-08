# Low Balance Notification System

## Overview

This document describes the Low Balance Notification System implemented as part of Task 2.7 in the message lifecycle rebuild specification.

## Requirements

The notification system implements the following requirements:

- **Requirement 11.4**: Send warning notification when balance < 10 units
- **Requirement 11.5**: Send critical notification when balance < 5 units  
- **Requirement 11.6**: Log all failed deduction attempts due to insufficient balance

## Architecture

### Components

#### 1. NotificationService Interface

```go
type NotificationService interface {
    CheckAndNotifyLowBalance(ctx context.Context, tenantID int64, balance decimal.Decimal) error
    LogInsufficientBalanceAttempt(ctx context.Context, tenantID int64, messageID string, required, available decimal.Decimal) error
    SendNotification(ctx context.Context, notification Notification) error
}
```

#### 2. Notification Types

- **LOW_BALANCE**: Warning notification when balance drops below 10 units
- **CRITICAL_BALANCE**: Critical notification when balance drops below 5 units
- **INSUFFICIENT_BALANCE**: Log entry for failed deduction attempts

#### 3. Notification Levels

- **WARNING**: For low balance (< 10 units) and insufficient balance attempts
- **CRITICAL**: For critically low balance (< 5 units)

## Integration with Balance Manager

The NotificationService is integrated into the Balance Manager in two places:

### 1. After Successful Deduction

After a successful balance deduction, the system checks if the new balance is below the warning or critical thresholds:

```go
// Check and notify for low balance after successful deduction
if notifyErr := m.notificationService.CheckAndNotifyLowBalance(ctx, op.TenantID, newBalance); notifyErr != nil {
    // Log error but don't fail the operation since deduction was successful
    m.logger.Warn("Failed to send low balance notification", ...)
}
```

**Note**: Notification failures do not cause the deduction operation to fail, as the financial transaction has already been committed successfully.

### 2. On Insufficient Balance Error

When a deduction attempt fails due to insufficient balance, the system logs the attempt:

```go
// Log insufficient balance attempt (Requirement 11.6)
if logErr := m.notificationService.LogInsufficientBalanceAttempt(ctx, op.TenantID, op.MessageID, op.Amount, currentBalance); logErr != nil {
    m.logger.Warn("Failed to log insufficient balance attempt", ...)
}
```

## Notification Thresholds

| Threshold | Value | Notification Type | Level |
|-----------|-------|-------------------|-------|
| Warning | < 10 units | LOW_BALANCE | WARNING |
| Critical | < 5 units | CRITICAL_BALANCE | CRITICAL |

**Important**: The critical threshold takes precedence. If balance is below 5 units, only a critical notification is sent (not both warning and critical).

## Current Implementation

The current implementation logs notifications using structured logging (zap). The `SendNotification` method can be extended in the future to support:

- Email notifications
- SMS notifications  
- Webhook notifications
- Database storage for notification history
- Per-tenant notification preferences
- Rate limiting to avoid notification spam

## Testing

The notification system includes comprehensive unit tests:

- `TestNotificationService_CheckAndNotifyLowBalance`: Tests threshold detection
- `TestNotificationService_LogInsufficientBalanceAttempt`: Tests insufficient balance logging
- `TestNotificationService_SendNotification`: Tests notification sending
- `TestNotificationThresholds`: Tests boundary conditions for thresholds

## Usage Example

```go
// Create notification service
logger := zap.NewProduction()
notificationService := balance.NewNotificationService(logger)

// Check and notify for low balance
err := notificationService.CheckAndNotifyLowBalance(ctx, tenantID, currentBalance)
if err != nil {
    // Handle error
}

// Log insufficient balance attempt
err = notificationService.LogInsufficientBalanceAttempt(ctx, tenantID, messageID, required, available)
if err != nil {
    // Handle error
}
```

## Future Enhancements

1. **Email Notifications**: Integrate with email service to send notifications to tenant admins
2. **SMS Notifications**: Send SMS alerts for critical balance situations
3. **Webhook Notifications**: Allow tenants to configure webhook URLs for balance alerts
4. **Notification History**: Store notifications in database for audit trail
5. **Notification Preferences**: Allow tenants to configure notification thresholds and channels
6. **Rate Limiting**: Implement rate limiting to avoid sending duplicate notifications
7. **Notification Templates**: Support customizable notification templates per tenant
8. **Multi-language Support**: Support notifications in different languages based on tenant preferences

## Files

- `notification.go`: NotificationService interface and implementation
- `notification_test.go`: Unit tests for notification system
- `manager.go`: Updated to integrate NotificationService
- `NOTIFICATION_SYSTEM.md`: This documentation file

## Related Requirements

- Requirement 11.4: Low balance warning notification
- Requirement 11.5: Critical balance warning notification  
- Requirement 11.6: Insufficient balance attempt logging

## Related Tasks

- Task 2.7: تطوير Low Balance Notification System (Current)
- Task 2.8: كتابة Unit Tests للإشعارات (Optional)
- Task 12.3: كتابة Property Test للـ Low Balance Notification
- Task 12.4: كتابة Property Test للـ Critical Balance Notification
- Task 12.5: كتابة Property Test للـ Insufficient Balance Logging
