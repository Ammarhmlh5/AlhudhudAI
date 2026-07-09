import os

ps1_content = """Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "بدء تشغيل الخوادم والواجهات..." -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Cyan

Write-Host "1. تشغيل خادم Go (Backend)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\services\\backend; go run main.go" -WindowStyle Normal

Write-Host "2. تشغيل لوحة التحكم (Admin Portal)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\admin-web; npm run dev" -WindowStyle Normal

Write-Host "3. تشغيل واجهة العميل (Client Web)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\client-web; npm run dev" -WindowStyle Normal

Write-Host "4. تشغيل خادم تطبيق الهاتف (Mobile Gateway Metro)" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd C:\\ALHudhud\\ALHudhudAI\\apps\\mobile_gateway; npx react-native start" -WindowStyle Normal

Write-Host ""
Write-Host "تم إطلاق جميع الخدمات في نوافذ منفصلة." -ForegroundColor Green
Read-Host -Prompt "اضغط Enter للإغلاق..."
"""

bat_content = """@echo off
chcp 65001 > nul
title تشغيل جميع خدمات نظام الهدهد
echo =======================================
echo بدء تشغيل الخوادم والواجهات...
echo =======================================

echo 1. تشغيل خادم Go (Backend)
start "Go Backend Server" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\services\\backend && go run main.go"

echo 2. تشغيل لوحة التحكم (Admin Portal)
start "Admin Portal Frontend" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\admin-web && npm run dev"

echo 3. تشغيل واجهة العميل (Client Web)
start "Client Web" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\client-web && npm run dev"

echo 4. تشغيل خادم تطبيق الهاتف (Mobile Gateway Metro)
start "Mobile Metro Bundler" cmd /c "cd /d C:\\ALHudhud\\ALHudhudAI\\apps\\mobile_gateway && npx react-native start"

echo.
echo تم إطلاق جميع الخدمات في نوافذ منفصلة.
pause
"""

with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\1_تشغيل_الكل.ps1', 'w', encoding='utf-8-sig') as f:
    f.write(ps1_content)

with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\1_تشغيل_الكل.bat', 'w', encoding='utf-8-sig') as f:
    f.write(bat_content)
