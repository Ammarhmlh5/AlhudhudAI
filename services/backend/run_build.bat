@echo off
cd /d %~dp0
go build 2>&1
echo Exit code: %ERRORLEVEL%
if %ERRORLEVEL% NEQ 0 pause