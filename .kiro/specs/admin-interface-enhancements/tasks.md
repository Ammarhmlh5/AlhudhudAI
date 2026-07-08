# خطة التنفيذ: تحسينات واجهة الإدارة (Admin Interface Enhancements)

## نظرة عامة

تهدف هذه الخطة إلى تطوير نظام إدارة شامل يتضمن:
- نظام تقييم الشركات بناءً على مؤشرات الأداء
- مراقبة صحة السيرفرات المخصصة
- توزيع ذكي للرسائل على أجهزة متعددة
- محرك توجيه Signal/Push مع مراقبة التوقيت
- واجهة إدارة متقدمة (React + TypeScript)
- تكامل مع الأنظمة الموجودة (التعرفات، البوابات، الموارد)

التقنيات المستخدمة:
- Backend: Go (Golang)
- Frontend: React 18+ with TypeScript
- Database: PostgreSQL 15+
- Cache: Redis 7+
- Monitoring: Prometheus + Grafana

## المهام

### المرحلة 1: إنشاء قاعدة البيانات (Database Schema)

- [x] 1. إنشاء جداول قاعدة البيانات الأساسية
  - [x] 1.1 إنشاء جدول company_ratings
    - إنشاء ملف migration: `services/backend/migrations/032_company_ratings_table.sql`
    - تعريف الحقول: tenant_id, success_rate, sending_volume, payment_compliance, service_quality_score, overall_rating, rating_category
    - إضافة indexes للأداء: idx_company_ratings_tenant, idx_company_ratings_category
    - تفعيل Row Level Security للعزل بين الشركات
    - إضافة constraints للتحقق من صحة البيانات
    - _المتطلبات: 9.1_
  
  - [x] 1.2 إنشاء جدول performance_metrics
    - إنشاء ملف migration: `services/backend/migrations/033_performance_metrics_table.sql`
    - تعريف الحقول: tenant_id, period_start, period_end, period_type, total_messages, successful_messages, failed_messages
    - إضافة computed columns: success_rate, error_rate, payment_compliance
    - إضافة indexes: idx_performance_metrics_tenant_period, idx_performance_metrics_type
    - _المتطلبات: 9.2_

  - [x] 1.3 إنشاء جدول routing_configurations
    - إنشاء ملف migration: `services/backend/migrations/034_routing_configurations_table.sql`
    - تعريف الحقول: tenant_id, routing_type, dedicated_server_id, dedicated_server_url, health_status
    - إضافة foreign key إلى dedicated_gateways
    - إضافة indexes: idx_routing_config_tenant, idx_routing_config_type
    - _المتطلبات: 9.3_
  
  - [x] 1.4 إنشاء جدول distribution_strategies
    - إنشاء ملف migration: `services/backend/migrations/035_distribution_strategies_table.sql`
    - تعريف الحقول: tenant_id, strategy_type, strategy_config (JSONB), custom_rules (JSONB)
    - إضافة constraint للتحقق من strategy_type
    - إضافة indexes: idx_distribution_strategies_tenant
    - _المتطلبات: 9.4_
  
  - [x] 1.5 إنشاء جدول phone_devices
    - إنشاء ملف migration: `services/backend/migrations/036_phone_devices_table.sql`
    - تعريف الحقول: id (UUID), tenant_id, device_name, priority_level, status, current_load
    - إضافة حقول metrics: total_messages_sent, successful_messages, failed_messages
    - إضافة indexes: idx_phone_devices_tenant, idx_phone_devices_status, idx_phone_devices_priority
    - _المتطلبات: 9.7_
  
  - [x] 1.6 إنشاء جدول distribution_metrics
    - إنشاء ملف migration: `services/backend/migrations/037_distribution_metrics_table.sql`
    - تعريف الحقول: tenant_id, device_id, period_start, period_end, messages_assigned, messages_sent
    - إضافة foreign key إلى phone_devices
    - إضافة unique constraint: (tenant_id, device_id, period_start)
    - _المتطلبات: 9.8_
  
  - [x] 1.7 إنشاء جدول server_health_logs
    - إنشاء ملف migration: `services/backend/migrations/038_server_health_logs_table.sql`
    - تعريف الحقول: server_id, tenant_id, check_timestamp, response_time_ms, status_code, health_status
    - إضافة indexes: idx_server_health_logs_server, idx_server_health_logs_tenant
    - _المتطلبات: 9.5_
  
  - [x] 1.8 إنشاء جدول alerts
    - إنشاء ملف migration: `services/backend/migrations/039_alerts_table.sql`
    - تعريف الحقول: tenant_id, alert_type, severity, title, message, status, context (JSONB)
    - إضافة حقول: acknowledged_at, acknowledged_by, resolved_at, dedup_key
    - إضافة indexes: idx_alerts_tenant, idx_alerts_type, idx_alerts_severity, idx_alerts_dedup
    - إضافة partial index للتنبيهات النشطة فقط
    - _المتطلبات: 9.6_

- [x] 2. إنشاء جداول نظام Signal/Push
  - [x] 2.1 إنشاء جدول message_queue
    - إنشاء ملف migration: `services/backend/migrations/040_message_queue_table.sql`
    - تعريف الحقول: id (UUID), tenant_id, message_id, message_type, message_content, recipient, device_id
    - إضافة حقول status: status, priority, retry_count, max_retries
    - إضافة indexes: idx_message_queue_tenant_status, idx_message_queue_device, idx_message_queue_priority
    - _المتطلبات: 9.1_
  
  - [x] 2.2 إنشاء جدول waiting_messages
    - إنشاء ملف migration: `services/backend/migrations/041_waiting_messages_table.sql`
    - تعريف الحقول: id (UUID), tenant_id, message_id, status, sent_to_device_at
    - إضافة unique constraint على message_id
    - إضافة indexes: idx_waiting_messages_tenant, idx_waiting_messages_message_id
    - _المتطلبات: 9.1_
  
  - [x] 2.3 إنشاء جدول timing_violations
    - إنشاء ملف migration: `services/backend/migrations/042_timing_violations_table.sql`
    - تعريف الحقول: tenant_id, message_id, message_type, expected_gap_ms, actual_gap_ms, violation_severity
    - إضافة indexes: idx_timing_violations_tenant, idx_timing_violations_severity
    - _المتطلبات: 9.1_
  
  - [x] 2.4 إنشاء جدول tenant_statistics
    - إنشاء ملف migration: `services/backend/migrations/043_tenant_statistics_table.sql`
    - تعريف الحقول: tenant_id, message_type, current_mpm, current_stddev, daily_avg_mpm, daily_peak_mpm
    - إضافة حقول anomaly: anomaly_score, anomaly_level, is_anomaly, anomaly_reason
    - إضافة unique constraint: (tenant_id, message_type)
    - _المتطلبات: 9.1_
  
  - [x] 2.5 إنشاء جدول enforcement_actions
    - إنشاء ملف migration: `services/backend/migrations/044_enforcement_actions_table.sql`
    - تعريف الحقول: tenant_id, action_type, reason, anomaly_level, violation_rate, z_score, peak_ratio
    - إضافة حقول: previous_mode, new_mode, created_at, expires_at
    - إضافة indexes: idx_enforcement_actions_tenant, idx_enforcement_actions_type
    - _المتطلبات: 9.1_
  
  - [x] 2.6 إنشاء جدول tenant_timing_config
    - إنشاء ملف migration: `services/backend/migrations/045_tenant_timing_config_table.sql`
    - تعريف الحقول: tenant_id, signal_min_gap_ms, push_min_gap_ms, violation_rate_threshold
    - إضافة حقول: z_score_threshold, peak_ratio_threshold, current_mode, forced_mode
    - إضافة unique constraint على tenant_id
    - _المتطلبات: 9.1_

- [x] 3. نقطة فحص - التحقق من Migrations
  - تشغيل جميع migrations والتأكد من نجاحها
  - التحقق من إنشاء جميع الجداول والـ indexes
  - اختبار Row Level Security policies
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 2: تطوير Backend - نظام تقييم الشركات

- [x] 4. تطوير Company Rating Engine
  - [x] 4.1 إنشاء نماذج البيانات (Data Models)
    - إنشاء ملف `services/backend/internal/admin/rating/types.go`
    - تعريف structs: CompanyRating, PerformanceMetrics, RatingWeights, RatingCategory
    - تعريف constants للفئات: Excellent, Good, Average, Poor
    - إضافة validation tags
    - _المتطلبات: 1.1, 1.4_
  
  - [x] 4.2 تطوير Rating Calculator
    - إنشاء ملف `services/backend/internal/admin/rating/calculator.go`
    - تطبيق دالة CalculateRating() بناءً على الأوزان
    - تطبيق دالة ClassifyRating() لتصنيف التقييم
    - تطبيق دالة NormalizeVolume() لتطبيع حجم الإرسال
    - تطبيق دالة CalculateQualityScore() لحساب جودة الخدمة
    - _المتطلبات: 1.1, 1.4_
  
  - [ ]* 4.3 كتابة Property Tests لـ Rating Calculator
    - **Property 1: Rating Calculation Correctness**
    - **Property 2: Rating Category Classification**
    - إنشاء ملف `services/backend/internal/admin/rating/calculator_property_test.go`
    - استخدام gopter لتوليد بيانات عشوائية
    - التحقق من صحة الحسابات لجميع المدخلات
    - _يتحقق من: المتطلبات 1.1, 1.4_
  
  - [x] 4.4 تطوير Metrics Collector
    - إنشاء ملف `services/backend/internal/admin/rating/collector.go`
    - تطبيق دالة CollectMetrics() لجمع البيانات من مصادر متعددة
    - التكامل مع gateway_statistics للحصول على إحصائيات الرسائل
    - التكامل مع tariff system للحصول على بيانات الدفع
    - التكامل مع resource_usage للحصول على استخدام الموارد
    - _المتطلبات: 4.1, 4.2, 4.3_
  
  - [x] 4.5 تطوير Rating Update Trigger
    - إنشاء ملف `services/backend/internal/admin/rating/trigger.go`
    - تطبيق نظام debouncing (تحديث كل 5 ثواني كحد أقصى)
    - تطبيق event handlers: OnMessageSent, OnMessageFailed, OnPaymentReceived
    - استخدام goroutines للتحديثات غير المتزامنة
    - _المتطلبات: 1.2_

  - [x] 4.6 تطوير History Manager
    - إنشاء ملف `services/backend/internal/admin/rating/history.go`
    - تطبيق دالة SaveRatingHistory() لحفظ التاريخ
    - تطبيق دالة GetRatingHistory() مع فلترة حسب الفترة الزمنية
    - تطبيق دالة CleanupOldRecords() لحذف السجلات الأقدم من 12 شهر
    - _المتطلبات: 1.6_
  
  - [ ]* 4.7 كتابة Unit Tests لـ Rating Engine
    - إنشاء ملف `services/backend/internal/admin/rating/calculator_test.go`
    - اختبار حالات محددة: rating = 0, rating = 100, rating = 75.5
    - اختبار edge cases: zero messages, negative values
    - اختبار error conditions: invalid weights, missing data
    - _المتطلبات: 1.1, 1.4_

- [x] 5. تطوير Performance Metrics System
  - [x] 5.1 إنشاء Metrics Repository
    - إنشاء ملف `services/backend/internal/admin/metrics/repository.go`
    - تطبيق دالة SaveMetrics() لحفظ المقاييس
    - تطبيق دالة GetMetrics() مع فلترة حسب period_type والفترة الزمنية
    - تطبيق دالة AggregateMetrics() لتجميع البيانات
    - استخدام prepared statements للأداء
    - _المتطلبات: 4.1, 4.2, 4.6_
  
  - [x] 5.2 تطوير Metrics Aggregator
    - إنشاء ملف `services/backend/internal/admin/metrics/aggregator.go`
    - تطبيق دالة AggregateHourly() للتجميع الساعي
    - تطبيق دالة AggregateDaily() للتجميع اليومي
    - تطبيق دالة AggregateMonthly() للتجميع الشهري
    - استخدام cron jobs للتشغيل التلقائي
    - _المتطلبات: 4.2_
  
  - [ ]* 5.3 كتابة Property Tests للـ Metrics
    - **Property 16: Success Rate Calculation**
    - **Property 17: Volume Aggregation**
    - **Property 18: Payment Compliance Calculation**
    - إنشاء ملف `services/backend/internal/admin/metrics/metrics_property_test.go`
    - التحقق من صحة الحسابات لجميع السيناريوهات
    - _يتحقق من: المتطلبات 4.1, 4.2, 4.3_
  
  - [x] 5.4 تطوير Threshold Monitor
    - إنشاء ملف `services/backend/internal/admin/metrics/threshold.go`
    - تطبيق دالة CheckThresholds() للتحقق من العتبات
    - تطبيق دالة TriggerAlert() عند تجاوز العتبات
    - التكامل مع Alert System
    - _المتطلبات: 4.8_

- [x] 6. نقطة فحص - اختبار نظام التقييم
  - تشغيل جميع الاختبارات والتأكد من نجاحها
  - اختبار حساب التقييم لشركات تجريبية
  - التحقق من حفظ التاريخ بشكل صحيح
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 3: تطوير Backend - مراقبة السيرفرات والتوجيه

- [x] 7. تطوير Server Health Monitor
  - [x] 7.1 إنشاء Health Checker
    - إنشاء ملف `services/backend/internal/admin/health/checker.go`
    - تطبيق دالة CheckServerHealth() لفحص صحة السيرفر
    - تطبيق HTTP health check مع timeout
    - قياس response_time_ms و error_rate
    - تشغيل الفحص كل 30 ثانية باستخدام ticker
    - _المتطلبات: 5.1, 5.2, 5.3_
  
  - [x] 7.2 تطوير Status Classifier
    - إنشاء ملف `services/backend/internal/admin/health/classifier.go`
    - تطبيق دالة ClassifyHealthStatus() بناءً على المعايير:
      - Healthy: response_time < 200ms AND error_rate < 1%
      - Degraded: response_time 200-500ms OR error_rate 1-5%
      - Unhealthy: response_time > 500ms OR error_rate > 5%
    - _المتطلبات: 5.4_
  
  - [ ]* 7.3 كتابة Property Tests للـ Health Monitor
    - **Property 21: Server Health Classification**
    - **Property 22: Health History Retention**
    - إنشاء ملف `services/backend/internal/admin/health/health_property_test.go`
    - التحقق من صحة التصنيف لجميع القيم الممكنة
    - _يتحقق من: المتطلبات 5.4, 5.7_
  
  - [x] 7.4 تطوير Alert Trigger
    - إنشاء ملف `services/backend/internal/admin/health/alert.go`
    - تطبيق دالة OnStatusChange() لإطلاق التنبيهات
    - إرسال تنبيه فوري عند تغيير الحالة
    - التكامل مع Alert System
    - _المتطلبات: 2.6_
  
  - [x] 7.5 تطوير History Logger
    - إنشاء ملف `services/backend/internal/admin/health/logger.go`
    - تطبيق دالة LogHealthCheck() لحفظ نتائج الفحص
    - تطبيق دالة GetHealthHistory() للاستعلام عن التاريخ
    - تطبيق دالة CleanupOldLogs() لحذف السجلات الأقدم من 30 يوم
    - _المتطلبات: 5.7_

- [x] 8. تطوير Dedicated Server Routing
  - [x] 8.1 إنشاء Routing Configuration Manager
    - إنشاء ملف `services/backend/internal/admin/routing/config.go`
    - تطبيق دالة GetRoutingConfig() للحصول على إعدادات التوجيه
    - تطبيق دالة UpdateRoutingConfig() لتحديث الإعدادات
    - التحقق من صحة البيانات (validation)
    - _المتطلبات: 2.1, 2.2_
  
  - [x] 8.2 تطوير Routing Engine
    - إنشاء ملف `services/backend/internal/admin/routing/engine.go`
    - تطبيق دالة RouteMessage() للتوجيه الذكي:
      - فحص نوع الشركة (dedicated vs shared)
      - فحص صحة السيرفر المخصص
      - التوجيه إلى السيرفر المخصص أو إرجاع خطأ
    - تطبيق سياسة "No Fallback" (عدم التحويل التلقائي)
    - _المتطلبات: 2.1, 2.2, 2.4_
  
  - [ ]* 8.3 كتابة Property Tests للـ Routing
    - **Property 5: Dedicated Server Routing**
    - **Property 6: Company Type Detection**
    - **Property 7: Routing Isolation**
    - **Property 8: Unavailable Server Error Handling**
    - إنشاء ملف `services/backend/internal/admin/routing/routing_property_test.go`
    - _يتحقق من: المتطلبات 2.1, 2.2, 2.3, 2.4_
  
  - [x] 8.4 تطوير Metrics Tracker
    - إنشاء ملف `services/backend/internal/admin/routing/metrics.go`
    - تطبيق دالة TrackRoutingMetrics() لتتبع المقاييس
    - تحديث: total_requests, successful_requests, failed_requests, average_latency_ms
    - استخدام atomic operations للـ concurrency safety
    - _المتطلبات: 2.8_
  
  - [ ]* 8.5 كتابة Unit Tests للـ Routing Engine
    - إنشاء ملف `services/backend/internal/admin/routing/engine_test.go`
    - اختبار سيناريوهات محددة: dedicated server available, unavailable, shared routing
    - اختبار error handling
    - _المتطلبات: 2.1, 2.4_

- [x] 9. نقطة فحص - اختبار التوجيه والمراقبة
  - تشغيل جميع الاختبارات والتأكد من نجاحها
  - اختبار health checks على سيرفرات تجريبية
  - اختبار routing لشركات مع سيرفرات مخصصة
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 4: تطوير Backend - نظام توزيع الرسائل

- [x] 10. تطوير Message Distribution System
  - [x] 10.1 إنشاء Distribution Strategy Interface
    - إنشاء ملف `services/backend/internal/admin/distribution/types.go`
    - تعريف interface: DistributionStrategy مع دالة SelectDevice()
    - تعريف structs: Device, DevicePool, DistributionMetrics
    - _المتطلبات: 3.1_
  
  - [x] 10.2 تطوير Round Robin Strategy
    - إنشاء ملف `services/backend/internal/admin/distribution/round_robin.go`
    - تطبيق RoundRobinStrategy مع thread-safe counter
    - استخدام mutex للحماية من race conditions
    - _المتطلبات: 3.2_
  
  - [x] 10.3 تطوير Load Balancing Strategy
    - إنشاء ملف `services/backend/internal/admin/distribution/load_balancing.go`
    - تطبيق LoadBalancingStrategy
    - اختيار الجهاز بأقل current_load
    - _المتطلبات: 3.3_
  
  - [x] 10.4 تطوير Priority-Based Strategy
    - إنشاء ملف `services/backend/internal/admin/distribution/priority.go`
    - تطبيق PriorityStrategy
    - الترتيب حسب priority (تنازلي) ثم load (تصاعدي)
    - _المتطلبات: 3.4_
  
  - [x] 10.5 تطوير Custom Rules Strategy
    - إنشاء ملف `services/backend/internal/admin/distribution/custom.go`
    - تطبيق CustomRulesStrategy
    - دعم قواعد مخصصة من JSONB
    - _المتطلبات: 3.5_
  
  - [ ]* 10.6 كتابة Property Tests للـ Distribution
    - **Property 10: Distribution Strategy Execution**
    - **Property 11: Round Robin Fairness**
    - **Property 12: Load Balancing Selection**
    - **Property 13: Priority-Based Selection**
    - إنشاء ملف `services/backend/internal/admin/distribution/distribution_property_test.go`
    - اختبار توزيع 1000 رسالة والتحقق من العدالة (5% variance)
    - _يتحقق من: المتطلبات 3.1, 3.2, 3.3, 3.4, 3.10_
  
  - [x] 10.7 تطوير Device Failure Handler
    - إنشاء ملف `services/backend/internal/admin/distribution/failure.go`
    - تطبيق DeviceFailureHandler مع excluded devices map
    - استبعاد الجهاز لمدة 5 دقائق عند الفشل
    - إعادة تضمين الجهاز تلقائياً بعد انتهاء المدة
    - _المتطلبات: 3.8_
  
  - [x] 10.8 تطوير Device Pool Manager
    - إنشاء ملف `services/backend/internal/admin/distribution/pool.go`
    - تطبيق دالة GetAvailableDevices() لجلب الأجهزة المتاحة
    - تطبيق دالة UpdateDeviceLoad() لتحديث الحمل
    - تطبيق دالة UpdateDeviceMetrics() لتحديث المقاييس
    - _المتطلبات: 3.7_

  - [x] 10.9 تطوير Strategy Selector
    - إنشاء ملف `services/backend/internal/admin/distribution/selector.go`
    - تطبيق دالة GetStrategy() لاختيار الاستراتيجية المناسبة
    - تطبيق factory pattern لإنشاء الاستراتيجيات
    - _المتطلبات: 3.1, 3.6_
  
  - [ ]* 10.10 كتابة Unit Tests للـ Distribution System
    - إنشاء ملف `services/backend/internal/admin/distribution/strategies_test.go`
    - اختبار كل استراتيجية بشكل منفصل
    - اختبار edge cases: empty device pool, all devices excluded
    - _المتطلبات: 3.1-3.5_

- [x] 11. نقطة فحص - اختبار نظام التوزيع
  - تشغيل جميع الاختبارات والتأكد من نجاحها
  - اختبار توزيع رسائل على أجهزة متعددة
  - التحقق من عدالة التوزيع في Round Robin
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 5: تطوير Backend - نظام Signal/Push

- [x] 12. تطوير Timing Monitor
  - [x] 12.1 إنشاء Timing Monitor
    - إنشاء ملف `services/backend/internal/admin/signalpush/timing.go`
    - تطبيق دالة CheckMessage() للتحقق من التوقيت
    - استخدام Redis لتخزين آخر وقت إرسال
    - حساب الفجوة الزمنية بين الرسائل
    - تسجيل الانتهاكات في قاعدة البيانات
    - _المتطلبات: 2.1, 2.2_
  
  - [x] 12.2 تطوير Statistical Analyzer
    - إنشاء ملف `services/backend/internal/admin/signalpush/analyzer.go`
    - تطبيق دالة UpdateStatistics() لحساب الإحصائيات:
      - Current MPM (Messages Per Minute)
      - Daily/Weekly Average MPM
      - Daily/Weekly Peak MPM
      - Standard Deviation
    - حفظ الإحصائيات في tenant_statistics
    - _المتطلبات: 2.1_
  
  - [x] 12.3 تطوير Anomaly Detector
    - إنشاء ملف `services/backend/internal/admin/signalpush/anomaly.go`
    - تطبيق دالة DetectAnomaly() بناءً على:
      - Z-Score calculation
      - Peak Ratio calculation
      - Violation Rate
    - تصنيف مستوى الشذوذ: Minor, Moderate, Severe
    - _المتطلبات: 2.1_
  
  - [ ]* 12.4 كتابة Property Tests للـ Timing System
    - إنشاء ملف `services/backend/internal/admin/signalpush/timing_property_test.go`
    - التحقق من دقة حساب الفجوات الزمنية
    - التحقق من صحة تصنيف الانتهاكات
    - _المتطلبات: 2.1, 2.2_

- [x] 13. تطوير Auto-Enforcement System
  - [x] 13.1 إنشاء Auto-Enforcement Engine
    - إنشاء ملف `services/backend/internal/admin/signalpush/enforcement.go`
    - تطبيق دالة EnforceAction() بناءً على مستوى الشذوذ:
      - Minor: Log only
      - Moderate: Send warning
      - Severe: Force Push mode
    - تسجيل الإجراءات في enforcement_actions
    - _المتطلبات: 2.1_
  
  - [x] 13.2 تطوير Mode Switcher
    - إنشاء ملف `services/backend/internal/admin/signalpush/mode.go`
    - تطبيق دالة ForcePushMode() لتبديل الوضع
    - تطبيق دالة RestoreSignalMode() لاستعادة وضع Signal
    - تحديث tenant_timing_config
    - _المتطلبات: 2.1_
  
  - [x] 13.3 تطوير Notification Service Integration
    - إنشاء ملف `services/backend/internal/admin/signalpush/notify.go`
    - تطبيق دالة NotifyCompany() لإرسال تنبيه للشركة
    - تطبيق دالة NotifyAdmins() لإرسال تنبيه للمسؤولين
    - التكامل مع notification service الموجود
    - _المتطلبات: 6.1, 6.5_
  
  - [ ]* 13.4 كتابة Unit Tests للـ Enforcement
    - إنشاء ملف `services/backend/internal/admin/signalpush/enforcement_test.go`
    - اختبار سيناريوهات مختلفة لمستويات الشذوذ
    - اختبار تبديل الأوضاع
    - _المتطلبات: 2.1_

- [x] 14. تطوير Adaptive Sending System
  - [x] 14.1 إنشاء Server Load Monitor
    - إنشاء ملف `services/backend/internal/admin/signalpush/load.go`
    - تطبيق دالة GetCurrentLoad() لقياس CPU و Memory
    - تطبيق دالة IsOverloaded() للتحقق من الحمل الزائد
    - استخدام system metrics libraries
    - _المتطلبات: 12.3_
  
  - [x] 14.2 تطوير Adaptive Sender
    - إنشاء ملف `services/backend/internal/admin/signalpush/sender.go`
    - تطبيق دالة AdjustSendingRate() لتعديل معدل الإرسال:
      - High load (CPU > 80%): تقليل 50%
      - Low load (CPU < 50%): زيادة 100%
      - Normal load: الحفاظ على المعدل
    - تطبيق دالة SendBatch() لإرسال دفعات
    - _المتطلبات: 12.3_
  
  - [x] 14.3 تطوير Message Queue Manager
    - إنشاء ملف `services/backend/internal/admin/signalpush/queue.go`
    - تطبيق دالة EnqueueMessage() لإضافة رسالة للطابور
    - تطبيق دالة DequeueMessages() لجلب دفعة من الرسائل
    - تطبيق دالة UpdateMessageStatus() لتحديث حالة الرسالة
    - استخدام PostgreSQL كـ persistent queue
    - _المتطلبات: 2.1_

- [x] 15. تطوير Delivery Report Processor
  - [x] 15.1 إنشاء Batch Processor
    - إنشاء ملف `services/backend/internal/admin/signalpush/delivery.go`
    - تطبيق دالة ProcessBatch() لمعالجة تقارير التسليم بالدفعات
    - معالجة 100 تقرير في transaction واحدة
    - نقل الرسائل من waiting_messages إلى delivered_messages
    - تحديث حالة الرسالة
    - معالجة الاسترداد (refund) في حالة الفشل
    - _المتطلبات: 2.1_
  
  - [x] 15.2 تطوير Worker Pool
    - إنشاء ملف `services/backend/internal/admin/signalpush/workers.go`
    - تطبيق worker pool مع 10 workers
    - استخدام buffered channel بحجم 10000
    - تطبيق periodic flush كل 5 ثواني
    - _المتطلبات: 12.3_
  
  - [ ]* 15.3 كتابة Unit Tests للـ Delivery Processor
    - إنشاء ملف `services/backend/internal/admin/signalpush/delivery_test.go`
    - اختبار معالجة الدفعات
    - اختبار الاسترداد عند الفشل
    - _المتطلبات: 2.1_

- [x] 16. نقطة فحص - اختبار نظام Signal/Push
  - تشغيل جميع الاختبارات والتأكد من نجاحها
  - اختبار كشف الانتهاكات الزمنية
  - اختبار التبديل التلقائي لوضع Push
  - اختبار معالجة تقارير التسليم
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 6: تطوير Backend - نظام التنبيهات والـ APIs

- [x] 17. تطوير Alert System
  - [x] 17.1 إنشاء Alert Manager
    - إنشاء ملف `services/backend/internal/admin/alerts/manager.go`
    - تطبيق دالة CreateAlert() لإنشاء تنبيه جديد
    - تطبيق دالة GetAlerts() مع فلترة حسب severity, type, status
    - تطبيق دالة AcknowledgeAlert() للإقرار بالتنبيه
    - تطبيق دالة ResolveAlert() لحل التنبيه
    - _المتطلبات: 6.1-6.4, 6.9_
  
  - [x] 17.2 تطوير Alert Deduplication
    - إنشاء ملف `services/backend/internal/admin/alerts/dedup.go`
    - تطبيق دالة CheckDuplicate() باستخدام dedup_key
    - منع التنبيهات المكررة خلال 5 دقائق
    - _المتطلبات: 6.8_
  
  - [x] 17.3 تطوير Multi-Channel Notifier
    - إنشاء ملف `services/backend/internal/admin/alerts/notifier.go`
    - تطبيق دالة SendAlert() للإرسال عبر قنوات متعددة
    - دعم: email, SMS, in-app notifications
    - التكامل مع notification service الموجود
    - _المتطلبات: 6.5_
  
  - [ ]* 17.4 كتابة Property Tests للـ Alert System
    - **Property 23: Event-Driven Alert Generation**
    - **Property 24: Threshold-Based Alert Generation**
    - **Property 26: Alert Deduplication**
    - إنشاء ملف `services/backend/internal/admin/alerts/alerts_property_test.go`
    - _يتحقق من: المتطلبات 6.1-6.4, 6.8_

- [x] 18. تطوير Admin API Endpoints
  - [x] 18.1 إنشاء Company Rating APIs
    - إنشاء ملف `services/backend/internal/api/admin_rating_handler.go`
    - تطبيق GET /api/admin/companies/{tenant_id}/rating
    - تطبيق GET /api/admin/companies/ratings (مع فلترة)
    - تطبيق GET /api/admin/companies/{tenant_id}/rating/history
    - إضافة validation و error handling
    - _المتطلبات: 8.1_
  
  - [x] 18.2 إنشاء Performance Metrics APIs
    - إنشاء ملف `services/backend/internal/api/admin_metrics_handler.go`
    - تطبيق GET /api/admin/companies/{tenant_id}/metrics
    - دعم query parameters: period_type, period_start, period_end
    - _المتطلبات: 8.1_
  
  - [x] 18.3 إنشاء Routing Configuration APIs
    - إنشاء ملف `services/backend/internal/api/admin_routing_handler.go`
    - تطبيق GET /api/admin/routing/{tenant_id}
    - تطبيق PUT /api/admin/routing/{tenant_id}
    - تطبيق GET /api/admin/routing/{tenant_id}/metrics
    - _المتطلبات: 8.2_
  
  - [x] 18.4 إنشاء Distribution Strategy APIs
    - إنشاء ملف `services/backend/internal/api/admin_distribution_handler.go`
    - تطبيق GET /api/admin/distribution/{tenant_id}
    - تطبيق PUT /api/admin/distribution/{tenant_id}
    - تطبيق GET /api/admin/distribution/{tenant_id}/devices
    - تطبيق GET /api/admin/distribution/{tenant_id}/metrics
    - _المتطلبات: 8.3_
  
  - [x] 18.5 إنشاء Server Health APIs
    - إنشاء ملف `services/backend/internal/api/admin_health_handler.go`
    - تطبيق GET /api/admin/servers
    - تطبيق GET /api/admin/servers/{server_id}
    - تطبيق GET /api/admin/servers/{server_id}/health/history
    - _المتطلبات: 8.4_
  
  - [x] 18.6 إنشاء Alert Management APIs
    - إنشاء ملف `services/backend/internal/api/admin_alerts_handler.go`
    - تطبيق GET /api/admin/alerts (مع فلترة)
    - تطبيق GET /api/admin/alerts/{alert_id}
    - تطبيق PUT /api/admin/alerts/{alert_id}/acknowledge
    - تطبيق PUT /api/admin/alerts/{alert_id}/resolve
    - تطبيق POST /api/admin/alerts/config
    - _المتطلبات: 8.5_
  
  - [x] 18.7 إنشاء Signal/Push System APIs
    - إنشاء ملف `services/backend/internal/api/admin_signalpush_handler.go`
    - تطبيق GET /api/admin/signalpush/{tenant_id}/config
    - تطبيق PUT /api/admin/signalpush/{tenant_id}/config
    - تطبيق GET /api/admin/signalpush/{tenant_id}/statistics
    - تطبيق GET /api/admin/signalpush/{tenant_id}/violations
    - تطبيق POST /api/admin/signalpush/{tenant_id}/force-mode
    - _المتطلبات: 8.1_
  
  - [x] 18.8 إنشاء Dashboard Statistics API
    - إنشاء ملف `services/backend/internal/api/admin_dashboard_handler.go`
    - تطبيق GET /api/admin/dashboard/overview
    - تجميع إحصائيات من جميع الأنظمة
    - _المتطلبات: 7.1_
  
  - [ ]* 18.9 كتابة Unit Tests للـ API Handlers
    - إنشاء ملفات اختبار لجميع handlers
    - اختبار authentication و authorization
    - اختبار error handling
    - _المتطلبات: 8.1-8.10_

- [x] 19. نقطة فحص - اختبار APIs
  - تشغيل جميع الاختبارات والتأكد من نجاحها
  - اختبار جميع endpoints باستخدام Postman أو curl
  - التحقق من authentication و rate limiting
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 7: تطوير Frontend - واجهة الإدارة (React + TypeScript)

- [-] 20. إعداد مشروع Frontend
  - [x] 20.1 إنشاء مشروع React
    - إنشاء مجلد `apps/admin-web-v2` (واجهة جديدة محسّنة)
    - تهيئة React 18+ مع TypeScript
    - تثبيت المكتبات: React Query, Zustand, Tailwind CSS, shadcn/ui, Recharts
    - إعداد i18next للترجمة مع دعم RTL
    - _المتطلبات: 7.1, 7.7, 7.9_
  
  - [ ] 20.2 إعداد البنية الأساسية
    - إنشاء هيكل المجلدات: components, pages, hooks, services, types
    - إعداد API client مع axios
    - إعداد React Query configuration
    - إعداد Zustand stores
    - إعداد routing مع React Router
    - _المتطلبات: 7.1_
  
  - [ ] 20.3 إعداد التصميم والـ UI
    - إعداد Tailwind CSS configuration
    - تثبيت shadcn/ui components
    - إنشاء theme configuration (colors, fonts, spacing)
    - إنشاء مكونات UI أساسية: Button, Card, Table, Modal, Alert
    - _المتطلبات: 7.7_

- [-] 21. تطوير صفحة Overview (نظرة عامة)
  - [ ] 21.1 إنشاء Dashboard Layout
    - إنشاء ملف `apps/admin-web-v2/src/components/Layout/DashboardLayout.tsx`
    - تطبيق sidebar navigation
    - تطبيق header مع user menu
    - تطبيق breadcrumbs
    - _المتطلبات: 7.1_
  
  - [x] 21.2 إنشاء Overview Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/Overview.tsx`
    - تطبيق Statistics Cards (إجمالي الشركات، الرسائل اليوم، معدل النجاح)
    - تطبيق Rating Distribution Chart (مخطط توزيع التقييمات)
    - تطبيق Active Alerts Panel (لوحة التنبيهات النشطة)
    - تطبيق Server Health Summary (ملخص صحة السيرفرات)
    - _المتطلبات: 7.1_
  
  - [ ] 21.3 إنشاء Custom Hooks
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useDashboardStats.ts`
    - تطبيق useQuery للحصول على الإحصائيات
    - تطبيق auto-refresh كل 30 ثانية
    - _المتطلبات: 4.5_

- [-] 22. تطوير صفحة Companies (الشركات)
  - [x] 22.1 إنشاء Companies List Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/Companies/CompaniesList.tsx`
    - تطبيق جدول الشركات مع pagination
    - تطبيق search bar للبحث بالاسم
    - تطبيق filters: rating category, date range
    - تطبيق sorting: by rating, by name, by messages
    - _المتطلبات: 7.2_
  
  - [x] 22.2 إنشاء Company Details Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/Companies/CompanyDetails.tsx`
    - تطبيق Company Info Header (معلومات الشركة)
    - تطبيق Rating Card مع تفاصيل المكونات
    - تطبيق Rating History Chart (مخطط تاريخ التقييم)
    - تطبيق Performance Metrics Tabs (مقاييس الأداء)
    - _المتطلبات: 7.3_
  
  - [ ] 22.3 إنشاء Rating Components
    - إنشاء ملف `apps/admin-web-v2/src/components/Rating/RatingBadge.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/Rating/RatingScore.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/Rating/RatingHistoryChart.tsx`
    - استخدام Recharts للمخططات
    - _المتطلبات: 1.3, 1.6_
  
  - [ ] 22.4 إنشاء Custom Hooks للشركات
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useCompanies.ts`
    - تطبيق useCompanyRating, useCompanyMetrics, useCompanyHistory
    - _المتطلبات: 1.1-1.7_

- [-] 23. تطوير صفحة Servers (السيرفرات)
  - [x] 23.1 إنشاء Servers List Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/Servers/ServersList.tsx`
    - تطبيق جدول السيرفرات
    - تطبيق Health Status Indicators (مؤشرات الحالة الصحية)
    - تطبيق filters: health status, company
    - تطبيق real-time updates باستخدام WebSocket
    - _المتطلبات: 7.4_
  
  - [x] 23.2 إنشاء Server Details Modal
    - إنشاء ملف `apps/admin-web-v2/src/components/Servers/ServerDetailsModal.tsx`
    - تطبيق Server Info
    - تطبيق Health Metrics (response time, error rate)
    - تطبيق Health History Chart
    - _المتطلبات: 5.1-5.7_
  
  - [x] 23.3 إنشاء Health Monitor Components
    - إنشاء ملف `apps/admin-web-v2/src/components/Servers/HealthStatusIndicator.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/Servers/HealthHistoryChart.tsx`
    - تطبيق color coding: green (healthy), yellow (degraded), red (unhealthy)
    - _المتطلبات: 5.4, 5.6_
  
  - [x] 23.4 إنشاء WebSocket Hook
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useWebSocket.ts`
    - تطبيق WebSocket connection management
    - تطبيق auto-reconnect
    - _المتطلبات: 7.4_

- [x] 24. تطوير صفحة Distribution (التوزيع)
  - [x] 24.1 إنشاء Distribution Config Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/Distribution/DistributionConfig.tsx`
    - تطبيق Strategy Selector (اختيار الاستراتيجية)
    - تطبيق Strategy Configuration Form
    - تطبيق Device Pool Manager
    - _المتطلبات: 7.5_
  
  - [x] 24.2 إنشاء Device Management Components
    - إنشاء ملف `apps/admin-web-v2/src/components/Distribution/DeviceList.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/Distribution/DeviceCard.tsx`
    - تطبيق device status indicators
    - تطبيق device metrics display
    - _المتطلبات: 3.7, 3.9_
  
  - [x] 24.3 إنشاء Distribution Metrics Components
    - إنشاء ملف `apps/admin-web-v2/src/components/Distribution/DistributionMetrics.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/Distribution/LoadVisualization.tsx`
    - تطبيق مخططات توزيع الرسائل
    - تطبيق load balancing visualization
    - _المتطلبات: 3.7_
  
  - [x] 24.4 إنشاء Custom Hooks للتوزيع
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useDistribution.ts`
    - تطبيق useDistributionStrategy, useDevices, useDistributionMetrics
    - _المتطلبات: 3.1-3.10_

- [-] 25. تطوير صفحة Alerts (التنبيهات)
  - [x] 25.1 إنشاء Alerts Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/Alerts/AlertsList.tsx`
    - تطبيق Active Alerts List
    - تطبيق Alert History
    - تطبيق filters: severity, type, status, date range
    - _المتطلبات: 7.6_
  
  - [x] 25.2 إنشاء Alert Components
    - إنشاء ملف `apps/admin-web-v2/src/components/Alerts/AlertCard.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/Alerts/AlertDetailsModal.tsx`
    - تطبيق severity badges (critical, warning, info)
    - تطبيق acknowledge و resolve actions
    - _المتطلبات: 6.1-6.9_
  
  - [x] 25.3 إنشاء Alert Configuration
    - إنشاء ملف `apps/admin-web-v2/src/pages/Alerts/AlertConfig.tsx`
    - تطبيق threshold configuration
    - تطبيق notification channels configuration
    - _المتطلبات: 6.6_
  
  - [x] 25.4 إنشاء Custom Hooks للتنبيهات
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useAlerts.ts`
    - تطبيق useAlerts, useAcknowledgeAlert, useResolveAlert
    - _المتطلبات: 6.1-6.9_

- [x] 26. تطوير صفحة Signal/Push Monitor
  - [x] 26.1 إنشاء Signal/Push Monitor Page
    - إنشاء ملف `apps/admin-web-v2/src/pages/SignalPush/Monitor.tsx`
    - تطبيق Timing Statistics Cards
    - تطبيق Violation Trends Chart
    - تطبيق Anomaly Detection Status
    - تطبيق Mode Override Controls
    - _المتطلبات: 8.1_
  
  - [x] 26.2 إنشاء Timing Components
    - إنشاء ملف `apps/admin-web-v2/src/components/SignalPush/TimingStats.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/SignalPush/ViolationChart.tsx`
    - إنشاء ملف `apps/admin-web-v2/src/components/SignalPush/AnomalyIndicator.tsx`
    - _المتطلبات: 2.1_
  
  - [x] 26.3 إنشاء Mode Control Component
    - إنشاء ملف `apps/admin-web-v2/src/components/SignalPush/ModeControl.tsx`
    - تطبيق force Signal/Push mode buttons
    - تطبيق confirmation dialogs
    - _المتطلبات: 2.1_
  
  - [x] 26.4 إنشاء Custom Hooks للـ Signal/Push
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useSignalPush.ts`
    - تطبيق useTimingConfig, useTimingStats, useViolations, useForceMod
    - _المتطلبات: 2.1_

- [ ] 27. تطوير Routing Configuration في Company Details
  - [x] 27.1 إنشاء Routing Config Component
    - إنشاء ملف `apps/admin-web-v2/src/components/Companies/RoutingConfig.tsx`
    - تطبيق routing type selector (shared/dedicated)
    - تطبيق dedicated server configuration form
    - تطبيق routing metrics display
    - _المتطلبات: 2.1-2.8_
  
  - [x] 27.2 إنشاء Custom Hooks للتوجيه
    - إنشاء ملف `apps/admin-web-v2/src/hooks/useRouting.ts`
    - تطبيق useRoutingConfig, useUpdateRoutingConfig, useRoutingMetrics
    - _المتطلبات: 2.1-2.8_

- [ ] 28. إضافة الترجمة والـ RTL Support
  - [x] 28.1 إعداد ملفات الترجمة
    - إنشاء ملف `apps/admin-web-v2/src/i18n/locales/ar.json`
    - إنشاء ملف `apps/admin-web-v2/src/i18n/locales/en.json`
    - ترجمة جميع النصوص في الواجهة
    - _المتطلبات: 7.9_
  
  - [x] 28.2 إضافة RTL Support
    - إنشاء ملف `apps/admin-web-v2/src/styles/rtl.css`
    - تطبيق RTL styles للغة العربية
    - تطبيق Language Selector component
    - _المتطلبات: 7.9_

- [ ] 29. إضافة Responsive Design
  - [x] 29.1 تطبيق Responsive Layouts
    - تحديث جميع الصفحات لتكون responsive
    - استخدام Tailwind breakpoints: mobile, tablet, desktop
    - اختبار على أحجام شاشات مختلفة
    - _المتطلبات: 7.7_
  
  - [x] 29.2 تطبيق Mobile Navigation
    - إنشاء mobile sidebar menu
    - تطبيق hamburger menu
    - _المتطلبات: 7.7_

- [x] 30. نقطة فحص - اختبار Frontend
  - اختبار جميع الصفحات والمكونات
  - اختبار التفاعل مع Backend APIs
  - اختبار الترجمة والـ RTL
  - اختبار Responsive design
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 8: الاختبارات الشاملة

- [x] 31. اختبارات التكامل (Integration Tests)
  - [x] 31.1 كتابة Integration Tests للـ Rating System
    - إنشاء ملف `services/backend/internal/integration/rating_system_test.go`
    - اختبار end-to-end rating calculation
    - اختبار alert generation عند تغيير التقييم
    - _المتطلبات: 1.1-1.7_
  
  - [x] 31.2 كتابة Integration Tests للـ Routing
    - إنشاء ملف `services/backend/internal/integration/routing_test.go`
    - اختبار routing إلى dedicated servers
    - اختبار health monitoring integration
    - _المتطلبات: 2.1-2.8_
  
  - [x] 31.3 كتابة Integration Tests للـ Distribution
    - إنشاء ملف `services/backend/internal/integration/distribution_test.go`
    - اختبار توزيع الرسائل على أجهزة متعددة
    - اختبار failure handling
    - _المتطلبات: 3.1-3.10_
  
  - [x] 31.4 كتابة Integration Tests للـ Signal/Push
    - إنشاء ملف `services/backend/internal/integration/signalpush_test.go`
    - اختبار timing monitoring
    - اختبار auto-enforcement
    - اختبار adaptive sending
    - _المتطلبات: 2.1_

- [x] 32. اختبارات الأداء (Performance Tests)
  - [x] 32.1 إنشاء Load Tests
    - إنشاء ملف `performance-tests/admin-load-test.js` (k6)
    - اختبار 100 concurrent admin users
    - اختبار 100 API requests/second
    - قياس response times
    - _المتطلبات: 12.1, 12.6_
  
  - [x] 32.2 إنشاء Stress Tests
    - إنشاء ملف `performance-tests/admin-stress-test.js`
    - اختبار 1000 companies مع real-time metrics
    - اختبار 10,000 messages/minute distribution
    - _المتطلبات: 12.3, 12.4_
  
  - [x] 32.3 قياس Benchmarks
    - قياس rating calculation latency (< 10ms)
    - قياس routing decision latency (< 50ms at p95)
    - قياس API response time (< 200ms average)
    - _المتطلبات: 12.2, 12.6_

- [x] 33. اختبارات الأمان (Security Tests)
  - [x] 33.1 اختبار Authentication
    - اختبار API access بدون API key
    - اختبار invalid API keys
    - _المتطلبات: 8.6, 13.1_
  
  - [x] 33.2 اختبار Authorization
    - اختبار cross-tenant data access
    - اختبار role-based permissions
    - _المتطلبات: 13.2, 13.4_
  
  - [x] 33.3 اختبار Input Validation
    - اختبار SQL injection attempts
    - اختبار XSS attempts
    - _المتطلبات: 10.1, 13.8_

- [x] 34. نقطة فحص - مراجعة الاختبارات
  - تشغيل جميع الاختبارات والتأكد من نجاحها
  - مراجعة نتائج اختبارات الأداء
  - مراجعة نتائج اختبارات الأمان
  - إصلاح أي مشاكل مكتشفة
  - السؤال عن أي مشاكل أو استفسارات

### المرحلة 9: التوثيق والنشر

- [-] 35. كتابة التوثيق
  - [x] 35.1 كتابة API Documentation
    - إنشاء ملف `docs/ADMIN_API_REFERENCE.md`
    - توثيق جميع endpoints مع أمثلة
    - توثيق authentication و rate limiting
    - _المتطلبات: 8.1-8.10_
  
  - [x] 35.2 كتابة User Guide
    - إنشاء ملف `docs/ADMIN_USER_GUIDE.md`
    - شرح كيفية استخدام واجهة الإدارة
    - شرح جميع الميزات والصفحات
    - إضافة screenshots
    - _المتطلبات: 7.1-7.9_
  
  - [x] 35.3 كتابة Developer Guide
    - إنشاء ملف `docs/ADMIN_DEVELOPER_GUIDE.md`
    - شرح البنية المعمارية
    - شرح كيفية إضافة ميزات جديدة
    - شرح كيفية تشغيل الاختبارات
    - _المتطلبات: جميع المتطلبات_
  
  - [x] 35.4 كتابة Deployment Guide
    - إنشاء ملف `docs/ADMIN_DEPLOYMENT_GUIDE.md`
    - شرح متطلبات البنية التحتية
    - شرح خطوات النشر
    - شرح إعدادات المراقبة
    - _المتطلبات: 12.1-12.7_

- [x] 36. إعداد المراقبة (Monitoring)
  - [x] 36.1 إنشاء Prometheus Metrics
    - تصدير metrics للـ rating calculations
    - تصدير metrics للـ server health checks
    - تصدير metrics للـ message distributions
    - تصدير metrics للـ timing violations
    - تصدير metrics للـ alerts
    - _المتطلبات: 12.1-12.7_
  
  - [x] 36.2 إنشاء Grafana Dashboards
    - إنشاء ملف `grafana/dashboards/admin-system-dashboard.json`
    - تطبيق dashboard للـ company ratings
    - تطبيق dashboard للـ server health
    - تطبيق dashboard للـ distribution metrics
    - تطبيق dashboard للـ Signal/Push monitoring
    - _المتطلبات: 12.1-12.7_
  
  - [x] 36.3 إعداد Alert Rules
    - إنشاء ملف `prometheus/admin_alert_rules.yml`
    - تطبيق alerts للـ API response time
    - تطبيق alerts للـ database issues
    - تطبيق alerts للـ server health failures
    - _المتطلبات: 12.1-12.7_

- [x] 37. إعداد النشر (Deployment)
  - [x] 37.1 إنشاء Docker Images
    - إنشاء Dockerfile للـ backend
    - إنشاء Dockerfile للـ frontend
    - اختبار Docker images محلياً
    - _المتطلبات: 12.1-12.7_
  
  - [x] 37.2 إعداد Docker Compose
    - تحديث `docker-compose.yml`
    - إضافة admin backend service
    - إضافة admin frontend service
    - _المتطلبات: 12.1-12.7_
  
  - [x] 37.3 إعداد Database Migrations
    - التأكد من جميع migrations جاهزة
    - إنشاء migration rollback scripts
    - اختبار migrations على بيئة staging
    - _المتطلبات: 9.1-9.11_

- [x] 38. نقطة فحص نهائية - الاستعداد للنشر
  - مراجعة جميع المهام والتأكد من اكتمالها
  - تشغيل جميع الاختبارات على بيئة staging
  - مراجعة التوثيق والتأكد من اكتماله
  - مراجعة إعدادات المراقبة
  - الحصول على موافقة المستخدم للنشر

## ملاحظات

- المهام المميزة بـ `*` هي اختيارية ويمكن تخطيها للحصول على MVP أسرع
- كل مهمة تشير إلى المتطلبات التي تحققها لسهولة التتبع
- نقاط الفحص (Checkpoints) مهمة للتأكد من جودة العمل قبل الانتقال للمرحلة التالية
- Property Tests تغطي 40 خاصية لضمان صحة النظام
- Unit Tests تغطي حالات محددة وedge cases
- يجب تشغيل جميع الاختبارات قبل كل commit

## إحصائيات المشروع

- **عدد جداول قاعدة البيانات**: 15 جدول
- **عدد Property Tests**: 40 اختبار خاصية
- **عدد API Endpoints**: 30+ endpoint
- **عدد صفحات Frontend**: 7 صفحات رئيسية
- **عدد المكونات**: 50+ مكون React
- **التقنيات المستخدمة**:
  - Backend: Go (Golang)
  - Frontend: React 18+ with TypeScript
  - Database: PostgreSQL 15+
  - Cache: Redis 7+
  - Monitoring: Prometheus + Grafana
  - Testing: gopter (property tests), k6 (load tests)

## الخطوات التالية بعد إكمال المهام

1. نشر النظام على بيئة الإنتاج
2. مراقبة الأداء والاستقرار
3. جمع ملاحظات المستخدمين
4. التخطيط للتحسينات المستقبلية
