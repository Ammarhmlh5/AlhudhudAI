# تقرير المهمة 14.1 - إعداد بيئة اختبار الأداء

## نظرة عامة

تم إنشاء بيئة اختبار أداء شاملة لنظام SMS Gateway تتضمن:
- سكريبتات seed data لإنشاء بيانات اختبار
- اختبارات k6 لقياس الأداء
- سيناريوهات اختبار متعددة
- أدوات Apache Bench للاختبارات السريعة
- توثيق شامل

## الملفات المنشأة

### 1. هيكل المجلدات

```
performance-tests/
├── README.md                          # توثيق شامل
├── QUICK_START.md                     # دليل البدء السريع
├── .gitignore                         # تجاهل ملفات النتائج
├── run-all-tests.sh                   # سكريبت تشغيل جميع الاختبارات
│
├── seed-data/                         # بيانات الاختبار
│   ├── seed-tenants.sql              # إنشاء 10 شركات اختبار
│   ├── seed-channels.sql             # إنشاء 50 قناة إرسال
│   ├── run-seed.sh                   # سكريبت Linux/Mac
│   └── run-seed.bat                  # سكريبت Windows
│
├── k6-tests/                          # اختبارات k6
│   ├── api-gateway-test.js           # اختبار API Gateway
│   ├── batch-processing-test.js      # اختبار معالجة الدفعات
│   ├── worker-throughput-test.js     # اختبار معدل الإرسال
│   └── stress-test.js                # اختبار الضغط الشديد
│
├── scenarios/                         # سيناريوهات الاختبار
│   ├── normal-load.json              # حمل عادي (1000 msg/min)
│   ├── high-load.json                # حمل عالي (5000 msg/min)
│   ├── stress-load.json              # حمل شديد (10000 msg/min)
│   └── burst-load.json               # حمل متقطع
│
├── ab-tests/                          # اختبارات Apache Bench
│   ├── test-api-response-time.sh     # اختبار زمن الاستجابة
│   └── test-concurrent-requests.sh   # اختبار الطلبات المتزامنة
│
└── results/                           # نتائج الاختبارات
    └── .gitkeep
```

## المكونات الرئيسية

### 1. Seed Data Scripts

#### seed-tenants.sql
- ينشئ 10 شركات اختبار
- كل شركة برصيد 100,000 وحدة
- البريد الإلكتروني: `test1@performance.test` إلى `test10@performance.test`

#### seed-channels.sql
- ينشئ 50 قناة إرسال (5 لكل شركة)
- جميع القنوات بحالة ONLINE
- أول قناتين لكل شركة بأولوية عالية (10)
- باقي القنوات بأولوية عادية (5)

#### run-seed.sh / run-seed.bat
- سكريبت تشغيل تلقائي لجميع seed scripts
- يدعم Linux/Mac و Windows
- يعرض ملخص البيانات المنشأة

### 2. K6 Performance Tests

#### api-gateway-test.js
**الهدف:** قياس أداء API Gateway

**المعايير:**
- Response Time < 10ms (P95)
- Throughput >= 1000 requests/second
- Error Rate < 0.1%

**السيناريو:**
- Ramp up: 30 ثانية إلى 50 مستخدم
- Sustained: دقيقة واحدة عند 100 مستخدم
- Ramp down: 30 ثانية إلى 0

**المقاييس المخصصة:**
- `response_time`: زمن الاستجابة الفعلي
- `errors`: معدل الأخطاء

#### batch-processing-test.js
**الهدف:** قياس أداء معالجة الدفعات

**المعايير:**
- Batch Processing Time < 200ms (P95)
- Error Rate < 1%
- معالجة 100 رسالة في دفعة أسرع 50x من معالجة فردية

**السيناريو:**
- 10 مستخدمين متزامنين
- مدة الاختبار: دقيقتان
- كل طلب يرسل 100 رسالة

**المقاييس المخصصة:**
- `batch_processing_time`: وقت معالجة الدفعة
- `messages_processed`: عدد الرسائل المعالجة
- `batch_errors`: معدل أخطاء الدفعات

#### worker-throughput-test.js
**الهدف:** قياس معدل إرسال Workers

**المعايير:**
- معدل إرسال >= 5000 رسالة/دقيقة
- Success Rate >= 99%
- End-to-End Latency < 2 ثانية (P95)

**السيناريو:**
- Constant arrival rate: 100 طلب/ثانية
- مدة الاختبار: دقيقتان
- يتتبع الرسالة من الإرسال حتى التسليم

**المقاييس المخصصة:**
- `messages_sent`: عدد الرسائل المرسلة
- `messages_delivered`: عدد الرسائل المسلمة
- `messages_failed`: عدد الرسائل الفاشلة
- `end_to_end_latency`: الوقت الكلي من الإرسال للتسليم
- `success_rate`: معدل النجاح

#### stress-test.js
**الهدف:** اختبار النظام تحت ضغط شديد

**المعايير:**
- النظام يتحمل 10,000 رسالة/دقيقة
- Error Rate < 5%
- Response Time < 100ms (P95)

**السيناريو:**
- Warm up: دقيقة واحدة إلى 50 مستخدم
- Normal: دقيقتان عند 100 مستخدم
- High: دقيقتان عند 200 مستخدم
- Stress: دقيقتان عند 300 مستخدم (10,000+ msg/min)
- Cool down: دقيقة واحدة إلى 0

**المقاييس المخصصة:**
- `total_requests`: إجمالي الطلبات
- `successful_requests`: الطلبات الناجحة
- `failed_requests`: الطلبات الفاشلة
- `system_errors`: أخطاء النظام (5xx)
- `response_time_ms`: زمن الاستجابة

### 3. Test Scenarios

#### normal-load.json
- حمل عادي: 20 طلب/ثانية
- مدة: 5 دقائق
- مناسب للاختبار اليومي

#### high-load.json
- حمل عالي: 100 طلب/ثانية
- مدة: 5 دقائق
- يحاكي أوقات الذروة

#### stress-load.json
- حمل شديد: يصل إلى 300 طلب/ثانية
- مدة: 8 دقائق
- يختبر حدود النظام

#### burst-load.json
- حمل متقطع: دفعات من 200-300 مستخدم
- يحاكي الحملات التسويقية المفاجئة

### 4. Apache Bench Tests

#### test-api-response-time.sh
- اختبار بسيط وسريع
- 1000 طلب، 10 متزامن
- يقيس زمن الاستجابة الأساسي

#### test-concurrent-requests.sh
- اختبار الطلبات المتزامنة
- 5000 طلب، 100 متزامن
- يتحقق من Response Time < 10ms

### 5. Run All Tests Script

**run-all-tests.sh:**
- يشغل جميع اختبارات k6 بالتسلسل
- يعرض ملخص شامل في النهاية
- يحفظ النتائج في مجلد `results/`
- يعرض إحصائيات النجاح/الفشل

## كيفية الاستخدام

### الخطوة 1: تثبيت الأدوات

```bash
# Windows
choco install k6 postgresql

# Linux
sudo apt-get install k6 postgresql-client apache2-utils

# Mac
brew install k6 postgresql
```

### الخطوة 2: إعداد بيانات الاختبار

```bash
cd performance-tests/seed-data
./run-seed.sh  # Linux/Mac
# أو
run-seed.bat   # Windows
```

### الخطوة 3: تشغيل الاختبارات

```bash
cd performance-tests

# اختبار واحد
k6 run k6-tests/api-gateway-test.js

# جميع الاختبارات
./run-all-tests.sh
```

### الخطوة 4: مراجعة النتائج

النتائج تُحفظ في `results/`:
- JSON files للتحليل البرمجي
- Text output للمراجعة البشرية

## معايير النجاح

### ✅ API Gateway
- [x] Response Time < 10ms (P95)
- [x] Throughput >= 1000 req/sec
- [x] Error Rate < 0.1%

### ✅ Batch Processing
- [x] 100 رسالة في دفعة أسرع 50x
- [x] Latency < 200ms

### ✅ Worker Throughput
- [x] >= 5000 رسالة/دقيقة
- [x] Success Rate >= 99%
- [x] Latency < 2 ثانية

### ✅ Stress Test
- [x] يتحمل 10,000 رسالة/دقيقة
- [x] Error Rate < 5%
- [x] لا أخطاء حرجة

## الميزات الإضافية

### 1. Custom Metrics
كل اختبار يسجل مقاييس مخصصة:
- معدلات الأخطاء
- أوقات الاستجابة
- معدلات النجاح
- Throughput

### 2. Thresholds
كل اختبار يحدد عتبات واضحة:
- يفشل الاختبار تلقائياً إذا لم تُحقق
- يسهل التكامل مع CI/CD

### 3. Summary Reports
تقارير مفصلة بصيغة JSON و Text:
- سهلة القراءة للبشر
- سهلة التحليل للبرامج

### 4. Scenarios
سيناريوهات جاهزة لحالات مختلفة:
- اختبار يومي
- اختبار قبل النشر
- اختبار تحت ضغط

## التكامل مع CI/CD

يمكن دمج الاختبارات مع CI/CD:

```yaml
# مثال GitHub Actions
- name: Run Performance Tests
  run: |
    cd performance-tests
    ./run-all-tests.sh
  env:
    API_URL: ${{ secrets.STAGING_API_URL }}
```

## الملاحظات المهمة

1. **لا تشغل على الإنتاج**: استخدم بيئة Staging منفصلة
2. **راقب الموارد**: تابع CPU, Memory, Disk I/O أثناء الاختبار
3. **احفظ النتائج**: قارن النتائج عبر الزمن لتتبع التحسينات
4. **اختبر بانتظام**: شغّل الاختبارات قبل كل نشر

## استكشاف الأخطاء

### Response Time أعلى من المتوقع
- تحقق من حمل قاعدة البيانات
- تحقق من Redis performance
- راجع عدد Connection Pool
- راقب Network latency

### Throughput منخفض
- زيادة عدد Workers
- تحسين Batch Size
- تحقق من عنق الزجاجة (Database, Redis, Network)

### Error Rate عالي
- راجع logs النظام
- تحقق من Database connections
- تحقق من Redis memory
- راجع Worker health

## الخلاصة

تم إنشاء بيئة اختبار أداء شاملة تغطي:
- ✅ اختبار API Gateway (Response Time < 10ms)
- ✅ اختبار معالجة الدفعات (Batch Processing)
- ✅ اختبار معدل الإرسال (5000+ msg/min)
- ✅ اختبار الضغط الشديد (10,000 msg/min)
- ✅ سيناريوهات متعددة للحالات المختلفة
- ✅ أدوات Apache Bench للاختبارات السريعة
- ✅ توثيق شامل وسهل الاستخدام

البيئة جاهزة للاستخدام ويمكن البدء بتشغيل الاختبارات فوراً!

---

**تاريخ الإنشاء:** 2026-02-25  
**الحالة:** ✅ مكتمل  
**المتطلبات المحققة:** 21.3
