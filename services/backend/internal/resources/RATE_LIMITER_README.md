# Rate Limiter per Gateway - Implementation Guide

## Overview

This document describes the implementation of the **TenantRateLimiter** for dedicated gateways using the **Token Bucket Algorithm**. This implementation fulfills task 18.2 of the Tariff and Pricing System.

## Requirements

- **Requirement 9.8**: THE النظام SHALL يدعم تحديد حد API Rate Limit لكل بوابة
- **Requirement 27.10**: THE النظام SHALL يحسّن الاستعلامات باستخدام Indexes مناسبة

## Token Bucket Algorithm

The Token Bucket Algorithm is a rate limiting algorithm that allows for burst traffic while maintaining average rate limits. It works as follows:

1. **Bucket**: A conceptual bucket that holds tokens
2. **Capacity**: Maximum number of tokens the bucket can hold (burst size)
3. **Refill Rate**: Rate at which tokens are added to the bucket
4. **Consumption**: Each request consumes one token

### Advantages

- **Burst Handling**: Allows temporary bursts of traffic up to the bucket capacity
- **Smooth Rate Limiting**: Refills tokens at a steady rate
- **Fair Distribution**: Prevents starvation of legitimate requests
- **Memory Efficient**: Uses minimal memory per limiter

## Implementation

### TenantRateLimiter Structure

```go
type TenantRateLimiter struct {
    tenantID  uuid.UUID
    gatewayID int
    limiters  map[string]*rate.Limiter
    mu        sync.RWMutex
}
```

### Supported Rate Limit Types

1. **API Rate Limit**: Limits API requests per minute
   - Refill Rate: `APIRatePerMinute / 60` tokens per second
   - Burst: `APIRatePerMinute` tokens

2. **Messages Rate Limit**: Limits messages per hour
   - Refill Rate: `MessagesPerHour / 3600` tokens per second
   - Burst: `MessagesPerHour / 10` tokens (10% burst)

3. **Queries Rate Limit**: Limits database queries per second
   - Refill Rate: `QueriesPerSecond` tokens per second
   - Burst: `QueriesPerSecond * 2` tokens (2x burst)

## Usage

### Creating a Rate Limiter

```go
// Method 1: Create with explicit configuration
config := &RateLimitConfig{
    APIRatePerMinute: 300,
    MessagesPerHour:  50000,
    QueriesPerSecond: 200,
}
limiter := NewTenantRateLimiter(tenantID, gatewayID, config)

// Method 2: Create from database configuration
resourceLimiter := NewResourceLimiter(db, gatewayID, tenantID)
rateLimiter, err := resourceLimiter.CreateTenantRateLimiter(ctx)
```

### Checking Rate Limits

```go
// Immediate check (returns error if limit exceeded)
err := limiter.AllowAPI(ctx)
if err != nil {
    // Rate limit exceeded
    return err
}

// Wait for token (blocks until token available or context cancelled)
err := limiter.AllowAPIWithWait(ctx)
if err != nil {
    // Context cancelled or other error
    return err
}
```

### Checking Different Resource Types

```go
// Check API rate limit
err := limiter.AllowAPI(ctx)

// Check message rate limit
err := limiter.AllowMessage(ctx)

// Check query rate limit
err := limiter.AllowQuery(ctx)
```

### Getting Rate Limit Status

```go
status := limiter.GetRateLimitStatus()
// Returns:
// {
//   "api": {
//     "limit": 5.0,           // tokens per second
//     "burst": 300,           // max tokens
//     "available": 250.5      // current tokens
//   },
//   "messages": { ... },
//   "queries": { ... }
// }
```

### Updating Rate Limits Dynamically

```go
newConfig := &RateLimitConfig{
    APIRatePerMinute: 600,
    MessagesPerHour:  100000,
    QueriesPerSecond: 400,
}
limiter.UpdateRateLimits(newConfig)
```

## Integration with Resource Quotas

The rate limiter integrates with the `resource_quotas` table:

```sql
SELECT 
    api_rate_limit_per_minute,
    max_messages_per_hour,
    max_queries_per_second
FROM resource_quotas
WHERE gateway_id = $1 AND is_active = true
```

If database values are NULL, default values are used:
- API Rate: 300 requests/minute
- Messages: 50,000 messages/hour
- Queries: 200 queries/second

## Isolation Between Gateways

Each gateway has its own independent rate limiter:

```go
// Gateway 1
limiter1 := NewTenantRateLimiter(tenant1, gateway1, config)

// Gateway 2
limiter2 := NewTenantRateLimiter(tenant2, gateway2, config)

// Rate limits are completely independent
limiter1.AllowAPI(ctx) // Does not affect limiter2
```

## Thread Safety

The implementation is thread-safe:

- Uses `sync.RWMutex` for concurrent access
- Multiple goroutines can safely call rate limit methods
- Read operations use `RLock()` for better performance
- Write operations (UpdateRateLimits) use `Lock()`

## Testing

Comprehensive tests cover:

1. **Basic Functionality**
   - Creating rate limiters
   - Allowing requests within limits
   - Rejecting requests exceeding limits

2. **Token Bucket Behavior**
   - Burst capability
   - Token refill over time
   - Independent limiters for different resource types

3. **Wait Functionality**
   - Blocking until token available
   - Context cancellation handling

4. **Database Integration**
   - Loading configuration from database
   - Using default values when NULL
   - Creating rate limiter from database

5. **Concurrent Access**
   - Multiple goroutines accessing rate limiter
   - Thread safety verification

6. **Dynamic Updates**
   - Updating rate limits at runtime
   - Verifying new limits take effect

## Performance Considerations

1. **Memory Usage**: Each rate limiter uses minimal memory (~100 bytes per limiter)
2. **CPU Usage**: Token bucket operations are O(1) time complexity
3. **Lock Contention**: Read locks allow concurrent reads, minimizing contention
4. **Token Precision**: Uses float64 for sub-second precision

## Error Handling

Rate limit errors include tenant and gateway information:

```go
"API rate limit exceeded for tenant 123e4567-e89b-12d3-a456-426614174000 gateway 1"
"message rate limit exceeded for tenant 123e4567-e89b-12d3-a456-426614174000 gateway 1"
"query rate limit exceeded for tenant 123e4567-e89b-12d3-a456-426614174000 gateway 1"
```

## Best Practices

1. **Use Wait Methods for User-Facing APIs**: Provides better UX by waiting instead of immediate rejection
2. **Use Immediate Methods for Background Jobs**: Fail fast for background processing
3. **Monitor Rate Limit Status**: Regularly check status to detect approaching limits
4. **Set Appropriate Burst Sizes**: Balance between allowing bursts and preventing abuse
5. **Update Limits Dynamically**: Adjust limits based on gateway performance and usage patterns

## Example: Complete Integration

```go
// Initialize
db := getDatabase()
tenantID := getTenantID()
gatewayID := getGatewayID()

// Create resource limiter
resourceLimiter := NewResourceLimiter(db, gatewayID, tenantID)

// Create rate limiter from database configuration
rateLimiter, err := resourceLimiter.CreateTenantRateLimiter(ctx)
if err != nil {
    log.Fatalf("Failed to create rate limiter: %v", err)
}

// Use in API handler
func handleAPIRequest(w http.ResponseWriter, r *http.Request) {
    // Check rate limit
    if err := rateLimiter.AllowAPI(r.Context()); err != nil {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }
    
    // Process request
    processRequest(w, r)
}

// Use in message sender
func sendMessage(ctx context.Context, message *Message) error {
    // Check rate limit
    if err := rateLimiter.AllowMessage(ctx); err != nil {
        return fmt.Errorf("message rate limit exceeded: %w", err)
    }
    
    // Send message
    return sendToGateway(message)
}

// Use in database query
func executeQuery(ctx context.Context, query string) error {
    // Check rate limit
    if err := rateLimiter.AllowQuery(ctx); err != nil {
        return fmt.Errorf("query rate limit exceeded: %w", err)
    }
    
    // Execute query
    return db.Exec(ctx, query)
}
```

## Monitoring and Alerting

Monitor rate limiter status to detect issues:

```go
// Periodic monitoring
ticker := time.NewTicker(1 * time.Minute)
go func() {
    for range ticker.C {
        status := rateLimiter.GetRateLimitStatus()
        
        for resourceType, info := range status {
            infoMap := info.(map[string]interface{})
            available := infoMap["available"].(float64)
            burst := infoMap["burst"].(int)
            
            // Alert if less than 10% tokens available
            if available < float64(burst)*0.1 {
                log.Warnf("Low tokens for %s: %.2f/%d", resourceType, available, burst)
            }
        }
    }
}()
```

## Related Files

- `limiter.go`: Main implementation
- `limiter_test.go`: Comprehensive tests
- `README.md`: Resource quotas documentation
- `../gateway/manager.go`: Gateway manager integration

## References

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
- Task 18.2: تطوير Rate Limiter per Gateway
- Requirements 9.8, 27.10
