# خطة التنفيذ - إعادة بناء دورة حياة الرسالة الكاملة

## نظرة عامة

هذه الخطة تغطي التنفيذ الكامل لإعادة بناء دورة حياة الرسالة من الاستقبال حتى التسليم النهائي، مع التركيز على:
- الحماية المالية المطلقة من Race Conditions
- الأداء العالي (5000+ رسالة/دقيقة)
- العزل التام بين الشركات
- التوافق الكامل مع الواجهات الحالية (admin-web, client-web)

**قيد حرج:** لا يجوز تغيير أي واجهات موجودة (APIs, WebSocket Events, Database Schema المستخدم من الواجهات).

## المهام

### 1. إعداد البنية التحتية والإعدادات

- [x] 1.1 إنشاء Database Migrations للجداول الجديدة
  - إنشاء migration لجدول balance_history
  - إنشاء migration لجدول accounting_transactions
  - إضافة indexes للأداء والعزل
  - إضافة constraints للحماية (balance >= 0)
  - _المتطلبات: 10.1, 10.2, 11.2_

- [x] 1.2 تحديث Docker Compose للخدمات الجديدة
  - إضافة Redis service مع persistence
  - تحديث PostgreSQL configuration (connection pool)
  - إضافة Prometheus و Grafana للمراقبة
  - إعداد environment variables للبيئات المختلفة
  - _المتطلبات: 12.1, 16.1, 38.1_

- [x] 1.3 إعداد Redis Configuration
  - تحديد Redis data structures (Lists, Hashes, Sets)
  - إعداد TTL policies للمفاتيح
  - إعداد persistence (AOF + RDB)
  - إعداد memory limits وeviction policies
  - _المتطلبات: 3.2, 5.2_

- [x] 1.4 إعداد Connection Pool Configuration
  - تحديد max connections = 100
  - تحديد idle connections = 10
  - تحديد connection max lifetime = 1 hour
  - تحديد idle timeout = 10 minutes
  - _المتطلبات: 12.1, 12.2, 12.3, 12.4_


### 2. تطوير المكونات الأساسية - Balance Manager

- [x] 2.1 تطوير Balance Manager مع Atomic Operations
  - إنشاء BalanceManager interface و implementation
  - تطبيق DeductBalance مع Atomic UPDATE + WHERE condition
  - تطبيق RefundBalance مع Atomic INCREMENT
  - تطبيق CheckSufficient للتحقق من كفاية الرصيد
  - إضافة retry logic مع exponential backoff للـ concurrent updates
  - _المتطلبات: 1.1, 1.2, 1.3, 1.5_

- [x] 2.2 كتابة Property Test للـ Balance Manager
  - **Property 1: Atomic Balance Deduction Correctness**
  - **يتحقق من: المتطلبات 1.1, 1.2, 1.3, 11.1, 11.7**
  - توليد عمليات خصم متزامنة عشوائية
  - التحقق من صحة الرصيد النهائي = الرصيد الأولي - مجموع الخصومات الناجحة

- [x]* 2.3 كتابة Property Test لرفض الخصم عند عدم كفاية الرصيد
  - **Property 2: Balance Deduction Rejection on Insufficient Funds**
  - **يتحقق من: المتطلبات 1.2, 1.4, 11.3**
  - توليد محاولات خصم عشوائية برصيد غير كافٍ
  - التحقق من رفض العملية وتحديث حالة الرسالة إلى FAILED

- [x] 2.4 تطوير Balance History Logger
  - تسجيل كل عملية خصم/استرداد في balance_history
  - تسجيل: tenant_id, message_id, old_balance, new_balance, amount, type, reason, created_by
  - إضافة IP address logging للعمليات المالية
  - _المتطلبات: 10.1, 10.2, 10.3, 10.4, 10.5, 20.2_

- [x] 2.5 كتابة Property Test لاكتمال Balance History
  - **Property 6: Balance History Completeness**
  - **يتحقق من: المتطلبات 10.1, 10.2, 10.3, 10.4, 10.5, 2.6**
  - توليد عمليات رصيد عشوائية
  - التحقق من وجود سجل كامل في balance_history لكل عملية

- [x]* 2.6 كتابة Property Test لـ Balance History Invariant
  - **Property 7: Balance History Invariant**
  - **يتحقق من: المتطلبات 10.6**
  - إجراء عمليات عشوائية
  - التحقق من: sum(balance_history.amount) = current_balance - initial_balance

- [x] 2.7 تطوير Low Balance Notification System
  - إرسال تنبيه عند الرصيد < 10 وحدات
  - إرسال تنبيه حرج عند الرصيد < 5 وحدات
  - تسجيل محاولات الخصم الفاشلة
  - _المتطلبات: 11.4, 11.5, 11.6_

- [x]* 2.8 كتابة Unit Tests للإشعارات
  - اختبار إرسال تنبيه عند الرصيد < 10
  - اختبار إرسال تنبيه حرج عند الرصيد < 5
  - اختبار تسجيل محاولات الخصم الفاشلة


### 3. تطوير API Gateway مع Redis Buffer

- [x] 3.1 تطوير API Gateway مع Validation
  - إنشاء MessageRequest و MessageResponse structs
  - تطبيق validation للبيانات (phone_number, content length)
  - رفض الطلبات غير الصحيحة مع HTTP 400
  - _المتطلبات: 3.6, 3.7, 12.1_

- [x]* 3.2 كتابة Property Test للـ Input Validation
  - **Property 12: Input Validation Rejection**
  - **يتحقق من: المتطلبات 3.6**
  - توليد طلبات عشوائية غير صحيحة
  - التحقق من رفضها مع HTTP 400

- [x] 3.2 تطوير Rate Limiter لكل Tenant
  - تطبيق rate limiting منفصل لكل tenant_id
  - استخدام Redis لتخزين العدادات (rate_limit:{tenant_id})
  - رفض الطلبات الزائدة مع HTTP 429
  - حد افتراضي: 100 طلب/دقيقة
  - _المتطلبات: 3.4, 3.5_

- [x]* 3.3 كتابة Property Test للـ Rate Limiting
  - **Property 11: Rate Limiting Enforcement**
  - **يتحقق من: المتطلبات 3.4, 3.5**
  - إرسال 101+ طلب في دقيقة واحدة
  - التحقق من رفض الطلبات الزائدة مع HTTP 429

- [x] 3.4 تطوير Redis Buffer Integration
  - تطبيق MessageBuffer interface
  - Push messages إلى Redis List: messages:pending:{tenant_id}
  - تحديث tenants:active Set
  - تحديث messages:count:{tenant_id}
  - _المتطلبات: 3.2_

- [x]* 3.5 كتابة Property Test للـ Message Buffering
  - **Property 9: Message Buffering in Redis**
  - **يتحقق من: المتطلبات 3.2**
  - إرسال رسائل عشوائية
  - التحقق من وجودها في Redis قبل إرجاع الاستجابة

- [x] 3.6 تحسين API Response Time
  - تطبيق non-blocking Redis operations
  - إرجاع استجابة فورية بعد LPUSH
  - قياس response time والتأكد من < 10ms
  - _المتطلبات: 3.1_

- [x]* 3.7 كتابة Property Test للـ API Response Time
  - **Property 8: API Response Time Performance**
  - **يتحقق من: المتطلبات 3.1**
  - إرسال طلبات صحيحة عشوائية
  - قياس وقت الاستجابة والتحقق من < 10ms

- [x]* 3.8 كتابة Property Test للـ API Response Format
  - **Property 10: API Response Format Completeness**
  - **يتحقق من: المتطلبات 3.3**
  - إرسال طلبات صحيحة عشوائية
  - التحقق من وجود message_id و status="QUEUED" في الاستجابة


### 4. تطوير Batch Processor

- [x] 4.1 تطوير Batch Processor الأساسي
  - إنشاء BatchProcessor interface و implementation
  - تطبيق ProcessBatch لاستخراج 100 رسالة من Redis
  - تطبيق batch insert في PostgreSQL ضمن transaction واحد
  - معالجة كل tenant بشكل منفصل
  - _المتطلبات: 4.1, 4.2_

- [x]* 4.2 كتابة Property Test للـ Batch Insert Atomicity
  - **Property 13: Batch Insert Atomicity**
  - **يتحقق من: المتطلبات 4.2**
  - محاكاة فشل batch insert
  - التحقق من عدم وجود إدخالات جزئية

- [x] 4.2 تطوير River Queue Integration
  - إنشاء River Queue jobs بعد batch insert
  - ربط كل job بـ tenant_id و message_id
  - تحديد max_attempts = 3
  - _المتطلبات: 4.4_

- [x]* 4.3 كتابة Property Test للـ River Job Creation
  - **Property 14: River Job Creation Completeness**
  - **يتحقق من: المتطلبات 4.4**
  - إدخال رسائل عشوائية
  - التحقق من وجود River job مقابل لكل رسالة

- [x] 4.4 تطوير Adaptive Batch Processing
  - تطبيق AdaptBatchSize بناءً على حجم Buffer
  - Buffer < 10: معالجة فورية (50-100ms)
  - Buffer 10-100: دفعات صغيرة كل 100ms
  - Buffer > 100: دفعات كبيرة (100 رسالة) بدون تأخير
  - _المتطلبات: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

- [x] 4.5 تطبيق Batch Failure Recovery
  - عند فشل batch insert، إعادة محاولة الرسائل فردياً
  - تسجيل الرسائل الفاشلة
  - _المتطلبات: 4.7_

- [x]* 4.6 كتابة Property Test للـ Batch Failure Recovery
  - **Property 16: Batch Failure Individual Retry**
  - **يتحقق من: المتطلبات 4.7**
  - محاكاة فشل batch
  - التحقق من إعادة محاولة الرسائل فردياً

- [x]* 4.7 كتابة Performance Test للـ Batch Processing
  - **Property 15: Batch Processing Performance**
  - **يتحقق من: المتطلبات 4.5**
  - قياس وقت معالجة 100 رسالة في دفعة
  - قياس وقت معالجة 100 رسالة فردياً
  - التحقق من أن الدفعة أسرع 50x على الأقل

- [x] 4.8 Checkpoint - التحقق من عمل Batch Processor
  - التأكد من استخراج الرسائل من Redis بنجاح
  - التأكد من إدخال الدفعات في PostgreSQL
  - التأكد من إنشاء River jobs
  - اسأل المستخدم إذا كانت هناك أسئلة


### 5. تطوير Routing Engine مع Caching

- [x] 5.1 تطوير Routing Engine الأساسي
  - إنشاء RoutingEngine interface و implementation
  - تطبيق SelectChannel للبحث عن أفضل قناة متاحة
  - فلترة القنوات بحالة ONLINE فقط
  - ترتيب حسب Priority (DESC) ثم CurrentLoad (ASC)
  - _المتطلبات: 5.3, 5.4_

- [x]* 5.2 كتابة Property Test للـ Channel Selection
  - **Property 17: Channel Selection Online Status**
  - **يتحقق من: المتطلبات 5.3**
  - اختيار قنوات عشوائية
  - التحقق من أن جميعها بحالة ONLINE

- [x]* 5.3 كتابة Property Test للـ Channel Priority Ordering
  - **Property 18: Channel Selection Priority Ordering**
  - **يتحقق من: المتطلبات 5.4**
  - الحصول على قنوات متاحة
  - التحقق من الترتيب الصحيح (Priority DESC, Load ASC)

- [x] 5.2 تطوير Redis Caching للقنوات
  - تخزين معلومات القنوات في Redis Hash: channels:cache:{tenant_id}
  - TTL = 5 دقائق
  - البحث في Cache أولاً، ثم Database عند Cache miss
  - _المتطلبات: 5.1, 5.2_

- [x]* 5.4 كتابة Property Test للـ Cache Hit Rate
  - **Property 21: Cache Hit Rate Performance**
  - **يتحقق من: المتطلبات 5.7**
  - إجراء 100 عملية اختيار قناة متتالية لنفس الـ tenant
  - التحقق من أن 99+ عملية تُخدم من Cache

- [x] 5.3 تطوير Load Balancing
  - توزيع الرسائل بالتساوي على القنوات المتاحة
  - اختيار القناة ذات أقل current_load
  - تحديث current_load بعد كل إرسال
  - _المتطلبات: 5.6_

- [x]* 5.5 كتابة Property Test للـ Load Balancing
  - **Property 20: Load Balancing Distribution**
  - **يتحقق من: المتطلبات 5.6**
  - إرسال N رسالة إلى M قناة
  - التحقق من التوزيع المتساوي (N/M ± tolerance)

- [x] 5.4 تطوير No Channel Available Handling
  - عند عدم وجود قنوات متاحة، تحديث حالة الرسالة إلى WAITING
  - تسجيل الحدث
  - _المتطلبات: 5.5_

- [x]* 5.6 كتابة Property Test للـ No Channel Handling
  - **Property 19: No Channel Available Handling**
  - **يتحقق من: المتطلبات 5.5**
  - محاكاة عدم وجود قنوات متاحة
  - التحقق من تحديث حالة الرسالة إلى WAITING


### 6. تطوير Worker Pool للمعالجة المتوازية

- [x] 6.1 تطوير River Worker Pool
  - إنشاء 50 worker متزامن
  - تطبيق River job handler للرسائل
  - توزيع Jobs بالتساوي على Workers
  - _المتطلبات: 13.1, 13.2_

- [x] 6.2 تطبيق Worker Health Monitoring
  - مراقبة صحة Workers كل 10 ثوانٍ
  - إعادة تشغيل Workers الفاشلة تلقائياً
  - تسجيل أداء كل Worker
  - _المتطلبات: 13.4, 13.5, 13.7_

- [x]* 6.3 كتابة Property Test للـ Worker Failure Recovery
  - **Property 34: Failed Worker Job Redistribution**
  - **يتحقق من: المتطلبات 17.3**
  - محاكاة فشل worker
  - التحقق من إعادة توزيع jobs على workers أخرى

- [x] 6.3 تطبيق Message Processing في Worker
  - خصم الرصيد عبر Balance Manager
  - اختيار القناة عبر Routing Engine
  - إرسال الرسالة عبر WebSocket Hub
  - تحديث حالة الرسالة
  - _المتطلبات: 1.1, 5.1, 6.1_

- [x] 6.4 تطبيق Retry Logic للرسائل الفاشلة
  - إعادة محاولة الرسائل الفاشلة 3 مرات
  - استخدام exponential backoff
  - نقل الرسائل الفاشلة نهائياً إلى Dead Letter Queue
  - _المتطلبات: 17.4, 17.5_

- [x]* 6.5 كتابة Property Test للـ Retry Limit
  - **Property 35: Message Retry Limit**
  - **يتحقق من: المتطلبات 17.4**
  - محاكاة فشل إرسال
  - التحقق من إعادة المحاولة 3 مرات بالضبط

- [x]* 6.6 كتابة Property Test للـ Dead Letter Queue
  - **Property 36: Dead Letter Queue Final Destination**
  - **يتحقق من: المتطلبات 17.5**
  - استنفاد جميع المحاولات
  - التحقق من نقل الرسالة إلى DLQ

- [x]* 6.7 كتابة Property Test للـ Message Reliability
  - **Property 37: Message Delivery or DLQ Guarantee**
  - **يتحقق من: المتطلبات 17.7**
  - إرسال رسائل عشوائية
  - التحقق من وصولها للمستلم أو DLQ (لا فقدان)

- [x]* 6.8 كتابة Performance Test للـ Throughput
  - **Property 13 (Performance): معالجة متوازية للرسائل**
  - **يتحقق من: المتطلبات 13.6**
  - قياس معدل الإرسال
  - التحقق من >= 5000 رسالة/دقيقة

- [x] 6.9 Checkpoint - التحقق من عمل Workers
  - التأكد من معالجة الرسائل بنجاح
  - التأكد من خصم الرصيد بشكل صحيح
  - التأكد من اختيار القنوات
  - اسأل المستخدم إذا كانت هناك أسئلة


### 7. تطوير WebSocket Hub

- [x] 7.1 تطوير WebSocket Hub الأساسي
  - إنشاء WebSocketHub interface و implementation
  - تطبيق RegisterConnection و UnregisterConnection
  - إدارة اتصالات منفصلة لكل tenant_id و channel_id
  - _المتطلبات: 6.4_

- [x] 7.2 تطبيق Non-blocking Message Send
  - استخدام Buffered Channel بسعة 1000 رسالة
  - تطبيق SendMessage بشكل غير متزامن
  - Fail fast عند امتلاء Buffer
  - _المتطلبات: 6.1, 6.2, 6.3_

- [x]* 7.3 كتابة Property Test للـ Buffer Overflow
  - **Property 22: WebSocket Buffer Overflow Rejection**
  - **يتحقق من: المتطلبات 6.3**
  - ملء Buffer إلى السعة الكاملة
  - التحقق من رفض الرسائل الجديدة فوراً

- [x] 7.3 تطبيق Automatic Reconnection
  - مراقبة حالة الاتصال
  - إعادة الاتصال تلقائياً خلال 5 ثوانٍ عند الانقطاع
  - _المتطلبات: 6.5_

- [x]* 7.4 كتابة Property Test للـ Reconnection Timeliness
  - **Property 23: WebSocket Reconnection Timeliness**
  - **يتحقق من: المتطلبات 6.5**
  - محاكاة انقطاع الاتصال
  - قياس وقت إعادة الاتصال والتحقق من <= 5 ثوانٍ

- [x] 7.4 تطبيق Heartbeat Mechanism
  - إرسال heartbeat كل 30 ثانية
  - التحقق من استجابة الجهاز
  - قطع الاتصال عند عدم الاستجابة
  - _المتطلبات: 6.6_

- [x] 7.5 تطبيق Message Logging
  - تسجيل جميع الرسائل المرسلة والمستقبلة
  - تسجيل timestamp و tenant_id و channel_id
  - _المتطلبات: 6.7_

- [x]* 7.6 كتابة Property Test للـ Message Logging
  - **Property 24: WebSocket Message Logging Completeness**
  - **يتحقق من: المتطلبات 6.7**
  - إرسال/استقبال رسائل عشوائية
  - التحقق من وجود سجلات كاملة

- [x]* 7.7 كتابة Property Test للـ Tenant Isolation
  - **Property 43: Tenant WebSocket Connection Isolation**
  - **يتحقق من: العزل بين الشركات**
  - محاولة إرسال رسالة لجهاز tenant آخر
  - التحقق من الرفض


### 8. تطوير Status Report Handler و Accounting Engine

- [x] 8.1 تطوير Status Report Handler الأساسي
  - إنشاء StatusReportHandler interface و implementation
  - استقبال تقارير الحالة من WebSocket
  - تحديث حالة الرسائل في قاعدة البيانات
  - _المتطلبات: 2.1_

- [x] 8.2 تطبيق Idempotency Mechanism
  - توليد idempotency key: {tenant_id}:{message_id}:{status}
  - التحقق من Redis قبل المعالجة
  - تخزين key في Redis مع TTL = 24 ساعة
  - تخطي التقارير المعالجة مسبقاً
  - _المتطلبات: 2.3, 2.4_

- [x]* 8.3 كتابة Property Test للـ Idempotency
  - **Property 3: Status Report Idempotency**
  - **يتحقق من: المتطلبات 2.3, 2.4**
  - معالجة نفس التقرير مرتين
  - التحقق من نفس النتيجة النهائية

- [x] 8.3 تطوير Accounting Engine للتسوية
  - إنشاء AccountingEngine interface و implementation
  - تطبيق SettleTransaction للرسائل الناجحة (COMPLETED)
  - تطبيق RefundTransaction للرسائل الفاشلة (REFUNDED)
  - _المتطلبات: 2.2, 2.5_

- [x] 8.4 تطبيق Atomic Refund للرسائل الفاشلة
  - استخدام Balance Manager للاسترداد الذري
  - تحديث accounting_transactions إلى REFUNDED
  - تسجيل العملية في balance_history
  - تسجيل في Audit Trail
  - _المتطلبات: 2.2, 2.5, 2.6_

- [x]* 8.5 كتابة Property Test للـ Failed Message Refund
  - **Property 4: Failed Message Refund Atomicity**
  - **يتحقق من: المتطلبات 2.2, 2.5**
  - توليد رسائل فاشلة عشوائية
  - التحقق من استرداد الرصيد مرة واحدة بالضبط

- [x] 8.5 تطبيق Successful Message Settlement
  - تحديث accounting_transactions إلى COMPLETED
  - تسجيل في Audit Trail
  - _المتطلبات: 2.1_

- [x]* 8.6 كتابة Property Test للـ Successful Message Settlement
  - **Property 5: Successful Message Accounting Settlement**
  - **يتحقق من: المتطلبات 2.1**
  - توليد رسائل ناجحة عشوائية
  - التحقق من تحديث accounting_transaction إلى COMPLETED مرة واحدة

- [x] 8.6 تطبيق Retry Logic مع Exponential Backoff
  - إعادة محاولة معالجة التقرير عند الفشل
  - استخدام exponential backoff
  - حد أقصى 3 محاولات
  - _المتطلبات: 2.7_

- [x] 8.7 Checkpoint - التحقق من معالجة التقارير
  - التأكد من معالجة تقارير SENT/DELIVERED بنجاح
  - التأكد من استرداد الرصيد للرسائل الفاشلة
  - التأكد من عدم تكرار المعالجة
  - اسأل المستخدم إذا كانت هناك أسئلة


### 9. تحديث Mobile Gateway (React Native)

- [x] 9.1 تحديث SQLite Schema في Mobile Gateway
  - إضافة tenant_id إلى جدول messages
  - إضافة indexes للأداء: idx_messages_tenant, idx_messages_status
  - إضافة UNIQUE constraint على (message_id, tenant_id)
  - _المتطلبات: 7.1_

- [x] 9.2 تطوير Event-Driven Message Processing
  - استقبال الرسائل عبر WebSocket فوراً
  - حفظ في SQLite قبل إرسال ACK
  - معالجة فورية بدون انتظار (Event-Driven)
  - _المتطلبات: 7.1, 7.3_

- [x]* 9.3 كتابة Property Test للـ SQLite Persistence
  - **Property 25: Mobile Gateway SQLite Persistence**
  - **يتحقق من: المتطلبات 7.1**
  - استقبال رسائل عشوائية
  - التحقق من حفظها في SQLite قبل ACK

- [x] 9.3 تطبيق BUFFERED Status Report
  - إرسال تقرير BUFFERED خلال 50ms من الاستقبال
  - _المتطلبات: 7.2_

- [x]* 9.4 كتابة Property Test للـ Buffered Report Timeliness
  - **Property 26: Mobile Gateway Buffered Report Timeliness**
  - **يتحقق من: المتطلبات 7.2**
  - استقبال رسائل عشوائية
  - قياس وقت إرسال تقرير BUFFERED والتحقق من <= 50ms

- [x] 9.4 تطبيق Native SMS Send
  - إرسال الرسالة عبر Native SMS خلال 100ms
  - دعم إرسال 5-10 رسائل متزامنة
  - _المتطلبات: 7.4, 7.5_

- [x]* 9.5 كتابة Property Test للـ SMS Send Timeliness
  - **Property 27: Mobile Gateway SMS Send Timeliness**
  - **يتحقق من: المتطلبات 7.4**
  - استقبال رسائل عشوائية
  - قياس وقت الإرسال والتحقق من <= 100ms

- [x]* 9.6 كتابة Property Test للـ Concurrent Send Capacity
  - **Property 28: Mobile Gateway Concurrent Send Capacity**
  - **يتحقق من: المتطلبات 7.5**
  - إرسال 10 رسائل متزامنة
  - التحقق من معالجتها جميعاً بدون blocking

- [x] 9.5 تطبيق Retry Logic للإرسال الفاشل
  - إعادة المحاولة 3 مرات مع exponential backoff
  - إرسال تقرير FAILED عند فشل جميع المحاولات
  - _المتطلبات: 7.6_

- [x] 9.6 تطبيق Immediate Status Report
  - إرسال تقرير SENT أو FAILED فور معرفة النتيجة
  - _المتطلبات: 7.7_

- [x]* 9.7 كتابة Property Test للـ Status Report Immediacy
  - **Property 29: Mobile Gateway Status Report Immediacy**
  - **يتحقق من: المتطلبات 7.7**
  - إرسال رسائل عشوائية
  - التحقق من إرسال تقرير الحالة فوراً


### 10. تطبيق آليات العزل التام بين الشركات

- [x] 10.1 مراجعة وتدقيق جميع Database Queries
  - التأكد من وجود WHERE tenant_id = ? في جميع Queries
  - إضافة tenant_id filtering حيث مفقود
  - مراجعة Joins للتأكد من العزل
  - _المتطلبات: 18.4_

- [x]* 10.2 كتابة Property Test للـ Database Isolation
  - **Property 41: Tenant Data Isolation in Database**
  - **يتحقق من: العزل في قاعدة البيانات**
  - تدقيق جميع Queries
  - التحقق من وجود WHERE tenant_id في كل query

- [x] 10.2 مراجعة وتدقيق جميع Redis Keys
  - التأكد من تضمين tenant_id في جميع Keys
  - استخدام نمط: {resource}:{tenant_id}:{identifier}
  - _المتطلبات: 3.2, 5.2_

- [x]* 10.3 كتابة Property Test للـ Redis Isolation
  - **Property 42: Tenant Data Isolation in Redis**
  - **يتحقق من: العزل في Redis**
  - تدقيق جميع Redis keys
  - التحقق من تضمين tenant_id في كل key

- [x] 10.3 تطبيق Rate Limiting Isolation
  - التأكد من عدادات منفصلة لكل tenant
  - اختبار عدم تأثير tenant على آخر
  - _المتطلبات: 3.4_

- [x]* 10.4 كتابة Property Test للـ Rate Limiting Isolation
  - **Property 44: Tenant Rate Limiting Isolation**
  - **يتحقق من: عزل Rate Limiting**
  - استنفاد rate limit لـ tenant واحد
  - التحقق من عدم تأثر tenant آخر

- [x] 10.4 تطبيق Balance Isolation
  - التأكد من عدم تأثير عمليات tenant على رصيد آخر
  - اختبار العمليات المتزامنة لـ tenants مختلفة
  - _المتطلبات: 1.1_

- [x]* 10.5 كتابة Property Test للـ Balance Isolation
  - **Property 45: Tenant Balance Isolation**
  - **يتحقق من: عزل الرصيد المالي**
  - إجراء عمليات لـ tenant واحد
  - التحقق من عدم تغير رصيد tenants أخرى

- [x] 10.5 Checkpoint - التحقق من العزل التام
  - اختبار عدم تسرب البيانات بين tenants
  - اختبار عدم تأثير عمليات tenant على آخر
  - مراجعة جميع نقاط الوصول للبيانات
  - اسأل المستخدم إذا كانت هناك أسئلة


### 11. ضمان التوافق مع الواجهات الحالية

- [x] 11.1 مراجعة API Endpoints الحالية
  - التأكد من عدم تغيير أي endpoint paths
  - التأكد من عدم تغيير Request/Response format
  - التأكد من عدم تغيير HTTP status codes
  - _المتطلبات: 18.1, 18.2_

- [x]* 11.2 كتابة Property Test للـ API Compatibility
  - **Property 38: API Request/Response Format Compatibility**
  - **يتحقق من: المتطلبات 18.2**
  - مقارنة request/response formats مع المواصفات الأصلية
  - التحقق من التطابق الكامل

- [x] 11.2 مراجعة WebSocket Events
  - التأكد من عدم تغيير event types
  - التأكد من عدم تغيير event payload structure
  - _المتطلبات: 18.3_

- [x]* 11.3 كتابة Property Test للـ WebSocket Compatibility
  - **Property 39: WebSocket Event Format Compatibility**
  - **يتحقق من: المتطلبات 18.3**
  - مقارنة event formats مع المواصفات الأصلية
  - التحقق من التطابق الكامل

- [x] 11.3 مراجعة Database Schema للجداول المستخدمة من الواجهات
  - التأكد من عدم تغيير أسماء الأعمدة
  - التأكد من عدم تغيير أنواع البيانات
  - التأكد من عدم حذف أعمدة مستخدمة
  - _المتطلبات: 18.4_

- [x] 11.4 اختبار التكامل مع admin-web
  - تشغيل admin-web مع Backend الجديد
  - اختبار جميع الوظائف الأساسية
  - التأكد من عدم وجود أخطاء
  - _المتطلبات: 18.6_

- [x] 11.5 اختبار التكامل مع client-web
  - تشغيل client-web مع Backend الجديد
  - اختبار جميع الوظائف الأساسية
  - التأكد من عدم وجود أخطاء
  - _المتطلبات: 18.6_

- [x]* 11.6 كتابة Property Test للـ Internal Changes Compatibility
  - **Property 40: Internal Changes External Compatibility**
  - **يتحقق من: المتطلبات 18.5, 18.7**
  - إجراء تغييرات داخلية
  - التحقق من استمرار عمل الواجهات بدون تعديل

- [x] 11.6 Checkpoint - التحقق من التوافق الكامل
  - التأكد من عمل admin-web بدون مشاكل
  - التأكد من عمل client-web بدون مشاكل
  - التأكد من عدم الحاجة لتعديل كود الواجهات
  - اسأل المستخدم إذا كانت هناك أسئلة


### 12. تطبيق المراقبة والتنبيهات

- [x] 12.1 تطبيق Prometheus Metrics
  - تسجيل: messages_received, messages_sent, messages_failed
  - تسجيل: average_latency, queue_size, success_rate
  - تسجيل: database_connections, cache_hit_rate
  - توفير endpoint /metrics
  - _المتطلبات: 16.1, 16.2, 16.3, 16.4_

- [x] 12.2 تطبيق Alert Rules
  - تنبيه عند queue_size > 1000
  - تنبيه عند average_latency > 2 ثانية
  - تنبيه حرج عند success_rate < 95%
  - _المتطلبات: 16.5, 16.6, 16.7_

- [x]* 12.3 كتابة Property Test للـ Low Balance Notification
  - **Property 30: Low Balance Warning Notification**
  - **يتحقق من: المتطلبات 11.4**
  - تقليل الرصيد إلى أقل من 10
  - التحقق من إرسال تنبيه

- [x]* 12.4 كتابة Property Test للـ Critical Balance Notification
  - **Property 31: Critical Balance Warning Notification**
  - **يتحقق من: المتطلبات 11.5**
  - تقليل الرصيد إلى أقل من 5
  - التحقق من إرسال تنبيه حرج

- [x]* 12.5 كتابة Property Test للـ Insufficient Balance Logging
  - **Property 32: Insufficient Balance Attempt Logging**
  - **يتحقق من: المتطلبات 11.6**
  - محاولة خصم برصيد غير كافٍ
  - التحقق من تسجيل المحاولة

- [x] 12.3 إنشاء Grafana Dashboards
  - Dashboard للأداء العام (Throughput, Latency, Success Rate)
  - Dashboard للموارد (CPU, Memory, Database Connections)
  - Dashboard للمحاسبة (Balance Operations, Refunds)
  - _المتطلبات: 22.1, 22.2, 22.3, 22.4_

- [x] 12.4 تطبيق Audit Trail Logging
  - تسجيل جميع العمليات الحرجة
  - تسجيل: User ID, Action, Timestamp, IP Address, Result
  - تخزين Audit Logs بشكل immutable
  - _المتطلبات: 39.1, 39.2, 39.3, 39.4, 39.5_


### 13. تطبيق معالجة الأخطاء والموثوقية

- [x] 13.1 تطبيق Error Handling في جميع المكونات
  - تعريف Error Types لكل مكون
  - تطبيق Error Response Format موحد
  - تسجيل الأخطاء مع Stack Trace
  - _المتطلبات: 19.1, 19.7_

- [x] 13.2 تطبيق Retry Logic مع Exponential Backoff
  - إعادة محاولة العمليات الفاشلة
  - استخدام exponential backoff
  - حد أقصى للمحاولات
  - _المتطلبات: 19.2_

- [x] 13.3 تطبيق Circuit Breaker للخدمات الخارجية
  - حماية من فشل الخدمات الخارجية
  - فتح الدائرة بعد عدد معين من الفشل
  - إعادة المحاولة بعد فترة زمنية
  - _المتطلبات: 19.3_

- [x] 13.4 تطبيق Fallback Strategies
  - التحول لمعالجة مباشرة عند فشل Redis
  - التحول لقاعدة بيانات احتياطية عند فشل الرئيسية
  - _المتطلبات: 19.5, 24.2, 24.3_

- [x] 13.5 تطبيق Message Persistence
  - حفظ جميع الرسائل في قاعدة البيانات قبل المعالجة
  - استخدام River Queue مع Persistent Storage
  - _المتطلبات: 17.1, 17.2_

- [x]* 13.6 كتابة Property Test للـ Message Persistence
  - **Property 33: Message Persistence Before Processing**
  - **يتحقق من: المتطلبات 17.1**
  - معالجة رسائل عشوائية
  - التحقق من وجودها في قاعدة البيانات أولاً

- [x] 13.6 تطبيق Dead Letter Queue
  - نقل الرسائل الفاشلة نهائياً إلى DLQ
  - الاحتفاظ بالرسائل لمدة 7 أيام
  - توفير API لإعادة معالجة رسائل DLQ
  - _المتطلبات: 17.5, 17.6_

- [x] 13.7 تطبيق Automatic Recovery
  - إعادة تشغيل Workers المتعطلة تلقائياً
  - استكمال معالجة الرسائل المتبقية بعد عطل
  - _المتطلبات: 24.4, 24.6_


### 14. اختبارات الأداء والتحميل

- [x] 14.1 إعداد بيئة اختبار الأداء
  - إعداد بيانات اختبار (seed data)
  - إعداد أدوات قياس الأداء (k6, Apache Bench)
  - إعداد سيناريوهات الاختبار
  - _المتطلبات: 21.3_

- [x] 14.2 اختبار أداء API Gateway
  - قياس response time لـ 1000 طلب
  - التحقق من response time < 10ms
  - قياس throughput (requests/second)
  - _المتطلبات: 3.1_

- [x] 14.3 اختبار أداء Batch Processing
  - قياس وقت معالجة دفعات مختلفة الأحجام
  - التحقق من تحسن الأداء مع الدفعات
  - قياس database load
  - _المتطلبات: 4.5_

- [x] 14.4 اختبار Load Testing
  - محاكاة 5000+ رسالة/دقيقة
  - قياس end-to-end latency
  - مراقبة استهلاك الموارد (CPU, Memory, Database)
  - _المتطلبات: 13.6, 21.6_

- [x] 14.5 اختبار Stress Testing
  - محاكاة 10,000 رسالة/دقيقة
  - اختبار استقرار النظام تحت ضغط شديد
  - تحديد نقاط الفشل
  - _المتطلبات: 21.6_

- [x] 14.6 اختبار Concurrent Operations
  - محاكاة 100 عملية خصم رصيد متزامنة
  - التحقق من عدم وجود race conditions
  - التحقق من صحة الرصيد النهائي
  - _المتطلبات: 1.3_

- [x] 14.7 اختبار Cache Performance
  - قياس cache hit rate
  - التحقق من تحسن الأداء مع Cache
  - قياس تأثير Cache على database load
  - _المتطلبات: 5.7_

- [x] 14.8 Checkpoint - مراجعة نتائج الأداء
  - التأكد من تحقيق الأداء المستهدف (5000+ رسالة/دقيقة)
  - التأكد من response time < 10ms
  - التأكد من success rate >= 99%
  - اسأل المستخدم إذا كانت هناك أسئلة أو تحسينات مطلوبة


### 15. اختبارات التكامل الشاملة

- [x] 15.1 اختبار End-to-End Flow الكامل
  - إرسال رسالة من API حتى التسليم النهائي
  - التحقق من جميع المراحل (Queue, Process, Send, Report)
  - التحقق من صحة خصم واسترداد الرصيد
  - _المتطلبات: 17.7_

- [x] 15.2 اختبار Bulk Message Processing
  - إرسال 10,000 رسالة دفعة واحدة
  - التحقق من معالجتها جميعاً
  - قياس وقت المعالجة الكامل
  - _المتطلبات: 8.1, 8.7_

- [x] 15.3 اختبار Failed Message Handling
  - محاكاة فشل إرسال رسائل
  - التحقق من إعادة المحاولات
  - التحقق من استرداد الرصيد
  - التحقق من نقل الرسائل إلى DLQ
  - _المتطلبات: 2.2, 17.4, 17.5_

- [x] 15.4 اختبار Multi-Tenant Scenarios
  - إرسال رسائل من عدة tenants متزامنة
  - التحقق من العزل التام بين tenants
  - التحقق من عدم تأثير tenant على آخر
  - _المتطلبات: 1.1, 3.4, 5.1_

- [x] 15.5 اختبار WebSocket Reconnection
  - محاكاة انقطاع اتصال WebSocket
  - التحقق من إعادة الاتصال التلقائي
  - التحقق من استكمال إرسال الرسائل المعلقة
  - _المتطلبات: 6.5, 24.6_

- [x] 15.6 اختبار Database Failure Recovery
  - محاكاة فشل قاعدة البيانات
  - التحقق من التحول للقاعدة الاحتياطية
  - التحقق من عدم فقدان البيانات
  - _المتطلبات: 24.3_

- [x] 15.7 اختبار Redis Failure Fallback
  - محاكاة فشل Redis
  - التحقق من التحول للمعالجة المباشرة
  - التحقق من استمرار عمل النظام
  - _المتطلبات: 19.5, 24.2_

- [x] 15.8 Checkpoint - مراجعة اختبارات التكامل
  - التأكد من نجاح جميع سيناريوهات التكامل
  - التأكد من عدم وجود أخطاء حرجة
  - التأكد من موثوقية النظام
  - اسأل المستخدم إذا كانت هناك أسئلة


### 16. التوثيق والنشر

- [x] 16.1 كتابة Architecture Documentation
  - توثيق البنية المعمارية الجديدة
  - رسم مخططات التدفق (Flow Diagrams)
  - شرح المكونات والواجهات
  - _المتطلبات: 21.7_

- [x] 16.2 كتابة API Documentation
  - توثيق جميع API endpoints
  - إضافة أمثلة للطلبات والاستجابات
  - توثيق Error Codes والرسائل
  - _المتطلبات: 21.7_

- [x] 16.3 كتابة Database Schema Documentation
  - توثيق جميع الجداول والأعمدة
  - توثيق العلاقات والـ Constraints
  - توثيق Indexes والأداء
  - _المتطلبات: 21.7_

- [x] 16.4 كتابة Deployment Guide
  - خطوات النشر التفصيلية
  - متطلبات البنية التحتية
  - إعدادات البيئة (Environment Variables)
  - _المتطلبات: 21.7_

- [x] 16.5 كتابة Migration Guide
  - خطوات الانتقال من النظام القديم
  - سكريبتات ترحيل البيانات
  - خطة Rollback عند الحاجة
  - _المتطلبات: 21.7_

- [x] 16.6 كتابة Operations Manual
  - دليل التشغيل اليومي
  - إجراءات المراقبة والصيانة
  - إجراءات النسخ الاحتياطي والاسترداد
  - _المتطلبات: 21.7_

- [x] 16.7 كتابة Troubleshooting Guide
  - المشاكل الشائعة وحلولها
  - كيفية قراءة Logs والمقاييس
  - إجراءات التعامل مع الطوارئ
  - _المتطلبات: 21.7_

- [x] 16.8 إعداد بيئة Staging للاختبار النهائي
  - نشر النظام الجديد في Staging
  - اختبار شامل في بيئة مشابهة للإنتاج
  - اختبار التوافق مع الواجهات
  - _المتطلبات: 38.2_

- [x] 16.9 إعداد خطة النشر للإنتاج
  - تحديد موعد النشر
  - إعداد خطة Rollback
  - إعداد فريق الدعم للطوارئ
  - إعداد خطة التواصل مع العملاء
  - _المتطلبات: 38.6_

- [x] 16.10 Checkpoint النهائي - الاستعداد للنشر
  - مراجعة جميع الاختبارات والتأكد من نجاحها
  - مراجعة التوثيق والتأكد من اكتماله
  - مراجعة خطة النشر والـ Rollback
  - الحصول على موافقة المستخدم للنشر


## ملاحظات هامة

### أولويات التنفيذ

المهام مرتبة حسب الأولوية والتبعيات:

**المرحلة 1 (حرجة):** المهام 1-2
- إعداد البنية التحتية
- Balance Manager مع الحماية المالية

**المرحلة 2 (أساسية):** المهام 3-6
- API Gateway و Redis Buffer
- Batch Processor
- Routing Engine
- Worker Pool

**المرحلة 3 (تكميلية):** المهام 7-9
- WebSocket Hub
- Status Report Handler
- Mobile Gateway

**المرحلة 4 (ضمان الجودة):** المهام 10-15
- العزل بين الشركات
- التوافق مع الواجهات
- المراقبة والتنبيهات
- معالجة الأخطاء
- اختبارات الأداء والتكامل

**المرحلة 5 (النشر):** المهمة 16
- التوثيق والنشر

### المهام الاختيارية (Optional)

المهام المميزة بـ `*` هي اختبارات اختيارية يمكن تخطيها للحصول على MVP أسرع:
- جميع Property-Based Tests
- جميع Unit Tests
- بعض Integration Tests

**ملاحظة:** يُنصح بشدة بتنفيذ الاختبارات للمهام الحرجة (Balance Manager, Status Report Handler).

### معايير القبول لكل مهمة

كل مهمة يجب أن تحقق:
1. الكود يعمل بدون أخطاء
2. جميع الاختبارات تنجح (إن وجدت)
3. الكود يتبع معايير Go best practices
4. التوثيق الداخلي (comments) كافٍ
5. لا توجد تحذيرات من linter

### تقدير الوقت الإجمالي

- المرحلة 1: 3-5 أيام
- المرحلة 2: 7-10 أيام
- المرحلة 3: 5-7 أيام
- المرحلة 4: 7-10 أيام
- المرحلة 5: 3-5 أيام

**الإجمالي:** 25-37 يوم عمل (5-7 أسابيع)

### الملفات المتأثرة الرئيسية

**Backend (Go):**
- `services/backend/internal/balance/manager.go` (جديد)
- `services/backend/internal/buffer/redis_buffer.go` (جديد)
- `services/backend/internal/batch/processor.go` (جديد)
- `services/backend/internal/routing/engine.go` (جديد)
- `services/backend/internal/websocket/hub.go` (تحديث)
- `services/backend/internal/status/handler.go` (جديد)
- `services/backend/internal/accounting/engine.go` (جديد)
- `services/backend/internal/worker/pool.go` (جديد)

**Mobile Gateway (React Native):**
- `services/mobile_gateway/src/database/schema.ts` (تحديث)
- `services/mobile_gateway/src/websocket/client.ts` (تحديث)
- `services/mobile_gateway/src/sms/sender.ts` (تحديث)

**Database:**
- `migrations/xxx_create_balance_history.sql` (جديد)
- `migrations/xxx_create_accounting_transactions.sql` (جديد)
- `migrations/xxx_update_messages_table.sql` (تحديث)

**Infrastructure:**
- `docker-compose.yml` (تحديث)
- `prometheus.yml` (جديد)
- `grafana/dashboards/` (جديد)

### القيود الحرجة

**يجب الالتزام بها:**
1. لا تغيير لأي API endpoints موجودة
2. لا تغيير لـ Request/Response format
3. لا تغيير لـ WebSocket events format
4. لا تغيير لـ Database schema المستخدم من الواجهات
5. لا تعديل لكود admin-web أو client-web

**مسموح:**
- إضافة جداول جديدة في قاعدة البيانات
- إضافة أعمدة جديدة للجداول الموجودة (optional)
- تحسين الأداء الداخلي
- إضافة مكونات Backend جديدة

### نقاط التحقق (Checkpoints)

تم إضافة 6 نقاط تحقق في المهام:
1. بعد Batch Processor (المهمة 4.8)
2. بعد Worker Pool (المهمة 6.9)
3. بعد Status Report Handler (المهمة 8.7)
4. بعد العزل بين الشركات (المهمة 10.5)
5. بعد التوافق مع الواجهات (المهمة 11.6)
6. بعد اختبارات الأداء (المهمة 14.8)
7. بعد اختبارات التكامل (المهمة 15.8)
8. قبل النشر (المهمة 16.10)

في كل نقطة تحقق، يجب:
- التأكد من نجاح جميع الاختبارات
- مراجعة الكود والتأكد من الجودة
- سؤال المستخدم إذا كانت هناك أسئلة أو تعديلات

---

**تاريخ الإنشاء:** 2026-02-25  
**الإصدار:** 1.0  
**الحالة:** جاهز للتنفيذ
