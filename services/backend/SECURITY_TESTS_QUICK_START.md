# Security Tests - Quick Start Guide

## Overview

This guide helps you quickly run comprehensive security tests for the tariff pricing system. These tests validate critical security requirements including Cross-Tenant Access Prevention, SQL Injection, XSS, CSRF, and API Authentication.

## Prerequisites

1. **Go 1.21+** installed
2. **PostgreSQL** running on localhost:5432
3. **Test database** (will be created automatically)

## Quick Start (Windows)

```powershell
# Run all security tests
.\run-security-tests.ps1
```

## Quick Start (Linux/Mac)

```bash
# Make script executable
chmod +x run-security-tests.ps1

# Run all security tests
./run-security-tests.ps1
```

## What Gets Tested

### 1. Cross-Tenant Access Prevention ✓
- Validates tenant isolation
- Blocks cross-tenant data access
- Tests proper error responses

### 2. SQL Injection Prevention ✓
- Tests 9+ injection attack vectors
- Validates parameterized queries
- Confirms database integrity

### 3. XSS Prevention ✓
- Tests 10+ XSS attack vectors
- Validates output encoding
- Confirms proper escaping

### 4. CSRF Protection ✓
- Tests token validation
- Validates state-changing operations
- Confirms proper error handling

### 5. API Authentication ✓
- Tests API key validation
- Tests JWT authentication
- Validates token expiration

### 6. Rate Limiting ✓
- Tests request throttling
- Validates IP-based limiting
- Confirms abuse prevention

### 7. Password Security ✓
- Tests password hashing
- Validates salt uniqueness
- Confirms secure storage

### 8. Security Headers ✓
- Tests HTTP security headers
- Validates CSP, HSTS, etc.
- Confirms proper configuration

## Expected Output

```
========================================
Security Tests - Tariff Pricing System
========================================

✓ Go is installed: go version go1.21.0
✓ PostgreSQL is running
✓ Test database exists

========================================
Running Security Tests
========================================

Test 1: Cross-Tenant Access Prevention
PASS: TestCrossTenantAccessPrevention

Test 2: SQL Injection Prevention
PASS: TestSQLInjectionPrevention

Test 3: XSS Prevention
PASS: TestXSSPrevention

Test 4: CSRF Protection
PASS: TestCSRFProtection

Test 5: API Authentication
PASS: TestAPIAuthentication

Test 6: Rate Limiting
PASS: TestRateLimitingByIP

Test 7: Password Hashing
PASS: TestPasswordHashing

Test 8: Security Headers
PASS: TestSecureHeaders

========================================
Test Results Summary
========================================

✓ Cross-Tenant Access Prevention
✓ SQL Injection Prevention
✓ XSS Prevention
✓ CSRF Protection
✓ API Authentication
✓ Rate Limiting
✓ Password Hashing
✓ Security Headers

Total: 8 tests
Passed: 8
Failed: 0

All security tests passed! ✓
```

## Running Individual Tests

```bash
cd services/backend

# Test cross-tenant access prevention
go test -v -run TestCrossTenantAccessPrevention ./internal/security/...

# Test SQL injection prevention
go test -v -run TestSQLInjectionPrevention ./internal/security/...

# Test XSS prevention
go test -v -run TestXSSPrevention ./internal/security/...

# Test CSRF protection
go test -v -run TestCSRFProtection ./internal/security/...

# Test API authentication
go test -v -run TestAPIAuthentication ./internal/security/...
```

## Coverage Report

```bash
cd services/backend

# Generate coverage report
go test -cover -coverprofile=security_coverage.out ./internal/security/...

# View coverage summary
go tool cover -func=security_coverage.out

# View HTML coverage report
go tool cover -html=security_coverage.out
```

## Troubleshooting

### PostgreSQL Connection Error

```
ERROR: Cannot connect to PostgreSQL
```

**Solution:**
```bash
# Check if PostgreSQL is running
pg_isready

# Start PostgreSQL (Windows)
net start postgresql-x64-14

# Start PostgreSQL (Linux)
sudo systemctl start postgresql

# Start PostgreSQL (Mac)
brew services start postgresql
```

### Test Database Creation Error

```
ERROR: permission denied to create database
```

**Solution:**
```bash
# Create database manually
createdb -U postgres sms_gateway_test

# Or using psql
psql -U postgres -c "CREATE DATABASE sms_gateway_test;"
```

### Go Not Found

```
ERROR: Go is not installed or not in PATH
```

**Solution:**
1. Download Go from https://golang.org/dl/
2. Install and add to PATH
3. Verify: `go version`

## Attack Vectors Tested

### SQL Injection Payloads
- `'; DROP TABLE tariffs; --`
- `1' OR '1'='1`
- `1' UNION SELECT * FROM users--`
- `admin'--`
- `' OR 1=1--`
- And 4 more variants

### XSS Payloads
- `<script>alert('XSS')</script>`
- `<img src=x onerror=alert('XSS')>`
- `<svg/onload=alert('XSS')>`
- `javascript:alert('XSS')`
- `<iframe src='javascript:alert("XSS")'></iframe>`
- And 5 more variants

## Security Requirements Validated

- ✓ Requirement 26.1: Tenant Isolation
- ✓ Requirement 26.2: Input Validation
- ✓ Requirement 26.3: Output Encoding
- ✓ Requirement 26.7: Cross-Tenant Prevention
- ✓ Requirement 26.8: CSRF Protection

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Security Tests

on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Security Tests
        run: |
          cd services/backend
          go test -v ./internal/security/...
```

## Best Practices

1. **Run tests before every deployment**
2. **Include in CI/CD pipeline**
3. **Review failed tests immediately**
4. **Update tests when adding features**
5. **Perform regular penetration testing**
6. **Monitor security logs in production**
7. **Keep dependencies updated**
8. **Implement security headers**
9. **Use HTTPS in production**
10. **Enable rate limiting**

## Next Steps

1. ✓ Run security tests
2. Review test results
3. Fix any failures
4. Integrate into CI/CD
5. Deploy with confidence

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://golang.org/doc/security/)
- [Security Testing Guide](./internal/security/README.md)

## Support

For security issues or questions:
1. Review the detailed README: `services/backend/internal/security/README.md`
2. Check test output for specific errors
3. Contact the security team for vulnerabilities

---

**Remember:** Security is not a one-time task. Run these tests regularly and stay vigilant!
