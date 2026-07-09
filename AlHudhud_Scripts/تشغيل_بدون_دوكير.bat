@echo off
chcp 65001 > nul
echo ========================================
echo   تشغيل النظام بدون Docker - ALHudhudAI
echo ========================================
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start_Without_Docker.ps1"
