@echo off
REM Run Tariff Migration Script

echo === Running Tariff Table Migration ===
echo.

cd /d %~dp0

echo Executing migration...
go run cmd\migrate\main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Migration completed successfully!
) else (
    echo.
    echo Migration failed with exit code: %ERRORLEVEL%
    exit /b %ERRORLEVEL%
)
