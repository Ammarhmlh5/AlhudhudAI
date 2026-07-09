# دليل الاختبارات الشاملة - المهمة 34

## نظرة عامة

هذا الدليل يغطي تنفيذ جميع أنواع الاختبارات الشاملة لنظام التعرفات والتسعير:
- Integration Tests للنظام الكامل
- Property-Based Tests الشاملة
- Load Tests
- Stress Tests
- Security Tests
- Concurrent Tests للعزل

## المتطلبات

### البرمجيات المطلوبة
- Go 1.21+
- PostgreSQL 15+ (بيئة اختبار)
- Redis 7+ (بيئة اختبار)
- k6 (Load Testing)
- Docker و Docker Compose

### المكتبات المطلوبة
```bash
# Property-Based Testing
go get pgregory.net/rapid

# Testing utilities
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require
go get github.com/stretchr/testify/suite

# Database testing
go get github.com/DATA-DOG/go-sqlmock
```

## المهمة 34.1: Integration Tests للنظام الكامل

### الهدف
اختبار التدفقات الكاملة للنظام من البداية للنهاية.

### خطوات التنفيذ

#### 1. إنشاء ملف الاختبار
```bash
mkdir -p services/backend/internal/integration
touch services/backend/internal/integration/tariff_system_test.go
```

#### 2. إعداد بيئة الاختبار

```go
// services/backend/internal/integration/tariff_system_test.go
package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/google/uuid"
    "github.com/stretchr/testify/suite"
    "gorm.io/gorm"
)

type TariffSystemTestSuite struct {
    suite.Suite
    db     *gorm.DB
    redis  *redis.Client
    ctx    context.Context
    cancel context.CancelFunc
}

func (suite *TariffSystemTestSuite) SetupSuite() {
    // إعداد قاعدة البيانات
    suite.db = setupTestDatabase()
    suite.redis = setupTestRedis()
    suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 5*time.Minute)
}

func (suite *TariffSystemTestSuite) TearDownSuite() {
    suite.cancel()
    cleanupTestDatabase(suite.db)
    cleanupTestRedis(suite.redis)
}

func TestTariffSystemSuite(t *testing.T) {
    suite.Run(t, new(TariffSystemTestSuite))
}
```

#### 3. اختبار التدفق الكامل: الاشتراك → الإرسال → الخصم → التجديد

```go
func (suite *TariffSystemTestSuite) TestCompleteSubscriptionFlow() {
    // 1. إنشاء tenant
    tenantID := uuid.New()
    suite.createTestTenant(tenantID, 100.0) // رصيد 100 دولار
    
    // 2. إنشاء باقة
    packageID := suite.createTestPackage(tenantID, "Standard", 20.0, 3000)
    
    // 3. الاشتراك في الباقة
    subscriptionID := suite.subscribeToPackage(tenantID, packageID)
    suite.NotNil(subscriptionID)
    
    // 4. التحقق من خصم قيمة الباقة
    balance := suite.getTenantBalance(tenantID)
    suite.Equal(80.0, balance) // 100 - 20
    
    // 5. إرسال رسائل
    for i := 0; i < 100; i++ {
        err := suite.sendMessage(tenantID, subscriptionID, "Test message")
        suite.NoError(err)
    }
    
    // 6. التحقق من خصم من عداد الرسائل
    subscription := suite.getSubscription(subscriptionID)
    suite.Equal(100, subscription.MessagesUsed)
    suite.Equal(2900, subscription.MessagesRemaining)
    
    // 7. محاكاة نهاية الشهر
    suite.simulateEndOfMonth(subscriptionID)
    
    // 8. التحقق من التجديد التلقائي
    subscription = suite.getSubscription(subscriptionID)
    suite.Equal("active", subscription.Status)
    suite.Equal(0, subscription.MessagesUsed)
    suite.Equal(3000, subscription.MessagesRemaining)
    
    // 9. التحقق من خصم قيمة التجديد
    balance = suite.getTenantBalance(tenantID)
    suite.Equal(60.0, balance) // 80 - 20
}
```

#### 4. اختبار تدفق Overage

```go
func (suite *TariffSystemTestSuite) TestOverageFlow() {
    tenantID := uuid.New()
    suite.createTestTenant(tenantID, 100.0)
    
    packageID := suite.createTestPackage(tenantID, "Basic", 10.0, 1000)
    subscriptionID := suite.subscribeToPackage(tenantID, packageID)
    
    // إرسال رسائل حتى نفاد الرسائل المضمنة
    for i := 0; i < 1000; i++ {
        err := suite.sendMessage(tenantID, subscriptionID, "Test")
        suite.NoError(err)
    }
    
    subscription := suite.getSubscription(subscriptionID)
    suite.Equal(0, subscription.MessagesRemaining)
    
    // إرسال رسائل إضافية (Overage)
    for i := 0; i < 100; i++ {
        err := suite.sendMessage(tenantID, subscriptionID, "Overage test")
        suite.NoError(err)
    }
    
    subscription = suite.getSubscription(subscriptionID)
    suite.Equal(100, subscription.OverageMessages)
    
    // التحقق من خصم تكلفة Overage من الرصيد
    // سعر الرسالة = 10 / 1000 = 0.01 دولار
    expectedCost := 100 * 0.01 // 1 دولار
    suite.Equal(expectedCost, subscription.OverageCost)
    
    balance := suite.getTenantBalance(tenantID)
    suite.Equal(89.0, balance) // 100 - 10 (باقة) - 1 (overage)
}
```

#### 5. اختبار تدفق Rollover

```go
func (suite *TariffSystemTestSuite) TestRolloverFlow() {
    tenantID := uuid.New()
    suite.createTestTenant(tenantID, 100.0)
    
    // إنشاء باقة مع Rollover مفعّل
    packageID := suite.createTestPackage(tenantID, "Professional", 40.0, 8000)
    suite.enableRollover(packageID, 2000, 2) // حد أقصى 2000، صلاحية شهرين
    
    subscriptionID := suite.subscribeToPackage(tenantID, packageID)
    
    // استخدام 6000 رسالة فقط
    for i := 0; i < 6000; i++ {
        suite.sendMessage(tenantID, subscriptionID, "Test")
    }
    
    subscription := suite.getSubscription(subscriptionID)
    suite.Equal(2000, subscription.MessagesRemaining)
    
    // محاكاة نهاية الشهر
    suite.simulateEndOfMonth(subscriptionID)
    
    // التحقق من ترحيل الرسائل
    rollover := suite.getRolloverMessages(subscriptionID)
    suite.NotNil(rollover)
    suite.Equal(2000, rollover.MessagesCount)
    suite.Equal("active", rollover.Status)
    
    // التحقق من إضافة الرسائل المرحّلة للشهر الجديد
    subscription = suite.getSubscription(subscriptionID)
    suite.Equal(10000, subscription.MessagesRemaining) // 8000 + 2000
}
```

### Checklist للتحقق

- [ ] جميع Integration Tests تنجح
- [ ] التدفق الكامل يعمل بدون أخطاء
- [ ] Overage يعمل بشكل صحيح
- [ ] Rollover يعمل بشكل صحيح
- [ ] التجديد التلقائي يعمل بشكل موثوق
- [ ] جميع العمليات المالية دقيقة

## المهمة 34.2: Property-Based Tests الشاملة

### الهدف
اختبار الخصائص العامة للنظام باستخدام Property-Based Testing.

### خطوات التنفيذ

#### 1. إنشاء ملف Property Tests

```go
// services/backend/internal/integration/tariff_properties_test.go
package integration

import (
    "testing"
    "pgregory.net/rapid"
)

// Property 1: الرصيد لا يمكن أن يكون سالباً
func TestProperty_BalanceNeverNegative(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        tenantID := uuid.New()
        initialBalance := rapid.Float64Range(0, 10000).Draw(t, "initial_balance")
        
        suite.createTestTenant(tenantID, initialBalance)
        
        // محاولة خصم مبلغ أكبر من الرصيد
        deductAmount := rapid.Float64Range(initialBalance+1, initialBalance+1000).Draw(t, "deduct_amount")
        
        err := suite.deductBalance(tenantID, deductAmount)
        
        // يجب أن تفشل العملية
        if err == nil {
            t.Fatalf("Expected error when deducting more than balance")
        }
        
        // التحقق من أن الرصيد لم يتغير
        balance := suite.getTenantBalance(tenantID)
        if balance != initialBalance {
            t.Fatalf("Balance changed after failed deduction: expected %.2f, got %.2f", 
                initialBalance, balance)
        }
    })
}

// Property 2: مجموع الرسائل المستخدمة والمتبقية = الرسائل المضمنة
func TestProperty_MessageCountConsistency(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        tenantID := uuid.New()
        suite.createTestTenant(tenantID, 1000.0)
        
        messagesIncluded := rapid.IntRange(100, 10000).Draw(t, "messages_included")
        packageID := suite.createTestPackage(tenantID, "Test", 10.0, messagesIncluded)
        subscriptionID := suite.subscribeToPackage(tenantID, packageID)
        
        messagesToSend := rapid.IntRange(0, messagesIncluded).Draw(t, "messages_to_send")
        
        for i := 0; i < messagesToSend; i++ {
            suite.sendMessage(tenantID, subscriptionID, "Test")
        }
        
        subscription := suite.getSubscription(subscriptionID)
        
        total := subscription.MessagesUsed + subscription.MessagesRemaining
        if total != messagesIncluded {
            t.Fatalf("Message count inconsistency: used=%d, remaining=%d, included=%d",
                subscription.MessagesUsed, subscription.MessagesRemaining, messagesIncluded)
        }
    })
}

// Property 3: Rollover لا يتجاوز الحد الأقصى
func TestProperty_RolloverRespectLimit(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        tenantID := uuid.New()
        suite.createTestTenant(tenantID, 1000.0)
        
        messagesIncluded := rapid.IntRange(1000, 10000).Draw(t, "messages_included")
        rolloverLimit := rapid.IntRange(100, messagesIncluded/2).Draw(t, "rollover_limit")
        
        packageID := suite.createTestPackage(tenantID, "Test", 10.0, messagesIncluded)
        suite.enableRollover(packageID, rolloverLimit, 2)
        
        subscriptionID := suite.subscribeToPackage(tenantID, packageID)
        
        // استخدام عدد عشوائي من الرسائل
        messagesToUse := rapid.IntRange(0, messagesIncluded-1).Draw(t, "messages_to_use")
        for i := 0; i < messagesToUse; i++ {
            suite.sendMessage(tenantID, subscriptionID, "Test")
        }
        
        suite.simulateEndOfMonth(subscriptionID)
        
        rollover := suite.getRolloverMessages(subscriptionID)
        if rollover != nil && rollover.MessagesCount > rolloverLimit {
            t.Fatalf("Rollover exceeded limit: got %d, limit %d", 
                rollover.MessagesCount, rolloverLimit)
        }
    })
}
```

### Checklist للتحقق

- [ ] Property 1-20 جميعها تنجح
- [ ] لا توجد حالات فشل في Property Tests
- [ ] جميع الخصائص العامة محترمة
- [ ] النظام يتصرف بشكل متسق

## المهمة 34.3: Load Tests

### الهدف
اختبار النظام تحت حمل عالي.

### خطوات التنفيذ

#### 1. إنشاء ملف k6 للاختبار

```javascript
// performance-tests/k6-tests/tariff-load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
    stages: [
        { duration: '2m', target: 100 },  // Ramp up
        { duration: '5m', target: 100 },  // Stay at 100
        { duration: '2m', target: 500 },  // Ramp to 500
        { duration: '5m', target: 500 },  // Stay at 500
        { duration: '2m', target: 1000 }, // Ramp to 1000
        { duration: '5m', target: 1000 }, // Stay at 1000
        { duration: '2m', target: 0 },    // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests < 500ms
        errors: ['rate<0.01'],            // Error rate < 1%
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
    // Test 1: Subscribe to package
    const subscribeRes = http.post(`${BASE_URL}/api/subscriptions`, JSON.stringify({
        package_id: 1,
    }), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${__ENV.API_KEY}`,
        },
    });
    
    check(subscribeRes, {
        'subscribe status is 200': (r) => r.status === 200,
    }) || errorRate.add(1);
    
    sleep(1);
    
    // Test 2: Send message
    const sendRes = http.post(`${BASE_URL}/api/messages`, JSON.stringify({
        to: '+967777777777',
        message: 'Load test message',
    }), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${__ENV.API_KEY}`,
        },
    });
    
    check(sendRes, {
        'send status is 200': (r) => r.status === 200,
    }) || errorRate.add(1);
    
    sleep(1);
    
    // Test 3: Get usage
    const usageRes = http.get(`${BASE_URL}/api/subscriptions/usage`, {
        headers: {
            'Authorization': `Bearer ${__ENV.API_KEY}`,
        },
    });
    
    check(usageRes, {
        'usage status is 200': (r) => r.status === 200,
    }) || errorRate.add(1);
    
    sleep(1);
}
```

#### 2. تشغيل Load Test

```bash
# إنشاء سكريبت PowerShell
# performance-tests/run-tariff-load-test.ps1

# تشغيل Load Test
k6 run --vus 1000 --duration 30m `
    --env BASE_URL=http://localhost:8080 `
    --env API_KEY=test_key `
    k6-tests/tariff-load-test.js

# حفظ النتائج
k6 run --vus 1000 --duration 30m `
    --out json=results/tariff-load-test-results.json `
    k6-tests/tariff-load-test.js
```

### Checklist للتحقق

- [ ] النظام يتحمل 1000 مستخدم متزامن
- [ ] 95% من الطلبات < 500ms
- [ ] معدل الخطأ < 1%
- [ ] لا توجد تسريبات ذاكرة
- [ ] CPU و Memory ضمن الحدود المقبولة

## المهمة 34.4: Stress Tests

### الهدف
اختبار النظام تحت ضغط شديد لمعرفة نقطة الانهيار.

### خطوات التنفيذ

```javascript
// performance-tests/k6-tests/tariff-stress-test.js
export const options = {
    stages: [
        { duration: '2m', target: 1000 },
        { duration: '5m', target: 1000 },
        { duration: '2m', target: 2000 },
        { duration: '5m', target: 2000 },
        { duration: '2m', target: 5000 },
        { duration: '5m', target: 5000 },
        { duration: '2m', target: 10000 },
        { duration: '5m', target: 10000 },
        { duration: '10m', target: 0 },
    ],
};

// نفس الكود السابق
```

### Checklist للتحقق

- [ ] تحديد نقطة الانهيار
- [ ] توثيق الحد الأقصى للمستخدمين
- [ ] تحديد الاختناقات (Bottlenecks)
- [ ] خطة لتحسين الأداء

## المهمة 34.5: Security Tests

### الهدف
اختبار أمان النظام ومنع الثغرات.

### خطوات التنفيذ

#### 1. اختبار Cross-Tenant Access Prevention

```go
func TestSecurity_CrossTenantAccessPrevention(t *testing.T) {
    tenant1 := uuid.New()
    tenant2 := uuid.New()
    
    suite.createTestTenant(tenant1, 100.0)
    suite.createTestTenant(tenant2, 100.0)
    
    // إنشاء باقة لـ tenant1
    package1 := suite.createTestPackage(tenant1, "Test", 10.0, 1000)
    
    // محاولة الوصول من tenant2
    err := suite.subscribeToPackageAs(tenant2, package1)
    
    // يجب أن تفشل
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cross-tenant access denied")
}
```

#### 2. اختبار SQL Injection

```go
func TestSecurity_SQLInjection(t *testing.T) {
    maliciousInput := "'; DROP TABLE subscriptions; --"
    
    err := suite.createTestPackage(uuid.New(), maliciousInput, 10.0, 1000)
    
    // يجب أن تفشل أو تتعامل بشكل آمن
    assert.NoError(t, err)
    
    // التحقق من أن الجدول لا يزال موجوداً
    var count int64
    suite.db.Table("subscriptions").Count(&count)
    // لا يجب أن يكون الجدول محذوفاً
}
```

### Checklist للتحقق

- [ ] Cross-Tenant Access محظور تماماً
- [ ] SQL Injection محمي
- [ ] XSS محمي
- [ ] CSRF محمي
- [ ] API Authentication يعمل بشكل صحيح

## المهمة 34.6: Concurrent Tests للعزل

### الهدف
اختبار العزل بين Tenants في عمليات متزامنة.

### خطوات التنفيذ

```go
func TestConcurrent_TenantIsolation(t *testing.T) {
    numTenants := 100
    numOperationsPerTenant := 1000
    
    var wg sync.WaitGroup
    errors := make(chan error, numTenants*numOperationsPerTenant)
    
    for i := 0; i < numTenants; i++ {
        tenantID := uuid.New()
        suite.createTestTenant(tenantID, 1000.0)
        
        wg.Add(1)
        go func(tid uuid.UUID) {
            defer wg.Done()
            
            for j := 0; j < numOperationsPerTenant; j++ {
                err := suite.deductBalance(tid, 0.01)
                if err != nil {
                    errors <- err
                }
            }
        }(tenantID)
    }
    
    wg.Wait()
    close(errors)
    
    // التحقق من عدم وجود أخطاء
    errorCount := 0
    for err := range errors {
        t.Logf("Error: %v", err)
        errorCount++
    }
    
    assert.Equal(t, 0, errorCount, "Should have no errors in concurrent operations")
}
```

### Checklist للتحقق

- [ ] لا توجد Race Conditions
- [ ] العزل يعمل في العمليات المتزامنة
- [ ] لا يوجد تأثير بين Tenants
- [ ] جميع العمليات دقيقة

## أفضل الممارسات

### 1. إعداد بيئة الاختبار
- استخدم قاعدة بيانات منفصلة للاختبار
- نظف البيانات بعد كل اختبار
- استخدم Transactions للعزل

### 2. كتابة الاختبارات
- اختبارات واضحة ومفهومة
- رسائل خطأ مفيدة
- تغطية جميع الحالات الحرجة

### 3. تشغيل الاختبارات
- شغّل الاختبارات بانتظام
- استخدم CI/CD للتشغيل التلقائي
- راقب النتائج وحلل الفشل

### 4. توثيق النتائج
- احتفظ بسجل النتائج
- وثّق المشاكل المكتشفة
- تتبع التحسينات

## الخلاصة

هذا الدليل يغطي جميع أنواع الاختبارات المطلوبة للمهمة 34. اتبع الخطوات بعناية وتأكد من نجاح جميع الاختبارات قبل الانتقال للمهمة التالية.

**Requirements Covered:** جميع المتطلبات (1-42)
