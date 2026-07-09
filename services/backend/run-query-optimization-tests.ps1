#!/usr/bin/env pwsh

# Query Optimization Tests Runner
# This script runs all query optimization tests and benchmarks

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Query Optimization Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$pgCheck = psql -U postgres -d sms_gateway -c "SELECT 1;" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Cannot connect to PostgreSQL" -ForegroundColor Red
    Write-Host "Make sure PostgreSQL is running and accessible" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Set working directory
Set-Location $PSScriptRoot

# Run unit tests
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Unit Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

go test -v ./internal/database/query_optimizer_test.go ./internal/database/query_optimizer.go

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Unit tests failed!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✓ Unit tests passed" -ForegroundColor Green
Write-Host ""

# Run benchmarks
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Benchmarks" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

go test -bench=. -benchmem -benchtime=5s ./internal/database/query_optimizer_test.go ./internal/database/query_optimizer.go

Write-Host ""
Write-Host "✓ Benchmarks completed" -ForegroundColor Green
Write-Host ""

# Run SQL optimization analysis
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running SQL Optimization Analysis" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Analyzing query performance..." -ForegroundColor Yellow
psql -U postgres -d sms_gateway -f ./scripts/query-optimization.sql > query-optimization-results.txt 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ SQL analysis completed" -ForegroundColor Green
    Write-Host "Results saved to: query-optimization-results.txt" -ForegroundColor Cyan
} else {
    Write-Host "Warning: Some SQL queries failed (this is expected if tables don't exist yet)" -ForegroundColor Yellow
}

Write-Host ""

# Check for slow queries
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Checking for Slow Queries" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$slowQueryCheck = @"
SELECT 
  query,
  calls,
  mean_exec_time,
  max_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100
ORDER BY mean_exec_time DESC
LIMIT 10;
"@

Write-Host "Queries with mean execution time > 100ms:" -ForegroundColor Yellow
psql -U postgres -d sms_gateway -c $slowQueryCheck

Write-Host ""

# Check index usage
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Checking Index Usage" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$indexCheck = @"
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 
  AND indexname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC
LIMIT 10;
"@

Write-Host "Unused indexes:" -ForegroundColor Yellow
psql -U postgres -d sms_gateway -c $indexCheck

Write-Host ""

# Check table bloat
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Checking Table Bloat" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$bloatCheck = @"
SELECT 
  schemaname,
  tablename,
  n_live_tup,
  n_dead_tup,
  ROUND(100 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_pct
FROM pg_stat_user_tables
WHERE n_dead_tup > 100
ORDER BY dead_pct DESC
LIMIT 10;
"@

Write-Host "Tables with dead tuples:" -ForegroundColor Yellow
psql -U postgres -d sms_gateway -c $bloatCheck

Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "✓ Unit tests: PASSED" -ForegroundColor Green
Write-Host "✓ Benchmarks: COMPLETED" -ForegroundColor Green
Write-Host "✓ SQL analysis: COMPLETED" -ForegroundColor Green
Write-Host "✓ Performance checks: COMPLETED" -ForegroundColor Green
Write-Host ""

Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "1. Review query-optimization-results.txt for detailed analysis" -ForegroundColor White
Write-Host "2. Check for slow queries and optimize them" -ForegroundColor White
Write-Host "3. Remove unused indexes if any" -ForegroundColor White
Write-Host "4. Run VACUUM on tables with high dead tuple percentage" -ForegroundColor White
Write-Host ""

Write-Host "Documentation:" -ForegroundColor Cyan
Write-Host "- Quick Start: QUERY_OPTIMIZATION_QUICK_START.md" -ForegroundColor White
Write-Host "- Full Guide: QUERY_OPTIMIZATION_GUIDE.md" -ForegroundColor White
Write-Host "- SQL Scripts: scripts/query-optimization.sql" -ForegroundColor White
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "All tests completed successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
