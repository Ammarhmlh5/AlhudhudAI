# API Reference - Tariff & Pricing System

## Overview

This document provides comprehensive API documentation for the Tariff & Pricing System. All APIs use REST principles with JSON request/response format.

## Base URL

```
Production: https://api.yourdomain.com/v1
Staging: https://staging-api.yourdomain.com/v1
Development: http://localhost:8080/v1
```

## Authentication

All API requests require authentication using one of the following methods:

### JWT Token (for web/mobile apps)
```http
Authorization: Bearer <jwt_token>
```

### API Key (for dedicated gateways)
```http
X-API-Key: <api_key>
```

## Common Response Format

### Success Response
```json
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": { ... }
  }
}
```

## HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict (e.g., duplicate)
- `422 Unprocessable Entity` - Validation error
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

---

## Tariff Management APIs

### 1. List Tariffs

Get all active tariffs for the authenticated tenant.

**Endpoint:** `GET /tariffs`

**Query Parameters:**
- `service_id` (optional) - Filter by service ID
- `country_code` (optional) - Filter by country code
- `network_operator` (optional) - Filter by network operator
- `page` (optional, default: 1) - Page number
- `limit` (optional, default: 20) - Items per page

**Example Request:**
```bash
curl -X GET "https://api.yourdomain.com/v1/tariffs?service_id=1&country_code=YE" \
  -H "Authorization: Bearer <token>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "tariffs": [
      {
        "id": 1,
        "service_id": 1,
        "service_name": "Hudud Gateway",
        "country_code": "YE",
        "network_operator": "Yemen Mobile",
        "cost_per_message": 9.0000,
        "currency": "YER",
        "effective_from": "2024-01-01",
        "effective_to": null,
        "is_active": true
      },
      {
        "id": 2,
        "service_id": 1,
        "service_name": "Hudud Gateway",
        "country_code": "YE",
        "network_operator": "Sabafon",
        "cost_per_message": 13.0000,
        "currency": "YER",
        "effective_from": "2024-01-01",
        "effective_to": null,
        "is_active": true
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 2,
      "total_pages": 1
    }
  }
}
```

### 2. Create Tariff (Admin Only)

Create a new tariff.

**Endpoint:** `POST /tariffs`

**Request Body:**
```json
{
  "service_id": 1,
  "country_code": "YE",
  "network_operator": "Yemen Mobile",
  "message_type": "standard",
  "cost_per_message": 9.0000,
  "currency": "YER",
  "effective_from": "2024-01-01",
  "effective_to": null
}
```

**Example Request:**
```bash
curl -X POST "https://api.yourdomain.com/v1/tariffs" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": 1,
    "country_code": "YE",
    "network_operator": "Yemen Mobile",
    "cost_per_message": 9.0000,
    "currency": "YER",
    "effective_from": "2024-01-01"
  }'
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "id": 3,
    "service_id": 1,
    "country_code": "YE",
    "network_operator": "Yemen Mobile",
    "cost_per_message": 9.0000,
    "currency": "YER",
    "effective_from": "2024-01-01",
    "is_active": true,
    "created_at": "2024-01-15T10:30:00Z"
  },
  "message": "Tariff created successfully"
}
```

### 3. Update Tariff (Admin Only)

Update an existing tariff.

**Endpoint:** `PUT /tariffs/:id`

**Request Body:**
```json
{
  "cost_per_message": 10.0000,
  "effective_from": "2024-02-01"
}
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "id": 3,
    "cost_per_message": 10.0000,
    "effective_from": "2024-02-01",
    "updated_at": "2024-01-15T10:35:00Z"
  },
  "message": "Tariff updated successfully"
}
```

### 4. Calculate Message Cost

Calculate the cost of sending a message.

**Endpoint:** `POST /tariffs/calculate`

**Request Body:**
```json
{
  "service_id": 1,
  "country_code": "YE",
  "network_operator": "Yemen Mobile",
  "message_text": "Hello, this is a test message",
  "encoding": "GSM-7"
}
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "segments": 1,
    "cost_per_segment": 9.0000,
    "total_cost": 9.0000,
    "currency": "YER",
    "encoding": "GSM-7",
    "message_length": 29
  }
}
```

---

## Package Management APIs

### 5. List Packages

Get all available packages.

**Endpoint:** `GET /packages`

**Query Parameters:**
- `service_id` (optional) - Filter by service ID
- `is_active` (optional) - Filter by active status
- `page` (optional, default: 1)
- `limit` (optional, default: 20)

**Example Request:**
```bash
curl -X GET "https://api.yourdomain.com/v1/packages?service_id=2" \
  -H "Authorization: Bearer <token>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "packages": [
      {
        "id": 1,
        "name": "Basic",
        "service_id": 2,
        "price": 10.00,
        "currency": "USD",
        "included_messages": 1000,
        "billing_cycle": "monthly",
        "rollover_enabled": true,
        "rollover_limit": 500,
        "rollover_expiry_months": 1,
        "overage_rate": 0.01,
        "is_active": true,
        "features": [
          "1000 messages per month",
          "Rollover up to 500 messages",
          "Email support"
        ]
      },
      {
        "id": 2,
        "name": "Standard",
        "service_id": 2,
        "price": 20.00,
        "currency": "USD",
        "included_messages": 3000,
        "billing_cycle": "monthly",
        "rollover_enabled": true,
        "rollover_limit": 1500,
        "rollover_expiry_months": 2,
        "overage_rate": 0.0067,
        "is_active": true,
        "features": [
          "3000 messages per month",
          "Rollover up to 1500 messages",
          "Priority email support",
          "API access"
        ]
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 5,
      "total_pages": 1
    }
  }
}
```

### 6. Create Package (Admin Only)

Create a new package.

**Endpoint:** `POST /packages`

**Request Body:**
```json
{
  "name": "Premium",
  "service_id": 2,
  "price": 50.00,
  "currency": "USD",
  "included_messages": 10000,
  "billing_cycle": "monthly",
  "rollover_enabled": true,
  "rollover_limit": 5000,
  "rollover_expiry_months": 3,
  "overage_rate": 0.005,
  "features": [
    "10000 messages per month",
    "Rollover up to 5000 messages",
    "24/7 phone support",
    "Dedicated account manager"
  ]
}
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "id": 6,
    "name": "Premium",
    "price": 50.00,
    "included_messages": 10000,
    "created_at": "2024-01-15T11:00:00Z"
  },
  "message": "Package created successfully"
}
```

---

## Subscription Management APIs

### 7. Subscribe to Package

Subscribe to a monthly package.

**Endpoint:** `POST /subscriptions`

**Request Body:**
```json
{
  "package_id": 1
}
```

**Example Request:**
```bash
curl -X POST "https://api.yourdomain.com/v1/subscriptions" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"package_id": 1}'
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "subscription_id": 123,
    "package_id": 1,
    "package_name": "Basic",
    "status": "active",
    "start_date": "2024-01-15",
    "end_date": "2024-02-15",
    "messages_included": 1000,
    "messages_remaining": 1000,
    "messages_used": 0,
    "auto_renewal": true,
    "amount_paid": 10.00,
    "currency": "USD"
  },
  "message": "Subscription activated successfully"
}
```

### 8. Get Subscription Usage

Get current subscription usage details.

**Endpoint:** `GET /subscriptions/usage`

**Example Request:**
```bash
curl -X GET "https://api.yourdomain.com/v1/subscriptions/usage" \
  -H "Authorization: Bearer <token>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "subscription_id": 123,
    "package_name": "Basic",
    "status": "active",
    "billing_cycle_start": "2024-01-15",
    "billing_cycle_end": "2024-02-15",
    "days_remaining": 25,
    "messages": {
      "included": 1000,
      "used": 450,
      "remaining": 550,
      "rollover": 200,
      "rollover_expiry": "2024-02-28"
    },
    "overage": {
      "messages_used": 0,
      "cost": 0.00
    },
    "next_renewal": {
      "date": "2024-02-15",
      "amount": 10.00,
      "auto_renewal": true
    }
  }
}
```

### 9. Cancel Subscription

Cancel an active subscription.

**Endpoint:** `DELETE /subscriptions/:id`

**Example Request:**
```bash
curl -X DELETE "https://api.yourdomain.com/v1/subscriptions/123" \
  -H "Authorization: Bearer <token>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "subscription_id": 123,
    "status": "cancelled",
    "cancelled_at": "2024-01-15T12:00:00Z",
    "refund_amount": 0.00
  },
  "message": "Subscription cancelled successfully"
}
```

---

## Dedicated Gateway APIs

### 10. Create Dedicated Gateway (Admin Only)

Create a new dedicated gateway for an enterprise client.

**Endpoint:** `POST /gateways`

**Request Body:**
```json
{
  "name": "Acme Corp Gateway",
  "code": "ACME_GW_01",
  "tenant_id": "uuid-here",
  "telecom_provider": "Yemen Mobile",
  "telecom_username": "acme_user",
  "telecom_password": "encrypted_password",
  "subscription_type": "annual",
  "subscription_fee": 1200.00,
  "server_rental_fee": 500.00,
  "balance_alert_threshold": 10000.00,
  "resource_quotas": {
    "cpu_cores": 4,
    "memory_gb": 8,
    "bandwidth_mbps": 100,
    "max_messages_per_hour": 10000,
    "max_messages_per_day": 100000
  }
}
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "gateway_id": 456,
    "name": "Acme Corp Gateway",
    "code": "ACME_GW_01",
    "status": "active",
    "created_at": "2024-01-15T13:00:00Z"
  },
  "message": "Dedicated gateway created successfully"
}
```

### 11. Get Gateway Statistics

Get statistics for a dedicated gateway.

**Endpoint:** `GET /gateways/:id/statistics`

**Query Parameters:**
- `start_date` (required) - Start date (YYYY-MM-DD)
- `end_date` (required) - End date (YYYY-MM-DD)

**Example Request:**
```bash
curl -X GET "https://api.yourdomain.com/v1/gateways/456/statistics?start_date=2024-01-01&end_date=2024-01-31" \
  -H "X-API-Key: <api_key>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "gateway_id": 456,
    "period": {
      "start_date": "2024-01-01",
      "end_date": "2024-01-31"
    },
    "messages": {
      "total_sent": 85000,
      "delivered": 82500,
      "failed": 2500,
      "pending": 0,
      "delivery_rate": 97.06
    },
    "performance": {
      "avg_response_time_ms": 250,
      "max_response_time_ms": 1200,
      "uptime_percentage": 99.95
    },
    "resources": {
      "avg_cpu_usage": 45.5,
      "avg_memory_usage": 62.3,
      "peak_bandwidth_mbps": 85.2
    }
  }
}
```

### 12. Create API Key for Gateway

Create a new API key for gateway access.

**Endpoint:** `POST /gateways/:id/api-keys`

**Request Body:**
```json
{
  "name": "Production API Key",
  "description": "Main API key for production environment",
  "permissions": ["send_message", "query_status", "view_reports"],
  "rate_limit": 1000,
  "allowed_ips": ["203.0.113.10", "203.0.113.11"],
  "expires_at": "2025-01-15T00:00:00Z"
}
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "api_key_id": 789,
    "api_key": "gw_live_1234567890abcdef",
    "name": "Production API Key",
    "permissions": ["send_message", "query_status", "view_reports"],
    "rate_limit": 1000,
    "created_at": "2024-01-15T14:00:00Z",
    "expires_at": "2025-01-15T00:00:00Z"
  },
  "message": "API key created successfully. Please save this key securely as it won't be shown again."
}
```

---

## Balance & Billing APIs

### 13. Get Balance

Get current balance for the authenticated tenant.

**Endpoint:** `GET /balance`

**Example Request:**
```bash
curl -X GET "https://api.yourdomain.com/v1/balance" \
  -H "Authorization: Bearer <token>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "balance": 1250.50,
    "currency": "USD",
    "last_topup": {
      "amount": 500.00,
      "date": "2024-01-10T10:00:00Z"
    },
    "low_balance_threshold": 100.00,
    "is_low_balance": false
  }
}
```

### 14. Top Up Balance (Admin Only)

Add balance to a tenant account.

**Endpoint:** `POST /balance/topup`

**Request Body:**
```json
{
  "tenant_id": "uuid-here",
  "amount": 500.00,
  "currency": "USD",
  "reason": "Manual top-up via admin panel",
  "payment_method": "bank_transfer",
  "reference_number": "TXN123456"
}
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "transaction_id": 9876,
    "tenant_id": "uuid-here",
    "amount": 500.00,
    "currency": "USD",
    "previous_balance": 1250.50,
    "new_balance": 1750.50,
    "created_at": "2024-01-15T15:00:00Z"
  },
  "message": "Balance topped up successfully"
}
```

### 15. Get Transaction History

Get transaction history for the authenticated tenant.

**Endpoint:** `GET /transactions`

**Query Parameters:**
- `type` (optional) - Filter by transaction type (subscription, topup, deduction, refund)
- `start_date` (optional) - Start date (YYYY-MM-DD)
- `end_date` (optional) - End date (YYYY-MM-DD)
- `page` (optional, default: 1)
- `limit` (optional, default: 20)

**Example Request:**
```bash
curl -X GET "https://api.yourdomain.com/v1/transactions?type=subscription&page=1&limit=10" \
  -H "Authorization: Bearer <token>"
```

**Example Response:**
```json
{
  "success": true,
  "data": {
    "transactions": [
      {
        "id": 1001,
        "type": "subscription",
        "amount": -10.00,
        "currency": "USD",
        "description": "Monthly subscription - Basic Package",
        "balance_before": 1260.50,
        "balance_after": 1250.50,
        "status": "completed",
        "created_at": "2024-01-15T00:00:00Z"
      },
      {
        "id": 1000,
        "type": "topup",
        "amount": 500.00,
        "currency": "USD",
        "description": "Balance top-up",
        "balance_before": 760.50,
        "balance_after": 1260.50,
        "status": "completed",
        "created_at": "2024-01-10T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 45,
      "total_pages": 5
    }
  }
}
```

### 16. Get Invoices

Get billing invoices for the authenticated tenant.

**Endpoint:** `GET /invoices`

**Query Parameters:**
- `status` (optional) - Filter by status (paid, pending, overdue)
- `start_date` (optional) - Start date (YYYY-MM-DD)
- `end_date` (optional) - End date (YYYY-MM-DD)
- `page` (optional, default: 1)
- `limit` (optional, default: 20)

**Example Response:**
```json
{
  "success": true,
  "data": {
    "invoices": [
      {
        "id": 5001,
        "invoice_number": "INV-2024-001",
        "amount": 10.00,
        "tax": 1.50,
        "total": 11.50,
        "currency": "USD",
        "status": "paid",
        "issue_date": "2024-01-15",
        "due_date": "2024-01-20",
        "paid_at": "2024-01-15T00:05:00Z",
        "description": "Monthly subscription - Basic Package",
        "pdf_url": "/invoices/5001/download"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 12,
      "total_pages": 1
    }
  }
}
```

---

## Rate Limiting

All APIs are subject to rate limiting to ensure fair usage:

- **Standard Users**: 100 requests per minute
- **Premium Users**: 500 requests per minute
- **Enterprise Gateways**: Custom limits based on subscription

Rate limit headers are included in all responses:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642252800
```

---

## Webhooks

The system supports webhooks for real-time notifications:

### Supported Events

- `subscription.renewed` - Subscription renewed successfully
- `subscription.failed` - Subscription renewal failed
- `subscription.cancelled` - Subscription cancelled
- `balance.low` - Balance below threshold
- `balance.topup` - Balance topped up
- `gateway.alert` - Gateway resource alert
- `gateway.down` - Gateway is down
- `rollover.activated` - Rollover messages activated
- `rollover.forfeited` - Rollover messages forfeited

### Webhook Payload Example

```json
{
  "event": "subscription.renewed",
  "timestamp": "2024-01-15T00:00:00Z",
  "data": {
    "subscription_id": 123,
    "package_name": "Basic",
    "amount": 10.00,
    "next_renewal_date": "2024-02-15"
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `INVALID_REQUEST` | Request validation failed |
| `UNAUTHORIZED` | Authentication required |
| `FORBIDDEN` | Insufficient permissions |
| `NOT_FOUND` | Resource not found |
| `INSUFFICIENT_BALANCE` | Insufficient balance for operation |
| `PACKAGE_NOT_ACTIVE` | Package is not active |
| `SUBSCRIPTION_EXISTS` | Active subscription already exists |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `GATEWAY_UNAVAILABLE` | Gateway is unavailable |
| `QUOTA_EXCEEDED` | Resource quota exceeded |
| `INTERNAL_ERROR` | Internal server error |

---

## Postman Collection

Download the complete Postman collection for easy API testing:

[Download Postman Collection](./postman/tariff-system-collection.json)

### Quick Start with Postman

1. Import the collection into Postman
2. Set up environment variables:
   - `base_url`: API base URL
   - `jwt_token`: Your JWT token
   - `api_key`: Your API key (for gateway APIs)
3. Run the requests

---

## SDK Support

Official SDKs are available for:

- **JavaScript/TypeScript**: `npm install @yourdomain/tariff-sdk`
- **Python**: `pip install yourdomain-tariff-sdk`
- **PHP**: `composer require yourdomain/tariff-sdk`
- **Go**: `go get github.com/yourdomain/tariff-sdk-go`

### JavaScript Example

```javascript
import { TariffClient } from '@yourdomain/tariff-sdk';

const client = new TariffClient({
  apiKey: 'your-api-key',
  baseUrl: 'https://api.yourdomain.com/v1'
});

// Subscribe to a package
const subscription = await client.subscriptions.create({
  packageId: 1
});

// Get usage
const usage = await client.subscriptions.getUsage();
console.log(`Messages remaining: ${usage.messages.remaining}`);
```

---

## Support

For API support, contact:
- Email: api-support@yourdomain.com
- Documentation: https://docs.yourdomain.com
- Status Page: https://status.yourdomain.com
