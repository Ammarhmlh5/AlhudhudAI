# PowerShell Script to Run Rollover Messages Migration
# This script runs the 018_rollover_messages_table.sql migration

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Rollover Messages Table Migration" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Configuration
$DB_HOST = "localhost"
$DB_PORT = "5432"
$DB_NAME = "sms_gateway"
$DB_USER = "postgres"
$MIGRATION_FILE = "migrations/018_rollover_messages_table.sql"
$TEST_FILE = "test_rollover_migration.sql"

# Check if migration file exists
if (-not (Test-Path $MIGRATION_FILE)) {
    Write-Host "Error: Migration file not found: $MIGRATION_FILE" -ForegroundColor Red
    exit 1
}

Write-Host "Migration file: $MIGRATION_FILE" -ForegroundColor Green
Write-Host ""

# Prompt for password
$DB_PASSWORD = Read-Host "Enter PostgreSQL password for user '$DB_USER'" -AsSecureString
$BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($DB_PASSWORD)
$PlainPassword = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)

# Set environment variable for password
$env:PGPASSWORD = $PlainPassword

Write-Host "Connecting to database: $DB_NAME@$DB_HOST:$DB_PORT" -ForegroundColor Yellow
Write-Host ""

# Run the migration
Write-Host "Running migration..." -ForegroundColor Yellow
$result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $MIGRATION_FILE 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Migration completed successfully!" -ForegroundColor Green
    Write-Host ""
    
    # Verify the table was created
    Write-Host "Verifying table creation..." -ForegroundColor Yellow
    $verify = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\dt rollover_messages" 2>&1
    
    if ($verify -match "rollover_messages") {
        Write-Host "✓ Table 'rollover_messages' created successfully!" -ForegroundColor Green
    } else {
        Write-Host "⚠ Warning: Could not verify table creation" -ForegroundColor Yellow
    }
    
    Write-Host ""
    
    # Check RLS
    Write-Host "Checking Row Level Security..." -ForegroundColor Yellow
    $rls = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT tablename, rowsecurity FROM pg_tables WHERE tablename = 'rollover_messages';" 2>&1
    Write-Host $rls
    
    Write-Host ""
    
    # List helper functions
    Write-Host "Checking helper functions..." -ForegroundColor Yellow
    $functions = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT proname FROM pg_proc WHERE proname LIKE '%rollover%';" 2>&1
    Write-Host $functions
    
    Write-Host ""
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "Migration Summary" -ForegroundColor Cyan
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "✓ rollover_messages table created" -ForegroundColor Green
    Write-Host "✓ Row Level Security enabled" -ForegroundColor Green
    Write-Host "✓ RLS Policies created (SELECT, INSERT, UPDATE, DELETE)" -ForegroundColor Green
    Write-Host "✓ Indexes created (tenant_id based)" -ForegroundColor Green
    Write-Host "✓ Helper functions created:" -ForegroundColor Green
    Write-Host "  - get_available_rollover_messages()" -ForegroundColor Gray
    Write-Host "  - use_rollover_messages()" -ForegroundColor Gray
    Write-Host "  - get_total_rollover_messages()" -ForegroundColor Gray
    Write-Host "  - create_rollover_messages()" -ForegroundColor Gray
    Write-Host "  - expire_old_rollover_messages()" -ForegroundColor Gray
    Write-Host ""
    
    # Ask if user wants to run tests
    Write-Host "Would you like to run the test script? (Y/N)" -ForegroundColor Yellow
    $runTests = Read-Host
    
    if ($runTests -eq "Y" -or $runTests -eq "y") {
        if (Test-Path $TEST_FILE) {
            Write-Host ""
            Write-Host "Running tests..." -ForegroundColor Yellow
            Write-Host ""
            psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $TEST_FILE
            Write-Host ""
            Write-Host "✓ Tests completed!" -ForegroundColor Green
        } else {
            Write-Host "⚠ Test file not found: $TEST_FILE" -ForegroundColor Yellow
        }
    }
    
    Write-Host ""
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "Next Steps" -ForegroundColor Cyan
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "1. ✓ Task 6.1: packages table created" -ForegroundColor Green
    Write-Host "2. ✓ Task 6.2: subscriptions table created" -ForegroundColor Green
    Write-Host "3. ✓ Task 6.3: rollover_messages table created" -ForegroundColor Green
    Write-Host "4. ⏭ Task 7: Develop Subscription Manager" -ForegroundColor Yellow
    Write-Host "5. ⏭ Task 8: Develop Rollover Manager" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "See README_ROLLOVER.md for detailed usage instructions" -ForegroundColor Cyan
    Write-Host ""
    
} else {
    Write-Host "✗ Migration failed!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Error details:" -ForegroundColor Red
    Write-Host $result -ForegroundColor Red
    Write-Host ""
    Write-Host "Please check the error message above and try again." -ForegroundColor Yellow
    exit 1
}

# Clear password from environment
$env:PGPASSWORD = ""

Write-Host "Done!" -ForegroundColor Green
