## Referral System - نظام الإحالة

نظام شامل لإدارة الإحالات (Referrals) يسمح للعملاء بدعوة أصدقائهم والحصول على مكافآت.

## المتطلبات المغطاة

- **Requirement 33.1**: توليد كود إحالة فريد لكل عميل
- **Requirement 33.2**: السماح للعميل بمشاركة كود الإحالة
- **Requirement 33.3**: تسجيل العلاقة بين المُحيل والمُحال
- **Requirement 33.4**: منح المُحيل مكافأة عند اشتراك المُحال
- **Requirement 33.5**: دعم تخصيص نوع المكافأة
- **Requirement 33.6**: دعم تحديد شروط المكافأة
- **Requirement 33.7**: عرض عدد الإحالات والمكافآت للعميل
- **Requirement 33.8**: تسجيل جميع عمليات الإحالة والمكافآت
- **Requirement 33.9**: توفير تقرير بأداء برنامج الإحالة

## المكونات

### 1. Types (types.go)

#### ReferralStatus
حالات الإحالة:
- `pending`: معلقة (المُحال سجّل لكن لم يستوفِ الشروط)
- `completed`: مكتملة (المُحال استوفى الشروط)
- `rewarded`: تم منح المكافأة
- `cancelled`: ملغاة

#### RewardType
أنواع المكافآت:
- `balance`: رصيد مالي
- `messages`: رسائل مجانية
- `discount`: خصم بالنسبة المئوية

#### RewardCondition
شروط منح المكافأة:
- `immediate`: فوري عند التسجيل
- `first_subscription`: عند أول اشتراك
- `first_payment`: عند أول دفعة

### 2. Manager (manager.go)

#### الوظائف الأساسية

##### GenerateReferralCode
```go
func (m *Manager) GenerateReferralCode(ctx context.Context, tenantID uuid.UUID) (string, error)
```
يولّد كود إحالة فريد لعميل.

**مثال:**
```go
code, err := manager.GenerateReferralCode(ctx, tenantID)
// code = "REF-A1B2C3D4"
```

##### ValidateReferralCode
```go
func (m *Manager) ValidateReferralCode(ctx context.Context, code string) (*ReferralCodeInfo, error)
```
يتحقق من صحة كود الإحالة.

**مثال:**
```go
info, err := manager.ValidateReferralCode(ctx, "REF-A1B2C3D4")
if err != nil {
    // كود غير صحيح أو منتهي
}
```

##### ProcessReferral
```go
func (m *Manager) ProcessReferral(ctx context.Context, req *CreateReferralRequest) (*Referral, error)
```
يعالج إحالة جديدة عند تسجيل عميل بكود إحالة.

**مثال:**
```go
req := &CreateReferralRequest{
    ReferrerTenantID: referrerID,
    RefereeTenantID:  refereeID,
    ReferralCode:     "REF-A1B2C3D4",
}

referral, err := manager.ProcessReferral(ctx, req)
```

##### MarkConditionMet
```go
func (m *Manager) MarkConditionMet(ctx context.Context, referralID uuid.UUID) error
```
يحدّث حالة الإحالة عند استيفاء الشرط.

**مثال:**
```go
// عند أول دفعة للمُحال
err := manager.MarkConditionMet(ctx, referralID)
```

##### GrantReward
```go
func (m *Manager) GrantReward(ctx context.Context, req *GrantRewardRequest) error
```
يمنح المكافأة للمُحيل.

**مثال:**
```go
req := &GrantRewardRequest{
    ReferralID: referralID,
}

err := manager.GrantReward(ctx, req)
```

##### GetReferralStats
```go
func (m *Manager) GetReferralStats(ctx context.Context, referrerID uuid.UUID) (*ReferralStats, error)
```
يسترجع إحصائيات الإحالات لمُحيل.

**مثال:**
```go
stats, err := manager.GetReferralStats(ctx, referrerID)
// stats.TotalReferrals = 10
// stats.RewardedReferrals = 7
// stats.TotalRewardsGranted = 70.00
```

##### GetReferralPerformance
```go
func (m *Manager) GetReferralPerformance(ctx context.Context) (*ReferralPerformance, error)
```
يسترجع مقاييس الأداء الإجمالية لبرنامج الإحالة.

**مثال:**
```go
perf, err := manager.GetReferralPerformance(ctx)
// perf.ConversionRate = 75.5%
// perf.AverageReward = 10.00
```

##### ProcessPendingRewards
```go
func (m *Manager) ProcessPendingRewards(ctx context.Context) (int, error)
```
يعالج جميع المكافآت المعلقة (للـ Cron Job).

**مثال:**
```go
processed, err := manager.ProcessPendingRewards(ctx)
// processed = 15 (عدد المكافآت الممنوحة)
```

## تدفق العمل (Workflow)

### 1. توليد كود الإحالة
```
1. العميل يطلب كود إحالة
2. النظام يولّد كود فريد (REF-XXXXXXXX)
3. العميل يشارك الكود مع أصدقائه
```

### 2. التسجيل باستخدام كود إحالة
```
1. عميل جديد يدخل كود الإحالة عند التسجيل
2. النظام يتحقق من صحة الكود
3. النظام ينشئ سجل إحالة بحالة 'pending'
4. النظام يحدد نوع المكافأة والشرط
```

### 3. استيفاء شرط المكافأة
```
1. المُحال يقوم بأول دفعة/اشتراك
2. النظام يستدعي MarkConditionMet()
3. النظام يحدّث status = 'completed'
```

### 4. منح المكافأة
```
1. Cron Job يستدعي ProcessPendingRewards()
2. النظام يمنح المكافأة للمُحيل
3. النظام يحدّث status = 'rewarded'
4. النظام يرسل إشعار للمُحيل
```

## أمثلة الاستخدام

### مثال 1: تسجيل عميل جديد بكود إحالة

```go
package main

import (
    "context"
    "fmt"
    "github.com/google/uuid"
    "internal/referral"
)

func handleSignup(referralCode string, newTenantID uuid.UUID) error {
    ctx := context.Background()
    
    // 1. التحقق من كود الإحالة
    codeInfo, err := referralManager.ValidateReferralCode(ctx, referralCode)
    if err != nil {
        return fmt.Errorf("invalid referral code: %w", err)
    }
    
    // 2. إنشاء الإحالة
    req := &referral.CreateReferralRequest{
        ReferrerTenantID: codeInfo.ReferrerTenantID,
        RefereeTenantID:  newTenantID,
        ReferralCode:     referralCode,
    }
    
    ref, err := referralManager.ProcessReferral(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to process referral: %w", err)
    }
    
    fmt.Printf("Referral created: %s\n", ref.ID)
    return nil
}
```

### مثال 2: معالجة أول دفعة

```go
func handleFirstPayment(tenantID uuid.UUID) error {
    ctx := context.Background()
    
    // 1. البحث عن الإحالة
    query := `
        SELECT id FROM referrals 
        WHERE referee_tenant_id = $1 
          AND reward_condition = 'first_payment'
          AND condition_met = false
    `
    
    var referralID uuid.UUID
    err := db.GetContext(ctx, &referralID, query, tenantID)
    if err != nil {
        // لا توجد إحالة أو تم استيفاء الشرط مسبقاً
        return nil
    }
    
    // 2. تحديث حالة الإحالة
    err = referralManager.MarkConditionMet(ctx, referralID)
    if err != nil {
        return fmt.Errorf("failed to mark condition met: %w", err)
    }
    
    return nil
}
```

### مثال 3: Cron Job لمعالجة المكافآت

```go
func processPendingRewardsCron() {
    ctx := context.Background()
    
    processed, err := referralManager.ProcessPendingRewards(ctx)
    if err != nil {
        log.Printf("Error processing pending rewards: %v", err)
        return
    }
    
    log.Printf("Processed %d pending rewards", processed)
}

// جدولة كل ساعة
func setupCronJobs() {
    c := cron.New()
    c.AddFunc("0 * * * *", processPendingRewardsCron)
    c.Start()
}
```

### مثال 4: عرض إحصائيات الإحالات

```go
func getReferralDashboard(tenantID uuid.UUID) (*ReferralDashboard, error) {
    ctx := context.Background()
    
    // 1. الحصول على الإحصائيات
    stats, err := referralManager.GetReferralStats(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    
    // 2. الحصول على آخر الإحالات
    referrals, err := referralManager.GetReferralsByReferrer(ctx, tenantID, 10, 0)
    if err != nil {
        return nil, err
    }
    
    return &ReferralDashboard{
        Stats:     stats,
        Referrals: referrals,
    }, nil
}
```

## التكوين (Configuration)

```go
config := &referral.ReferralConfig{
    // إعدادات المكافأة الافتراضية
    DefaultRewardType:      referral.RewardTypeBalance,
    DefaultRewardAmount:    10.0,
    DefaultRewardCondition: referral.RewardConditionFirstPayment,
    
    // إعدادات توليد الكود
    CodePrefix: "REF",
    CodeLength: 8,
    
    // الحدود
    MaxReferralsPerUser: 100,
    
    // التفعيل
    Enabled: true,
}

manager := referral.NewManager(db, config)
```

## التكامل

### مع نظام الاشتراكات

```go
// في subscription manager
func (sm *SubscriptionManager) Subscribe(ctx context.Context, req *SubscribeRequest) error {
    // ... منطق الاشتراك
    
    // التحقق من الإحالة
    err := referralManager.MarkConditionMet(ctx, req.TenantID)
    if err != nil {
        log.Printf("Failed to mark referral condition: %v", err)
    }
    
    return nil
}
```

### مع نظام الدفع

```go
// في payment manager
func (pm *PaymentManager) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    // ... منطق الدفع
    
    // التحقق من الإحالة
    err := referralManager.MarkConditionMet(ctx, req.TenantID)
    if err != nil {
        log.Printf("Failed to mark referral condition: %v", err)
    }
    
    return nil
}
```

### مع نظام الإشعارات

```go
// عند منح المكافأة
func (m *Manager) GrantReward(ctx context.Context, req *GrantRewardRequest) error {
    // ... منطق منح المكافأة
    
    // إرسال إشعار
    notification := &Notification{
        TenantID: referral.ReferrerTenantID,
        Type:     "referral_reward",
        Title:    "مكافأة إحالة جديدة!",
        Message:  fmt.Sprintf("تم منحك %v من مكافأة الإحالة", reward),
    }
    
    notificationService.Send(ctx, notification)
    
    return nil
}
```

## الأمان

### 1. منع الاحتيال

```go
// لا يمكن للعميل إحالة نفسه
if req.ReferrerTenantID == req.RefereeTenantID {
    return ErrSelfReferral
}

// لا يمكن للعميل استخدام أكثر من كود إحالة
var existingCount int
err := m.db.GetContext(ctx, &existingCount, 
    "SELECT COUNT(*) FROM referrals WHERE referee_tenant_id = $1", 
    req.RefereeTenantID)
if existingCount > 0 {
    return ErrAlreadyReferred
}
```

### 2. Row Level Security

جميع الاستعلامات تحترم RLS Policies المحددة في قاعدة البيانات.

### 3. Audit Trail

جميع العمليات مسجلة في جدول `referrals` مع timestamps.

## الأداء

### Caching

يمكن تخزين إحصائيات المُحيلين في Redis:

```go
func (m *Manager) GetReferralStats(ctx context.Context, referrerID uuid.UUID) (*ReferralStats, error) {
    // محاولة الحصول من Cache
    cacheKey := fmt.Sprintf("referral:stats:%s", referrerID)
    cached, err := redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var stats ReferralStats
        json.Unmarshal([]byte(cached), &stats)
        return &stats, nil
    }
    
    // الحصول من قاعدة البيانات
    stats, err := m.getReferralStatsFromDB(ctx, referrerID)
    if err != nil {
        return nil, err
    }
    
    // حفظ في Cache
    data, _ := json.Marshal(stats)
    redis.Set(ctx, cacheKey, data, 5*time.Minute)
    
    return stats, nil
}
```

### Batch Processing

معالجة المكافآت المعلقة بشكل دفعي:

```go
func (m *Manager) ProcessPendingRewards(ctx context.Context) (int, error) {
    // معالجة 100 إحالة في كل مرة
    query := `
        SELECT id FROM referrals
        WHERE condition_met = true 
          AND reward_granted = false
        LIMIT 100
    `
    
    // ... معالجة
}
```

## الاختبار

### Unit Tests

```bash
go test -v ./internal/referral/...
```

### Integration Tests

```bash
go test -v -tags=integration ./internal/referral/...
```

### Load Tests

```bash
k6 run performance-tests/referral-load-test.js
```

## الصيانة

### تنظيف الإحالات القديمة

```sql
-- حذف الإحالات الملغاة الأقدم من سنة
DELETE FROM referrals
WHERE status = 'cancelled' 
  AND created_at < CURRENT_TIMESTAMP - INTERVAL '1 year';
```

### إعادة معالجة المكافآت الفاشلة

```go
func retryFailedRewards(ctx context.Context) error {
    // البحث عن الإحالات التي فشل منح مكافأتها
    query := `
        SELECT id FROM referrals
        WHERE condition_met = true
          AND reward_granted = false
          AND status = 'completed'
          AND condition_met_at < CURRENT_TIMESTAMP - INTERVAL '1 hour'
    `
    
    // ... إعادة المحاولة
}
```

## الأخطاء الشائعة

### 1. ErrSelfReferral
```go
// الحل: التحقق من أن المُحيل والمُحال مختلفان
if referrerID == refereeID {
    return ErrSelfReferral
}
```

### 2. ErrAlreadyReferred
```go
// الحل: التحقق من عدم استخدام العميل لكود إحالة سابقاً
var count int
db.Get(&count, "SELECT COUNT(*) FROM referrals WHERE referee_tenant_id = $1", tenantID)
if count > 0 {
    return ErrAlreadyReferred
}
```

### 3. ErrConditionNotMet
```go
// الحل: التأكد من استيفاء الشرط قبل منح المكافأة
if !referral.ConditionMet {
    return ErrConditionNotMet
}
```

## الدعم

للمزيد من المعلومات:
- راجع `migrations/README_REFERRALS.md`
- راجع المتطلب 33 في `requirements.md`
- راجع التصميم في `design.md`

## التاريخ

- **2026-02-27**: إنشاء النظام الأولي
- **Version**: 1.0
- **Status**: Ready for Testing
