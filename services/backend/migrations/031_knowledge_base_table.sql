-- Migration: Knowledge Base System
-- Description: Creates tables for knowledge base articles and FAQs
-- Requirements: 37.10

-- Knowledge Base Articles Table
CREATE TABLE IF NOT EXISTS knowledge_base_articles (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT, -- NULL for global articles, specific tenant_id for tenant-specific
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    tags TEXT[],
    author_id BIGINT NOT NULL,
    published BOOLEAN NOT NULL DEFAULT false,
    view_count BIGINT NOT NULL DEFAULT 0,
    helpful_count BIGINT NOT NULL DEFAULT 0,
    not_helpful_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE
);

-- Article Feedback Table
CREATE TABLE IF NOT EXISTS kb_article_feedback (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL REFERENCES knowledge_base_articles(id) ON DELETE CASCADE,
    user_id BIGINT,
    is_helpful BOOLEAN NOT NULL,
    feedback_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Enable Row Level Security
ALTER TABLE knowledge_base_articles ENABLE ROW LEVEL SECURITY;
ALTER TABLE kb_article_feedback ENABLE ROW LEVEL SECURITY;

-- RLS Policies for knowledge_base_articles
-- Allow reading global articles (tenant_id IS NULL) or tenant-specific articles
CREATE POLICY kb_articles_read_policy ON knowledge_base_articles
    FOR SELECT
    USING (
        published = true AND (
            tenant_id IS NULL OR 
            tenant_id = current_setting('app.current_tenant')::bigint
        )
    );

-- Only admins can insert/update/delete articles
CREATE POLICY kb_articles_admin_policy ON knowledge_base_articles
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant')::bigint);

-- RLS Policies for kb_article_feedback
CREATE POLICY kb_feedback_tenant_isolation ON kb_article_feedback
    USING (article_id IN (
        SELECT id FROM knowledge_base_articles 
        WHERE tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant')::bigint
    ));

-- Indexes for knowledge_base_articles
CREATE INDEX idx_kb_articles_tenant ON knowledge_base_articles(tenant_id);
CREATE INDEX idx_kb_articles_category ON knowledge_base_articles(category);
CREATE INDEX idx_kb_articles_published ON knowledge_base_articles(published);
CREATE INDEX idx_kb_articles_slug ON knowledge_base_articles(slug);
CREATE INDEX idx_kb_articles_created ON knowledge_base_articles(created_at DESC);
CREATE INDEX idx_kb_articles_tags ON knowledge_base_articles USING gin(tags);

-- Full text search index
CREATE INDEX idx_kb_articles_search ON knowledge_base_articles 
    USING gin(to_tsvector('english', title || ' ' || content));

-- Indexes for kb_article_feedback
CREATE INDEX idx_kb_feedback_article ON kb_article_feedback(article_id);
CREATE INDEX idx_kb_feedback_user ON kb_article_feedback(user_id);

-- Function to update article view count
CREATE OR REPLACE FUNCTION increment_article_views()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE knowledge_base_articles
    SET view_count = view_count + 1
    WHERE id = NEW.article_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update helpful counts
CREATE OR REPLACE FUNCTION update_article_helpful_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_helpful THEN
        UPDATE knowledge_base_articles
        SET helpful_count = helpful_count + 1
        WHERE id = NEW.article_id;
    ELSE
        UPDATE knowledge_base_articles
        SET not_helpful_count = not_helpful_count + 1
        WHERE id = NEW.article_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update helpful counts
CREATE TRIGGER update_helpful_counts
    AFTER INSERT ON kb_article_feedback
    FOR EACH ROW
    EXECUTE FUNCTION update_article_helpful_counts();

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_kb_article_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update timestamp on article update
CREATE TRIGGER update_kb_articles_timestamp
    BEFORE UPDATE ON knowledge_base_articles
    FOR EACH ROW
    EXECUTE FUNCTION update_kb_article_timestamp();

-- Insert some default categories
INSERT INTO knowledge_base_articles (title, slug, content, category, author_id, published, published_at) VALUES
('Getting Started with SMS Gateway', 'getting-started', 'Learn how to get started with our SMS Gateway service...', 'Getting Started', 1, true, NOW()),
('How to Send Your First Message', 'send-first-message', 'Step-by-step guide to sending your first SMS message...', 'Getting Started', 1, true, NOW()),
('Understanding Pricing and Packages', 'pricing-packages', 'Learn about our pricing structure and available packages...', 'Billing', 1, true, NOW()),
('API Integration Guide', 'api-integration', 'Complete guide to integrating with our API...', 'Technical', 1, true, NOW()),
('Troubleshooting Common Issues', 'troubleshooting', 'Solutions to common problems you might encounter...', 'Technical', 1, true, NOW())
ON CONFLICT (slug) DO NOTHING;

-- Comments
COMMENT ON TABLE knowledge_base_articles IS 'Knowledge base articles and FAQs';
COMMENT ON TABLE kb_article_feedback IS 'User feedback on knowledge base articles';
COMMENT ON COLUMN knowledge_base_articles.tenant_id IS 'NULL for global articles, specific tenant_id for tenant-specific articles';
COMMENT ON COLUMN knowledge_base_articles.published IS 'Whether the article is published and visible to users';
COMMENT ON COLUMN knowledge_base_articles.view_count IS 'Number of times the article has been viewed';
COMMENT ON COLUMN knowledge_base_articles.helpful_count IS 'Number of users who found the article helpful';
