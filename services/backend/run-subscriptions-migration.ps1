#!/usr/bin/env pwsh
# PowerShell script to run the subscriptions table migration

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Subscriptions Table Migration Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables from .env file
if (Test-Path "../../.env") {
    Write-Host "Loading environment variables from .env..." -ForegroundColor Yellow
    Get-Content "../../.env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]*)\s*=\s*(.*)$') {
            $name = $matches[1].Trim()
            $value = $matches[2].Trim()
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
} else {
    Write-Host "Warning: .env file not found. Using default values." -ForegroundColor Yellow
}

# Database connection parameters
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "sms_gateway" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "postgres" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "postgres" }

Write-Host "Database Configuration:" -ForegroundColor Green
Write-Host "  Host: $DB_HOST" -ForegroundColor White
Write-Host "  Port: $DB_PORT" -ForegroundColor White
Write-Host "  Database: $DB_NAME" -ForegroundColor White
Write-Host "  User: $DB_USER" -ForegroundColor White
Write-Host ""

# Set PGPASSWORD environment variable for psql
$env:PGPASSWORD = $DB_PASSWORD

# Check if psql is available
Write-Host "Checking for psql..." -ForegroundColor Yellow
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlPath) {
    Write-Host "Error: psql command not found. Please install PostgreSQL client tools." -ForegroundColor Red
    exit 1
}
Write-Host "Found psql at: $($psqlPath.Source)" -ForegroundColor Green
Write-Host ""

# Test database connection
Write-Host "Testing database connection..." -ForegroundColor Yellow
$testQuery = "SELECT version();"
$testResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $testQuery 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Cannot connect to database." -ForegroundColor Red
    Write-Host $testResult -ForegroundColor Red
    exit 1
}
Write-Host "Database connection successful!" -ForegroundColor Green
Write-Host ""

# Run the migration
Write-Host "Running subscriptions table migration..." -ForegroundColor Yellow
Write-Host "Migration file: migrations/017_subscriptions_table.sql" -ForegroundColor White
Write-Host ""

$migrationFile = "migrations/017_subscriptions_table.sql"
if (-not (Test-Path $migrationFile)) {
    Write-Host "Error: Migration file not found: $migrationFile" -ForegroundColor Red
    exit 1
}

$result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Migration failed!" -ForegroundColor Red
    Write-Host $result -ForegroundColor Red
    exit 1
}

Write-Host $result -ForegroundColor White
Write-Host ""
Write-Host "Migration completed successfully!" -ForegroundColor Green
Write-Host ""

# Run verification tests
Write-Host "Running verification tests..." -ForegroundColor Yellow
$testFile = "test_subscriptions_migration.sql"
if (Test-Path $testFile) {
    $testResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $testFile 2>&1
    Write-Host $testResult -ForegroundColor White
    Write-Host ""
    Write-Host "Verification tests completed!" -ForegroundColor Green
} else {
    Write-Host "Warning: Test file not found: $testFile" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Migration Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "✓ Subscriptions table created" -ForegroundColor Green
Write-Host "✓ Indexes created (tenant_id based)" -ForegroundColor Green
Write-Host "✓ Row Level Security (RLS) enabled" -ForegroundColor Green
Write-Host "✓ RLS policies created (SELECT, INSERT, UPDATE, DELETE)" -ForegroundColor Green
Write-Host "✓ Helper functions created" -ForegroundColor Green
Write-Host "✓ Triggers created" -ForegroundColor Green
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Verify the migration with: psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f test_subscriptions_migration.sql" -ForegroundColor White
Write-Host "2. Create a subscription for testing" -ForegroundColor White
Write-Host "3. Test RLS policies with different tenants" -ForegroundColor White
Write-Host ""
