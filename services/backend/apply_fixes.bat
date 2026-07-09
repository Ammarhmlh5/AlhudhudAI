@echo off
REM ================================================================
REM Phase 1 Critical Fixes - Batch Script
REM ================================================================

echo ========================================
echo Phase 1 Critical Fixes
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
    echo Please install PostgreSQL or update the path in this script.
    echo.
    echo Manual execution:
    echo 1. Open pgAdmin
    echo 2. Connect to database: al_hudhud_new
    echo 3. Open Query Tool
    echo 4. Execute file: migrations\050_fix_rbac_tables.sql
    pause
    exit /b 1
)

echo [1/3] Found PostgreSQL at: %PSQL_PATH%
echo.

REM Test connection
echo [2/3] Testing database connection...
"%PSQL_PATH%" -h localhost -p 5433 -U postgres -d al_hudhud_new -c "SELECT version();" >nul 2>&1

if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Failed to connect to database!
    echo Please check:
    echo - PostgreSQL is running
    echo - Database al_hudhud_new exists
    echo - Password is correct
    pause
    exit /b 1
)

echo [OK] Connected successfully
echo.

REM Apply migration
echo [3/3] Applying migration 050_fix_rbac_tables.sql...
"%PSQL_PATH%" -h localhost -p 5433 -U postgres -d al_hudhud_new -f migrations\050_fix_rbac_tables.sql

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ========================================
    echo [SUCCESS] Migration applied successfully!
    echo ========================================
    echo.
    echo Next steps:
    echo 1. Secure Redis - see MANUAL_EXECUTION_GUIDE.md
    echo 2. Apply SQLite fixes - see SQLITE_FIXES.md
    echo.
) else (
    echo.
    echo ========================================
    echo [ERROR] Migration failed!
    echo ========================================
    echo.
    echo Please check the error messages above.
    echo.
)

REM Clear password
set PGPASSWORD=

pause
