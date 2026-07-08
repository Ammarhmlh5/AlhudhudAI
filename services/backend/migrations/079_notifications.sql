-- Migration 079: Notifications table with tenant isolation
-- Provides complete isolation between tenants (المستأجرين)

-- Create notifications table with tenant isolation
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,  -- The user/owner of this notification
    type VARCHAR(50) NOT NULL CHECK (type IN ('message', 'status', 'system', 'delivery', 'alert')),
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    data JSONB DEFAULT '{}',
    read_at TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Tenant isolation constraint - notifications belong to a specific tenant
    CONSTRAINT fk_notifications_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- User constraint
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for efficient queries with tenant isolation
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_id 
    ON notifications(tenant_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id 
    ON notifications(user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_tenant_unread 
    ON notifications(tenant_id, read_at) 
    WHERE read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_tenant_created 
    ON notifications(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created 
    ON notifications(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_type 
    ON notifications(tenant_id, type);

-- Composite index for common query pattern
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_user_unread 
    ON notifications(tenant_id, user_id, created_at DESC) 
    WHERE read_at IS NULL;

-- Function to get unread count for a specific tenant and user
CREATE OR REPLACE FUNCTION get_unread_notifications_count(
    p_tenant_id UUID,
    p_user_id UUID DEFAULT NULL
) RETURNS INTEGER AS $$
BEGIN
    IF p_user_id IS NULL THEN
        RETURN (
            SELECT COUNT(*) FROM notifications 
            WHERE tenant_id = p_tenant_id AND read_at IS NULL
        );
    ELSE
        RETURN (
            SELECT COUNT(*) FROM notifications 
            WHERE tenant_id = p_tenant_id 
            AND user_id = p_user_id 
            AND read_at IS NULL
        );
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to mark notification as read with tenant validation
CREATE OR REPLACE FUNCTION mark_notification_read(
    p_notification_id UUID,
    p_tenant_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    UPDATE notifications 
    SET read_at = NOW()
    WHERE id = p_notification_id 
    AND tenant_id = p_tenant_id
    AND read_at IS NULL;
    
    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count > 0;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to mark all notifications as read for a tenant/user
CREATE OR REPLACE FUNCTION mark_all_notifications_read(
    p_tenant_id UUID,
    p_user_id UUID DEFAULT NULL
) RETURNS INTEGER AS $$
DECLARE
    v_updated_count INTEGER;
BEGIN
    IF p_user_id IS NULL THEN
        UPDATE notifications 
        SET read_at = NOW()
        WHERE tenant_id = p_tenant_id AND read_at IS NULL;
    ELSE
        UPDATE notifications 
        SET read_at = NOW()
        WHERE tenant_id = p_tenant_id 
        AND user_id = p_user_id 
        AND read_at IS NULL;
    END IF;
    
    GET DIAGNOSTICS v_updated_count = ROW_COUNT;
    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to create notification with tenant isolation
CREATE OR REPLACE FUNCTION create_notification(
    p_tenant_id UUID,
    p_user_id UUID,
    p_type VARCHAR(50),
    p_title VARCHAR(255),
    p_body TEXT,
    p_data JSONB DEFAULT '{}'::JSONB
) RETURNS notifications AS $$
DECLARE
    v_notification notifications;
BEGIN
    -- Verify tenant isolation - only create notification for valid tenant
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE id = p_tenant_id) THEN
        RAISE EXCEPTION 'Invalid tenant_id: tenant does not exist';
    END IF;

    INSERT INTO notifications (tenant_id, user_id, type, title, body, data)
    VALUES (p_tenant_id, p_user_id, p_type, p_title, p_body, p_data)
    RETURNING * INTO v_notification;

    RETURN v_notification;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to delete old notifications for a tenant (with isolation)
CREATE OR REPLACE FUNCTION delete_old_notifications(
    p_tenant_id UUID,
    p_days INTEGER DEFAULT 30
) RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM notifications 
    WHERE tenant_id = p_tenant_id 
    AND created_at < NOW() - (p_days || ' days')::INTERVAL;

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get notification statistics for a tenant (with isolation)
CREATE OR REPLACE FUNCTION get_notification_stats(p_tenant_id UUID)
RETURNS TABLE (
    total BIGINT,
    unread BIGINT,
    messages_count BIGINT,
    deliveries_count BIGINT,
    statuses_count BIGINT,
    systems_count BIGINT,
    alerts_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::BIGINT as total,
        COUNT(*) FILTER (WHERE read_at IS NULL)::BIGINT as unread,
        COUNT(*) FILTER (WHERE type = 'message')::BIGINT as messages_count,
        COUNT(*) FILTER (WHERE type = 'delivery')::BIGINT as deliveries_count,
        COUNT(*) FILTER (WHERE type = 'status')::BIGINT as statuses_count,
        COUNT(*) FILTER (WHERE type = 'system')::BIGINT as systems_count,
        COUNT(*) FILTER (WHERE type = 'alert')::BIGINT as alerts_count
    FROM notifications 
    WHERE tenant_id = p_tenant_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Grant execute on functions to authenticated users
GRANT EXECUTE ON FUNCTION get_unread_notifications_count(UUID, UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION mark_notification_read(UUID, UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION mark_all_notifications_read(UUID, UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION create_notification(UUID, UUID, VARCHAR, VARCHAR, TEXT, JSONB) TO authenticated;
GRANT EXECUTE ON FUNCTION delete_old_notifications(UUID, INTEGER) TO authenticated;
GRANT EXECUTE ON FUNCTION get_notification_stats(UUID) TO authenticated;

-- Add RLS policies for tenant isolation
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see notifications for their tenant
CREATE POLICY notifications_tenant_isolation ON notifications
    FOR ALL
    USING (tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

-- Policy: Users can only create notifications for their tenant
CREATE POLICY notifications_create_tenant_isolation ON notifications
    FOR INSERT
    WITH CHECK (tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

-- Policy: Users can only update their own tenant's notifications
CREATE POLICY notifications_update_tenant_isolation ON notifications
    FOR UPDATE
    USING (tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

-- Policy: Users can only delete their own tenant's notifications
CREATE POLICY notifications_delete_tenant_isolation ON notifications
    FOR DELETE
    USING (tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

COMMENT ON TABLE notifications IS 'Stores notifications with complete tenant isolation';
COMMENT ON COLUMN notifications.tenant_id IS 'Tenant ID - notifications are strictly isolated per tenant';
COMMENT ON COLUMN notifications.user_id IS 'User ID - the owner of this notification';
COMMENT ON FUNCTION get_unread_notifications_count IS 'Returns unread count for a specific tenant (with optional user filter)';
COMMENT ON FUNCTION mark_notification_read IS 'Marks a notification as read with tenant isolation validation';
COMMENT ON FUNCTION create_notification IS 'Creates a new notification with tenant isolation enforced';