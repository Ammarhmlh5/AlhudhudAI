# ============================================================================
# PowerShell Script: تشغيل Migration جدول البوابات المخصصة
# ============================================================================
# الوصف: هذا السكريبت يقوم بتشغيل migration جدول dedicated_gateways
#         واختبار جميع المزايا (RLS، Indexes، Constraints، Triggers)
# ============================================================================

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "تشغيل Migration: جدول البوابات المخصصة (Dedicated Gateways)" -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

# ============================================================================
# إعدادات قاعدة البيانات
# ============================================================================

# قراءة إعدادات قاعدة البيانات من ملف .env
$envFile = "../../.env"
if (Test-Path $envFile) {
    Write-Host "قراءة إعدادات قاعدة البيانات من .env..." -ForegroundColor Yellow
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^([^=]+)=(.*)$') {
            $key = $matches[1].Trim()
            $value = $matches[2].Trim()
            Set-Variable -Name $key -Value $value -Scope Script
        }
    }
} else {
    Write-Host "تحذير: ملف .env غير موجود، استخدام القيم الافتراضية..." -ForegroundColor Yellow
}

# الإعدادات الافتراضية
if (-not $DB_HOST) { $DB_HOST = "localhost" }
if (-not $DB_PORT) { $DB_PORT = "5432" }
if (-not $DB_NAME) { $DB_NAME = "sms_gateway" }
if (-not $DB_USER) { $DB_USER = "postgres" }
if (-not $DB_PASSWORD) { $DB_PASSWORD = "postgres" }

Write-Host "إعدادات الاتصال:" -ForegroundColor Green
Write-Host "  Host: $DB_HOST" -ForegroundColor Gray
Write-Host "  Port: $DB_PORT" -ForegroundColor Gray
Write-Host "  Database: $DB_NAME" -ForegroundColor Gray
Write-Host "  User: $DB_USER" -ForegroundColor Gray
Write-Host ""

# ============================================================================
# التحقق من وجود psql
# ============================================================================

Write-Host "التحقق من وجود psql..." -ForegroundColor Yellow
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue

if (-not $psqlPath) {
    Write-Host "✗ خطأ: psql غير موجود في PATH" -ForegroundColor Red
    Write-Host "يرجى تثبيت PostgreSQL أو إضافة psql إلى PATH" -ForegroundColor Red
    exit 1
}

Write-Host "✓ psql موجود: $($psqlPath.Source)" -ForegroundColor Green
Write-Host ""

# ============================================================================
# التحقق من الاتصال بقاعدة البيانات
# ============================================================================

Write-Host "التحقق من الاتصال بقاعدة البيانات..." -ForegroundColor Yellow

$env:PGPASSWORD = $DB_PASSWORD
$testConnection = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ خطأ: فشل الاتصال بقاعدة البيانات" -ForegroundColor Red
    Write-Host $testConnection -ForegroundColor Red
    exit 1
}

Write-Host "✓ تم الاتصال بقاعدة البيانات بنجاح" -ForegroundColor Green
Write-Host ""

# ============================================================================
# التحقق من وجود جدول tenants
# ============================================================================

Write-Host "التحقق من وجود جدول tenants..." -ForegroundColor Yellow

$checkTenants = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants');" 2>&1

if ($checkTenants -match "f") {
    Write-Host "✗ خطأ: جدول tenants غير موجود" -ForegroundColor Red
    Write-Host "يرجى تشغيل migrations الأساسية أولاً" -ForegroundColor Red
    exit 1
}

Write-Host "✓ جدول tenants موجود" -ForegroundColor Green
Write-Host ""

# ============================================================================
# تشغيل Migration
# ============================================================================

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "تشغيل Migration..." -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

$migrationFile = "migrations/020_dedicated_gateways_table.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "✗ خطأ: ملف Migration غير موجود: $migrationFile" -ForegroundColor Red
    exit 1
}

Write-Host "تشغيل: $migrationFile" -ForegroundColor Yellow
$migrationResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ خطأ: فشل تشغيل Migration" -ForegroundColor Red
    Write-Host $migrationResult -ForegroundColor Red
    exit 1
}

Write-Host "✓ تم تشغيل Migration بنجاح" -ForegroundColor Green
Write-Host ""

# ============================================================================
# تشغيل الاختبارات
# ============================================================================

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "تشغيل الاختبارات..." -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

$testFile = "test_dedicated_gateways_migration.sql"

if (-not (Test-Path $testFile)) {
    Write-Host "⚠ تحذير: ملف الاختبار غير موجود: $testFile" -ForegroundColor Yellow
    Write-Host "تخطي الاختبارات..." -ForegroundColor Yellow
} else {
    Write-Host "تشغيل: $testFile" -ForegroundColor Yellow
    Write-Host ""
    
    $testResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $testFile 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host "✗ خطأ: فشلت بعض الاختبارات" -ForegroundColor Red
        Write-Host $testResult -ForegroundColor Red
        exit 1
    }
    
    Write-Host ""
    Write-Host "✓ نجحت جميع الاختبارات" -ForegroundColor Green
}

Write-Host ""

# ============================================================================
# عرض معلومات الجدول
# ============================================================================

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "معلومات الجدول" -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "عدد الأعمدة:" -ForegroundColor Yellow
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'dedicated_gateways';"

Write-Host ""
Write-Host "عدد Indexes:" -ForegroundColor Yellow
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'dedicated_gateways';"

Write-Host ""
Write-Host "عدد Constraints:" -ForegroundColor Yellow
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM pg_constraint WHERE conrelid = 'dedicated_gateways'::regclass;"

Write-Host ""
Write-Host "عدد RLS Policies:" -ForegroundColor Yellow
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM pg_policies WHERE tablename = 'dedicated_gateways';"

Write-Host ""
Write-Host "عدد Triggers:" -ForegroundColor Yellow
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM information_schema.triggers WHERE event_object_table = 'dedicated_gateways';"

Write-Host ""

# ============================================================================
# عرض البوابات الموجودة
# ============================================================================

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "البوابات المخصصة الموجودة" -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

# إلغاء RLS مؤقتاً للعرض الشامل
$env:PGOPTIONS = "-c app.current_tenant=''"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT id, gateway_code, gateway_name, status, subscription_type, subscription_fee, server_rental_fee FROM dedicated_gateways ORDER BY created_at DESC LIMIT 10;"

Write-Host ""

# ============================================================================
# النهاية
# ============================================================================

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host "اكتمل التنفيذ بنجاح!" -ForegroundColor Green
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "الخطوات التالية:" -ForegroundColor Yellow
Write-Host "1. مراجعة ملف README_DEDICATED_GATEWAYS.md للتوثيق الكامل" -ForegroundColor Gray
Write-Host "2. إنشاء جدول resource_quotas (المهمة 16.2)" -ForegroundColor Gray
Write-Host "3. إنشاء جدول resource_usage (المهمة 16.3)" -ForegroundColor Gray
Write-Host "4. إنشاء جدول gateway_users (المهمة 16.4)" -ForegroundColor Gray
Write-Host "5. إنشاء جدول gateway_api_keys (المهمة 16.5)" -ForegroundColor Gray
Write-Host "6. إنشاء جدول gateway_statistics (المهمة 16.6)" -ForegroundColor Gray
Write-Host ""
Write-Host "ملاحظة مهمة:" -ForegroundColor Yellow
Write-Host "- الشركة المؤسسية تدفع لشركة الاتصالات مباشرة لتكلفة الرسائل" -ForegroundColor Gray
Write-Host "- المنصة تتقاضى فقط رسوم الاشتراك والإيجار" -ForegroundColor Gray
Write-Host "- المنصة تراقب الرصيد وترسل تنبيهات فقط" -ForegroundColor Gray
Write-Host ""
Write-Host "============================================================================" -ForegroundColor Cyan

# تنظيف
$env:PGPASSWORD = $null
$env:PGOPTIONS = $null
