-- Migration: 011_segmentation_rules_table.sql
-- Description: Creates the segmentation_rules table for managing SMS message segmentation rules
-- Requirements: 7.1, 7.2, 7.3, 7.4

-- +goose Up
-- +goose StatementBegin

-- Create segmentation_rules table
CREATE TABLE segmentation_rules (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER REFERENCES dedicated_gateways(id), -- NULL = default rules
    country_code VARCHAR(2),
    network_operator VARCHAR(100),
    
    -- Encoding type
    encoding_type VARCHAR(20) NOT NULL, -- 'gsm7', 'unicode'
    
    -- Segmentation rules
    segment_1_chars INTEGER NOT NULL, -- 70 for first segment
    segment_2_chars INTEGER NOT NULL, -- 65 for second segment
    segment_3_chars INTEGER NOT NULL, -- 62 for third segment
    segment_n_chars INTEGER NOT NULL, -- for subsequent segments
    
    -- Additional information
    header_overhead INTEGER DEFAULT 0, -- characters used for header
    description TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_segment_1_positive CHECK (segment_1_chars > 0),
    CONSTRAINT check_segment_2_positive CHECK (segment_2_chars > 0),
    CONSTRAINT check_segment_3_positive CHECK (segment_3_chars > 0),
    CONSTRAINT check_segment_n_positive CHECK (segment_n_chars > 0),
    CONSTRAINT check_header_overhead_non_negative CHECK (header_overhead >= 0),
    CONSTRAINT check_encoding_type CHECK (encoding_type IN ('gsm7', 'unicode'))
);

-- Create indexes
CREATE INDEX idx_segmentation_gateway ON segmentation_rules(gateway_id);
CREATE INDEX idx_segmentation_country ON segmentation_rules(country_code);
CREATE INDEX idx_segmentation_encoding ON segmentation_rules(encoding_type);
CREATE INDEX idx_segmentation_active ON segmentation_rules(is_active);

-- Insert default segmentation rules
-- GSM-7 encoding: 70, 65, 62 characters
INSERT INTO segmentation_rules (
    gateway_id, 
    country_code, 
    network_operator, 
    encoding_type, 
    segment_1_chars, 
    segment_2_chars, 
    segment_3_chars, 
    segment_n_chars,
    description
) VALUES
(NULL, NULL, NULL, 'gsm7', 70, 65, 62, 62, 'Default GSM-7 segmentation rules'),
(NULL, NULL, NULL, 'unicode', 70, 65, 62, 62, 'Default Unicode segmentation rules');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop indexes
DROP INDEX IF EXISTS idx_segmentation_active;
DROP INDEX IF EXISTS idx_segmentation_encoding;
DROP INDEX IF EXISTS idx_segmentation_country;
DROP INDEX IF EXISTS idx_segmentation_gateway;

-- Drop table
DROP TABLE IF EXISTS segmentation_rules;

-- +goose StatementEnd
