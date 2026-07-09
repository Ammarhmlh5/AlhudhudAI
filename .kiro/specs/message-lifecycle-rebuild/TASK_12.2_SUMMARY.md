# Task 12.2: تطبيق Alert Rules - ملخص التنفيذ

## نظرة عامة

تم تنفيذ المهمة 12.2 بنجاح، والتي تتضمن إنشاء قواعد التنبيه (Alert Rules) لنظام Prometheus لمراقبة أداء وصحة نظام Al-Hudhud SMS Gateway.

## المتطلبات المنفذة

- ✅ **المتطلب 16.5**: تنبيه عند queue_size > 1000
- ✅ **المتطلب 16.6**: تنبيه عند average_latency > 2 ثانية
- ✅ **المتطلب 16.7**: تنبيه حرج عند success_rate < 95%

## الملفات المنشأة

### 1. prometheus/alert_rules.yml
ملف قواعد التنبيه الرئيسي يحتوي على:

#### مجموعات التنبيهات (Alert Groups):

**أ. Queue and Performance Alerts (queue_performance)**
- `HighQueueSize`: تنبيه عند حجم الطابور > 1000 رسالة (warning)
- `CriticalQueueSize`: تنبيه حرج عند حجم الطابور > 5000 رسالة (critical)
- `HighAverageLatency`: تنبيه عند متوسط التأخير > 2 ثانية (warning)
- `CriticalLatency`: تنبيه حرج عند متوسط التأخير > 5 ثوانٍ (critical)

**ب. Reliability Alerts (reliability)**
- `LowSuccessRate`: تنبيه حرج عند معدل النجاح < 95% (critical) ✅ المتطلب 16.7
- `DegradedSuccessRate`: تنبيه عند معدل النجاح < 98% (warning)
- `HighFailureRate`: تنبيه عند معدل الفشل > 10 فشل/ثانية (warning)

**ج. Database Alerts (database)**
- `HighDatabaseConnections`: تنبيه عند الاتصالات > 80 (warning)
- `CriticalDatabaseConnections`: تنبيه حرج عند الاتصالات > 95 (critical)
- `SlowDatabaseQueries`: تنبيه عند متوسط وقت الاستعلام > 1 ثانية (warning)

**د. Cache Performance Alerts (cache)**
- `LowCacheHitRate`: تنبيه عند معدل إصابة الكاش < 90% (warning)
- `CriticalCacheHitRate`: تنبيه حرج عند معدل إصابة الكاش < 70% (critical)
- `HighRedisMemoryUsage`: تنبيه عند استخدام ذاكرة Redis > 80% (warning)

**هـ. Worker Pool Alerts (workers)**
- `WorkerFailures`: تنبيه عند فشل العمال > 1 فشل/ثانية (warning)
- `HighWorkerFailures`: تنبيه حرج عند فشل العمال > 5 فشل/ثانية (critical)
- `LowWorkerCount`: تنبيه عند عدد العمال النشطين < 30 (warning)

**و. Financial Alerts (financial)**
- `LowTenantBalance`: تنبيه عند رصيد الشركة < 10 وحدات (warning)
- `CriticalTenantBalance`: تنبيه حرج عند رصيد الشركة < 5 وحدات (critical)
- `HighInsufficientBalanceAttempts`: تنبيه عند محاولات الرصيد غير الكافي > 5/ثانية (warning)

**ز. System Health Alerts (system_health)**
- `SlowAPIResponses`: تنبيه عند وقت استجابة API > 10ms (warning)
- `HighAPIErrorRate`: تنبيه عند معدل أخطاء API > 10 خطأ/ثانية (warning)
- `HighWebSocketDisconnections`: تنبيه عند انقطاعات WebSocket > 5/ثانية (warning)

**ح. Throughput Alerts (throughput)**
- `LowMessageThroughput`: تنبيه عند معدل المعالجة < 1000 رسالة/دقيقة (warning)
- `CriticalLowThroughput`: تنبيه حرج عند معدل المعالجة < 500 رسالة/دقيقة (critical)

### 2. prometheus/ALERTMANAGER_SETUP.md
دليل شامل لإعداد Alertmanager يتضمن:

- **نظرة عامة**: شرح Alertmanager ووظائفه
- **خطوات التثبيت**: إعداد Alertmanager مع Docker Compose
- **التكوين**: أمثلة تكوين لقنوات الإشعارات المختلفة:
  - Email (Gmail)
  - Slack
  - PagerDuty
  - Telegram
- **اختبار التنبيهات**: كيفية اختبار قواعد التنبيه
- **مستويات الخطورة**: تعريف مستويات warning و critical
- **إجراءات الاستجابة**: خطوات الاستجابة لكل نوع تنبيه
- **المراقبة**: كيفية مراقبة صحة Alertmanager
- **استكشاف الأخطاء**: حل المشاكل الشائعة
- **أفضل الممارسات**: نصائح لإدارة التنبيهات بفعالية

### 3. prometheus/README.md
وثائق شاملة لقواعد التنبيه تتضمن:

- **جدول ملخص التنبيهات**: جدول شامل لجميع التنبيهات مع العتبات والخطورة
- **التعليقات التوضيحية**: شرح annotations المستخدمة في كل تنبيه
- **التسميات**: شرح labels المستخدمة
- **استخدام قواعد التنبيه**: أوامر للتحقق والاختبار
- **حالات التنبيه**: شرح Inactive, Pending, Firing
- **تخصيص القواعد**: كيفية تعديل العتبات والمدد
- **المقاييس المستخدمة**: قائمة بجميع المقاييس المطلوبة
- **التكامل مع Grafana**: كيفية عرض التنبيهات في Grafana
- **أفضل الممارسات**: نصائح لإدارة التنبيهات
- **استكشاف الأخطاء**: حل المشاكل الشائعة

## التحديثات على الملفات الموجودة

### 1. prometheus.yml
تم تحديث ملف التكوين لتضمين قواعد التنبيه:

```yaml
# Load rules once and periodically evaluate them
rule_files:
  - "/etc/prometheus/alert_rules.yml"
```

### 2. docker-compose.yml
تم تحديث خدمة Prometheus لتحميل ملف قواعد التنبيه:

```yaml
volumes:
  - ./prometheus.yml:/etc/prometheus/prometheus.yml
  - ./prometheus/alert_rules.yml:/etc/prometheus/alert_rules.yml
  - prometheus_data:/prometheus
```

## مميزات التنبيهات المنفذة

### 1. Severity Levels
- **Warning**: تنبيهات تحتاج متابعة خلال 30 دقيقة
- **Critical**: تنبيهات حرجة تحتاج استجابة فورية (خلال 5 دقائق)

### 2. Annotations الشاملة
كل تنبيه يحتوي على:
- **summary**: وصف مختصر للتنبيه
- **description**: وصف تفصيلي مع القيمة الحالية والعتبة
- **impact**: تأثير المشكلة على النظام أو المستخدمين
- **action**: خطوات موصى بها لحل المشكلة

### 3. Labels المنظمة
- **severity**: مستوى الخطورة (warning/critical)
- **component**: المكون المتأثر (queue, database, cache, workers, financial, api, websocket, performance)

### 4. For Duration
كل تنبيه له مدة انتظار قبل الإطلاق لتجنب التنبيهات الكاذبة:
- Critical alerts: 30s - 2m
- Warning alerts: 2m - 10m

## المقاييس المطلوبة

التنبيهات تتوقع المقاييس التالية من Backend:

### Queue Metrics
- `hudhud_queue_size`: حجم الطابور الحالي

### Performance Metrics
- `hudhud_message_latency_seconds_avg`: متوسط زمن معالجة الرسالة
- `hudhud_api_response_time_seconds_avg`: متوسط وقت استجابة API

### Reliability Metrics
- `hudhud_messages_sent_total`: إجمالي الرسائل المرسلة بنجاح
- `hudhud_messages_failed_total`: إجمالي الرسائل الفاشلة
- `hudhud_messages_processed_total`: إجمالي الرسائل المعالجة

### Database Metrics
- `hudhud_db_connections_active`: الاتصالات النشطة
- `hudhud_db_query_duration_seconds_avg`: متوسط وقت الاستعلام

### Cache Metrics
- `hudhud_cache_hits_total`: إجمالي إصابات الكاش
- `hudhud_cache_misses_total`: إجمالي إخفاقات الكاش
- `hudhud_redis_memory_used_bytes`: استخدام ذاكرة Redis
- `hudhud_redis_memory_max_bytes`: الحد الأقصى لذاكرة Redis

### Worker Metrics
- `hudhud_worker_failures_total`: إجمالي فشل العمال
- `hudhud_workers_active`: عدد العمال النشطين

### Financial Metrics
- `hudhud_tenant_balance`: رصيد الشركة الحالي
- `hudhud_balance_insufficient_total`: إجمالي محاولات الرصيد غير الكافي

### System Metrics
- `hudhud_api_errors_total`: إجمالي أخطاء API
- `hudhud_websocket_disconnections_total`: إجمالي انقطاعات WebSocket

## كيفية الاستخدام

### 1. بدء الخدمات

```bash
# بدء جميع الخدمات بما في ذلك Prometheus
docker-compose up -d

# التحقق من تحميل قواعد التنبيه
curl http://localhost:9090/api/v1/rules | jq
```

### 2. عرض التنبيهات النشطة

```bash
# عرض التنبيهات المطلقة حالياً
curl http://localhost:9090/api/v1/alerts | jq
```

### 3. الوصول إلى واجهة Prometheus

افتح المتصفح على: http://localhost:9090
- انتقل إلى **Alerts** لعرض جميع قواعد التنبيه وحالتها
- استخدم **Graph** لاستعلام المقاييس وتصورها

### 4. إعداد Alertmanager (اختياري)

اتبع الدليل في `prometheus/ALERTMANAGER_SETUP.md` لإعداد إرسال التنبيهات عبر:
- Email
- Slack
- PagerDuty
- Telegram

## اختبار التنبيهات

### اختبار HighQueueSize Alert

```bash
# محاكاة حجم طابور عالي (يتطلب تعديل المقياس في Backend)
# أو انتظار حدوث حمل فعلي يتجاوز 1000 رسالة

# التحقق من حالة التنبيه
curl "http://localhost:9090/api/v1/query?query=ALERTS{alertname='HighQueueSize'}" | jq
```

### اختبار HighAverageLatency Alert

```bash
# التحقق من متوسط التأخير الحالي
curl "http://localhost:9090/api/v1/query?query=hudhud_message_latency_seconds_avg" | jq

# التحقق من حالة التنبيه
curl "http://localhost:9090/api/v1/query?query=ALERTS{alertname='HighAverageLatency'}" | jq
```

### اختبار LowSuccessRate Alert

```bash
# حساب معدل النجاح الحالي
curl "http://localhost:9090/api/v1/query?query=(hudhud_messages_sent_total/(hudhud_messages_sent_total+hudhud_messages_failed_total))*100" | jq

# التحقق من حالة التنبيه
curl "http://localhost:9090/api/v1/query?query=ALERTS{alertname='LowSuccessRate'}" | jq
```

## إعادة تحميل التكوين

بعد تعديل قواعد التنبيه:

```bash
# إعادة تحميل تكوين Prometheus بدون إعادة تشغيل
curl -X POST http://localhost:9090/-/reload

# أو إعادة تشغيل الحاوية
docker restart hudhud-prometheus
```

## التحقق من صحة التكوين

```bash
# التحقق من صحة ملف قواعد التنبيه
docker run --rm -v $(pwd)/prometheus:/prometheus prom/prometheus:latest \
  promtool check rules /prometheus/alert_rules.yml

# التحقق من صحة تكوين Prometheus
docker run --rm -v $(pwd):/workspace prom/prometheus:latest \
  promtool check config /workspace/prometheus.yml
```

## الخطوات التالية

1. **إعداد Alertmanager**: اتبع الدليل في `ALERTMANAGER_SETUP.md` لإعداد إرسال التنبيهات
2. **تكوين قنوات الإشعارات**: إعداد Email, Slack, أو قنوات أخرى
3. **اختبار التنبيهات**: التأكد من عمل جميع التنبيهات بشكل صحيح
4. **ضبط العتبات**: تعديل العتبات بناءً على البيانات الفعلية
5. **إنشاء Grafana Dashboards**: عرض التنبيهات والمقاييس في Grafana

## الملاحظات الهامة

1. **المقاييس المطلوبة**: يجب أن يكون Backend يعرض جميع المقاييس المطلوبة على `/metrics`
2. **العتبات**: العتبات الحالية هي قيم افتراضية وقد تحتاج للتعديل بناءً على البيئة الفعلية
3. **For Duration**: المدد المحددة تمنع التنبيهات الكاذبة من المشاكل العابرة
4. **Severity Levels**: استخدم critical للمشاكل التي تحتاج استجابة فورية فقط

## المراجع

- [Prometheus Alerting Documentation](https://prometheus.io/docs/alerting/latest/overview/)
- [Alert Rule Best Practices](https://prometheus.io/docs/practices/alerting/)
- [PromQL Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)

## الحالة

✅ **مكتمل** - تم تنفيذ جميع متطلبات المهمة 12.2 بنجاح

## التاريخ

- **تاريخ الإنشاء**: 2024-01-XX
- **آخر تحديث**: 2024-01-XX
- **الحالة**: مكتمل ✅
