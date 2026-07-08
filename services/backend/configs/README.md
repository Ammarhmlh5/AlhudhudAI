# Configuration Files

هذا المجلد يحتوي على ملفات الإعدادات والتوثيق للخدمات المختلفة.

## الملفات المتاحة

### 1. Redis Configuration
- **ملف الإعداد:** `../../redis.conf` (في الجذر)
- **التوثيق الشامل:** [REDIS_CONFIGURATION.md](./REDIS_CONFIGURATION.md)
- **ملف الأداء:** [redis-performance.conf](./redis-performance.conf)

#### محتويات التوثيق:
- ✅ هياكل البيانات المستخدمة (7 أنواع)
- ✅ سياسات TTL لكل نوع
- ✅ إعدادات Persistence (AOF + RDB)
- ✅ إعدادات Memory Management
- ✅ إعدادات الأداء
- ✅ أفضل الممارسات
- ✅ خطة الصيانة

#### المتطلبات المحققة:
- **3.2**: Message Buffer مع Redis Lists
- **5.2**: Channel Cache مع TTL = 5 دقائق

---

## روابط سريعة

- [Redis Configuration Documentation](./REDIS_CONFIGURATION.md) - توثيق شامل لإعدادات Redis
- [Redis Performance Config](./redis-performance.conf) - إعدادات الأداء

---

## ملاحظات

جميع ملفات الإعدادات موثقة بشكل شامل ومرتبطة بالمتطلبات في spec "message-lifecycle-rebuild".
