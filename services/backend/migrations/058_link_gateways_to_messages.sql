-- Migration: Link Messages and Channels to Dedicated Gateways
-- Purpose: Enable efficient aggregation and reporting without N+1 joins

-- 1. Update dispatch_channels
ALTER TABLE dispatch_channels ADD COLUMN IF NOT EXISTS gateway_id UUID REFERENCES dedicated_gateways(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_dispatch_channels_gateway ON dispatch_channels(gateway_id);

-- 2. Update messages
-- Ensure channel_id exists (it was in model but maybe not in DB correctly as UUID)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS gateway_id UUID REFERENCES dedicated_gateways(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_messages_gateway_id ON messages(gateway_id);
CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);

-- 3. Comments
COMMENT ON COLUMN dispatch_channels.gateway_id IS 'Link to dedicated gateway if this channel represents one';
COMMENT ON COLUMN messages.gateway_id IS 'Stored at send-time to allow fast daily aggregation';
