# دليل الامتثال القانوني - المهمة 38

## نظرة عامة

دليل شامل للامتثال للمتطلبات القانونية وحماية البيانات.

## المهمة 38.1: Terms of Service

### إنشاء صفحة شروط الخدمة

```typescript
// apps/client-web/src/pages/TermsOfService.tsx
import React from 'react';

export default function TermsOfService() {
  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">شروط الخدمة</h1>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">1. القبول بالشروط</h2>
        <p>
          باستخدامك لخدمات منصة SMS Gateway، فإنك توافق على الالتزام بهذه الشروط.
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">2. الاشتراكات والدفع</h2>
        <p>
          - جميع الاشتراكات تُجدد تلقائياً ما لم يتم إلغاؤها
          - يتم خصم قيمة الاشتراك من رصيدك في تاريخ التجديد
          - في حالة عدم كفاية الرصيد، ستحصل على فترة سماح 3 أيام
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">3. الاستخدام المقبول</h2>
        <p>
          يُمنع استخدام الخدمة لـ:
          - إرسال رسائل غير مرغوب فيها (Spam)
          - المحتوى غير القانوني أو المسيء
          - انتهاك حقوق الآخرين
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">4. الإلغاء والاسترداد</h2>
        <p>
          - يمكنك إلغاء اشتراكك في أي وقت
          - لا يتم استرداد المبالغ المدفوعة
          - الرسائل المتبقية تبقى متاحة حتى نهاية فترة الاشتراك
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">5. المسؤولية</h2>
        <p>
          - نحن غير مسؤولين عن فشل تسليم الرسائل بسبب مشاكل في شبكة الاتصالات
          - نحن غير مسؤولين عن محتوى الرسائل المرسلة
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">6. التعديلات</h2>
        <p>
          نحتفظ بالحق في تعديل هذه الشروط في أي وقت. سيتم إشعارك بأي تغييرات جوهرية.
        </p>
      </section>
      
      <p className="text-sm text-gray-600 mt-8">
        آخر تحديث: {new Date().toLocaleDateString('ar')}
      </p>
    </div>
  );
}
```

## المهمة 38.2: Privacy Policy

### إنشاء صفحة سياسة الخصوصية

```typescript
// apps/client-web/src/pages/PrivacyPolicy.tsx
export default function PrivacyPolicy() {
  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">سياسة الخصوصية</h1>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">1. البيانات التي نجمعها</h2>
        <ul className="list-disc list-inside">
          <li>معلومات الحساب (الاسم، البريد الإلكتروني، رقم الهاتف)</li>
          <li>معلومات الدفع (تُخزن بشكل آمن عبر بوابات الدفع)</li>
          <li>سجل الرسائل (الأرقام، التواريخ، الحالة)</li>
          <li>معلومات الاستخدام (عدد الرسائل، الاشتراكات)</li>
        </ul>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">2. كيف نستخدم بياناتك</h2>
        <ul className="list-disc list-inside">
          <li>تقديم الخدمة وإرسال الرسائل</li>
          <li>معالجة الدفعات والفواتير</li>
          <li>تحسين الخدمة وتطويرها</li>
          <li>إرسال إشعارات مهمة</li>
        </ul>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">3. حماية البيانات</h2>
        <p>
          نستخدم تشفير AES-256 لحماية بياناتك. جميع الاتصالات تتم عبر HTTPS/TLS 1.3.
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">4. مشاركة البيانات</h2>
        <p>
          لا نشارك بياناتك مع أطراف ثالثة إلا:
          - لمعالجة الدفعات (Stripe, PayPal)
          - لإرسال الرسائل (شركات الاتصالات)
          - عند الطلب القانوني
        </p>
      </section>
      
      <section className="mb-6">
        <h2 className="text-2xl font-semibold mb-3">5. حقوقك</h2>
        <ul className="list-disc list-inside">
          <li>الوصول إلى بياناتك</li>
          <li>تصحيح البيانات غير الصحيحة</li>
          <li>حذف حسابك وبياناتك</li>
          <li>تصدير بياناتك</li>
        </ul>
      </section>
    </div>
  );
}
```

## المهمة 38.3: Consent Management

### تطبيق طلب الموافقة

```typescript
// apps/client-web/src/components/ConsentForm.tsx
import React, { useState } from 'react';

export default function ConsentForm({ onAccept }: { onAccept: () => void }) {
  const [consents, setConsents] = useState({
    terms: false,
    privacy: false,
    marketing: false,
  });

  const handleSubmit = async () => {
    if (!consents.terms || !consents.privacy) {
      alert('يجب الموافقة على الشروط وسياسة الخصوصية');
      return;
    }

    // حفظ الموافقة
    await fetch('/api/consent', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        terms_accepted: true,
        privacy_accepted: true,
        marketing_accepted: consents.marketing,
        accepted_at: new Date().toISOString(),
      }),
    });

    onAccept();
  };

  return (
    <div className="max-w-md mx-auto p-6 bg-white rounded-lg shadow">
      <h2 className="text-2xl font-bold mb-4">الموافقة على الشروط</h2>
      
      <div className="space-y-4">
        <label className="flex items-start">
          <input
            type="checkbox"
            checked={consents.terms}
            onChange={(e) => setConsents({ ...consents, terms: e.target.checked })}
            className="mt-1 mr-2"
          />
          <span>
            أوافق على <a href="/terms" className="text-blue-600">شروط الخدمة</a>
            <span className="text-red-500">*</span>
          </span>
        </label>
        
        <label className="flex items-start">
          <input
            type="checkbox"
            checked={consents.privacy}
            onChange={(e) => setConsents({ ...consents, privacy: e.target.checked })}
            className="mt-1 mr-2"
          />
          <span>
            أوافق على <a href="/privacy" className="text-blue-600">سياسة الخصوصية</a>
            <span className="text-red-500">*</span>
          </span>
        </label>
        
        <label className="flex items-start">
          <input
            type="checkbox"
            checked={consents.marketing}
            onChange={(e) => setConsents({ ...consents, marketing: e.target.checked })}
            className="mt-1 mr-2"
          />
          <span>
            أوافق على تلقي رسائل تسويقية (اختياري)
          </span>
        </label>
      </div>
      
      <button
        onClick={handleSubmit}
        disabled={!consents.terms || !consents.privacy}
        className="w-full mt-6 bg-blue-600 text-white py-2 rounded disabled:bg-gray-400"
      >
        متابعة
      </button>
    </div>
  );
}
```

## المهمة 38.4: Data Deletion

### تطبيق حذف الحساب

```go
// services/backend/internal/api/account_handler.go
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    
    // تأكيد من المستخدم
    var req struct {
        Confirmation string `json:"confirmation" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    if req.Confirmation != "DELETE MY ACCOUNT" {
        c.JSON(400, gin.H{"error": "Invalid confirmation"})
        return
    }
    
    // حذف البيانات الشخصية
    err := h.db.Transaction(func(tx *gorm.DB) error {
        // حذف الاشتراكات
        tx.Where("tenant_id = ?", tenantID).Delete(&models.Subscription{})
        
        // حذف الرسائل المرحّلة
        tx.Where("tenant_id = ?", tenantID).Delete(&models.RolloverMessage{})
        
        // حذف API Keys
        tx.Where("tenant_id = ?", tenantID).Delete(&models.GatewayAPIKey{})
        
        // إخفاء البيانات الشخصية (لا نحذف المعاملات المالية)
        tx.Model(&models.Tenant{}).Where("id = ?", tenantID).Updates(map[string]interface{}{
            "email":        "deleted@example.com",
            "phone":        "DELETED",
            "name":         "Deleted User",
            "deleted_at":   time.Now(),
            "is_active":    false,
        })
        
        return nil
    })
    
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to delete account"})
        return
    }
    
    c.JSON(200, gin.H{"message": "Account deleted successfully"})
}
```

## المهمة 38.5: Data Export

### تطبيق تصدير البيانات

```go
func (h *AccountHandler) ExportData(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    
    // جمع جميع البيانات
    data := make(map[string]interface{})
    
    // معلومات الحساب
    var tenant models.Tenant
    h.db.First(&tenant, "id = ?", tenantID)
    data["account"] = tenant
    
    // الاشتراكات
    var subscriptions []models.Subscription
    h.db.Where("tenant_id = ?", tenantID).Find(&subscriptions)
    data["subscriptions"] = subscriptions
    
    // المعاملات المالية
    var transactions []models.AccountingTransaction
    h.db.Where("tenant_id = ?", tenantID).Find(&transactions)
    data["transactions"] = transactions
    
    // تحويل إلى JSON
    jsonData, _ := json.MarshalIndent(data, "", "  ")
    
    // إرسال كملف
    c.Header("Content-Disposition", "attachment; filename=my_data.json")
    c.Data(200, "application/json", jsonData)
}
```

## المهمة 38.6: Fraud Prevention

### تطبيق كشف الاحتيال

```go
// services/backend/internal/fraud/detector.go
package fraud

type FraudDetector struct {
    db *gorm.DB
}

func (fd *FraudDetector) CheckSuspiciousActivity(tenantID uuid.UUID) (bool, string) {
    // 1. فحص عدد الاشتراكات في فترة قصيرة
    var subscriptionCount int64
    fd.db.Model(&models.Subscription{}).
        Where("tenant_id = ? AND created_at > ?", tenantID, time.Now().Add(-1*time.Hour)).
        Count(&subscriptionCount)
    
    if subscriptionCount > 10 {
        return true, "Too many subscriptions in short time"
    }
    
    // 2. فحص محاولات الدفع الفاشلة
    var failedPayments int64
    fd.db.Model(&models.PaymentTransaction{}).
        Where("tenant_id = ? AND status = 'failed' AND created_at > ?", 
            tenantID, time.Now().Add(-24*time.Hour)).
        Count(&failedPayments)
    
    if failedPayments > 5 {
        return true, "Too many failed payment attempts"
    }
    
    // 3. فحص أنماط الإرسال غير الطبيعية
    var messageCount int64
    fd.db.Model(&models.Message{}).
        Where("tenant_id = ? AND created_at > ?", tenantID, time.Now().Add(-1*time.Hour)).
        Count(&messageCount)
    
    if messageCount > 1000 {
        return true, "Unusual message sending pattern"
    }
    
    return false, ""
}
```

## المهمة 38.7: Audit Logging

### تسجيل العمليات الحساسة

```go
// services/backend/internal/audit/logger.go
func (al *AuditLogger) LogSensitiveOperation(ctx context.Context, operation string, details map[string]interface{}) {
    log := &models.AuditLog{
        TenantID:   getTenantID(ctx),
        UserID:     getUserID(ctx),
        Operation:  operation,
        Details:    details,
        IPAddress:  getIPAddress(ctx),
        UserAgent:  getUserAgent(ctx),
        Timestamp:  time.Now(),
    }
    
    al.db.Create(log)
}
```

## Checklist للتحقق

- [ ] شروط الخدمة وسياسة الخصوصية منشورة
- [ ] Consent Management مطبّق
- [ ] Data Deletion يعمل بشكل صحيح
- [ ] Data Export متاح
- [ ] Fraud Prevention نشط
- [ ] Audit Logging شامل

**Requirements Covered:** 36.1-36.11
