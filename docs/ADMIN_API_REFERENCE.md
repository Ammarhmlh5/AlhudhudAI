# Admin API Reference

**Version:** 1.0.0  
**Base URL:** `/api/admin`  
**Authentication:** Bearer Token (JWT)

---

## Table of Contents

1. [Authentication](#authentication)
2. [Dashboard APIs](#dashboard-apis)
3. [Company Rating APIs](#company-rating-apis)
4. [Server Health APIs](#server-health-apis)
5. [Alert Management APIs](#alert-management-apis)
6. [Routing Configuration APIs](#routing-configuration-apis)
7. [Distribution Strategy APIs](#distribution-strategy-apis)
8. [Signal/Push System APIs](#signalpush-system-apis)
9. [Error Codes](#error-codes)
10. [Rate Limiting](#rate-limiting)

---

## Authentication

All Admin API endpoints require authentication using a Bearer token.

### Headers

```http
Authorization: Bearer <your-jwt-token>
Content-Type: application/json
```

### Roles

- **SUPER_ADMIN**: Full access to all endpoints
- **STAFF**: Read-only access to most endpoints
- **TENANT_ADMIN**: Limited access to own tenant data

---

## Dashboard APIs

### Get Dashboard Overview

Get overall system statistics and metrics.

**Endpoint:** `GET /api/admin/dashboard/overview`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "total_companies": 150,
  "active_servers": 45,
  "total_messages_today": 1500000,
  "success_rate": 98.5,
  "active_alerts": 3,
  "rating_distribution": {
    "excellent": 45,
    "good": 70,
    "average": 25,
    "poor": 10
  }
}
```

---

## Company Rating APIs

### Get All Company Ratings

Get ratings for all companies with optional filtering.

**Endpoint:** `GET /api/admin/companies/ratings`

**Authorization:** SUPER_ADMIN, STAFF

**Query Parameters:**
- `category` (optional): Filter by rating category (excellent, good, average, poor)
- `min_rating` (optional): Minimum rating score (0-100)
- `max_rating` (optional): Maximum rating score (0-100)
- `limit` (optional): Number of results (default: 50, max: 100)
- `offset` (optional): Pagination offset (default: 0)

**Example Request:**
```http
GET /api/admin/companies/ratings?category=excellent&limit=20
```

**Response:**
```json
{
  "companies": [
    {
      "tenant_id": "tenant-123",
      "company_name": "ABC Corp",
      "overall_rating": 92.5,
      "rating_category": "excellent",
      "success_rate": 99.2,
      "sending_volume": 50000,
      "payment_compliance": 100.0,
      "service_quality_score": 95.0,
      "last_updated": "2026-04-09T10:30:00Z"
    }
  ],
  "total": 45,
  "limit": 20,
  "offset": 0
}
```

### Get Company Rating

Get detailed rating for a specific company.

**Endpoint:** `GET /api/admin/companies/{tenant_id}/rating`

**Authorization:** SUPER_ADMIN, STAFF, TENANT_ADMIN (own data only)

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "company_name": "ABC Corp",
  "overall_rating": 92.5,
  "rating_category": "excellent",
  "components": {
    "success_rate": 99.2,
    "sending_volume": 50000,
    "payment_compliance": 100.0,
    "service_quality_score": 95.0
  },
  "weights": {
    "success_rate": 0.40,
    "sending_volume": 0.20,
    "payment_compliance": 0.25,
    "service_quality": 0.15
  },
  "last_updated": "2026-04-09T10:30:00Z"
}
```

### Get Company Rating History

Get historical rating data for a company.

**Endpoint:** `GET /api/admin/companies/{tenant_id}/rating/history`

**Authorization:** SUPER_ADMIN, STAFF, TENANT_ADMIN (own data only)

**Query Parameters:**
- `period_start` (required): Start date (ISO 8601)
- `period_end` (required): End date (ISO 8601)

**Example Request:**
```http
GET /api/admin/companies/tenant-123/rating/history?period_start=2026-03-01T00:00:00Z&period_end=2026-04-09T23:59:59Z
```

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "history": [
    {
      "timestamp": "2026-03-01T00:00:00Z",
      "overall_rating": 88.5,
      "rating_category": "good"
    },
    {
      "timestamp": "2026-03-15T00:00:00Z",
      "overall_rating": 90.2,
      "rating_category": "excellent"
    }
  ]
}
```

### Get Company Performance Metrics

Get detailed performance metrics for a company.

**Endpoint:** `GET /api/admin/companies/{tenant_id}/metrics`

**Authorization:** SUPER_ADMIN, STAFF, TENANT_ADMIN (own data only)

**Query Parameters:**
- `period_type` (optional): hourly, daily, monthly (default: daily)
- `period_start` (optional): Start date (ISO 8601)
- `period_end` (optional): End date (ISO 8601)

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "metrics": [
    {
      "period_start": "2026-04-09T00:00:00Z",
      "period_end": "2026-04-09T23:59:59Z",
      "period_type": "daily",
      "total_messages": 50000,
      "successful_messages": 49600,
      "failed_messages": 400,
      "success_rate": 99.2,
      "error_rate": 0.8,
      "payment_compliance": 100.0
    }
  ]
}
```

---

## Server Health APIs

### Get All Servers

Get list of all dedicated servers with health status.

**Endpoint:** `GET /api/admin/servers`

**Authorization:** SUPER_ADMIN, STAFF

**Query Parameters:**
- `health_status` (optional): Filter by status (healthy, degraded, unhealthy)
- `tenant_id` (optional): Filter by tenant

**Response:**
```json
{
  "servers": [
    {
      "server_id": "server-001",
      "tenant_id": "tenant-123",
      "server_url": "https://dedicated-server-1.example.com",
      "health_status": "healthy",
      "response_time_ms": 150,
      "error_rate": 0.2,
      "last_check": "2026-04-09T10:30:00Z"
    }
  ]
}
```

### Get Server Details

Get detailed information about a specific server.

**Endpoint:** `GET /api/admin/servers/{server_id}`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "server_id": "server-001",
  "tenant_id": "tenant-123",
  "server_url": "https://dedicated-server-1.example.com",
  "health_status": "healthy",
  "current_metrics": {
    "response_time_ms": 150,
    "error_rate": 0.2,
    "uptime_percentage": 99.9
  },
  "last_check": "2026-04-09T10:30:00Z"
}
```

### Get Server Health History

Get historical health data for a server.

**Endpoint:** `GET /api/admin/servers/{server_id}/health/history`

**Authorization:** SUPER_ADMIN, STAFF

**Query Parameters:**
- `days` (optional): Number of days of history (default: 7, max: 30)

**Response:**
```json
{
  "server_id": "server-001",
  "history": [
    {
      "timestamp": "2026-04-09T10:00:00Z",
      "health_status": "healthy",
      "response_time_ms": 150,
      "status_code": 200
    }
  ]
}
```

---

## Alert Management APIs

### Get Alerts

Get list of alerts with filtering.

**Endpoint:** `GET /api/admin/alerts`

**Authorization:** SUPER_ADMIN, STAFF

**Query Parameters:**
- `status` (optional): active, acknowledged, resolved
- `severity` (optional): critical, warning, info
- `type` (optional): Alert type
- `tenant_id` (optional): Filter by tenant
- `limit` (optional): Number of results (default: 50)
- `offset` (optional): Pagination offset

**Response:**
```json
{
  "alerts": [
    {
      "alert_id": "alert-001",
      "tenant_id": "tenant-123",
      "alert_type": "rating_degraded",
      "severity": "critical",
      "title": "Company Rating Dropped",
      "message": "Rating dropped below 60%",
      "status": "active",
      "created_at": "2026-04-09T10:00:00Z",
      "context": {
        "previous_rating": 75.5,
        "current_rating": 58.2
      }
    }
  ],
  "total": 3
}
```

### Get Alert Details

Get detailed information about a specific alert.

**Endpoint:** `GET /api/admin/alerts/{alert_id}`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "alert_id": "alert-001",
  "tenant_id": "tenant-123",
  "alert_type": "rating_degraded",
  "severity": "critical",
  "title": "Company Rating Dropped",
  "message": "Rating dropped below 60%",
  "status": "active",
  "created_at": "2026-04-09T10:00:00Z",
  "acknowledged_at": null,
  "acknowledged_by": null,
  "resolved_at": null,
  "context": {
    "previous_rating": 75.5,
    "current_rating": 58.2
  }
}
```

### Acknowledge Alert

Mark an alert as acknowledged.

**Endpoint:** `PUT /api/admin/alerts/{alert_id}/acknowledge`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "alert_id": "alert-001",
  "status": "acknowledged",
  "acknowledged_at": "2026-04-09T10:35:00Z",
  "acknowledged_by": "admin-user"
}
```

### Resolve Alert

Mark an alert as resolved.

**Endpoint:** `PUT /api/admin/alerts/{alert_id}/resolve`

**Authorization:** SUPER_ADMIN

**Response:**
```json
{
  "alert_id": "alert-001",
  "status": "resolved",
  "resolved_at": "2026-04-09T11:00:00Z"
}
```

### Get Alert Configuration

Get alert threshold configuration.

**Endpoint:** `GET /api/admin/alerts/config`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "thresholds": {
    "response_time": {
      "warning": 200,
      "critical": 500
    },
    "error_rate": {
      "warning": 1.0,
      "critical": 5.0
    },
    "success_rate": {
      "warning": 95.0,
      "critical": 90.0
    }
  },
  "notification_channels": {
    "email": true,
    "sms": true,
    "in_app": true
  }
}
```

### Update Alert Configuration

Update alert threshold configuration.

**Endpoint:** `POST /api/admin/alerts/config`

**Authorization:** SUPER_ADMIN

**Request Body:**
```json
{
  "thresholds": {
    "response_time": {
      "warning": 200,
      "critical": 500
    },
    "error_rate": {
      "warning": 1.0,
      "critical": 5.0
    }
  },
  "notification_channels": {
    "email": true,
    "sms": false,
    "in_app": true
  }
}
```

**Response:**
```json
{
  "message": "Alert configuration updated successfully"
}
```

---

## Routing Configuration APIs

### Get Routing Configuration

Get routing configuration for a tenant.

**Endpoint:** `GET /api/admin/routing/{tenant_id}`

**Authorization:** SUPER_ADMIN, STAFF, TENANT_ADMIN (own data only)

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "routing_type": "dedicated",
  "dedicated_server_id": "server-001",
  "dedicated_server_url": "https://dedicated-server-1.example.com",
  "health_status": "healthy"
}
```

### Update Routing Configuration

Update routing configuration for a tenant.

**Endpoint:** `PUT /api/admin/routing/{tenant_id}`

**Authorization:** SUPER_ADMIN

**Request Body:**
```json
{
  "routing_type": "dedicated",
  "dedicated_server_url": "https://dedicated-server-1.example.com"
}
```

**Response:**
```json
{
  "message": "Routing configuration updated successfully",
  "tenant_id": "tenant-123",
  "routing_type": "dedicated"
}
```

### Get Routing Metrics

Get routing metrics for a tenant.

**Endpoint:** `GET /api/admin/routing/{tenant_id}/metrics`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "total_requests": 50000,
  "successful_requests": 49800,
  "failed_requests": 200,
  "average_latency_ms": 145.5
}
```

---

## Distribution Strategy APIs

### Get Distribution Configuration

Get distribution strategy configuration for a tenant.

**Endpoint:** `GET /api/admin/distribution/{tenant_id}`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "strategy_type": "round_robin",
  "strategy_config": {},
  "custom_rules": null
}
```

### Update Distribution Configuration

Update distribution strategy for a tenant.

**Endpoint:** `PUT /api/admin/distribution/{tenant_id}`

**Authorization:** SUPER_ADMIN

**Request Body:**
```json
{
  "strategy_type": "load_balancing",
  "strategy_config": {
    "max_load_per_device": 1000
  }
}
```

**Response:**
```json
{
  "message": "Distribution configuration updated successfully"
}
```

### Get Distribution Devices

Get list of devices for a tenant.

**Endpoint:** `GET /api/admin/distribution/{tenant_id}/devices`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "devices": [
    {
      "device_id": "device-001",
      "device_name": "Phone 1",
      "priority_level": 1,
      "status": "active",
      "current_load": 250,
      "total_messages_sent": 10000,
      "successful_messages": 9950,
      "failed_messages": 50
    }
  ]
}
```

### Get Distribution Metrics

Get distribution metrics for a tenant.

**Endpoint:** `GET /api/admin/distribution/{tenant_id}/metrics`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "total_messages_distributed": 50000,
  "devices_active": 5,
  "average_load_per_device": 10000,
  "distribution_fairness": 0.95
}
```

---

## Signal/Push System APIs

### Get Timing Configuration

Get Signal/Push timing configuration for a tenant.

**Endpoint:** `GET /api/admin/signalpush/{tenant_id}/config`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "signal_min_gap_ms": 5000,
  "push_min_gap_ms": 1000,
  "current_mode": "signal",
  "forced_mode": null,
  "violation_rate_threshold": 10.0,
  "z_score_threshold": 3.0,
  "peak_ratio_threshold": 2.0
}
```

### Update Timing Configuration

Update Signal/Push timing configuration.

**Endpoint:** `PUT /api/admin/signalpush/{tenant_id}/config`

**Authorization:** SUPER_ADMIN

**Request Body:**
```json
{
  "signal_min_gap_ms": 5000,
  "push_min_gap_ms": 1000,
  "violation_rate_threshold": 10.0
}
```

**Response:**
```json
{
  "message": "Timing configuration updated successfully"
}
```

### Get Timing Statistics

Get timing statistics for a tenant.

**Endpoint:** `GET /api/admin/signalpush/{tenant_id}/statistics`

**Authorization:** SUPER_ADMIN, STAFF

**Response:**
```json
{
  "tenant_id": "tenant-123",
  "message_type": "signal",
  "current_mpm": 120.5,
  "current_stddev": 15.2,
  "daily_avg_mpm": 115.0,
  "daily_peak_mpm": 150.0,
  "anomaly_score": 0.36,
  "anomaly_level": "none",
  "is_anomaly": false
}
```

### Get Timing Violations

Get timing violations for a tenant.

**Endpoint:** `GET /api/admin/signalpush/{tenant_id}/violations`

**Authorization:** SUPER_ADMIN, STAFF

**Query Parameters:**
- `limit` (optional): Number of results (default: 50)
- `severity` (optional): Filter by severity

**Response:**
```json
{
  "violations": [
    {
      "violation_id": "viol-001",
      "tenant_id": "tenant-123",
      "message_type": "signal",
      "expected_gap_ms": 5000,
      "actual_gap_ms": 2000,
      "violation_severity": "moderate",
      "timestamp": "2026-04-09T10:15:00Z"
    }
  ]
}
```

### Force Mode Change

Force Signal or Push mode for a tenant.

**Endpoint:** `POST /api/admin/signalpush/{tenant_id}/force-mode`

**Authorization:** SUPER_ADMIN

**Request Body:**
```json
{
  "mode": "push",
  "reason": "Manual intervention due to repeated violations"
}
```

**Response:**
```json
{
  "message": "Mode forced to push successfully",
  "tenant_id": "tenant-123",
  "previous_mode": "signal",
  "new_mode": "push"
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Missing or invalid token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 409 | Conflict - Resource already exists |
| 413 | Payload Too Large |
| 415 | Unsupported Media Type |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Invalid tenant ID format",
    "details": {
      "field": "tenant_id",
      "reason": "Must be alphanumeric"
    }
  }
}
```

---

## Rate Limiting

- **Default Limit**: 100 requests per minute per API key
- **Burst Limit**: 20 requests per second
- **Headers**:
  - `X-RateLimit-Limit`: Total requests allowed
  - `X-RateLimit-Remaining`: Remaining requests
  - `X-RateLimit-Reset`: Time when limit resets (Unix timestamp)

### Rate Limit Response

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again later.",
    "retry_after": 60
  }
}
```

---

## Best Practices

1. **Use pagination** for list endpoints to avoid large responses
2. **Cache responses** where appropriate to reduce API calls
3. **Handle rate limits** gracefully with exponential backoff
4. **Validate input** before sending requests
5. **Use appropriate HTTP methods** (GET for reads, PUT for updates)
6. **Include proper error handling** for all API calls
7. **Monitor API usage** to stay within rate limits

---

## Support

For API support, contact: api-support@example.com

**Documentation Version:** 1.0.0  
**Last Updated:** April 9, 2026
