# سكريبتات إصلاح قواعد البيانات

هذا المجلد يحتوي على السكريبتات المساعدة لتطبيق والتحقق من إصلاحات قواعد البيانات.

---

## 📁 الملفات

### 1. `setup_redis_security.sh`
**الغرض**: تأمين Redis بشكل تلقائي

**الوظائف**:
- توليد كلمة مرور قوية (32 حرف)
- تحديث `redis.conf` بكلمة المرور
- تحديث `.env` بكلمة المرور
- إنشاء المجلدات المطلوبة
- عرض تعليمات الخطوات التالية

**الاستخدام**:
```bash
cd services/backend/scripts
chmod +x setup_redis_security.sh
./setup_redis_security.sh
```

**المتطلبات**:
- `openssl` مثبت
- صلاحيات sudo (لإنشاء المجلدات)

---

### 2. `verify_redis.sh`
**الغرض**: التحقق من أمان Redis

**الوظائف**:
- اختبار حماية كلمة المرور
- التحقق من bind address
- التحقق من تعطيل الأوامر الخطيرة
- فحص إعدادات الذاكرة
- فحص AOF persistence
- عرض تقرير شامل

**الاستخدام**:
```bash
cd services/backend/scripts
chmod +x verify_redis.sh
./verify_redis.sh
```

**المتطلبات**:
- Redis يعمل
- ملف `.env` موجود مع `REDIS_PASSWORD`

**النتيجة المتوقعة**:
```
Score: 8/8 (100%)
✓ Redis Security: EXCELLENT
```

---

### 3. `verify_phase1.sql`
**الغرض**: التحقق من إصلاحات PostgreSQL (المرحلة 1)

**الوظائف**:
- التحقق من وجود جداول RBAC
- التحقق من Foreign Keys
- التحقق من Indexes
- عد السجلات
- التحقق من التعليقات
- اختبار العلاقات
- عرض ملخص شامل

**الاستخدام**:
```bash
cd services/backend/scripts
psql -U postgres -d al_hudhud_new -f verify_phase1.sql
```

**المتطلبات**:
- PostgreSQL يعمل
- قاعدة البيانات `al_hudhud_new` موجودة
- migration 050 مطبق

**النتيجة المتوقعة**:
```
✓ Phase 1 PostgreSQL: SUCCESS
```

---

## 🚀 الترتيب الموصى به

### 1. تأمين Redis
```bash
# تشغيل سكريبت التأمين
./setup_redis_security.sh

# إعادة تشغيل Redis
sudo systemctl restart redis

# التحقق من النجاح
./verify_redis.sh
```

### 2. إصلاح PostgreSQL
```bash
# تطبيق migration
cd ..
psql -U postgres -d al_hudhud_new -f migrations/050_fix_rbac_tables.sql

# التحقق من النجاح
cd scripts
psql -U postgres -d al_hudhud_new -f verify_phase1.sql
```

### 3. إصلاح SQLite
راجع: `apps/mobile_gateway/SQLITE_FIXES.md`

---

## 🔧 استكشاف الأخطاء

### مشكلة: `permission denied`
```bash
chmod +x *.sh
```

### مشكلة: `openssl: command not found`
```bash
# Ubuntu/Debian
sudo apt-get install openssl

# macOS
brew install openssl

# Windows (Git Bash)
# openssl مثبت مع Git
```

### مشكلة: `redis-cli: command not found`
```bash
# Ubuntu/Debian
sudo apt-get install redis-tools

# macOS
brew install redis

# Windows
# استخدم Redis for Windows
```

### مشكلة: `psql: command not found`
```bash
# Ubuntu/Debian
sudo apt-get install postgresql-client

# macOS
brew install postgresql

# Windows
# استخدم PostgreSQL installer
```

---

## 📊 معايير النجاح

### Redis
- ✅ Score: 8/8 (100%)
- ✅ Password protection working
- ✅ Bind to 127.0.0.1
- ✅ Dangerous commands disabled
- ✅ AOF enabled

### PostgreSQL
- ✅ RBAC tables exist
- ✅ Foreign keys: 4+ found
- ✅ Indexes: 6+ found
- ✅ Phase 1 PostgreSQL: SUCCESS

---

## 🆘 الدعم

إذا واجهت مشاكل:
1. تحقق من logs: `/var/log/redis/redis-server.log`
2. تحقق من PostgreSQL logs
3. راجع `QUICK_START_GUIDE.md`
4. راجع `PHASE_1_CRITICAL_FIXES.md`

---

## 📝 ملاحظات

- جميع السكريبتات آمنة للتشغيل المتكرر
- النسخ الاحتياطية تُنشأ تلقائياً
- السكريبتات تعرض رسائل واضحة
- يمكن التراجع عن التغييرات

**تاريخ الإنشاء**: 22 مارس 2026  
**الإصدار**: 1.0
