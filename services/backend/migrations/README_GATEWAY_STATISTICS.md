# Gateway Statistics Table Migration

## نظرة عامة

هذا الـ Migration ينشئ جدول `gateway_statistics` لتسجيل الإحصائيات اليومية للبوابات المخصصة في نظام التعرفات والتسعير.

## معلومات الـ Migration

- **رقم الـ Migration**: 025
- **اسم الملف**: `025_gateway_statistics_table.sql`
- **التاريخ**: 2024
- **المتطلبات المغطاة**: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7

## الهدف

تسجيل ومراقبة الإحصائيات اليومية لكل بوابة مخصصة، بما في ذلك:
- إحصائيات الرسائل (المرسلة، المسلّمة، الفاشلة، المعلقة)
- إحصائيات الأداء (أوقات الاستجابة، وقت التشغيل)
- إحصائيات مالية (التكلفة، الإيرادات، هامش الربح)

## بنية الجدول

### الحقول الأساسية

```sql
id SERIAL PRIMARY KEY
tenant_id UUID NOT NULL REFERENCES tenants(id)
gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id)
date DATE NOT NULL
```

### إحصائيات الرسائل

```sql
messages_sent INTEGER DEFAULT 0
messages_delivered INTEGER DEFAULT 0
messages_failed INTEGER DEFAULT 0
messages_pending INTEGER DEFAULT 0
```

### إحصائيات الأداء

```sql
avg_response_time_ms INTEGER
max_response_time_ms INTEGER
min_response_time_ms INTEGER
uptime_percentage DECIMAL(5, 2)
downtime_minutes INTEGER DEFAULT 0
```

### إحصائيات مالية

```sql
total_cost DECIMAL(10, 2) DEFAULT 0.00
total_revenue DECIMAL(10, 2) DEFAULT 0.00
profit_margin DECIMAL(10, 2) DEFAULT 0.00
```

## القيود (Constraints)

### 1. Unique Constraint

```sql
UNIQUE(gateway_id, date)
```

يضمن عدم تكرار الإحصائيات لنفس البوابة في نفس اليوم.

### 2. Foreign Key Constraint

```sql
FOREIGN KEY (tenant_id, gateway_id) 
    REFERENCES dedicated_gateways(tenant_id, id)
```

يضمن أن البوابة تنتمي للشركة الصحيحة.

### 3. Check Constraints

- **check_messages_non_negative**: جميع أعداد الرسائل يجب أن تكون >= 0
- **check_response_times_positive**: أوقات الاستجابة يجب أن تكون >= 0
- **check_uptime_percentage**: نسبة وقت التشغيل بين 0 و 100
- **check_downtime_non_negative**: وقت التوقف يجب أن يكون >= 0
- **check_messages_consistency**: messages_sent >= (messages_delivered + messages_failed)

## Indexes

### Indexes للعزل (Tenant Isolation)

```sql
idx_gateway_statistics_tenant (tenant_id)
idx_gateway_statistics_tenant_gateway (tenant_id, gateway_id)
idx_gateway_statistics_tenant_date (tenant_id, date DESC)
idx_gateway_statistics_tenant_gateway_date (tenant_id, gateway_id, date DESC)
```

### Indexes للأداء

```sql
idx_gateway_statistics_gateway_date (gateway_id, date DESC)
idx_gateway_statistics_date (date DESC)
idx_gateway_statistics_gateway (gateway_id)
```

## Row Level Security (RLS)

تم تفعيل RLS مع 4 Policies:

1. **tenant_isolation_policy**: للقراءة
2. **tenant_isolation_insert_policy**: للإدراج
3. **tenant_isolation_update_policy**: للتحديث
4. **tenant_isolation_delete_policy**: للحذف

جميع الـ Policies تضمن أن كل شركة ترى فقط إحصائياتها الخاصة.

## Triggers

### 1. update_gateway_statistics_updated_at

يحدث `updated_at` تلقائياً عند كل تحديث.

### 2. check_gateway_statistics_tenant_consistency

يتحقق من أن `gateway_id` ينتمي لنفس `tenant_id`.

### 3. calculate_gateway_statistics_profit_margin

يحسب `profit_margin` تلقائياً: (total_revenue - total_cost)

### 4. validate_gateway_statistics_consistency

يتحقق من:
- messages_sent >= messages_delivered + messages_failed
- يحذر إذا كان معدل النجاح < 50%

## Functions

### 1. aggregate_daily_gateway_statistics

```sql
aggregate_daily_gateway_statistics(p_gateway_id INTEGER, p_date DATE)
```

**الوظيفة**: حساب وتجميع الإحصائيات اليومية لبوابة محددة

**الاستخدام**:
```sql
SELECT aggregate_daily_gateway_statistics(1, '2024-01-15');
```

**ما تفعله**:
- تحسب عدد الرسائل من جدول `messages`
- تحسب أوقات الاستجابة من جدول `message_logs`
- تحسب وقت التشغيل من جدول `gateway_health_checks`
- تحسب الإحصائيات المالية من جدول `accounting_transactions`
- تدرج أو تحدث السجل في `gateway_statistics`

### 2. get_gateway_success_rate

```sql
get_gateway_success_rate(p_gateway_id INTEGER, p_date DATE) RETURNS DECIMAL(5, 2)
```

**الوظيفة**: حساب معدل النجاح لبوابة في تاريخ محدد

**الاستخدام**:
```sql
SELECT get_gateway_success_rate(1, '2024-01-15');
-- النتيجة: 95.50 (يعني 95.50% من الرسائل تم تسليمها بنجاح)
```

### 3. get_gateway_statistics_range

```sql
get_gateway_statistics_range(
    p_gateway_id INTEGER,
    p_start_date DATE,
    p_end_date DATE
)
```

**الوظيفة**: الحصول على إحصائيات مجمعة لفترة زمنية

**الاستخدام**:
```sql
SELECT * FROM get_gateway_statistics_range(1, '2024-01-01', '2024-01-31');
```

**النتيجة**:
```
total_messages_sent | total_messages_delivered | total_messages_failed | avg_success_rate | avg_response_time | avg_uptime_percentage | total_cost | total_revenue | total_profit
--------------------|--------------------------|----------------------|------------------|-------------------|----------------------|------------|---------------|-------------
50000               | 47500                    | 2500                 | 95.00            | 250               | 99.50                | 5000.00    | 7500.00       | 2500.00
```

### 4. cleanup_old_gateway_statistics

```sql
cleanup_old_gateway_statistics(p_retention_days INTEGER DEFAULT 365) RETURNS INTEGER
```

**الوظيفة**: تنظيف الإحصائيات القديمة

**الاستخدام**:
```sql
-- حذف الإحصائيات الأقدم من سنة
SELECT cleanup_old_gateway_statistics();

-- حذف الإحصائيات الأقدم من 90 يوم
SELECT cleanup_old_gateway_statistics(90);
```

## أمثلة الاستخدام

### 1. إدراج إحصائيات يدوياً

```sql
INSERT INTO gateway_statistics (
    tenant_id,
    gateway_id,
    date,
    messages_sent,
    messages_delivered,
    messages_failed,
    messages_pending,
    avg_response_time_ms,
    max_response_time_ms,
    min_response_time_ms,
    uptime_percentage,
    downtime_minutes,
    total_cost,
    total_revenue
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000',
    1,
    '2024-01-15',
    1000,
    950,
    50,
    0,
    200,
    500,
    50,
    99.50,
    7,
    100.00,
    150.00
);
```

### 2. الاستعلام عن إحصائيات بوابة محددة

```sql
-- إحصائيات اليوم
SELECT * FROM gateway_statistics
WHERE gateway_id = 1
  AND date = CURRENT_DATE;

-- إحصائيات آخر 7 أيام
SELECT * FROM gateway_statistics
WHERE gateway_id = 1
  AND date >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY date DESC;
```

### 3. حساب معدل النجاح

```sql
SELECT 
    date,
    messages_sent,
    messages_delivered,
    CASE 
        WHEN messages_sent > 0 THEN 
            ROUND((messages_delivered::DECIMAL / messages_sent::DECIMAL * 100), 2)
        ELSE 
            0.00
    END AS success_rate
FROM gateway_statistics
WHERE gateway_id = 1
  AND date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY date DESC;
```

### 4. مقارنة الأداء بين البوابات

```sql
SELECT 
    dg.name AS gateway_name,
    AVG(gs.uptime_percentage) AS avg_uptime,
    AVG(gs.avg_response_time_ms) AS avg_response_time,
    SUM(gs.messages_sent) AS total_messages,
    CASE 
        WHEN SUM(gs.messages_sent) > 0 THEN
            ROUND((SUM(gs.messages_delivered)::DECIMAL / SUM(gs.messages_sent)::DECIMAL * 100), 2)
        ELSE
            0.00
    END AS overall_success_rate
FROM gateway_statistics gs
JOIN dedicated_gateways dg ON gs.gateway_id = dg.id
WHERE gs.date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY dg.id, dg.name
ORDER BY overall_success_rate DESC;
```

### 5. تقرير الإيرادات الشهرية

```sql
SELECT 
    DATE_TRUNC('month', date) AS month,
    SUM(total_revenue) AS monthly_revenue,
    SUM(total_cost) AS monthly_cost,
    SUM(profit_margin) AS monthly_profit,
    CASE 
        WHEN SUM(total_cost) > 0 THEN
            ROUND((SUM(profit_margin) / SUM(total_cost) * 100), 2)
        ELSE
            0.00
    END AS profit_percentage
FROM gateway_statistics
WHERE gateway_id = 1
  AND date >= DATE_TRUNC('year', CURRENT_DATE)
GROUP BY DATE_TRUNC('month', date)
ORDER BY month DESC;
```

## Cron Jobs المقترحة

### 1. تجميع الإحصائيات اليومية

```sql
-- يُشغل يومياً في منتصف الليل
SELECT aggregate_daily_gateway_statistics(gateway_id, CURRENT_DATE - INTERVAL '1 day')
FROM dedicated_gateways
WHERE is_active = true;
```

### 2. تنظيف الإحصائيات القديمة

```sql
-- يُشغل أسبوعياً
SELECT cleanup_old_gateway_statistics(365);
```

### 3. إرسال تقارير أسبوعية

```sql
-- يُشغل كل يوم اثنين
SELECT 
    dg.name,
    dg.contact_email,
    get_gateway_statistics_range(
        dg.id,
        CURRENT_DATE - INTERVAL '7 days',
        CURRENT_DATE - INTERVAL '1 day'
    )
FROM dedicated_gateways dg
WHERE dg.is_active = true;
```

## التكامل مع Go

### مثال: حساب الإحصائيات اليومية

```go
// internal/monitoring/statistics.go

func (s *StatisticsService) AggregateDaily(ctx context.Context, gatewayID int, date time.Time) error {
    query := `SELECT aggregate_daily_gateway_statistics($1, $2)`
    
    _, err := s.db.ExecContext(ctx, query, gatewayID, date)
    if err != nil {
        return fmt.Errorf("failed to aggregate statistics: %w", err)
    }
    
    return nil
}
```

### مثال: الحصول على معدل النجاح

```go
func (s *StatisticsService) GetSuccessRate(ctx context.Context, gatewayID int, date time.Time) (float64, error) {
    var successRate float64
    
    query := `SELECT get_gateway_success_rate($1, $2)`
    
    err := s.db.QueryRowContext(ctx, query, gatewayID, date).Scan(&successRate)
    if err != nil {
        return 0, fmt.Errorf("failed to get success rate: %w", err)
    }
    
    return successRate, nil
}
```

### مثال: الحصول على إحصائيات فترة

```go
type StatisticsRange struct {
    TotalMessagesSent      int64
    TotalMessagesDelivered int64
    TotalMessagesFailed    int64
    AvgSuccessRate         float64
    AvgResponseTime        int
    AvgUptimePercentage    float64
    TotalCost              float64
    TotalRevenue           float64
    TotalProfit            float64
}

func (s *StatisticsService) GetRange(ctx context.Context, gatewayID int, startDate, endDate time.Time) (*StatisticsRange, error) {
    var stats StatisticsRange
    
    query := `SELECT * FROM get_gateway_statistics_range($1, $2, $3)`
    
    err := s.db.QueryRowContext(ctx, query, gatewayID, startDate, endDate).Scan(
        &stats.TotalMessagesSent,
        &stats.TotalMessagesDelivered,
        &stats.TotalMessagesFailed,
        &stats.AvgSuccessRate,
        &stats.AvgResponseTime,
        &stats.AvgUptimePercentage,
        &stats.TotalCost,
        &stats.TotalRevenue,
        &stats.TotalProfit,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to get statistics range: %w", err)
    }
    
    return &stats, nil
}
```

## الاختبار

### 1. اختبار إدراج إحصائيات

```sql
-- تعيين tenant_id
SET app.current_tenant = '123e4567-e89b-12d3-a456-426614174000';

-- إدراج إحصائيات
INSERT INTO gateway_statistics (
    tenant_id, gateway_id, date,
    messages_sent, messages_delivered, messages_failed
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000', 1, CURRENT_DATE,
    1000, 950, 50
);

-- التحقق
SELECT * FROM gateway_statistics WHERE gateway_id = 1 AND date = CURRENT_DATE;
```

### 2. اختبار Unique Constraint

```sql
-- يجب أن يفشل (تكرار gateway_id + date)
INSERT INTO gateway_statistics (
    tenant_id, gateway_id, date,
    messages_sent, messages_delivered, messages_failed
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000', 1, CURRENT_DATE,
    500, 475, 25
);
-- ERROR: duplicate key value violates unique constraint
```

### 3. اختبار Check Constraints

```sql
-- يجب أن يفشل (messages_sent < messages_delivered + messages_failed)
INSERT INTO gateway_statistics (
    tenant_id, gateway_id, date,
    messages_sent, messages_delivered, messages_failed
) VALUES (
    '123e4567-e89b-12d3-a456-426614174000', 2, CURRENT_DATE,
    100, 80, 30  -- 100 < 80 + 30
);
-- ERROR: Invalid message counts
```

### 4. اختبار RLS

```sql
-- تعيين tenant مختلف
SET app.current_tenant = '987fcdeb-51a2-43f7-b890-123456789abc';

-- محاولة قراءة إحصائيات tenant آخر
SELECT * FROM gateway_statistics WHERE gateway_id = 1;
-- النتيجة: 0 rows (RLS يمنع الوصول)
```

### 5. اختبار Functions

```sql
-- اختبار aggregate_daily_gateway_statistics
SELECT aggregate_daily_gateway_statistics(1, CURRENT_DATE - INTERVAL '1 day');

-- اختبار get_gateway_success_rate
SELECT get_gateway_success_rate(1, CURRENT_DATE);

-- اختبار get_gateway_statistics_range
SELECT * FROM get_gateway_statistics_range(1, CURRENT_DATE - INTERVAL '7 days', CURRENT_DATE);

-- اختبار cleanup_old_gateway_statistics
SELECT cleanup_old_gateway_statistics(30);
```

## الأمان

### 1. Row Level Security

- جميع الاستعلامات تُفلتر تلقائياً حسب `tenant_id`
- لا يمكن لأي شركة رؤية إحصائيات شركة أخرى

### 2. Tenant Consistency

- Trigger يتحقق من أن `gateway_id` ينتمي لنفس `tenant_id`
- يمنع إدراج إحصائيات لبوابة من tenant آخر

### 3. Data Validation

- Check Constraints تضمن صحة البيانات
- Trigger يتحقق من consistency الرسائل

## الأداء

### 1. Indexes

- جميع الـ Indexes تبدأ بـ `tenant_id` للعزل
- Indexes على `date DESC` للاستعلامات الزمنية
- Composite Indexes للاستعلامات المعقدة

### 2. Partitioning (مقترح للمستقبل)

```sql
-- يمكن تطبيق Partitioning حسب الشهر للأداء الأفضل
CREATE TABLE gateway_statistics_2024_01 PARTITION OF gateway_statistics
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

### 3. Archiving

- Function `cleanup_old_gateway_statistics` لحذف البيانات القديمة
- يُنصح بالاحتفاظ بالبيانات لمدة سنة على الأقل

## الصيانة

### 1. تنظيف دوري

```sql
-- حذف الإحصائيات الأقدم من سنة (شهرياً)
SELECT cleanup_old_gateway_statistics(365);
```

### 2. إعادة بناء Indexes

```sql
-- إعادة بناء Indexes (سنوياً)
REINDEX TABLE gateway_statistics;
```

### 3. تحليل الجدول

```sql
-- تحديث إحصائيات الجدول (أسبوعياً)
ANALYZE gateway_statistics;
```

## Rollback

لإلغاء هذا الـ Migration:

```sql
DROP FUNCTION IF EXISTS cleanup_old_gateway_statistics(INTEGER);
DROP FUNCTION IF EXISTS get_gateway_statistics_range(INTEGER, DATE, DATE);
DROP FUNCTION IF EXISTS get_gateway_success_rate(INTEGER, DATE);
DROP FUNCTION IF EXISTS aggregate_daily_gateway_statistics(INTEGER, DATE);
DROP TRIGGER IF EXISTS trigger_validate_gateway_statistics_consistency ON gateway_statistics;
DROP TRIGGER IF EXISTS trigger_calculate_gateway_statistics_profit_margin ON gateway_statistics;
DROP TRIGGER IF EXISTS trigger_check_gateway_statistics_tenant_consistency ON gateway_statistics;
DROP TRIGGER IF EXISTS trigger_update_gateway_statistics_updated_at ON gateway_statistics;
DROP FUNCTION IF EXISTS validate_gateway_statistics_consistency();
DROP FUNCTION IF EXISTS calculate_gateway_statistics_profit_margin();
DROP FUNCTION IF EXISTS check_gateway_statistics_tenant_consistency();
DROP FUNCTION IF EXISTS update_gateway_statistics_updated_at();
DROP TABLE IF EXISTS gateway_statistics CASCADE;
```

## المراجع

- **Requirements**: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7
- **Design Document**: `.kiro/specs/tariff-pricing-system/design.md`
- **Tasks Document**: `.kiro/specs/tariff-pricing-system/tasks.md`

## المهام السابقة

1. ✅ جدول `services` (المهمة 1.1)
2. ✅ جدول `tariffs` (المهمة 1.2)
3. ✅ جدول `dedicated_gateways` (المهمة 16.1)
4. ✅ جدول `resource_quotas` (المهمة 16.2)
5. ✅ جدول `resource_usage` (المهمة 16.3)
6. ✅ جدول `gateway_users` (المهمة 16.4)
7. ✅ جدول `gateway_api_keys` (المهمة 16.5)
8. ✅ **الحالي**: جدول `gateway_statistics` (المهمة 16.6)

## المهام القادمة

9. ⏳ **التالي**: تطوير Gateway Manager (المهمة 17)
10. ⏳ تطوير Resource Limiter (المهمة 18)
11. ⏳ تطوير Monitoring Service (المهمة 19)

## الخلاصة

جدول `gateway_statistics` يوفر:
- ✅ تسجيل شامل للإحصائيات اليومية
- ✅ عزل تام بين الشركات (RLS)
- ✅ حساب تلقائي للمقاييس المهمة
- ✅ Functions جاهزة للتجميع والتحليل
- ✅ أداء عالي مع Indexes محسّنة
- ✅ أمان قوي مع Triggers للتحقق

**تاريخ الإنشاء**: 2024  
**الحالة**: ✅ مكتمل  
**المهمة التالية**: 17 - تطوير Gateway Manager  
**المتطلبات المغطاة**: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7
