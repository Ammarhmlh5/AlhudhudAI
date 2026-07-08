@echo off
chcp 65001 > nul
echo ========================================
echo   فحص المتطلبات - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Check_Requirements.ps1"
