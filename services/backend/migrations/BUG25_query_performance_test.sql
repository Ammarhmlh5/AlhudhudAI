-- Bug 25: Query Performance Test
-- Purpose: Verify that idx_messages_event_custom_id index exists and is being used
-- Date: 2025-01-XX

-- ============================================================================
-- TEST 1: Verify Index Exists
-- ============================================================================

SELECT 
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes 
WHERE tablename = 'messages' 
  AND indexname = 'idx_messages_event_custom_id';

-- Expected Output:
-- schemaname | tablename | indexname                      | indexdef
-- -----------+-----------+--------------------------------+------------------------------------------
-- public     | messages  | idx_messages_event_custom_id   | CREATE INDEX idx_messages_event_custom_id ON public.messages USING btree (event_custom_id)

-- ============================================================================
-- TEST 2: Query Performance - EXPLAIN ANALYZE
-- ============================================================================

-- Test query: Find all messages for a specific event
EXPLAIN ANALYZE 
SELECT * FROM messages 
WHERE event_custom_id = 'EV2601012312345000100';

-- Expected Output (Index Scan):
-- Index Scan using idx_messages_event_custom_id on messages
-- Index Cond: ((event_custom_id)::text = 'EV2601012312345000100'::text)
-- Planning Time: X ms
-- Execution Time: X ms

-- NOT Expected (Seq Scan):
-- Seq Scan on messages
-- Filter: ((event_custom_id)::text = 'EV2601012312345000100'::text)

-- ============================================================================
-- TEST 3: Query Performance - Count Messages by Event
-- ============================================================================

EXPLAIN ANALYZE 
SELECT COUNT(*) FROM messages 
WHERE event_custom_id = 'EV2601012312345000100';

-- Expected: Index Scan or Index Only Scan

-- ============================================================================
-- TEST 4: Query Performance - Filter by Event and Status
-- ============================================================================

EXPLAIN ANALYZE 
SELECT * FROM messages 
WHERE event_custom_id = 'EV2601012312345000100'
  AND status = 'sent';

-- Expected: Index Scan using idx_messages_event_custom_id
-- Note: May benefit from composite index (event_custom_id, status)

-- ============================================================================
-- TEST 5: Query Performance - Filter by Event with Time Range
-- ============================================================================

EXPLAIN ANALYZE 
SELECT * FROM messages 
WHERE event_custom_id = 'EV2601012312345000100'
  AND created_at >= NOW() - INTERVAL '1 day'
ORDER BY created_at DESC;

-- Expected: Index Scan using idx_messages_event_custom_id
-- Note: May benefit from composite index (event_custom_id, created_at)

-- ============================================================================
-- TEST 6: Index Statistics
-- ============================================================================

SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
WHERE indexname = 'idx_messages_event_custom_id';

-- Shows how often the index is being used

-- ============================================================================
-- TEST 7: Table Statistics
-- ============================================================================

SELECT 
    schemaname,
    tablename,
    n_live_tup as live_rows,
    n_dead_tup as dead_rows,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE tablename = 'messages';

-- Shows table size and maintenance status

-- ============================================================================
-- PERFORMANCE COMPARISON
-- ============================================================================

-- Without Index (Hypothetical - DO NOT RUN):
-- DROP INDEX idx_messages_event_custom_id;
-- EXPLAIN ANALYZE SELECT * FROM messages WHERE event_custom_id = 'EV...';
-- Result: Seq Scan on messages (cost=0.00..X rows=Y width=Z)
-- Time: SLOW (proportional to table size)

-- With Index (Current State):
-- EXPLAIN ANALYZE SELECT * FROM messages WHERE event_custom_id = 'EV...';
-- Result: Index Scan using idx_messages_event_custom_id (cost=0.42..X rows=Y width=Z)
-- Time: FAST (logarithmic lookup)

-- ============================================================================
-- EXPECTED RESULTS SUMMARY
-- ============================================================================

-- ✅ Index exists: idx_messages_event_custom_id
-- ✅ Index type: B-tree
-- ✅ Query plan: Index Scan (NOT Seq Scan)
-- ✅ Performance: Fast lookup (O(log n))

-- ============================================================================
-- OPTIONAL COMPOSITE INDEXES (Task 25.3)
-- ============================================================================

-- Composite Index 1: event_custom_id + status
-- Use case: Queries filtering by both event and status
-- CREATE INDEX IF NOT EXISTS idx_messages_event_status 
--     ON messages(event_custom_id, status);

-- Composite Index 2: event_custom_id + created_at
-- Use case: Time-based queries for specific events
-- CREATE INDEX IF NOT EXISTS idx_messages_event_created 
--     ON messages(event_custom_id, created_at DESC);

-- Note: Only create composite indexes if query patterns show they are needed
-- Each index adds overhead to INSERT/UPDATE operations

-- ============================================================================
-- END OF TESTS
-- ============================================================================
