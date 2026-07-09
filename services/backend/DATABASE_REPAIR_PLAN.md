# خطة الإصلاح الشاملة لقواعد البيانات
## الهدف: الوصول إلى 100% جاهزية للإنتاج

**التاريخ**: 22 مارس 2026  
**الحالة الحالية**: 78/100  
**الهدف النهائي**: 100/100

---

## ملخص تنفيذي

هذه الخطة تركز على **إصلاح المشاكل الموجودة فقط** دون إضافة ميزات جديدة. تشمل الإصلاحات:

### قواعد البيانات المستهدفة:
1. **PostgreSQL** (Backend) - التقييم الحالي: 78/100
2. **Redis** (Cache) - التقييم الحالي: 85/100
3. **SQLite** (Mobile) - التقييم الحالي: 72/100

---

## المرحلة 1: إصلاحات حرجة (أولوية قصوى)
**المدة المتوقعة**: 2-3 أيام  
**التأثير**: أمان وسلامة البيانات

### 1.1 PostgreSQL - إصلاح نظام RBAC
**المشكلة**: جداول role_permissions و user_roles مفقودة رغم وجود العلاقات في models.go

**الإصلاح**:
- إنشاء migration جديد: `050_fix_rbac_tables.sql`
- إنشاء جدول `role_permissions` (many-to-many)
- إنشاء جدول `user_roles` (many-to-many)
- إضافة Foreign Keys مع CASCADE
- إضافة Indexes للأداء

**الملفات المتأثرة**:
- `services/backend/migrations/050_fix_rbac_tables.sql` (جديد)

---

### 1.2 Redis - تأمين الاتصال
**المشكلة**: 
- REDIS_PASSWORD فارغ (خطر أمني حرج)
- Bind على جميع الواجهات (0.0.0.0)

**الإصلاح**:
- إنشاء ملف `redis.conf` مخصص
- تعيين requirepass قوي
- تقييد bind إلى 127.0.0.1 فقط
- تحديث `.env` بكلمة المرور

**الملفات المتأثرة**:
- `services/backend/redis.conf` (جديد)
- `services/backend/.env` (تحديث)
- `docker-compose.yml` (إن وجد)

---

### 1.3 SQLite - تفعيل Foreign Keys
**المشكلة**: Foreign Keys معطلة افتراضياً في SQLite

**الإصلاح**:
- تفعيل PRAGMA foreign_keys = ON عند فتح الاتصال
- إضافة Foreign Key constraints في schema
- تحديث DatabaseService.ts

**الملفات المتأثرة**:
- `apps/mobile_gateway/src/services/DatabaseService.ts`
- `apps/mobile_gateway/src/services/migrations.ts`

---

## المرحلة 2: إصلاحات عالية الأولوية
**المدة المتوقعة**: 3-4 أيام  
**التأثير**: سلامة البيانات والأداء

### 2.1 PostgreSQL - إضافة NOT NULL Constraints
**المشكلة**: أعمدة مهمة بدون NOT NULL

**الإصلاح**:
- إنشاء migration: `051_add_not_null_constraints.sql`
- إضافة NOT NULL للأعمدة:
  - `tenants.balance` (مع default 0)
  - `messages.tenant_id`
  - `messages.phone_number`
  - `messages.content`
  - `dispatch_channels.type`
  - `dispatch_channels.status`

**الملفات المتأثرة**:
- `services/backend/migrations/051_add_not_null_constraints.sql` (جديد)

---

### 2.2 PostgreSQL - توحيد دقة الأعمدة المالية
**المشكلة**: دقة غير متسقة (بعضها DECIMAL(12,2) وبعضها DECIMAL(12,4))

**الإصلاح**:
- إنشاء migration: `052_unify_decimal_precision.sql`
- توحيد جميع الأعمدة المالية إلى DECIMAL(12,4):
  - `accounting_transactions.amount`
  - `payment_transactions.amount`
  - `payment_transactions.refund_amount`
  - أي أعمدة مالية أخرى

**الملفات المتأثرة**:
- `services/backend/migrations/052_unify_decimal_precision.sql` (جديد)

---

### 2.3 SQLite - إضافة Indexes على جدول logs
**المشكلة**: جدول logs بدون indexes (بطء في الاستعلامات)

**الإصلاح**:
- إضافة migration جديد: version 2
- إنشاء indexes:
  - `idx_logs_message_id` على message_id
  - `idx_logs_timestamp` على timestamp
  - `idx_logs_event` على event

**الملفات المتأثرة**:
- `apps/mobile_gateway/src/services/migrations.ts`

---

### 2.4 Redis - تحسين الأداء والأمان
**المشكلة**: إعدادات افتراضية غير محسّنة

**الإصلاح**:
- تفعيل lazy freeing (lazyfree-lazy-eviction yes)
- تعيين maxmemory-policy (allkeys-lru)
- تفعيل AOF persistence
- تعيين timeout للاتصالات الخاملة

**الملفات المتأثرة**:
- `services/backend/redis.conf`

---

## المرحلة 3: إصلاحات متوسطة الأولوية
**المدة المتوقعة**: 4-5 أيام  
**التأثير**: اكتمال النظام والتوثيق

### 3.1 PostgreSQL - إضافة الجداول الإدارية المفقودة
**المشكلة**: 15 جدول إداري مفقود (مذكورة في التقرير)

**الإصلاح**:
- إنشاء migration: `053_add_admin_tables.sql`
- إضافة الجداول التالية فقط إذا كانت **مطلوبة للميزات الموجودة**:
  1. `admin_settings` - إعدادات النظام العامة
  2. `tenant_settings` - إعدادات خاصة بكل tenant
  3. `notifications` - إشعارات النظام
  4. `notification_preferences` - تفضيلات الإشعارات
  5. `system_health_checks` - فحوصات صحة النظام
  6. `api_rate_limits` - حدود معدل الاستخدام
  7. `webhook_endpoints` - نقاط webhook
  8. `webhook_deliveries` - سجل تسليم webhook
  9. `scheduled_tasks` - المهام المجدولة
  10. `feature_flags` - أعلام الميزات

**ملاحظة مهمة**: سيتم إضافة هذه الجداول **فقط** إذا كانت الميزات المرتبطة بها موجودة بالفعل في الكود.

**الملفات المتأثرة**:
- `services/backend/migrations/053_add_admin_tables.sql` (جديد)
- `services/backend/internal/models/models.go` (إضافة structs)

---

### 3.2 PostgreSQL - تحسين Indexes
**المشكلة**: بعض الاستعلامات الشائعة بدون indexes

**الإصلاح**:
- إنشاء migration: `054_optimize_indexes.sql`
- إضافة composite indexes:
  - `idx_messages_tenant_status_created` على (tenant_id, status, created_at)
  - `idx_staff_activities_tenant_created` على (tenant_id, created_at)
  - `idx_sessions_user_expires` على (user_id, expires_at)

**الملفات المتأثرة**:
- `services/backend/migrations/054_optimize_indexes.sql` (جديد)

---

### 3.3 SQLite - إضافة CHECK Constraints
**المشكلة**: بيانات غير صالحة يمكن إدخالها

**الإصلاح**:
- إضافة migration: version 3
- إضافة constraints:
  - `attempts >= 0`
  - `phone_number NOT NULL`
  - `content NOT NULL`

**الملفات المتأثرة**:
- `apps/mobile_gateway/src/services/migrations.ts`

---

## المرحلة 4: التوثيق والتحقق
**المدة المتوقعة**: 2 أيام  
**التأثير**: صيانة مستقبلية

### 4.1 تحديث التوثيق
**الإصلاح**:
- تحديث `docs/DATABASE_SCHEMA.md`
- توثيق جميع التغييرات
- إضافة ERD diagrams محدثة
- توثيق Redis configuration

**الملفات المتأثرة**:
- `docs/DATABASE_SCHEMA.md`
- `services/backend/README.md`
- `apps/mobile_gateway/README.md`

---

### 4.2 إنشاء سكريبتات التحقق
**الإصلاح**:
- إنشاء سكريبت `verify_database.sql` للتحقق من:
  - وجود جميع الجداول
  - وجود جميع Constraints
  - وجود جميع Indexes
  - دقة الأعمدة المالية
- إنشاء سكريبت `test_redis.sh` للتحقق من:
  - الاتصال بـ Redis
  - وجود كلمة المرور
  - إعدادات الأمان

**الملفات المتأثرة**:
- `services/backend/scripts/verify_database.sql` (جديد)
- `services/backend/scripts/test_redis.sh` (جديد)

---

## جدول التنفيذ

| المرحلة | الأولوية | المدة | الحالة |
|---------|----------|-------|--------|
| 1.1 - RBAC Tables | حرجة | 1 يوم | ⏳ قيد الانتظار |
| 1.2 - Redis Security | حرجة | 1 يوم | ⏳ قيد الانتظار |
| 1.3 - SQLite Foreign Keys | حرجة | 1 يوم | ⏳ قيد الانتظار |
| 2.1 - NOT NULL Constraints | عالية | 1 يوم | ⏳ قيد الانتظار |
| 2.2 - Decimal Precision | عالية | 1 يوم | ⏳ قيد الانتظار |
| 2.3 - SQLite Indexes | عالية | 1 يوم | ⏳ قيد الانتظار |
| 2.4 - Redis Optimization | عالية | 1 يوم | ⏳ قيد الانتظار |
| 3.1 - Admin Tables | متوسطة | 2 أيام | ⏳ قيد الانتظار |
| 3.2 - Optimize Indexes | متوسطة | 1 يوم | ⏳ قيد الانتظار |
| 3.3 - SQLite Constraints | متوسطة | 1 يوم | ⏳ قيد الانتظار |
| 4.1 - Documentation | منخفضة | 1 يوم | ⏳ قيد الانتظار |
| 4.2 - Verification Scripts | منخفضة | 1 يوم | ⏳ قيد الانتظار |

**المدة الإجمالية المتوقعة**: 12-14 يوم عمل

---

## التقييم المتوقع بعد الإصلاح

| قاعدة البيانات | التقييم الحالي | التقييم المتوقع | التحسين |
|----------------|----------------|-----------------|---------|
| PostgreSQL | 78/100 | 98/100 | +20 |
| Redis | 85/100 | 98/100 | +13 |
| SQLite | 72/100 | 95/100 | +23 |
| **الإجمالي** | **78/100** | **97/100** | **+19** |

**ملاحظة**: الوصول إلى 100% يتطلب اختبارات شاملة في بيئة الإنتاج.

---

## المخاطر والاحتياطات

### المخاطر المحتملة:
1. **فقدان البيانات**: عند تعديل schema
2. **توقف الخدمة**: أثناء تطبيق migrations
3. **مشاكل التوافق**: مع الكود الموجود

### الاحتياطات:
1. ✅ **نسخ احتياطي كامل** قبل كل migration
2. ✅ **اختبار في بيئة تطوير** أولاً
3. ✅ **تطبيق migrations تدريجياً** (واحد تلو الآخر)
4. ✅ **مراقبة الأداء** بعد كل تغيير
5. ✅ **خطة rollback** لكل migration

---

## الخطوات التالية

1. **مراجعة الخطة** والموافقة عليها
2. **إنشاء نسخة احتياطية** من جميع قواعد البيانات
3. **البدء بالمرحلة 1** (الإصلاحات الحرجة)
4. **اختبار كل مرحلة** قبل الانتقال للتالية
5. **تحديث التوثيق** بشكل مستمر

---

## ملاحظات مهمة

⚠️ **هذه الخطة تركز على الإصلاحات فقط - لا ميزات جديدة**

✅ جميع الإصلاحات المقترحة تعالج مشاكل موجودة في النظام الحالي

✅ لن يتم إضافة أي وظائف جديدة غير موجودة في الكود

✅ التركيز على: الأمان، سلامة البيانات، الأداء، والتوثيق
