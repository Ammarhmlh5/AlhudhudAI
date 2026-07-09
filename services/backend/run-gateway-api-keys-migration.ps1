# =====================================================
# تشغيل Migration: gateway_api_keys
# =====================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Gateway API Keys Migration Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# التحقق من وجود PostgreSQL
$pgVersion = psql --version 2>$null
if (-not $pgVersion) {
    Write-Host "❌ خطأ: PostgreSQL غير مثبت أو غير موجود في PATH" -ForegroundColor Red
    Write-Host "يرجى تثبيت PostgreSQL أولاً" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ PostgreSQL مثبت: $pgVersion" -ForegroundColor Green
Write-Host ""

# قراءة معلومات الاتصال
Write-Host "معلومات الاتصال بقاعدة البيانات:" -ForegroundColor Yellow
Write-Host "-----------------------------------" -ForegroundColor Yellow

$DB_HOST = Read-Host "Database Host (default: localhost)"
if ([string]::IsNullOrWhiteSpace($DB_HOST)) { $DB_HOST = "localhost" }

$DB_PORT = Read-Host "Database Port (default: 5432)"
if ([string]::IsNullOrWhiteSpace($DB_PORT)) { $DB_PORT = "5432" }

$DB_NAME = Read-Host "Database Name (default: sms_gateway)"
if ([string]::IsNullOrWhiteSpace($DB_NAME)) { $DB_NAME = "sms_gateway" }

$DB_USER = Read-Host "Database User (default: postgres)"
if ([string]::IsNullOrWhiteSpace($DB_USER)) { $DB_USER = "postgres" }

$DB_PASSWORD = Read-Host "Database Password" -AsSecureString
$DB_PASSWORD_PLAIN = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($DB_PASSWORD)
)

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  تشغيل Migration" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# تعيين متغير البيئة لكلمة المرور
$env:PGPASSWORD = $DB_PASSWORD_PLAIN

# تشغيل Migration
Write-Host "📝 تشغيل Migration: 024_gateway_api_keys_table.sql..." -ForegroundColor Yellow

$migrationFile = "migrations/024_gateway_api_keys_table.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "❌ خطأ: ملف Migration غير موجود: $migrationFile" -ForegroundColor Red
    exit 1
}

try {
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✅ تم تشغيل Migration بنجاح!" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "❌ فشل تشغيل Migration" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host ""
    Write-Host "❌ خطأ أثناء تشغيل Migration: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  التحقق من الجدول" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# التحقق من إنشاء الجدول
Write-Host "🔍 التحقق من إنشاء جدول gateway_api_keys..." -ForegroundColor Yellow

$checkQuery = @"
SELECT 
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'gateway_api_keys') as column_count,
    (SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'gateway_api_keys') as index_count,
    (SELECT COUNT(*) FROM pg_trigger WHERE tgrelid = 'gateway_api_keys'::regclass) as trigger_count
FROM information_schema.tables
WHERE table_name = 'gateway_api_keys';
"@

try {
    $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c $checkQuery
    
    if ($result) {
        Write-Host "✅ الجدول موجود" -ForegroundColor Green
        Write-Host $result
    } else {
        Write-Host "❌ الجدول غير موجود" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ خطأ أثناء التحقق: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  تشغيل الاختبارات" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$runTests = Read-Host "هل تريد تشغيل الاختبارات؟ (y/n)"

if ($runTests -eq "y" -or $runTests -eq "Y") {
    Write-Host "🧪 تشغيل اختبارات Migration..." -ForegroundColor Yellow
    
    $testFile = "test_gateway_api_keys_migration.sql"
    
    if (-not (Test-Path $testFile)) {
        Write-Host "❌ خطأ: ملف الاختبارات غير موجود: $testFile" -ForegroundColor Red
    } else {
        try {
            psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $testFile
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host ""
                Write-Host "✅ تم تشغيل الاختبارات بنجاح!" -ForegroundColor Green
            } else {
                Write-Host ""
                Write-Host "❌ فشلت بعض الاختبارات" -ForegroundColor Red
            }
        } catch {
            Write-Host ""
            Write-Host "❌ خطأ أثناء تشغيل الاختبارات: $_" -ForegroundColor Red
        }
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  معلومات إضافية" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "📚 الوثائق:" -ForegroundColor Yellow
Write-Host "   - README: migrations/README_GATEWAY_API_KEYS.md" -ForegroundColor White
Write-Host "   - Migration: migrations/024_gateway_api_keys_table.sql" -ForegroundColor White
Write-Host "   - Tests: test_gateway_api_keys_migration.sql" -ForegroundColor White
Write-Host ""

Write-Host "🔧 الأوامر المفيدة:" -ForegroundColor Yellow
Write-Host "   - عرض جميع المفاتيح:" -ForegroundColor White
Write-Host "     SELECT * FROM gateway_api_keys;" -ForegroundColor Gray
Write-Host ""
Write-Host "   - التحقق من صلاحية مفتاح:" -ForegroundColor White
Write-Host "     SELECT * FROM validate_api_key('hash', '192.168.1.1'::inet);" -ForegroundColor Gray
Write-Host ""
Write-Host "   - تنظيف المفاتيح المنتهية:" -ForegroundColor White
Write-Host "     SELECT cleanup_expired_api_keys();" -ForegroundColor Gray
Write-Host ""

Write-Host "✅ اكتمل!" -ForegroundColor Green
Write-Host ""

# تنظيف متغير البيئة
$env:PGPASSWORD = $null
