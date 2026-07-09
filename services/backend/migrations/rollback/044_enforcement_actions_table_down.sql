-- Rollback: Drop enforcement_actions table
-- Migration: 044_enforcement_actions_table.sql

BEGIN;

-- Drop indexes
DROP INDEX IF EXISTS idx_enforcement_actions_tenant;
DROP INDEX IF EXISTS idx_enforcement_actions_type;

-- Drop table
DROP TABLE IF EXISTS enforcement_actions CASCADE;

-- Log rollback
INSERT INTO migration_log (migration_name, action, executed_at)
VALUES ('044_enforcement_actions_table', 'rollback', NOW())
ON CONFLICT DO NOTHING;

COMMIT;
