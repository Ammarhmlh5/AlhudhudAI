@echo off
REM Script to apply Company Rating System migrations
REM Migrations: 032, 033

echo ========================================
echo Company Rating System - Apply Migrations
echo ========================================
echo.

REM Database connection details
set DB_HOST=localhost
set DB_PORT=5433
set DB_USER=postgres
set DB_NAME=al_hudhud_new
set PGPASSWORD=Ammar1983hmlh##

echo Database Connection:
echo   Host: %DB_HOST%
echo   Port: %DB_PORT%
echo   Database: %DB_NAME%
echo   User: %DB_USER%
echo.

REM Check if psql is available
echo Checking PostgreSQL client...
psql --version >nul 2>&1
if errorlevel 1 (
    echo X PostgreSQL client ^(psql^) not found
    echo Please install PostgreSQL client tools
    exit /b 1
)
echo + PostgreSQL client found
echo.

REM Test database connection
echo Testing database connection...
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -c "SELECT version();" >nul 2>&1
if errorlevel 1 (
    echo X Failed to connect to database
    exit /b 1
)
echo + Database connection successful
echo.

REM Apply Migration 032
echo ========================================
echo Migration 032: company_ratings_table
echo ========================================
echo Description: Create company_ratings table
echo File: migrations/032_company_ratings_table.sql
echo.

if not exist "migrations\032_company_ratings_table.sql" (
    echo X Migration file not found
    exit /b 1
)

echo Applying migration...
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -f migrations/032_company_ratings_table.sql
if errorlevel 1 (
    echo X Migration 032 failed
    exit /b 1
)
echo + Migration 032 applied successfully
echo.

REM Apply Migration 033
echo ========================================
echo Migration 033: performance_metrics_table
echo ========================================
echo Description: Create performance_metrics table
echo File: migrations/033_performance_metrics_table.sql
echo.

if not exist "migrations\033_performance_metrics_table.sql" (
    echo X Migration file not found
    exit /b 1
)

echo Applying migration...
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -f migrations/033_performance_metrics_table.sql
if errorlevel 1 (
    echo X Migration 033 failed
    exit /b 1
)
echo + Migration 033 applied successfully
echo.

REM Verify tables were created
echo ========================================
echo Verification
echo ========================================
echo.

echo Checking if tables exist...
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -c "SELECT table_name FROM information_schema.tables WHERE table_name IN ('company_ratings', 'performance_metrics') ORDER BY table_name;"
echo.

echo Checking indexes...
psql -h %DB_HOST% -p %DB_PORT% -U %DB_USER% -d %DB_NAME% -c "SELECT tablename, indexname FROM pg_indexes WHERE tablename IN ('company_ratings', 'performance_metrics') ORDER BY tablename, indexname;"
echo.

echo ========================================
echo Migration Summary
echo ========================================
echo.
echo + All migrations applied successfully!
echo.
echo Next steps:
echo   1. Run test-rating-system-db.sql to verify the schema
echo   2. Test the rating system with sample data
echo   3. Integrate with existing systems
echo.
echo ========================================
echo Complete!
echo ========================================
