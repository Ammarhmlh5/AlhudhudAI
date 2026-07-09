@echo off
chcp 65001 > nul
echo ========================================
echo   إيقاف جميع الخدمات - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Stop_All.ps1"
