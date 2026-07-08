-- Database Optimization Script for Tariff Pricing System
-- Task 35.1: Database Optimization
-- Requirements: 27.10

-- ============================================================================
-- 1. Enable Slow Query Logging
-- ============================================================================

-- Log queries slower than 100ms
ALTER SYSTEM SET log_min_duration_statement = 100;

-- Log all statements for analysis
ALTER SYSTEM SET log_statement = 'all';

-- Log query plans for slow queries
ALTER SYSTEM SET auto_explain.log_min_duration = 100;

-- Reload configuration
SELECT pg_reload_conf();

-- ============================================================================
-- 2. Analyze Slow Queries
-- ============================================================================

-- View slow queries from pg_stat_statements
-- (Requires pg_stat_statements extension)
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Query to find slow queries
SELECT 
    query,
    calls,
    total_exec_time,
    mean_exec_time,
    max_exec_time,
    stddev_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100
ORDER BY mean_exec_time DESC
LIMIT 20;

-- ============================================================================
-- 3. Add Missing Indexes
-- ============================================================================

-- Subscriptions indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_tenant_status_renewal 
ON subscriptions(tenant_id, status, next_renewal_date)
WHERE status IN ('active', 'grace_period');

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_tenant_package
ON subscriptions(tenant_id, package_id, status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_tenant_created
ON subscriptions(tenant_id, created_at DESC);

-- Rollover messages indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rollover_tenant_subscription_status
ON rollover_messages(tenant_id, subscription_id, status)
WHERE status = 'active';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rollover_tenant_expiry
ON rollover_messages(tenant_id, expires_at)
WHERE status = 'active' AND expires_at IS NOT NULL;

-- Accounting transactions indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounting_tenant_created_type
ON accounting_transactions(tenant_id, created_at DESC, transaction_type);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounting_tenant_status
ON accounting_transactions(tenant_id, status);

-- Balance history indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_balance_history_tenant_created
ON balance_history(tenant_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_balance_history_tenant_type
ON balance_history(tenant_id, change_type);

-- Tariffs indexes (if not already created)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tariffs_tenant_service_active
ON tariffs(tenant_id, service_id, is_active)
WHERE is_active = true;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tariffs_tenant_effective
ON tariffs(tenant_id, effective_from, effective_to)
WHERE is_active = true;

-- Packages indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_packages_tenant_active
ON packages(tenant_id, is_active)
WHERE is_active = true;

-- Dedicated gateways indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_dedicated_gateways_tenant_status
ON dedicated_gateways(tenant_id, status);

-- Gateway statistics indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_gateway_stats_tenant_date
ON gateway_statistics(tenant_id, date DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_gateway_stats_gateway_date
ON gateway_statistics(gateway_id, date DESC);

-- Resource usage indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_resource_usage_tenant_timestamp
ON resource_usage(tenant_id, timestamp DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_resource_usage_gateway_timestamp
ON resource_usage(gateway_id, timestamp DESC);

-- ============================================================================
-- 4. Optimize Existing Indexes
-- ============================================================================

-- Rebuild fragmented indexes
REINDEX INDEX CONCURRENTLY idx_subscriptions_tenant_status_renewal;
REINDEX INDEX CONCURRENTLY idx_accounting_tenant_created_type;
REINDEX INDEX CONCURRENTLY idx_balance_history_tenant_created;

-- ============================================================================
-- 5. Table Statistics Update
-- ============================================================================

-- Update statistics for query planner
ANALYZE subscriptions;
ANALYZE rollover_messages;
ANALYZE accounting_transactions;
ANALYZE balance_history;
ANALYZE tariffs;
ANALYZE packages;
ANALYZE dedicated_gateways;
ANALYZE gateway_statistics;
ANALYZE resource_usage;

-- ============================================================================
-- 6. Vacuum and Cleanup
-- ============================================================================

-- Vacuum tables to reclaim space and update statistics
VACUUM ANALYZE subscriptions;
VACUUM ANALYZE rollover_messages;
VACUUM ANALYZE accounting_transactions;
VACUUM ANALYZE balance_history;
VACUUM ANALYZE tariffs;
VACUUM ANALYZE packages;

-- ============================================================================
-- 7. Query Optimization Examples
-- ============================================================================

-- Example 1: Optimized subscription query
-- Before: SELECT * FROM subscriptions WHERE tenant_id = 'xxx' AND status = 'active'
-- After: Use covering index
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT id, package_id, status, messages_remaining, next_renewal_date
FROM subscriptions 
WHERE tenant_id = 'xxx'::uuid 
AND status = 'active'
ORDER BY created_at DESC
LIMIT 10;

-- Example 2: Optimized rollover query
-- Before: SELECT * FROM rollover_messages WHERE subscription_id = xxx
-- After: Use tenant_id in WHERE clause
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT id, messages_count, expires_at, status
FROM rollover_messages
WHERE tenant_id = 'xxx'::uuid
AND subscription_id = 123
AND status = 'active'
ORDER BY created_at ASC;

-- Example 3: Optimized accounting query
-- Before: SELECT * FROM accounting_transactions WHERE tenant_id = 'xxx' ORDER BY created_at DESC
-- After: Use covering index and limit columns
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT id, transaction_type, amount, status, created_at
FROM accounting_transactions
WHERE tenant_id = 'xxx'::uuid
ORDER BY created_at DESC
LIMIT 50;

-- ============================================================================
-- 8. Materialized Views for Reports (Optional)
-- ============================================================================

-- Daily subscription statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_subscription_stats AS
SELECT 
    tenant_id,
    DATE(created_at) as date,
    COUNT(*) as total_subscriptions,
    COUNT(*) FILTER (WHERE status = 'active') as active_subscriptions,
    COUNT(*) FILTER (WHERE status = 'expired') as expired_subscriptions,
    SUM(messages_remaining) as total_messages_remaining
FROM subscriptions
GROUP BY tenant_id, DATE(created_at);

CREATE INDEX ON mv_daily_subscription_stats(tenant_id, date DESC);

-- Refresh materialized view (run daily via cron)
-- REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_subscription_stats;

-- ============================================================================
-- 9. Partitioning Strategy (For Large Tables)
-- ============================================================================

-- Example: Partition accounting_transactions by month
-- (Uncomment if table grows very large)

/*
-- Create partitioned table
CREATE TABLE accounting_transactions_partitioned (
    LIKE accounting_transactions INCLUDING ALL
) PARTITION BY RANGE (created_at);

-- Create partitions for each month
CREATE TABLE accounting_transactions_2024_01 
PARTITION OF accounting_transactions_partitioned
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE accounting_transactions_2024_02 
PARTITION OF accounting_transactions_partitioned
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- ... create more partitions as needed
*/

-- ============================================================================
-- 10. Connection Pool Optimization
-- ============================================================================

-- Set optimal connection pool settings
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;
ALTER SYSTEM SET work_mem = '4MB';
ALTER SYSTEM SET min_wal_size = '1GB';
ALTER SYSTEM SET max_wal_size = '4GB';

-- Reload configuration
SELECT pg_reload_conf();

-- ============================================================================
-- 11. Monitoring Queries
-- ============================================================================

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan ASC;

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check cache hit ratio
SELECT 
    sum(heap_blks_read) as heap_read,
    sum(heap_blks_hit) as heap_hit,
    sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as ratio
FROM pg_statio_user_tables;

-- Check active connections
SELECT 
    datname,
    count(*) as connections,
    max(state) as state
FROM pg_stat_activity
GROUP BY datname;

-- ============================================================================
-- 12. Maintenance Schedule
-- ============================================================================

-- Run these commands regularly (via cron):

-- Daily:
-- VACUUM ANALYZE;
-- REINDEX INDEX CONCURRENTLY <index_name>;

-- Weekly:
-- VACUUM FULL; (during low traffic)
-- REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_subscription_stats;

-- Monthly:
-- pg_dump for backup
-- Check for unused indexes
-- Review slow query log

-- ============================================================================
-- Verification Queries
-- ============================================================================

-- Verify indexes are created
SELECT 
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
AND tablename IN ('subscriptions', 'rollover_messages', 'accounting_transactions', 'balance_history')
ORDER BY tablename, indexname;

-- Verify statistics are up to date
SELECT 
    schemaname,
    tablename,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY tablename;

-- Check for missing indexes (tables with sequential scans)
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    seq_tup_read / seq_scan as avg_seq_tup_read
FROM pg_stat_user_tables
WHERE schemaname = 'public'
AND seq_scan > 0
ORDER BY seq_tup_read DESC
LIMIT 20;

-- ============================================================================
-- Notes
-- ============================================================================

-- 1. Run this script during low traffic periods
-- 2. Monitor query performance before and after
-- 3. Use EXPLAIN ANALYZE to verify query plans
-- 4. Adjust connection pool settings based on workload
-- 5. Schedule regular VACUUM and ANALYZE
-- 6. Monitor disk space usage
-- 7. Consider partitioning for very large tables
-- 8. Use materialized views for complex reports
-- 9. Keep statistics up to date
-- 10. Review slow query log regularly

-- ============================================================================
-- Expected Results
-- ============================================================================

-- After optimization:
-- - Query response time < 50ms for most queries
-- - Index usage > 95% for frequently accessed tables
-- - Cache hit ratio > 99%
-- - No table bloat
-- - Efficient query plans
-- - Reduced disk I/O
-- - Better connection pool utilization
