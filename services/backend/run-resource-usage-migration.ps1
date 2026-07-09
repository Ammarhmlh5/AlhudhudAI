# =====================================================
# تشغيل Migration: 022_resource_usage_table.sql
# =====================================================
# هذا السكريبت يقوم بتشغيل migration جدول resource_usage

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "تشغيل Migration: Resource Usage Table" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# =====================================================
# 1. التحقق من المتطلبات
# =====================================================
Write-Host "1. التحقق من المتطلبات..." -ForegroundColor Yellow

# التحقق من وجود psql
if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
    Write-Host "✗ خطأ: psql غير مثبت" -ForegroundColor Red
    Write-Host "يرجى تثبيت PostgreSQL Client" -ForegroundColor Red
    exit 1
}
Write-Host "✓ psql موجود" -ForegroundColor Green

# التحقق من متغيرات البيئة
if (-not $env:DB_HOST) {
    Write-Host "✗ خطأ: DB_HOST غير معرّف" -ForegroundColor Red
    Write-Host "يرجى تعريف متغيرات البيئة في ملف .env" -ForegroundColor Red
    exit 1
}
Write-Host "✓ متغيرات البيئة معرّفة" -ForegroundColor Green
Write-Host ""

# =====================================================
# 2. عرض معلومات الاتصال
# =====================================================
Write-Host "2. معلومات الاتصال:" -ForegroundColor Yellow
Write-Host "   Host: $env:DB_HOST" -ForegroundColor Gray
Write-Host "   Port: $env:DB_PORT" -ForegroundColor Gray
Write-Host "   Database: $env:DB_NAME" -ForegroundColor Gray
Write-Host "   User: $env:DB_USER" -ForegroundColor Gray
Write-Host ""

# =====================================================
# 3. التحقق من الاتصال بقاعدة البيانات
# =====================================================
Write-Host "3. التحقق من الاتصال بقاعدة البيانات..." -ForegroundColor Yellow

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
# 4. التحقق من وجود الجداول المطلوبة
# =====================================================
Write-Host "4. التحقق من وجود الجداول المطلوبة..." -ForegroundColor Yellow

# التحقق من جدول tenants
$tenantsExists = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants');" 2>&1

if ($tenantsExists -notmatch "t") {
    Write-Host "✗ جدول tenants غير موجود" -ForegroundColor Red
    Write-Host "يرجى تشغيل migrations الأساسية أولاً" -ForegroundColor Red
    exit 1
}
Write-Host "✓ جدول tenants موجود" -ForegroundColor Green

# التحقق من جدول dedicated_gateways
$gatewaysExists = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'dedicated_gateways');" 2>&1

if ($gatewaysExists -notmatch "t") {
    Write-Host "✗ جدول dedicated_gateways غير موجود" -ForegroundColor Red
    Write-Host "يرجى تشغيل migration 020_dedicated_gateways_table.sql أولاً" -ForegroundColor Red
    exit 1
}
Write-Host "✓ جدول dedicated_gateways موجود" -ForegroundColor Green
Write-Host ""

# =====================================================
# 5. التحقق من عدم وجود الجدول مسبقاً
# =====================================================
Write-Host "5. التحقق من عدم وجود الجدول مسبقاً..." -ForegroundColor Yellow

$tableExists = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'resource_usage');" 2>&1

if ($tableExists -match "t") {
    Write-Host "⚠ تحذير: جدول resource_usage موجود مسبقاً" -ForegroundColor Yellow
    $response = Read-Host "هل تريد حذف الجدول وإعادة إنشائه؟ (yes/no)"
    
    if ($response -eq "yes") {
        Write-Host "حذف الجدول القديم..." -ForegroundColor Yellow
        psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -c "DROP TABLE IF EXISTS resource_usage CASCADE;" 2>&1
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "✗ فشل حذف الجدول" -ForegroundColor Red
            exit 1
        }
        Write-Host "✓ تم حذف الجدول القديم" -ForegroundColor Green
    } else {
        Write-Host "تم إلغاء العملية" -ForegroundColor Yellow
        exit 0
    }
} else {
    Write-Host "✓ الجدول غير موجود - جاهز للإنشاء" -ForegroundColor Green
}
Write-Host ""

# =====================================================
# 6. تشغيل Migration
# =====================================================
Write-Host "6. تشغيل Migration..." -ForegroundColor Yellow
Write-Host "   الملف: migrations/022_resource_usage_table.sql" -ForegroundColor Gray

$migrationResult = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f "migrations/022_resource_usage_table.sql" 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ فشل تشغيل Migration" -ForegroundColor Red
    Write-Host $migrationResult -ForegroundColor Red
    exit 1
}

Write-Host "✓ تم تشغيل Migration بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 7. التحقق من إنشاء الجدول
# =====================================================
Write-Host "7. التحقق من إنشاء الجدول..." -ForegroundColor Yellow

$tableCreated = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'resource_usage');" 2>&1

if ($tableCreated -notmatch "t") {
    Write-Host "✗ فشل إنشاء الجدول" -ForegroundColor Red
    exit 1
}
Write-Host "✓ تم إنشاء الجدول بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 8. التحقق من Partitions
# =====================================================
Write-Host "8. التحقق من Partitions..." -ForegroundColor Yellow

$partitionsCount = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT COUNT(*) FROM pg_tables WHERE tablename LIKE 'resource_usage_%';" 2>&1

Write-Host "   عدد Partitions: $($partitionsCount.Trim())" -ForegroundColor Gray

if ($partitionsCount.Trim() -lt 1) {
    Write-Host "✗ لم يتم إنشاء أي Partitions" -ForegroundColor Red
    exit 1
}
Write-Host "✓ تم إنشاء Partitions بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 9. التحقق من Indexes
# =====================================================
Write-Host "9. التحقق من Indexes..." -ForegroundColor Yellow

$indexesCount = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'resource_usage';" 2>&1

Write-Host "   عدد Indexes: $($indexesCount.Trim())" -ForegroundColor Gray

if ($indexesCount.Trim() -lt 1) {
    Write-Host "⚠ تحذير: لم يتم إنشاء Indexes" -ForegroundColor Yellow
} else {
    Write-Host "✓ تم إنشاء Indexes بنجاح" -ForegroundColor Green
}
Write-Host ""

# =====================================================
# 10. التحقق من RLS
# =====================================================
Write-Host "10. التحقق من Row Level Security..." -ForegroundColor Yellow

$rlsEnabled = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT rowsecurity FROM pg_tables WHERE tablename = 'resource_usage';" 2>&1

if ($rlsEnabled -match "t") {
    Write-Host "✓ RLS مفعّل" -ForegroundColor Green
} else {
    Write-Host "✗ RLS غير مفعّل" -ForegroundColor Red
    exit 1
}

$policiesCount = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT COUNT(*) FROM pg_policies WHERE tablename = 'resource_usage';" 2>&1

Write-Host "   عدد Policies: $($policiesCount.Trim())" -ForegroundColor Gray

if ($policiesCount.Trim() -lt 4) {
    Write-Host "⚠ تحذير: عدد Policies أقل من المتوقع (4)" -ForegroundColor Yellow
} else {
    Write-Host "✓ تم إنشاء Policies بنجاح" -ForegroundColor Green
}
Write-Host ""

# =====================================================
# 11. التحقق من Functions
# =====================================================
Write-Host "11. التحقق من Functions..." -ForegroundColor Yellow

$functionsCount = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -t -c "SELECT COUNT(*) FROM pg_proc WHERE proname IN ('create_resource_usage_partition', 'cleanup_old_resource_usage_partitions');" 2>&1

Write-Host "   عدد Functions: $($functionsCount.Trim())" -ForegroundColor Gray

if ($functionsCount.Trim() -lt 2) {
    Write-Host "✗ لم يتم إنشاء جميع Functions" -ForegroundColor Red
    exit 1
}
Write-Host "✓ تم إنشاء Functions بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 12. تشغيل الاختبارات
# =====================================================
Write-Host "12. تشغيل الاختبارات..." -ForegroundColor Yellow
$runTests = Read-Host "هل تريد تشغيل الاختبارات؟ (yes/no)"

if ($runTests -eq "yes") {
    Write-Host "تشغيل test_resource_usage_migration.sql..." -ForegroundColor Yellow
    
    $testResult = psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f "test_resource_usage_migration.sql" 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "✗ فشلت بعض الاختبارات" -ForegroundColor Red
        Write-Host $testResult -ForegroundColor Red
    } else {
        Write-Host "✓ نجحت جميع الاختبارات" -ForegroundColor Green
    }
} else {
    Write-Host "تم تخطي الاختبارات" -ForegroundColor Gray
}
Write-Host ""

# =====================================================
# الخلاصة
# =====================================================
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "اكتمل Migration بنجاح!" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "الجدول جاهز للاستخدام:" -ForegroundColor Green
Write-Host "  - ✓ جدول resource_usage تم إنشاؤه" -ForegroundColor Green
Write-Host "  - ✓ Partitioning حسب الشهر" -ForegroundColor Green
Write-Host "  - ✓ Indexes للأداء" -ForegroundColor Green
Write-Host "  - ✓ Row Level Security" -ForegroundColor Green
Write-Host "  - ✓ Functions للصيانة" -ForegroundColor Green
Write-Host ""
Write-Host "الخطوات التالية:" -ForegroundColor Yellow
Write-Host "  1. راجع README_RESOURCE_USAGE.md للتوثيق الكامل" -ForegroundColor Gray
Write-Host "  2. قم بتطوير Monitoring Service لتسجيل البيانات" -ForegroundColor Gray
Write-Host "  3. قم بجدولة Cron Jobs للصيانة" -ForegroundColor Gray
Write-Host ""
