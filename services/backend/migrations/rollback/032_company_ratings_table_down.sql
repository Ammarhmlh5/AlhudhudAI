-- Rollback: Drop company_ratings table
-- Migration: 032_company_ratings_table.sql

BEGIN;

-- Optional: Backup data before dropping (uncomment if needed)
-- CREATE TABLE company_ratings_backup AS SELECT * FROM company_ratings;

-- Drop RLS policies
DROP POLICY IF EXISTS company_ratings_isolation ON company_ratings;

-- Disable RLS
ALTER TABLE IF EXISTS company_ratings DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_company_ratings_tenant;
DROP INDEX IF EXISTS idx_company_ratings_category;
DROP INDEX IF EXISTS idx_company_ratings_updated;

-- Drop table
DROP TABLE IF EXISTS company_ratings CASCADE;

-- Log rollback
INSERT INTO migration_log (migration_name, action, executed_at)
VALUES ('032_company_ratings_table', 'rollback', NOW())
ON CONFLICT DO NOTHING;

COMMIT;
