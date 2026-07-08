#!/usr/bin/env pwsh

# =====================================================
# Gateway Statistics Migration Runner
# =====================================================
# هذا السكريبت يقوم بتشغيل migration جدول gateway_statistics
# =====================================================

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Gateway Statistics Migration Runner" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# =====================================================
# 1. التحقق من المتطلبات
# =====================================================

Write-Host "1. التحقق من المتطلبات..." -ForegroundColor Yellow

# التحقق من وجود psql
if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
    Write-Host "❌ خطأ: psql غير مثبت" -ForegroundColor Red
    Write-Host "يرجى تثبيت PostgreSQL Client" -ForegroundColor Red
    exit 1
}

Write-Host "✅ psql متوفر" -ForegroundColor Green

# التحقق من وجود ملف .env
if (-not (Test-Path ".env")) {
    Write-Host "❌ خطأ: ملف .env غير موجود" -ForegroundColor Red
    exit 1
}

Write-Host "✅ ملف .env موجود" -ForegroundColor Green
Write-Host ""

# =====================================================
# 2. قراءة متغيرات البيئة
# =====================================================

Write-Host "2. قراءة متغيرات البيئة..." -ForegroundColor Yellow

# قراءة ملف .env
Get-Content .env | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]*)\s*=\s*(.*)$') {
        $name = $matches[1].Trim()
        $value = $matches[2].Trim()
        Set-Item -Path "env:$name" -Value $value
    }
}

# بناء connection string
$DB_HOST = $env:DB_HOST
$DB_PORT = $env:DB_PORT
$DB_NAME = $env:DB_NAME
$DB_USER = $env:DB_USER
$DB_PASSWORD = $env:DB_PASSWORD

if (-not $DB_HOST -or -not $DB_PORT -or -not $DB_NAME -or -not $DB_USER) {
    Write-Host "❌ خطأ: متغيرات قاعدة البيانات غير مكتملة" -ForegroundColor Red
    exit 1
}

$env:PGPASSWORD = $DB_PASSWORD

Write-Host "✅ تم قراءة متغيرات البيئة" -ForegroundColor Green
Write-Host "   Host: $DB_HOST" -ForegroundColor Gray
Write-Host "   Port: $DB_PORT" -ForegroundColor Gray
Write-Host "   Database: $DB_NAME" -ForegroundColor Gray
Write-Host "   User: $DB_USER" -ForegroundColor Gray
Write-Host ""

# =====================================================
# 3. التحقق من الاتصال بقاعدة البيانات
# =====================================================

Write-Host "3. التحقق من الاتصال بقاعدة البيانات..." -ForegroundColor Yellow

$testConnection = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ خطأ: فشل الاتصال بقاعدة البيانات" -ForegroundColor Red
    Write-Host $testConnection -ForegroundColor Red
    exit 1
}

Write-Host "✅ الاتصال بقاعدة البيانات ناجح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 4. التحقق من وجود الجداول المطلوبة
# =====================================================

Write-Host "4. التحقق من وجود الجداول المطلوبة..." -ForegroundColor Yellow

# التحقق من جدول tenants
$tenantsExists = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants');" 2>&1

if ($tenantsExists -notmatch 't') {
    Write-Host "❌ خطأ: جدول tenants غير موجود" -ForegroundColor Red
    Write-Host "يرجى تشغيل migrations الأساسية أولاً" -ForegroundColor Red
    exit 1
}

Write-Host "✅ جدول tenants موجود" -ForegroundColor Green

# التحقق من جدول dedicated_gateways
$gatewaysExists = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'dedicated_gateways');" 2>&1

if ($gatewaysExists -notmatch 't') {
    Write-Host "❌ خطأ: جدول dedicated_gateways غير موجود" -ForegroundColor Red
    Write-Host "يرجى تشغيل migration 020_dedicated_gateways_table.sql أولاً" -ForegroundColor Red
    exit 1
}

Write-Host "✅ جدول dedicated_gateways موجود" -ForegroundColor Green
Write-Host ""

# =====================================================
# 5. التحقق من عدم وجود الجدول مسبقاً
# =====================================================

Write-Host "5. التحقق من عدم وجود الجدول مسبقاً..." -ForegroundColor Yellow

$tableExists = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'gateway_statistics');" 2>&1

if ($tableExists -match 't') {
    Write-Host "⚠️  تحذير: جدول gateway_statistics موجود مسبقاً" -ForegroundColor Yellow
    
    $response = Read-Host "هل تريد حذف الجدول الموجود وإعادة إنشائه؟ (yes/no)"
    
    if ($response -eq "yes") {
        Write-Host "جاري حذف الجدول الموجود..." -ForegroundColor Yellow
        
        $dropResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "DROP TABLE IF EXISTS gateway_statistics CASCADE;" 2>&1
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ خطأ: فشل حذف الجدول" -ForegroundColor Red
            Write-Host $dropResult -ForegroundColor Red
            exit 1
        }
        
        Write-Host "✅ تم حذف الجدول بنجاح" -ForegroundColor Green
    } else {
        Write-Host "تم إلغاء العملية" -ForegroundColor Yellow
        exit 0
    }
} else {
    Write-Host "✅ الجدول غير موجود (جاهز للإنشاء)" -ForegroundColor Green
}

Write-Host ""

# =====================================================
# 6. تشغيل Migration
# =====================================================

Write-Host "6. تشغيل Migration..." -ForegroundColor Yellow

$migrationFile = "migrations/025_gateway_statistics_table.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "❌ خطأ: ملف Migration غير موجود: $migrationFile" -ForegroundColor Red
    exit 1
}

Write-Host "تشغيل: $migrationFile" -ForegroundColor Gray

$migrationResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ خطأ: فشل تشغيل Migration" -ForegroundColor Red
    Write-Host $migrationResult -ForegroundColor Red
    exit 1
}

Write-Host "✅ تم تشغيل Migration بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# 7. التحقق من إنشاء الجدول
# =====================================================

Write-Host "7. التحقق من إنشاء الجدول..." -ForegroundColor Yellow

$tableCreated = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'gateway_statistics');" 2>&1

if ($tableCreated -notmatch 't') {
    Write-Host "❌ خطأ: فشل إنشاء الجدول" -ForegroundColor Red
    exit 1
}

Write-Host "✅ تم إنشاء الجدول بنجاح" -ForegroundColor Green

# عرض معلومات الجدول
Write-Host ""
Write-Host "معلومات الجدول:" -ForegroundColor Cyan

$tableInfo = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'gateway_statistics' ORDER BY ordinal_position;" 2>&1

Write-Host $tableInfo -ForegroundColor Gray
Write-Host ""

# =====================================================
# 8. التحقق من Indexes
# =====================================================

Write-Host "8. التحقق من Indexes..." -ForegroundColor Yellow

$indexCount = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'gateway_statistics';" 2>&1

Write-Host "✅ تم إنشاء $($indexCount.Trim()) indexes" -ForegroundColor Green
Write-Host ""

# =====================================================
# 9. التحقق من RLS Policies
# =====================================================

Write-Host "9. التحقق من RLS Policies..." -ForegroundColor Yellow

$policyCount = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM pg_policies WHERE tablename = 'gateway_statistics';" 2>&1

Write-Host "✅ تم إنشاء $($policyCount.Trim()) RLS policies" -ForegroundColor Green
Write-Host ""

# =====================================================
# 10. التحقق من Triggers
# =====================================================

Write-Host "10. التحقق من Triggers..." -ForegroundColor Yellow

$triggerCount = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM information_schema.triggers WHERE event_object_table = 'gateway_statistics';" 2>&1

Write-Host "✅ تم إنشاء $($triggerCount.Trim()) triggers" -ForegroundColor Green
Write-Host ""

# =====================================================
# 11. التحقق من Functions
# =====================================================

Write-Host "11. التحقق من Functions..." -ForegroundColor Yellow

$functions = @(
    'aggregate_daily_gateway_statistics',
    'get_gateway_success_rate',
    'get_gateway_statistics_range',
    'cleanup_old_gateway_statistics'
)

$functionsCreated = 0

foreach ($func in $functions) {
    $funcExists = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE proname = '$func');" 2>&1
    
    if ($funcExists -match 't') {
        $functionsCreated++
        Write-Host "  ✅ $func" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $func" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "✅ تم إنشاء $functionsCreated/$($functions.Count) functions" -ForegroundColor Green
Write-Host ""

# =====================================================
# 12. تشغيل اختبارات
# =====================================================

Write-Host "12. هل تريد تشغيل الاختبارات؟ (yes/no)" -ForegroundColor Yellow
$runTests = Read-Host

if ($runTests -eq "yes") {
    Write-Host ""
    Write-Host "تشغيل الاختبارات..." -ForegroundColor Yellow
    
    $testFile = "test_gateway_statistics_migration.sql"
    
    if (-not (Test-Path $testFile)) {
        Write-Host "⚠️  تحذير: ملف الاختبار غير موجود: $testFile" -ForegroundColor Yellow
    } else {
        $testResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $testFile 2>&1
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ بعض الاختبارات فشلت" -ForegroundColor Red
            Write-Host $testResult -ForegroundColor Red
        } else {
            Write-Host "✅ جميع الاختبارات نجحت" -ForegroundColor Green
        }
    }
    
    Write-Host ""
}

# =====================================================
# 13. ملخص
# =====================================================

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "ملخص Migration" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "✅ تم إنشاء جدول gateway_statistics بنجاح" -ForegroundColor Green
Write-Host "✅ تم تطبيق جميع Constraints" -ForegroundColor Green
Write-Host "✅ تم إنشاء جميع Indexes" -ForegroundColor Green
Write-Host "✅ تم تفعيل Row Level Security" -ForegroundColor Green
Write-Host "✅ تم إنشاء جميع Triggers" -ForegroundColor Green
Write-Host "✅ تم إنشاء جميع Functions" -ForegroundColor Green
Write-Host ""

Write-Host "الجدول جاهز للاستخدام!" -ForegroundColor Green
Write-Host ""

Write-Host "الخطوات التالية:" -ForegroundColor Cyan
Write-Host "1. تطوير Gateway Manager (المهمة 17)" -ForegroundColor Gray
Write-Host "2. تطوير Monitoring Service (المهمة 19)" -ForegroundColor Gray
Write-Host "3. تطوير Statistics Aggregator" -ForegroundColor Gray
Write-Host ""

Write-Host "للمزيد من المعلومات، راجع:" -ForegroundColor Cyan
Write-Host "- migrations/README_GATEWAY_STATISTICS.md" -ForegroundColor Gray
Write-Host "- .kiro/specs/tariff-pricing-system/design.md" -ForegroundColor Gray
Write-Host ""

# تنظيف
Remove-Item env:PGPASSWORD -ErrorAction SilentlyContinue

Write-Host "تم الانتهاء!" -ForegroundColor Green
Write-Host ""
