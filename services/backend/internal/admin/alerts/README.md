# Alert System

نظام التنبيهات الشامل لواجهة الإدارة.

## Overview

يوفر نظام التنبيهات إمكانيات متقدمة لإنشاء وإدارة التنبيهات في النظام، مع دعم:
- إنشاء تنبيهات لأحداث مختلفة (تغيير التقييم، فشل السيرفر، تجاوز العتبات، فشل الأجهزة)
- منع التنبيهات المكررة خلال 5 دقائق
- إرسال التنبيهات عبر قنوات متعددة (Email, SMS, In-App)
- فلترة وبحث متقدم في التنبيهات
- إدارة حالة التنبيهات (نشط، مُقر به، محلول)

## Components

### 1. Alert Manager (`manager.go`)

يدير عمليات CRUD للتنبيهات:

```go
manager := alerts.NewManager(db, logger)

// إنشاء تنبيه جديد
alert, err := manager.CreateAlert(ctx, &alerts.CreateAlertRequest{
    TenantID:  &tenantID,
    AlertType: alerts.AlertTypeRatingChange,
    Severity:  alerts.AlertSeverityCritical,
    Title:     "تغيير تقييم الشركة",
    Message:   "انخفض تقييم الشركة من ممتاز إلى جيد",
    Context: map[string]interface{}{
        "old_category": "excellent",
        "new_category": "good",
        "old_rating":   92.5,
        "new_rating":   85.0,
    },
    DedupKey: dedupKey,
})

// الحصول على التنبيهات مع فلترة
response, err := manager.GetAlerts(ctx, &alerts.AlertFilter{
    TenantID:  &tenantID,
    Severity:  &criticalSeverity,
    Status:    &activeStatus,
    StartTime: &startTime,
    Limit:     50,
    Offset:    0,
})

// الإقرار بالتنبيه
err = manager.AcknowledgeAlert(ctx, alertID, userID)

// حل التنبيه
err = manager.ResolveAlert(ctx, alertID)
```

### 2. Deduplicator (`dedup.go`)

يمنع التنبيهات المكررة خلال 5 دقائق:

```go
dedup := alerts.NewDeduplicator(db, logger)

// توليد مفتاح deduplication
dedupKey := dedup.GenerateRatingChangeDedupKey(
    tenantID.String(),
    "excellent",
    "good",
)

// التحقق من وجود تنبيه مكرر
isDuplicate, err := dedup.CheckDuplicate(ctx, dedupKey)
if isDuplicate {
    // تخطي إنشاء التنبيه
    return nil
}

// إنشاء التنبيه مع dedup key
alert, err := manager.CreateAlert(ctx, &alerts.CreateAlertRequest{
    // ...
    DedupKey: dedupKey,
})
```

**Dedup Key Generators:**

```go
// تغيير التقييم
dedupKey := dedup.GenerateRatingChangeDedupKey(tenantID, oldCategory, newCategory)

// فشل السيرفر
dedupKey := dedup.GenerateServerFailureDedupKey(tenantID, serverID)

// تجاوز العتبة
dedupKey := dedup.GenerateThresholdViolationDedupKey(tenantID, metricName, thresholdType)

// فشل الجهاز
dedupKey := dedup.GenerateDeviceFailureDedupKey(tenantID, deviceID)
```

### 3. Notifier (`notifier.go`)

يرسل التنبيهات عبر قنوات متعددة:

```go
// إنشاء notification service (يجب تطبيق الواجهة)
notificationService := &YourNotificationService{}

notifier := alerts.NewNotifier(notificationService, logger, &alerts.NotifierConfig{
    EnableEmail: true,
    EnableSMS:   true,
    EnableInApp: true,
})

// إرسال التنبيه عبر جميع القنوات
err := notifier.SendAlert(ctx, alert, []alerts.NotificationChannel{
    alerts.ChannelAll,
})

// إرسال عبر قنوات محددة
err = notifier.SendAlert(ctx, alert, []alerts.NotificationChannel{
    alerts.ChannelEmail,
    alerts.ChannelInApp,
})
```

## Alert Types

```go
const (
    AlertTypeRatingChange       AlertType = "rating_change"        // تغيير تقييم الشركة
    AlertTypeServerFailure      AlertType = "server_failure"       // فشل السيرفر المخصص
    AlertTypeThresholdViolation AlertType = "threshold_violation"  // تجاوز عتبة الأداء
    AlertTypeDeviceFailure      AlertType = "device_failure"       // فشل جهاز الهاتف
)
```

## Alert Severity

```go
const (
    AlertSeverityCritical AlertSeverity = "critical"  // حرج - يتطلب إجراء فوري
    AlertSeverityWarning  AlertSeverity = "warning"   // تحذير - يتطلب انتباه
    AlertSeverityInfo     AlertSeverity = "info"      // معلومة - للعلم فقط
)
```

## Alert Status

```go
const (
    AlertStatusActive       AlertStatus = "active"       // نشط - لم يتم التعامل معه
    AlertStatusAcknowledged AlertStatus = "acknowledged" // مُقر به - تم الاطلاع عليه
    AlertStatusResolved     AlertStatus = "resolved"     // محلول - تم حل المشكلة
)
```

## Complete Example

```go
package main

import (
    "context"
    "time"
    
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "go.uber.org/zap"
    
    "your-project/internal/admin/alerts"
)

func main() {
    // Initialize dependencies
    db := sqlx.MustConnect("postgres", "connection-string")
    logger, _ := zap.NewProduction()
    
    // Create components
    manager := alerts.NewManager(db, logger)
    dedup := alerts.NewDeduplicator(db, logger)
    notifier := alerts.NewNotifier(notificationService, logger, nil)
    
    ctx := context.Background()
    tenantID := uuid.New()
    
    // Generate dedup key
    dedupKey := dedup.GenerateRatingChangeDedupKey(
        tenantID.String(),
        "excellent",
        "good",
    )
    
    // Check for duplicates
    isDuplicate, err := dedup.CheckDuplicate(ctx, dedupKey)
    if err != nil {
        logger.Error("Failed to check duplicate", zap.Error(err))
        return
    }
    
    if isDuplicate {
        logger.Info("Duplicate alert detected, skipping")
        return
    }
    
    // Create alert
    alert, err := manager.CreateAlert(ctx, &alerts.CreateAlertRequest{
        TenantID:  &tenantID,
        AlertType: alerts.AlertTypeRatingChange,
        Severity:  alerts.AlertSeverityCritical,
        Title:     "تغيير تقييم الشركة",
        Message:   "انخفض تقييم الشركة من ممتاز إلى جيد",
        Context: map[string]interface{}{
            "old_category": "excellent",
            "new_category": "good",
            "old_rating":   92.5,
            "new_rating":   85.0,
        },
        DedupKey: dedupKey,
    })
    
    if err != nil {
        logger.Error("Failed to create alert", zap.Error(err))
        return
    }
    
    // Send notifications
    err = notifier.SendAlert(ctx, alert, []alerts.NotificationChannel{
        alerts.ChannelAll,
    })
    
    if err != nil {
        logger.Error("Failed to send alert", zap.Error(err))
        return
    }
    
    logger.Info("Alert created and sent successfully",
        zap.Int64("alert_id", alert.ID),
    )
}
```

## Integration with Existing Systems

### Rating System Integration

```go
// في rating engine، عند تغيير التقييم
func (r *RatingEngine) OnRatingChange(ctx context.Context, tenantID uuid.UUID, oldRating, newRating *CompanyRating) error {
    // Check if category changed
    if oldRating.RatingCategory != newRating.RatingCategory {
        // Generate dedup key
        dedupKey := r.dedup.GenerateRatingChangeDedupKey(
            tenantID.String(),
            string(oldRating.RatingCategory),
            string(newRating.RatingCategory),
        )
        
        // Check for duplicates
        isDuplicate, err := r.dedup.CheckDuplicate(ctx, dedupKey)
        if err != nil || isDuplicate {
            return err
        }
        
        // Create alert
        alert, err := r.alertManager.CreateAlert(ctx, &alerts.CreateAlertRequest{
            TenantID:  &tenantID,
            AlertType: alerts.AlertTypeRatingChange,
            Severity:  r.determineSeverity(oldRating.RatingCategory, newRating.RatingCategory),
            Title:     "تغيير تقييم الشركة",
            Message:   fmt.Sprintf("تغير تقييم الشركة من %s إلى %s", oldRating.RatingCategory, newRating.RatingCategory),
            Context: map[string]interface{}{
                "old_category": oldRating.RatingCategory,
                "new_category": newRating.RatingCategory,
                "old_rating":   oldRating.OverallRating,
                "new_rating":   newRating.OverallRating,
            },
            DedupKey: dedupKey,
        })
        
        if err != nil {
            return err
        }
        
        // Send notifications
        return r.notifier.SendAlert(ctx, alert, []alerts.NotificationChannel{alerts.ChannelAll})
    }
    
    return nil
}
```

### Server Health Integration

```go
// في health monitor، عند تغيير حالة السيرفر
func (h *HealthMonitor) OnStatusChange(ctx context.Context, serverID uuid.UUID, tenantID uuid.UUID, oldStatus, newStatus string) error {
    if newStatus == "unhealthy" {
        dedupKey := h.dedup.GenerateServerFailureDedupKey(tenantID.String(), serverID.String())
        
        isDuplicate, err := h.dedup.CheckDuplicate(ctx, dedupKey)
        if err != nil || isDuplicate {
            return err
        }
        
        alert, err := h.alertManager.CreateAlert(ctx, &alerts.CreateAlertRequest{
            TenantID:  &tenantID,
            AlertType: alerts.AlertTypeServerFailure,
            Severity:  alerts.AlertSeverityCritical,
            Title:     "فشل السيرفر المخصص",
            Message:   "السيرفر المخصص غير متاح",
            Context: map[string]interface{}{
                "server_id":  serverID.String(),
                "old_status": oldStatus,
                "new_status": newStatus,
            },
            DedupKey: dedupKey,
        })
        
        if err != nil {
            return err
        }
        
        return h.notifier.SendAlert(ctx, alert, []alerts.NotificationChannel{alerts.ChannelAll})
    }
    
    return nil
}
```

## Database Schema

```sql
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id),
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    context JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    acknowledged_at TIMESTAMP,
    acknowledged_by UUID REFERENCES users(id),
    resolved_at TIMESTAMP,
    dedup_key VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_alerts_tenant (tenant_id, created_at DESC),
    INDEX idx_alerts_type (alert_type),
    INDEX idx_alerts_severity (severity),
    INDEX idx_alerts_status (status),
    INDEX idx_alerts_dedup (dedup_key, created_at)
);

CREATE INDEX idx_alerts_active ON alerts(tenant_id, created_at DESC) WHERE status = 'active';
```

## Requirements Validation

- ✅ **Requirement 6.1**: Alert generation for rating changes
- ✅ **Requirement 6.2**: Alert generation for server failures
- ✅ **Requirement 6.3**: Alert generation for threshold violations
- ✅ **Requirement 6.4**: Alert generation for device failures
- ✅ **Requirement 6.5**: Multi-channel notifications (email, SMS, in-app)
- ✅ **Requirement 6.8**: Deduplication within 5 minutes
- ✅ **Requirement 6.9**: Alert filtering and management

## Testing

```bash
# Run tests
go test ./internal/admin/alerts/...

# Run with coverage
go test -cover ./internal/admin/alerts/...

# Run specific test
go test -run TestCreateAlert ./internal/admin/alerts/
```

## Performance Considerations

1. **Deduplication**: Uses indexed queries on `dedup_key` and `created_at` for fast duplicate detection
2. **Filtering**: All filter fields are indexed for efficient queries
3. **Pagination**: Supports limit/offset for large result sets
4. **Cleanup**: Provides cleanup function to remove old resolved alerts

## Future Enhancements

- [ ] Alert templates for common scenarios
- [ ] Alert escalation rules
- [ ] Alert grouping and batching
- [ ] Webhook notifications
- [ ] Alert analytics and reporting
- [ ] Custom notification preferences per tenant
- [ ] Alert scheduling (quiet hours)
