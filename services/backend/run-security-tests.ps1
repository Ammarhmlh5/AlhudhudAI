#!/usr/bin/env pwsh

# Security Tests Runner Script
# This script runs comprehensive security tests for the tariff pricing system

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Security Tests - Tariff Pricing System" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
Write-Host "Checking Go installation..." -ForegroundColor Yellow
$goVersion = go version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}
Write-Host "✓ Go is installed: $goVersion" -ForegroundColor Green
Write-Host ""

# Check if PostgreSQL is running
Write-Host "Checking PostgreSQL connection..." -ForegroundColor Yellow
$env:PGPASSWORD = "postgres"
$pgCheck = psql -U postgres -h localhost -c "SELECT 1;" 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Cannot connect to PostgreSQL" -ForegroundColor Red
    Write-Host "Please ensure PostgreSQL is running on localhost:5432" -ForegroundColor Yellow
    exit 1
}
Write-Host "✓ PostgreSQL is running" -ForegroundColor Green
Write-Host ""

# Create test database if it doesn't exist
Write-Host "Setting up test database..." -ForegroundColor Yellow
$dbExists = psql -U postgres -h localhost -lqt 2>$null | Select-String -Pattern "sms_gateway_test"
if (-not $dbExists) {
    Write-Host "Creating test database..." -ForegroundColor Yellow
    psql -U postgres -h localhost -c "CREATE DATABASE sms_gateway_test;" 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Test database created" -ForegroundColor Green
    }
} else {
    Write-Host "✓ Test database exists" -ForegroundColor Green
}
Write-Host ""

# Navigate to backend directory
Set-Location services/backend

# Run security tests
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running Security Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Test 1: Cross-Tenant Access Prevention" -ForegroundColor Yellow
Write-Host "Testing that tenants cannot access each other's data..." -ForegroundColor Gray
go test -v -run TestCrossTenantAccessPrevention ./internal/security/...
$test1Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 2: SQL Injection Prevention" -ForegroundColor Yellow
Write-Host "Testing protection against SQL injection attacks..." -ForegroundColor Gray
go test -v -run TestSQLInjectionPrevention ./internal/security/...
$test2Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 3: XSS Prevention" -ForegroundColor Yellow
Write-Host "Testing protection against cross-site scripting..." -ForegroundColor Gray
go test -v -run TestXSSPrevention ./internal/security/...
$test3Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 4: CSRF Protection" -ForegroundColor Yellow
Write-Host "Testing CSRF token validation..." -ForegroundColor Gray
go test -v -run TestCSRFProtection ./internal/security/...
$test4Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 5: API Authentication" -ForegroundColor Yellow
Write-Host "Testing API key and JWT authentication..." -ForegroundColor Gray
go test -v -run TestAPIAuthentication ./internal/security/...
$test5Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 6: Rate Limiting" -ForegroundColor Yellow
Write-Host "Testing rate limiting by IP address..." -ForegroundColor Gray
go test -v -run TestRateLimitingByIP ./internal/security/...
$test6Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 7: Password Hashing" -ForegroundColor Yellow
Write-Host "Testing password hashing and verification..." -ForegroundColor Gray
go test -v -run TestPasswordHashing ./internal/security/...
$test7Result = $LASTEXITCODE
Write-Host ""

Write-Host "Test 8: Security Headers" -ForegroundColor Yellow
Write-Host "Testing HTTP security headers..." -ForegroundColor Gray
go test -v -run TestSecureHeaders ./internal/security/...
$test8Result = $LASTEXITCODE
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Results Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$results = @(
    @{Name="Cross-Tenant Access Prevention"; Result=$test1Result},
    @{Name="SQL Injection Prevention"; Result=$test2Result},
    @{Name="XSS Prevention"; Result=$test3Result},
    @{Name="CSRF Protection"; Result=$test4Result},
    @{Name="API Authentication"; Result=$test5Result},
    @{Name="Rate Limiting"; Result=$test6Result},
    @{Name="Password Hashing"; Result=$test7Result},
    @{Name="Security Headers"; Result=$test8Result}
)

$passedCount = 0
$failedCount = 0

foreach ($result in $results) {
    if ($result.Result -eq 0) {
        Write-Host "✓ $($result.Name)" -ForegroundColor Green
        $passedCount++
    } else {
        Write-Host "✗ $($result.Name)" -ForegroundColor Red
        $failedCount++
    }
}

Write-Host ""
Write-Host "Total: $($results.Count) tests" -ForegroundColor Cyan
Write-Host "Passed: $passedCount" -ForegroundColor Green
Write-Host "Failed: $failedCount" -ForegroundColor Red
Write-Host ""

# Run all tests with coverage
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Running All Tests with Coverage" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

go test -v -cover -coverprofile=security_coverage.out ./internal/security/...
$coverageResult = $LASTEXITCODE

if ($coverageResult -eq 0) {
    Write-Host ""
    Write-Host "Coverage Report:" -ForegroundColor Cyan
    go tool cover -func=security_coverage.out
    Write-Host ""
    Write-Host "To view HTML coverage report, run:" -ForegroundColor Yellow
    Write-Host "go tool cover -html=security_coverage.out" -ForegroundColor Gray
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Security Recommendations" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. Run these tests in CI/CD pipeline" -ForegroundColor Yellow
Write-Host "2. Perform regular penetration testing" -ForegroundColor Yellow
Write-Host "3. Keep dependencies updated" -ForegroundColor Yellow
Write-Host "4. Monitor security logs in production" -ForegroundColor Yellow
Write-Host "5. Implement security headers in production" -ForegroundColor Yellow
Write-Host "6. Use HTTPS in production" -ForegroundColor Yellow
Write-Host "7. Enable rate limiting in production" -ForegroundColor Yellow
Write-Host "8. Implement audit logging" -ForegroundColor Yellow
Write-Host ""

# Exit with appropriate code
if ($failedCount -gt 0) {
    Write-Host "Some tests failed. Please review the output above." -ForegroundColor Red
    exit 1
} else {
    Write-Host "All security tests passed! ✓" -ForegroundColor Green
    exit 0
}
