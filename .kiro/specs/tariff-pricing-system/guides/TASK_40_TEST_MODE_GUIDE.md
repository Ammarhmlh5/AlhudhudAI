# دليل Test Mode - المهمة 40

## نظرة عامة

دليل شامل لتطوير بيئة اختبار منفصلة للمطورين.

## المهمة 40.1: Test Mode Environment

### تطوير Test Mode Manager

```go
// services/backend/internal/testmode/manager.go
package testmode

import (
    "context"
    "fmt"
    "github.com/google/uuid"
)

type TestModeManager struct {
    enabled bool
}

func NewTestModeManager() *TestModeManager {
    return &TestModeManager{
        enabled: true,
    }
}

func (tmm *TestModeManager) IsTestMode(ctx context.Context) bool {
    // التحقق من API Key
    apiKey := ctx.Value("api_key").(string)
    return tmm.isTestAPIKey(apiKey)
}

func (tmm *TestModeManager) isTestAPIKey(apiKey string) bool {
    // مفاتيح الاختبار تبدأ بـ test_
    return len(apiKey) > 5 && apiKey[:5] == "test_"
}

func (tmm *TestModeManager) MockSendMessage(ctx context.Context, to, message string) (*MockMessageResult, error) {
    // محاكاة إرسال الرسالة
    
    // أرقام اختبار خاصة
    result := &MockMessageResult{
        MessageID: uuid.New().String(),
        To:        to,
        Message:   message,
        Status:    tmm.getTestStatus(to),
        Timestamp: time.Now(),
    }
    
    // تسجيل في سجل منفصل
    tmm.logTestMessage(result)
    
    return result, nil
}

func (tmm *TestModeManager) getTestStatus(phoneNumber string) string {
    // أرقام اختبار محددة مسبقاً
    switch phoneNumber {
    case "+967777000001":
        return "delivered"
    case "+967777000002":
        return "failed"
    case "+967777000003":
        return "pending"
    case "+967777000004":
        return "rejected"
    default:
        return "delivered" // افتراضياً
    }
}

func (tmm *TestModeManager) logTestMessage(result *MockMessageResult) {
    // تسجيل في جدول منفصل
    testLog := &TestMessageLog{
        MessageID: result.MessageID,
        To:        result.To,
        Message:   result.Message,
        Status:    result.Status,
        Timestamp: result.Timestamp,
    }
    
    // حفظ في قاعدة البيانات
    db.Create(testLog)
}
```

## المهمة 40.2: Test Phone Numbers

### أرقام الاختبار المحددة مسبقاً

```go
var TestPhoneNumbers = map[string]string{
    "+967777000001": "delivered",      // نجاح
    "+967777000002": "failed",         // فشل
    "+967777000003": "pending",        // معلق
    "+967777000004": "rejected",       // مرفوض
    "+967777000005": "invalid_number", // رقم غير صحيح
    "+967777000006": "blocked",        // محظور
    "+967777000007": "expired",        // منتهي
    "+967777000008": "delivered",      // نجاح مع تأخير
    "+967777000009": "partial",        // نجاح جزئي
    "+967777000010": "delivered",      // نجاح فوري
}

func (tmm *TestModeManager) GetTestPhoneNumbers() []TestPhoneNumber {
    numbers := []TestPhoneNumber{}
    
    for phone, status := range TestPhoneNumbers {
        numbers = append(numbers, TestPhoneNumber{
            PhoneNumber: phone,
            Status:      status,
            Description: tmm.getStatusDescription(status),
        })
    }
    
    return numbers
}

func (tmm *TestModeManager) getStatusDescription(status string) string {
    descriptions := map[string]string{
        "delivered":      "الرسالة تُسلّم بنجاح",
        "failed":         "فشل التسليم",
        "pending":        "الرسالة معلقة",
        "rejected":       "الرسالة مرفوضة",
        "invalid_number": "رقم غير صحيح",
        "blocked":        "الرقم محظور",
        "expired":        "الرسالة منتهية",
        "partial":        "تسليم جزئي",
    }
    
    return descriptions[status]
}
```

## المهمة 40.3: Test API Keys

### توليد API Keys للاختبار

```go
func (tmm *TestModeManager) GenerateTestAPIKey(tenantID uuid.UUID) (string, error) {
    // توليد مفتاح اختبار
    key := fmt.Sprintf("test_%s_%s", 
        tenantID.String()[:8],
        uuid.New().String()[:16])
    
    // حفظ في قاعدة البيانات
    apiKey := &APIKey{
        TenantID:  tenantID,
        Key:       key,
        IsTest:    true,
        IsActive:  true,
        CreatedAt: time.Now(),
    }
    
    if err := db.Create(apiKey).Error; err != nil {
        return "", err
    }
    
    return key, nil
}

func (tmm *TestModeManager) ValidateTestAPIKey(key string) (*APIKey, error) {
    var apiKey APIKey
    
    if err := db.Where("key = ? AND is_test = true AND is_active = true", key).
        First(&apiKey).Error; err != nil {
        return nil, fmt.Errorf("invalid test API key")
    }
    
    return &apiKey, nil
}
```

### واجهة إدارة مفاتيح الاختبار

```typescript
// apps/client-web/src/pages/TestMode.tsx
import React, { useState, useEffect } from 'react';

export default function TestMode() {
  const [testAPIKey, setTestAPIKey] = useState('');
  const [testNumbers, setTestNumbers] = useState([]);

  useEffect(() => {
    loadTestAPIKey();
    loadTestNumbers();
  }, []);

  const loadTestAPIKey = async () => {
    const response = await fetch('/api/test-mode/api-key');
    const data = await response.json();
    setTestAPIKey(data.key);
  };

  const loadTestNumbers = async () => {
    const response = await fetch('/api/test-mode/phone-numbers');
    const data = await response.json();
    setTestNumbers(data.numbers);
  };

  const generateNewKey = async () => {
    const response = await fetch('/api/test-mode/api-key', {
      method: 'POST',
    });
    const data = await response.json();
    setTestAPIKey(data.key);
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">وضع الاختبار</h1>
      
      <div className="bg-yellow-50 border border-yellow-200 rounded p-4 mb-6">
        <p className="text-yellow-800">
          ⚠️ وضع الاختبار: جميع الرسائل محاكاة ولن يتم إرسالها فعلياً
        </p>
      </div>
      
      <section className="mb-8">
        <h2 className="text-2xl font-semibold mb-4">مفتاح API للاختبار</h2>
        <div className="bg-gray-100 p-4 rounded">
          <code className="text-sm">{testAPIKey}</code>
        </div>
        <button
          onClick={generateNewKey}
          className="mt-2 bg-blue-600 text-white px-4 py-2 rounded"
        >
          توليد مفتاح جديد
        </button>
      </section>
      
      <section>
        <h2 className="text-2xl font-semibold mb-4">أرقام الاختبار</h2>
        <table className="w-full border">
          <thead>
            <tr className="bg-gray-100">
              <th className="border p-2">رقم الهاتف</th>
              <th className="border p-2">الحالة</th>
              <th className="border p-2">الوصف</th>
            </tr>
          </thead>
          <tbody>
            {testNumbers.map((number) => (
              <tr key={number.phone_number}>
                <td className="border p-2 font-mono">{number.phone_number}</td>
                <td className="border p-2">
                  <span className={`px-2 py-1 rounded text-sm ${
                    number.status === 'delivered' ? 'bg-green-100 text-green-800' :
                    number.status === 'failed' ? 'bg-red-100 text-red-800' :
                    'bg-yellow-100 text-yellow-800'
                  }`}>
                    {number.status}
                  </span>
                </td>
                <td className="border p-2">{number.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
      
      <section className="mt-8">
        <h2 className="text-2xl font-semibold mb-4">مثال على الاستخدام</h2>
        <pre className="bg-gray-900 text-white p-4 rounded overflow-x-auto">
{`curl -X POST https://api.example.com/v1/messages \\
  -H "Authorization: Bearer ${testAPIKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "+967777000001",
    "message": "Test message"
  }'`}
        </pre>
      </section>
    </div>
  );
}
```

## المهمة 40.4: Test Logging

### تسجيل منفصل لعمليات الاختبار

```sql
-- جدول سجل الاختبار
CREATE TABLE test_message_logs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    message_id VARCHAR(100) NOT NULL,
    to_number VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    api_key VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_test_logs_tenant ON test_message_logs(tenant_id);
CREATE INDEX idx_test_logs_created ON test_message_logs(created_at DESC);
```

### عرض سجل الاختبار

```go
func (tmm *TestModeManager) GetTestLogs(tenantID uuid.UUID, limit int) ([]TestMessageLog, error) {
    var logs []TestMessageLog
    
    err := db.Where("tenant_id = ?", tenantID).
        Order("created_at DESC").
        Limit(limit).
        Find(&logs).Error
    
    return logs, err
}

func (tmm *TestModeManager) ClearTestLogs(tenantID uuid.UUID) error {
    return db.Where("tenant_id = ?", tenantID).
        Delete(&TestMessageLog{}).Error
}
```

### API لعرض السجل

```go
func (h *TestModeHandler) GetLogs(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    limit := c.DefaultQuery("limit", "100")
    
    logs, err := h.testMode.GetTestLogs(uuid.MustParse(tenantID), atoi(limit))
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to get logs"})
        return
    }
    
    c.JSON(200, gin.H{
        "success": true,
        "data": gin.H{
            "logs": logs,
            "count": len(logs),
        },
    })
}
```

## التمييز الواضح بين الوضعين

### مؤشر بصري في الواجهة

```typescript
// مكون لعرض مؤشر Test Mode
export function TestModeIndicator() {
  const isTestMode = useTestMode();
  
  if (!isTestMode) return null;
  
  return (
    <div className="fixed top-0 left-0 right-0 bg-yellow-500 text-black text-center py-2 z-50">
      <strong>⚠️ وضع الاختبار نشط</strong> - جميع العمليات محاكاة
    </div>
  );
}
```

### التحقق من الوضع في كل طلب

```go
func TestModeMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("Authorization")
        
        if strings.HasPrefix(apiKey, "Bearer test_") {
            c.Set("test_mode", true)
            c.Header("X-Test-Mode", "true")
        }
        
        c.Next()
    }
}
```

## أفضل الممارسات

### 1. عدم التأثير على الإحصائيات الحقيقية
- سجل منفصل للاختبار
- لا تُحسب في التقارير
- لا تؤثر على الفوترة

### 2. التمييز الواضح
- مفاتيح API تبدأ بـ test_
- مؤشر بصري في الواجهة
- رسائل تحذير واضحة

### 3. البيانات الواقعية
- محاكاة سلوك حقيقي
- أرقام اختبار متنوعة
- حالات مختلفة

## Checklist للتحقق

- [ ] Test Mode Environment يعمل
- [ ] أرقام الاختبار محددة
- [ ] Test API Keys تُولّد بشكل صحيح
- [ ] التسجيل منفصل
- [ ] التمييز واضح بين الوضعين
- [ ] لا تأثير على الإحصائيات الحقيقية

**Requirements Covered:** 38.1-38.10
