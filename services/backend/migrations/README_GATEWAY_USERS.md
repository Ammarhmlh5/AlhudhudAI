# جدول مستخدمي البوابات المخصصة (gateway_users)

## نظرة عامة

جدول `gateway_users` يسجل المستخدمين المصرح لهم بالوصول لكل بوابة مخصصة مع تحديد أدوارهم وصلاحياتهم. يدعم الجدول ثلاثة أدوار رئيسية: **admin** (كامل الصلاحيات)، **operator** (إرسال ومراقبة)، **viewer** (عرض فقط).

## المتطلبات المرتبطة

- **Requirement 11.1**: دعم إضافة مستخدمين متعددين لكل بوابة مخصصة
- **Requirement 11.2**: دعم أدوار مختلفة (مدير، مشغّل، مشاهد)
- **Requirement 11.3**: تحديد صلاحيات المستخدم حسب دوره

## هيكل الجدول

### الحقول الأساسية

| الحقل | النوع | الوصف |
|------|------|-------|
| `id` | SERIAL | المعرف الفريد للمستخدم في البوابة |
| `tenant_id` | UUID | معرف الشركة (للعزل بين الشركات) |
| `gateway_id` | INTEGER | معرف البوابة المخصصة |
| `user_id` | UUID | معرف المستخدم |

### الصلاحيات

| الحقل | النوع | الوصف |
|------|------|-------|
| `role` | VARCHAR(50) | دور المستخدم: admin، operator، viewer |
| `permissions` | JSONB | الصلاحيات التفصيلية بصيغة JSON |

### الحالة

| الحقل | النوع | الوصف |
|------|------|-------|
| `is_active` | BOOLEAN | حالة المستخدم (نشط/معطل) |

### التواريخ

| الحقل | النوع | الوصف |
|------|------|-------|
| `created_at` | TIMESTAMP | تاريخ إضافة المستخدم |
| `updated_at` | TIMESTAMP | تاريخ آخر تحديث |

## الأدوار والصلاحيات

### 1. Admin (المدير)

**الوصف**: كامل الصلاحيات على البوابة

**الصلاحيات الافتراضية**:
```json
{
  "send_messages": true,
  "view_reports": true,
  "manage_users": true,
  "manage_gateway": true,
  "manage_api_keys": true,
  "view_statistics": true,
  "export_reports": true
}
```

**الاستخدامات**:
- إدارة البوابة والإعدادات
- إضافة وحذف المستخدمين
- إنشاء وإدارة API Keys
- إرسال الرسائل
- عرض جميع التقارير والإحصائيات
- تصدير التقارير

### 2. Operator (المشغّل)

**الوصف**: إرسال الرسائل ومراقبة الأداء

**الصلاحيات الافتراضية**:
```json
{
  "send_messages": true,
  "view_reports": true,
  "manage_users": false,
  "manage_gateway": false,
  "manage_api_keys": false,
  "view_statistics": true,
  "export_reports": true
}
```

**الاستخدامات**:
- إرسال الرسائل
- عرض التقارير والإحصائيات
- تصدير التقارير
- **لا يمكنه**: إدارة المستخدمين أو البوابة أو API Keys

### 3. Viewer (المشاهد)

**الوصف**: عرض التقارير فقط

**الصلاحيات الافتراضية**:
```json
{
  "send_messages": false,
  "view_reports": true,
  "manage_users": false,
  "manage_gateway": false,
  "manage_api_keys": false,
  "view_statistics": true,
  "export_reports": false
}
```

**الاستخدامات**:
- عرض التقارير والإحصائيات فقط
- **لا يمكنه**: إرسال رسائل، إدارة المستخدمين، تصدير التقارير

## العزل بين الشركات (Multi-Tenancy)

### Row Level Security (RLS)

الجدول محمي بـ RLS لضمان العزل التام بين الشركات:

```sql
-- Policy للقراءة
CREATE POLICY tenant_isolation_policy ON gateway_users
    FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للإدراج
CREATE POLICY tenant_isolation_insert_policy ON gateway_users
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للتحديث
CREATE POLICY tenant_isolation_update_policy ON gateway_users
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

-- Policy للحذف
CREATE POLICY tenant_isolation_delete_policy ON gateway_users
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### Indexes للأداء

جميع الـ Indexes تبدأ بـ `tenant_id` لتحسين الأداء:

```sql
CREATE INDEX idx_gateway_users_tenant ON gateway_users(tenant_id);
CREATE INDEX idx_gateway_users_tenant_gateway ON gateway_users(tenant_id, gateway_id);
CREATE INDEX idx_gateway_users_tenant_user ON gateway_users(tenant_id, user_id);
CREATE INDEX idx_gateway_users_tenant_role ON gateway_users(tenant_id, role);
CREATE INDEX idx_gateway_users_tenant_active ON gateway_users(tenant_id, is_active);
```

## Triggers التلقائية

### 1. تحديث updated_at

يتم تحديث `updated_at` تلقائياً عند أي تعديل:

```sql
CREATE TRIGGER trigger_update_gateway_users_updated_at
    BEFORE UPDATE ON gateway_users
    FOR EACH ROW
    EXECUTE FUNCTION update_gateway_users_updated_at();
```

### 2. التحقق من صلاحيات الأدوار

يتم التحقق من الدور وتعيين الصلاحيات الافتراضية تلقائياً:

```sql
CREATE TRIGGER trigger_validate_gateway_user_role
    BEFORE INSERT OR UPDATE ON gateway_users
    FOR EACH ROW
    EXECUTE FUNCTION validate_gateway_user_role();
```

**الوظائف**:
- التحقق من أن الدور صحيح (admin، operator، viewer)
- تعيين الصلاحيات الافتراضية إذا لم تكن محددة
- رفع استثناء إذا كان الدور غير صحيح

### 3. التحقق من tenant_id consistency

يتم التحقق من أن البوابة والمستخدم ينتميان لنفس الشركة:

```sql
CREATE TRIGGER trigger_check_gateway_user_tenant_consistency
    BEFORE INSERT OR UPDATE ON gateway_users
    FOR EACH ROW
    EXECUTE FUNCTION check_gateway_user_tenant_consistency();
```

**الوظائف**:
- التحقق من أن `gateway_id` ينتمي لنفس `tenant_id`
- التحقق من أن `user_id` ينتمي لنفس `tenant_id`
- رفع استثناء إذا كان هناك عدم تطابق

## القيود (Constraints)

### 1. Unique Constraint

```sql
CONSTRAINT uq_gateway_users_gateway_user UNIQUE(gateway_id, user_id)
```

**الغرض**: منع إضافة نفس المستخدم مرتين لنفس البوابة

### 2. Foreign Key Constraint

```sql
CONSTRAINT fk_gateway_users_tenant_gateway FOREIGN KEY (tenant_id, gateway_id) 
    REFERENCES dedicated_gateways(tenant_id, id) ON DELETE CASCADE
```

**الغرض**: ضمان أن البوابة والمستخدم ينتميان لنفس الشركة

### 3. Check Constraint

```sql
CHECK (role IN ('admin', 'operator', 'viewer'))
```

**الغرض**: ضمان أن الدور صحيح

## أمثلة الاستخدام

### 1. إضافة مستخدم Admin

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- إضافة مستخدم Admin
INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    '987fcdeb-51a2-43f7-b890-123456789abc',
    'admin'
);

-- الصلاحيات الافتراضية سيتم تعيينها تلقائياً
```

### 2. إضافة مستخدم Operator بصلاحيات مخصصة

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- إضافة مستخدم Operator مع صلاحيات مخصصة
INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role, permissions)
VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    '456789ab-cdef-0123-4567-89abcdef0123',
    'operator',
    '{"send_messages": true, "view_reports": true, "export_reports": false}'::jsonb
);
```

### 3. عرض جميع مستخدمي بوابة معينة

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- عرض المستخدمين
SELECT 
    gu.id,
    gu.gateway_id,
    u.email,
    u.full_name,
    gu.role,
    gu.permissions,
    gu.is_active,
    gu.created_at
FROM gateway_users gu
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 1
ORDER BY gu.created_at DESC;
```

### 4. تحديث صلاحيات مستخدم

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- تحديث الصلاحيات
UPDATE gateway_users
SET permissions = jsonb_set(
    permissions,
    '{export_reports}',
    'true'
)
WHERE gateway_id = 1 AND user_id = '456789ab-cdef-0123-4567-89abcdef0123';
```

### 5. تعطيل مستخدم

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- تعطيل المستخدم
UPDATE gateway_users
SET is_active = false
WHERE gateway_id = 1 AND user_id = '456789ab-cdef-0123-4567-89abcdef0123';
```

### 6. حذف مستخدم

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- حذف المستخدم
DELETE FROM gateway_users
WHERE gateway_id = 1 AND user_id = '456789ab-cdef-0123-4567-89abcdef0123';
```

### 7. عرض المستخدمين حسب الدور

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- عرض جميع المديرين
SELECT 
    gu.id,
    dg.gateway_name,
    u.email,
    u.full_name,
    gu.created_at
FROM gateway_users gu
JOIN dedicated_gateways dg ON gu.gateway_id = dg.id
JOIN users u ON gu.user_id = u.id
WHERE gu.role = 'admin' AND gu.is_active = true
ORDER BY dg.gateway_name, u.full_name;
```

### 8. التحقق من صلاحية معينة

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- التحقق من صلاحية إرسال الرسائل
SELECT 
    gu.id,
    u.email,
    gu.role,
    (gu.permissions->>'send_messages')::boolean AS can_send_messages
FROM gateway_users gu
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 1 
  AND gu.user_id = '456789ab-cdef-0123-4567-89abcdef0123'
  AND gu.is_active = true
  AND (gu.permissions->>'send_messages')::boolean = true;
```

## إحصائيات مفيدة

### 1. عدد المستخدمين لكل بوابة

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

SELECT 
    dg.id,
    dg.gateway_name,
    COUNT(gu.id) AS total_users,
    COUNT(CASE WHEN gu.role = 'admin' THEN 1 END) AS admin_count,
    COUNT(CASE WHEN gu.role = 'operator' THEN 1 END) AS operator_count,
    COUNT(CASE WHEN gu.role = 'viewer' THEN 1 END) AS viewer_count,
    COUNT(CASE WHEN gu.is_active = true THEN 1 END) AS active_users
FROM dedicated_gateways dg
LEFT JOIN gateway_users gu ON dg.id = gu.gateway_id
GROUP BY dg.id, dg.gateway_name
ORDER BY dg.gateway_name;
```

### 2. المستخدمون النشطون في آخر 30 يوم

```sql
-- تعيين tenant_id في الجلسة
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

SELECT 
    dg.gateway_name,
    u.email,
    u.full_name,
    gu.role,
    gu.created_at
FROM gateway_users gu
JOIN dedicated_gateways dg ON gu.gateway_id = dg.id
JOIN users u ON gu.user_id = u.id
WHERE gu.is_active = true
  AND gu.created_at >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY gu.created_at DESC;
```

## الأمان

### 1. تشفير البيانات

- جميع البيانات الحساسة محمية بـ RLS
- لا يمكن الوصول لبيانات شركة أخرى

### 2. التحقق من الصلاحيات

- يجب التحقق من صلاحيات المستخدم قبل أي عملية
- استخدام JSONB للصلاحيات التفصيلية

### 3. Audit Trail

- يتم تسجيل `created_at` و `updated_at` لكل مستخدم
- يمكن إضافة جدول `audit_log` لتسجيل جميع التغييرات

## الأداء

### 1. Indexes

- جميع الـ Indexes تبدأ بـ `tenant_id` لتحسين الأداء
- Indexes على `gateway_id`، `user_id`، `role`

### 2. JSONB Performance

- استخدام JSONB للصلاحيات يوفر مرونة عالية
- يمكن إضافة GIN Index على `permissions` إذا لزم الأمر:

```sql
CREATE INDEX idx_gateway_users_permissions ON gateway_users USING GIN (permissions);
```

## التكامل مع الجداول الأخرى

### 1. dedicated_gateways

```sql
-- العلاقة: gateway_users.gateway_id -> dedicated_gateways.id
-- Cascade: عند حذف بوابة، يتم حذف جميع مستخدميها
```

### 2. users

```sql
-- العلاقة: gateway_users.user_id -> users.id
-- Cascade: عند حذف مستخدم، يتم حذف جميع ارتباطاته بالبوابات
```

### 3. tenants

```sql
-- العلاقة: gateway_users.tenant_id -> tenants.id
-- Cascade: عند حذف شركة، يتم حذف جميع مستخدمي بواباتها
```

## الخطوات التالية

بعد إنشاء جدول `gateway_users`، الخطوات التالية هي:

1. ✅ **تم**: جدول `dedicated_gateways` (المهمة 16.1)
2. ✅ **تم**: جدول `resource_quotas` (المهمة 16.2)
3. ✅ **تم**: جدول `resource_usage` (المهمة 16.3)
4. ✅ **الحالي**: جدول `gateway_users` (المهمة 16.4)
5. ⏳ **التالي**: جدول `gateway_api_keys` (المهمة 16.5)
6. ⏳ جدول `gateway_statistics` (المهمة 16.6)
7. ⏳ تطوير Gateway Manager (المهمة 17)
8. ⏳ تطوير User Management (المهمة 20)

## الدعم والمساعدة

للمزيد من المعلومات:
- راجع ملف `design.md` في `.kiro/specs/tariff-pricing-system/`
- راجع ملف `requirements.md` للمتطلبات التفصيلية
- راجع ملف `DEDICATED_GATEWAYS_QUICK_START.md` للبدء السريع
