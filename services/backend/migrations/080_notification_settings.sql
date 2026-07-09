-- Migration 080: Notification Settings System
-- Provides configurable notification settings for tenant isolation and control

-- Notification Settings Table
CREATE TABLE IF NOT EXISTS notification_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}', -- Additional config like days_before_expiry, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_notification_settings_tenant 
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    
    CONSTRAINT uq_notification_settings_type 
        UNIQUE (tenant_id, notification_type)
);

-- Notification Templates Table
CREATE TABLE IF NOT EXISTS notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- NULL means system-wide template
    notification_type VARCHAR(50) NOT NULL,
    title_template VARCHAR(255) NOT NULL,
    body_template TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_notification_templates_tenant 
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Notification Logs Table (for tracking sent notifications)
CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    notification_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'sent', -- sent, failed, pending
    error_message TEXT,

    CONSTRAINT fk_notification_logs_tenant 
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_notification_settings_tenant 
    ON notification_settings(tenant_id);

CREATE INDEX IF NOT EXISTS idx_notification_templates_tenant_type 
    ON notification_templates(tenant_id, notification_type);

CREATE INDEX IF NOT EXISTS idx_notification_logs_tenant_created 
    ON notification_logs(tenant_id, sent_at DESC);

-- Default notification settings for new tenants
CREATE OR REPLACE FUNCTION set_default_notification_settings(p_tenant_id UUID)
RETURNS VOID AS $$
DECLARE
    v_notification_type VARCHAR;
BEGIN
    -- Loop through all notification types and create default settings
    FOR v_notification_type IN 
        SELECT unnest(ARRAY[
            'reconnection_alert',      -- إشعار عند إعادة الاتصال بعد انقطاع
            'package_expiry_warning',  -- تحذير قرب انتهاء الباقة
            'package_expiry_alert',    -- إشعار انتهاء الباقة
            'low_balance_warning',     -- تحذير انخفاض الرصيد
            'message_pending_alert',   -- رسائل معلقة
            'delivery_failed_alert',   -- فشل التسليم
            'system_update',           -- تحديثات النظام
            'new_message'              -- رسالة جديدة (معطلة افتراضياً)
        ])
    LOOP
        INSERT INTO notification_settings (tenant_id, notification_type, is_enabled, config)
        VALUES (
            p_tenant_id, 
            v_notification_type,
            CASE 
                WHEN v_notification_type = 'new_message' THEN false 
                ELSE true 
            END,
            '{}'::JSONB
        )
        ON CONFLICT (tenant_id, notification_type) DO NOTHING;
    END LOOP;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to check if notification is enabled for tenant
CREATE OR REPLACE FUNCTION is_notification_enabled(
    p_tenant_id UUID,
    p_notification_type VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    v_is_enabled BOOLEAN;
BEGIN
    SELECT is_enabled INTO v_is_enabled
    FROM notification_settings
    WHERE tenant_id = p_tenant_id AND notification_type = p_notification_type;

    -- If no settings found, return true (enabled by default)
    RETURN COALESCE(v_is_enabled, true);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get notification config for tenant
CREATE OR REPLACE FUNCTION get_notification_config(
    p_tenant_id UUID,
    p_notification_type VARCHAR
) RETURNS JSONB AS $$
DECLARE
    v_config JSONB;
BEGIN
    SELECT config INTO v_config
    FROM notification_settings
    WHERE tenant_id = p_tenant_id AND notification_type = p_notification_type;

    RETURN COALESCE(v_config, '{}'::JSONB);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to log sent notification
CREATE OR REPLACE FUNCTION log_notification(
    p_tenant_id UUID,
    p_user_id UUID,
    p_notification_type VARCHAR,
    p_title VARCHAR,
    p_body TEXT,
    p_status VARCHAR DEFAULT 'sent',
    p_error_message TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_log_id UUID;
BEGIN
    INSERT INTO notification_logs (tenant_id, user_id, notification_type, title, body, status, error_message)
    VALUES (p_tenant_id, p_user_id, p_notification_type, p_title, p_body, p_status, p_error_message)
    RETURNING id INTO v_log_id;

    RETURN v_log_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get notification statistics
CREATE OR REPLACE FUNCTION get_notification_stats(p_tenant_id UUID)
RETURNS TABLE (
    notification_type VARCHAR,
    sent_count BIGINT,
    failed_count BIGINT,
    last_sent_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        nl.notification_type,
        COUNT(*) FILTER (WHERE nl.status = 'sent')::BIGINT as sent_count,
        COUNT(*) FILTER (WHERE nl.status = 'failed')::BIGINT as failed_count,
        MAX(nl.sent_at) as last_sent_at
    FROM notification_logs nl
    WHERE nl.tenant_id = p_tenant_id
    GROUP BY nl.notification_type
    ORDER BY last_sent_at DESC;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- RLS Policies
ALTER TABLE notification_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_logs ENABLE ROW LEVEL SECURITY;

-- Settings: Tenant admins can manage their own settings
CREATE POLICY notification_settings_tenant_manage ON notification_settings
    FOR ALL
    USING (tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

-- Templates: Tenant admins can manage their own templates
CREATE POLICY notification_templates_tenant_manage ON notification_templates
    FOR ALL
    USING (tenant_id IS NULL OR tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

-- Logs: Tenant admins can view their notification logs
CREATE POLICY notification_logs_tenant_view ON notification_logs
    FOR SELECT
    USING (tenant_id IN (
        SELECT tenant_id FROM user_tenants WHERE user_id = auth.uid()
    ));

-- Grant permissions
GRANT EXECUTE ON FUNCTION set_default_notification_settings(UUID) TO authenticated;
GRANT EXECUTE ON FUNCTION is_notification_enabled(UUID, VARCHAR) TO authenticated;
GRANT EXECUTE ON FUNCTION get_notification_config(UUID, VARCHAR) TO authenticated;
GRANT EXECUTE ON FUNCTION log_notification(UUID, UUID, VARCHAR, VARCHAR, TEXT, VARCHAR, TEXT) TO authenticated;
GRANT EXECUTE ON FUNCTION get_notification_stats(UUID) TO authenticated;

COMMENT ON TABLE notification_settings IS 'Stores notification preferences per tenant';
COMMENT ON TABLE notification_templates IS 'Stores notification message templates';
COMMENT ON TABLE notification_logs IS 'Logs all sent notifications for tracking';
COMMENT ON FUNCTION is_notification_enabled IS 'Checks if a notification type is enabled for a tenant';
COMMENT ON FUNCTION get_notification_config IS 'Returns configuration for a notification type';