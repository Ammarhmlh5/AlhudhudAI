@echo off
chcp 65001 > nul
echo ========================================
echo   تشغيل السيرفر الخلفي - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start_Backend.ps1"
