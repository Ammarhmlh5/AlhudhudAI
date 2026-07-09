-- Migration: Create dead_letter_queue table
-- Implements Requirements 17.5, 17.6
-- Purpose: Store messages that failed all retry attempts

-- Create dead_letter_queue table
CREATE TABLE IF NOT EXISTS dead_letter_queue (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    message_id UUID NOT NULL,
    phone_number VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    original_status VARCHAR(20) NOT NULL,
    fail_reason TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    cost DECIMAL(10, 4) NOT NULL DEFAULT 0,
    moved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT dlq_tenant_message_unique UNIQUE(tenant_id, message_id)
);

-- Indexes for performance
CREATE INDEX idx_dlq_tenant_moved ON dead_letter_queue(tenant_id, moved_at DESC);
CREATE INDEX idx_dlq_message_id ON dead_letter_queue(message_id);
CREATE INDEX idx_dlq_moved_at ON dead_letter_queue(moved_at);

-- Add comment
COMMENT ON TABLE dead_letter_queue IS 'Stores messages that failed all retry attempts (Requirement 17.5)';
COMMENT ON COLUMN dead_letter_queue.retry_count IS 'Number of retry attempts before moving to DLQ';
COMMENT ON COLUMN dead_letter_queue.moved_at IS 'Timestamp when message was moved to DLQ';

-- Add DEAD_LETTER status to messages table if not exists
DO $$
BEGIN
    -- Check if DEAD_LETTER status is already in the constraint
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'messages_status_check' 
        AND conbin::text LIKE '%DEAD_LETTER%'
    ) THEN
        -- Drop old constraint
        ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_status_check;
        
        -- Add new constraint with DEAD_LETTER status
        ALTER TABLE messages ADD CONSTRAINT messages_status_check 
        CHECK (status IN ('PENDING', 'QUEUED', 'PROCESSING', 'SENT', 'DELIVERED', 'FAILED', 'SCHEDULED', 'WAITING', 'DEAD_LETTER'));
    END IF;
END $$;

-- Add fail_reason column to messages table if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'messages' AND column_name = 'fail_reason'
    ) THEN
        ALTER TABLE messages ADD COLUMN fail_reason TEXT;
    END IF;
END $$;

COMMENT ON COLUMN messages.fail_reason IS 'Reason for message failure';
