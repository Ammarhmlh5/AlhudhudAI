# Dedicated Server Routing System

## Overview

The Dedicated Server Routing system provides intelligent message routing for companies with dedicated servers. It automatically detects company type (dedicated vs shared) and routes messages accordingly, with comprehensive health monitoring and metrics tracking.

## Features

- **Automatic Company Type Detection**: Detects within 100ms whether a company uses dedicated or shared infrastructure
- **Intelligent Routing**: Routes messages to dedicated servers when configured
- **Health-Aware Routing**: Only routes to healthy servers, returns errors for unhealthy servers
- **No Fallback Policy**: Maintains isolation by not falling back to shared infrastructure
- **Metrics Tracking**: Tracks request count, success rate, and average latency per server
- **Concurrency Safe**: Uses atomic operations for thread-safe metrics tracking

## Components

### 1. Routing Configuration Manager (`config.go`)

Manages routing configurations for tenants.

**Key Functions:**
- `GetRoutingConfig(tenantID)`: Retrieves routing configuration for a tenant
- `UpdateRoutingConfig(input)`: Updates routing configuration with validation
- `UpdateHealthStatus(tenantID, status)`: Updates server health status

**Example Usage:**
```go
configManager := routing.NewRoutingConfigManager(db)

// Get routing config
config, err := configManager.GetRoutingConfig(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}

// Update routing config
input := &routing.UpdateRoutingConfigInput{
    TenantID:           tenantID,
    RoutingType:        "dedicated",
    DedicatedServerID:  &serverID,
    DedicatedServerURL: &serverURL,
}
updatedConfig, err := configManager.UpdateRoutingConfig(ctx, input)
```

### 2. Routing Engine (`engine.go`)

Handles intelligent message routing decisions.

**Key Functions:**
- `RouteMessage(message)`: Makes routing decision for a message
- `GetRoutingDecision(tenantID)`: Gets routing decision without side effects
- `CheckServerHealth(tenantID)`: Checks if dedicated server is healthy
- `GetRoutingStats(tenantID)`: Returns routing statistics

**Example Usage:**
```go
engine := routing.NewRoutingEngine(db, configManager, metricsTracker)

// Route a message
message := &routing.Message{
    ID:        uuid.New(),
    TenantID:  tenantID,
    Content:   "Hello World",
    Recipient: "+1234567890",
}

decision, err := engine.RouteMessage(ctx, message)
if err != nil {
    if errors.Is(err, routing.ErrServerUnavailable) {
        // Handle unavailable server
        log.Printf("Server unavailable for tenant %s", tenantID)
    }
    return err
}

if decision.ShouldRoute {
    if decision.RoutingType == "dedicated" {
        // Route to dedicated server
        err = engine.SendToDedicatedServer(ctx, *decision.DedicatedServerURL, apiKey, message)
    } else {
        // Route to shared infrastructure
        err = engine.RouteToSharedInfrastructure(ctx, message)
    }
}
```

### 3. Metrics Tracker (`metrics.go`)

Tracks routing metrics with atomic operations for concurrency safety.

**Key Functions:**
- `TrackRoutingDecision(tenantID, success, latencyMs)`: Tracks a routing decision
- `GetMetrics(tenantID)`: Returns current metrics for a tenant
- `Start()`: Starts periodic metrics flushing
- `Stop()`: Stops metrics tracker and flushes remaining metrics

**Example Usage:**
```go
metricsTracker := routing.NewMetricsTracker(db)

// Start metrics tracker
metricsTracker.Start(ctx)
defer metricsTracker.Stop()

// Track routing decision
metricsTracker.TrackRoutingDecision(ctx, tenantID, true, 45) // success, 45ms latency

// Get metrics
metrics, err := metricsTracker.GetMetrics(ctx, tenantID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Total Requests: %d, Success Rate: %.2f%%\n", 
    metrics.TotalRequests, *metrics.SuccessRate)
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Routing Engine                          │
│                                                             │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │   Config Manager │  │  Metrics Tracker │               │
│  │                  │  │                  │               │
│  │  - Get Config    │  │  - Track Metrics │               │
│  │  - Update Config │  │  - Flush Metrics │               │
│  │  - Validate      │  │  - Get Stats     │               │
│  └──────────────────┘  └──────────────────┘               │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │           Routing Decision Logic                     │  │
│  │                                                       │  │
│  │  1. Get routing config (< 100ms)                    │  │
│  │  2. Check company type (dedicated vs shared)        │  │
│  │  3. Check server health                             │  │
│  │  4. Make routing decision                           │  │
│  │  5. Track metrics                                   │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                ┌───────────────────────┐
                │  PostgreSQL Database  │
                │                       │
                │  - routing_configs    │
                │  - server_health_logs │
                └───────────────────────┘
```

## Database Schema

### routing_configurations Table

```sql
CREATE TABLE routing_configurations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE,
    routing_type VARCHAR(20) NOT NULL DEFAULT 'shared',
    dedicated_server_id UUID REFERENCES dedicated_gateways(id),
    dedicated_server_url VARCHAR(500),
    dedicated_server_api_key VARCHAR(255),
    total_requests BIGINT NOT NULL DEFAULT 0,
    successful_requests BIGINT NOT NULL DEFAULT 0,
    failed_requests BIGINT NOT NULL DEFAULT 0,
    average_latency_ms INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_health_check TIMESTAMP,
    health_status VARCHAR(20),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## Requirements Validation

### Requirement 2.1: Dedicated Server Routing
✅ **Implemented**: `RouteMessage()` routes all messages to dedicated server when configured

### Requirement 2.2: Company Type Detection
✅ **Implemented**: `GetRoutingConfig()` detects company type within 100ms

### Requirement 2.3: Multiple Routing Paths
✅ **Implemented**: Each tenant has independent routing configuration

### Requirement 2.4: No Fallback Policy
✅ **Implemented**: Returns `ErrServerUnavailable` when server is unhealthy, no automatic fallback

### Requirement 2.8: Routing Metrics
✅ **Implemented**: `MetricsTracker` maintains request count, success rate, and average latency

## Integration with Health Monitor

The routing system integrates with the Server Health Monitor:

```go
// Health monitor updates routing config health status
healthChecker := health.NewHealthChecker(db, classifier, alertTrigger, logger)
healthChecker.Start(ctx)

// Routing engine checks health status before routing
decision, err := engine.RouteMessage(ctx, message)
if err == routing.ErrServerUnavailable {
    // Server is unhealthy, handle error
}
```

## Performance Considerations

1. **Fast Configuration Lookup**: Uses database indexes for sub-100ms lookups
2. **Atomic Metrics**: Uses `sync/atomic` for lock-free metrics updates
3. **Batched Writes**: Flushes metrics every 10 seconds to reduce database load
4. **Connection Pooling**: Reuses database connections for efficiency

## Error Handling

The system defines specific errors for different scenarios:

- `ErrServerUnavailable`: Dedicated server is unhealthy or unavailable
- `ErrNoRoutingConfig`: No routing configuration found for tenant
- `ErrInvalidRoutingType`: Invalid routing type specified

## Testing

Unit tests should cover:
- Configuration validation
- Routing decision logic
- Health status checking
- Metrics tracking accuracy
- Concurrent access safety

Property-based tests should verify:
- Routing isolation between tenants
- Metrics accuracy across all scenarios
- No fallback to shared infrastructure

## Future Enhancements

1. **Circuit Breaker**: Automatically disable routing after consecutive failures
2. **Load Balancing**: Support multiple dedicated servers per tenant
3. **Retry Logic**: Configurable retry policies for transient failures
4. **Caching**: Redis cache for routing configurations
5. **Monitoring**: Prometheus metrics export

## Related Documentation

- [Server Health Monitor](../health/README.md)
- [Admin Interface Design](.kiro/specs/admin-interface-enhancements/design.md)
- [Requirements Document](.kiro/specs/admin-interface-enhancements/requirements.md)
