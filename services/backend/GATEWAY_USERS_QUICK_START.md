# دليل البدء السريع - جدول مستخدمي البوابات المخصصة

## نظرة سريعة

جدول `gateway_users` يسجل المستخدمين المصرح لهم بالوصول لكل بوابة مخصصة مع تحديد أدوارهم وصلاحياتهم.

### الأدوار الثلاثة

1. **Admin** (المدير): كامل الصلاحيات
2. **Operator** (المشغّل): إرسال ومراقبة
3. **Viewer** (المشاهد): عرض فقط

## التثبيت السريع

### 1. تشغيل Migration

```powershell
# Windows
.\run-gateway-users-migration.ps1

# Linux/Mac
chmod +x run-gateway-users-migration.sh
./run-gateway-users-migration.sh
```

### 2. التحقق من التثبيت

```sql
-- التحقق من الجدول
SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'gateway_users';

-- التحقق من RLS
SELECT COUNT(*) FROM pg_policies WHERE tablename = 'gateway_users';
```

## الاستخدام السريع

### إضافة مستخدم Admin

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-id';

-- إضافة مستخدم
INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    'your-tenant-id',
    1,  -- gateway_id
    'user-uuid',
    'admin'
);
```

### إضافة مستخدم Operator

```sql
INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    'your-tenant-id',
    1,
    'user-uuid',
    'operator'
);
```

### إضافة مستخدم Viewer

```sql
INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role)
VALUES (
    'your-tenant-id',
    1,
    'user-uuid',
    'viewer'
);
```

### عرض جميع المستخدمين

```sql
SELECT 
    gu.id,
    u.email,
    u.full_name,
    gu.role,
    gu.is_active,
    gu.created_at
FROM gateway_users gu
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 1
ORDER BY gu.role, u.full_name;
```

### تحديث صلاحيات مستخدم

```sql
-- تعطيل صلاحية معينة
UPDATE gateway_users
SET permissions = jsonb_set(
    permissions,
    '{export_reports}',
    'false'
)
WHERE gateway_id = 1 AND user_id = 'user-uuid';
```

### تعطيل مستخدم

```sql
UPDATE gateway_users
SET is_active = false
WHERE gateway_id = 1 AND user_id = 'user-uuid';
```

### حذف مستخدم

```sql
DELETE FROM gateway_users
WHERE gateway_id = 1 AND user_id = 'user-uuid';
```

## الصلاحيات الافتراضية

### Admin

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

### Operator

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

### Viewer

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

## التحقق من الصلاحيات

### التحقق من صلاحية معينة

```sql
-- التحقق من صلاحية إرسال الرسائل
SELECT 
    (permissions->>'send_messages')::boolean AS can_send
FROM gateway_users
WHERE gateway_id = 1 
  AND user_id = 'user-uuid'
  AND is_active = true;
```

### التحقق من جميع الصلاحيات

```sql
SELECT 
    role,
    permissions->>'send_messages' AS send_messages,
    permissions->>'view_reports' AS view_reports,
    permissions->>'manage_users' AS manage_users,
    permissions->>'manage_gateway' AS manage_gateway,
    permissions->>'manage_api_keys' AS manage_api_keys,
    permissions->>'view_statistics' AS view_statistics,
    permissions->>'export_reports' AS export_reports
FROM gateway_users
WHERE gateway_id = 1 AND user_id = 'user-uuid';
```

## إحصائيات سريعة

### عدد المستخدمين لكل دور

```sql
SELECT 
    role,
    COUNT(*) AS user_count,
    COUNT(CASE WHEN is_active = true THEN 1 END) AS active_count
FROM gateway_users
WHERE gateway_id = 1
GROUP BY role
ORDER BY role;
```

### المستخدمون النشطون

```sql
SELECT 
    u.email,
    u.full_name,
    gu.role,
    gu.created_at
FROM gateway_users gu
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 1 
  AND gu.is_active = true
ORDER BY gu.created_at DESC;
```

## استكشاف الأخطاء

### خطأ: دور غير صحيح

```
ERROR: Invalid role: xxx. Must be admin, operator, or viewer
```

**الحل**: استخدم أحد الأدوار الصحيحة: `admin`، `operator`، `viewer`

### خطأ: مستخدم مكرر

```
ERROR: duplicate key value violates unique constraint "uq_gateway_users_gateway_user"
```

**الحل**: المستخدم موجود بالفعل في هذه البوابة. استخدم UPDATE بدلاً من INSERT.

### خطأ: tenant_id غير متطابق

```
ERROR: Gateway X does not belong to tenant Y
```

**الحل**: تأكد من أن البوابة والمستخدم ينتميان لنفس الشركة.

### خطأ: لا يمكن رؤية البيانات

**السبب**: RLS يمنع الوصول لبيانات شركة أخرى

**الحل**: تأكد من تعيين `app.current_tenant` الصحيح:

```sql
SET app.current_tenant = 'your-tenant-id';
```

## أمثلة متقدمة

### إضافة مستخدم بصلاحيات مخصصة

```sql
INSERT INTO gateway_users (tenant_id, gateway_id, user_id, role, permissions)
VALUES (
    'your-tenant-id',
    1,
    'user-uuid',
    'operator',
    '{"send_messages": true, "view_reports": true, "export_reports": false}'::jsonb
);
```

### تغيير دور مستخدم

```sql
-- تغيير من Operator إلى Admin
UPDATE gateway_users
SET role = 'admin'
WHERE gateway_id = 1 AND user_id = 'user-uuid';

-- الصلاحيات الافتراضية للـ Admin سيتم تعيينها تلقائياً
```

### البحث عن مستخدمين بصلاحية معينة

```sql
-- البحث عن جميع المستخدمين الذين يمكنهم إرسال الرسائل
SELECT 
    u.email,
    gu.role,
    gu.permissions->>'send_messages' AS can_send
FROM gateway_users gu
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 1
  AND gu.is_active = true
  AND (gu.permissions->>'send_messages')::boolean = true;
```

### تقرير شامل للمستخدمين

```sql
SELECT 
    dg.gateway_name,
    u.email,
    u.full_name,
    gu.role,
    CASE 
        WHEN gu.is_active THEN 'نشط'
        ELSE 'معطل'
    END AS status,
    gu.permissions->>'send_messages' AS send_messages,
    gu.permissions->>'manage_users' AS manage_users,
    gu.created_at,
    gu.updated_at
FROM gateway_users gu
JOIN dedicated_gateways dg ON gu.gateway_id = dg.id
JOIN users u ON gu.user_id = u.id
WHERE gu.gateway_id = 1
ORDER BY gu.role, gu.is_active DESC, u.full_name;
```

## التكامل مع Go

### نموذج البيانات

```go
type GatewayUser struct {
    ID          int                    `json:"id"`
    TenantID    uuid.UUID              `json:"tenant_id"`
    GatewayID   int                    `json:"gateway_id"`
    UserID      uuid.UUID              `json:"user_id"`
    Role        string                 `json:"role"` // admin, operator, viewer
    Permissions map[string]interface{} `json:"permissions"`
    IsActive    bool                   `json:"is_active"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}
```

### إضافة مستخدم

```go
func AddGatewayUser(db *gorm.DB, tenantID uuid.UUID, gatewayID int, userID uuid.UUID, role string) error {
    // تعيين tenant_id في الجلسة
    db.Exec("SET app.current_tenant = ?", tenantID.String())
    
    user := &GatewayUser{
        TenantID:  tenantID,
        GatewayID: gatewayID,
        UserID:    userID,
        Role:      role,
        IsActive:  true,
    }
    
    return db.Create(user).Error
}
```

### التحقق من الصلاحيات

```go
func CheckPermission(db *gorm.DB, tenantID uuid.UUID, gatewayID int, userID uuid.UUID, permission string) (bool, error) {
    // تعيين tenant_id في الجلسة
    db.Exec("SET app.current_tenant = ?", tenantID.String())
    
    var user GatewayUser
    err := db.Where("gateway_id = ? AND user_id = ? AND is_active = ?", gatewayID, userID, true).
        First(&user).Error
    
    if err != nil {
        return false, err
    }
    
    // التحقق من الصلاحية
    if val, ok := user.Permissions[permission]; ok {
        if boolVal, ok := val.(bool); ok {
            return boolVal, nil
        }
    }
    
    return false, nil
}
```

## الأداء

### Indexes المستخدمة

- `idx_gateway_users_tenant`: للعزل بين الشركات
- `idx_gateway_users_tenant_gateway`: للبحث حسب البوابة
- `idx_gateway_users_tenant_user`: للبحث حسب المستخدم
- `idx_gateway_users_tenant_role`: للبحث حسب الدور
- `idx_gateway_users_tenant_active`: للبحث حسب الحالة

### نصائح للأداء

1. استخدم `tenant_id` في جميع الاستعلامات
2. استخدم Indexes المركبة للبحث المتقدم
3. تجنب `SELECT *` واختر الحقول المطلوبة فقط
4. استخدم `LIMIT` للاستعلامات الكبيرة

## الأمان

### Row Level Security (RLS)

- تم تفعيل RLS على الجدول
- لا يمكن الوصول لبيانات شركة أخرى
- يجب تعيين `app.current_tenant` في كل جلسة

### التحقق من الصلاحيات

- تحقق من `is_active` قبل السماح بأي عملية
- تحقق من الصلاحيات التفصيلية في `permissions`
- سجل جميع العمليات الحساسة في Audit Log

## الموارد الإضافية

- **README الكامل**: `migrations/README_GATEWAY_USERS.md`
- **Migration**: `migrations/023_gateway_users_table.sql`
- **اختبارات**: `test_gateway_users_migration.sql`
- **التصميم**: `.kiro/specs/tariff-pricing-system/design.md`
- **المتطلبات**: `.kiro/specs/tariff-pricing-system/requirements.md`

## الخطوات التالية

1. ✅ **تم**: جدول `dedicated_gateways` (المهمة 16.1)
2. ✅ **تم**: جدول `resource_quotas` (المهمة 16.2)
3. ✅ **تم**: جدول `resource_usage` (المهمة 16.3)
4. ✅ **الحالي**: جدول `gateway_users` (المهمة 16.4)
5. ⏳ **التالي**: جدول `gateway_api_keys` (المهمة 16.5)
6. ⏳ جدول `gateway_statistics` (المهمة 16.6)
7. ⏳ تطوير Gateway Manager (المهمة 17)
8. ⏳ تطوير User Management (المهمة 20)

## الدعم

للمزيد من المساعدة:
- راجع ملف README الكامل
- راجع ملف التصميم للتفاصيل الفنية
- راجع ملف المتطلبات للمتطلبات التفصيلية
