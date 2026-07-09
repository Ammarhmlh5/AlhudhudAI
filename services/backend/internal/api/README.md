# Package Management APIs

This document describes the Package Management and Subscription APIs for the Tariff and Pricing System.

## Overview

The APIs provide functionality for:
- Creating and managing subscription packages (Admin only)
- Listing available packages
- Subscribing to packages
- Viewing subscription usage and rollover messages
- Managing subscription auto-renewal

## Authentication & Authorization

All APIs require authentication via JWT or API Key. The middleware extracts `tenant_id` and `user_role` from the authentication token.

### Tenant Isolation

All APIs enforce strict tenant isolation using Row Level Security (RLS). Each request automatically filters data by the authenticated tenant's ID.

## Package Management APIs

### 1. Create Package

Creates a new subscription package.

**Endpoint:** `POST /api/v1/packages`

**Authorization:** Admin only

**Requirements:** 3.6, 18.6

**Request Body:**
```json
{
  "service_id": 2,
  "name": "Professional Package",
  "code": "PROF_PKG",
  "price_usd": "40.00",
  "included_messages": 8000,
  "overage_rate": "0.005",
  "is_unlimited": false,
  "billing_cycle": "monthly",
  "rollover_enabled": true,
  "rollover_limit": 2000,
  "rollover_expiry_months": 3,
  "display_order": 3
}
```

**Response (Success):**
```json
{
  "success": true,
  "package_id": 123,
  "message": "Package 'Professional Package' created successfully"
}
```

**Response (Forbidden):**
```json
{
  "success": false,
  "error": "Forbidden: Admin access required"
}
```

**Validation Rules:**
- `price_usd` must be greater than 0
- Unlimited packages (`is_unlimited: true`) must have `included_messages: null`
- Limited packages must have `included_messages > 0`
- Limited packages must have `overage_rate > 0`
- `billing_cycle` must be "monthly" or "yearly"
- If `rollover_enabled: true`, `rollover_limit` and `rollover_expiry_months` must be > 0 if specified

### 2. List Packages

Lists all packages with filtering and pagination.

**Endpoint:** `GET /api/v1/packages`

**Authorization:** Any authenticated user

**Requirements:** 18.5, 21.1

**Query Parameters:**
- `service_id` (optional): Filter by service ID
- `is_active` (optional): Filter by active status (true/false)
- `page` (optional, default: 1): Page number
- `page_size` (optional, default: 20, max: 100): Items per page

**Example Request:**
```
GET /api/v1/packages?service_id=2&is_active=true&page=1&page_size=10
```

**Response:**
```json
{
  "success": true,
  "packages": [
    {
      "id": 1,
      "service_id": 2,
      "name": "Basic Package",
      "code": "BASIC",
      "price_usd": "10.00",
      "included_messages": 1000,
      "overage_rate": "0.01",
      "is_unlimited": false,
      "billing_cycle": "monthly",
      "rollover_enabled": true,
      "rollover_limit": 500,
      "rollover_expiry_months": 2,
      "is_active": true,
      "display_order": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total_count": 5,
  "page": 1,
  "page_size": 10
}
```

### 3. Get Package

Retrieves a single package by ID.

**Endpoint:** `GET /api/v1/packages/:id`

**Authorization:** Any authenticated user

**Response:**
```json
{
  "success": true,
  "package": {
    "id": 1,
    "service_id": 2,
    "name": "Basic Package",
    "code": "BASIC",
    "price_usd": "10.00",
    "included_messages": 1000,
    "overage_rate": "0.01",
    "is_unlimited": false,
    "billing_cycle": "monthly",
    "rollover_enabled": true,
    "rollover_limit": 500,
    "rollover_expiry_months": 2,
    "is_active": true,
    "display_order": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**Response (Not Found):**
```json
{
  "success": false,
  "error": "Package not found"
}
```

## Subscription Management APIs

### 4. Subscribe to Package

Subscribes the authenticated tenant to a package.

**Endpoint:** `POST /api/v1/subscriptions`

**Authorization:** Any authenticated user

**Requirements:** 4.1, 4.2, 21.3

**Request Body:**
```json
{
  "package_id": 1,
  "auto_renew": true
}
```

**Response (Success):**
```json
{
  "success": true,
  "subscription_id": 456,
  "message": "Successfully subscribed to Basic Package",
  "old_balance": "100.00",
  "new_balance": "90.00"
}
```

**Response (Insufficient Balance):**
```json
{
  "success": false,
  "message": "Insufficient balance",
  "old_balance": "5.00",
  "new_balance": "5.00"
}
```

**Response (Inactive Package):**
```json
{
  "success": false,
  "message": "Package is not active"
}
```

**Business Logic:**
1. Validates package exists and is active
2. Checks tenant has sufficient balance
3. Atomically deducts package price from balance
4. Creates subscription record
5. Records transaction in accounting_transactions
6. Returns subscription ID and updated balance

### 5. Get Usage

Returns subscription usage information including rollover messages.

**Endpoint:** `GET /api/v1/subscriptions/usage`

**Authorization:** Any authenticated user

**Requirements:** 4.6, 21.5, 21.6, 21.9

**Response:**
```json
{
  "success": true,
  "usage": {
    "subscription_id": 456,
    "package_id": 1,
    "package_name": "Basic Package",
    "messages_used": 250,
    "messages_remaining": 750,
    "included_messages": 1000,
    "overage_messages": 0,
    "overage_cost": "0.00",
    "overage_rate": "0.01",
    "status": "active",
    "start_date": "2024-01-01",
    "end_date": "2024-02-01",
    "next_renewal_date": "2024-02-01"
  },
  "rollover_messages": [
    {
      "id": 789,
      "messages_count": 150,
      "source_month": "2023-12",
      "expiry_date": "2024-03-01",
      "created_at": "2024-01-01 00:00:00"
    }
  ]
}
```

**Response (No Active Subscription):**
```json
{
  "success": false,
  "error": "No active subscription found"
}
```

**Notes:**
- `messages_remaining` is `null` for unlimited packages
- `rollover_messages` array shows all active rollover messages with expiry dates
- Rollover messages are ordered by creation date (FIFO)

### 6. Get Active Subscription

Returns the active subscription for the authenticated tenant.

**Endpoint:** `GET /api/v1/subscriptions/active`

**Authorization:** Any authenticated user

**Response:**
```json
{
  "success": true,
  "subscription": {
    "id": 456,
    "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
    "package_id": 1,
    "start_date": "2024-01-01T00:00:00Z",
    "end_date": "2024-02-01T00:00:00Z",
    "messages_used": 250,
    "messages_remaining": 750,
    "overage_messages": 0,
    "overage_cost": "0.00",
    "status": "active",
    "auto_renew": true,
    "grace_period_days": 3,
    "next_renewal_date": "2024-02-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 7. Cancel Subscription

Cancels auto-renewal for a subscription.

**Endpoint:** `PUT /api/v1/subscriptions/:id/cancel`

**Authorization:** Any authenticated user

**Response:**
```json
{
  "success": true,
  "message": "Auto-renewal cancelled successfully"
}
```

**Notes:**
- Only disables auto-renewal; subscription remains active until end_date
- Tenant can still use remaining messages
- Subscription will not be automatically renewed at end_date

### 8. Enable Auto-Renewal

Enables auto-renewal for a subscription.

**Endpoint:** `PUT /api/v1/subscriptions/:id/enable-auto-renew`

**Authorization:** Any authenticated user

**Response:**
```json
{
  "success": true,
  "message": "Auto-renewal enabled successfully"
}
```

## Error Responses

All APIs return consistent error responses:

```json
{
  "success": false,
  "error": "Error message describing what went wrong"
}
```

**Common HTTP Status Codes:**
- `200 OK`: Successful GET request
- `201 Created`: Successful POST request (resource created)
- `400 Bad Request`: Invalid request parameters or validation error
- `401 Unauthorized`: Missing or invalid authentication
- `402 Payment Required`: Insufficient balance
- `403 Forbidden`: Insufficient permissions (e.g., non-admin trying to create package)
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Integration with Existing Managers

The APIs integrate with existing managers:

### Subscription Manager
- `Subscribe()`: Creates subscription and deducts balance atomically
- `GetUsage()`: Returns usage information
- `GetActiveSubscription()`: Returns active subscription
- `DeductMessages()`: Deducts messages from quota (used by message sending)
- `CalculateOverageCost()`: Calculates and charges overage cost

### Rollover Manager
- `GetRolloverMessages()`: Returns rollover messages for display
- `UseRolloverMessages()`: Uses rollover messages (FIFO)
- `RolloverMessages()`: Creates rollover at end of month
- `CleanupExpiredRollover()`: Deletes expired rollover messages

## Testing

Unit tests are provided in:
- `package_handler_test.go`: Tests for package management APIs
- `subscription_handler_test.go`: Tests for subscription APIs

Run tests:
```bash
# Run all tests
go test ./internal/api/...

# Run with coverage
go test -cover ./internal/api/...

# Run integration tests (requires database)
go test -v ./internal/api/...

# Skip integration tests
go test -short ./internal/api/...
```

## Security Considerations

1. **Tenant Isolation**: All queries use RLS to ensure data isolation
2. **Admin-Only Operations**: Package creation requires admin role
3. **Atomic Operations**: Balance deductions use SERIALIZABLE transactions
4. **Input Validation**: All inputs are validated before processing
5. **Error Handling**: Errors don't leak sensitive information

## Future Enhancements

Potential future additions:
- Package update API (admin only)
- Package deactivation API (admin only)
- Subscription history API
- Usage analytics API
- Bulk package operations
- Package templates
