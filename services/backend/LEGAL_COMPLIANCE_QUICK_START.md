# Legal Compliance Quick Start Guide

## Overview

Quick reference for implementing legal compliance features in the tariff pricing system.

## Setup

### 1. Run Migrations

```bash
# Run all legal compliance migrations
psql -d your_database -f migrations/029_user_consents_table.sql
psql -d your_database -f migrations/030_data_deletion_requests_table.sql
psql -d your_database -f migrations/031_fraud_detection_tables.sql
```

### 2. Initialize Managers

```go
import (
    "your-project/internal/consent"
    "your-project/internal/gdpr"
    "your-project/internal/fraud"
    "your-project/internal/audit"
)

// Initialize managers
consentManager := consent.NewManager(db)
deletionManager := gdpr.NewDataDeletionManager(db)
exportManager := gdpr.NewDataExportManager(db, "/tmp/exports")
fraudDetector := fraud.NewFraudDetector(db)
auditLogger := audit.NewLogger(db)
```

## Common Use Cases

### 1. User Registration with Consent

```go
func RegisterUser(ctx context.Context, req *RegistrationRequest) error {
    // Create user account
    userID, err := createUser(ctx, req)
    if err != nil {
        return err
    }

    // Record consent
    consentReq := &consent.ConsentRequest{
        UserID:    userID,
        Type:      consent.ConsentTermsOfService,
        Version:   "1.0",
        Granted:   req.AcceptedTerms,
        IPAddress: req.IPAddress,
        UserAgent: req.UserAgent,
    }

    err = consentManager.RecordConsent(ctx, tenantID, consentReq)
    if err != nil {
        return err
    }

    // Log consent event
    err = auditLogger.LogConsentGranted(
        ctx,
        &userID,
        &tenantID,
        string(consent.ConsentTermsOfService),
        "1.0",
        req.IPAddress,
    )

    return err
}
```

### 2. Process Payment with Fraud Check

```go
func ProcessPayment(ctx context.Context, payment *Payment) error {
    // Check for fraud
    transaction := &fraud.Transaction{
        TenantID:  payment.TenantID,
        UserID:    payment.UserID,
        Type:      "payment",
        Amount:    payment.Amount,
        Currency:  payment.Currency,
        IPAddress: payment.IPAddress,
        Country:   payment.Country,
        CreatedAt: time.Now(),
    }

    result, err := fraudDetector.CheckTransaction(ctx, transaction)
    if err != nil {
        return err
    }

    // Handle fraud detection
    if result.IsFraud {
        // Log fraud detection
        auditLogger.LogFraudDetected(
            ctx,
            &payment.UserID,
            &payment.TenantID,
            result.Alerts[0].ID,
            string(result.Severity),
            payment.IPAddress,
        )

        if result.Severity == fraud.SeverityCritical {
            return errors.New("transaction blocked due to fraud detection")
        }
    }

    // Process payment
    return processPaymentInternal(ctx, payment)
}
```

### 3. Handle Data Export Request

```go
func HandleDataExportRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := getUserID(ctx)
    tenantID := getTenantID(ctx)

    // Generate export
    downloadURL, err := exportManager.GenerateExportFile(ctx, tenantID, userID)
    if err != nil {
        http.Error(w, "Failed to generate export", http.StatusInternalServerError)
        return
    }

    // Log export event
    auditLogger.LogDataExported(
        ctx,
        &userID,
        &tenantID,
        filepath.Base(downloadURL),
        getIPAddress(r),
    )

    // Send download link via email
    sendEmail(getUserEmail(userID), downloadURL)

    json.NewEncoder(w).Encode(map[string]string{
        "message": "Export will be sent to your email",
    })
}
```

### 4. Handle Data Deletion Request

```go
func HandleDataDeletionRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := getUserID(ctx)
    tenantID := getTenantID(ctx)

    var req struct {
        Reason string `json:"reason"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Create deletion request
    requestID, err := deletionManager.RequestDeletion(
        ctx,
        tenantID,
        userID,
        req.Reason,
    )
    if err != nil {
        http.Error(w, "Failed to create deletion request", http.StatusInternalServerError)
        return
    }

    // Send confirmation email
    sendDeletionConfirmationEmail(userID, requestID)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "request_id": requestID,
        "message":    "Your deletion request has been received. Data will be deleted within 30 days.",
    })
}
```

### 5. Revoke Consent

```go
func RevokeConsent(ctx context.Context, userID, tenantID int64, consentType consent.ConsentType) error {
    // Revoke consent
    err := consentManager.RevokeConsent(ctx, tenantID, userID, consentType)
    if err != nil {
        return err
    }

    // Log revocation
    err = auditLogger.LogConsentRevoked(
        ctx,
        &userID,
        &tenantID,
        string(consentType),
        getIPAddress(ctx),
    )

    return err
}
```

## Background Jobs

### Process Pending Deletions

```go
// Run daily to process deletion requests after grace period
func ProcessPendingDeletions(ctx context.Context) error {
    requests, err := deletionManager.ListDeletionRequests(
        ctx,
        tenantID,
        gdpr.DeletionStatusPending,
    )
    if err != nil {
        return err
    }

    for _, req := range requests {
        // Check if grace period has passed (30 days)
        if time.Since(req.RequestedAt) > 30*24*time.Hour {
            err := deletionManager.ProcessDeletion(ctx, req.ID)
            if err != nil {
                log.Printf("Failed to process deletion %d: %v", req.ID, err)
                continue
            }

            // Send completion email
            sendDeletionCompletedEmail(req.UserID)
        }
    }

    return nil
}
```

### Cleanup Old Exports

```go
// Run daily to cleanup exports older than 7 days
func CleanupOldExports(ctx context.Context) error {
    return exportManager.CleanupOldExports(ctx)
}
```

### Review Fraud Alerts

```go
// Run hourly to review new fraud alerts
func ReviewFraudAlerts(ctx context.Context) error {
    alerts, err := fraudDetector.GetAlerts(
        ctx,
        tenantID,
        fraud.AlertStatusNew,
        100,
    )
    if err != nil {
        return err
    }

    for _, alert := range alerts {
        if alert.Severity == fraud.SeverityCritical {
            // Notify security team immediately
            notifySecurityTeam(alert)
        }
    }

    return nil
}
```

## API Endpoints

### Consent Management

```go
// POST /api/v1/consent
func GrantConsent(w http.ResponseWriter, r *http.Request)

// DELETE /api/v1/consent/:type
func RevokeConsent(w http.ResponseWriter, r *http.Request)

// GET /api/v1/consent
func GetConsents(w http.ResponseWriter, r *http.Request)
```

### Data Rights

```go
// POST /api/v1/gdpr/export
func RequestDataExport(w http.ResponseWriter, r *http.Request)

// POST /api/v1/gdpr/delete
func RequestDataDeletion(w http.ResponseWriter, r *http.Request)

// GET /api/v1/gdpr/deletion/:id
func GetDeletionStatus(w http.ResponseWriter, r *http.Request)
```

### Fraud Management (Admin Only)

```go
// GET /api/v1/admin/fraud/alerts
func GetFraudAlerts(w http.ResponseWriter, r *http.Request)

// PUT /api/v1/admin/fraud/alerts/:id/resolve
func ResolveAlert(w http.ResponseWriter, r *http.Request)

// GET /api/v1/admin/fraud/rules
func GetFraudRules(w http.ResponseWriter, r *http.Request)
```

## Testing

### Unit Tests

```bash
# Test all legal compliance modules
go test ./internal/consent/...
go test ./internal/gdpr/...
go test ./internal/fraud/...
go test ./internal/audit/...
```

### Integration Tests

```bash
# Test complete flows
go test ./internal/integration/legal_compliance_test.go
```

## Monitoring

### Key Metrics

```go
// Prometheus metrics
var (
    consentGrantedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "consent_granted_total",
            Help: "Total number of consents granted",
        },
        []string{"consent_type"},
    )

    dataExportRequestsTotal = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "data_export_requests_total",
            Help: "Total number of data export requests",
        },
    )

    dataDeletionRequestsTotal = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "data_deletion_requests_total",
            Help: "Total number of data deletion requests",
        },
    )

    fraudAlertsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "fraud_alerts_total",
            Help: "Total number of fraud alerts",
        },
        []string{"severity"},
    )
)
```

### Alerts

```yaml
# Prometheus alert rules
groups:
  - name: legal_compliance
    rules:
      - alert: HighDeletionRate
        expr: rate(data_deletion_requests_total[1h]) > 10
        annotations:
          summary: "High rate of deletion requests"

      - alert: CriticalFraudAlert
        expr: fraud_alerts_total{severity="critical"} > 0
        annotations:
          summary: "Critical fraud alert detected"
```

## Best Practices

1. **Always log legal events**: Use audit logger for all sensitive operations
2. **Include context**: Always include IP address and user agent
3. **Handle errors gracefully**: Don't block operations if audit logging fails
4. **Use transactions**: Log within same transaction as operation
5. **Monitor metrics**: Set up alerts for unusual patterns
6. **Regular reviews**: Review fraud alerts and deletion requests daily
7. **Test thoroughly**: Test all legal compliance features before deployment

## Troubleshooting

### Common Issues

**Issue**: Consent not recorded
**Solution**: Check database connection and RLS policies

**Issue**: Deletion request stuck in pending
**Solution**: Check background job is running and grace period has passed

**Issue**: Export file not generated
**Solution**: Check export directory permissions and disk space

**Issue**: Fraud alerts not triggering
**Solution**: Check fraud rules are enabled and thresholds are correct

## Support

For questions or issues:
- Check documentation in `internal/*/README.md`
- Review audit logs for errors
- Contact security team for fraud-related issues
- Contact legal team for compliance questions

---

**Status**: ✅ Ready for Production
**Last Updated**: 2024-01-XX
