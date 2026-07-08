-- Seed Tenants for Performance Testing
-- Creates 10 test tenants with sufficient balance

-- Clean up existing test data
DELETE FROM tenants WHERE email LIKE 'test%@performance.test';

-- Insert test tenants
INSERT INTO tenants (name, email, phone, balance, status, created_at, updated_at)
VALUES
    ('Performance Test Tenant 1', 'test1@performance.test', '+966500000001', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 2', 'test2@performance.test', '+966500000002', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 3', 'test3@performance.test', '+966500000003', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 4', 'test4@performance.test', '+966500000004', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 5', 'test5@performance.test', '+966500000005', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 6', 'test6@performance.test', '+966500000006', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 7', 'test7@performance.test', '+966500000007', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 8', 'test8@performance.test', '+966500000008', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 9', 'test9@performance.test', '+966500000009', 100000.0000, 'active', NOW(), NOW()),
    ('Performance Test Tenant 10', 'test10@performance.test', '+966500000010', 100000.0000, 'active', NOW(), NOW());

-- Display created tenants
SELECT id, name, email, balance, status 
FROM tenants 
WHERE email LIKE 'test%@performance.test'
ORDER BY id;

-- Summary
SELECT 
    COUNT(*) as total_test_tenants,
    SUM(balance) as total_balance
FROM tenants 
WHERE email LIKE 'test%@performance.test';
