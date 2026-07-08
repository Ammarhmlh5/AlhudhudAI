-- Migration: 078_message_delivery_logs
-- Purpose: Create message_delivery_logs table to track message delivery to different channels (mobile, SMS, etc.)
-- Created: 2026-05-23
-- Part of: mobile-message-delivery-fix

-- Create message_delivery_logs table
CREATE TABLE IF NOT EXISTS message_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    device_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    delivery_channel VARCHAR(20) NOT NULL, -- 'MOBILE', 'SMS', etc.
    delivered BOOLEAN DEFAULT FALSE,
    delivered_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_message_id ON message_delivery_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_tenant_id ON message_delivery_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_device_id ON message_delivery_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_delivery_channel ON message_delivery_logs(delivery_channel);
CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_delivered ON message_delivery_logs(delivered) WHERE delivered = FALSE;

-- Add comments for documentation
COMMENT ON TABLE message_delivery_logs IS 'Tracks message delivery status across different channels (mobile, SMS, etc.)';
COMMENT ON COLUMN message_delivery_logs.delivery_channel IS 'Channel type: MOBILE, SMS, WEB, etc.';
COMMENT ON COLUMN message_delivery_logs.delivered_at IS 'Timestamp when the message was successfully delivered';
COMMENT ON COLUMN message_delivery_logs.error_message IS 'Error details if delivery failed';