@echo off
chcp 65001 > nul
title ALHudhudAI - إيقاف جميع الخدمات
echo ========================================
echo   إيقاف جميع خدمات نظام الهدهد
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" stop
echo.
pause
