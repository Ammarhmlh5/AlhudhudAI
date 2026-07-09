@echo off
chcp 65001 > nul
echo ========================================
echo   تشغيل النظام الكامل - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start_All.ps1"
