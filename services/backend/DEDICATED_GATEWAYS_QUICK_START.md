# دليل البدء السريع - جدول البوابات المخصصة

## نظرة عامة سريعة

جدول `dedicated_gateways` يدير البوابات المخصصة للشركات المؤسسية الكبيرة في نموذج **Gateway-as-a-Service**.

### نموذج العمل المالي (مهم جداً!)

```
┌─────────────────────────────────────────────────────────────┐
│  الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل │
│  المنصة تتقاضى فقط: رسوم الاشتراك + رسوم إيجار السيرفر      │
│  المنصة تراقب الرصيد وترسل تنبيهات فقط                      │
└─────────────────────────────────────────────────────────────┘
```

## التثبيت السريع

### 1. تشغيل Migration

```powershell
# Windows PowerShell
cd services/backend
.\run-dedicated-gateways-migration.ps1
```

```bash
# Linux/Mac
cd services/backend
psql -h localhost -U postgres -d sms_gateway -f migrations/020_dedicated_gateways_table.sql
```

### 2. تشغيل الاختبارات

```powershell
# Windows PowerShell
$env:PGPASSWORD="your_password"
psql -h localhost -U postgres -d sms_gateway -f test_dedicated_gateways_migration.sql
```

## الاستخدام السريع

### إنشاء بوابة مخصصة

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = 'your-tenant-uuid';

-- إنشاء بوابة جديدة
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    telecom_provider,
    telecom_api_endpoint,
    telecom_username,
    telecom_password_encrypted,
    server_type,
    max_throughput,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    balance_alert_threshold,
    contract_start_date,
    next_billing_date
) VALUES (
    'your-tenant-uuid',
    'بوابة شركتي',
    'MY_COMPANY_GW',
    'Yemen Mobile',
    'https://api.yemenmobile.ye/sms',
    'my_username',
    'ENCRYPTED_PASSWORD', -- يجب تشفيرها!
    'dedicated',
    1000, -- 1000 رسالة/ثانية
    'monthly',
    1000.00, -- $1000 شهرياً
    200.00,  -- $200 إيجار شهري
    5000.00, -- تنبيه عند انخفاض الرصيد عن $5000
    CURRENT_DATE,
    CURRENT_DATE + INTERVAL '1 month'
);
```

### الاستعلام عن البوابات

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = 'your-tenant-uuid';

-- عرض جميع البوابات
SELECT 
    gateway_code,
    gateway_name,
    status,
    subscription_fee,
    server_rental_fee,
    telecom_balance,
    next_billing_date
FROM dedicated_gateways
WHERE status = 'active';
```

### تحديث رصيد شركة الاتصالات

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = 'your-tenant-uuid';

-- تحديث الرصيد (بعد فحصه من API شركة الاتصالات)
UPDATE dedicated_gateways
SET 
    telecom_balance = 15000.00,
    last_balance_check = CURRENT_TIMESTAMP
WHERE gateway_code = 'MY_COMPANY_GW';
```

### البحث عن البوابات التي تحتاج فوترة

```sql
-- البوابات التي حان موعد فوترتها
SELECT 
    gateway_code,
    gateway_name,
    subscription_fee,
    server_rental_fee,
    next_billing_date
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND next_billing_date <= CURRENT_DATE;
```

### البحث عن البوابات التي تحتاج تنبيه رصيد

```sql
-- البوابات التي انخفض رصيدها
SELECT 
    gateway_code,
    gateway_name,
    telecom_balance,
    balance_alert_threshold
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND balance_check_enabled = true
    AND telecom_balance < balance_alert_threshold;
```

## الحقول المهمة

### رسوم الاشتراك والإيجار

- `subscription_fee`: رسوم الاشتراك (شهري أو سنوي)
- `server_rental_fee`: رسوم إيجار السيرفر (شهري دائماً)
- `setup_fee`: رسوم الإعداد (مرة واحدة)

**مثال على الفوترة:**

```
اشتراك شهري:
- subscription_fee: $1000/شهر
- server_rental_fee: $200/شهر
- الإجمالي الشهري: $1200

اشتراك سنوي:
- subscription_fee: $10000/سنة
- server_rental_fee: $200/شهر
- الشهر الأول: $10000 + $200 = $10200
- الأشهر التالية: $200 فقط
- بعد سنة: $10000 + $200 = $10200 (تجديد)
```

### مراقبة الرصيد

- `telecom_balance`: رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
- `balance_alert_threshold`: عتبة التنبيه
- `last_balance_check`: آخر فحص
- `balance_check_enabled`: تفعيل/تعطيل المراقبة

**ملاحظة:** المنصة تراقب الرصيد فقط ولا تعالج أي دفعات!

## الأمان

### تشفير بيانات الاعتماد

**يجب** تشفير هذه الحقول قبل حفظها:

- `telecom_password_encrypted`
- `telecom_api_key_encrypted`

```go
// مثال في Go
import (
    "crypto/aes"
    "crypto/cipher"
    "encoding/base64"
)

func EncryptPassword(password, key string) (string, error) {
    block, err := aes.NewCipher([]byte(key))
    if err != nil {
        return "", err
    }
    
    // تطبيق AES-256 encryption
    // ...
    
    return base64.StdEncoding.EncodeToString(encrypted), nil
}
```

### Row Level Security (RLS)

RLS مفعّل تلقائياً:

```sql
-- يجب تعيين tenant_id قبل أي استعلام
SET app.current_tenant = 'your-tenant-uuid';

-- الآن جميع الاستعلامات محمية تلقائياً
SELECT * FROM dedicated_gateways; -- ترى بوابات شركتك فقط
```

## العمليات الدورية

### 1. فحص الرصيد (كل ساعة)

```sql
-- البوابات التي تحتاج فحص رصيد
SELECT 
    id,
    gateway_code,
    telecom_api_endpoint,
    last_balance_check
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND balance_check_enabled = true
    AND (
        last_balance_check IS NULL 
        OR last_balance_check < CURRENT_TIMESTAMP - INTERVAL '1 hour'
    );
```

### 2. الفوترة (يومياً)

```sql
-- البوابات التي تحتاج فوترة
SELECT 
    id,
    tenant_id,
    gateway_code,
    subscription_fee,
    server_rental_fee,
    next_billing_date
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND next_billing_date <= CURRENT_DATE;
```

### 3. فحص الصحة (كل 5 دقائق)

```sql
-- البوابات التي تحتاج فحص صحة
SELECT 
    id,
    gateway_code,
    telecom_api_endpoint,
    last_health_check
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND (
        last_health_check IS NULL 
        OR last_health_check < CURRENT_TIMESTAMP - INTERVAL '5 minutes'
    );
```

## الأخطاء الشائعة

### 1. نسيان تعيين tenant_id

```sql
-- ❌ خطأ: لم يتم تعيين tenant_id
SELECT * FROM dedicated_gateways;
-- النتيجة: خطأ أو نتائج فارغة

-- ✅ صحيح
SET app.current_tenant = 'your-tenant-uuid';
SELECT * FROM dedicated_gateways;
```

### 2. حفظ كلمة المرور بدون تشفير

```sql
-- ❌ خطأ: كلمة مرور غير مشفرة
INSERT INTO dedicated_gateways (..., telecom_password_encrypted)
VALUES (..., 'plain_password');

-- ✅ صحيح: تشفير أولاً
-- في التطبيق: encrypted = EncryptPassword('plain_password', key)
INSERT INTO dedicated_gateways (..., telecom_password_encrypted)
VALUES (..., 'ENCRYPTED_STRING_HERE');
```

### 3. خلط رسوم الاشتراك مع تكلفة الرسائل

```sql
-- ❌ خطأ: محاولة خصم تكلفة الرسائل من المنصة
-- المنصة لا تعالج تكلفة الرسائل!

-- ✅ صحيح: خصم رسوم الاشتراك والإيجار فقط
INSERT INTO accounting_transactions (
    tenant_id,
    transaction_type,
    amount,
    description
) VALUES (
    'tenant-uuid',
    'GATEWAY_SUBSCRIPTION',
    subscription_fee + server_rental_fee,
    'رسوم الاشتراك والإيجار الشهري'
);
```

## الخطوات التالية

بعد إنشاء هذا الجدول:

1. ✅ **تم**: جدول `dedicated_gateways`
2. ⏳ **التالي**: جدول `resource_quotas` (المهمة 16.2)
3. ⏳ جدول `resource_usage` (المهمة 16.3)
4. ⏳ جدول `gateway_users` (المهمة 16.4)
5. ⏳ جدول `gateway_api_keys` (المهمة 16.5)
6. ⏳ جدول `gateway_statistics` (المهمة 16.6)

## الموارد

- **التوثيق الكامل**: `migrations/README_DEDICATED_GATEWAYS.md`
- **Migration**: `migrations/020_dedicated_gateways_table.sql`
- **الاختبارات**: `test_dedicated_gateways_migration.sql`
- **السكريبت**: `run-dedicated-gateways-migration.ps1`

## الدعم

إذا واجهت أي مشاكل:

1. راجع ملف `README_DEDICATED_GATEWAYS.md` للتوثيق الكامل
2. تحقق من أن جدول `tenants` موجود
3. تحقق من أن RLS مفعّل: `SELECT rowsecurity FROM pg_tables WHERE tablename = 'dedicated_gateways';`
4. تحقق من تعيين `app.current_tenant` قبل الاستعلامات

---

**ملاحظة مهمة:** تذكر دائماً أن الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل. المنصة تتقاضى فقط رسوم الاشتراك والإيجار!
