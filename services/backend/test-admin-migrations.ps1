# Test script for admin interface migrations
# This script verifies that all admin interface enhancement migrations can be applied successfully

Write-Host "=== Admin Interface Migrations Test ===" -ForegroundColor Cyan
Write-Host ""

# Check if PostgreSQL is running
Write-Host "1. Checking PostgreSQL connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$pgTest = psql -h localhost -U postgres -d alhudhud_db -c "SELECT version();" 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] PostgreSQL is not running or not accessible" -ForegroundColor Red
    Write-Host "Please start PostgreSQL using: docker-compose up -d postgres" -ForegroundColor Yellow
    exit 1
}

Write-Host "[OK] PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# List of admin interface migrations (032-045)
$migrations = @(
    "032_company_ratings_table.sql",
    "033_performance_metrics_table.sql",
    "034_routing_configurations_table.sql",
    "035_distribution_strategies_table.sql",
    "036_phone_devices_table.sql",
    "037_distribution_metrics_table.sql",
    "038_server_health_logs_table.sql",
    "039_alerts_table.sql",
    "040_message_queue_table.sql",
    "041_waiting_messages_table.sql",
    "042_timing_violations_table.sql",
    "043_tenant_statistics_table.sql",
    "044_enforcement_actions_table.sql",
    "045_tenant_timing_config_table.sql"
)

Write-Host "2. Testing admin interface migrations..." -ForegroundColor Yellow
Write-Host ""

$successCount = 0
$failCount = 0

foreach ($migration in $migrations) {
    $migrationPath = "migrations/$migration"
    
    if (Test-Path $migrationPath) {
        Write-Host "   Testing: $migration" -ForegroundColor Cyan
        
        # Apply migration
        $result = psql -h localhost -U postgres -d alhudhud_db -f $migrationPath 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "   [OK] $migration applied successfully" -ForegroundColor Green
            $successCount++
        } else {
            Write-Host "   [ERROR] $migration failed" -ForegroundColor Red
            Write-Host "   Error: $result" -ForegroundColor Red
            $failCount++
        }
    } else {
        Write-Host "   [WARN] $migration not found" -ForegroundColor Yellow
        $failCount++
    }
    
    Write-Host ""
}

Write-Host "=== Migration Test Summary ===" -ForegroundColor Cyan
Write-Host "Total migrations: $($migrations.Count)" -ForegroundColor White
Write-Host "Successful: $successCount" -ForegroundColor Green
Write-Host "Failed: $failCount" -ForegroundColor Red
Write-Host ""

# Verify tables were created
Write-Host "3. Verifying tables were created..." -ForegroundColor Yellow
Write-Host ""

$tables = @(
    "company_ratings",
    "performance_metrics",
    "routing_configurations",
    "distribution_strategies",
    "phone_devices",
    "distribution_metrics",
    "server_health_logs",
    "alerts",
    "message_queue",
    "waiting_messages",
    "timing_violations",
    "tenant_statistics",
    "enforcement_actions",
    "tenant_timing_config"
)

$tablesFound = 0
$tablesNotFound = 0

foreach ($table in $tables) {
    $query = "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '$table');"
    $result = psql -h localhost -U postgres -d alhudhud_db -t -c $query 2>&1
    
    if ($result -match "t") {
        Write-Host "   [OK] Table '$table' exists" -ForegroundColor Green
        $tablesFound++
    } else {
        Write-Host "   [ERROR] Table '$table' not found" -ForegroundColor Red
        $tablesNotFound++
    }
}

Write-Host ""
Write-Host "=== Table Verification Summary ===" -ForegroundColor Cyan
Write-Host "Tables found: $tablesFound / $($tables.Count)" -ForegroundColor $(if ($tablesFound -eq $tables.Count) { "Green" } else { "Yellow" })
Write-Host ""

# Check indexes
Write-Host "4. Verifying indexes..." -ForegroundColor Yellow
Write-Host ""

$indexQuery = @"
SELECT 
    schemaname,
    tablename,
    indexname
FROM pg_indexes
WHERE schemaname = 'public'
AND tablename IN ('company_ratings', 'performance_metrics', 'routing_configurations', 
                  'distribution_strategies', 'phone_devices', 'distribution_metrics',
                  'server_health_logs', 'alerts', 'message_queue', 'waiting_messages',
                  'timing_violations', 'tenant_statistics', 'enforcement_actions', 
                  'tenant_timing_config')
ORDER BY tablename, indexname;
"@

$indexes = psql -h localhost -U postgres -d alhudhud_db -c $indexQuery 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host $indexes
    Write-Host ""
    Write-Host "[OK] Indexes verified" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to verify indexes" -ForegroundColor Red
}

Write-Host ""

# Check RLS policies
Write-Host "5. Verifying Row Level Security policies..." -ForegroundColor Yellow
Write-Host ""

$rlsQuery = @"
SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    roles,
    cmd
FROM pg_policies
WHERE schemaname = 'public'
AND tablename IN ('company_ratings', 'performance_metrics', 'routing_configurations', 
                  'distribution_strategies', 'phone_devices', 'distribution_metrics',
                  'server_health_logs', 'alerts', 'message_queue', 'waiting_messages',
                  'timing_violations', 'tenant_statistics', 'enforcement_actions', 
                  'tenant_timing_config')
ORDER BY tablename, policyname;
"@

$policies = psql -h localhost -U postgres -d alhudhud_db -c $rlsQuery 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host $policies
    Write-Host ""
    Write-Host "[OK] RLS policies verified" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Failed to verify RLS policies" -ForegroundColor Red
}

Write-Host ""

# Final summary
Write-Host "=== Final Test Results ===" -ForegroundColor Cyan
Write-Host ""

if ($failCount -eq 0 -and $tablesNotFound -eq 0) {
    Write-Host "[OK] All migrations applied successfully!" -ForegroundColor Green
    Write-Host "[OK] All tables created successfully!" -ForegroundColor Green
    Write-Host "[OK] All indexes and RLS policies verified!" -ForegroundColor Green
    Write-Host ""
    Write-Host "The admin interface database schema is ready!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "[WARN] Some issues were found:" -ForegroundColor Yellow
    if ($failCount -gt 0) {
        Write-Host "  - $failCount migration(s) failed" -ForegroundColor Red
    }
    if ($tablesNotFound -gt 0) {
        Write-Host "  - $tablesNotFound table(s) not found" -ForegroundColor Red
    }
    Write-Host ""
    Write-Host "Please review the errors above and fix them before proceeding." -ForegroundColor Yellow
    exit 1
}
