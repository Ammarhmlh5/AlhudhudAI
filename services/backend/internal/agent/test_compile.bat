@echo off
cd /d c:\ALHudhud\ALHudhud\ALHudhudAI\services\backend\internal\agent
go build -v 2>&1
echo.
echo Exit code: %ERRORLEVEL%
pause