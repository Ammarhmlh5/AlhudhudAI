# المرحلة 1: الإصلاحات الحرجة
## الأولوية: قصوى | المدة: 2-3 أيام

---

## 1.1 إصلاح نظام RBAC في PostgreSQL

### المشكلة
- جدول `role_permissions` مفقود (many-to-many بين roles و permissions)
- جدول `user_roles` مفقود (many-to-many بين users و roles)
- العلاقات موجودة في `models.go` لكن الجداول غير موجودة في قاعدة البيانات

### التأثير
- نظام الصلاحيات غير مكتمل
- لا يمكن ربط المستخدمين بالأدوار
- لا يمكن ربط الأدوار بالصلاحيات

### الحل المقترح

#### Migration: `050_fix_rbac_tables.sql`

```sql
-- Migration: 050_fix_rbac_tables.sql
-- Description: إصلاح نظام RBAC بإضافة الجداول المفقودة

BEGIN;

-- ================================================================
-- 1. إنشاء جدول role_permissions (many-to-many)
-- ================================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

-- Index للأداء
CREATE INDEX IF NOT EXISTS idx_role_permissions_role 
    ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission 
    ON role_permissions(permission_id);

COMMENT ON TABLE role_permissions IS 
    'جدول ربط الأدوار بالصلاحيات (many-to-many)';

-- ================================================================
-- 2. إنشاء جدول user_roles (many-to-many)
-- ================================================================
CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (user_id, role_id)
);

-- Indexes للأداء
CREATE INDEX IF NOT EXISTS idx_user_roles_user 
    ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role 
    ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_assigned_by 
    ON user_roles(assigned_by);

COMMENT ON TABLE user_roles IS 
    'جدول ربط المستخدمين بالأدوار (many-to-many)';

-- ================================================================
-- 3. التحقق من البيانات
-- ================================================================
-- عرض عدد الأدوار والصلاحيات الموجودة
DO $$
DECLARE
    role_count INT;
    permission_count INT;
BEGIN
    SELECT COUNT(*) INTO role_count FROM roles;
    SELECT COUNT(*) INTO permission_count FROM permissions;
    
    RAISE NOTICE 'عدد الأدوار الموجودة: %', role_count;
    RAISE NOTICE 'عدد الصلاحيات الموجودة: %', permission_count;
END $$;

COMMIT;
```

### خطوات التنفيذ
1. ✅ إنشاء ملف migration
2. ✅ مراجعة الكود
3. ⏳ نسخ احتياطي من قاعدة البيانات
4. ⏳ تطبيق migration في بيئة التطوير
5. ⏳ اختبار نظام RBAC
6. ⏳ تطبيق في الإنتاج

### الاختبار
```sql
-- التحقق من إنشاء الجداول
SELECT table_name 
FROM information_schema.tables 
WHERE table_name IN ('role_permissions', 'user_roles');

-- التحقق من Foreign Keys
SELECT 
    tc.table_name, 
    kcu.column_name, 
    ccu.table_name AS foreign_table_name
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY' 
    AND tc.table_name IN ('role_permissions', 'user_roles');
```

---

## 1.2 تأمين Redis

### المشكلة
- `REDIS_PASSWORD` فارغ في `.env` (خطر أمني حرج)
- Redis يستمع على جميع الواجهات (0.0.0.0) افتراضياً
- لا يوجد ملف `redis.conf` مخصص

### التأثير
- أي شخص يمكنه الوصول إلى Redis بدون مصادقة
- خطر تسريب البيانات الحساسة (sessions, cache)
- إمكانية تنفيذ أوامر خطيرة (FLUSHALL, CONFIG)

### الحل المقترح

#### 1. إنشاء ملف `redis.conf`

```conf
# Redis Configuration for Al-Hudhud SMS Platform
# Security and Performance Optimizations

# ================================================================
# NETWORK
# ================================================================
# Bind to localhost only (not accessible from outside)
bind 127.0.0.1

# Port
port 6379

# TCP listen() backlog
tcp-backlog 511

# Close connection after client idle for N seconds (0 = disabled)
timeout 300

# TCP keepalive
tcp-keepalive 300

# ================================================================
# SECURITY
# ================================================================
# Require password for all connections
requirepass YOUR_STRONG_PASSWORD_HERE

# Disable dangerous commands
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command CONFIG ""
rename-command SHUTDOWN ""

# ================================================================
# MEMORY MANAGEMENT
# ================================================================
# Max memory (adjust based on your server)
maxmemory 256mb

# Eviction policy when maxmemory is reached
maxmemory-policy allkeys-lru

# ================================================================
# LAZY FREEING
# ================================================================
lazyfree-lazy-eviction yes
lazyfree-lazy-expire yes
lazyfree-lazy-server-del yes
replica-lazy-flush yes

# ================================================================
# PERSISTENCE
# ================================================================
# AOF (Append Only File) for better durability
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec

# RDB Snapshots (backup)
save 900 1
save 300 10
save 60 10000

# ================================================================
# LOGGING
# ================================================================
loglevel notice
logfile "/var/log/redis/redis-server.log"

# ================================================================
# PERFORMANCE
# ================================================================
# Number of databases
databases 16

# Max number of clients
maxclients 10000
```

#### 2. تحديث `.env`

```env
# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=YOUR_STRONG_PASSWORD_HERE
REDIS_DB=0
```

#### 3. توليد كلمة مرور قوية

```bash
# استخدم هذا الأمر لتوليد كلمة مرور قوية
openssl rand -base64 32
```

### خطوات التنفيذ
1. ✅ إنشاء ملف `redis.conf`
2. ⏳ توليد كلمة مرور قوية
3. ⏳ تحديث `.env` بكلمة المرور
4. ⏳ إعادة تشغيل Redis بالتكوين الجديد
5. ⏳ اختبار الاتصال
6. ⏳ تحديث الكود إذا لزم الأمر

### الاختبار
```bash
# اختبار الاتصال بدون كلمة مرور (يجب أن يفشل)
redis-cli ping
# Expected: (error) NOAUTH Authentication required.

# اختبار الاتصال مع كلمة المرور (يجب أن ينجح)
redis-cli -a YOUR_PASSWORD ping
# Expected: PONG

# التحقق من bind address
redis-cli -a YOUR_PASSWORD CONFIG GET bind
# Expected: 127.0.0.1

# التحقق من الأوامر المعطلة
redis-cli -a YOUR_PASSWORD FLUSHALL
# Expected: (error) ERR unknown command
```

---

## 1.3 تفعيل Foreign Keys في SQLite

### المشكلة
- Foreign Keys معطلة افتراضياً في SQLite
- لا يوجد تحقق من سلامة العلاقات بين الجداول
- يمكن حذف بيانات مرتبطة دون تحذير

### التأثير
- بيانات يتيمة (orphaned data)
- عدم تناسق البيانات
- صعوبة في الصيانة

### الحل المقترح

#### 1. تحديث `DatabaseService.ts`

```typescript
import { open } from 'react-native-quick-sqlite';
import { runMigrations } from './migrations';

const db = open({ name: 'alhudhudai.db' });

// تفعيل Foreign Keys فوراً بعد فتح الاتصال
db.execute('PRAGMA foreign_keys = ON');

export const initDatabase = () => {
    // التحقق من تفعيل Foreign Keys
    const fkCheck = db.execute('PRAGMA foreign_keys');
    console.log('Foreign Keys enabled:', fkCheck.rows?._array[0]?.foreign_keys === 1);
    
    // Run migrations
    runMigrations();
    
    // ... باقي الكود
};
```

#### 2. إضافة migration جديد

```typescript
// في ملف migrations.ts
{
    version: 2,
    name: 'add_foreign_keys_and_indexes',
    up: () => {
        console.log('Adding Foreign Keys and Indexes...');
        
        // إعادة إنشاء جدول messages مع Foreign Keys
        db.execute(`
            CREATE TABLE IF NOT EXISTS messages_new (
                id TEXT PRIMARY KEY,
                tenant_id INTEGER NOT NULL,
                phone_number TEXT NOT NULL,
                content TEXT NOT NULL,
                status TEXT CHECK(status IN ('PENDING', 'SENT', 'DELIVERED', 'FAILED', 'SKIPPED')) DEFAULT 'PENDING',
                failure_reason TEXT,
                attempts INTEGER DEFAULT 0 CHECK(attempts >= 0),
                server_status_synced INTEGER DEFAULT 0,
                buffered_ack_sent INTEGER DEFAULT 0,
                idempotency_key TEXT UNIQUE,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                sent_at DATETIME,
                delivered_at DATETIME,
                UNIQUE(id, tenant_id)
            )
        `);
        
        // نسخ البيانات
        db.execute(`
            INSERT INTO messages_new 
            SELECT * FROM messages
        `);
        
        // حذف الجدول القديم
        db.execute('DROP TABLE messages');
        
        // إعادة تسمية الجدول الجديد
        db.execute('ALTER TABLE messages_new RENAME TO messages');
        
        // إعادة إنشاء الـ indexes
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_messages_tenant 
            ON messages(tenant_id)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_messages_status 
            ON messages(status)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_messages_tenant_status 
            ON messages(tenant_id, status)
        `);
        
        // إضافة indexes على جدول logs
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_message_id 
            ON logs(message_id)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_timestamp 
            ON logs(timestamp)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_event 
            ON logs(event)
        `);
        
        console.log('Foreign Keys and Indexes added successfully');
    },
}
```

### خطوات التنفيذ
1. ✅ تحديث `DatabaseService.ts`
2. ✅ إضافة migration جديد
3. ⏳ اختبار في بيئة التطوير
4. ⏳ التحقق من عمل Foreign Keys
5. ⏳ نشر التحديث

### الاختبار
```typescript
// في الكود
const testForeignKeys = () => {
    // التحقق من تفعيل Foreign Keys
    const result = db.execute('PRAGMA foreign_keys');
    console.log('FK Status:', result.rows?._array[0]);
    
    // التحقق من وجود الـ indexes
    const indexes = db.execute('PRAGMA index_list(logs)');
    console.log('Logs Indexes:', indexes.rows?._array);
};
```

---

## ملخص المرحلة 1

| الإصلاح | الحالة | الأولوية | المدة |
|---------|--------|----------|-------|
| 1.1 - RBAC Tables | ⏳ | حرجة | 1 يوم |
| 1.2 - Redis Security | ⏳ | حرجة | 1 يوم |
| 1.3 - SQLite Foreign Keys | ⏳ | حرجة | 1 يوم |

**المدة الإجمالية**: 2-3 أيام

---

## الخطوة التالية
بعد إكمال المرحلة 1، الانتقال إلى **المرحلة 2: الإصلاحات عالية الأولوية**
