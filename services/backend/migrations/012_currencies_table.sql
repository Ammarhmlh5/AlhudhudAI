-- Migration: 012_currencies_table.sql
-- Description: إنشاء جدول العملات لدعم عملات متعددة
-- Requirements: 31.1, 31.2, 31.9

-- +migrate Up
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    code VARCHAR(3) UNIQUE NOT NULL, -- 'USD', 'YER', 'SAR'
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    decimal_places INTEGER DEFAULT 2,
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_currencies_code ON currencies(code);
CREATE INDEX idx_currencies_active ON currencies(is_active);

-- البيانات الأولية - العملات المدعومة
INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
('USD', 'US Dollar', '$', 2),
('YER', 'Yemeni Rial', 'ر.ي', 2),
('SAR', 'Saudi Riyal', 'ر.س', 2);

-- +migrate Down
DROP TABLE IF EXISTS currencies CASCADE;
