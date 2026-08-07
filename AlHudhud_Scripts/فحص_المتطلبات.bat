@echo off
chcp 65001 > nul
title ALHudhudAI - فحص المتطلبات
echo ========================================
echo   فحص متطلبات نظام الهدهد
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\ALHudhud\ALHudhud\ALHudhudAI\alhudhud.ps1" check
echo.
pause
