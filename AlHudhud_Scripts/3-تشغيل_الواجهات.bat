@echo off
chcp 65001 > nul
echo ========================================
echo   تشغيل واجهات المستخدم - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start_Web_Interfaces.ps1"
