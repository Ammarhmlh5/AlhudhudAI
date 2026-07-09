#!/usr/bin/env pwsh
# PowerShell script to run the billing invoices migration

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Billing Invoices Migration Runner" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables from .env file
if (Test-Path "../../.env") {
    Write-Host "Loading environment variables from .env file..." -ForegroundColor Yellow
    Get-Content "../../.env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]*?)\s*=\s*(.*?)\s*$') {
            $name = $matches[1]
            $value = $matches[2]
            Set-Item -Path "env:$name" -Value $value
        }
    }
    Write-Host "✓ Environment variables loaded" -ForegroundColor Green
} else {
    Write-Host "⚠ .env file not found, using default values" -ForegroundColor Yellow
}

# Database connection parameters
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "sms_gateway" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "postgres" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "postgres" }

Write-Host ""
Write-Host "Database Connection Details:" -ForegroundColor Cyan
Write-Host "  Host: $DB_HOST" -ForegroundColor White
Write-Host "  Port: $DB_PORT" -ForegroundColor White
Write-Host "  Database: $DB_NAME" -ForegroundColor White
Write-Host "  User: $DB_USER" -ForegroundColor White
Write-Host ""

# Set PostgreSQL password environment variable
$env:PGPASSWORD = $DB_PASSWORD

# Check if psql is available
Write-Host "Checking for PostgreSQL client (psql)..." -ForegroundColor Yellow
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlPath) {
    Write-Host "✗ psql not found in PATH" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install PostgreSQL client tools or add psql to your PATH" -ForegroundColor Red
    Write-Host "Download from: https://www.postgresql.org/download/" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ psql found: $($psqlPath.Source)" -ForegroundColor Green
Write-Host ""

# Test database connection
Write-Host "Testing database connection..." -ForegroundColor Yellow
$testConnection = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to connect to database" -ForegroundColor Red
    Write-Host $testConnection -ForegroundColor Red
    exit 1
}
Write-Host "✓ Database connection successful" -ForegroundColor Green
Write-Host ""

# Check if migration file exists
$migrationFile = "migrations/019_billing_invoices_table.sql"
if (-not (Test-Path $migrationFile)) {
    Write-Host "✗ Migration file not found: $migrationFile" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Migration file found: $migrationFile" -ForegroundColor Green
Write-Host ""

# Ask for confirmation
Write-Host "This will create the billing_invoices table with:" -ForegroundColor Yellow
Write-Host "  • Complete tenant isolation (RLS)" -ForegroundColor White
Write-Host "  • Automatic invoice number generation" -ForegroundColor White
Write-Host "  • Tax calculation for different countries" -ForegroundColor White
Write-Host "  • Invoice status tracking" -ForegroundColor White
Write-Host "  • PDF and email delivery tracking" -ForegroundColor White
Write-Host ""
$confirmation = Read-Host "Do you want to proceed? (yes/no)"
if ($confirmation -ne "yes") {
    Write-Host "Migration cancelled" -ForegroundColor Yellow
    exit 0
}
Write-Host ""

# Run the migration
Write-Host "Running migration..." -ForegroundColor Yellow
Write-Host "----------------------------------------" -ForegroundColor Gray

$migrationOutput = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "----------------------------------------" -ForegroundColor Gray
    Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
    Write-Host ""
    
    # Display migration output
    Write-Host "Migration Output:" -ForegroundColor Cyan
    Write-Host $migrationOutput -ForegroundColor White
    Write-Host ""
    
    # Verify table creation
    Write-Host "Verifying table creation..." -ForegroundColor Yellow
    $tableCheck = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\d billing_invoices" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Table billing_invoices created successfully" -ForegroundColor Green
        Write-Host ""
        Write-Host "Table Structure:" -ForegroundColor Cyan
        Write-Host $tableCheck -ForegroundColor White
    } else {
        Write-Host "⚠ Could not verify table creation" -ForegroundColor Yellow
    }
    
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Cyan
    Write-Host "  1. Run the test script to verify functionality:" -ForegroundColor White
    Write-Host "     psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f test_billing_invoices_migration.sql" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  2. Review the README for usage instructions:" -ForegroundColor White
    Write-Host "     migrations/README_BILLING_INVOICES.md" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  3. Integrate invoice generation in your application:" -ForegroundColor White
    Write-Host "     - Subscription Manager" -ForegroundColor Gray
    Write-Host "     - Balance Manager" -ForegroundColor Gray
    Write-Host "     - Payment Processors" -ForegroundColor Gray
    Write-Host ""
    
} else {
    Write-Host "----------------------------------------" -ForegroundColor Gray
    Write-Host "✗ Migration failed!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error Output:" -ForegroundColor Red
    Write-Host $migrationOutput -ForegroundColor Red
    Write-Host ""
    Write-Host "Please check the error message above and fix any issues" -ForegroundColor Yellow
    exit 1
}

# Clean up
Remove-Item env:PGPASSWORD -ErrorAction SilentlyContinue

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Migration process completed" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
