# Multi-Tenant Scenarios Test - Quick Start Guide

## نظرة عامة

هذا الدليل يشرح كيفية تشغيل اختبار Multi-Tenant Scenarios (المهمة 15.4) الذي يتحقق من العزل التام بين الشركات في النظام.

## الهدف من الاختبار

التحقق من:
1. ✅ إرسال رسائل من عدة tenants متزامنة
2. ✅ العزل التام بين tenants في جميع المكونات
3. ✅ عدم تأثير tenant على رصيد tenant آخر
4. ✅ عزل Rate Limiting لكل tenant
5. ✅ عزل القنوات والرسائل بشكل صحيح
6. ✅ عدم تسرب البيانات بين tenants

## المتطلبات المرتبطة

- **Requirement 1.1**: حماية الرصيد المالي من Race Conditions
- **Requirement 3.4**: Rate Limiting لكل Tenant
- **Requirement 5.1**: اختيار قناة الإرسال بذكاء

## المتطلبات الأساسية

### 1. البنية التحتية
```bash
# تشغيل PostgreSQL و Redis
docker-compose up -d postgres redis

# التحقق من التشغيل
docker ps
```

### 2. قاعدة البيانات
```bash
# تشغيل Migrations
cd services/backend
go run cmd/migrate/main.go up
```

### 3. التبعيات
```bash
# تثبيت Go dependencies
cd services/backend
go mod download
```

## تشغيل الاختبار

### الطريقة السريعة (Recommended)

#### Windows (PowerShell):
```powershell
.\services\backend\run-multi-tenant-test.ps1
```

#### Linux/Mac:
```bash
chmod +x services/backend/run-multi-tenant-test.sh
./services/backend/run-multi-tenant-test.sh
```

### الطريقة اليدوية

```bash
cd services/backend/internal/integration
go test -v -run TestMultiTenant -timeout 5m
```

## سيناريوهات الاختبار

### Test 1: TestMultiTenantScenarios

اختبار شامل للعزل بين الشركات:

#### الإعداد:
- 3 tenants مع أرصدة مختلفة
- 50 رسالة لكل tenant
- إرسال متزامن من جميع الـ tenants

#### المراحل:

**Phase 1: Setup Test Tenants**
```
✅ Created Tenant-1 (ID: 1) with balance: 100.00
✅ Created Tenant-2 (ID: 2) with balance: 150.00
✅ Created Tenant-3 (ID: 3) with balance: 200.00
```

**Phase 2: Send Messages Concurrently**
```
📤 Sending messages from all tenants concurrently...
✅ All tenants finished sending in 5.2s
   Tenant-1: Sent 50 messages, 0 errors
   Tenant-2: Sent 50 messages, 0 errors
   Tenant-3: Sent 50 messages, 0 errors
```

**Phase 3: Verify Buffer Isolation**
```
🔍 Verifying buffer isolation...
   ✅ Tenant-1: Buffer size = 50 (isolated)
   ✅ Tenant-2: Buffer size = 50 (isolated)
   ✅ Tenant-3: Buffer size = 50 (isolated)
```

**Phase 4: Process Batches**
```
⚙️  Processing batches for all tenants...
   ✅ Tenant-1: All batches processed
   ✅ Tenant-2: All batches processed
   ✅ Tenant-3: All batches processed
```

**Phase 5: Verify Database Isolation**
```
🗄️  Verifying database isolation...
   ✅ Tenant-1: 50 messages in database (no cross-tenant data)
   ✅ Tenant-2: 50 messages in database (no cross-tenant data)
   ✅ Tenant-3: 50 messages in database (no cross-tenant data)
```

**Phase 6: Verify Balance Isolation**
```
💰 Verifying balance isolation...
   ✅ Tenant-1: Balance = 100.00 (unchanged), History entries = 0
   ✅ Tenant-2: Balance = 150.00 (unchanged), History entries = 0
   ✅ Tenant-3: Balance = 200.00 (unchanged), History entries = 0
```

**Phase 7: Verify Channel Isolation**
```
📡 Verifying channel isolation...
   ✅ Tenant-1: Has 1 channel(s) (isolated)
   ✅ Tenant-2: Has 1 channel(s) (isolated)
   ✅ Tenant-3: Has 1 channel(s) (isolated)
```

**Phase 8: Verify Rate Limiting Isolation**
```
⏱️  Verifying rate limiting isolation...
   ✅ Tenant-1: Rate limit triggered at message 101
   ✅ Tenant-2: Not affected by Tenant-1 rate limit
   ✅ Tenant-3: Not affected by Tenant-1 rate limit
```

**Phase 9: Verify Redis Key Isolation**
```
🔑 Verifying Redis key isolation...
   ✅ Tenant-1: Redis key 'messages:pending:1' exists = 1
   ✅ Tenant-2: Redis key 'messages:pending:2' exists = 1
   ✅ Tenant-3: Redis key 'messages:pending:3' exists = 1
```

### Test 2: TestMultiTenantBalanceIsolation

اختبار عزل العمليات المالية:

#### الإعداد:
- 5 tenants مع رصيد متساوي (100.00)
- 100 عملية خصم لكل tenant
- عمليات متزامنة من جميع الـ tenants

#### المراحل:

**Phase 1: Setup Test Tenants**
```
✅ Created Tenant-1 (ID: 1) with balance: 100.00
✅ Created Tenant-2 (ID: 2) with balance: 100.00
✅ Created Tenant-3 (ID: 3) with balance: 100.00
✅ Created Tenant-4 (ID: 4) with balance: 100.00
✅ Created Tenant-5 (ID: 5) with balance: 100.00
```

**Phase 2: Perform Concurrent Balance Operations**
```
⚙️  Performing concurrent balance operations...
✅ All operations completed in 2.3s
```

**Phase 3: Verify Balance Isolation**
```
🔍 Verifying balance isolation...
   ✅ Tenant-1: Balance = 95.00 (expected: 95.00), Success = 100, Failed = 0, History = 100
   ✅ Tenant-2: Balance = 95.00 (expected: 95.00), Success = 100, Failed = 0, History = 100
   ✅ Tenant-3: Balance = 95.00 (expected: 95.00), Success = 100, Failed = 0, History = 100
   ✅ Tenant-4: Balance = 95.00 (expected: 95.00), Success = 100, Failed = 0, History = 100
   ✅ Tenant-5: Balance = 95.00 (expected: 95.00), Success = 100, Failed = 0, History = 100
```

**Phase 4: Verify No Cross-Tenant Balance History**
```
🔒 Verifying no cross-tenant balance history...
   ✅ Tenant-1: No cross-tenant history entries
   ✅ Tenant-2: No cross-tenant history entries
   ✅ Tenant-3: No cross-tenant history entries
   ✅ Tenant-4: No cross-tenant history entries
   ✅ Tenant-5: No cross-tenant history entries
```

## النتائج المتوقعة

### نجاح الاختبار ✅

```
✅ Multi-Tenant Scenarios test PASSED!

Verified:
  ✓ Complete data isolation between tenants
  ✓ No cross-tenant data leakage
  ✓ Independent rate limiting per tenant
  ✓ Isolated balance operations
  ✓ Isolated channel management

📋 Requirements validated: 1.1, 3.4, 5.1

Next steps:
  1. ✅ Task 15.4 completed
  2. ➡️  Continue to Task 15.5 (WebSocket Reconnection)
  3. ➡️  Continue to Task 15.6 (Database Failure Recovery)
```

### فشل الاختبار ❌

إذا فشل الاختبار، تحقق من:

1. **Database Connection Issues**
   ```bash
   # التحقق من PostgreSQL
   docker ps | grep postgres
   docker logs postgres
   ```

2. **Redis Connection Issues**
   ```bash
   # التحقق من Redis
   docker ps | grep redis
   docker logs redis
   ```

3. **Missing Migrations**
   ```bash
   # تشغيل Migrations
   cd services/backend
   go run cmd/migrate/main.go up
   ```

4. **Port Conflicts**
   ```bash
   # التحقق من المنافذ
   netstat -an | grep 5432  # PostgreSQL
   netstat -an | grep 6379  # Redis
   ```

## التحقق من النتائج

### 1. فحص قاعدة البيانات

```sql
-- عدد الرسائل لكل tenant
SELECT tenant_id, COUNT(*) as message_count
FROM messages
WHERE deleted_at IS NULL
GROUP BY tenant_id
ORDER BY tenant_id;

-- التحقق من عدم وجود cross-tenant data
SELECT 
    m.tenant_id,
    COUNT(*) as total_messages,
    COUNT(DISTINCT m.phone_number) as unique_phones
FROM messages m
WHERE m.deleted_at IS NULL
GROUP BY m.tenant_id;

-- فحص balance history
SELECT 
    tenant_id,
    COUNT(*) as history_count,
    SUM(amount) as total_amount
FROM balance_history
GROUP BY tenant_id
ORDER BY tenant_id;
```

### 2. فحص Redis

```bash
# الاتصال بـ Redis
docker exec -it redis redis-cli

# فحص المفاتيح
KEYS messages:pending:*

# فحص حجم Buffer لكل tenant
LLEN messages:pending:1
LLEN messages:pending:2
LLEN messages:pending:3

# فحص rate limiting
KEYS rate_limit:*
```

## معايير النجاح

الاختبار يعتبر ناجحاً إذا:

1. ✅ جميع الرسائل تم إرسالها بنجاح من جميع الـ tenants
2. ✅ كل tenant له buffer منفصل في Redis
3. ✅ كل tenant له رسائل منفصلة في قاعدة البيانات
4. ✅ لا يوجد cross-tenant data في أي جدول
5. ✅ Rate limiting يعمل بشكل مستقل لكل tenant
6. ✅ العمليات المالية معزولة تماماً بين tenants
7. ✅ Balance history منفصلة لكل tenant
8. ✅ القنوات معزولة لكل tenant

## الخطوات التالية

بعد نجاح هذه الاختبارات:
1. ✅ المهمة 15.4 مكتملة
2. ➡️ الانتقال إلى المهمة 15.5 (WebSocket Reconnection)
3. ➡️ الانتقال إلى المهمة 15.6 (Database Failure Recovery)

## الدعم والمساعدة

إذا واجهت مشاكل:
1. راجع logs الاختبار بعناية
2. تحقق من البنية التحتية (Docker containers)
3. تحقق من الاتصال بقاعدة البيانات و Redis
4. راجع الوثائق في `services/backend/internal/integration/README.md`

## الملفات ذات الصلة

- `services/backend/internal/integration/multi_tenant_test.go` - ملف الاختبار
- `services/backend/run-multi-tenant-test.ps1` - سكريبت Windows
- `services/backend/run-multi-tenant-test.sh` - سكريبت Linux/Mac
- `.kiro/specs/message-lifecycle-rebuild/tasks.md` - خطة التنفيذ
- `.kiro/specs/message-lifecycle-rebuild/requirements.md` - المتطلبات
- `.kiro/specs/message-lifecycle-rebuild/design.md` - التصميم
