-- Query Optimization Scripts for Tariff Pricing System
-- This file contains EXPLAIN ANALYZE queries and optimization recommendations

-- ============================================================================
-- 1. TARIFF QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query (without proper indexing)
EXPLAIN ANALYZE
SELECT * FROM tariffs 
WHERE country_code = 'YE' 
  AND network_operator = 'Yemen Mobile' 
  AND is_active = true;

-- Optimized query (with tenant_id and specific columns)
EXPLAIN ANALYZE
SELECT id, tenant_id, service_id, cost_per_message, currency
FROM tariffs 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND service_id = 1
  AND country_code = 'YE' 
  AND network_operator = 'Yemen Mobile' 
  AND is_active = true
  AND effective_from <= CURRENT_DATE
  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
ORDER BY effective_from DESC
LIMIT 1;

-- ============================================================================
-- 2. SUBSCRIPTION QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query
EXPLAIN ANALYZE
SELECT * FROM subscriptions 
WHERE user_id = '123e4567-e89b-12d3-a456-426614174000'
  AND status = 'active';

-- Optimized query
EXPLAIN ANALYZE
SELECT id, tenant_id, user_id, package_id, status, 
       messages_included, messages_used, messages_remaining,
       end_date, next_renewal_date
FROM subscriptions 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND user_id = '987fcdeb-51a2-43f7-b890-123456789abc'
  AND status IN ('active', 'grace_period')
  AND end_date >= CURRENT_DATE
ORDER BY end_date DESC
LIMIT 1;

-- ============================================================================
-- 3. ROLLOVER MESSAGES QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query
EXPLAIN ANALYZE
SELECT * FROM rollover_messages 
WHERE subscription_id = '123e4567-e89b-12d3-a456-426614174000'
  AND status = 'active'
ORDER BY created_at;

-- Optimized query (FIFO with proper indexing)
EXPLAIN ANALYZE
SELECT id, tenant_id, subscription_id, messages_remaining, expiry_date
FROM rollover_messages 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND subscription_id = '987fcdeb-51a2-43f7-b890-123456789abc'
  AND status = 'active'
  AND messages_remaining > 0
  AND expiry_date >= CURRENT_DATE
ORDER BY created_at ASC;

-- ============================================================================
-- 4. GATEWAY STATISTICS QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query (full table scan)
EXPLAIN ANALYZE
SELECT * FROM gateway_statistics 
WHERE gateway_id = 1
  AND date >= '2024-01-01';

-- Optimized query (with tenant_id and date range)
EXPLAIN ANALYZE
SELECT gateway_id, date, messages_sent, messages_delivered, 
       avg_response_time, uptime_percentage
FROM gateway_statistics 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND gateway_id = 1
  AND date BETWEEN '2024-01-01' AND '2024-12-31'
ORDER BY date DESC;

-- ============================================================================
-- 5. RESOURCE USAGE QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query (time-series without proper indexing)
EXPLAIN ANALYZE
SELECT * FROM resource_usage 
WHERE gateway_id = 1
  AND timestamp >= NOW() - INTERVAL '1 hour';

-- Optimized query (with tenant_id and time-series index)
EXPLAIN ANALYZE
SELECT gateway_id, timestamp, cpu_usage, memory_usage, 
       bandwidth_usage, active_connections
FROM resource_usage 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND gateway_id = 1
  AND timestamp BETWEEN NOW() - INTERVAL '1 hour' AND NOW()
ORDER BY timestamp DESC;

-- ============================================================================
-- 6. ACCOUNTING TRANSACTIONS QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query (without pagination optimization)
EXPLAIN ANALYZE
SELECT * FROM accounting_transactions 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
ORDER BY created_at DESC
LIMIT 100 OFFSET 1000;

-- Optimized query (cursor-based pagination)
EXPLAIN ANALYZE
SELECT id, tenant_id, transaction_type, amount, currency, 
       balance_before, balance_after, created_at
FROM accounting_transactions 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND created_at < '2024-01-01 00:00:00'
ORDER BY created_at DESC
LIMIT 100;

-- ============================================================================
-- 7. BALANCE HISTORY QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query
EXPLAIN ANALYZE
SELECT * FROM balance_history 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
ORDER BY created_at DESC;

-- Optimized query (with date range and specific columns)
EXPLAIN ANALYZE
SELECT id, tenant_id, amount, change_type, reason, created_at
FROM balance_history 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND created_at >= NOW() - INTERVAL '30 days'
ORDER BY created_at DESC
LIMIT 100;

-- ============================================================================
-- 8. COMPLEX JOIN QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query (multiple joins without proper indexing)
EXPLAIN ANALYZE
SELECT s.*, p.name, p.price, u.email
FROM subscriptions s
JOIN packages p ON s.package_id = p.id
JOIN users u ON s.user_id = u.id
WHERE s.status = 'active';

-- Optimized query (with tenant_id and specific columns)
EXPLAIN ANALYZE
SELECT s.id, s.status, s.messages_remaining, 
       p.name AS package_name, p.price,
       u.email
FROM subscriptions s
INNER JOIN packages p ON s.tenant_id = p.tenant_id AND s.package_id = p.id
INNER JOIN users u ON s.tenant_id = u.tenant_id AND s.user_id = u.id
WHERE s.tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND s.status = 'active'
  AND s.end_date >= CURRENT_DATE;

-- ============================================================================
-- 9. AGGREGATION QUERIES OPTIMIZATION
-- ============================================================================

-- Original slow query (full table aggregation)
EXPLAIN ANALYZE
SELECT COUNT(*), SUM(amount)
FROM accounting_transactions
WHERE transaction_type = 'subscription';

-- Optimized query (with tenant_id and date range)
EXPLAIN ANALYZE
SELECT 
  COUNT(*) AS transaction_count,
  SUM(amount) AS total_amount,
  AVG(amount) AS avg_amount
FROM accounting_transactions
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND transaction_type = 'subscription'
  AND created_at >= NOW() - INTERVAL '30 days';

-- ============================================================================
-- 10. SUBQUERY OPTIMIZATION
-- ============================================================================

-- Original slow query (correlated subquery)
EXPLAIN ANALYZE
SELECT t.*, 
  (SELECT COUNT(*) FROM subscriptions s WHERE s.package_id = p.id) AS subscriber_count
FROM packages p
JOIN tariffs t ON p.service_id = t.service_id;

-- Optimized query (using JOIN instead of subquery)
EXPLAIN ANALYZE
SELECT p.id, p.name, p.price, COUNT(s.id) AS subscriber_count
FROM packages p
LEFT JOIN subscriptions s ON p.tenant_id = s.tenant_id 
  AND p.id = s.package_id 
  AND s.status = 'active'
WHERE p.tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND p.is_active = true
GROUP BY p.id, p.name, p.price;

-- ============================================================================
-- 11. QUERY PERFORMANCE ANALYSIS
-- ============================================================================

-- Find slow queries from pg_stat_statements
SELECT 
  query,
  calls,
  total_exec_time,
  mean_exec_time,
  max_exec_time,
  rows
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- queries taking more than 100ms on average
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Find queries with high I/O
SELECT 
  query,
  calls,
  shared_blks_hit,
  shared_blks_read,
  shared_blks_dirtied,
  shared_blks_written
FROM pg_stat_statements
WHERE shared_blks_read > 1000
ORDER BY shared_blks_read DESC
LIMIT 20;

-- ============================================================================
-- 12. INDEX USAGE ANALYSIS
-- ============================================================================

-- Find unused indexes
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan,
  idx_tup_read,
  idx_tup_fetch
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexname NOT LIKE '%_pkey'
ORDER BY schemaname, tablename;

-- Find missing indexes (tables with sequential scans)
SELECT 
  schemaname,
  tablename,
  seq_scan,
  seq_tup_read,
  idx_scan,
  seq_tup_read / seq_scan AS avg_seq_tup_read
FROM pg_stat_user_tables
WHERE seq_scan > 0
  AND seq_tup_read / seq_scan > 1000
ORDER BY seq_tup_read DESC
LIMIT 20;

-- ============================================================================
-- 13. TABLE BLOAT ANALYSIS
-- ============================================================================

-- Check table bloat
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
  n_live_tup,
  n_dead_tup,
  ROUND(100 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_tup_percent
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
ORDER BY n_dead_tup DESC;

-- ============================================================================
-- 14. VACUUM AND ANALYZE RECOMMENDATIONS
-- ============================================================================

-- Tables that need VACUUM
SELECT 
  schemaname,
  tablename,
  last_vacuum,
  last_autovacuum,
  n_dead_tup
FROM pg_stat_user_tables
WHERE n_dead_tup > 1000
  AND (last_autovacuum IS NULL OR last_autovacuum < NOW() - INTERVAL '1 day')
ORDER BY n_dead_tup DESC;

-- Tables that need ANALYZE
SELECT 
  schemaname,
  tablename,
  last_analyze,
  last_autoanalyze,
  n_mod_since_analyze
FROM pg_stat_user_tables
WHERE n_mod_since_analyze > 1000
  AND (last_autoanalyze IS NULL OR last_autoanalyze < NOW() - INTERVAL '1 day')
ORDER BY n_mod_since_analyze DESC;

-- ============================================================================
-- 15. QUERY REWRITE EXAMPLES
-- ============================================================================

-- Example 1: Replace IN with EXISTS
-- BEFORE:
-- SELECT * FROM subscriptions WHERE package_id IN (SELECT id FROM packages WHERE price > 50);

-- AFTER:
EXPLAIN ANALYZE
SELECT s.id, s.tenant_id, s.status
FROM subscriptions s
WHERE EXISTS (
  SELECT 1 FROM packages p 
  WHERE p.id = s.package_id 
    AND p.tenant_id = s.tenant_id
    AND p.price > 50
);

-- Example 2: Replace OR with UNION
-- BEFORE:
-- SELECT * FROM tariffs WHERE country_code = 'YE' OR country_code = 'SA';

-- AFTER:
EXPLAIN ANALYZE
SELECT id, tenant_id, country_code, cost_per_message
FROM tariffs 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND country_code = 'YE'
UNION ALL
SELECT id, tenant_id, country_code, cost_per_message
FROM tariffs 
WHERE tenant_id = '123e4567-e89b-12d3-a456-426614174000'
  AND country_code = 'SA';

-- Example 3: Use LATERAL JOIN for correlated subqueries
-- BEFORE:
-- SELECT p.*, (SELECT COUNT(*) FROM subscriptions WHERE package_id = p.id) FROM packages p;

-- AFTER:
EXPLAIN ANALYZE
SELECT p.id, p.name, p.price, s.subscriber_count
FROM packages p
CROSS JOIN LATERAL (
  SELECT COUNT(*) AS subscriber_count
  FROM subscriptions
  WHERE package_id = p.id
    AND tenant_id = p.tenant_id
    AND status = 'active'
) s
WHERE p.tenant_id = '123e4567-e89b-12d3-a456-426614174000';

-- ============================================================================
-- 16. PREPARED STATEMENT EXAMPLES
-- ============================================================================

-- Prepare frequently used queries
PREPARE get_active_tariff (uuid, int, varchar, varchar) AS
SELECT id, tenant_id, service_id, cost_per_message, currency
FROM tariffs 
WHERE tenant_id = $1
  AND service_id = $2
  AND country_code = $3
  AND network_operator = $4
  AND is_active = true
  AND effective_from <= CURRENT_DATE
  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
ORDER BY effective_from DESC
LIMIT 1;

-- Execute prepared statement
EXECUTE get_active_tariff('123e4567-e89b-12d3-a456-426614174000', 1, 'YE', 'Yemen Mobile');

-- Deallocate when done
DEALLOCATE get_active_tariff;

-- ============================================================================
-- 17. OPTIMIZATION RECOMMENDATIONS
-- ============================================================================

/*
GENERAL OPTIMIZATION GUIDELINES:

1. Always include tenant_id in WHERE clause (first condition)
2. Use specific column names instead of SELECT *
3. Add appropriate indexes on frequently queried columns
4. Use LIMIT for queries that may return many rows
5. Avoid correlated subqueries, use JOINs instead
6. Use prepared statements for frequently executed queries
7. Add date ranges to time-series queries
8. Use cursor-based pagination instead of OFFSET
9. Regularly run VACUUM and ANALYZE
10. Monitor query performance with pg_stat_statements

INDEX RECOMMENDATIONS:

1. Composite indexes should start with tenant_id
2. Include columns used in WHERE, JOIN, and ORDER BY
3. Consider partial indexes for filtered queries
4. Use covering indexes for frequently accessed columns
5. Remove unused indexes to improve write performance

QUERY REWRITE PATTERNS:

1. Replace IN with EXISTS for better performance
2. Replace OR with UNION for indexed columns
3. Use LATERAL JOIN for correlated subqueries
4. Use window functions instead of self-joins
5. Use CTEs for complex queries with multiple references
*/
