-- Migration: Recreate resource_quotas with UUID
-- Purpose: Match Go struct and standardize on UUID PKs/FKs

DROP TABLE IF EXISTS resource_quotas CASCADE;

CREATE TABLE resource_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway_id UUID NOT NULL UNIQUE REFERENCES dedicated_gateways(id) ON DELETE CASCADE,
    cpu_cores DECIMAL(4,2),
    memory_mb INTEGER,
    bandwidth_mbps INTEGER,
    max_concurrent_connections INTEGER,
    storage_gb INTEGER,
    max_db_connections INTEGER,
    max_queries_per_second INTEGER,
    redis_memory_mb INTEGER,
    redis_max_keys INTEGER,
    max_messages_per_hour INTEGER,
    max_messages_per_day INTEGER,
    max_queue_depth INTEGER,
    api_rate_limit_per_minute INTEGER,
    api_rate_limit_per_hour INTEGER,
    max_worker_threads INTEGER,
    is_active BOOLEAN DEFAULT true,
    enforcement_mode VARCHAR(20) DEFAULT 'soft',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_cpu_positive CHECK (cpu_cores IS NULL OR cpu_cores > 0),
    CONSTRAINT check_memory_positive CHECK (memory_mb IS NULL OR memory_mb > 0),
    CONSTRAINT check_bandwidth_positive CHECK (bandwidth_mbps IS NULL OR bandwidth_mbps > 0),
    CONSTRAINT check_concurrent_connections_positive CHECK (max_concurrent_connections IS NULL OR max_concurrent_connections > 0),
    CONSTRAINT check_storage_positive CHECK (storage_gb IS NULL OR storage_gb > 0),
    CONSTRAINT check_db_connections_positive CHECK (max_db_connections IS NULL OR max_db_connections > 0),
    CONSTRAINT check_queries_positive CHECK (max_queries_per_second IS NULL OR max_queries_per_second > 0),
    CONSTRAINT check_redis_memory_positive CHECK (redis_memory_mb IS NULL OR redis_memory_mb > 0),
    CONSTRAINT check_redis_keys_positive CHECK (redis_max_keys IS NULL OR redis_max_keys > 0),
    CONSTRAINT check_messages_hour_positive CHECK (max_messages_per_hour IS NULL OR max_messages_per_hour > 0),
    CONSTRAINT check_messages_day_positive CHECK (max_messages_per_day IS NULL OR max_messages_per_day > 0),
    CONSTRAINT check_queue_depth_positive CHECK (max_queue_depth IS NULL OR max_queue_depth > 0)
);

CREATE INDEX idx_resource_quotas_gateway ON resource_quotas(gateway_id);
