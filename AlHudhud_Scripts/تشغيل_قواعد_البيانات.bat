@echo off
chcp 65001 > nul
echo ========================================
echo   تشغيل قواعد البيانات - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start_Databases.ps1"
