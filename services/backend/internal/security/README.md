# Security Testing Suite

This package contains comprehensive security tests for the tariff pricing system, covering critical security requirements from the specification.

## Test Coverage

### 1. Cross-Tenant Access Prevention (Requirement 26.1, 26.7)
- **TestCrossTenantAccessPrevention**: Verifies that tenants cannot access each other's data
- Tests same-tenant access (should succeed)
- Tests cross-tenant access (should be blocked)
- Validates proper error responses

### 2. SQL Injection Prevention (Requirement 26.2)
- **TestSQLInjectionPrevention**: Tests various SQL injection attack vectors
- Payloads tested:
  - DROP TABLE attempts
  - OR 1=1 conditions
  - UNION SELECT attacks
  - Comment-based injections
  - Nested queries
- Verifies parameterized queries prevent injection
- Confirms database integrity after attacks

### 3. XSS Prevention (Requirement 26.3)
- **TestXSSPrevention**: Tests cross-site scripting attack prevention
- Payloads tested:
  - Script tag injection
  - Event handler injection
  - SVG-based XSS
  - JavaScript protocol
  - HTML tag injection
- Verifies proper JSON encoding and escaping

### 4. CSRF Protection (Requirement 26.8)
- **TestCSRFProtection**: Validates CSRF token implementation
- Tests valid token acceptance
- Tests missing token rejection
- Tests invalid token rejection
- Verifies state-changing operations require tokens

### 5. API Authentication (Requirement 26.1, 26.2)
- **TestAPIAuthentication**: Tests API key and JWT validation
- Tests valid authentication
- Tests missing credentials
- Tests invalid credentials
- Tests malformed headers
- Tests expired tokens

### 6. Rate Limiting
- **TestRateLimitingByIP**: Prevents abuse through rate limiting
- Tests request counting
- Tests limit enforcement
- Verifies proper error responses

### 7. Password Security
- **TestPasswordHashing**: Validates password hashing
- Tests hash generation
- Tests salt uniqueness
- Tests password verification
- Ensures plaintext passwords are never stored

### 8. Security Headers
- **TestSecureHeaders**: Validates HTTP security headers
- X-Content-Type-Options: nosniff
- X-Frame-Options: DENY
- X-XSS-Protection: 1; mode=block
- Strict-Transport-Security
- Content-Security-Policy

## Running the Tests

### Run all security tests:
```bash
cd services/backend
go test -v ./internal/security/...
```

### Run specific test:
```bash
go test -v ./internal/security -run TestCrossTenantAccessPrevention
```

### Run with coverage:
```bash
go test -v -cover ./internal/security/...
```

### Run with race detection:
```bash
go test -v -race ./internal/security/...
```

## Test Database Setup

The tests require a PostgreSQL test database:

```bash
# Create test database
createdb sms_gateway_test

# Or using psql
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
```

## Security Best Practices Validated

1. **Input Validation**: All user inputs are validated and sanitized
2. **Parameterized Queries**: SQL queries use parameters to prevent injection
3. **Output Encoding**: All outputs are properly encoded (JSON escaping)
4. **Authentication**: All endpoints require valid authentication
5. **Authorization**: Cross-tenant access is prevented
6. **CSRF Protection**: State-changing operations require CSRF tokens
7. **Rate Limiting**: Prevents brute force and DoS attacks
8. **Secure Headers**: HTTP security headers are set
9. **Password Hashing**: Passwords are hashed with salt
10. **Session Management**: Tokens expire and are validated

## Attack Vectors Tested

### SQL Injection
- Classic injection ('; DROP TABLE)
- Boolean-based blind (OR 1=1)
- Union-based (UNION SELECT)
- Comment-based (admin'--)
- Stacked queries

### XSS
- Reflected XSS
- Stored XSS
- DOM-based XSS
- Event handler injection
- Protocol-based XSS

### Authentication Bypass
- Missing credentials
- Invalid credentials
- Expired tokens
- Token tampering
- Session fixation

### CSRF
- Missing token
- Invalid token
- Token reuse
- Cross-origin requests

## Compliance

These tests help ensure compliance with:
- OWASP Top 10 security risks
- Requirements 26.1, 26.2, 26.3, 26.7, 26.8 from the specification
- Industry security best practices

## Continuous Security Testing

Integrate these tests into your CI/CD pipeline:

```yaml
# Example GitHub Actions
- name: Run Security Tests
  run: |
    cd services/backend
    go test -v ./internal/security/...
```

## Security Monitoring

In production, implement:
1. **Logging**: Log all security events (failed auth, cross-tenant attempts)
2. **Alerting**: Alert on suspicious patterns
3. **Audit Trail**: Maintain audit logs for compliance
4. **Penetration Testing**: Regular security audits
5. **Dependency Scanning**: Keep dependencies updated

## Reporting Security Issues

If you discover a security vulnerability:
1. Do NOT open a public issue
2. Contact the security team directly
3. Provide detailed reproduction steps
4. Allow time for patching before disclosure

## Additional Resources

- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [Go Security Best Practices](https://golang.org/doc/security/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)


## Concurrent Isolation Tests

### 9. Concurrent Tenant Isolation (Requirement 26.5)

New comprehensive tests for tenant isolation under high concurrency loads:

- **TestConcurrentTenantOperations**: 10 tenants × 50 operations = 500 concurrent operations
  - Verifies no interference between tenant operations
  - Validates data integrity under load
  - Ensures proper isolation

- **TestConcurrentBalanceOperations**: 5 tenants × 100 deductions = 500 balance operations
  - Tests atomic balance updates
  - Verifies no lost updates
  - Validates correct final balances

- **TestConcurrentDataIsolation**: 2 tenants × 100 inserts = 200 concurrent inserts
  - Verifies no data leakage between tenants
  - Validates correct record counts
  - Ensures no cross-tenant data visibility

- **TestConcurrentRLSEnforcement**: 2 tenants × 200 queries = 400 concurrent queries
  - Tests Row Level Security under load
  - Verifies each tenant sees only their own data
  - Validates RLS policy enforcement

- **TestConcurrentCacheIsolation**: 5 tenants × 100 operations = 500 cache operations
  - Tests Redis cache isolation
  - Verifies no cache pollution
  - Validates correct key isolation

- **TestConcurrentTransactionIsolation**: 2 tenants × 50 transactions = 100 concurrent transactions
  - Tests transaction atomicity
  - Verifies transaction isolation
  - Validates correct balances and counts

### Running Concurrent Isolation Tests

```bash
# Run all concurrent isolation tests
cd services/backend
.\run-concurrent-isolation-tests.ps1

# Or run with Go directly
go test -v -race -timeout 10m ./internal/security -run TestConcurrent

# Run individual concurrent tests
go test -v -run TestConcurrentTenantOperations ./internal/security
go test -v -run TestConcurrentBalanceOperations ./internal/security
go test -v -run TestConcurrentDataIsolation ./internal/security
```

### Concurrent Test Documentation

- **CONCURRENT_ISOLATION_TEST_GUIDE.md**: Complete test documentation
- **CONCURRENT_ISOLATION_QUICK_START.md**: Quick reference guide
- **run-concurrent-isolation-tests.ps1**: Automated test runner

### Performance Expectations

- **Duration**: < 15 seconds total
- **Throughput**: 500+ operations/second
- **Memory**: < 500MB
- **CPU**: < 80%
- **Error Rate**: 0%

