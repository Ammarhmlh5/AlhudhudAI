-- 071_global_event_progress_tracking.sql
-- Adding support for real-time progress monitoring of bulk ingestion.

ALTER TABLE global_incoming_events
ADD COLUMN IF NOT EXISTS total_items            INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS processed_items        INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS failed_items           INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS processing_started_at  TIMESTAMP,
ADD COLUMN IF NOT EXISTS processing_finished_at TIMESTAMP;

-- Performance: Index for finding active bulk jobs
CREATE INDEX IF NOT EXISTS idx_global_events_active 
ON global_incoming_events(status) 
WHERE status = 'PROCESSING';
