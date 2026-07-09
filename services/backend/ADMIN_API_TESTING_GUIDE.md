# Admin Interface APIs Testing Guide

## Overview

This guide provides comprehensive instructions for testing all Admin Interface API endpoints. The admin APIs provide functionality for monitoring companies, managing routing, distribution strategies, server health, alerts, and Signal/Push timing systems.

## Prerequisites

### 1. Environment Setup

- Backend server running on `http://localhost:8080`
- PostgreSQL database with all migrations applied
- Redis server running
- Valid API key for authentication

### 2. Test Data

Ensure you have test data in the database:
- At least one tenant/company
- Routing configurations
- Distribution strategies
- Server health records
- Alerts
- Timing violations

## API Endpoints Overview

### 1. Company Rating APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/companies/{tenant_id}/rating` | GET | Get current rating for a company |
| `/api/admin/companies/ratings` | GET | Get all company ratings with filters |
| `/api/admin/companies/{tenant_id}/rating/history` | GET | Get rating history for a company |

### 2. Performance Metrics APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/companies/{tenant_id}/metrics` | GET | Get performance metrics with filters |

### 3. Routing Configuration APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/routing/{tenant_id}` | GET | Get routing configuration |
| `/api/admin/routing/{tenant_id}` | PUT | Update routing configuration |
| `/api/admin/routing/{tenant_id}/metrics` | GET | Get routing metrics |

### 4. Distribution Strategy APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/distribution/{tenant_id}` | GET | Get distribution configuration |
| `/api/admin/distribution/{tenant_id}` | PUT | Update distribution configuration |
| `/api/admin/distribution/{tenant_id}/devices` | GET | Get available devices |
| `/api/admin/distribution/{tenant_id}/metrics` | GET | Get distribution metrics |

### 5. Server Health APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/servers` | GET | Get all servers with health status |
| `/api/admin/servers/{server_id}` | GET | Get server details |
| `/api/admin/servers/{server_id}/health/history` | GET | Get server health history |

### 6. Alert Management APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/alerts` | GET | Get all alerts with filters |
| `/api/admin/alerts/{alert_id}` | GET | Get alert details |
| `/api/admin/alerts/{alert_id}/acknowledge` | PUT | Acknowledge an alert |
| `/api/admin/alerts/{alert_id}/resolve` | PUT | Resolve an alert |
| `/api/admin/alerts/config` | POST | Configure alert settings |

### 7. Signal/Push System APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/signalpush/{tenant_id}/config` | GET | Get timing configuration |
| `/api/admin/signalpush/{tenant_id}/config` | PUT | Update timing configuration |
| `/api/admin/signalpush/{tenant_id}/statistics` | GET | Get timing statistics |
| `/api/admin/signalpush/{tenant_id}/violations` | GET | Get timing violations |
| `/api/admin/signalpush/{tenant_id}/force-mode` | POST | Force Signal or Push mode |

### 8. Dashboard Overview API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/dashboard/overview` | GET | Get aggregated dashboard statistics |

## Testing Methods

### Method 1: Automated Testing Script (Recommended)

Run the PowerShell testing script:

```powershell
cd services/backend
./test-admin-apis.ps1
```

The script will:
- Test all API endpoints
- Validate response status codes
- Check authentication and authorization
- Test input validation
- Provide a summary report

### Method 2: Manual Testing with cURL

#### Example: Get Company Rating

```bash
curl -X GET \
  http://localhost:8080/api/admin/companies/00000000-0000-0000-0000-000000000001/rating \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json"
```

#### Example: Update Routing Configuration

```bash
curl -X PUT \
  http://localhost:8080/api/admin/routing/00000000-0000-0000-0000-000000000001 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "routing_type": "dedicated",
    "dedicated_server_id": "00000000-0000-0000-0000-000000000002",
    "dedicated_server_url": "https://dedicated-server.example.com",
    "dedicated_server_api_key": "server-api-key"
  }'
```

#### Example: Force Push Mode

```bash
curl -X POST \
  http://localhost:8080/api/admin/signalpush/00000000-0000-0000-0000-000000000001/force-mode \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "push",
    "reason": "High violation rate detected",
    "duration": 60
  }'
```

### Method 3: Postman Collection

Import the Postman collection (if available) and run the collection runner to test all endpoints.

## Authentication Testing

### Test 1: No API Key

```bash
curl -X GET http://localhost:8080/api/admin/companies/ratings
```

**Expected Result:** 401 Unauthorized

### Test 2: Invalid API Key

```bash
curl -X GET \
  http://localhost:8080/api/admin/companies/ratings \
  -H "Authorization: Bearer invalid-key"
```

**Expected Result:** 401 Unauthorized

### Test 3: Valid API Key

```bash
curl -X GET \
  http://localhost:8080/api/admin/companies/ratings \
  -H "Authorization: Bearer YOUR_VALID_API_KEY"
```

**Expected Result:** 200 OK with data

## Input Validation Testing

### Test 1: Invalid UUID Format

```bash
curl -X GET \
  http://localhost:8080/api/admin/companies/invalid-uuid/rating \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** 400 Bad Request with error message

### Test 2: Invalid Query Parameters

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings?category=invalid" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** 400 Bad Request with error message

### Test 3: Invalid JSON Body

```bash
curl -X PUT \
  http://localhost:8080/api/admin/routing/00000000-0000-0000-0000-000000000001 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"invalid_json"'
```

**Expected Result:** 400 Bad Request with error message

## Rate Limiting Testing

Test rate limiting by sending multiple requests in quick succession:

```bash
for i in {1..1100}; do
  curl -X GET \
    http://localhost:8080/api/admin/companies/ratings \
    -H "Authorization: Bearer YOUR_API_KEY"
done
```

**Expected Result:** After 1000 requests, should receive 429 Too Many Requests

## Performance Testing

### Response Time Testing

Measure API response times:

```bash
time curl -X GET \
  http://localhost:8080/api/admin/dashboard/overview \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Response time < 2 seconds

### Concurrent Requests Testing

Use Apache Bench or similar tools:

```bash
ab -n 1000 -c 100 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/api/admin/companies/ratings
```

**Expected Result:** 
- Average response time < 200ms
- No failed requests
- Consistent performance

## Error Handling Testing

### Test 1: Non-existent Resource

```bash
curl -X GET \
  http://localhost:8080/api/admin/companies/00000000-0000-0000-0000-999999999999/rating \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** 404 Not Found with error message

### Test 2: Database Connection Error

Temporarily stop the database and test:

```bash
curl -X GET \
  http://localhost:8080/api/admin/companies/ratings \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** 500 Internal Server Error with generic error message (no sensitive info)

## Pagination Testing

### Test 1: Default Pagination

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Returns first 100 records

### Test 2: Custom Pagination

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings?limit=10&offset=20" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Returns 10 records starting from offset 20

### Test 3: Maximum Limit

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings?limit=10000" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Returns maximum allowed records (1000)

## Filtering Testing

### Test 1: Filter by Category

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings?category=excellent" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Returns only companies with excellent rating

### Test 2: Filter by Date Range

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings?start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Returns ratings within the specified date range

### Test 3: Multiple Filters

```bash
curl -X GET \
  "http://localhost:8080/api/admin/alerts?severity=critical&status=active&limit=10" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Returns active critical alerts

## Data Consistency Testing

### Test 1: Update and Verify

1. Update routing configuration:
```bash
curl -X PUT \
  http://localhost:8080/api/admin/routing/00000000-0000-0000-0000-000000000001 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"routing_type": "dedicated", "dedicated_server_id": "00000000-0000-0000-0000-000000000002", "dedicated_server_url": "https://test.example.com"}'
```

2. Retrieve and verify:
```bash
curl -X GET \
  http://localhost:8080/api/admin/routing/00000000-0000-0000-0000-000000000001 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** Retrieved data matches updated data

## Security Testing

### Test 1: SQL Injection Attempt

```bash
curl -X GET \
  "http://localhost:8080/api/admin/companies/ratings?category=excellent' OR '1'='1" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Expected Result:** 400 Bad Request or properly sanitized query

### Test 2: XSS Attempt

```bash
curl -X POST \
  http://localhost:8080/api/admin/alerts/config \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "00000000-0000-0000-0000-000000000001", "alert_name": "<script>alert(\"XSS\")</script>"}'
```

**Expected Result:** Input sanitized or rejected

## Test Checklist

- [ ] All endpoints return correct status codes
- [ ] Authentication works correctly
- [ ] Authorization prevents unauthorized access
- [ ] Input validation catches invalid data
- [ ] Rate limiting works as expected
- [ ] Response times meet requirements (< 200ms average)
- [ ] Pagination works correctly
- [ ] Filtering works correctly
- [ ] Error messages are informative but not revealing sensitive info
- [ ] Data consistency is maintained
- [ ] Security measures prevent common attacks
- [ ] API documentation matches implementation

## Troubleshooting

### Issue: 401 Unauthorized

**Solution:** Check API key configuration and ensure it's valid

### Issue: 500 Internal Server Error

**Solution:** Check server logs for detailed error messages

### Issue: Slow Response Times

**Solution:** 
- Check database query performance
- Verify indexes are created
- Check Redis cache hit rate
- Monitor server resources

### Issue: Rate Limiting Too Aggressive

**Solution:** Adjust rate limiting configuration in middleware

## Next Steps

After completing API testing:

1. Document any issues found
2. Create bug reports for failures
3. Update API documentation if needed
4. Proceed to frontend integration testing
5. Perform end-to-end testing

## Additional Resources

- API Documentation: `docs/ADMIN_API_REFERENCE.md`
- Requirements: `.kiro/specs/admin-interface-enhancements/requirements.md`
- Design Document: `.kiro/specs/admin-interface-enhancements/design.md`
- Implementation Summary: `services/backend/internal/api/ADMIN_API_IMPLEMENTATION_SUMMARY.md`
