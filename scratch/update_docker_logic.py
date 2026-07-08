import os

ps1_content = """Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "بدء تشغيل النظام بالكامل (خوادم، قواعد بيانات، وواجهات)..." -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Cyan

# التحقق من تشغيل Docker Desktop وتفعيله تلقائياً
$dockerRunning = $false
try {
    $null = docker ps 2>&1
    if ($LASTEXITCODE -eq 0) {
        $dockerRunning = $true
    }
} catch {}

if (-not $dockerRunning) {
    Write-Host "⚠ Docker Desktop ليس قيد التشغيل. جاري محاولة تشغيله تلقائياً..." -ForegroundColor Yellow
    $dockerPath = "C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe"
    if (Test-Path $dockerPath) {
        Start-Process $dockerPath
        Write-Host "تم إطلاق Docker Desktop. جاري الانتظار حتى يصبح محرك Docker جاهزاً (قد يستغرق ذلك دقيقة)..." -ForegroundColor Gray
        
        # الانتظار حتى يعمل Docker (بحد أقصى 45 ثانية)
        for ($i = 0; $i -lt 15; $i++) {
            Start-Sleep -Seconds 3
            try {
                $null = docker ps 2>&1
                if ($LASTEXITCODE -eq 0) {
                    Write-Host "✓ محرك Docker جاهز الآن!" -ForegroundColor Green
                    $dockerRunning = $true
                    break
                }
            } catch {}
            Write-Host "." -NoNewline
        }
        Write-Host ""
        if (-not $dockerRunning) {
            Write-Host "❌ استغرق تشغيل Docker وقتاً طويلاً. يرجى التأكد من تشغيله يدوياً من سطح المكتب ثم إعادة تشغيل السكربت." -ForegroundColor Red
            Read-Host "اضغط Enter للخروج..."
            exit 1
        }
    } else {
        Write-Host "❌ لم يتم العثور على Docker Desktop في المسار الافتراضي. يرجى تشغيله يدوياً أولاً." -ForegroundColor Red
        Read-Host "اضغط Enter للخروج..."
        exit 1
    }
}

Write-Host "1. تشغيل قواعد البيانات (Docker)" -ForegroundColor Yellow
cd C:\\ALHudhud\\ALHudhudAI
docker compose up -d
Write-Host "الانتظار 8 ثوانٍ لتجهيز قواعد البيانات..." -ForegroundColor Gray
Start-Sleep -Seconds 8

Write-Host "2. تشغيل خادم Go (Backend)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\services\\backend; go run main.go" -WindowStyle Normal

Write-Host "3. تشغيل خادم API الشركات (Company API)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\services\\company-api; `$env:APP_ENV='development'; .\\server.exe" -WindowStyle Normal

Write-Host "4. تشغيل لوحة التحكم (Admin Portal)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\admin-web; npm run dev -- --port 5174" -WindowStyle Normal

Write-Host "5. تشغيل واجهة العميل (Client Web)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\client-web; npm run dev -- --port 5173" -WindowStyle Normal

Write-Host "6. تشغيل واجهة الشركات (Company Portal)" -ForegroundColor Yellow
if (Test-Path "C:\\ALHudhud\\ALHudhudAI\\apps\\company-portal\\package.json") {
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\company-portal; npm run dev -- --port 5176" -WindowStyle Normal
} else {
    Write-Host "⚠ واجهة الشركات (Company Portal) غير موجودة أو كود المصدر الخاص بها ناقص." -ForegroundColor Red
}

Write-Host "7. تشغيل واجهة الموزعين (Reseller Web)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\reseller-web; npm run dev -- --port 5175" -WindowStyle Normal

Write-Host "8. تشغيل خادم تطبيق الهاتف (Mobile Gateway Metro)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\mobile_gateway; npm start" -WindowStyle Normal

Write-Host ""
Write-Host "تم إطلاق جميع الخدمات، وقواعد البيانات تعمل في الخلفية." -ForegroundColor Green
Read-Host -Prompt "اضغط Enter للإغلاق..."
"""

# Update in Workspace Root
with open(r'C:\ALHudhud\1_تشغيل_الكل.ps1', 'w', encoding='utf-8-sig') as f:
    f.write(ps1_content)

# Update on Desktop
with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\1_تشغيل_الكل.ps1', 'w', encoding='utf-8-sig') as f:
    f.write(ps1_content)

print("PowerShell scripts updated with auto-docker logic.")
