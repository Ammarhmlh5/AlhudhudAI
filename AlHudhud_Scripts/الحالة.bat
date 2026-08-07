@echo off
chcp 65001 > nul
title ALHudhudAI - حالة الخدمات
echo ========================================
echo   حالة خدمات نظام الهدهد
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" status
echo.
pause
