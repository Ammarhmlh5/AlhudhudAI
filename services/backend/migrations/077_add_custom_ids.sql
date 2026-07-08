-- Migration 077: Add Custom ID System
-- Purpose: Add Redis-generated IDs for events and messages
-- Date: 2026-03-23

-- 1. Add custom_id to global_incoming_events
ALTER TABLE global_incoming_events 
    ADD COLUMN IF NOT EXISTS custom_id VARCHAR(20) UNIQUE;

-- Index for fast lookup by custom_id
CREATE INDEX IF NOT EXISTS idx_global_incoming_events_custom_id 
    ON global_incoming_events(custom_id);

COMMENT ON COLUMN global_incoming_events.custom_id IS 
    'Redis-generated Event ID (Format: EV + YYMMDDHHmmss + TenantHash + Sequence). Example: EV2601012312345000100';

-- 2. Add custom_id to messages
ALTER TABLE messages 
    ADD COLUMN IF NOT EXISTS custom_id VARCHAR(23) UNIQUE;

-- Index for fast lookup by custom_id
CREATE INDEX IF NOT EXISTS idx_messages_custom_id 
    ON messages(custom_id);

COMMENT ON COLUMN messages.custom_id IS 
    'Redis-generated Message ID (Format: MS + YYMMDDHHmmss + TenantHash + EventSeq + MsgSeq). Example: MS26010123123450001001';

-- 3. Add event_custom_id reference to messages
ALTER TABLE messages 
    ADD COLUMN IF NOT EXISTS event_custom_id VARCHAR(20);

-- Index for fast lookup by event
CREATE INDEX IF NOT EXISTS idx_messages_event_custom_id 
    ON messages(event_custom_id);

COMMENT ON COLUMN messages.event_custom_id IS 
    'Reference to the Event custom_id that this message belongs to';

-- 4. Keep existing id columns as internal counters
COMMENT ON COLUMN global_incoming_events.id IS 
    'Internal PostgreSQL counter for admin statistics only. Use custom_id for tracking and tracing.';

COMMENT ON COLUMN messages.id IS 
    'Internal PostgreSQL counter for admin statistics only. Use custom_id for tracking and tracing.';

-- 5. Create view for easy querying with custom IDs
CREATE OR REPLACE VIEW v_messages_with_custom_ids AS
SELECT 
    m.custom_id as message_id,
    m.event_custom_id as event_id,
    m.tenant_id,
    m.phone_number,
    m.content,
    m.status,
    m.cost,
    m.created_at,
    m.updated_at,
    ge.custom_id as event_custom_id_full,
    ge.sender_ip as event_sender_ip,
    ge.created_at as event_created_at
FROM messages m
LEFT JOIN global_incoming_events ge ON m.event_custom_id = ge.custom_id
WHERE m.custom_id IS NOT NULL;

COMMENT ON VIEW v_messages_with_custom_ids IS 
    'View for querying messages with their custom IDs and event information';

-- 6. Create function to extract timestamp from custom_id
CREATE OR REPLACE FUNCTION extract_timestamp_from_custom_id(custom_id VARCHAR)
RETURNS TIMESTAMP AS $$
DECLARE
    timestamp_part VARCHAR(8);
    year INT;
    month INT;
    day INT;
    hour INT;
BEGIN
    -- Extract timestamp part (positions 3-10 for both EV and MS)
    timestamp_part := SUBSTRING(custom_id FROM 3 FOR 8);
    
    -- Parse: YYMMDDHHmmss
    year := 2000 + CAST(SUBSTRING(timestamp_part FROM 1 FOR 2) AS INT);
    month := CAST(SUBSTRING(timestamp_part FROM 3 FOR 2) AS INT);
    day := CAST(SUBSTRING(timestamp_part FROM 5 FOR 2) AS INT);
    hour := CAST(SUBSTRING(timestamp_part FROM 7 FOR 2) AS INT);
    
    RETURN make_timestamp(year, month, day, hour, 0, 0);
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION extract_timestamp_from_custom_id IS 
    'Extracts timestamp from custom_id (EV or MS format)';

-- 7. Create function to extract tenant hash from custom_id
CREATE OR REPLACE FUNCTION extract_tenant_hash_from_custom_id(custom_id VARCHAR)
RETURNS VARCHAR AS $$
BEGIN
    -- Extract tenant hash (positions 11-15 for both EV and MS)
    RETURN SUBSTRING(custom_id FROM 11 FOR 5);
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION extract_tenant_hash_from_custom_id IS 
    'Extracts tenant hash from custom_id (EV or MS format)';

-- 8. Create analytics view for load monitoring
CREATE OR REPLACE VIEW v_event_load_analytics AS
SELECT 
    extract_timestamp_from_custom_id(custom_id) as event_hour,
    extract_tenant_hash_from_custom_id(custom_id) as tenant_hash,
    COUNT(*) as event_count,
    SUM(total_items) as total_messages,
    AVG(total_items) as avg_messages_per_event,
    MAX(total_items) as max_messages_per_event,
    COUNT(CASE WHEN total_items >= 900 THEN 1 END) as near_limit_events
FROM global_incoming_events
WHERE custom_id IS NOT NULL
GROUP BY event_hour, tenant_hash
ORDER BY event_hour DESC;

COMMENT ON VIEW v_event_load_analytics IS 
    'Analytics view for monitoring event load and detecting pressure';

-- 9. Backfill existing records with custom_id values
-- This ensures existing production data gets custom_id values
-- Process in batches to avoid locking issues

DO $$
DECLARE
    batch_size INT := 1000;
    total_events INT;
    total_messages INT;
    processed_events INT := 0;
    processed_messages INT := 0;
    batch_count INT := 0;
BEGIN
    -- Count records needing backfill
    SELECT COUNT(*) INTO total_events FROM global_incoming_events WHERE custom_id IS NULL;
    SELECT COUNT(*) INTO total_messages FROM messages WHERE custom_id IS NULL;
    
    RAISE NOTICE 'Starting custom_id backfill...';
    RAISE NOTICE 'Events to backfill: %', total_events;
    RAISE NOTICE 'Messages to backfill: %', total_messages;
    
    -- Backfill events table
    -- Format: EV + YYMMDD + HH + TenantHash (5 chars) + Sequence (5 chars) = 20 chars
    IF total_events > 0 THEN
        RAISE NOTICE 'Backfilling events table...';
        
        LOOP
            WITH batch AS (
                SELECT id, created_at, tenant_id
                FROM global_incoming_events
                WHERE custom_id IS NULL
                ORDER BY id
                LIMIT batch_size
                FOR UPDATE SKIP LOCKED
            )
            UPDATE global_incoming_events e
            SET custom_id = 'EV' || 
                TO_CHAR(b.created_at, 'YYMMDD') ||
                LPAD(EXTRACT(HOUR FROM b.created_at)::TEXT, 2, '0') ||
                LPAD(SUBSTR(MD5(b.tenant_id::TEXT), 1, 5), 5, '0') ||
                LPAD(b.id::TEXT, 5, '0')
            FROM batch b
            WHERE e.id = b.id;
            
            GET DIAGNOSTICS batch_count = ROW_COUNT;
            processed_events := processed_events + batch_count;
            
            RAISE NOTICE 'Backfilled % events (% / %)', batch_count, processed_events, total_events;
            
            EXIT WHEN batch_count = 0;
            
            -- Small delay to avoid overwhelming the database
            PERFORM pg_sleep(0.1);
        END LOOP;
        
        RAISE NOTICE 'Events backfill complete: % records updated', processed_events;
    ELSE
        RAISE NOTICE 'No events need backfill';
    END IF;
    
    -- Backfill messages table
    -- Format: MS + YYMMDD + HH + TenantHash (5 chars) + EventSeq (4 chars) + MsgSeq (3 chars) = 23 chars
    IF total_messages > 0 THEN
        RAISE NOTICE 'Backfilling messages table...';
        
        processed_messages := 0;
        batch_count := 0;
        
        LOOP
            WITH batch AS (
                SELECT m.id, m.created_at, m.tenant_id, m.global_event_id
                FROM messages m
                WHERE m.custom_id IS NULL
                ORDER BY m.id
                LIMIT batch_size
                FOR UPDATE SKIP LOCKED
            )
            UPDATE messages m
            SET custom_id = 'MS' || 
                TO_CHAR(b.created_at, 'YYMMDD') ||
                LPAD(EXTRACT(HOUR FROM b.created_at)::TEXT, 2, '0') ||
                LPAD(SUBSTR(MD5(b.tenant_id::TEXT), 1, 5), 5, '0') ||
                LPAD(COALESCE(b.global_event_id::TEXT, '0'), 4, '0') ||
                LPAD(SUBSTRING(b.id::TEXT FROM 1 FOR 3), 3, '0')
            FROM batch b
            WHERE m.id = b.id;
            
            GET DIAGNOSTICS batch_count = ROW_COUNT;
            processed_messages := processed_messages + batch_count;
            
            RAISE NOTICE 'Backfilled % messages (% / %)', batch_count, processed_messages, total_messages;
            
            EXIT WHEN batch_count = 0;
            
            -- Small delay to avoid overwhelming the database
            PERFORM pg_sleep(0.1);
        END LOOP;
        
        RAISE NOTICE 'Messages backfill complete: % records updated', processed_messages;
    ELSE
        RAISE NOTICE 'No messages need backfill';
    END IF;
    
    -- Update event_custom_id references in messages table
    -- Link messages to their events using custom_id
    RAISE NOTICE 'Updating event_custom_id references...';
    
    processed_messages := 0;
    batch_count := 0;
    
    LOOP
        WITH batch AS (
            SELECT m.id, e.custom_id as event_custom_id
            FROM messages m
            INNER JOIN global_incoming_events e ON m.global_event_id = e.id
            WHERE m.event_custom_id IS NULL 
              AND e.custom_id IS NOT NULL
            ORDER BY m.id
            LIMIT batch_size
            FOR UPDATE OF m SKIP LOCKED
        )
        UPDATE messages m
        SET event_custom_id = b.event_custom_id
        FROM batch b
        WHERE m.id = b.id;
        
        GET DIAGNOSTICS batch_count = ROW_COUNT;
        processed_messages := processed_messages + batch_count;
        
        RAISE NOTICE 'Updated % message event references (total: %)', batch_count, processed_messages;
        
        EXIT WHEN batch_count = 0;
        
        -- Small delay to avoid overwhelming the database
        PERFORM pg_sleep(0.1);
    END LOOP;
    
    RAISE NOTICE 'Event reference update complete: % records updated', processed_messages;
    RAISE NOTICE 'Custom ID backfill complete!';
END $$;
