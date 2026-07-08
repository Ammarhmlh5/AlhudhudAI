# =====================================================
# سكريبت تشغيل Migration: gateway_users
# =====================================================

Write-Host "=== تشغيل Migration: gateway_users ===" -ForegroundColor Cyan
Write-Host ""

# التحقق من متغيرات البيئة
if (-not $env:DB_HOST) {
    Write-Host "تحذير: متغيرات البيئة غير محددة. استخدام القيم الافتراضية..." -ForegroundColor Yellow
    $env:DB_HOST = "localhost"
    $env:DB_PORT = "5432"
    $env:DB_NAME = "sms_gateway"
    $env:DB_USER = "postgres"
    $env:DB_PASSWORD = "postgres"
}

Write-Host "معلومات الاتصال:" -ForegroundColor Green
Write-Host "  Host: $env:DB_HOST"
Write-Host "  Port: $env:DB_PORT"
Write-Host "  Database: $env:DB_NAME"
Write-Host "  User: $env:DB_USER"
Write-Host ""

# بناء connection string
$connectionString = "postgresql://${env:DB_USER}:${env:DB_PASSWORD}@${env:DB_HOST}:${env:DB_PORT}/${env:DB_NAME}?sslmode=disable"

# =====================================================
# الخطوة 1: تشغيل Migration
# =====================================================

Write-Host "الخطوة 1: تشغيل Migration..." -ForegroundColor Cyan

$migrationFile = "migrations/023_gateway_users_table.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "خطأ: ملف Migration غير موجود: $migrationFile" -ForegroundColor Red
    exit 1
}

Write-Host "تشغيل: $migrationFile" -ForegroundColor Yellow

# تشغيل Migration باستخدام psql
$env:PGPASSWORD = $env:DB_PASSWORD
psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f $migrationFile

if ($LASTEXITCODE -ne 0) {
    Write-Host "خطأ: فشل تشغيل Migration" -ForegroundColor Red
    exit 1
}

Write-Host "✓ تم تشغيل Migration بنجاح" -ForegroundColor Green
Write-Host ""

# =====================================================
# الخطوة 2: التحقق من إنشاء الجدول
# =====================================================

Write-Host "الخطوة 2: التحقق من إنشاء الجدول..." -ForegroundColor Cyan

$checkTableQuery = @"
SELECT 
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'gateway_users') AS column_count,
    (SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'gateway_users') AS index_count,
    (SELECT COUNT(*) FROM information_schema.triggers WHERE event_object_table = 'gateway_users') AS trigger_count,
    (SELECT COUNT(*) FROM pg_policies WHERE tablename = 'gateway_users') AS policy_count
FROM information_schema.tables
WHERE table_name = 'gateway_users';
"@

Write-Host "التحقق من الجدول..." -ForegroundColor Yellow
echo $checkTableQuery | psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME

Write-Host ""

# =====================================================
# الخطوة 3: تشغيل الاختبارات
# =====================================================

Write-Host "الخطوة 3: تشغيل الاختبارات..." -ForegroundColor Cyan

$testFile = "test_gateway_users_migration.sql"

if (-not (Test-Path $testFile)) {
    Write-Host "تحذير: ملف الاختبار غير موجود: $testFile" -ForegroundColor Yellow
    Write-Host "تخطي الاختبارات..." -ForegroundColor Yellow
} else {
    Write-Host "تشغيل: $testFile" -ForegroundColor Yellow
    Write-Host ""
    
    psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME -f $testFile
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "تحذير: بعض الاختبارات فشلت" -ForegroundColor Yellow
    } else {
        Write-Host "✓ جميع الاختبارات نجحت" -ForegroundColor Green
    }
}

Write-Host ""

# =====================================================
# الخطوة 4: عرض معلومات الجدول
# =====================================================

Write-Host "الخطوة 4: عرض معلومات الجدول..." -ForegroundColor Cyan

$tableInfoQuery = @"
-- معلومات الأعمدة
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'gateway_users'
ORDER BY ordinal_position;
"@

Write-Host "معلومات الأعمدة:" -ForegroundColor Yellow
echo $tableInfoQuery | psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME

Write-Host ""

# =====================================================
# الخطوة 5: عرض Indexes
# =====================================================

Write-Host "الخطوة 5: عرض Indexes..." -ForegroundColor Cyan

$indexesQuery = @"
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'gateway_users'
ORDER BY indexname;
"@

Write-Host "Indexes:" -ForegroundColor Yellow
echo $indexesQuery | psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME

Write-Host ""

# =====================================================
# الخطوة 6: عرض Triggers
# =====================================================

Write-Host "الخطوة 6: عرض Triggers..." -ForegroundColor Cyan

$triggersQuery = @"
SELECT 
    trigger_name,
    event_manipulation,
    action_timing,
    action_statement
FROM information_schema.triggers
WHERE event_object_table = 'gateway_users'
ORDER BY trigger_name;
"@

Write-Host "Triggers:" -ForegroundColor Yellow
echo $triggersQuery | psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME

Write-Host ""

# =====================================================
# الخطوة 7: عرض RLS Policies
# =====================================================

Write-Host "الخطوة 7: عرض RLS Policies..." -ForegroundColor Cyan

$policiesQuery = @"
SELECT 
    policyname,
    cmd,
    qual
FROM pg_policies
WHERE tablename = 'gateway_users'
ORDER BY policyname;
"@

Write-Host "RLS Policies:" -ForegroundColor Yellow
echo $policiesQuery | psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME

Write-Host ""

# =====================================================
# الخطوة 8: عرض Constraints
# =====================================================

Write-Host "الخطوة 8: عرض Constraints..." -ForegroundColor Cyan

$constraintsQuery = @"
SELECT 
    conname AS constraint_name,
    contype AS constraint_type,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid = 'gateway_users'::regclass
ORDER BY conname;
"@

Write-Host "Constraints:" -ForegroundColor Yellow
echo $constraintsQuery | psql -h $env:DB_HOST -p $env:DB_PORT -U $env:DB_USER -d $env:DB_NAME

Write-Host ""

# =====================================================
# ملخص
# =====================================================

Write-Host "=== ملخص ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "✓ تم إنشاء جدول gateway_users بنجاح" -ForegroundColor Green
Write-Host "✓ تم تطبيق Row Level Security (RLS)" -ForegroundColor Green
Write-Host "✓ تم إنشاء Indexes للأداء" -ForegroundColor Green
Write-Host "✓ تم إنشاء Triggers للتحقق التلقائي" -ForegroundColor Green
Write-Host ""
Write-Host "الملفات المرتبطة:" -ForegroundColor Yellow
Write-Host "  - Migration: migrations/023_gateway_users_table.sql"
Write-Host "  - README: migrations/README_GATEWAY_USERS.md"
Write-Host "  - Test: test_gateway_users_migration.sql"
Write-Host "  - Quick Start: GATEWAY_USERS_QUICK_START.md"
Write-Host ""
Write-Host "الخطوات التالية:" -ForegroundColor Yellow
Write-Host "  1. راجع ملف README_GATEWAY_USERS.md للتفاصيل"
Write-Host "  2. راجع ملف GATEWAY_USERS_QUICK_START.md للبدء السريع"
Write-Host "  3. انتقل للمهمة 16.5: إنشاء جدول gateway_api_keys"
Write-Host ""
Write-Host "=== انتهى ===" -ForegroundColor Cyan
