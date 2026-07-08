# Grafana Dashboards - SMS Gateway Monitoring

## نظرة عامة

هذا المجلد يحتوي على Grafana Dashboards لمراقبة نظام SMS Gateway في الوقت الفعلي. تم تصميم هذه اللوحات لتوفير رؤية شاملة للأداء والموارد والعمليات المالية.

## المحتويات

### 1. Dashboards

#### Performance Dashboard (`performance-dashboard.json`)
لوحة مراقبة الأداء العام للنظام:

**المقاييس الرئيسية:**
- **Throughput**: معدل معالجة الرسائل (رسالة/دقيقة)
  - أخضر: < 3000 رسالة/دقيقة
  - أصفر: 3000-5000 رسالة/دقيقة
  - أحمر: > 5000 رسالة/دقيقة

- **Average Latency**: متوسط وقت معالجة الرسالة (ميلي ثانية)
  - أخضر: < 1000ms
  - أصفر: 1000-2000ms
  - أحمر: > 2000ms

- **Success Rate**: نسبة نجاح إرسال الرسائل (%)
  - أحمر: < 95%
  - أصفر: 95-98%
  - أخضر: > 98%

- **Queue Size**: عدد الرسائل في الانتظار
  - أخضر: < 500
  - أصفر: 500-1000
  - أحمر: > 1000

**الرسوم البيانية:**
- Message Throughput Over Time (Received, Sent, Failed)
- Latency Percentiles (p50, p95, p99)
- Message Status Distribution (Pie Chart)
- Queue Size & Active Workers

**الاستخدام:**
- مراقبة الأداء العام للنظام
- اكتشاف الاختناقات (Bottlenecks)
- تتبع معدل النجاح والفشل
- مراقبة حجم الطابور

---

#### Resources Dashboard (`resources-dashboard.json`)
لوحة مراقبة الموارد والبنية التحتية:

**المقاييس الرئيسية:**
- **CPU Usage**: استخدام المعالج (%)
  - أخضر: < 70%
  - أصفر: 70-90%
  - أحمر: > 90%

- **Memory Usage**: استخدام الذاكرة (%)
  - أخضر: < 75%
  - أصفر: 75-90%
  - أحمر: > 90%

- **Active DB Connections**: عدد اتصالات قاعدة البيانات النشطة
  - أخضر: < 80
  - أصفر: 80-100
  - أحمر: > 100

- **Redis Connections**: عدد اتصالات Redis
  - أخضر: < 40
  - أصفر: 40-50
  - أحمر: > 50

**الرسوم البيانية:**
- CPU & Memory Usage Over Time
- Database Connection Pool (Active, Idle, Waiting)
- Redis Memory Usage (Used vs Max)
- Workers & WebSocket Connections
- Cache Hit Rate
- Database Query Duration

**الاستخدام:**
- مراقبة استهلاك الموارد
- تحديد مشاكل الأداء
- تحسين استخدام Connection Pool
- مراقبة Cache Performance

---

#### Financial Dashboard (`financial-dashboard.json`)
لوحة مراقبة العمليات المالية والمحاسبة:

**المقاييس الرئيسية:**
- **Total Balance Deductions**: إجمالي الخصومات ($)
- **Total Refunds**: إجمالي المبالغ المستردة ($)
- **Total Balance Operations**: إجمالي العمليات المالية

**الرسوم البيانية:**
- Balance Operations Over Time (Deductions, Refunds, Charges)
- Balance Operations by Type (Pie Chart)
- Refunds vs Failed Messages
- Low Balance Alerts Table (< $10)
- Insufficient Balance Attempts
- Accounting Transactions per Hour
- Tenants with Low Balance (< $10)
- Tenants with Critical Balance (< $5)

**التنبيهات:**
- تنبيه عند رصيد < $10 (أصفر)
- تنبيه حرج عند رصيد < $5 (أحمر)

**الاستخدام:**
- مراقبة العمليات المالية
- تتبع الخصومات والاستردادات
- اكتشاف الحسابات ذات الرصيد المنخفض
- تحليل محاولات الخصم الفاشلة

---

### 2. Provisioning

#### Datasources (`provisioning/datasources/prometheus.yml`)
تكوين مصدر البيانات Prometheus:
- URL: `http://prometheus:9090`
- Interval: 5 ثوانٍ
- Query Timeout: 60 ثانية
- HTTP Method: POST

#### Dashboards (`provisioning/dashboards/dashboards.yml`)
تكوين تحميل اللوحات تلقائياً:
- Folder: SMS Gateway
- Update Interval: 10 ثوانٍ
- Allow UI Updates: نعم

---

## التثبيت والإعداد

### 1. المتطلبات الأساسية

- Docker و Docker Compose
- Prometheus يعمل ويجمع المقاييس
- Backend يُصدّر المقاييس على `/metrics`

### 2. بدء التشغيل

```bash
# بدء جميع الخدمات (بما في ذلك Grafana)
docker-compose up -d

# التحقق من حالة Grafana
docker-compose ps grafana

# عرض سجلات Grafana
docker-compose logs -f grafana
```

### 3. الوصول إلى Grafana

- **URL**: http://localhost:3000
- **Username**: admin (افتراضي)
- **Password**: admin (افتراضي)

**ملاحظة:** يُنصح بتغيير كلمة المرور الافتراضية عند أول تسجيل دخول.

### 4. التحقق من اللوحات

بعد تسجيل الدخول:
1. انتقل إلى **Dashboards** → **Browse**
2. افتح مجلد **SMS Gateway**
3. يجب أن ترى 3 لوحات:
   - SMS Gateway - Performance Dashboard
   - SMS Gateway - Resources Dashboard
   - SMS Gateway - Financial Dashboard

---

## المقاييس المطلوبة (Prometheus Metrics)

لكي تعمل اللوحات بشكل صحيح، يجب أن يُصدّر Backend المقاييس التالية:

### Performance Metrics
```
# Message counters
messages_received_total
messages_sent_total
messages_failed_total
messages_by_status_total{status="SENT|FAILED|PENDING|..."}

# Latency metrics
message_processing_latency_milliseconds
message_processing_duration_milliseconds_bucket

# Queue metrics
redis_queue_size
active_workers
```

### Resource Metrics
```
# System metrics
node_cpu_seconds_total{mode="idle"}
node_memory_MemAvailable_bytes
node_memory_MemTotal_bytes

# Database metrics
database_connections_active
database_connections_idle
database_connections_waiting
database_query_duration_milliseconds

# Redis metrics
redis_connected_clients
redis_memory_used_bytes
redis_memory_max_bytes
redis_cache_hits
redis_cache_misses

# WebSocket metrics
websocket_connections_active
```

### Financial Metrics
```
# Balance operations
balance_deductions_total
balance_refunds_total
balance_charges_total
balance_operations_total
balance_operations_by_type_total{type="DEDUCT|REFUND|CHARGE"}

# Tenant balance
tenant_balance{tenant_id="..."}

# Refunds
refund_operations_total
insufficient_balance_attempts_total

# Accounting
accounting_transactions_total
```

---

## التخصيص

### تعديل اللوحات

يمكنك تعديل اللوحات من خلال:

1. **من واجهة Grafana:**
   - افتح اللوحة
   - اضغط على أيقونة الإعدادات (⚙️)
   - قم بالتعديلات المطلوبة
   - احفظ التغييرات

2. **من ملفات JSON:**
   - عدّل الملفات في `grafana/dashboards/`
   - أعد تشغيل Grafana: `docker-compose restart grafana`

### إضافة لوحات جديدة

1. أنشئ ملف JSON جديد في `grafana/dashboards/`
2. استخدم نفس البنية الموجودة في اللوحات الحالية
3. أعد تشغيل Grafana

### تغيير فترة التحديث

في كل لوحة، يمكنك تغيير:
- **Refresh Rate**: افتراضياً 5 ثوانٍ
- **Time Range**: افتراضياً آخر ساعة

---

## استكشاف الأخطاء

### اللوحات لا تظهر

```bash
# تحقق من سجلات Grafana
docker-compose logs grafana

# تحقق من ملفات provisioning
ls -la grafana/provisioning/datasources/
ls -la grafana/provisioning/dashboards/
ls -la grafana/dashboards/
```

### لا توجد بيانات في اللوحات

1. **تحقق من Prometheus:**
   ```bash
   # افتح Prometheus UI
   http://localhost:9090
   
   # تحقق من Targets
   http://localhost:9090/targets
   ```

2. **تحقق من Backend Metrics:**
   ```bash
   curl http://localhost:8080/metrics
   ```

3. **تحقق من Datasource في Grafana:**
   - Configuration → Data Sources → Prometheus
   - اضغط "Test" للتحقق من الاتصال

### الأداء بطيء

- قلل فترة التحديث (Refresh Rate)
- قلل نطاق الوقت (Time Range)
- استخدم Aggregation في الاستعلامات

---

## أفضل الممارسات

### 1. المراقبة المستمرة
- راقب اللوحات بانتظام
- اضبط التنبيهات للمقاييس الحرجة
- راجع الاتجاهات الأسبوعية والشهرية

### 2. التنبيهات
- اضبط تنبيهات Prometheus للمقاييس الحرجة:
  - Queue Size > 1000
  - Success Rate < 95%
  - CPU Usage > 90%
  - Low Balance < $10

### 3. الأمان
- غيّر كلمة مرور admin الافتراضية
- استخدم HTTPS في الإنتاج
- قيّد الوصول إلى Grafana

### 4. النسخ الاحتياطي
- احفظ نسخة احتياطية من ملفات JSON
- صدّر اللوحات المخصصة بانتظام

---

## المتغيرات البيئية

يمكن تخصيص Grafana عبر المتغيرات البيئية في `.env`:

```env
# Grafana Configuration
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=your_secure_password
GRAFANA_ROOT_URL=http://localhost:3000
GRAFANA_PLUGINS=
```

---

## الموارد الإضافية

- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboard Best Practices](https://grafana.com/docs/grafana/latest/best-practices/)

---

## الدعم

للمساعدة أو الإبلاغ عن مشاكل:
1. راجع سجلات Grafana: `docker-compose logs grafana`
2. تحقق من Prometheus Targets: http://localhost:9090/targets
3. تحقق من Backend Metrics: http://localhost:8080/metrics

---

## الإصدار

- **Version**: 1.0
- **Last Updated**: 2024
- **Grafana Version**: Latest (10.x)
- **Prometheus Version**: Latest

---

## الترخيص

هذه اللوحات جزء من نظام SMS Gateway وتخضع لنفس الترخيص.
