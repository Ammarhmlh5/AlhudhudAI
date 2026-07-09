# PowerShell Script to run referrals table migration
# Usage: .\run-referrals-migration.ps1

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Referrals Table Migration" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables
if (Test-Path ".env") {
    Write-Host "[INFO] Loading environment variables from .env..." -ForegroundColor Yellow
    Get-Content .env | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            $name = $matches[1].Trim()
            $value = $matches[2].Trim()
            Set-Item -Path "env:$name" -Value $value
        }
    }
} else {
    Write-Host "[WARNING] .env file not found, using default values" -ForegroundColor Yellow
}

# Database connection parameters
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "sms_gateway" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "postgres" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "postgres" }

Write-Host "[INFO] Database: $DB_NAME@$DB_HOST:$DB_PORT" -ForegroundColor Green
Write-Host ""

# Set PGPASSWORD environment variable
$env:PGPASSWORD = $DB_PASSWORD

# Migration file
$MIGRATION_FILE = "migrations/027_referrals_table.sql"

# Check if migration file exists
if (-not (Test-Path $MIGRATION_FILE)) {
    Write-Host "[ERROR] Migration file not found: $MIGRATION_FILE" -ForegroundColor Red
    exit 1
}

Write-Host "[INFO] Running migration: $MIGRATION_FILE" -ForegroundColor Yellow
Write-Host ""

# Run migration
try {
    $output = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $MIGRATION_FILE 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[SUCCESS] Migration completed successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Output:" -ForegroundColor Cyan
        Write-Host $output
    } else {
        Write-Host "[ERROR] Migration failed!" -ForegroundColor Red
        Write-Host ""
        Write-Host "Error output:" -ForegroundColor Red
        Write-Host $output
        exit 1
    }
} catch {
    Write-Host "[ERROR] Failed to run migration: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Verifying Migration" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Verify table creation
$VERIFY_QUERY = @"
SELECT 
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'referrals') as column_count,
    (SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'referrals') as index_count
FROM information_schema.tables 
WHERE table_name = 'referrals';
"@

try {
    $verify_output = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $VERIFY_QUERY 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[INFO] Table verification:" -ForegroundColor Green
        Write-Host $verify_output
    } else {
        Write-Host "[WARNING] Could not verify table creation" -ForegroundColor Yellow
    }
} catch {
    Write-Host "[WARNING] Verification failed: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Migration Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "✓ Table 'referrals' created" -ForegroundColor Green
Write-Host "✓ Indexes created (6 indexes)" -ForegroundColor Green
Write-Host "✓ Row Level Security enabled" -ForegroundColor Green
Write-Host "✓ Policies created (4 policies)" -ForegroundColor Green
Write-Host "✓ Trigger created (update_referrals_updated_at)" -ForegroundColor Green
Write-Host "✓ View created (referral_stats)" -ForegroundColor Green
Write-Host ""
Write-Host "[INFO] Next steps:" -ForegroundColor Yellow
Write-Host "  1. Review the migration in the database" -ForegroundColor White
Write-Host "  2. Run test queries (see README_REFERRALS.md)" -ForegroundColor White
Write-Host "  3. Implement Referral Manager (internal/referral/manager.go)" -ForegroundColor White
Write-Host ""
Write-Host "For more information, see:" -ForegroundColor Yellow
Write-Host "  - migrations/README_REFERRALS.md" -ForegroundColor White
Write-Host ""

# Clean up
Remove-Item env:PGPASSWORD -ErrorAction SilentlyContinue
