# Balance Manager Package

## Overview

The Balance Manager package provides atomic balance operations with race condition protection for the SMS Gateway system. It implements critical financial protection mechanisms to prevent negative balances and ensure accurate accounting.

The package consists of two main components:
1. **Balance Manager**: Handles atomic balance operations (deduct, refund, check)
2. **History Logger**: Provides comprehensive audit trail for all financial operations

## Features

- **Atomic Operations**: All balance operations use atomic SQL with WHERE conditions
- **Race Condition Protection**: SERIALIZABLE isolation level prevents concurrent update issues
- **Retry Logic**: Exponential backoff for handling concurrent updates
- **Complete Audit Trail**: Every operation is logged in balance_history via History Logger
- **Tenant Isolation**: All operations include tenant_id for complete isolation
- **IP Address Logging**: Records IP address for security audit (Requirement 20.2)
- **Modular Design**: History Logger can be used independently for other financial operations
- **Low Balance Notifications**: Automatic warnings when balance drops below thresholds (Requirements 11.4, 11.5)
- **Insufficient Balance Logging**: Tracks all failed deduction attempts (Requirement 11.6)

## Requirements Implemented

### Balance Manager
- **Requirement 1.1**: Atomic UPDATE with WHERE condition for balance deduction
- **Requirement 1.2**: Reject deduction if balance insufficient
- **Requirement 1.3**: Atomic operations produce correct final balance
- **Requirement 1.5**: SERIALIZABLE transaction isolation level
- **Requirement 2.2**: Atomic refund for failed messages
- **Requirement 2.5**: Atomic INCREMENT for refunds

### History Logger
- **Requirement 10.1**: Log every balance deduction/refund operation
- **Requirement 10.2**: Log tenant_id, message_id, old_balance, new_balance, amount, type, timestamp
- **Requirement 10.3**: Log refund operations with type=REFUND
- **Requirement 10.4**: Log reason for each operation
- **Requirement 10.5**: Log user/system that performed the operation (created_by)
- **Requirement 20.2**: Log IP address for financial operations security audit

### Notification Service
- **Requirement 11.4**: Send warning notification when balance < 10 units
- **Requirement 11.5**: Send critical notification when balance < 5 units
- **Requirement 11.6**: Log all failed deduction attempts due to insufficient balance

## Usage

### Creating a Balance Manager

```go
import (
    "github.com/al-hudhud/backend/internal/balance"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
)

// Create manager
pool := // ... your pgxpool.Pool
logger := // ... your zap.Logger
manager := balance.NewManager(pool, logger)
```

### Deducting Balance

```go
op := balance.BalanceOperation{
    TenantID:  1,
    MessageID: "msg-123",
    Amount:    decimal.NewFromFloat(10.5),
    Type:      balance.OperationTypeDeduct,
    Reason:    "MESSAGE_SENT",
    CreatedBy: "system",
    IPAddress: "192.168.1.1",
}

result, err := manager.DeductBalance(ctx, op)
if err != nil {
    // Handle error
    if balErr, ok := err.(*balance.BalanceError); ok {
        switch balErr.Type {
        case balance.ErrorInsufficientBalance:
            // Handle insufficient balance
        case balance.ErrorTenantNotFound:
            // Handle tenant not found
        case balance.ErrorConcurrentUpdate:
            // Handle concurrent update (rare after retries)
        }
    }
    return err
}

// Success
fmt.Printf("Old Balance: %s, New Balance: %s\n", 
    result.OldBalance.String(), 
    result.NewBalance.String())
```

### Refunding Balance

```go
op := balance.BalanceOperation{
    TenantID:  1,
    MessageID: "msg-123",
    Amount:    decimal.NewFromFloat(10.5),
    Type:      balance.OperationTypeRefund,
    Reason:    "MESSAGE_FAILED",
    CreatedBy: "system",
}

result, err := manager.RefundBalance(ctx, op)
if err != nil {
    // Handle error
    return err
}

// Success - balance refunded
```

### Checking Balance

```go
// Get current balance
balance, err := manager.GetBalance(ctx, tenantID)
if err != nil {
    return err
}

// Check if sufficient
sufficient, err := manager.CheckSufficient(ctx, tenantID, amount)
if err != nil {
    return err
}

if !sufficient {
    // Handle insufficient balance
}
```

## Error Handling

The package provides typed errors for better error handling:

```go
type ErrorType string

const (
    ErrorInsufficientBalance ErrorType = "INSUFFICIENT_BALANCE"
    ErrorTenantNotFound      ErrorType = "TENANT_NOT_FOUND"
    ErrorConcurrentUpdate    ErrorType = "CONCURRENT_UPDATE"
    ErrorDatabaseError       ErrorType = "DATABASE_ERROR"
)
```

## Atomic Operations

### Deduct Operation

```sql
UPDATE tenants 
SET 
    balance = balance - $1,
    updated_at = NOW()
WHERE 
    id = $2 
    AND balance >= $1  -- Critical: prevents negative balance
    AND deleted_at IS NULL
RETURNING 
    id,
    balance AS new_balance,
    (balance + $1) AS old_balance;
```

### Refund Operation

```sql
UPDATE tenants 
SET 
    balance = balance + $1,
    updated_at = NOW()
WHERE 
    id = $2 
    AND deleted_at IS NULL
RETURNING 
    id,
    balance AS new_balance,
    (balance - $1) AS old_balance;
```

## Retry Logic

The manager implements exponential backoff for concurrent updates:

- **Max Retries**: 3 attempts
- **Base Delay**: 10ms
- **Backoff**: Exponential (10ms, 20ms, 40ms)

## Balance History

Every operation is logged in the `balance_history` table:

```go
type BalanceHistory struct {
    ID         int64
    TenantID   int64
    MessageID  *string
    OldBalance decimal.Decimal
    NewBalance decimal.Decimal
    Amount     decimal.Decimal
    Type       OperationType
    Reason     string
    CreatedBy  string
    IPAddress  *string
    CreatedAt  time.Time
}
```

## History Logger Component

The History Logger is a separate component that can be used independently for logging financial operations.

### Creating a History Logger

```go
import (
    "github.com/al-hudhud/backend/internal/balance"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"
)

// Create history logger
pool := // ... your pgxpool.Pool
logger := // ... your zap.Logger
historyLogger := balance.NewHistoryLogger(pool, logger)
```

### Logging Operations

The History Logger provides methods for different operation types:

```go
// Log a deduction
entry := balance.HistoryEntry{
    TenantID:   1,
    MessageID:  "msg-123",
    OldBalance: decimal.NewFromFloat(100.0),
    NewBalance: decimal.NewFromFloat(95.0),
    Amount:     decimal.NewFromFloat(5.0),
    Reason:     "MESSAGE_SENT",
    CreatedBy:  "system",
    IPAddress:  "192.168.1.1", // Required for security audit
}

// Must be called within a transaction
tx, _ := pool.Begin(ctx)
err := historyLogger.LogDeduction(ctx, tx, entry)
if err != nil {
    tx.Rollback(ctx)
    return err
}
tx.Commit(ctx)
```

### Available Methods

```go
// Log different operation types
historyLogger.LogDeduction(ctx, tx, entry)  // For balance deductions
historyLogger.LogRefund(ctx, tx, entry)     // For refunds
historyLogger.LogCharge(ctx, tx, entry)     // For adding balance
historyLogger.LogAdjustment(ctx, tx, entry) // For manual adjustments

// Retrieve history
history, err := historyLogger.GetHistory(ctx, tenantID, limit)
```

### History Entry Structure

```go
type HistoryEntry struct {
    TenantID   int64           // Required: Tenant identifier
    MessageID  string          // Optional: Related message ID
    OldBalance decimal.Decimal // Required: Balance before operation
    NewBalance decimal.Decimal // Required: Balance after operation
    Amount     decimal.Decimal // Required: Operation amount
    Type       OperationType   // Set automatically by Log* methods
    Reason     string          // Required: Human-readable reason
    CreatedBy  string          // Required: User/system identifier
    IPAddress  string          // Optional but recommended: IP for audit
}
```

### Operation Types

```go
const (
    OperationTypeDeduct     OperationType = "DEDUCT"     // Balance deduction
    OperationTypeRefund     OperationType = "REFUND"     // Balance refund
    OperationTypeCharge     OperationType = "CHARGE"     // Add balance
    OperationTypeAdjustment OperationType = "ADJUSTMENT" // Manual correction
)
```

## Notification Service

The Notification Service provides automatic low balance alerts and tracks insufficient balance attempts.

### Features

- **Automatic Threshold Monitoring**: Checks balance after every deduction
- **Two-Level Alerts**: Warning (< 10 units) and Critical (< 5 units)
- **Failed Attempt Logging**: Records all insufficient balance attempts
- **Extensible Design**: Easy to add email, SMS, or webhook notifications

### Usage

```go
// Create notification service
notificationService := balance.NewNotificationService(logger)

// Check and notify for low balance (called automatically by Balance Manager)
err := notificationService.CheckAndNotifyLowBalance(ctx, tenantID, currentBalance)

// Log insufficient balance attempt (called automatically on deduction failure)
err = notificationService.LogInsufficientBalanceAttempt(ctx, tenantID, messageID, required, available)
```

### Notification Thresholds

| Threshold | Value | Type | Level |
|-----------|-------|------|-------|
| Warning | < 10 units | LOW_BALANCE | WARNING |
| Critical | < 5 units | CRITICAL_BALANCE | CRITICAL |

**Note**: Critical threshold takes precedence. If balance < 5, only critical notification is sent.

### Notification Types

```go
const (
    NotificationTypeLowBalance         NotificationType = "LOW_BALANCE"
    NotificationTypeCriticalBalance    NotificationType = "CRITICAL_BALANCE"
    NotificationTypeInsufficientBalance NotificationType = "INSUFFICIENT_BALANCE"
)
```

### Integration with Balance Manager

The notification service is automatically integrated:

1. **After Successful Deduction**: Checks if new balance is below thresholds
2. **On Insufficient Balance**: Logs the failed attempt with details

See [NOTIFICATION_SYSTEM.md](./NOTIFICATION_SYSTEM.md) for detailed documentation.

```

### Retrieving History

```go
// Get last 100 operations for a tenant
history, err := historyLogger.GetHistory(ctx, tenantID, 100)
if err != nil {
    return err
}

for _, h := range history {
    fmt.Printf("Operation: %s, Amount: %s, Balance: %s -> %s\n",
        h.Type, h.Amount, h.OldBalance, h.NewBalance)
}
```

### Integration with Balance Manager

The Balance Manager automatically uses History Logger for all operations:

```go
// When you call DeductBalance or RefundBalance,
// the history is automatically logged
manager := balance.NewManager(pool, logger)
result, err := manager.DeductBalance(ctx, op)
// History is logged automatically within the same transaction
```

## Testing

Run unit tests:

```bash
go test ./internal/balance -v
```

### Balance Manager Tests
- TestBalanceOperation_Validation
- TestBalanceError_Types
- TestOperationType_Constants
- TestBalanceResult_Structure
- TestManager_CheckSufficient_Logic
- TestBalanceOperation_MessageIDOptional
- TestBalanceHistory_Structure

### History Logger Tests
- TestHistoryLogger_LogDeduction
- TestHistoryLogger_LogRefund
- TestHistoryLogger_LogCharge
- TestHistoryLogger_LogAdjustment
- TestHistoryLogger_MultipleEntries
- TestHistoryLogger_InvalidTenantID
- TestHistoryLogger_GetHistoryWithLimit
- TestHistoryLogger_IPAddressLogging

## Thread Safety

The Balance Manager is thread-safe:

- Uses database-level atomic operations
- SERIALIZABLE isolation level prevents phantom reads
- Retry logic handles concurrent updates
- No shared mutable state in the manager

## Performance Considerations

- **Connection Pool**: Uses pgxpool for efficient connection management
- **Atomic Operations**: Single UPDATE statement (no SELECT-UPDATE pattern)
- **Minimal Locks**: Database handles locking automatically
- **Fast Retries**: Exponential backoff minimizes retry overhead

## Security

- **Tenant Isolation**: All queries include `WHERE tenant_id = ?`
- **IP Logging**: Records IP address for audit trail
- **Immutable History**: balance_history records are never updated
- **Input Validation**: Validates all inputs before database operations

## Future Enhancements

Potential improvements for future versions:

1. ~~Low balance notifications (Requirements 11.4, 11.5)~~ ✅ **Implemented in Task 2.7**
2. ~~Balance alerts and thresholds~~ ✅ **Implemented in Task 2.7**
3. Enhanced notification channels (email, SMS, webhooks)
4. Notification preferences per tenant
5. Metrics and monitoring integration
6. Batch balance operations
7. Balance snapshots for reporting
8. Rate limiting for notifications to avoid spam

## Related Packages

- `internal/database`: Database connection pool
- `internal/config`: Configuration management
- `internal/models`: Data models (if needed)

## License

Internal package for ALHudhudAI SMS Gateway system.
