-- Migration: Support Tickets System
-- Description: Creates tables for support ticket management system
-- Requirements: 37.1, 37.2, 37.3, 37.4

-- Support Tickets Table
CREATE TABLE IF NOT EXISTS support_tickets (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    ticket_number VARCHAR(20) UNIQUE NOT NULL,
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium', -- low, medium, high, urgent
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- open, in_progress, waiting_customer, resolved, closed
    category VARCHAR(50), -- billing, technical, feature_request, bug_report
    assigned_to BIGINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    closed_at TIMESTAMP WITH TIME ZONE,
    first_response_at TIMESTAMP WITH TIME ZONE,
    sla_due_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT check_priority CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    CONSTRAINT check_status CHECK (status IN ('open', 'in_progress', 'waiting_customer', 'resolved', 'closed'))
);

-- Ticket Comments Table
CREATE TABLE IF NOT EXISTS ticket_comments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    is_staff BOOLEAN NOT NULL DEFAULT false,
    comment TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Ticket Attachments Table
CREATE TABLE IF NOT EXISTS ticket_attachments (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    uploaded_by BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE support_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_attachments ENABLE ROW LEVEL SECURITY;

-- RLS Policies for support_tickets
CREATE POLICY support_tickets_tenant_isolation ON support_tickets
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

CREATE POLICY support_tickets_tenant_insert ON support_tickets
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::bigint);

CREATE POLICY support_tickets_tenant_update ON support_tickets
    FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

CREATE POLICY support_tickets_tenant_delete ON support_tickets
    FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

-- RLS Policies for ticket_comments
CREATE POLICY ticket_comments_tenant_isolation ON ticket_comments
    USING (ticket_id IN (SELECT id FROM support_tickets WHERE tenant_id = current_setting('app.current_tenant')::bigint));

CREATE POLICY ticket_comments_tenant_insert ON ticket_comments
    FOR INSERT
    WITH CHECK (ticket_id IN (SELECT id FROM support_tickets WHERE tenant_id = current_setting('app.current_tenant')::bigint));

-- RLS Policies for ticket_attachments
CREATE POLICY ticket_attachments_tenant_isolation ON ticket_attachments
    USING (ticket_id IN (SELECT id FROM support_tickets WHERE tenant_id = current_setting('app.current_tenant')::bigint));

CREATE POLICY ticket_attachments_tenant_insert ON ticket_attachments
    FOR INSERT
    WITH CHECK (ticket_id IN (SELECT id FROM support_tickets WHERE tenant_id = current_setting('app.current_tenant')::bigint));

-- Indexes for support_tickets
CREATE INDEX idx_support_tickets_tenant ON support_tickets(tenant_id);
CREATE INDEX idx_support_tickets_user ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_priority ON support_tickets(priority);
CREATE INDEX idx_support_tickets_assigned ON support_tickets(assigned_to);
CREATE INDEX idx_support_tickets_created ON support_tickets(created_at DESC);
CREATE INDEX idx_support_tickets_sla ON support_tickets(sla_due_at) WHERE status NOT IN ('resolved', 'closed');

-- Indexes for ticket_comments
CREATE INDEX idx_ticket_comments_ticket ON ticket_comments(ticket_id);
CREATE INDEX idx_ticket_comments_created ON ticket_comments(created_at DESC);

-- Indexes for ticket_attachments
CREATE INDEX idx_ticket_attachments_ticket ON ticket_attachments(ticket_id);

-- Function to generate ticket number
CREATE OR REPLACE FUNCTION generate_ticket_number()
RETURNS TRIGGER AS $$
BEGIN
    NEW.ticket_number := 'TKT-' || TO_CHAR(NOW(), 'YYYYMMDD') || '-' || LPAD(NEW.id::TEXT, 6, '0');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to set ticket number after insert
CREATE TRIGGER set_ticket_number
    AFTER INSERT ON support_tickets
    FOR EACH ROW
    EXECUTE FUNCTION generate_ticket_number();

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_ticket_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update timestamp on ticket update
CREATE TRIGGER update_support_tickets_timestamp
    BEFORE UPDATE ON support_tickets
    FOR EACH ROW
    EXECUTE FUNCTION update_ticket_timestamp();

-- Comments
COMMENT ON TABLE support_tickets IS 'Support tickets for customer service';
COMMENT ON TABLE ticket_comments IS 'Comments on support tickets';
COMMENT ON TABLE ticket_attachments IS 'File attachments for support tickets';
COMMENT ON COLUMN support_tickets.priority IS 'Ticket priority: low, medium, high, urgent';
COMMENT ON COLUMN support_tickets.status IS 'Ticket status: open, in_progress, waiting_customer, resolved, closed';
COMMENT ON COLUMN support_tickets.category IS 'Ticket category: billing, technical, feature_request, bug_report';
COMMENT ON COLUMN support_tickets.sla_due_at IS 'SLA deadline for ticket resolution';
