# API Key Manager - Quick Start Guide

## Overview

The API Key Manager provides comprehensive functionality for managing API keys for dedicated gateways. It implements all requirements from المتطلب 12 (Requirements 12.1-12.11).

## Features Implemented

### ✅ Task 21.1: Create API Key (Requirements 12.1, 12.2, 12.3)
- Generate cryptographically secure API keys
- Hash keys using bcrypt (cost 12)
- Store key metadata (name, description, permissions)
- Support custom rate limits
- Support IP whitelisting
- Support expiration dates

### ✅ Task 21.2: Set Permissions (Requirements 12.4, 12.5)
- Custom permissions per API key
- Rate limit configuration (requests per minute)
- Default permissions based on use case

### ✅ Task 21.3: IP Whitelisting (Requirements 12.6, 12.10)
- Add/remove IP addresses
- Support IPv4, IPv6, and CIDR notation
- Validate IP addresses on API key usage
- Log unauthorized access attempts

### ✅ Task 21.4: Expiry Management (Requirements 12.7, 12.11)
- Set expiration dates
- Remove expiration (set to nil)
- Validate expiry on API key usage
- Reject expired keys with appropriate error

### ✅ Task 21.5: Validation Middleware (Requirements 12.8, 12.10, 12.11)
- Validate API keys against bcrypt hash
- Check permissions
- Check IP whitelist
- Check expiry date
- Check active status
- Update last_used_at and usage_count

## File Structure

```
services/backend/internal/gateway/
├── api_key_manager.go          # Main implementation
├── api_key_manager_test.go     # Comprehensive unit tests
└── types.go                     # Type definitions
```

## Usage Examples

### 1. Create API Key Manager

```go
import (
    "github.com/al-hudhud/backend/internal/gateway"
    "gorm.io/gorm"
)

// Initialize manager
manager := gateway.NewAPIKeyManager(db)
```

### 2. Create an API Key

```go
req := &gateway.CreateAPIKeyRequest{
    TenantID:    tenantID,
    GatewayID:   gatewayID,
    KeyName:     "Production API Key",
    Description: "API key for production environment",
    RateLimit:   120, // 120 requests per minute
    AllowedIPs:  []string{"192.168.1.0/24", "10.0.0.1"},
    ExpiresAt:   &expiryDate, // Optional
    Permissions: map[string]interface{}{
        "send_sms":     true,
        "view_stats":   true,
        "view_reports": true,
    },
}

resp, err := manager.CreateAPIKey(ctx, req)
if err != nil {
    log.Fatal(err)
}

// IMPORTANT: Save the API key - it's only shown once!
fmt.Println("API Key:", resp.APIKey)
fmt.Println("Key ID:", resp.KeyInfo.ID)
```

### 3. Validate API Key

```go
req := &gateway.ValidateAPIKeyRequest{
    APIKey:    "gw_abc123...",
    ClientIP:  "192.168.1.100",
    GatewayID: gatewayID,
}

resp, err := manager.ValidateAPIKey(ctx, req)
if err != nil {
    log.Fatal(err)
}

if !resp.IsValid {
    fmt.Println("Validation failed:", resp.Error)
    return
}

// API key is valid - proceed with request
fmt.Println("Tenant ID:", resp.TenantID)
fmt.Println("Gateway ID:", resp.GatewayID)
fmt.Println("Permissions:", resp.Permissions)
fmt.Println("Rate Limit:", resp.RateLimit)
```

### 4. Update Permissions

```go
req := &gateway.SetPermissionsRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    KeyID:     keyID,
    Permissions: map[string]interface{}{
        "send_sms":        true,
        "view_stats":      true,
        "view_reports":    true,
        "manage_gateway":  false,
        "manage_users":    false,
        "manage_api_keys": false,
    },
}

apiKey, err := manager.SetPermissions(ctx, req)
```

### 5. Manage IP Whitelist

```go
// Add IP
addReq := &gateway.AddAllowedIPRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    KeyID:     keyID,
    IPAddress: "192.168.1.100",
}
apiKey, err := manager.AddAllowedIP(ctx, addReq)

// Remove IP
removeReq := &gateway.RemoveAllowedIPRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    KeyID:     keyID,
    IPAddress: "192.168.1.100",
}
apiKey, err := manager.RemoveAllowedIP(ctx, removeReq)
```

### 6. Set Expiry Date

```go
expiresAt := time.Now().Add(90 * 24 * time.Hour) // 90 days

req := &gateway.SetExpiryRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    KeyID:     keyID,
    ExpiresAt: &expiresAt,
}

apiKey, err := manager.SetExpiry(ctx, req)
```

### 7. Update API Key Status

```go
// Deactivate
req := &gateway.UpdateAPIKeyStatusRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
    KeyID:     keyID,
    IsActive:  false,
}
err := manager.UpdateAPIKeyStatus(ctx, req)

// Reactivate
req.IsActive = true
err = manager.UpdateAPIKeyStatus(ctx, req)
```

### 8. Get All API Keys for Gateway

```go
req := &gateway.GetAPIKeysByGatewayRequest{
    TenantID:  tenantID,
    GatewayID: gatewayID,
}

apiKeys, err := manager.GetAPIKeysByGateway(ctx, req)
for _, key := range apiKeys {
    fmt.Printf("Key: %s, Active: %v, Expires: %v\n", 
        key.KeyName, key.IsActive, key.ExpiresAt)
}
```

## Security Features

### 1. Bcrypt Hashing
- API keys are hashed using bcrypt with cost 12
- Original keys are never stored
- Keys are only shown once during creation

### 2. IP Whitelisting
- Support for IPv4, IPv6, and CIDR notation
- Validation on every API key usage
- Unauthorized access attempts are logged

### 3. Expiration
- Optional expiration dates
- Automatic validation on usage
- Clear error messages for expired keys

### 4. Tenant Isolation
- All operations include tenant_id validation
- Row Level Security (RLS) at database level
- Cross-tenant access prevention

### 5. Usage Tracking
- last_used_at timestamp updated on each use
- usage_count incremented automatically
- Helps identify unused or compromised keys

## Default Permissions

When creating an API key without specifying permissions, the following defaults are applied:

```go
{
    "send_sms":        true,
    "view_stats":      true,
    "view_reports":    true,
    "manage_gateway":  false,
    "manage_users":    false,
    "manage_api_keys": false,
}
```

## Validation Errors

The ValidateAPIKey function returns specific error messages:

- `"Invalid API key"` - Key not found or hash doesn't match
- `"API key is not active"` - Key has been deactivated
- `"API key has expired"` - Key expiration date has passed
- `"IP address not whitelisted"` - Client IP not in allowed list
- `"Invalid client IP address"` - Malformed IP address

## Running Tests

The test suite includes comprehensive unit tests for all functionality.

### Prerequisites

Tests require CGO enabled for SQLite:

```bash
# Windows
$env:CGO_ENABLED=1
go test ./internal/gateway/api_key_manager_test.go ./internal/gateway/api_key_manager.go ./internal/gateway/types.go -v

# Linux/Mac
CGO_ENABLED=1 go test ./internal/gateway/api_key_manager_test.go ./internal/gateway/api_key_manager.go ./internal/gateway/types.go -v
```

### Test Coverage

- ✅ Create API key with default settings
- ✅ Create API key with custom settings
- ✅ Duplicate key name validation
- ✅ Invalid tenant/gateway validation
- ✅ Invalid IP address validation
- ✅ Set permissions
- ✅ Invalid permission validation
- ✅ Set rate limit
- ✅ Add/remove IP addresses (IPv4, IPv6, CIDR)
- ✅ Set/remove expiry dates
- ✅ Validate API key (all scenarios)
- ✅ Update API key status
- ✅ Get API keys by gateway
- ✅ Delete API key
- ✅ Helper function tests

## Database Schema

The `gateway_api_keys` table was created in migration `024_gateway_api_keys_table.sql` with:

- Tenant isolation (tenant_id)
- Bcrypt hash storage (api_key_hash)
- JSONB permissions
- Rate limiting
- IP whitelisting (text array)
- Expiration support
- Usage tracking
- Row Level Security (RLS)

## Integration with Middleware

To use API key validation in HTTP middleware:

```go
func APIKeyAuthMiddleware(manager *gateway.APIKeyManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.JSON(401, gin.H{"error": "API key required"})
            c.Abort()
            return
        }

        clientIP := c.ClientIP()
        gatewayID := c.GetInt("gateway_id") // From route or context

        req := &gateway.ValidateAPIKeyRequest{
            APIKey:    apiKey,
            ClientIP:  clientIP,
            GatewayID: gatewayID,
        }

        resp, err := manager.ValidateAPIKey(c.Request.Context(), req)
        if err != nil {
            c.JSON(500, gin.H{"error": "Validation failed"})
            c.Abort()
            return
        }

        if !resp.IsValid {
            c.JSON(401, gin.H{"error": resp.Error})
            c.Abort()
            return
        }

        // Set context values for downstream handlers
        c.Set("tenant_id", resp.TenantID)
        c.Set("gateway_id", resp.GatewayID)
        c.Set("permissions", resp.Permissions)
        c.Set("rate_limit", resp.RateLimit)

        c.Next()
    }
}
```

## Best Practices

1. **Key Generation**: Always use the built-in `CreateAPIKey` function - never generate keys manually
2. **Key Storage**: Save the API key immediately after creation - it's only shown once
3. **Key Rotation**: Implement regular key rotation (e.g., every 90 days)
4. **IP Whitelisting**: Always use IP whitelisting for production keys
5. **Permissions**: Follow principle of least privilege - only grant necessary permissions
6. **Monitoring**: Monitor usage_count and last_used_at to detect anomalies
7. **Expiration**: Set expiration dates for temporary or testing keys
8. **Deactivation**: Deactivate keys instead of deleting them to maintain audit trail

## Requirements Mapping

| Requirement | Implementation | Status |
|-------------|----------------|--------|
| 12.1 | Multiple API keys per gateway | ✅ |
| 12.2 | Generate unique encrypted key | ✅ |
| 12.3 | Store name and description | ✅ |
| 12.4 | Custom permissions per key | ✅ |
| 12.5 | Rate limit per key | ✅ |
| 12.6 | IP whitelisting | ✅ |
| 12.7 | Expiration date support | ✅ |
| 12.8 | Track last usage | ✅ |
| 12.9 | Activate/deactivate keys | ✅ |
| 12.10 | Reject unauthorized IPs | ✅ |
| 12.11 | Reject expired keys | ✅ |

## Next Steps

1. Integrate with HTTP API handlers
2. Add rate limiting enforcement
3. Implement security logging
4. Add metrics and monitoring
5. Create admin UI for key management

## Support

For questions or issues, refer to:
- Main documentation: `services/backend/internal/gateway/GATEWAY_MANAGER_README.md`
- Migration details: `services/backend/migrations/README_GATEWAY_API_KEYS.md`
- Database schema: `services/backend/migrations/024_gateway_api_keys_table.sql`
