# Gateway Statistics - دليل البدء السريع

## نظرة عامة سريعة

جدول `gateway_statistics` يسجل الإحصائيات اليومية لكل بوابة مخصصة، بما في ذلك:
- 📊 إحصائيات الرسائل (المرسلة، المسلّمة، الفاشلة)
- ⚡ إحصائيات الأداء (أوقات الاستجابة، وقت التشغيل)
- 💰 إحصائيات مالية (التكلفة، الإيرادات، هامش الربح)

## التثبيت السريع

### 1. تشغيل Migration

```powershell
# Windows
.\run-gateway-statistics-migration.ps1

# Linux/Mac
chmod +x run-gateway-statistics-migration.ps1
./run-gateway-statistics-migration.ps1
```

### 2. التحقق من التثبيت

```sql
-- التحقق من وجود الجدول
SELECT * FROM information_schema.tables 
WHERE table_name = 'gateway_statistics';

-- عرض الأعمدة
\d gateway_statistics
```

## الاستخدام الأساسي

### 1. إدراج إحصائيات يومية

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-id';

-- إدراج إحصائيات
INSERT INTO gateway_statistics (
    tenant_id,
    gateway_id,
    date,
    messages_sent,
    messages_delivered,
    messages_failed,
    avg_response_time_ms,
    uptime_percentage,
    total_cost,
    total_revenue
) VALUES (
    'your-tenant-id',
    1,
    CURRENT_DATE,
    1000,
    950,
    50,
    200,
    99.50,
    100.00,
    150.00
);
```

### 2. الاستعلام عن الإحصائيات

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
-- معدل النجاح ليوم محدد
SELECT get_gateway_success_rate(1, CURRENT_DATE);

-- معدل النجاح لآخر 30 يوم
SELECT 
    date,
    get_gateway_success_rate(gateway_id, date) AS success_rate
FROM gateway_statistics
WHERE gateway_id = 1
  AND date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY date DESC;
```

### 4. إحصائيات فترة زمنية

```sql
-- إحصائيات آخر شهر
SELECT * FROM get_gateway_statistics_range(
    1,  -- gateway_id
    CURRENT_DATE - INTERVAL '30 days',
    CURRENT_DATE
);
```

## Functions الأساسية

### 1. aggregate_daily_gateway_statistics

حساب وتجميع الإحصائيات اليومية تلقائياً:

```sql
-- حساب إحصائيات الأمس
SELECT aggregate_daily_gateway_statistics(1, CURRENT_DATE - INTERVAL '1 day');
```

### 2. get_gateway_success_rate

حساب معدل النجاح:

```sql
SELECT get_gateway_success_rate(1, '2024-01-15');
-- النتيجة: 95.50
```

### 3. get_gateway_statistics_range

إحصائيات مجمعة لفترة:

```sql
SELECT * FROM get_gateway_statistics_range(
    1,
    '2024-01-01',
    '2024-01-31'
);
```

### 4. cleanup_old_gateway_statistics

تنظيف الإحصائيات القديمة:

```sql
-- حذف الإحصائيات الأقدم من سنة
SELECT cleanup_old_gateway_statistics(365);
```

## أمثلة عملية

### مثال 1: تقرير الأداء اليومي

```sql
SELECT 
    date,
    messages_sent,
    messages_delivered,
    messages_failed,
    ROUND((messages_delivered::DECIMAL / messages_sent::DECIMAL * 100), 2) AS success_rate,
    avg_response_time_ms,
    uptime_percentage
FROM gateway_statistics
WHERE gateway_id = 1
  AND date >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY date DESC;
```

### مثال 2: تقرير الإيرادات الشهرية

```sql
SELECT 
    DATE_TRUNC('month', date) AS month,
    SUM(total_revenue) AS monthly_revenue,
    SUM(total_cost) AS monthly_cost,
    SUM(profit_margin) AS monthly_profit
FROM gateway_statistics
WHERE gateway_id = 1
  AND date >= DATE_TRUNC('year', CURRENT_DATE)
GROUP BY DATE_TRUNC('month', date)
ORDER BY month DESC;
```

### مثال 3: مقارنة الأداء بين البوابات

```sql
SELECT 
    dg.name AS gateway_name,
    AVG(gs.uptime_percentage) AS avg_uptime,
    AVG(gs.avg_response_time_ms) AS avg_response_time,
    SUM(gs.messages_sent) AS total_messages,
    ROUND((SUM(gs.messages_delivered)::DECIMAL / SUM(gs.messages_sent)::DECIMAL * 100), 2) AS success_rate
FROM gateway_statistics gs
JOIN dedicated_gateways dg ON gs.gateway_id = dg.id
WHERE gs.date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY dg.id, dg.name
ORDER BY success_rate DESC;
```

## التكامل مع Go

### مثال: حساب الإحصائيات اليومية

```go
package monitoring

import (
    "context"
    "database/sql"
    "time"
)

type StatisticsService struct {
    db *sql.DB
}

func (s *StatisticsService) AggregateDaily(ctx context.Context, gatewayID int, date time.Time) error {
    query := `SELECT aggregate_daily_gateway_statistics($1, $2)`
    
    _, err := s.db.ExecContext(ctx, query, gatewayID, date)
    if err != nil {
        return fmt.Errorf("failed to aggregate statistics: %w", err)
    }
    
    return nil
}

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

### مثال: Cron Job للتجميع اليومي

```go
package jobs

import (
    "context"
    "log"
    "time"
)

func AggregateStatisticsJob(ctx context.Context, statsService *StatisticsService) {
    // يُشغل يومياً في منتصف الليل
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            yesterday := time.Now().AddDate(0, 0, -1)
            
            // الحصول على جميع البوابات النشطة
            gateways, err := getActiveGateways(ctx)
            if err != nil {
                log.Printf("Error getting active gateways: %v", err)
                continue
            }
            
            // تجميع الإحصائيات لكل بوابة
            for _, gateway := range gateways {
                err := statsService.AggregateDaily(ctx, gateway.ID, yesterday)
                if err != nil {
                    log.Printf("Error aggregating statistics for gateway %d: %v", gateway.ID, err)
                }
            }
            
        case <-ctx.Done():
            return
        }
    }
}
```

## Cron Jobs المقترحة

### 1. تجميع الإحصائيات اليومية (يومياً في منتصف الليل)

```sql
SELECT aggregate_daily_gateway_statistics(gateway_id, CURRENT_DATE - INTERVAL '1 day')
FROM dedicated_gateways
WHERE is_active = true;
```

### 2. تنظيف الإحصائيات القديمة (أسبوعياً)

```sql
SELECT cleanup_old_gateway_statistics(365);
```

### 3. إرسال تقارير أسبوعية (كل يوم اثنين)

```sql
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

## الأمان

### Row Level Security (RLS)

الجدول محمي بـ RLS - كل شركة ترى فقط إحصائياتها:

```sql
-- تعيين tenant_id
SET app.current_tenant = 'your-tenant-id';

-- الاستعلام (يُفلتر تلقائياً حسب tenant_id)
SELECT * FROM gateway_statistics;
```

### Tenant Isolation

Trigger يتحقق من أن البوابة تنتمي للشركة الصحيحة:

```sql
-- سيفشل: محاولة إدراج إحصائيات لبوابة من tenant آخر
INSERT INTO gateway_statistics (
    tenant_id,
    gateway_id,
    date,
    messages_sent
) VALUES (
    'tenant-A',
    gateway_from_tenant_B,  -- ❌ سيفشل
    CURRENT_DATE,
    100
);
```

## الأداء

### Indexes المحسّنة

جميع الـ Indexes تبدأ بـ `tenant_id` للعزل والأداء:

```sql
-- استعلام محسّن
EXPLAIN ANALYZE
SELECT * FROM gateway_statistics
WHERE tenant_id = 'your-tenant-id'
  AND gateway_id = 1
  AND date >= CURRENT_DATE - INTERVAL '7 days';
```

### Partitioning (مقترح للمستقبل)

للبيانات الكبيرة، يمكن تطبيق Partitioning حسب الشهر:

```sql
CREATE TABLE gateway_statistics_2024_01 PARTITION OF gateway_statistics
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## استكشاف الأخطاء

### مشكلة: فشل إدراج إحصائيات

```sql
-- التحقق من وجود البوابة
SELECT * FROM dedicated_gateways WHERE id = 1;

-- التحقق من tenant_id
SELECT current_setting('app.current_tenant');

-- التحقق من RLS
SELECT * FROM pg_policies WHERE tablename = 'gateway_statistics';
```

### مشكلة: معدل النجاح غير صحيح

```sql
-- التحقق من البيانات
SELECT 
    messages_sent,
    messages_delivered,
    messages_failed,
    messages_sent - (messages_delivered + messages_failed) AS pending_or_other
FROM gateway_statistics
WHERE gateway_id = 1 AND date = CURRENT_DATE;
```

### مشكلة: profit_margin غير صحيح

```sql
-- التحقق من الحساب
SELECT 
    total_revenue,
    total_cost,
    profit_margin,
    total_revenue - total_cost AS expected_profit
FROM gateway_statistics
WHERE gateway_id = 1 AND date = CURRENT_DATE;
```

## الموارد

### الملفات المهمة

- **Migration**: `migrations/025_gateway_statistics_table.sql`
- **README**: `migrations/README_GATEWAY_STATISTICS.md`
- **Test**: `test_gateway_statistics_migration.sql`
- **Runner**: `run-gateway-statistics-migration.ps1`

### الوثائق

- **Design**: `.kiro/specs/tariff-pricing-system/design.md`
- **Tasks**: `.kiro/specs/tariff-pricing-system/tasks.md`
- **Requirements**: `.kiro/specs/tariff-pricing-system/requirements.md`

### المتطلبات المغطاة

- ✅ 13.1: تسجيل إحصائيات يومية
- ✅ 13.2: تسجيل عدد الرسائل المرسلة
- ✅ 13.3: تسجيل الرسائل المسلّمة والفاشلة والمعلقة
- ✅ 13.4: تسجيل متوسط وقت الاستجابة
- ✅ 13.5: تسجيل أقصى وقت استجابة
- ✅ 13.6: تسجيل نسبة وقت التشغيل
- ✅ 13.7: تسجيل التكلفة والإيرادات

## الخطوات التالية

### المهام المكتملة

1. ✅ جدول `services` (المهمة 1.1)
2. ✅ جدول `tariffs` (المهمة 1.2)
3. ✅ جدول `dedicated_gateways` (المهمة 16.1)
4. ✅ جدول `resource_quotas` (المهمة 16.2)
5. ✅ جدول `resource_usage` (المهمة 16.3)
6. ✅ جدول `gateway_users` (المهمة 16.4)
7. ✅ جدول `gateway_api_keys` (المهمة 16.5)
8. ✅ **الحالي**: جدول `gateway_statistics` (المهمة 16.6)

### المهام القادمة

9. ⏳ **التالي**: تطوير Gateway Manager (المهمة 17)
   - إنشاء بوابة مخصصة
   - تحديث إعدادات البوابة
   - تعليق/إيقاف البوابة
   - Health Check

10. ⏳ تطوير Resource Limiter (المهمة 18)
    - فحص الحصص الموردية
    - Rate Limiting per Gateway
    - Worker Pool محدود
    - Enforcement Policies

11. ⏳ تطوير Monitoring Service (المهمة 19)
    - Metrics Collector
    - Usage Logger
    - Alert Manager
    - Statistics Aggregator

## نصائح سريعة

### 1. استخدم Functions الجاهزة

```sql
-- بدلاً من كتابة استعلامات معقدة
SELECT * FROM get_gateway_statistics_range(1, '2024-01-01', '2024-01-31');
```

### 2. جدولة Cron Jobs

```go
// استخدم River Queue أو Cron للتجميع التلقائي
func ScheduleStatisticsAggregation() {
    // كل يوم في منتصف الليل
    c := cron.New()
    c.AddFunc("0 0 * * *", func() {
        AggregateYesterdayStatistics()
    })
    c.Start()
}
```

### 3. مراقبة الأداء

```sql
-- تحقق من معدل النجاح يومياً
SELECT 
    gateway_id,
    date,
    get_gateway_success_rate(gateway_id, date) AS success_rate
FROM gateway_statistics
WHERE date = CURRENT_DATE
  AND get_gateway_success_rate(gateway_id, date) < 95.00;
```

### 4. تنظيف دوري

```sql
-- احذف الإحصائيات القديمة شهرياً
SELECT cleanup_old_gateway_statistics(365);
```

## الدعم

للمساعدة أو الأسئلة:
1. راجع `README_GATEWAY_STATISTICS.md` للتفاصيل الكاملة
2. راجع Design Document للمعمارية
3. راجع Test File للأمثلة

---

**تاريخ الإنشاء**: 2024  
**الحالة**: ✅ جاهز للاستخدام  
**المهمة**: 16.6 - إنشاء جدول gateway_statistics  
**المتطلبات**: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7
