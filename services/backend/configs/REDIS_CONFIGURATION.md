# Redis Configuration Documentation

## نظرة عامة

هذا المستند يوثق إعدادات Redis المستخدمة في نظام Al-Hudhud SMS Gateway، بما في ذلك هياكل البيانات، سياسات TTL، والإعدادات المتعلقة بالأداء والموثوقية.

**المتطلبات المرتبطة:** 3.2, 5.2

## هياكل البيانات المستخدمة (Data Structures)

### 1. Message Buffer (Lists)

**الغرض:** تخزين مؤقت سريع للرسائل الواردة قبل معالجتها

**الهيكل:**
```
Key Pattern: messages:pending:{tenant_id}
Type: List (FIFO Queue)
Operations: LPUSH (إضافة), RPOP (استخراج)
```

**مثال:**
```redis
# إضافة رسالة جديدة
LPUSH messages:pending:1 '{"message_id":"uuid","tenant_id":1,"phone":"966500000000","content":"test"}'

# استخراج دفعة من الرسائل
RPOP messages:pending:1 100
```

**TTL Policy:** 24 ساعة (تنظيف تلقائي للرسائل القديمة)

**الحجم المتوقع:** 
- في الأوقات العادية: 10-100 رسالة لكل tenant
- في أوقات الذروة: 1000-5000 رسالة لكل tenant
- الحد الأقصى: 10,000 رسالة لكل tenant

---

### 2. Active Tenants Set (Sets)

**الغرض:** تتبع الشركات التي لديها رسائل معلقة

**الهيكل:**
```
Key: tenants:active
Type: Set
Operations: SADD (إضافة), SREM (حذف), SMEMBERS (قراءة الكل)
```

**مثال:**
```redis
# إضافة tenant نشط
SADD tenants:active 1

# الحصول على جميع tenants النشطة
SMEMBERS tenants:active

# حذف tenant بعد معالجة جميع رسائله
SREM tenants:active 1
```

**TTL Policy:** لا يوجد (يُدار يدوياً من قبل Batch Processor)

---

### 3. Buffer Size Counter (Strings)

**الغرض:** تتبع عدد الرسائل المعلقة لكل tenant

**الهيكل:**
```
Key Pattern: messages:count:{tenant_id}
Type: String (Integer)
Operations: INCR (زيادة), DECR (نقصان), GET (قراءة)
```

**مثال:**
```redis
# زيادة العداد عند إضافة رسالة
INCR messages:count:1

# نقصان العداد عند معالجة رسالة
DECR messages:count:1

# قراءة العدد الحالي
GET messages:count:1
```

**TTL Policy:** 1 ساعة (يُعاد إنشاؤه تلقائياً عند الحاجة)

---

### 4. Rate Limiting Counter (Strings)

**الغرض:** تطبيق حد الطلبات لكل tenant (100 طلب/دقيقة)

**الهيكل:**
```
Key Pattern: rate_limit:{tenant_id}
Type: String (Integer)
Operations: INCR, EXPIRE
```

**مثال:**
```redis
# زيادة العداد وتعيين TTL
INCR rate_limit:1
EXPIRE rate_limit:1 60

# التحقق من الحد
GET rate_limit:1
```

**TTL Policy:** 60 ثانية (نافذة متحركة)

**الحد الأقصى:** 100 طلب/دقيقة لكل tenant

---

### 5. Channel Cache (Hashes)

**الغرض:** تخزين معلومات القنوات المتاحة لتقليل Database Queries

**الهيكل:**
```
Key Pattern: channels:cache:{tenant_id}
Type: Hash
Fields: channel_id → JSON (channel info)
Operations: HSET, HGETALL, DEL
```

**مثال:**
```redis
# تخزين معلومات القنوات
HSET channels:cache:1 "101" '{"id":101,"status":"ONLINE","priority":10,"capacity":100}'
HSET channels:cache:1 "102" '{"id":102,"status":"ONLINE","priority":5,"capacity":50}'

# قراءة جميع القنوات
HGETALL channels:cache:1

# حذف الـ cache
DEL channels:cache:1
```

**TTL Policy:** 5 دقائق (المتطلب 5.2)

**الحجم المتوقع:** 1-10 قنوات لكل tenant

---

### 6. Idempotency Keys (Strings)

**الغرض:** ضمان عدم معالجة نفس تقرير الحالة مرتين

**الهيكل:**
```
Key Pattern: idempotency:{tenant_id}:{message_id}:{status}
Type: String
Value: "1" (علامة فقط)
Operations: SET NX (set if not exists), EXISTS
```

**مثال:**
```redis
# تعيين مفتاح idempotency
SET idempotency:1:uuid-123:SENT "1" EX 86400 NX

# التحقق من وجود المفتاح
EXISTS idempotency:1:uuid-123:SENT
```

**TTL Policy:** 24 ساعة

---

### 7. WebSocket Connection Status (Hashes)

**الغرض:** تتبع حالة اتصالات WebSocket للأجهزة

**الهيكل:**
```
Key Pattern: ws:connections:{tenant_id}
Type: Hash
Fields: channel_id → JSON (connection info)
Operations: HSET, HDEL, HGET
```

**مثال:**
```redis
# تسجيل اتصال جديد
HSET ws:connections:1 "101" '{"connected_at":"2024-01-01T10:00:00Z","last_heartbeat":"2024-01-01T10:05:00Z"}'

# حذف اتصال منقطع
HDEL ws:connections:1 "101"

# قراءة حالة اتصال
HGET ws:connections:1 "101"
```

**TTL Policy:** 1 ساعة (يُحدث عند كل heartbeat)

---

## سياسات TTL (Time-To-Live Policies)

### ملخص سياسات TTL

| Data Structure | Key Pattern | TTL | السبب |
|---------------|-------------|-----|-------|
| Message Buffer | `messages:pending:{tenant_id}` | 24 ساعة | تنظيف الرسائل القديمة التي لم تُعالج |
| Active Tenants | `tenants:active` | لا يوجد | يُدار يدوياً |
| Buffer Counter | `messages:count:{tenant_id}` | 1 ساعة | يُعاد إنشاؤه عند الحاجة |
| Rate Limiting | `rate_limit:{tenant_id}` | 60 ثانية | نافذة متحركة للحد من الطلبات |
| Channel Cache | `channels:cache:{tenant_id}` | 5 دقائق | تحديث دوري لمعلومات القنوات |
| Idempotency | `idempotency:{tenant_id}:{message_id}:{status}` | 24 ساعة | منع معالجة مكررة |
| WS Connections | `ws:connections:{tenant_id}` | 1 ساعة | تنظيف الاتصالات القديمة |

### آلية تطبيق TTL

#### 1. TTL تلقائي عند الإنشاء
```go
// مثال: Rate Limiting
redis.Incr(ctx, fmt.Sprintf("rate_limit:%d", tenantID))
redis.Expire(ctx, fmt.Sprintf("rate_limit:%d", tenantID), 60*time.Second)
```

#### 2. TTL عند التحديث
```go
// مثال: Channel Cache
redis.HSet(ctx, fmt.Sprintf("channels:cache:%d", tenantID), channelID, channelJSON)
redis.Expire(ctx, fmt.Sprintf("channels:cache:%d", tenantID), 5*time.Minute)
```

#### 3. تنظيف دوري (Passive Expiration)
Redis يحذف المفاتيح المنتهية تلقائياً عند الوصول إليها أو عبر عملية تنظيف دورية في الخلفية.

---

## إعدادات Persistence

### 1. RDB (Redis Database) Snapshots

**الغرض:** نسخ احتياطية دورية للبيانات

**الإعدادات:**
```conf
# حفظ snapshot كل 15 دقيقة إذا تغير 1 مفتاح على الأقل
save 900 1

# حفظ snapshot كل 5 دقائق إذا تغير 10 مفاتيح على الأقل
save 300 10

# حفظ snapshot كل دقيقة إذا تغير 10,000 مفتاح على الأقل
save 60 10000

# إعدادات إضافية
rdbcompression yes      # ضغط الملف لتوفير المساحة
rdbchecksum yes         # التحقق من سلامة البيانات
dbfilename dump.rdb     # اسم الملف
dir /data               # مسار الحفظ
```

**الفوائد:**
- نسخ احتياطية سريعة وصغيرة الحجم
- استرداد سريع عند إعادة التشغيل
- مناسب للبيانات التي يمكن تحمل فقدان بعضها (cache)

**العيوب:**
- قد يُفقد بعض البيانات بين snapshots
- يستهلك CPU عند الحفظ

---

### 2. AOF (Append Only File)

**الغرض:** تسجيل جميع عمليات الكتابة لضمان عدم فقدان البيانات

**الإعدادات:**
```conf
appendonly yes                      # تفعيل AOF
appendfilename "appendonly.aof"     # اسم الملف
appendfsync everysec                # مزامنة كل ثانية (توازن بين الأداء والأمان)
aof-use-rdb-preamble yes            # استخدام RDB كمقدمة لتسريع التحميل
auto-aof-rewrite-percentage 100     # إعادة كتابة عند مضاعفة الحجم
auto-aof-rewrite-min-size 64mb      # الحد الأدنى للحجم قبل إعادة الكتابة
```

**خيارات appendfsync:**
- `always`: مزامنة بعد كل عملية كتابة (أبطأ، أكثر أماناً)
- `everysec`: مزامنة كل ثانية (توازن - **مستخدم حالياً**)
- `no`: يترك المزامنة للنظام (أسرع، أقل أماناً)

**الفوائد:**
- حماية قصوى من فقدان البيانات (فقدان ثانية واحدة كحد أقصى)
- يمكن إعادة بناء البيانات بالكامل من AOF
- مناسب للبيانات الحرجة (message buffer, idempotency)

**العيوب:**
- ملف أكبر من RDB
- أبطأ قليلاً في الاسترداد

---

### 3. استراتيجية Hybrid (RDB + AOF)

**الإعداد الحالي:** نستخدم كلا الطريقتين معاً للحصول على أفضل النتيجتين:

```conf
# RDB للنسخ الاحتياطية السريعة
save 900 1
save 300 10
save 60 10000

# AOF لحماية البيانات
appendonly yes
appendfsync everysec
aof-use-rdb-preamble yes
```

**آلية الاسترداد:**
1. عند إعادة التشغيل، Redis يبحث عن AOF أولاً
2. إذا وُجد AOF، يُستخدم لاسترداد البيانات (أكثر دقة)
3. إذا لم يوجد AOF، يُستخدم RDB snapshot
4. مع `aof-use-rdb-preamble yes`، يبدأ AOF بـ RDB snapshot ثم يضيف العمليات الجديدة

**النتيجة:**
- ✅ حماية قصوى من فقدان البيانات (فقدان ثانية واحدة كحد أقصى)
- ✅ استرداد سريع (RDB preamble)
- ✅ ملفات أصغر (إعادة كتابة AOF تلقائياً)

---

## إعدادات Memory Management

### 1. Memory Limit

**الإعداد:**
```conf
maxmemory 2gb
```

**السبب:**
- يمنع استهلاك Redis لكل ذاكرة السيرفر
- يضمن توفر ذاكرة للخدمات الأخرى (PostgreSQL, Backend)
- مناسب لحجم البيانات المتوقع:
  - Message Buffer: ~500MB في الذروة
  - Caches: ~200MB
  - Counters & Keys: ~100MB
  - احتياطي: ~1.2GB

---

### 2. Eviction Policy

**الإعداد:**
```conf
maxmemory-policy allkeys-lru
```

**الخيارات المتاحة:**
- `noeviction`: رفض عمليات الكتابة عند امتلاء الذاكرة (غير مناسب)
- `allkeys-lru`: حذف أقل المفاتيح استخداماً من جميع المفاتيح (**مستخدم حالياً**)
- `allkeys-lfu`: حذف أقل المفاتيح تكراراً
- `volatile-lru`: حذف أقل المفاتيح استخداماً من المفاتيح ذات TTL فقط
- `volatile-ttl`: حذف المفاتيح ذات أقصر TTL

**لماذا allkeys-lru؟**
- ✅ يعمل على جميع المفاتيح (حتى بدون TTL)
- ✅ يحذف البيانات الأقل أهمية تلقائياً (cache قديم)
- ✅ يحافظ على البيانات الحديثة والمستخدمة بكثرة
- ✅ مناسب لنظام يحتوي على cache + message buffer

**آلية العمل:**
1. عند الوصول لـ maxmemory، Redis يبدأ eviction
2. يختار عينة عشوائية من المفاتيح (maxmemory-samples = 5)
3. يحذف المفتاح الأقل استخداماً من العينة
4. يكرر العملية حتى يتوفر مساحة كافية

---

### 3. Memory Samples

**الإعداد:**
```conf
maxmemory-samples 5
```

**السبب:**
- توازن بين الدقة والأداء
- عينة أكبر = دقة أعلى في اختيار المفتاح الأقل استخداماً
- عينة أصغر = أداء أسرع
- 5 هو القيمة الافتراضية والموصى بها

---

## إعدادات الأداء (Performance)

### 1. Threaded I/O

**الإعدادات:**
```conf
io-threads 4
io-threads-do-reads yes
```

**الفوائد:**
- استخدام 4 threads لعمليات I/O
- تسريع القراءة والكتابة
- استغلال أفضل لـ multi-core CPUs
- مناسب لحمل عالي (5000+ رسالة/دقيقة)

**ملاحظة:** عدد threads يجب أن يكون أقل من عدد CPU cores

---

### 2. TCP Backlog

**الإعداد:**
```conf
tcp-backlog 511
```

**السبب:**
- يسمح بـ 511 اتصال معلق في الانتظار
- مناسب لحمل عالي مع اتصالات متعددة
- يمنع رفض الاتصالات عند الذروة

---

### 3. TCP Keep-Alive

**الإعداد:**
```conf
tcp-keepalive 300
```

**السبب:**
- إرسال TCP keep-alive كل 300 ثانية (5 دقائق)
- اكتشاف الاتصالات المنقطعة
- تنظيف الاتصالات الميتة

---

### 4. Active Rehashing

**الإعداد:**
```conf
activerehashing yes
```

**السبب:**
- إعادة توزيع hash tables تدريجياً
- تحسين استخدام الذاكرة
- تحسين أداء العمليات

---

## إعدادات المراقبة (Monitoring)

### 1. Slow Log

**الإعدادات:**
```conf
slowlog-log-slower-than 10000  # 10ms
slowlog-max-len 128
```

**الغرض:**
- تسجيل العمليات التي تستغرق أكثر من 10ms
- الاحتفاظ بآخر 128 عملية بطيئة
- مفيد لتحديد مشاكل الأداء

**الاستخدام:**
```redis
# عرض آخر 10 عمليات بطيئة
SLOWLOG GET 10

# عرض عدد العمليات البطيئة
SLOWLOG LEN

# مسح السجل
SLOWLOG RESET
```

---

### 2. Latency Monitor

**الإعداد:**
```conf
latency-monitor-threshold 100  # 100ms
```

**الغرض:**
- تسجيل الأحداث التي تسبب تأخير أكثر من 100ms
- مفيد لتشخيص مشاكل الأداء

**الاستخدام:**
```redis
# عرض آخر أحداث التأخير
LATENCY LATEST

# عرض تاريخ حدث معين
LATENCY HISTORY command

# مسح السجل
LATENCY RESET
```

---

## أفضل الممارسات (Best Practices)

### 1. استخدام Pipeline للعمليات المتعددة

```go
// ❌ سيء: عمليات منفصلة
for _, msg := range messages {
    redis.LPush(ctx, key, msg)
}

// ✅ جيد: استخدام pipeline
pipe := redis.Pipeline()
for _, msg := range messages {
    pipe.LPush(ctx, key, msg)
}
pipe.Exec(ctx)
```

**الفائدة:** تقليل round-trips بين التطبيق و Redis

---

### 2. استخدام SCAN بدلاً من KEYS

```go
// ❌ سيء: يحجب Redis
keys := redis.Keys(ctx, "messages:pending:*")

// ✅ جيد: استخدام SCAN
iter := redis.Scan(ctx, 0, "messages:pending:*", 100).Iterator()
for iter.Next(ctx) {
    key := iter.Val()
    // معالجة المفتاح
}
```

**الفائدة:** عدم حجب Redis أثناء البحث

---

### 3. تعيين TTL دائماً للمفاتيح المؤقتة

```go
// ❌ سيء: بدون TTL
redis.Set(ctx, key, value, 0)

// ✅ جيد: مع TTL
redis.Set(ctx, key, value, 5*time.Minute)
```

**الفائدة:** منع تراكم المفاتيح القديمة

---

### 4. استخدام Lua Scripts للعمليات الذرية المعقدة

```go
// مثال: خصم من counter مع حد أدنى
script := redis.NewScript(`
    local current = tonumber(redis.call('GET', KEYS[1]) or '0')
    local amount = tonumber(ARGV[1])
    local min = tonumber(ARGV[2])
    
    if current - amount >= min then
        redis.call('DECRBY', KEYS[1], amount)
        return current - amount
    else
        return -1
    end
`)

result, err := script.Run(ctx, redis, []string{key}, amount, minValue).Int()
```

**الفائدة:** ضمان atomicity للعمليات المعقدة

---

### 5. مراقبة استخدام الذاكرة

```redis
# معلومات الذاكرة
INFO memory

# إحصائيات المفاتيح
INFO keyspace

# معلومات الأداء
INFO stats
```

---

## خطة الصيانة (Maintenance Plan)

### 1. مراقبة يومية
- ✅ فحص استخدام الذاكرة: `INFO memory`
- ✅ فحص Slow Log: `SLOWLOG GET 10`
- ✅ فحص Latency: `LATENCY LATEST`
- ✅ فحص عدد المفاتيح: `INFO keyspace`

### 2. صيانة أسبوعية
- ✅ مراجعة RDB snapshots
- ✅ مراجعة AOF file size
- ✅ تنظيف المفاتيح القديمة (إن وجدت)
- ✅ مراجعة logs للأخطاء

### 3. صيانة شهرية
- ✅ تحليل أنماط الاستخدام
- ✅ تحسين TTL policies إن لزم
- ✅ مراجعة eviction statistics
- ✅ اختبار استرداد من backup

---

## الخلاصة

### ✅ الإعدادات الحالية ممتازة وتلبي المتطلبات

| المتطلب | الإعداد | الحالة |
|---------|---------|--------|
| **3.2: Message Buffer** | Lists + TTL 24h | ✅ موثق |
| **5.2: Channel Cache** | Hash + TTL 5min | ✅ موثق |
| **Persistence** | AOF + RDB | ✅ مفعّل |
| **Memory Limit** | 2GB | ✅ مناسب |
| **Eviction Policy** | allkeys-lru | ✅ مناسب |
| **Performance** | Threaded I/O | ✅ مفعّل |
| **Monitoring** | Slow Log + Latency | ✅ مفعّل |

### 📋 التوثيق المضاف

1. ✅ هياكل البيانات المستخدمة (7 أنواع)
2. ✅ سياسات TTL لكل نوع
3. ✅ إعدادات Persistence (AOF + RDB)
4. ✅ إعدادات Memory Management
5. ✅ إعدادات الأداء
6. ✅ أفضل الممارسات
7. ✅ خطة الصيانة

### 🎯 النتيجة

النظام جاهز لتحقيق:
- ⚡ أداء عالي: 5000+ رسالة/دقيقة
- 🛡️ حماية البيانات: فقدان ثانية واحدة كحد أقصى
- 💾 استخدام ذاكرة فعّال: 2GB كحد أقصى
- 🔍 مراقبة شاملة: Slow Log + Latency Monitor
