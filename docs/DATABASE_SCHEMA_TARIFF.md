# Database Schema - Tariff & Pricing System

## Overview

This document details the database schema for the Tariff & Pricing System, which extends the main SMS Gateway database.

**Database**: PostgreSQL 15+  
**Encoding**: UTF-8  
**Timezone**: UTC  
**Multi-Tenancy**: Row Level Security (RLS) enabled

## Entity Relationship Diagram

```
┌──────────────┐       ┌──────────────┐       ┌──────────────┐
│   services   │───────│   tariffs    │       │   packages   │
└──────────────┘   1:N └──────────────┘       └──────────────┘
                                                      │
                                                      │ 1:N
                                                      │
┌──────────────┐       ┌──────────────────┐   ┌──────────────────┐
│   tenants    │───────│  subscriptions   │───│ rollover_messages│
└──────────────┘   1:N └──────────────────┘   └──────────────────┘
      │
      │ 1:N
      │
┌──────────────────────┐       ┌──────────────────────┐
│ dedicated_gateways   │───────│   resource_quotas    │
└──────────────────────┘   1:1 └──────────────────────┘
      │
      │ 1:N
      │
┌──────────────────────┐       ┌──────────────────────┐
│   gateway_users      │       │   gateway_api_keys   │
└──────────────────────┘       └──────────────────────┘
```

## Core Tables

### 1. services

Service types (Hudud Gateway, Mobile Gateway, Dedicated Gateway).

```sql
CREATE TABLE services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_services_type ON services(type);
CREATE INDEX idx_services_active ON services(is_active);
```

**Initial Data:**
```sql
INSERT INTO services (name, code, type, description) VALUES
('Hudud Gateway', 'HUDUD_GATEWAY', 'gateway', 'Pay-as-you-go SMS service'),
('Mobile Gateway', 'MOBILE_GATEWAY', 'mobile', 'Monthly subscription service'),
('Dedicated Gateway', 'DEDICATED_GATEWAY', 'dedicated', 'Enterprise gateway service');
```


### 2. tariffs

Pricing for different networks and countries.

```sql
CREATE TABLE tariffs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    service_id INTEGER NOT NULL REFERENCES services(id),
    country_code VARCHAR(2) NOT NULL,
    network_operator VARCHAR(100),
    message_type VARCHAR(50) DEFAULT 'standard',
    cost_per_message DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'YER',
    effective_from DATE NOT NULL,
    effective_to DATE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_cost_positive CHECK (cost_per_message > 0)
);

-- Composite indexes starting with tenant_id
CREATE INDEX idx_tariffs_tenant_service ON tariffs(tenant_id, service_id);
CREATE INDEX idx_tariffs_tenant_country ON tariffs(tenant_id, country_code);
CREATE INDEX idx_tariffs_tenant_operator ON tariffs(tenant_id, network_operator);
CREATE INDEX idx_tariffs_tenant_active ON tariffs(tenant_id, is_active);
CREATE INDEX idx_tariffs_tenant_effective ON tariffs(tenant_id, effective_from, effective_to);

-- Row Level Security
ALTER TABLE tariffs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON tariffs
    USING (tenant_id = current_setting('app.current_tenant')::uuid);

CREATE POLICY tenant_isolation_insert_policy ON tariffs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `tenant_id`: Tenant identifier (for multi-tenancy isolation)
- `service_id`: Service type
- `country_code`: ISO 3166-1 alpha-2 country code
- `network_operator`: Network operator name
- `message_type`: 'standard' or 'unicode'
- `cost_per_message`: Cost per message segment
- `currency`: Currency code (YER, USD, SAR)
- `effective_from`: Start date for this tariff
- `effective_to`: End date (null = no end date)
- `is_active`: Whether this tariff is currently active

### 3. packages

Monthly subscription packages.

```sql
CREATE TABLE packages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    service_id INTEGER NOT NULL REFERENCES services(id),
    price DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    included_messages INTEGER,
    billing_cycle VARCHAR(50) DEFAULT 'monthly',
    rollover_enabled BOOLEAN DEFAULT true,
    rollover_limit INTEGER,
    rollover_expiry_months INTEGER DEFAULT 1,
    overage_rate DECIMAL(10, 4),
    features JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_price_positive CHECK (price > 0),
    CONSTRAINT check_messages_positive CHECK (included_messages IS NULL OR included_messages > 0)
);

CREATE INDEX idx_packages_tenant_service ON packages(tenant_id, service_id);
CREATE INDEX idx_packages_tenant_active ON packages(tenant_id, is_active);

ALTER TABLE packages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON packages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Columns:**
- `included_messages`: Number of messages included (NULL = unlimited)
- `billing_cycle`: 'monthly' or 'annual'
- `rollover_enabled`: Whether unused messages roll over
- `rollover_limit`: Maximum messages that can roll over
- `rollover_expiry_months`: How long rolled messages are valid
- `overage_rate`: Cost per message beyond included amount
- `features`: JSON array of package features

### 4. subscriptions

Customer subscriptions to packages.

```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    package_id INTEGER NOT NULL REFERENCES packages(id),
    status VARCHAR(50) DEFAULT 'active',
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    next_renewal_date DATE NOT NULL,
    messages_included INTEGER,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    overage_messages INTEGER DEFAULT 0,
    overage_cost DECIMAL(10, 2) DEFAULT 0.00,
    auto_renewal BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_messages_valid CHECK (
        messages_used >= 0 AND 
        messages_remaining >= 0 AND
        overage_messages >= 0
    )
);

CREATE INDEX idx_subscriptions_tenant_status ON subscriptions(tenant_id, status);
CREATE INDEX idx_subscriptions_tenant_renewal ON subscriptions(tenant_id, next_renewal_date);

ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON subscriptions
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Status Values:**
- `active`: Subscription is active
- `grace_period`: Payment failed, in grace period
- `suspended`: Suspended due to non-payment
- `cancelled`: Cancelled by user

### 5. rollover_messages

Rolled over messages from previous billing cycles.

```sql
CREATE TABLE rollover_messages (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id INTEGER NOT NULL REFERENCES subscriptions(id),
    messages_count INTEGER NOT NULL,
    source_month DATE NOT NULL,
    expiry_date DATE NOT NULL,
    messages_used INTEGER DEFAULT 0,
    messages_remaining INTEGER,
    status VARCHAR(50) DEFAULT 'pending',
    payment_status VARCHAR(50) DEFAULT 'pending',
    payment_verified_at TIMESTAMP,
    forfeited_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_messages_valid CHECK (
        messages_count > 0 AND
        messages_used >= 0 AND
        messages_remaining >= 0 AND
        messages_used + messages_remaining <= messages_count
    )
);

CREATE INDEX idx_rollover_tenant_subscription ON rollover_messages(tenant_id, subscription_id);
CREATE INDEX idx_rollover_tenant_expiry ON rollover_messages(tenant_id, expiry_date);
CREATE INDEX idx_rollover_status ON rollover_messages(status);

ALTER TABLE rollover_messages ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON rollover_messages
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Status Values:**
- `pending`: Waiting for payment verification
- `active`: Available for use
- `expired`: Past expiry date
- `forfeited`: Forfeited due to payment failure

**Payment Status:**
- `pending`: Waiting for auto-renewal payment
- `success`: Payment succeeded, messages activated
- `failed`: Payment failed, messages forfeited


## Dedicated Gateway Tables

### 6. dedicated_gateways

Enterprise dedicated gateways.

```sql
CREATE TABLE dedicated_gateways (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    telecom_provider VARCHAR(255) NOT NULL,
    telecom_username VARCHAR(255) NOT NULL,
    telecom_password_encrypted TEXT NOT NULL,
    telecom_api_key_encrypted TEXT,
    telecom_api_url VARCHAR(500),
    subscription_type VARCHAR(50) DEFAULT 'monthly',
    subscription_fee DECIMAL(10, 2) NOT NULL,
    server_rental_fee DECIMAL(10, 2) NOT NULL,
    setup_fee DECIMAL(10, 2) DEFAULT 0.00,
    contract_start_date DATE NOT NULL,
    contract_end_date DATE,
    telecom_balance DECIMAL(10, 2),
    balance_alert_threshold DECIMAL(10, 2),
    last_balance_check TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active',
    last_health_check TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dedicated_gateways_tenant ON dedicated_gateways(tenant_id);
CREATE INDEX idx_dedicated_gateways_status ON dedicated_gateways(status);
CREATE INDEX idx_dedicated_gateways_code ON dedicated_gateways(code);

ALTER TABLE dedicated_gateways ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON dedicated_gateways
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

**Important Notes:**
- `telecom_balance`: Balance with telecom provider (read-only, for monitoring)
- `subscription_fee`: Monthly/annual subscription fee charged by platform
- `server_rental_fee`: Server rental fee charged by platform
- Enterprise pays telecom provider directly for message costs
- Platform only charges subscription + rental fees

### 7. resource_quotas

Resource limits for dedicated gateways.

```sql
CREATE TABLE resource_quotas (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    cpu_cores DECIMAL(4, 2) NOT NULL,
    memory_gb INTEGER NOT NULL,
    bandwidth_mbps INTEGER NOT NULL,
    storage_gb INTEGER NOT NULL,
    max_db_connections INTEGER NOT NULL,
    redis_memory_mb INTEGER NOT NULL,
    max_messages_per_hour INTEGER NOT NULL,
    max_messages_per_day INTEGER NOT NULL,
    api_rate_limit INTEGER NOT NULL,
    max_worker_threads INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_quotas_positive CHECK (
        cpu_cores > 0 AND
        memory_gb > 0 AND
        bandwidth_mbps > 0 AND
        storage_gb > 0
    )
);

CREATE UNIQUE INDEX idx_resource_quotas_gateway ON resource_quotas(gateway_id);

ALTER TABLE resource_quotas ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON resource_quotas
    USING (gateway_id IN (
        SELECT id FROM dedicated_gateways 
        WHERE tenant_id = current_setting('app.current_tenant')::uuid
    ));
```

### 8. resource_usage

Actual resource usage tracking.

```sql
CREATE TABLE resource_usage (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cpu_usage DECIMAL(5, 2) NOT NULL,
    memory_usage_gb DECIMAL(10, 2) NOT NULL,
    bandwidth_usage_mbps DECIMAL(10, 2) NOT NULL,
    db_connections_active INTEGER NOT NULL,
    db_queries_per_sec DECIMAL(10, 2) NOT NULL,
    redis_memory_mb DECIMAL(10, 2) NOT NULL,
    redis_keys INTEGER NOT NULL,
    redis_hit_rate DECIMAL(5, 2),
    messages_last_hour INTEGER NOT NULL,
    messages_today INTEGER NOT NULL,
    queue_depth INTEGER NOT NULL,
    api_requests_per_min INTEGER NOT NULL,
    worker_threads_active INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Partitioning by month for performance
CREATE INDEX idx_resource_usage_gateway_time ON resource_usage(gateway_id, timestamp DESC);

ALTER TABLE resource_usage ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON resource_usage
    USING (gateway_id IN (
        SELECT id FROM dedicated_gateways 
        WHERE tenant_id = current_setting('app.current_tenant')::uuid
    ));
```

### 9. gateway_users

Users with access to dedicated gateways.

```sql
CREATE TABLE gateway_users (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    permissions JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uq_gateway_user UNIQUE (gateway_id, user_id)
);

CREATE INDEX idx_gateway_users_gateway ON gateway_users(gateway_id);
CREATE INDEX idx_gateway_users_user ON gateway_users(user_id);

ALTER TABLE gateway_users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON gateway_users
    USING (gateway_id IN (
        SELECT id FROM dedicated_gateways 
        WHERE tenant_id = current_setting('app.current_tenant')::uuid
    ));
```

**Roles:**
- `admin`: Full access
- `operator`: Send messages, view reports
- `viewer`: View reports only

### 10. gateway_api_keys

API keys for dedicated gateways.

```sql
CREATE TABLE gateway_api_keys (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_hash VARCHAR(255) NOT NULL,
    permissions JSONB,
    rate_limit INTEGER NOT NULL,
    allowed_ips TEXT[],
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_gateway_api_keys_gateway ON gateway_api_keys(gateway_id);
CREATE INDEX idx_gateway_api_keys_hash ON gateway_api_keys(key_hash);

ALTER TABLE gateway_api_keys ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON gateway_api_keys
    USING (gateway_id IN (
        SELECT id FROM dedicated_gateways 
        WHERE tenant_id = current_setting('app.current_tenant')::uuid
    ));
```

### 11. gateway_statistics

Daily statistics for dedicated gateways.

```sql
CREATE TABLE gateway_statistics (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    messages_sent INTEGER DEFAULT 0,
    messages_delivered INTEGER DEFAULT 0,
    messages_failed INTEGER DEFAULT 0,
    messages_pending INTEGER DEFAULT 0,
    avg_response_time_ms DECIMAL(10, 2),
    max_response_time_ms INTEGER,
    uptime_percentage DECIMAL(5, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uq_gateway_stats_date UNIQUE (gateway_id, date)
);

CREATE INDEX idx_gateway_statistics_gateway_date ON gateway_statistics(gateway_id, date DESC);

ALTER TABLE gateway_statistics ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON gateway_statistics
    USING (gateway_id IN (
        SELECT id FROM dedicated_gateways 
        WHERE tenant_id = current_setting('app.current_tenant')::uuid
    ));
```

## Supporting Tables

### 12. segmentation_rules

Message segmentation rules.

```sql
CREATE TABLE segmentation_rules (
    id SERIAL PRIMARY KEY,
    gateway_type VARCHAR(50) NOT NULL,
    encoding VARCHAR(50) NOT NULL,
    first_segment_length INTEGER NOT NULL,
    subsequent_segment_length INTEGER NOT NULL,
    third_segment_length INTEGER,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_segmentation_rules_gateway ON segmentation_rules(gateway_type, encoding);
```

**Default Rules:**
```sql
INSERT INTO segmentation_rules (gateway_type, encoding, first_segment_length, subsequent_segment_length, third_segment_length) VALUES
('hudud', 'GSM-7', 70, 65, 62),
('hudud', 'Unicode', 70, 65, 62);
```

### 13. currencies

Supported currencies.

```sql
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    code VARCHAR(3) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO currencies (code, name, symbol) VALUES
('USD', 'US Dollar', '$'),
('YER', 'Yemeni Rial', 'ر.ي'),
('SAR', 'Saudi Riyal', 'ر.س');
```

### 14. exchange_rates

Currency exchange rates.

```sql
CREATE TABLE exchange_rates (
    id SERIAL PRIMARY KEY,
    from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
    rate DECIMAL(15, 6) NOT NULL,
    effective_from TIMESTAMP NOT NULL,
    effective_to TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_rate_positive CHECK (rate > 0),
    CONSTRAINT uq_exchange_rate UNIQUE (from_currency, to_currency, effective_from)
);

CREATE INDEX idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency);
CREATE INDEX idx_exchange_rates_active ON exchange_rates(is_active, effective_from DESC);
```

### 15. billing_invoices

Billing invoices.

```sql
CREATE TABLE billing_invoices (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    invoice_number VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    tax DECIMAL(10, 2) DEFAULT 0.00,
    total DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50) DEFAULT 'pending',
    issue_date DATE NOT NULL,
    due_date DATE NOT NULL,
    paid_at TIMESTAMP,
    description TEXT,
    items JSONB,
    pdf_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invoices_tenant ON billing_invoices(tenant_id, issue_date DESC);
CREATE INDEX idx_invoices_status ON billing_invoices(status);
CREATE INDEX idx_invoices_number ON billing_invoices(invoice_number);

ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON billing_invoices
    USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

## Row Level Security (RLS) Summary

All tenant-specific tables have RLS enabled with policies that ensure:
1. Users can only see data for their tenant
2. Users can only insert data with their tenant_id
3. Users can only update/delete their own tenant's data

**Setting tenant context:**
```sql
SET app.current_tenant = 'tenant-uuid-here';
```

This is done automatically by the application middleware for every request.

## Indexes Strategy

All indexes on tenant-specific tables follow this pattern:
1. **Start with tenant_id** for optimal query performance
2. **Add frequently filtered columns** next
3. **Add sort columns** last

Example:
```sql
CREATE INDEX idx_subscriptions_tenant_status_renewal 
ON subscriptions(tenant_id, status, next_renewal_date);
```

## Data Retention

| Table | Retention Period | Action |
|-------|------------------|--------|
| tariffs | Indefinite | Keep historical records |
| packages | Indefinite | Keep historical records |
| subscriptions | 2 years | Archive to cold storage |
| rollover_messages | 1 year | Archive to cold storage |
| resource_usage | 90 days | Archive to cold storage |
| gateway_statistics | 2 years | Archive to cold storage |
| billing_invoices | 7 years | Archive to cold storage |

## Migration Order

Execute migrations in this order:

1. `008_tariff_services_table.sql`
2. `009_insert_default_services.sql`
3. `010_tariffs_table.sql`
4. `011_segmentation_rules_table.sql`
5. `012_currencies_table.sql`
6. `013_exchange_rates_table.sql`
7. `014_balance_history_rls.sql`
8. `016_packages_table.sql`
9. `017_subscriptions_table.sql`
10. `018_rollover_messages_table.sql`
11. `019_billing_invoices_table.sql`
12. `020_dedicated_gateways_table.sql`
13. `021_resource_quotas_table.sql`
14. `022_resource_usage_table.sql`
15. `023_gateway_users_table.sql`
16. `024_gateway_api_keys_table.sql`
17. `025_gateway_statistics_table.sql`

## Performance Considerations

### Connection Pooling
```go
// Recommended settings
MaxOpenConns: 100
MaxIdleConns: 10
ConnMaxLifetime: 1 hour
```

### Query Optimization
- Use prepared statements
- Leverage composite indexes
- Use EXPLAIN ANALYZE for slow queries
- Consider read replicas for reporting

### Caching Strategy
- Cache tariffs (TTL: 1 hour)
- Cache packages (TTL: 1 hour)
- Cache exchange rates (TTL: 1 day)
- Invalidate on updates

---

**Created**: 2026-03-02  
**Version**: 1.0  
**Last Updated**: 2026-03-02
