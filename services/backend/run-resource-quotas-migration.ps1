# =====================================================
# سكريبت تشغيل Migration: Resource Quotas
# =====================================================
# يقوم بتشغيل migration جدول resource_quotas واختباره

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "تشغيل Migration: Resource Quotas" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# =====================================================
# 1. التحقق من متغيرات البيئة
# =====================================================
Write-Host "1. التحقق من متغيرات البيئة..." -ForegroundColor Yellow

if (-not $env:DB_HOST) {
    Write-Host "✗ متغير DB_HOST غير موجود" -ForegroundColor Red
    Write-Host "يرجى تعيين متغيرات البيئة أولاً" -ForegroundColor Red
    exit 1
}

Write-Host "✓ متغيرات البيئة موجودة" -ForegroundColor Green
Write-Host "  - DB_HOST: $env:DB_HOST" -ForegroundColor Gray
Write-Host "  - DB_PORT: $env:DB_PORT" -ForegroundColor Gray
Write-Host "  - DB_NAME: $env:DB_NAME" -ForegroundColor Gray
Write-Host "  - DB_USER: $env:DB_USER" -ForegroundColor Gray
Write-Host ""

# =====================================================
# 2. التحقق من الاتصال بقاعدة البيانات
# =====================================================
Write-Host "2. التحقق من الاتصال بقاعدة البيانات..." -ForegroundColor Yellow

$env:PGPASSWORD = $env:DB_PASSWORD
$testConnection = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "SELECT 1;" 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ فشل الاتصال بقاعدة البيانات" -ForegroundColor Red
    Write-Host $testConnection -ForegroundColor Red
    exit 1
}

Write-Host "✓ الاتصال بقاعدة البيانات ناجح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 3. التحقق من وجود جدول dedicated_gateways
# =====================================================
Write-Host "3. التحقق من وجود جدول dedicated_gateways..." -ForegroundColor Yellow

$checkTable = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'dedicated_gateways');" 2>&1

if ($checkTable -notmatch "t") {
    Write-Host "✗ جدول dedicated_gateways غير موجود" -ForegroundColor Red
    Write-Host "يرجى تشغيل migration 020_dedicated_gateways_table.sql أولاً" -ForegroundColor Red
    exit 1
}

Write-Host "✓ جدول dedicated_gateways موجود" -ForegroundColor Green
Write-Host ""

# =====================================================
# 4. التحقق من عدم وجود الجدول مسبقاً
# =====================================================
Write-Host "4. التحقق من عدم وجود جدول resource_quotas..." -ForegroundColor Yellow

$checkExisting = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'resource_quotas');" 2>&1

if ($checkExisting -match "t") {
    Write-Host "⚠ جدول resource_quotas موجود مسبقاً" -ForegroundColor Yellow
    $response = Read-Host "هل تريد حذف الجدول الموجود وإعادة إنشائه؟ (y/n)"
    
    if ($response -eq "y" -or $response -eq "Y") {
        Write-Host "جاري حذف الجدول الموجود..." -ForegroundColor Yellow
        psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "DROP TABLE IF EXISTS resource_quotas CASCADE;" | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ تم حذف الجدول الموجود" -ForegroundColor Green
        } else {
            Write-Host "✗ فشل حذف الجدول" -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "تم إلغاء العملية" -ForegroundColor Yellow
        exit 0
    }
} else {
    Write-Host "✓ الجدول غير موجود (جاهز للإنشاء)" -ForegroundColor Green
}
Write-Host ""

# =====================================================
# 5. تشغيل Migration
# =====================================================
Write-Host "5. تشغيل Migration..." -ForegroundColor Yellow

$migrationFile = "migrations/021_resource_quotas_table.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "✗ ملف Migration غير موجود: $migrationFile" -ForegroundColor Red
    exit 1
}

Write-Host "جاري تنفيذ: $migrationFile" -ForegroundColor Gray

$migrationResult = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f $migrationFile 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ فشل تشغيل Migration" -ForegroundColor Red
    Write-Host $migrationResult -ForegroundColor Red
    exit 1
}

Write-Host "✓ تم تشغيل Migration بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 6. التحقق من إنشاء الجدول
# =====================================================
Write-Host "6. التحقق من إنشاء الجدول..." -ForegroundColor Yellow

$verifyTable = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'resource_quotas');" 2>&1

if ($verifyTable -notmatch "t") {
    Write-Host "✗ فشل إنشاء الجدول" -ForegroundColor Red
    exit 1
}

Write-Host "✓ تم إنشاء الجدول بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 7. عرض بنية الجدول
# =====================================================
Write-Host "7. عرض بنية الجدول..." -ForegroundColor Yellow

psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "\d resource_quotas"
Write-Host ""

# =====================================================
# 8. تشغيل الاختبارات
# =====================================================
Write-Host "8. تشغيل الاختبارات..." -ForegroundColor Yellow

$testFile = "test_resource_quotas_migration.sql"

if (-not (Test-Path $testFile)) {
    Write-Host "⚠ ملف الاختبار غير موجود: $testFile" -ForegroundColor Yellow
    Write-Host "تخطي الاختبارات..." -ForegroundColor Yellow
} else {
    Write-Host "جاري تنفيذ الاختبارات..." -ForegroundColor Gray
    
    $testResult = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f $testFile 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "✗ فشلت بعض الاختبارات" -ForegroundColor Red
        Write-Host $testResult -ForegroundColor Red
    } else {
        Write-Host "✓ نجحت جميع الاختبارات" -ForegroundColor Green
    }
}
Write-Host ""

# =====================================================
# 9. عرض الإحصائيات
# =====================================================
Write-Host "9. عرض الإحصائيات..." -ForegroundColor Yellow

Write-Host "عدد السجلات في resource_quotas:" -ForegroundColor Gray
psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "SELECT COUNT(*) as total_quotas FROM resource_quotas;"
Write-Host ""

Write-Host "عدد الحصص النشطة:" -ForegroundColor Gray
psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "SELECT COUNT(*) as active_quotas FROM resource_quotas WHERE is_active = true;"
Write-Host ""

Write-Host "توزيع Enforcement Modes:" -ForegroundColor Gray
psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "SELECT enforcement_mode, COUNT(*) as count FROM resource_quotas GROUP BY enforcement_mode;"
Write-Host ""

# =====================================================
# 10. عرض أمثلة الاستخدام
# =====================================================
Write-Host "10. أمثلة الاستخدام..." -ForegroundColor Yellow

Write-Host @"

أمثلة على الاستعلامات:

1. إنشاء حصص لبوابة جديدة (Small):
   INSERT INTO resource_quotas (
       gateway_id, cpu_cores, memory_mb, bandwidth_mbps,
       storage_gb, max_db_connections, max_queries_per_second,
       redis_memory_mb, redis_max_keys, max_messages_per_hour,
       max_messages_per_day, max_queue_depth,
       api_rate_limit_per_minute, api_rate_limit_per_hour,
       max_worker_threads, enforcement_mode
   ) VALUES (
       1, 2.00, 4096, 100, 50, 20, 100, 512, 100000,
       10000, 100000, 1000, 100, 5000, 10, 'soft'
   );

2. عرض حصص بوابة معينة:
   SELECT * FROM resource_quotas WHERE gateway_id = 1;

3. تحديث الحصص (ترقية من Small إلى Medium):
   UPDATE resource_quotas
   SET cpu_cores = 4.00, memory_mb = 8192,
       bandwidth_mbps = 200, storage_gb = 100
   WHERE gateway_id = 1;

4. تغيير Enforcement Mode:
   UPDATE resource_quotas
   SET enforcement_mode = 'hard'
   WHERE gateway_id = 1;

5. عرض جميع البوابات مع حصصها:
   SELECT 
       dg.gateway_name,
       dg.company_name,
       rq.cpu_cores,
       rq.memory_mb,
       rq.max_messages_per_day,
       rq.enforcement_mode
   FROM resource_quotas rq
   JOIN dedicated_gateways dg ON rq.gateway_id = dg.id;

"@ -ForegroundColor Gray

# =====================================================
# النتيجة النهائية
# =====================================================
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "اكتمل تشغيل Migration بنجاح! ✓" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "الخطوات التالية:" -ForegroundColor Yellow
Write-Host "1. إنشاء جدول resource_usage (المهمة 16.3)" -ForegroundColor Gray
Write-Host "2. تطوير Resource Limiter في الكود" -ForegroundColor Gray
Write-Host "3. تطوير Monitoring Service" -ForegroundColor Gray
Write-Host "4. تطوير Alert Manager" -ForegroundColor Gray
Write-Host ""
Write-Host "للمزيد من المعلومات، راجع:" -ForegroundColor Yellow
Write-Host "- migrations/README_RESOURCE_QUOTAS.md" -ForegroundColor Gray
Write-Host "- .kiro/specs/tariff-pricing-system/design.md" -ForegroundColor Gray
Write-Host ""
