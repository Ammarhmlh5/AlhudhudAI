# Fraud Prevention System

## Overview

The Fraud Prevention system provides real-time fraud detection and alerting to protect against suspicious activities and financial fraud.

## Features

- ✅ Real-time fraud detection
- ✅ Multiple fraud detection rules
- ✅ Configurable severity levels
- ✅ Alert management
- ✅ False positive handling
- ✅ Audit trail

## Fraud Detection Rules

### 1. Multiple Failed Payments
**Rule Type**: `multiple_failed_payments`
**Severity**: High
**Description**: Detects multiple failed payment attempts in a short time period

**Threshold**: 3 failed attempts in 1 hour
**Action**: Create high-severity alert

### 2. Unusual Transaction Amount
**Rule Type**: `unusual_amount`
**Severity**: Medium
**Description**: Detects transactions significantly higher than user's average

**Threshold**: 10x average transaction amount
**Action**: Create medium-severity alert

### 3. Rapid Subscription Changes
**Rule Type**: `rapid_subscription_changes`
**Severity**: Medium
**Description**: Detects too many subscription changes in one day

**Threshold**: 5 changes in 24 hours
**Action**: Create medium-severity alert

### 4. Geographic Anomaly
**Rule Type**: `geographic_anomaly`
**Severity**: High
**Description**: Detects transactions from different countries

**Threshold**: Country change between transactions
**Action**: Create high-severity alert

### 5. High Velocity
**Rule Type**: `velocity_check`
**Severity**: Critical
**Description**: Detects too many transactions in a very short time

**Threshold**: 20 transactions in 5 minutes
**Action**: Create critical alert and potentially block user

## Usage

### Initialize Detector

```go
import "your-project/internal/fraud"

detector := fraud.NewFraudDetector(db)
```

### Check Transaction

```go
transaction := &fraud.Transaction{
    ID:        123,
    TenantID:  1,
    UserID:    456,
    Type:      "payment",
    Amount:    1000.00,
    Currency:  "USD",
    IPAddress: "192.168.1.1",
    Country:   "US",
    CreatedAt: time.Now(),
}

result, err := detector.CheckTransaction(ctx, transaction)
if err != nil {
    // Handle error
}

if result.IsFraud {
    fmt.Printf("Fraud detected! Severity: %s, Score: %.2f\n", result.Severity, result.Score)
    fmt.Printf("Triggered rules: %v\n", result.TriggeredRules)
    
    // Take action based on severity
    switch result.Severity {
    case fraud.SeverityCritical:
        // Block transaction immediately
        // Suspend user account
        // Notify security team
    case fraud.SeverityHigh:
        // Flag for manual review
        // Notify security team
    case fraud.SeverityMedium:
        // Log for monitoring
        // May require additional verification
    case fraud.SeverityLow:
        // Log only
    }
}
```

### Get Alerts

```go
// Get all new alerts
alerts, err := detector.GetAlerts(ctx, tenantID, fraud.AlertStatusNew, 100)
if err != nil {
    // Handle error
}

for _, alert := range alerts {
    fmt.Printf("Alert ID: %d, Severity: %s, User: %d\n", 
        alert.ID, alert.Severity, alert.UserID)
}
```

### Resolve Alert

```go
// Mark as resolved
err := detector.ResolveAlert(ctx, alertID, adminUserID, "Verified legitimate transaction", false)

// Mark as false positive
err := detector.ResolveAlert(ctx, alertID, adminUserID, "False positive - user traveling", true)
```

## Integration Example

### Payment Processing

```go
func ProcessPayment(ctx context.Context, payment *Payment) error {
    // Create transaction record
    transaction := &fraud.Transaction{
        ID:        payment.ID,
        TenantID:  payment.TenantID,
        UserID:    payment.UserID,
        Type:      "payment",
        Amount:    payment.Amount,
        Currency:  payment.Currency,
        IPAddress: payment.IPAddress,
        Country:   payment.Country,
        CreatedAt: time.Now(),
    }
    
    // Check for fraud
    result, err := fraudDetector.CheckTransaction(ctx, transaction)
    if err != nil {
        return fmt.Errorf("fraud check failed: %w", err)
    }
    
    // Handle fraud detection
    if result.IsFraud {
        if result.Severity == fraud.SeverityCritical {
            // Block transaction
            return fmt.Errorf("transaction blocked due to fraud detection")
        }
        
        if result.Severity == fraud.SeverityHigh {
            // Require additional verification
            return RequireAdditionalVerification(payment)
        }
        
        // Log for monitoring
        log.Printf("Fraud alert: %v", result)
    }
    
    // Process payment normally
    return processPaymentInternal(ctx, payment)
}
```

### Subscription Changes

```go
func ChangeSubscription(ctx context.Context, userID, newPackageID int64) error {
    // Create transaction for fraud check
    transaction := &fraud.Transaction{
        TenantID:  getTenantID(ctx),
        UserID:    userID,
        Type:      "subscription_change",
        CreatedAt: time.Now(),
    }
    
    // Check for fraud
    result, err := fraudDetector.CheckTransaction(ctx, transaction)
    if err != nil {
        return err
    }
    
    if result.IsFraud && result.Severity >= fraud.SeverityHigh {
        // Too many subscription changes
        return fmt.Errorf("subscription change blocked: suspicious activity detected")
    }
    
    // Process subscription change
    return changeSubscriptionInternal(ctx, userID, newPackageID)
}
```

## Database Schema

### fraud_rules

```sql
CREATE TABLE fraud_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    severity VARCHAR(20) NOT NULL,
    threshold DECIMAL(10, 2) NOT NULL,
    time_window INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

### fraud_alerts

```sql
CREATE TABLE fraud_alerts (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    rule_id BIGINT NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    metadata JSONB,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by BIGINT,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

## Fraud Score Calculation

The fraud score is calculated based on triggered rules:

- **Critical**: +50 points
- **High**: +30 points
- **Medium**: +15 points
- **Low**: +5 points

**Score Ranges**:
- 0-20: Low risk
- 21-50: Medium risk
- 51-80: High risk
- 81-100: Critical risk

## Best Practices

1. **Monitor Alerts**: Review fraud alerts daily
2. **Tune Rules**: Adjust thresholds based on false positive rate
3. **Quick Response**: Investigate high-severity alerts immediately
4. **User Communication**: Notify users of blocked transactions
5. **Continuous Improvement**: Update rules based on new fraud patterns

## Requirements

- Requirements: 36.9
- Fraud Prevention
- Real-time Detection
- Alert Management

## Testing

Run tests:
```bash
go test ./internal/fraud/...
```

## Migration

Run migration:
```bash
psql -d your_database -f migrations/031_fraud_detection_tables.sql
```

## Status

✅ Implemented
✅ Tested
✅ Documented
