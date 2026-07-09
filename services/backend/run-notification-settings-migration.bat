@echo off
REM ================================================================
REM Apply Notification Settings Migration (080)
REM ================================================================

echo ========================================
echo Apply Notification Settings Migration
echo ========================================
echo.

REM Set PostgreSQL password
set PGPASSWORD=Ammar1983hmlh##

REM Find PostgreSQL installation
set PSQL_PATH=
if exist "C:\Program Files\PostgreSQL\16\bin\psql.exe" set PSQL_PATH=C:\Program Files\PostgreSQL\16\bin\psql.exe
if exist "C:\Program Files\PostgreSQL\15\bin\psql.exe" set PSQL_PATH=C:\Program Files\PostgreSQL\15\bin\psql.exe
if exist "C:\Program Files\PostgreSQL\14\bin\psql.exe" set PSQL_PATH=C:\Program Files\PostgreSQL\14\bin\psql.exe

if "%PSQL_PATH%"=="" (
    echo [ERROR] PostgreSQL not found!
    echo Please install PostgreSQL or add to PATH.
    pause
    exit /b 1
)

echo [INFO] Found PostgreSQL at: %PSQL_PATH%
echo.

REM Test connection
echo [1/2] Testing database connection...
"%PSQL_PATH%" -h localhost -p 5433 -U postgres -d al_hudhud_new -c "SELECT 1;" >nul 2>&1

if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to connect to database!
    echo Make sure PostgreSQL is running on port 5433
    pause
    exit /b 1
)

echo [OK] Connected successfully
echo.

REM Apply migration
echo [2/2] Applying migration 080_notification_settings.sql...
"%PSQL_PATH%" -h localhost -p 5433 -U postgres -d al_hudhud_new -f migrations\080_notification_settings.sql

if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Migration 080 failed!
    pause
    exit /b 1
)

echo [OK] Migration 080 applied
echo.

REM Clear password
set PGPASSWORD=

echo ========================================
echo [SUCCESS] Notification settings migration applied!
echo ========================================
echo.
pause