# PowerShell script to run payment_transactions migration

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Payment Transactions Migration" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables
if (Test-Path "../../.env") {
    Write-Host "Loading environment variables from .env..." -ForegroundColor Yellow
    Get-Content "../../.env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            $name = $matches[1].Trim()
            $value = $matches[2].Trim()
            Set-Item -Path "env:$name" -Value $value
        }
    }
}

# Database connection parameters
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "sms_gateway" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "sms_user" }

Write-Host "Database: $DB_NAME" -ForegroundColor Green
Write-Host "Host: $DB_HOST:$DB_PORT" -ForegroundColor Green
Write-Host "User: $DB_USER" -ForegroundColor Green
Write-Host ""

# Set PGPASSWORD environment variable
if ($env:DB_PASSWORD) {
    $env:PGPASSWORD = $env:DB_PASSWORD
}

# Run migration
Write-Host "Running migration 028_payment_transactions_table.sql..." -ForegroundColor Yellow
$migrationFile = "migrations/028_payment_transactions_table.sql"

if (Test-Path $migrationFile) {
    psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migrationFile
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
        Write-Host ""
        
        # Verify table creation
        Write-Host "Verifying table creation..." -ForegroundColor Yellow
        $query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'payment_transactions';"
        $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c $query
        
        if ($result.Trim() -eq "1") {
            Write-Host "✓ Table 'payment_transactions' created successfully!" -ForegroundColor Green
        } else {
            Write-Host "✗ Table 'payment_transactions' not found!" -ForegroundColor Red
        }
        
        # Verify RLS
        Write-Host "Verifying Row Level Security..." -ForegroundColor Yellow
        $query = "SELECT relrowsecurity FROM pg_class WHERE relname = 'payment_transactions';"
        $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c $query
        
        if ($result.Trim() -eq "t") {
            Write-Host "✓ Row Level Security enabled!" -ForegroundColor Green
        } else {
            Write-Host "✗ Row Level Security not enabled!" -ForegroundColor Red
        }
        
        # Verify indexes
        Write-Host "Verifying indexes..." -ForegroundColor Yellow
        $query = "SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'payment_transactions';"
        $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c $query
        Write-Host "✓ Created $($result.Trim()) indexes" -ForegroundColor Green
        
    } else {
        Write-Host ""
        Write-Host "✗ Migration failed!" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✗ Migration file not found: $migrationFile" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Migration Complete!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
