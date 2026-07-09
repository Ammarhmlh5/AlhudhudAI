# Database Optimization Script
# Task 35.1: Database Optimization
# Requirements: 27.10

Write-Host "================================" -ForegroundColor Cyan
Write-Host "Database Optimization" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match "^([^=]+)=(.*)$") {
            [System.Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
}

# Database connection parameters
$DB_HOST = $env:DB_HOST
$DB_PORT = $env:DB_PORT
$DB_NAME = $env:DB_NAME
$DB_USER = $env:DB_USER
$DB_PASSWORD = $env:DB_PASSWORD

if (-not $DB_HOST) { $DB_HOST = "localhost" }
if (-not $DB_PORT) { $DB_PORT = "5432" }
if (-not $DB_NAME) { $DB_NAME = "sms_gateway" }
if (-not $DB_USER) { $DB_USER = "postgres" }

Write-Host "Database: $DB_NAME@$DB_HOST:$DB_PORT" -ForegroundColor Yellow
Write-Host ""

# Set PGPASSWORD environment variable
$env:PGPASSWORD = $DB_PASSWORD

# Function to run SQL file
function Run-SQLFile {
    param (
        [string]$FilePath,
        [string]$Description
    )
    
    Write-Host "Running: $Description" -ForegroundColor Green
    
    $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $FilePath 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $Description completed successfully" -ForegroundColor Green
        return $true
    } else {
        Write-Host "✗ $Description failed" -ForegroundColor Red
        Write-Host $result -ForegroundColor Red
        return $false
    }
}

# Function to run SQL query
function Run-SQLQuery {
    param (
        [string]$Query,
        [string]$Description
    )
    
    Write-Host "Running: $Description" -ForegroundColor Green
    
    $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $Query 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $Description completed successfully" -ForegroundColor Green
        Write-Host $result
        return $true
    } else {
        Write-Host "✗ $Description failed" -ForegroundColor Red
        Write-Host $result -ForegroundColor Red
        return $false
    }
}

# Check if psql is available
Write-Host "Checking PostgreSQL client..." -ForegroundColor Yellow
$psqlVersion = psql --version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ PostgreSQL client (psql) not found" -ForegroundColor Red
    Write-Host "Please install PostgreSQL client tools" -ForegroundColor Red
    exit 1
}
Write-Host "✓ PostgreSQL client found: $psqlVersion" -ForegroundColor Green
Write-Host ""

# Test database connection
Write-Host "Testing database connection..." -ForegroundColor Yellow
$testResult = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Cannot connect to database" -ForegroundColor Red
    Write-Host $testResult -ForegroundColor Red
    exit 1
}
Write-Host "✓ Database connection successful" -ForegroundColor Green
Write-Host ""

# Menu
Write-Host "Select optimization task:" -ForegroundColor Cyan
Write-Host "1. Run full optimization (all steps)" -ForegroundColor White
Write-Host "2. Enable slow query logging" -ForegroundColor White
Write-Host "3. Add missing indexes" -ForegroundColor White
Write-Host "4. Update table statistics" -ForegroundColor White
Write-Host "5. Vacuum and cleanup" -ForegroundColor White
Write-Host "6. View slow queries" -ForegroundColor White
Write-Host "7. Check index usage" -ForegroundColor White
Write-Host "8. Check table sizes" -ForegroundColor White
Write-Host "9. Check cache hit ratio" -ForegroundColor White
Write-Host "0. Exit" -ForegroundColor White
Write-Host ""

$choice = Read-Host "Enter your choice (0-9)"

switch ($choice) {
    "1" {
        Write-Host ""
        Write-Host "Running full optimization..." -ForegroundColor Cyan
        Write-Host ""
        
        Run-SQLFile "scripts/database-optimization.sql" "Full database optimization"
    }
    
    "2" {
        Write-Host ""
        Write-Host "Enabling slow query logging..." -ForegroundColor Cyan
        Write-Host ""
        
        Run-SQLQuery "ALTER SYSTEM SET log_min_duration_statement = 100;" "Set log_min_duration_statement"
        Run-SQLQuery "ALTER SYSTEM SET log_statement = 'all';" "Set log_statement"
        Run-SQLQuery "SELECT pg_reload_conf();" "Reload configuration"
    }
    
    "3" {
        Write-Host ""
        Write-Host "Adding missing indexes..." -ForegroundColor Cyan
        Write-Host ""
        
        $indexes = @(
            "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_tenant_status_renewal ON subscriptions(tenant_id, status, next_renewal_date) WHERE status IN ('active', 'grace_period');",
            "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rollover_tenant_subscription_status ON rollover_messages(tenant_id, subscription_id, status) WHERE status = 'active';",
            "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounting_tenant_created_type ON accounting_transactions(tenant_id, created_at DESC, transaction_type);",
            "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_balance_history_tenant_created ON balance_history(tenant_id, created_at DESC);"
        )
        
        foreach ($index in $indexes) {
            Run-SQLQuery $index "Creating index"
        }
    }
    
    "4" {
        Write-Host ""
        Write-Host "Updating table statistics..." -ForegroundColor Cyan
        Write-Host ""
        
        $tables = @("subscriptions", "rollover_messages", "accounting_transactions", "balance_history", "tariffs", "packages")
        
        foreach ($table in $tables) {
            Run-SQLQuery "ANALYZE $table;" "Analyzing $table"
        }
    }
    
    "5" {
        Write-Host ""
        Write-Host "Running vacuum and cleanup..." -ForegroundColor Cyan
        Write-Host ""
        
        $tables = @("subscriptions", "rollover_messages", "accounting_transactions", "balance_history", "tariffs", "packages")
        
        foreach ($table in $tables) {
            Run-SQLQuery "VACUUM ANALYZE $table;" "Vacuuming $table"
        }
    }
    
    "6" {
        Write-Host ""
        Write-Host "Viewing slow queries..." -ForegroundColor Cyan
        Write-Host ""
        
        Run-SQLQuery "SELECT query, calls, total_exec_time, mean_exec_time, max_exec_time FROM pg_stat_statements WHERE mean_exec_time > 100 ORDER BY mean_exec_time DESC LIMIT 20;" "Slow queries"
    }
    
    "7" {
        Write-Host ""
        Write-Host "Checking index usage..." -ForegroundColor Cyan
        Write-Host ""
        
        Run-SQLQuery "SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read FROM pg_stat_user_indexes WHERE schemaname = 'public' ORDER BY idx_scan ASC LIMIT 20;" "Index usage"
    }
    
    "8" {
        Write-Host ""
        Write-Host "Checking table sizes..." -ForegroundColor Cyan
        Write-Host ""
        
        Run-SQLQuery "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size FROM pg_tables WHERE schemaname = 'public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;" "Table sizes"
    }
    
    "9" {
        Write-Host ""
        Write-Host "Checking cache hit ratio..." -ForegroundColor Cyan
        Write-Host ""
        
        Run-SQLQuery "SELECT sum(heap_blks_read) as heap_read, sum(heap_blks_hit) as heap_hit, sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as ratio FROM pg_statio_user_tables;" "Cache hit ratio"
    }
    
    "0" {
        Write-Host "Exiting..." -ForegroundColor Yellow
        exit 0
    }
    
    default {
        Write-Host "Invalid choice" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "================================" -ForegroundColor Cyan
Write-Host "Optimization Complete" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
