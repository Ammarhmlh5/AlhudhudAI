@echo off
chcp 65001 > nul
title ALHudhudAI - تشغيل تطبيق الهاتف
echo ========================================
echo   تشغيل تطبيق الهاتف (Expo)
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" mobile
echo.
pause
