# ================================================================
# Phase 1 Critical Fixes - Application Script
# ================================================================
# Purpose: Apply all Phase 1 critical fixes automatically
# Date: 2026-03-22
# ================================================================

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 1 Critical Fixes - Application" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Database connection parameters
$DB_HOST = "localhost"
$DB_PORT = "5433"
$DB_USER = "postgres"
$DB_PASSWORD = "Ammar1983hmlh##"
$DB_NAME = "al_hudhud_new"

# Set PGPASSWORD environment variable
$env:PGPASSWORD = $DB_PASSWORD

# ================================================================
# Step 1: Test PostgreSQL Connection
# ================================================================
Write-Host "[1/4] Testing PostgreSQL connection..." -ForegroundColor Yellow

# Try to find psql in common locations
$psqlPaths = @(
    "C:\Program Files\PostgreSQL\16\bin\psql.exe",
    "C:\Program Files\PostgreSQL\15\bin\psql.exe",
    "C:\Program Files\PostgreSQL\14\bin\psql.exe",
    "C:\Program Files\PostgreSQL\13\bin\psql.exe",
    "C:\Program Files (x86)\PostgreSQL\16\bin\psql.exe",
    "C:\Program Files (x86)\PostgreSQL\15\bin\psql.exe"
)

$psqlPath = $null
foreach ($path in $psqlPaths) {
    if (Test-Path $path) {
        $psqlPath = $path
        break
    }
}

if ($null -eq $psqlPath) {
    # Try to find psql in PATH
    $psqlPath = (Get-Command psql -ErrorAction SilentlyContinue).Source
}

if ($null -eq $psqlPath) {
    Write-Host "✗ PostgreSQL psql not found!" -ForegroundColor Red
    Write-Host "Please install PostgreSQL or add it to PATH" -ForegroundColor Red
    Write-Host ""
    Write-Host "Manual steps:" -ForegroundColor Yellow
    Write-Host "1. Find your PostgreSQL installation directory" -ForegroundColor Yellow
    Write-Host "2. Navigate to the bin folder" -ForegroundColor Yellow
    Write-Host "3. Run: .\psql.exe -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f ..\migrations\050_fix_rbac_tables.sql" -ForegroundColor Yellow
    exit 1
}

Write-Host "✓ Found psql at: $psqlPath" -ForegroundColor Green

# Test connection
$testQuery = "SELECT version();"
$result = & $psqlPath -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c $testQuery 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Connected to database successfully" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to connect to database" -ForegroundColor Red
    Write-Host $result -ForegroundColor Red
    exit 1
}

Write-Host ""

# ================================================================
# Step 2: Create Backup
# ================================================================
Write-Host "[2/4] Creating database backup..." -ForegroundColor Yellow

$backupFile = "backup_$(Get-Date -Format 'yyyyMMdd_HHmmss').sql"
$backupPath = Join-Path $PSScriptRoot $backupFile

# Find pg_dump
$pgDumpPath = $psqlPath -replace "psql\.exe$", "pg_dump.exe"

if (Test-Path $pgDumpPath) {
    Write-Host "Creating backup: $backupFile" -ForegroundColor Cyan
    & $pgDumpPath -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $backupPath 2>&1 | Out-Null
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Backup created successfully" -ForegroundColor Green
    } else {
        Write-Host "⚠ Backup failed, but continuing..." -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠ pg_dump not found, skipping backup" -ForegroundColor Yellow
}

Write-Host ""

# ================================================================
# Step 3: Apply PostgreSQL Migration
# ================================================================
Write-Host "[3/4] Applying PostgreSQL migration..." -ForegroundColor Yellow

$migrationFile = Join-Path $PSScriptRoot "..\migrations\050_fix_rbac_tables.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "✗ Migration file not found: $migrationFile" -ForegroundColor Red
    exit 1
}

Write-Host "Applying: 050_fix_rbac_tables.sql" -ForegroundColor Cyan

$output = & $psqlPath -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Migration applied successfully" -ForegroundColor Green
    Write-Host ""
    Write-Host "Migration output:" -ForegroundColor Cyan
    Write-Host $output -ForegroundColor Gray
} else {
    Write-Host "✗ Migration failed" -ForegroundColor Red
    Write-Host $output -ForegroundColor Red
    exit 1
}

Write-Host ""

# ================================================================
# Step 4: Verify Migration
# ================================================================
Write-Host "[4/4] Verifying migration..." -ForegroundColor Yellow

$verifyQuery = @"
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'role_permissions')
        THEN 'role_permissions: OK'
        ELSE 'role_permissions: MISSING'
    END as status
UNION ALL
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_roles')
        THEN 'user_roles: OK'
        ELSE 'user_roles: MISSING'
    END as status;
"@

$verifyResult = & $psqlPath -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c $verifyQuery 2>&1

Write-Host $verifyResult -ForegroundColor Cyan

if ($verifyResult -match "MISSING") {
    Write-Host "✗ Verification failed" -ForegroundColor Red
    exit 1
} else {
    Write-Host "✓ Verification passed" -ForegroundColor Green
}

Write-Host ""

# ================================================================
# Summary
# ================================================================
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Phase 1 PostgreSQL: COMPLETE" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Secure Redis: .\setup_redis_security.ps1" -ForegroundColor Yellow
Write-Host "2. Verify Redis: .\verify_redis.ps1" -ForegroundColor Yellow
Write-Host "3. Apply SQLite fixes (see SQLITE_FIXES.md)" -ForegroundColor Yellow
Write-Host ""

# Clean up
$env:PGPASSWORD = $null
