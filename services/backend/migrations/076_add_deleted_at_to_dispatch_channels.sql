-- Migration: Add missing deleted_at column to dispatch_channels
-- Purpose: Fix GRORM model mismatch causing backend errors

ALTER TABLE dispatch_channels ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_dispatch_channels_deleted_at ON dispatch_channels(deleted_at);
