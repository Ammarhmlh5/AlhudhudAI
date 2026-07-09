# جدول البوابات المخصصة (Dedicated Gateways)

## نظرة عامة

جدول `dedicated_gateways` يدير البوابات المخصصة للشركات المؤسسية الكبيرة. هذا الجدول يدعم نموذج **Gateway-as-a-Service** حيث تحصل كل شركة على بوابة مخصصة للإرسال مع عزل تام عن الشركات الأخرى.

## نموذج العمل المالي المهم

### الشركة تدفع لشركة الاتصالات مباشرة

**ملاحظة حرجة:** في هذا النموذج، الشركة المؤسسية تدفع **مباشرة** لشركة الاتصالات لتكلفة الرسائل. المنصة **لا تعالج** أي دفعات متعلقة بتكلفة الرسائل.

### المنصة تتقاضى فقط

المنصة تتقاضى من الشركة المؤسسية:

1. **رسوم الاشتراك** (`subscription_fee`): رسوم شهرية أو سنوية لاستخدام المنصة
2. **رسوم إيجار السيرفر** (`server_rental_fee`): رسوم شهرية لإيجار البنية التحتية
3. **رسوم الإعداد** (`setup_fee`): رسوم مرة واحدة عند إنشاء البوابة

### دور المنصة في الرصيد

المنصة تراقب رصيد الشركة لدى شركة الاتصالات (`telecom_balance`) وترسل تنبيهات عند انخفاضه، لكنها **لا تخصم** أي مبالغ متعلقة بتكلفة الرسائل.

## الحقول الرئيسية

### معلومات البوابة

- `gateway_name`: اسم البوابة (مثل: "بوابة شركة الاتصالات الحديثة")
- `gateway_code`: رمز فريد للبوابة (مثل: "MODERN_TELECOM_GW")
- `description`: وصف البوابة

### معلومات الاتصال بشركة الاتصالات

- `telecom_provider`: اسم شركة الاتصالات (مثل: "Yemen Mobile")
- `telecom_api_endpoint`: عنوان API لشركة الاتصالات
- `telecom_username`: اسم المستخدم للاتصال
- `telecom_password_encrypted`: كلمة المرور **مشفرة** بـ AES-256
- `telecom_api_key_encrypted`: مفتاح API **مشفر** بـ AES-256

**تحذير أمني:** يجب تشفير بيانات الاعتماد قبل حفظها في قاعدة البيانات.

### معلومات السيرفر

- `server_type`: نوع السيرفر
  - `dedicated`: سيرفر مخصص كامل
  - `vps`: سيرفر افتراضي خاص
  - `container`: حاوية Docker/Kubernetes
  - `shared`: بنية تحتية مشتركة مع عزل منطقي

- `server_specs`: مواصفات السيرفر بصيغة JSON
  ```json
  {
    "cpu": 4,
    "ram": "8GB",
    "storage": "100GB",
    "bandwidth": "1Gbps"
  }
  ```

### حدود الاستخدام

- `max_throughput`: الحد الأقصى للإنتاجية (رسائل/ثانية)
- `daily_limit`: الحد اليومي للرسائل
- `monthly_limit`: الحد الشهري للرسائل

### نموذج الاشتراك

- `subscription_type`: نوع الاشتراك
  - `monthly`: اشتراك شهري
  - `yearly`: اشتراك سنوي

- `subscription_fee`: رسوم الاشتراك (شهري أو سنوي حسب النوع)
- `server_rental_fee`: رسوم إيجار السيرفر الشهرية
- `setup_fee`: رسوم الإعداد الأولية (تُدفع مرة واحدة)
- `billing_cycle`: دورة الفوترة (`monthly` أو `yearly`)
- `next_billing_date`: تاريخ الفوترة القادم

### مراقبة الرصيد

**ملاحظة:** هذه الحقول للمراقبة فقط، المنصة لا تعالج دفعات الرسائل.

- `telecom_balance`: رصيد الشركة لدى شركة الاتصالات (للقراءة فقط)
- `balance_alert_threshold`: عتبة التنبيه (يتم إرسال تنبيه عند الانخفاض عن هذا الحد)
- `last_balance_check`: آخر فحص للرصيد
- `balance_check_enabled`: تفعيل/تعطيل مراقبة الرصيد

### معلومات العقد

- `contract_start_date`: تاريخ بداية العقد
- `contract_end_date`: تاريخ نهاية العقد (NULL = مفتوح)
- `auto_renew`: التجديد التلقائي للعقد

### الحالة والصحة

- `status`: حالة البوابة
  - `active`: نشطة وتعمل
  - `suspended`: معلقة (مؤقتاً)
  - `maintenance`: تحت الصيانة
  - `terminated`: منتهية

- `health_status`: حالة صحة البوابة
  - `healthy`: صحية وتعمل بشكل طبيعي
  - `degraded`: أداء منخفض
  - `unhealthy`: غير صحية
  - `unknown`: غير معروفة

- `last_health_check`: آخر فحص صحة

## Row Level Security (RLS)

الجدول محمي بـ **Row Level Security** لضمان العزل التام بين الشركات:

- كل شركة (`tenant`) ترى بواباتها فقط
- لا يمكن لشركة الوصول لبيانات بوابات شركة أخرى
- جميع العمليات (SELECT، INSERT، UPDATE، DELETE) محمية بـ RLS

### كيفية استخدام RLS

قبل أي استعلام، يجب تعيين `tenant_id` في الجلسة:

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- الآن جميع الاستعلامات ستُطبق RLS تلقائياً
SELECT * FROM dedicated_gateways; -- سيعرض بوابات هذه الشركة فقط
```

## Indexes

الجدول يحتوي على indexes محسّنة للأداء:

1. **idx_dedicated_gateways_tenant**: للعزل بين الشركات
2. **idx_dedicated_gateways_code**: للبحث برمز البوابة
3. **idx_dedicated_gateways_status**: لفلترة البوابات حسب الحالة
4. **idx_dedicated_gateways_billing**: للفحص اليومي للفوترة
5. **idx_dedicated_gateways_balance_check**: لمراقبة الرصيد الدورية

## أمثلة الاستخدام

### إنشاء بوابة مخصصة جديدة

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- إنشاء بوابة جديدة
INSERT INTO dedicated_gateways (
    tenant_id,
    gateway_name,
    gateway_code,
    description,
    telecom_provider,
    telecom_api_endpoint,
    telecom_username,
    telecom_password_encrypted,
    server_type,
    server_specs,
    max_throughput,
    daily_limit,
    monthly_limit,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    setup_fee,
    balance_alert_threshold,
    contract_start_date,
    next_billing_date
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    'بوابة شركة الاتصالات الحديثة',
    'MODERN_TELECOM_GW',
    'بوابة مخصصة للإرسال الجماعي',
    'Yemen Mobile',
    'https://api.yemenmobile.ye/sms',
    'enterprise_user',
    'ENCRYPTED_PASSWORD_HERE', -- يجب تشفيرها قبل الإدراج
    'dedicated',
    '{"cpu": 8, "ram": "16GB", "storage": "500GB", "bandwidth": "10Gbps"}',
    1000, -- 1000 رسالة/ثانية
    100000, -- 100,000 رسالة/يوم
    3000000, -- 3 مليون رسالة/شهر
    'yearly',
    12000.00, -- 12,000 دولار سنوياً
    500.00, -- 500 دولار شهرياً لإيجار السيرفر
    2000.00, -- 2,000 دولار رسوم إعداد
    10000.00, -- تنبيه عند انخفاض الرصيد عن 10,000
    '2024-01-01',
    '2024-02-01'
);
```

### الاستعلام عن البوابات النشطة

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- عرض جميع البوابات النشطة
SELECT 
    id,
    gateway_name,
    gateway_code,
    status,
    health_status,
    subscription_type,
    subscription_fee,
    server_rental_fee,
    next_billing_date,
    telecom_balance,
    balance_alert_threshold
FROM dedicated_gateways
WHERE status = 'active'
ORDER BY created_at DESC;
```

### تحديث رصيد شركة الاتصالات (للمراقبة)

```sql
-- تعيين الشركة الحالية
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- تحديث الرصيد بعد فحصه من API شركة الاتصالات
UPDATE dedicated_gateways
SET 
    telecom_balance = 45000.00,
    last_balance_check = CURRENT_TIMESTAMP
WHERE gateway_code = 'MODERN_TELECOM_GW';
```

### البحث عن البوابات التي تحتاج فوترة

```sql
-- البحث عن البوابات التي حان موعد فوترتها
SELECT 
    id,
    tenant_id,
    gateway_name,
    gateway_code,
    subscription_fee,
    server_rental_fee,
    next_billing_date
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND next_billing_date <= CURRENT_DATE
ORDER BY next_billing_date;
```

### البحث عن البوابات التي تحتاج تنبيه رصيد

```sql
-- البحث عن البوابات التي انخفض رصيدها
SELECT 
    id,
    tenant_id,
    gateway_name,
    gateway_code,
    telecom_balance,
    balance_alert_threshold
FROM dedicated_gateways
WHERE 
    status = 'active'
    AND balance_check_enabled = true
    AND telecom_balance IS NOT NULL
    AND balance_alert_threshold IS NOT NULL
    AND telecom_balance < balance_alert_threshold
ORDER BY telecom_balance;
```

## الأمان

### تشفير بيانات الاعتماد

**يجب** تشفير الحقول التالية قبل حفظها:

- `telecom_password_encrypted`
- `telecom_api_key_encrypted`

استخدم AES-256 للتشفير:

```go
// مثال في Go
import "crypto/aes"

func EncryptCredential(plaintext, key string) (string, error) {
    // تطبيق AES-256 encryption
    // ...
}
```

### Row Level Security

RLS مفعّل تلقائياً ويضمن:

- عزل تام بين الشركات
- عدم إمكانية Cross-Tenant Access
- حماية من SQL Injection على مستوى البيانات

## الفوترة التلقائية

### كيفية عمل الفوترة

1. **Cron Job يومي** يفحص `next_billing_date`
2. عند حلول موعد الفوترة:
   - خصم `subscription_fee + server_rental_fee` من رصيد الشركة
   - تسجيل في `accounting_transactions`
   - تحديث `next_billing_date` للشهر/السنة القادمة
   - إرسال فاتورة للشركة

3. **ملاحظة مهمة:** لا يتم خصم أي تكلفة متعلقة بالرسائل

### مثال على حساب الفوترة

```
اشتراك سنوي:
- subscription_fee: $12,000 (سنوياً)
- server_rental_fee: $500 (شهرياً)

الفوترة الشهرية:
- الشهر الأول: $12,000 + $500 = $12,500 (اشتراك + إيجار)
- الأشهر التالية: $500 فقط (إيجار فقط)
- بعد سنة: $12,000 + $500 = $12,500 (تجديد الاشتراك + إيجار)
```

## مراقبة الرصيد

### كيفية عمل المراقبة

1. **Cron Job كل ساعة** يفحص البوابات التي `balance_check_enabled = true`
2. استدعاء API شركة الاتصالات للحصول على الرصيد الحالي
3. تحديث `telecom_balance` و `last_balance_check`
4. إذا `telecom_balance < balance_alert_threshold`:
   - إرسال تنبيه للشركة المؤسسية
   - تسجيل في سجل التنبيهات

### مثال على التنبيه

```
عتبة التنبيه: $10,000
الرصيد الحالي: $8,500

التنبيه:
"تحذير: رصيدك لدى شركة الاتصالات انخفض إلى $8,500
يرجى التواصل مع شركة الاتصالات لإعادة الشحن"
```

## الصيانة

### تحديث الجدول

```sql
-- إضافة عمود جديد
ALTER TABLE dedicated_gateways 
ADD COLUMN new_column VARCHAR(255);

-- تحديث RLS بعد التعديل
-- (RLS يبقى نشط تلقائياً)
```

### النسخ الاحتياطي

```bash
# نسخ احتياطي للجدول
pg_dump -t dedicated_gateways sms_gateway > dedicated_gateways_backup.sql

# استرجاع
psql sms_gateway < dedicated_gateways_backup.sql
```

## المتطلبات المرتبطة

هذا الجدول يلبي المتطلبات التالية:

- **8.1**: حفظ اسم البوابة ورمزها الفريد
- **8.2**: ربط البوابة بشركة مؤسسية واحدة (tenant_id)
- **8.3**: حفظ معلومات الاتصال بشركة الاتصالات
- **8.4**: تشفير بيانات الاعتماد
- **8.10**: حفظ نوع الاشتراك ورسوم الاشتراك
- **8.11**: حفظ رسوم إيجار السيرفر
- **8.13**: مراقبة رصيد الشركة لدى شركة الاتصالات
- **8.14**: إرسال تنبيه عند انخفاض الرصيد

## الخطوات التالية

بعد إنشاء هذا الجدول، يجب:

1. إنشاء جدول `resource_quotas` للحصص الموردية
2. إنشاء جدول `resource_usage` لتسجيل الاستخدام
3. إنشاء جدول `gateway_users` لإدارة المستخدمين
4. إنشاء جدول `gateway_api_keys` لإدارة مفاتيح API
5. إنشاء جدول `gateway_statistics` للإحصائيات اليومية
6. تطوير `Gateway Manager` في Go
7. تطوير `Balance Monitor` للمراقبة الدورية
8. تطوير `Billing Service` للفوترة التلقائية
