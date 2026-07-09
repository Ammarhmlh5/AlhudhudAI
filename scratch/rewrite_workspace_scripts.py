import os

ps1_content = """Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "بدء تشغيل النظام بالكامل (خوادم، قواعد بيانات، وواجهات)..." -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Cyan

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

bat_content = """@echo off
chcp 65001 > nul
title تشغيل جميع خدمات نظام الهدهد
echo =======================================
echo بدء تشغيل النظام بالكامل (خوادم، قواعد بيانات، وواجهات)...
echo =======================================

echo 1. تشغيل قواعد البيانات (Docker)
cd /d C:\\ALHudhud\\ALHudhudAI && docker compose up -d
echo الانتظار 8 ثوانٍ لتجهيز قواعد البيانات...
timeout /t 8 /nobreak > nul

echo 2. تشغيل خادم Go (Backend)
start "Go Backend Server" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\services\\backend && go run main.go"

echo 3. تشغيل خادم API الشركات (Company API)
start "Company API Server" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\services\\company-api && set APP_ENV=development && .\\server.exe"

echo 4. تشغيل لوحة التحكم (Admin Portal)
start "Admin Portal Frontend" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\admin-web && npm run dev -- --port 5174"

echo 5. تشغيل واجهة العميل (Client Web)
start "Client Web" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\client-web && npm run dev -- --port 5173"

echo 6. تشغيل واجهة الشركات (Company Portal)
if exist "C:\\ALHudhud\\ALHudhudAI\\apps\\company-portal\\package.json" (
    start "Company Portal" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\company-portal && npm run dev -- --port 5176"
) else (
    echo ⚠ واجهة الشركات (Company Portal) غير موجودة أو كود المصدر الخاص بها ناقص.
)

echo 7. تشغيل واجهة الموزعين (Reseller Web)
start "Reseller Web" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\reseller-web && npm run dev -- --port 5175"

echo 8. تشغيل خادم تطبيق الهاتف (Mobile Gateway Metro)
start "Mobile Metro Bundler" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\mobile_gateway && npm start"

echo.
echo تم إطلاق جميع الخدمات، وقواعد البيانات تعمل في الخلفية.
pause
"""

with open(r'C:\ALHudhud\1_تشغيل_الكل.ps1', 'w', encoding='utf-8-sig') as f:
    f.write(ps1_content)

with open(r'C:\ALHudhud\1_تشغيل_الكل.bat', 'w', encoding='utf-8-sig') as f:
    f.write(bat_content)
print("Scripts created in workspace root successfully.")
