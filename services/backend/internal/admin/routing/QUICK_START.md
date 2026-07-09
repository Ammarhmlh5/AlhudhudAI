# Dedicated Server Routing - Quick Start Guide

## Overview

This guide helps you quickly get started with the Dedicated Server Routing system.

## Installation

The routing system is part of the admin interface enhancements. Ensure you have:

1. PostgreSQL database with migrations applied
2. Go 1.21 or higher
3. Required dependencies installed

## Quick Setup

### 1. Initialize Components

```go
package main

import (
    "context"
    "log"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    
    "your-project/internal/admin/routing"
)

func main() {
    // Connect to database
    db, err := sqlx.Connect("postgres", "your-connection-string")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Initialize routing components
    configManager := routing.NewRoutingConfigManager(db)
    metricsTracker := routing.NewMetricsTracker(db)
    engine := routing.NewRoutingEngine(db, configManager, metricsTracker)
    
    // Start metrics tracker
    ctx := context.Background()
    if err := metricsTracker.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer metricsTracker.Stop()
    
    // Your application code here
}
```

### 2. Configure Dedicated Server for a Tenant

```go
import (
    "github.com/google/uuid"
    "your-project/internal/admin/routing"
)

func configureDedicatedServer(ctx context.Context, configManager *routing.RoutingConfigManager) {
    tenantID := uuid.MustParse("your-tenant-id")
    serverID := uuid.MustParse("your-server-id")
    serverURL := "https://dedicated-server.example.com"
    apiKey := "your-api-key"
    
    input := &routing.UpdateRoutingConfigInput{
        TenantID:             tenantID,
        RoutingType:          "dedicated",
        DedicatedServerID:    &serverID,
        DedicatedServerURL:   &serverURL,
        DedicatedServerAPIKey: &apiKey,
    }
    
    config, err := configManager.UpdateRoutingConfig(ctx, input)
    if err != nil {
        log.Printf("Failed to configure dedicated server: %v", err)
        return
    }
    
    log.Printf("Configured dedicated server for tenant %s", config.TenantID)
}
```

### 3. Route a Message

```go
func routeMessage(ctx context.Context, engine *routing.RoutingEngine) {
    message := &routing.Message{
        ID:        uuid.New(),
        TenantID:  uuid.MustParse("your-tenant-id"),
        Content:   "Hello World",
        Recipient: "+1234567890",
    }
    
    decision, err := engine.RouteMessage(ctx, message)
    if err != nil {
        if errors.Is(err, routing.ErrServerUnavailable) {
            log.Printf("Dedicated server unavailable for tenant %s", message.TenantID)
            // Handle error - do NOT fallback to shared infrastructure
            return
        }
        log.Printf("Routing error: %v", err)
        return
    }
    
    if decision.ShouldRoute {
        if decision.RoutingType == "dedicated" {
            log.Printf("Routing to dedicated server: %s", *decision.DedicatedServerURL)
            // Send to dedicated server
        } else {
            log.Printf("Routing to shared infrastructure")
            // Send to shared infrastructure
        }
    }
}
```

### 4. Check Routing Statistics

```go
func checkStats(ctx context.Context, engine *routing.RoutingEngine) {
    tenantID := uuid.MustParse("your-tenant-id")
    
    stats, err := engine.GetRoutingStats(ctx, tenantID)
    if err != nil {
        log.Printf("Failed to get stats: %v", err)
        return
    }
    
    log.Printf("Routing Stats for %s:", tenantID)
    log.Printf("  Total Requests: %d", stats.TotalRequests)
    log.Printf("  Successful: %d", stats.SuccessfulRequests)
    log.Printf("  Failed: %d", stats.FailedRequests)
    if stats.SuccessRate != nil {
        log.Printf("  Success Rate: %.2f%%", *stats.SuccessRate)
    }
    if stats.AverageLatencyMs != nil {
        log.Printf("  Avg Latency: %dms", *stats.AverageLatencyMs)
    }
}
```

## Common Use Cases

### Use Case 1: Check if Tenant Has Dedicated Server

```go
func hasDedicatedServer(ctx context.Context, configManager *routing.RoutingConfigManager, tenantID uuid.UUID) bool {
    config, err := configManager.GetRoutingConfig(ctx, tenantID)
    if err != nil {
        return false
    }
    return config.HasDedicatedServer()
}
```

### Use Case 2: Update Server Health Status

```go
func updateHealthStatus(ctx context.Context, configManager *routing.RoutingConfigManager, tenantID uuid.UUID, status string) {
    err := configManager.UpdateHealthStatus(ctx, tenantID, status, time.Now())
    if err != nil {
        log.Printf("Failed to update health status: %v", err)
    }
}
```

### Use Case 3: Get Real-Time Metrics

```go
func getRealTimeMetrics(ctx context.Context, metricsTracker *routing.MetricsTracker, tenantID uuid.UUID) {
    metrics, err := metricsTracker.GetMetrics(ctx, tenantID)
    if err != nil {
        log.Printf("Failed to get metrics: %v", err)
        return
    }
    
    log.Printf("Real-time metrics:")
    log.Printf("  Total: %d, Success: %d, Failed: %d", 
        metrics.TotalRequests, 
        metrics.SuccessfulRequests, 
        metrics.FailedRequests)
}
```

## Integration with Health Monitor

The routing system works seamlessly with the Server Health Monitor:

```go
import (
    "your-project/internal/admin/health"
    "your-project/internal/admin/routing"
)

func setupHealthMonitoring(ctx context.Context, db *sqlx.DB) {
    // Setup routing
    configManager := routing.NewRoutingConfigManager(db)
    metricsTracker := routing.NewMetricsTracker(db)
    engine := routing.NewRoutingEngine(db, configManager, metricsTracker)
    
    // Setup health monitoring
    classifier := health.NewStatusClassifier()
    alertTrigger := health.NewAlertTrigger(db)
    logger := health.NewHistoryLogger(db)
    healthChecker := health.NewHealthChecker(db, classifier, alertTrigger, logger)
    
    // Start both systems
    metricsTracker.Start(ctx)
    healthChecker.Start(ctx)
    
    // Health monitor will update routing config health status automatically
}
```

## Error Handling

Always handle routing errors appropriately:

```go
decision, err := engine.RouteMessage(ctx, message)
if err != nil {
    switch {
    case errors.Is(err, routing.ErrServerUnavailable):
        // Dedicated server is down - do NOT fallback
        log.Printf("Server unavailable, message cannot be sent")
        return fmt.Errorf("dedicated server unavailable")
        
    case errors.Is(err, routing.ErrInvalidRoutingType):
        // Configuration error
        log.Printf("Invalid routing configuration")
        return fmt.Errorf("invalid routing config")
        
    default:
        // Other errors
        log.Printf("Routing error: %v", err)
        return err
    }
}
```

## Performance Tips

1. **Reuse Components**: Create routing components once and reuse them
2. **Batch Operations**: Use metrics tracker's automatic batching
3. **Monitor Metrics**: Regularly check routing statistics
4. **Health Checks**: Ensure health monitor is running for accurate routing decisions

## Testing

### Unit Test Example

```go
func TestRouteMessage(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    // Create components
    configManager := routing.NewRoutingConfigManager(db)
    metricsTracker := routing.NewMetricsTracker(db)
    engine := routing.NewRoutingEngine(db, configManager, metricsTracker)
    
    ctx := context.Background()
    tenantID := uuid.New()
    
    // Configure dedicated server
    serverID := uuid.New()
    serverURL := "https://test-server.example.com"
    input := &routing.UpdateRoutingConfigInput{
        TenantID:           tenantID,
        RoutingType:        "dedicated",
        DedicatedServerID:  &serverID,
        DedicatedServerURL: &serverURL,
    }
    _, err := configManager.UpdateRoutingConfig(ctx, input)
    require.NoError(t, err)
    
    // Test routing
    message := &routing.Message{
        ID:       uuid.New(),
        TenantID: tenantID,
        Content:  "Test",
    }
    
    decision, err := engine.RouteMessage(ctx, message)
    require.NoError(t, err)
    assert.True(t, decision.ShouldRoute)
    assert.Equal(t, "dedicated", decision.RoutingType)
}
```

## Troubleshooting

### Problem: Routing always fails

**Solution**: Check server health status:
```go
healthy, err := engine.CheckServerHealth(ctx, tenantID)
if !healthy {
    log.Printf("Server is unhealthy, check health monitor logs")
}
```

### Problem: Metrics not updating

**Solution**: Ensure metrics tracker is started:
```go
metricsTracker.Start(ctx)
defer metricsTracker.Stop()
```

### Problem: Configuration validation fails

**Solution**: Verify all required fields:
- For dedicated routing: `dedicated_server_id` and `dedicated_server_url` are required
- Routing type must be "shared" or "dedicated"

## Next Steps

- Read the [full documentation](README.md)
- Review [requirements](.kiro/specs/admin-interface-enhancements/requirements.md)
- Check [design document](.kiro/specs/admin-interface-enhancements/design.md)
- Explore [health monitoring](../health/README.md)

## Support

For issues or questions:
1. Check the README.md for detailed documentation
2. Review the requirements and design documents
3. Examine the test files for usage examples
