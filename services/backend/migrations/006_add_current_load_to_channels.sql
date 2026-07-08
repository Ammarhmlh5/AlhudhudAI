-- Migration: Add current_load column to dispatch_channels table
-- Purpose: Support load balancing by tracking current message load per channel
-- Requirement: 5.6 - Load Balancing

-- Add current_load column with default value 0
ALTER TABLE dispatch_channels 
ADD COLUMN IF NOT EXISTS current_load INTEGER DEFAULT 0 NOT NULL;

-- Add check constraint to prevent negative values
ALTER TABLE dispatch_channels 
ADD CONSTRAINT check_current_load_non_negative 
CHECK (current_load >= 0);

-- Add index for load-based queries (used in channel selection)
CREATE INDEX IF NOT EXISTS idx_dispatch_channels_load 
ON dispatch_channels(tenant_id, status, priority DESC, current_load ASC)
WHERE status = 'ONLINE';

-- Add comment for documentation
COMMENT ON COLUMN dispatch_channels.current_load IS 
'Current number of messages being processed by this channel. Updated by IncrementChannelLoad/DecrementChannelLoad methods.';
