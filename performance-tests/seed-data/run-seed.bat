@echo off
REM Performance Test Seed Data Runner for Windows
REM This script runs all seed data scripts to prepare the database for performance testing

setlocal enabledelayedexpansion

REM Database connection parameters
if "%DB_HOST%"=="" set DB_HOST=localhost
if "%DB_PORT%"=="" set DB_PORT=5432
if "%DB_NAME%"=="" set DB_NAME=sms_gateway
if "%DB_USER%"=="" set DB_USER=postgres
if "%DB_PASSWORD%"=="" set DB_PASSWORD=postgres

echo === Performance Test Seed Data Runner ===
echo.
echo Database: %DB_NAME%@%DB_HOST%:%DB_PORT%
echo User: %DB_USER%
echo.

REM Function to run SQL file
call :run_sql_file "seed-tenants.sql" "Creating test tenants"
if errorlevel 1 goto :error

call :run_sql_file "seed-channels.sql" "Creating test channels"
if errorlevel 1 goto :error

echo.
echo === All seed data created successfully! ===
echo.
echo Summary:
echo - 10 test tenants created with 100,000 balance each
echo - 50 test channels created (5 per tenant)
echo.
echo You can now run performance tests!
goto :end

:run_sql_file
echo Running: %~2
echo File: %~1
set PGPASSWORD=%DB_PASSWORD%
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -f %~1
if errorlevel 1 (
    echo Failed to run %~1
    exit /b 1
)
echo Success
echo.
exit /b 0

:error
echo.
echo Error occurred during seed data creation!
exit /b 1

:end
endlocal
