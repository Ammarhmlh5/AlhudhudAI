@echo off
REM Run Packages Table Migration Script
REM This script runs the packages table migration (016_packages_table.sql)

echo === Running Packages Table Migration ===
echo.
echo This migration creates the packages table with:
echo   - Full tenant isolation (RLS policies)
echo   - Five default packages (Basic, Standard, Professional, Business, Enterprise)
echo   - Rollover configuration support
echo.

cd /d %~dp0

REM Check if .env file exists
if not exist ".env" (
    echo Warning: .env file not found
    echo   Using system environment variables
    echo.
)

echo Executing migration 016_packages_table.sql...
go run cmd\migrate\main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Migration completed successfully!
    echo.
    echo Next steps:
    echo   1. Insert default packages for your tenant:
    echo      SELECT insert_default_packages_for_tenant^('<tenant-uuid>', ^<service-id^>^);
    echo.
    echo   2. Verify the packages table:
    echo      SELECT * FROM packages;
    echo.
) else (
    echo.
    echo Migration failed with exit code: %ERRORLEVEL%
    echo.
    echo Troubleshooting:
    echo   1. Check database connection settings in .env
    echo   2. Ensure PostgreSQL is running
    echo   3. Verify previous migrations have been applied
    echo.
    exit /b %ERRORLEVEL%
)
