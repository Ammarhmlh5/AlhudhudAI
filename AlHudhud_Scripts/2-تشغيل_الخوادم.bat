@echo off
chcp 65001 > nul
title ALHudhudAI - تشغيل الخوادم
echo ========================================
echo   تشغيل قواعد البيانات + الخوادم
echo   (Backend API 8085 + Company API 8082)
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" backend
echo.
pause
