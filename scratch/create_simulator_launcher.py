import os

bat_content = """@echo off
chcp 65001 > nul
title تشغيل أداة محاكاة إرسال الرسائل
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp03_أداة_المحاكاة.ps1"
"""

ps1_content = """Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "  أداة محاكاة إرسال الرسائل (API Tester)  " -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan

$htmlPath = "C:\\ALHudhud\\ALHudhudAI\\apps\\client-web\\public\\test.html"
$backendUrl = "http://localhost:8085/api/v1/messaging/send"

# التحقق من وجود ملف الأداة
if (-not (Test-Path $htmlPath)) {
    Write-Host "❌ لم يتم العثور على ملف الأداة في المسار:" -ForegroundColor Red
    Write-Host "   $htmlPath" -ForegroundColor Red
    Read-Host "اضغط Enter للخروج..."
    exit 1
}

# التحقق من تشغيل السيرفر (Backend)
Write-Host "جاري التحقق من حالة السيرفر (Backend)..." -ForegroundColor Gray
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8085/health" -TimeoutSec 3 -ErrorAction SilentlyContinue
    Write-Host "✓ السيرفر يعمل بشكل طبيعي على المنفذ 8085." -ForegroundColor Green
} catch {
    Write-Host "⚠ السيرفر غير متاح على المنفذ 8085!" -ForegroundColor Yellow
    Write-Host "  تأكد من تشغيل 1_تشغيل_الكل.bat أولاً قبل استخدام الأداة." -ForegroundColor Yellow
    Write-Host "  سيتم فتح الأداة الآن، لكنها ستعمل فقط عند تشغيل السيرفر." -ForegroundColor Gray
}

Write-Host ""
Write-Host "جاري فتح أداة المحاكاة في المتصفح..." -ForegroundColor Cyan
Start-Process $htmlPath

Write-Host ""
Write-Host "=======================================" -ForegroundColor Green
Write-Host "  تم فتح الأداة في المتصفح بنجاح!      " -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Green
Write-Host ""
Write-Host "تعليمات الاستخدام:" -ForegroundColor Yellow
Write-Host "  1. أدخل مفتاح الـ API الخاص بالمستأجر (API Key)" -ForegroundColor White
Write-Host "  2. أدخل رقم الهاتف المستقبل" -ForegroundColor White
Write-Host "  3. اكتب نص الرسالة" -ForegroundColor White
Write-Host "  4. اضغط 'إرسال الرسالة الآن'" -ForegroundColor White
Write-Host ""
Write-Host "رابط الأداة المباشر:" -ForegroundColor Gray
Write-Host "  $htmlPath" -ForegroundColor Gray
Write-Host ""
Start-Sleep -Seconds 3
"""

desktop_path = r'C:\Users\Admin\Desktop\سكربتات_الهدهد'

with open(os.path.join(desktop_path, '3_أداة_المحاكاة.bat'), 'w', encoding='utf-8-sig') as f:
    f.write(bat_content)

with open(os.path.join(desktop_path, '3_أداة_المحاكاة.ps1'), 'w', encoding='utf-8-sig') as f:
    f.write(ps1_content)

print("Simulator launcher scripts created successfully on Desktop.")
