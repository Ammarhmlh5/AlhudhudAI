import os

ps1_content = """Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "إيقاف تشغيل الخوادم والواجهات..." -ForegroundColor Red
Write-Host "=======================================" -ForegroundColor Cyan

Write-Host "إيقاف خادم Go Backend..." -ForegroundColor Yellow
Stop-Process -Name "go" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "main" -Force -ErrorAction SilentlyContinue

Write-Host "إيقاف خدمات Node.js (الواجهات وتطبيق الهاتف)..." -ForegroundColor Yellow
Stop-Process -Name "node" -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "تم إيقاف جميع الخدمات." -ForegroundColor Green
Read-Host -Prompt "اضغط Enter للإغلاق..."
"""

bat_content = """@echo off
chcp 65001 > nul
title إيقاف جميع خدمات نظام الهدهد
echo =======================================
echo إيقاف تشغيل الخوادم والواجهات...
echo =======================================

echo إيقاف خادم Go Backend...
taskkill /F /IM go.exe /T 2>nul
taskkill /F /IM main.exe /T 2>nul

echo إيقاف خدمات Node.js (الواجهات وتطبيق الهاتف)...
taskkill /F /IM node.exe /T 2>nul

echo تم إيقاف جميع الخدمات.
pause
"""

with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\2_إيقاف_الكل.ps1', 'w', encoding='utf-8-sig') as f:
    f.write(ps1_content)

with open(r'C:\Users\Admin\Desktop\سكربتات_الهدهد\2_إيقاف_الكل.bat', 'w', encoding='utf-8-sig') as f:
    f.write(bat_content)
