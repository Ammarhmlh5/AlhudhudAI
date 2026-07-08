-- Seed Channels for Performance Testing
-- Creates 50 test channels (5 per tenant)

-- Clean up existing test channels
DELETE FROM channels WHERE device_id LIKE 'PERF_TEST_%';

-- Insert test channels for each tenant
-- Tenant 1 channels
INSERT INTO channels (tenant_id, device_id, device_name, status, priority, capacity, current_load, created_at, updated_at)
SELECT 
    t.id,
    'PERF_TEST_' || t.id || '_' || s.n,
    'Performance Test Device ' || s.n,
    'ONLINE',
    CASE WHEN s.n <= 2 THEN 10 ELSE 5 END, -- First 2 channels have high priority
    100,
    0,
    NOW(),
    NOW()
FROM 
    (SELECT id FROM tenants WHERE email LIKE 'test%@performance.test') t
CROSS JOIN 
    generate_series(1, 5) s(n);

-- Display created channels
SELECT 
    c.id,
    t.name as tenant_name,
    c.device_id,
    c.status,
    c.priority,
    c.capacity
FROM channels c
JOIN tenants t ON c.tenant_id = t.id
WHERE c.device_id LIKE 'PERF_TEST_%'
ORDER BY t.id, c.priority DESC, c.id;

-- Summary
SELECT 
    COUNT(*) as total_test_channels,
    COUNT(DISTINCT tenant_id) as tenants_with_channels,
    SUM(CASE WHEN status = 'ONLINE' THEN 1 ELSE 0 END) as online_channels
FROM channels 
WHERE device_id LIKE 'PERF_TEST_%';
