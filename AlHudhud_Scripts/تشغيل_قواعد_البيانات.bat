@echo off
chcp 65001 > nul
title ALHudhudAI - تشغيل قواعد البيانات
echo ========================================
echo   تشغيل قواعد البيانات (PostgreSQL + Redis)
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" databases
echo.
pause
