# PowerShell script to run currency migrations
# This script runs migrations 012 and 013 for currencies and exchange rates

Write-Host "=== Currency Migration Runner ===" -ForegroundColor Cyan
Write-Host ""

# Check if PostgreSQL client is available
$psqlAvailable = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlAvailable) {
    Write-Host "Error: psql command not found. Please install PostgreSQL client." -ForegroundColor Red
    exit 1
}

# Load environment variables from .env file
if (Test-Path ".env") {
    Write-Host "Loading environment variables from .env file..." -ForegroundColor Yellow
    Get-Content .env | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
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

Write-Host "Database Configuration:" -ForegroundColor Cyan
Write-Host "  Host: $DB_HOST"
Write-Host "  Port: $DB_PORT"
Write-Host "  Database: $DB_NAME"
Write-Host "  User: $DB_USER"
Write-Host ""

# Set PGPASSWORD environment variable
$env:PGPASSWORD = $DB_PASSWORD

# Function to run a migration
function Run-Migration {
    param (
        [string]$MigrationFile,
        [string]$Description
    )
    
    Write-Host "Running migration: $Description" -ForegroundColor Yellow
    Write-Host "  File: $MigrationFile"
    
    if (-not (Test-Path $MigrationFile)) {
        Write-Host "  Error: Migration file not found!" -ForegroundColor Red
        return $false
    }
    
    # Extract the Up migration (between +migrate Up and +migrate Down)
    $content = Get-Content $MigrationFile -Raw
    if ($content -match '-- \+migrate Up\s+(.*?)\s+-- \+migrate Down') {
        $upMigration = $matches[1]
        
        # Run the migration
        $upMigration | psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -v ON_ERROR_STOP=1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Success!" -ForegroundColor Green
            return $true
        } else {
            Write-Host "  Failed!" -ForegroundColor Red
            return $false
        }
    } else {
        Write-Host "  Error: Invalid migration format!" -ForegroundColor Red
        return $false
    }
}

# Run migrations
Write-Host "Starting currency migrations..." -ForegroundColor Cyan
Write-Host ""

$success = $true

# Migration 012: currencies table
if (-not (Run-Migration -MigrationFile "migrations/012_currencies_table.sql" -Description "Create currencies table")) {
    $success = $false
}
Write-Host ""

# Migration 013: exchange_rates table
if ($success) {
    if (-not (Run-Migration -MigrationFile "migrations/013_exchange_rates_table.sql" -Description "Create exchange_rates table")) {
        $success = $false
    }
    Write-Host ""
}

# Summary
Write-Host "=== Migration Summary ===" -ForegroundColor Cyan
if ($success) {
    Write-Host "All migrations completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "1. Verify the tables were created:"
    Write-Host "   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\dt currencies'"
    Write-Host "   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\dt exchange_rates'"
    Write-Host ""
    Write-Host "2. Check the initial data:"
    Write-Host "   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c 'SELECT * FROM currencies;'"
    Write-Host "   psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c 'SELECT * FROM exchange_rates;'"
    Write-Host ""
    Write-Host "3. Run the tests:"
    Write-Host "   go test -v ./internal/currency"
} else {
    Write-Host "Some migrations failed. Please check the errors above." -ForegroundColor Red
    exit 1
}
