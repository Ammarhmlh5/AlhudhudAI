@echo off
chcp 65001 > nul
title ALHudhudAI - تشغيل النظام الكامل
echo ========================================
echo   تشغيل نظام الهدهد بالكامل
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" start
echo.
pause
