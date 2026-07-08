-- Test script to validate segmentation_rules migration
-- This is a dry-run to check SQL syntax

BEGIN;

-- Create segmentation_rules table
CREATE TABLE IF NOT EXISTS segmentation_rules_test (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER, -- NULL = default rules
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

-- Test insert
INSERT INTO segmentation_rules_test (
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

-- Verify data
SELECT * FROM segmentation_rules_test;

-- Cleanup
DROP TABLE segmentation_rules_test;

ROLLBACK;
