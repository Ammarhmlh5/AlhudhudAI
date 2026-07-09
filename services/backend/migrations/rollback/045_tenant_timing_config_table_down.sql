-- Rollback: Drop tenant_timing_config table
-- Migration: 045_tenant_timing_config_table.sql

BEGIN;

-- Drop table
DROP TABLE IF EXISTS tenant_timing_config CASCADE;

-- Log rollback
INSERT INTO migration_log (migration_name, action, executed_at)
VALUES ('045_tenant_timing_config_table', 'rollback', NOW())
ON CONFLICT DO NOTHING;

COMMIT;
