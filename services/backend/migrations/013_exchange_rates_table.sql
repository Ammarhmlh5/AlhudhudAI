-- Migration: 013_exchange_rates_table.sql
-- Description: إنشاء جدول أسعار الصرف مع سجل تاريخي
-- Requirements: 31.3, 31.4, 31.5, 31.6

-- +migrate Up
CREATE TABLE exchange_rates (
    id SERIAL PRIMARY KEY,
    from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(15, 6) NOT NULL,
    effective_from TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    effective_to TIMESTAMP,
    source VARCHAR(100), -- 'manual', 'api', 'central_bank'
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_rate_positive CHECK (rate > 0),
    CONSTRAINT check_different_currencies CHECK (from_currency != to_currency)
);

-- Indexes
CREATE INDEX idx_exchange_rates_from ON exchange_rates(from_currency);
CREATE INDEX idx_exchange_rates_to ON exchange_rates(to_currency);
CREATE INDEX idx_exchange_rates_effective ON exchange_rates(effective_from, effective_to);
CREATE INDEX idx_exchange_rates_pair ON exchange_rates(from_currency, to_currency);

-- أسعار الصرف الأولية (تقريبية - يجب تحديثها حسب السوق)
-- USD إلى YER: 1 دولار = 250 ريال يمني تقريباً
INSERT INTO exchange_rates (from_currency, to_currency, rate, effective_from, source, notes) VALUES
('USD', 'YER', 250.000000, CURRENT_TIMESTAMP, 'manual', 'سعر صرف أولي تقريبي'),
('YER', 'USD', 0.004000, CURRENT_TIMESTAMP, 'manual', 'سعر صرف أولي تقريبي'),
('USD', 'SAR', 3.750000, CURRENT_TIMESTAMP, 'manual', 'سعر صرف ثابت تقريباً'),
('SAR', 'USD', 0.266667, CURRENT_TIMESTAMP, 'manual', 'سعر صرف ثابت تقريباً'),
('YER', 'SAR', 0.015000, CURRENT_TIMESTAMP, 'manual', 'سعر صرف أولي تقريبي'),
('SAR', 'YER', 66.666667, CURRENT_TIMESTAMP, 'manual', 'سعر صرف أولي تقريبي');

-- +migrate Down
DROP TABLE IF EXISTS exchange_rates CASCADE;
