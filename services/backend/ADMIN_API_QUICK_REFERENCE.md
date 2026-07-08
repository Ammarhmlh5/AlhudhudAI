# Admin APIs Quick Reference Card

## Quick Start

```powershell
# Run automated tests
cd services/backend
./test-admin-apis.ps1
```

## Base Configuration

```bash
BASE_URL="http://localhost:8080"
API_KEY="your-api-key-here"
TENANT_ID="00000000-0000-0000-0000-000000000001"
```

## Common Headers

```bash
-H "Authorization: Bearer $API_KEY"
-H "Content-Type: application/json"
```

## Quick Test Commands

### 1. Company Rating

```bash
# Get rating
curl -X GET "$BASE_URL/api/admin/companies/$TENANT_ID/rating" \
  -H "Authorization: Bearer $API_KEY"

# Get all ratings
curl -X GET "$BASE_URL/api/admin/companies/ratings?limit=10" \
  -H "Authorization: Bearer $API_KEY"

# Get rating history
curl -X GET "$BASE_URL/api/admin/companies/$TENANT_ID/rating/history" \
  -H "Authorization: Bearer $API_KEY"
```

### 2. Performance Metrics

```bash
# Get hourly metrics
curl -X GET "$BASE_URL/api/admin/companies/$TENANT_ID/metrics?period_type=hourly&limit=24" \
  -H "Authorization: Bearer $API_KEY"

# Get daily metrics
curl -X GET "$BASE_URL/api/admin/companies/$TENANT_ID/metrics?period_type=daily&limit=7" \
  -H "Authorization: Bearer $API_KEY"
```

### 3. Routing Configuration

```bash
# Get routing config
curl -X GET "$BASE_URL/api/admin/routing/$TENANT_ID" \
  -H "Authorization: Bearer $API_KEY"

# Update to shared routing
curl -X PUT "$BASE_URL/api/admin/routing/$TENANT_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"routing_type":"shared"}'

# Update to dedicated routing
curl -X PUT "$BASE_URL/api/admin/routing/$TENANT_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "routing_type":"dedicated",
    "dedicated_server_id":"00000000-0000-0000-0000-000000000002",
    "dedicated_server_url":"https://dedicated.example.com"
  }'

# Get routing metrics
curl -X GET "$BASE_URL/api/admin/routing/$TENANT_ID/metrics" \
  -H "Authorization: Bearer $API_KEY"
```

### 4. Distribution Strategy

```bash
# Get distribution config
curl -X GET "$BASE_URL/api/admin/distribution/$TENANT_ID" \
  -H "Authorization: Bearer $API_KEY"

# Update to round robin
curl -X PUT "$BASE_URL/api/admin/distribution/$TENANT_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"strategy_type":"round_robin","strategy_config":{}}'

# Get devices
curl -X GET "$BASE_URL/api/admin/distribution/$TENANT_ID/devices" \
  -H "Authorization: Bearer $API_KEY"

# Get distribution metrics
curl -X GET "$BASE_URL/api/admin/distribution/$TENANT_ID/metrics" \
  -H "Authorization: Bearer $API_KEY"
```

### 5. Server Health

```bash
# Get all servers
curl -X GET "$BASE_URL/api/admin/servers" \
  -H "Authorization: Bearer $API_KEY"

# Get healthy servers only
curl -X GET "$BASE_URL/api/admin/servers?status=healthy" \
  -H "Authorization: Bearer $API_KEY"

# Get server details
curl -X GET "$BASE_URL/api/admin/servers/$SERVER_ID" \
  -H "Authorization: Bearer $API_KEY"

# Get health history
curl -X GET "$BASE_URL/api/admin/servers/$SERVER_ID/health/history?limit=100" \
  -H "Authorization: Bearer $API_KEY"
```

### 6. Alert Management

```bash
# Get all alerts
curl -X GET "$BASE_URL/api/admin/alerts?limit=10" \
  -H "Authorization: Bearer $API_KEY"

# Get critical alerts
curl -X GET "$BASE_URL/api/admin/alerts?severity=critical" \
  -H "Authorization: Bearer $API_KEY"

# Get active alerts
curl -X GET "$BASE_URL/api/admin/alerts?status=active" \
  -H "Authorization: Bearer $API_KEY"

# Get alert details
curl -X GET "$BASE_URL/api/admin/alerts/$ALERT_ID" \
  -H "Authorization: Bearer $API_KEY"

# Acknowledge alert
curl -X PUT "$BASE_URL/api/admin/alerts/$ALERT_ID/acknowledge" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"acknowledged_by":"00000000-0000-0000-0000-000000000003"}'

# Resolve alert
curl -X PUT "$BASE_URL/api/admin/alerts/$ALERT_ID/resolve" \
  -H "Authorization: Bearer $API_KEY"
```

### 7. Signal/Push System

```bash
# Get timing config
curl -X GET "$BASE_URL/api/admin/signalpush/$TENANT_ID/config" \
  -H "Authorization: Bearer $API_KEY"

# Update timing config
curl -X PUT "$BASE_URL/api/admin/signalpush/$TENANT_ID/config" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "signal_min_gap_ms":2000,
    "push_min_gap_ms":500,
    "violation_rate_threshold":50,
    "z_score_threshold":3.0,
    "peak_ratio_threshold":2.0
  }'

# Get statistics
curl -X GET "$BASE_URL/api/admin/signalpush/$TENANT_ID/statistics" \
  -H "Authorization: Bearer $API_KEY"

# Get violations
curl -X GET "$BASE_URL/api/admin/signalpush/$TENANT_ID/violations?limit=100" \
  -H "Authorization: Bearer $API_KEY"

# Get severe violations
curl -X GET "$BASE_URL/api/admin/signalpush/$TENANT_ID/violations?severity=severe" \
  -H "Authorization: Bearer $API_KEY"

# Force push mode
curl -X POST "$BASE_URL/api/admin/signalpush/$TENANT_ID/force-mode" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"mode":"push","reason":"Testing","duration":60}'

# Force signal mode
curl -X POST "$BASE_URL/api/admin/signalpush/$TENANT_ID/force-mode" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"mode":"signal","reason":"Restoring normal operation"}'
```

### 8. Dashboard Overview

```bash
# Get dashboard overview
curl -X GET "$BASE_URL/api/admin/dashboard/overview" \
  -H "Authorization: Bearer $API_KEY"
```

## Common Query Parameters

### Pagination
```
?limit=10&offset=20
```

### Date Range
```
?start_date=2024-01-01T00:00:00Z&end_date=2024-12-31T23:59:59Z
```

### Filtering
```
?category=excellent
?severity=critical
?status=active
?period_type=daily
?message_type=signal
```

## Response Format

### Success Response
```json
{
  "success": true,
  "data": { ... }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Error message",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## HTTP Status Codes

- `200 OK` - Success
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing or invalid API key
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Testing Checklist

- [ ] Authentication works
- [ ] All GET endpoints return data
- [ ] All PUT endpoints update data
- [ ] All POST endpoints create/trigger actions
- [ ] Pagination works
- [ ] Filtering works
- [ ] Input validation catches errors
- [ ] Rate limiting works
- [ ] Response times < 200ms

## Troubleshooting

### 401 Unauthorized
- Check API key is correct
- Verify Authorization header format

### 400 Bad Request
- Check request body JSON format
- Verify all required fields are present
- Check data types match expected

### 404 Not Found
- Verify resource ID exists
- Check endpoint URL is correct

### 500 Internal Server Error
- Check server logs
- Verify database connection
- Check Redis connection

## Performance Benchmarks

```bash
# Test response time
time curl -X GET "$BASE_URL/api/admin/dashboard/overview" \
  -H "Authorization: Bearer $API_KEY"

# Test concurrent requests
ab -n 1000 -c 100 \
  -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/api/admin/companies/ratings"
```

## Next Steps

1. Run automated test script
2. Test each endpoint manually
3. Verify authentication and authorization
4. Test input validation
5. Check rate limiting
6. Measure performance
7. Document any issues
8. Proceed to frontend testing
