# Resource Limiter Package

## Overview

The `resources` package provides quota checking functionality for dedicated gateways in the tariff pricing system. It implements Requirements 9.1, 9.2, and 9.3 from the specification.

## Features

- **CPU Quota Checking**: Monitor and enforce CPU usage limits per gateway
- **Memory Quota Checking**: Monitor and enforce RAM usage limits per gateway
- **Bandwidth Quota Checking**: Monitor and enforce bandwidth usage limits per gateway
- **Flexible Enforcement**: Support for both "soft" (warning only) and "hard" (reject requests) enforcement modes
- **Comprehensive Monitoring**: Track current usage against configured quotas

## Requirements Covered

- **Requirement 9.1**: THE النظام SHALL يدعم تحديد حصة CPU لكل بوابة مخصصة
- **Requirement 9.2**: THE النظام SHALL يدعم تحديد حصة RAM لكل بوابة مخصصة
- **Requirement 9.3**: THE النظام SHALL يدعم تحديد حصة Bandwidth لكل بوابة مخصصة

## Usage

### Creating a Resource Limiter

```go
import (
    "github.com/al-hudhud/backend/internal/resources"
    "github.com/google/uuid"
)

// Create a new resource limiter for a gateway
gatewayID := 1
tenantID := uuid.MustParse("your-tenant-id")
limiter := resources.NewResourceLimiter(db, gatewayID, tenantID)
```

### Checking Individual Quotas

#### Check CPU Quota

```go
ctx := context.Background()
result, err := limiter.CheckCPUQuota(ctx)
if err != nil {
    log.Printf("Error checking CPU quota: %v", err)
    return err
}

if result.Exceeded {
    log.Printf("CPU quota exceeded: %.2f%% usage", result.UsagePercent)
    
    if result.EnforcementMode == "hard" {
        return errors.New("CPU quota exceeded - request rejected")
    } else {
        log.Println("Warning: CPU quota exceeded (soft limit)")
    }
}
```

#### Check Memory Quota

```go
result, err := limiter.CheckMemoryQuota(ctx)
if err != nil {
    log.Printf("Error checking memory quota: %v", err)
    return err
}

if result.Exceeded {
    log.Printf("Memory quota exceeded: %d MB / %d MB (%.2f%%)",
        result.CurrentUsage, result.Quota, result.UsagePercent)
}
```

#### Check Bandwidth Quota

```go
result, err := limiter.CheckBandwidthQuota(ctx)
if err != nil {
    log.Printf("Error checking bandwidth quota: %v", err)
    return err
}

if result.Exceeded {
    log.Printf("Bandwidth quota exceeded: %.2f Mbps / %d Mbps",
        result.CurrentUsage, result.Quota)
}
```

### Checking All Quotas

```go
results, err := limiter.CheckAllQuotas(ctx)
if err != nil {
    log.Printf("Error checking quotas: %v", err)
    return err
}

for resourceType, result := range results {
    if result.Exceeded {
        log.Printf("%s quota exceeded: %.2f%% usage",
            resourceType, result.UsagePercent)
    }
}
```

### Quick Check for Any Exceeded Quota

```go
exceeded, resources, err := limiter.IsQuotaExceeded(ctx)
if err != nil {
    log.Printf("Error checking quotas: %v", err)
    return err
}

if exceeded {
    log.Printf("Quotas exceeded for resources: %v", resources)
    // Take appropriate action
}
```

## Data Structures

### ResourceQuota

Represents the configured resource quotas for a gateway:

```go
type ResourceQuota struct {
    ID                      int
    GatewayID               int
    CPUCores                *float64  // vCPU cores (e.g., 4.0)
    MemoryMB                *int      // Memory in MB (e.g., 8192)
    BandwidthMbps           *int      // Bandwidth in Mbps (e.g., 200)
    // ... other quota fields
    IsActive                bool
    EnforcementMode         string    // 'soft' or 'hard'
}
```

### ResourceUsage

Represents the current resource usage for a gateway:

```go
type ResourceUsage struct {
    ID                        int64
    GatewayID                 int
    TenantID                  uuid.UUID
    Timestamp                 time.Time
    CPUUsagePercent           *float64  // CPU usage percentage (0-100)
    MemoryUsageMB             *int      // Memory usage in MB
    BandwidthUsageMbps        *float64  // Bandwidth usage in Mbps
    // ... other usage fields
}
```

### QuotaCheckResult

Represents the result of a quota check:

```go
type QuotaCheckResult struct {
    Exceeded        bool          // Whether the quota is exceeded
    CurrentUsage    interface{}   // Current usage value
    Quota           interface{}   // Quota limit value
    UsagePercent    float64       // Usage as percentage of quota
    ResourceType    string        // "CPU", "Memory", or "Bandwidth"
    EnforcementMode string        // "soft" or "hard"
}
```

## Enforcement Modes

### Soft Limit

- Quota exceeded triggers a warning
- Request is allowed to proceed
- Useful for monitoring and alerting

```go
if result.Exceeded && result.EnforcementMode == "soft" {
    log.Println("Warning: Quota exceeded (soft limit)")
    // Continue processing
}
```

### Hard Limit

- Quota exceeded rejects the request
- Request is not allowed to proceed
- Useful for strict resource control

```go
if result.Exceeded && result.EnforcementMode == "hard" {
    return errors.New("Quota exceeded - request rejected")
}
```

## Database Tables

### resource_quotas

Stores the configured quotas for each gateway:

```sql
CREATE TABLE resource_quotas (
    id SERIAL PRIMARY KEY,
    gateway_id INTEGER NOT NULL,
    cpu_cores DECIMAL(4, 2),
    memory_mb INTEGER,
    bandwidth_mbps INTEGER,
    -- ... other quota fields
    is_active BOOLEAN DEFAULT true,
    enforcement_mode VARCHAR(20) DEFAULT 'soft'
);
```

### resource_usage

Stores the current usage metrics for each gateway:

```sql
CREATE TABLE resource_usage (
    id BIGSERIAL,
    gateway_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    cpu_usage_percent DECIMAL(5, 2),
    memory_usage_mb INTEGER,
    bandwidth_usage_mbps DECIMAL(10, 2),
    -- ... other usage fields
) PARTITION BY RANGE (timestamp);
```

## Error Handling

The package returns descriptive errors for various failure scenarios:

```go
result, err := limiter.CheckCPUQuota(ctx)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "no active resource quota found"):
        // No quota configured for this gateway
        log.Println("No quota configured - allowing request")
        
    case strings.Contains(err.Error(), "no resource usage found"):
        // No usage data available
        log.Println("No usage data - assuming zero usage")
        
    default:
        // Other database or system errors
        log.Printf("Error checking quota: %v", err)
        return err
    }
}
```

## Testing

The package includes comprehensive unit tests using sqlmock:

```bash
# Run tests
go test -v ./internal/resources/

# Run tests with coverage
go test -v -cover ./internal/resources/
```

Test coverage includes:
- Creating resource limiters
- Retrieving quotas and usage data
- Checking individual quotas (CPU, Memory, Bandwidth)
- Checking all quotas together
- Handling missing quotas (NULL values)
- Handling exceeded quotas
- Both soft and hard enforcement modes

## Integration Example

Here's a complete example of integrating quota checking into a request handler:

```go
func (h *GatewayHandler) ProcessRequest(c *gin.Context) {
    gatewayID := c.GetInt("gateway_id")
    tenantID := c.GetString("tenant_id")
    
    // Create resource limiter
    limiter := resources.NewResourceLimiter(
        h.db,
        gatewayID,
        uuid.MustParse(tenantID),
    )
    
    // Check all quotas
    exceeded, exceededResources, err := limiter.IsQuotaExceeded(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to check quotas"})
        return
    }
    
    if exceeded {
        // Get detailed results for logging
        results, _ := limiter.CheckAllQuotas(c.Request.Context())
        
        // Check enforcement mode
        hardLimitExceeded := false
        for _, resource := range exceededResources {
            if results[resource].EnforcementMode == "hard" {
                hardLimitExceeded = true
                break
            }
        }
        
        if hardLimitExceeded {
            c.JSON(429, gin.H{
                "error": "Resource quota exceeded",
                "resources": exceededResources,
            })
            return
        } else {
            // Log warning but continue
            log.Printf("Warning: Quotas exceeded for gateway %d: %v",
                gatewayID, exceededResources)
        }
    }
    
    // Process request normally
    // ...
}
```

## Future Enhancements

Potential future enhancements to this package:

1. **Real-time Usage Tracking**: Integrate with monitoring systems for real-time usage data
2. **Predictive Alerts**: Alert before quotas are exceeded based on usage trends
3. **Auto-scaling**: Automatically adjust quotas based on usage patterns
4. **Historical Analysis**: Analyze usage patterns over time
5. **Custom Thresholds**: Allow custom warning thresholds (e.g., alert at 80% usage)

## Related Packages

- `internal/gateway`: Gateway management functionality
- `internal/monitoring`: Resource usage monitoring and collection
- `internal/notifications`: Alert notifications for quota violations
