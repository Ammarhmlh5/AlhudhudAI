# دليل التنفيذ اليدوي - المرحلة الأولى
## إصلاحات قواعد البيانات الحرجة

**التاريخ**: 22 مارس 2026  
**الحالة**: جاهز للتنفيذ اليدوي

---

## ⚠️ قبل البدء

### 1. تأكد من تشغيل الخدمات
```powershell
# تحقق من PostgreSQL
Get-Service -Name postgresql*

# تحقق من Redis
Get-Service -Name redis*
```

### 2. حدد مسار PostgreSQL
```powershell
# ابحث عن psql.exe
Get-ChildItem "C:\Program Files\PostgreSQL" -Recurse -Filter "psql.exe"
```

**المسارات الشائعة**:
- `C:\Program Files\PostgreSQL\16\bin\psql.exe`
- `C:\Program Files\PostgreSQL\15\bin\psql.exe`
- `C:\Program Files\PostgreSQL\14\bin\psql.exe`

---

## 📋 الخطوة 1: نسخ احتياطي من PostgreSQL

### الطريقة 1: باستخدام pg_dump
```powershell
cd "D:\ALHudhudAI\services\backend"

# استبدل المسار بمسار PostgreSQL لديك
$env:PGPASSWORD = "Ammar1983hmlh##"
& "C:\Program Files\PostgreSQL\16\bin\pg_dump.exe" `
    -h localhost `
    -p 5433 `
    -U postgres `
    -d al_hudhud_new `
    -f "backup_$(Get-Date -Format 'yyyyMMdd_HHmmss').sql"
```

### الطريقة 2: باستخدام pgAdmin
1. افتح pgAdmin
2. انقر بزر الماوس الأيمن على قاعدة البيانات `al_hudhud_new`
3. اختر **Backup...**
4. احفظ الملف في مكان آمن

---

## 🔧 الخطوة 2: تطبيق Migration PostgreSQL

### الطريقة 1: من سطر الأوامر
```powershell
cd "D:\ALHudhudAI\services\backend"

# تعيين كلمة المرور
$env:PGPASSWORD = "Ammar1983hmlh##"

# تطبيق migration
& "C:\Program Files\PostgreSQL\16\bin\psql.exe" `
    -h localhost `
    -p 5433 `
    -U postgres `
    -d al_hudhud_new `
    -f "migrations\050_fix_rbac_tables.sql"

# مسح كلمة المرور
$env:PGPASSWORD = $null
```

### الطريقة 2: باستخدام pgAdmin
1. افتح pgAdmin
2. اتصل بقاعدة البيانات `al_hudhud_new`
3. افتح **Query Tool** (أيقونة SQL)
4. افتح الملف: `migrations/050_fix_rbac_tables.sql`
5. اضغط **Execute** (F5)

### الطريقة 3: نسخ ولصق
1. افتح ملف `migrations/050_fix_rbac_tables.sql`
2. انسخ المحتوى بالكامل
3. افتح pgAdmin Query Tool
4. الصق المحتوى
5. اضغط Execute

---

## ✅ الخطوة 3: التحقق من النجاح

### من سطر الأوامر
```powershell
$env:PGPASSWORD = "Ammar1983hmlh##"

& "C:\Program Files\PostgreSQL\16\bin\psql.exe" `
    -h localhost `
    -p 5433 `
    -U postgres `
    -d al_hudhud_new `
    -c "SELECT table_name FROM information_schema.tables WHERE table_name IN ('role_permissions', 'user_roles');"

$env:PGPASSWORD = $null
```

**النتيجة المتوقعة**:
```
     table_name      
---------------------
 role_permissions
 user_roles
(2 rows)
```

### من pgAdmin
```sql
-- تشغيل هذا الاستعلام
SELECT table_name 
FROM information_schema.tables 
WHERE table_name IN ('role_permissions', 'user_roles');
```

---

## 🔴 الخطوة 4: تأمين Redis

### 4.1 توليد كلمة مرور قوية

#### الطريقة 1: PowerShell
```powershell
# توليد كلمة مرور عشوائية
$bytes = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
$password = [Convert]::ToBase64String($bytes)
Write-Host "Generated Password: $password"
```

#### الطريقة 2: موقع ويب
استخدم: https://www.random.org/passwords/

**الإعدادات**:
- Length: 32
- Characters: All

### 4.2 تحديث redis.conf

1. افتح ملف: `services/backend/redis.conf`
2. ابحث عن السطر:
   ```
   requirepass CHANGE_THIS_TO_STRONG_PASSWORD
   ```
3. استبدله بـ:
   ```
   requirepass YOUR_GENERATED_PASSWORD_HERE
   ```
4. احفظ الملف

### 4.3 تحديث .env

1. افتح ملف: `services/backend/.env`
2. ابحث عن السطر:
   ```
   REDIS_PASSWORD=
   ```
3. استبدله بـ:
   ```
   REDIS_PASSWORD=YOUR_GENERATED_PASSWORD_HERE
   ```
4. احفظ الملف

### 4.4 إعادة تشغيل Redis

#### إذا كان Redis خدمة Windows
```powershell
Restart-Service redis
```

#### إذا كان Redis يعمل يدوياً
1. أوقف Redis الحالي (Ctrl+C)
2. شغّل Redis بالتكوين الجديد:
```powershell
cd "D:\ALHudhudAI\services\backend"
redis-server redis.conf
```

### 4.5 اختبار Redis

```powershell
# اختبار بدون كلمة مرور (يجب أن يفشل)
redis-cli ping
# Expected: (error) NOAUTH Authentication required.

# اختبار مع كلمة المرور (يجب أن ينجح)
redis-cli -a YOUR_PASSWORD ping
# Expected: PONG
```

---

## 📱 الخطوة 5: إصلاح SQLite (Mobile)

### 5.1 تحديث DatabaseService.ts

افتح: `apps/mobile_gateway/src/services/DatabaseService.ts`

**أضف بعد السطر**:
```typescript
const db = open({ name: 'alhudhudai.db' });
```

**هذا الكود**:
```typescript
// تفعيل Foreign Keys
db.execute('PRAGMA foreign_keys = ON');

// تفعيل Write-Ahead Logging
db.execute('PRAGMA journal_mode = WAL');

// تحسين الأداء
db.execute('PRAGMA synchronous = NORMAL');
db.execute('PRAGMA temp_store = MEMORY');
```

### 5.2 إضافة Migration جديد

افتح: `apps/mobile_gateway/src/services/migrations.ts`

**أضف في نهاية مصفوفة migrations**:
```typescript
{
    version: 2,
    name: 'add_foreign_keys_indexes_and_constraints',
    up: () => {
        console.log('Migration 2: Adding Foreign Keys, Indexes, and Constraints...');
        
        // إضافة indexes على جدول logs
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_message_id 
            ON logs(message_id)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_timestamp 
            ON logs(timestamp DESC)
        `);
        
        db.execute(`
            CREATE INDEX IF NOT EXISTS idx_logs_event 
            ON logs(event)
        `);
        
        console.log('✓ Migration 2 completed successfully');
    },
}
```

### 5.3 اختبار التطبيق

1. أعد بناء التطبيق
2. شغّل التطبيق على جهاز تطوير
3. تحقق من logs للتأكد من تطبيق migration

---

## 📊 التحقق النهائي

### PostgreSQL
```sql
-- عدد الجداول الجديدة
SELECT COUNT(*) FROM information_schema.tables 
WHERE table_name IN ('role_permissions', 'user_roles');
-- Expected: 2

-- عدد Foreign Keys
SELECT COUNT(*) FROM information_schema.table_constraints
WHERE constraint_type = 'FOREIGN KEY'
    AND table_name IN ('role_permissions', 'user_roles');
-- Expected: 4+

-- عدد Indexes
SELECT COUNT(*) FROM pg_indexes
WHERE tablename IN ('role_permissions', 'user_roles');
-- Expected: 6+
```

### Redis
```powershell
# التحقق من كلمة المرور
redis-cli -a YOUR_PASSWORD CONFIG GET requirepass
# Expected: requirepass YOUR_PASSWORD

# التحقق من bind
redis-cli -a YOUR_PASSWORD CONFIG GET bind
# Expected: bind 127.0.0.1
```

---

## 🎉 الانتهاء

بعد إكمال جميع الخطوات:

✅ PostgreSQL: جداول RBAC مُنشأة  
✅ Redis: مؤمّن بكلمة مرور  
✅ SQLite: Foreign Keys مفعّلة و Indexes مضافة

**التقييم المتوقع**: من 78/100 إلى 89/100

---

## 🆘 استكشاف الأخطاء

### مشكلة: "psql: command not found"
**الحل**: أضف PostgreSQL إلى PATH أو استخدم المسار الكامل

### مشكلة: "FATAL: password authentication failed"
**الحل**: تحقق من كلمة المرور في `.env`

### مشكلة: "relation already exists"
**الحل**: الجداول موجودة بالفعل - تخطى هذه الخطوة

### مشكلة: "redis-cli: command not found"
**الحل**: ثبّت Redis أو أضفه إلى PATH

---

## 📞 الدعم

إذا واجهت مشاكل:
1. راجع `QUICK_START_GUIDE.md`
2. راجع `PHASE_1_CRITICAL_FIXES.md`
3. تحقق من logs قاعدة البيانات

**حظاً موفقاً! 🚀**
