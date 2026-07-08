# Script to apply Company Rating System migrations
# Migrations: 032, 033

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Company Rating System - Apply Migrations" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Load environment variables
if (Test-Path ".env") {
    Write-Host "Loading environment variables from .env..." -ForegroundColor Yellow
    Get-Content .env | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]*?)\s*=\s*(.*?)\s*$') {
            $name = $matches[1]
            $value = $matches[2]
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
    Write-Host "✓ Environment variables loaded" -ForegroundColor Green
} else {
    Write-Host "⚠ .env file not found, using default values" -ForegroundColor Yellow
}

# Get database connection details
$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5432" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "hudhud" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "postgres" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "postgres" }

Write-Host ""
Write-Host "Database Connection:" -ForegroundColor Yellow
Write-Host "  Host: $DB_HOST" -ForegroundColor White
Write-Host "  Port: $DB_PORT" -ForegroundColor White
Write-Host "  Database: $DB_NAME" -ForegroundColor White
Write-Host "  User: $DB_USER" -ForegroundColor White
Write-Host ""

# Set PGPASSWORD environment variable
$env:PGPASSWORD = $DB_PASSWORD

# Check if psql is available
Write-Host "Checking PostgreSQL client..." -ForegroundColor Yellow
$psqlVersion = psql --version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ PostgreSQL client (psql) not found" -ForegroundColor Red
    Write-Host "Please install PostgreSQL client tools" -ForegroundColor Red
    exit 1
}
Write-Host "✓ PostgreSQL client found: $psqlVersion" -ForegroundColor Green
Write-Host ""

# Test database connection
Write-Host "Testing database connection..." -ForegroundColor Yellow
$testQuery = "SELECT version();"
$result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $testQuery 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "✗ Failed to connect to database" -ForegroundColor Red
    Write-Host "Error: $result" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Database connection successful" -ForegroundColor Green
Write-Host ""

# Apply migrations
$migrations = @(
    @{
        Number = "032"
        Name = "company_ratings_table"
        File = "migrations/032_company_ratings_table.sql"
        Description = "Create company_ratings table"
    },
    @{
        Number = "033"
        Name = "performance_metrics_table"
        File = "migrations/033_performance_metrics_table.sql"
        Description = "Create performance_metrics table"
    }
)

$successCount = 0
$failCount = 0

foreach ($migration in $migrations) {
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "Migration $($migration.Number): $($migration.Name)" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "Description: $($migration.Description)" -ForegroundColor White
    Write-Host "File: $($migration.File)" -ForegroundColor White
    Write-Host ""
    
    if (-not (Test-Path $migration.File)) {
        Write-Host "✗ Migration file not found: $($migration.File)" -ForegroundColor Red
        $failCount++
        Write-Host ""
        continue
    }
    
    Write-Host "Applying migration..." -ForegroundColor Yellow
    $result = psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migration.File 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Migration $($migration.Number) applied successfully" -ForegroundColor Green
        $successCount++
    } else {
        Write-Host "✗ Migration $($migration.Number) failed" -ForegroundColor Red
        Write-Host "Error: $result" -ForegroundColor Red
        $failCount++
    }
    Write-Host ""
}

# Verify tables were created
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Checking if tables exist..." -ForegroundColor Yellow
$verifyQuery = @"
SELECT 
    table_name,
    CASE 
        WHEN table_name = 'company_ratings' THEN '✓'
        WHEN table_name = 'performance_metrics' THEN '✓'
        ELSE '?'
    END as status
FROM information_schema.tables
WHERE table_name IN ('company_ratings', 'performance_metrics')
ORDER BY table_name;
"@

psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $verifyQuery

Write-Host ""
Write-Host "Checking indexes..." -ForegroundColor Yellow
$indexQuery = @"
SELECT 
    tablename,
    indexname
FROM pg_indexes
WHERE tablename IN ('company_ratings', 'performance_metrics')
ORDER BY tablename, indexname;
"@

psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $indexQuery

Write-Host ""
Write-Host "Checking Row Level Security..." -ForegroundColor Yellow
$rlsQuery = @"
SELECT 
    schemaname,
    tablename,
    rowsecurity
FROM pg_tables
WHERE tablename IN ('company_ratings', 'performance_metrics');
"@

psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c $rlsQuery

# Summary
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Migration Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Total migrations: $($migrations.Count)" -ForegroundColor White
Write-Host "Successful: $successCount" -ForegroundColor Green
Write-Host "Failed: $failCount" -ForegroundColor $(if ($failCount -gt 0) { "Red" } else { "White" })
Write-Host ""

if ($failCount -eq 0) {
    Write-Host "✓ All migrations applied successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "  1. Run test-rating-system-db.sql to verify the schema" -ForegroundColor White
    Write-Host "  2. Test the rating system with sample data" -ForegroundColor White
    Write-Host "  3. Integrate with existing systems" -ForegroundColor White
} else {
    Write-Host "✗ Some migrations failed. Please check the errors above." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
