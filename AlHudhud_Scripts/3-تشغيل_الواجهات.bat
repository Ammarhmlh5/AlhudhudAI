@echo off
chcp 65001 > nul
title ALHudhudAI - تشغيل الواجهات
echo ========================================
echo   تشغيل الواجهات
echo   (Client 5173 + Admin 5174 + Reseller 5175 + Company Portal 5176)
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" web
echo.
pause
