# Message Distribution System

## Overview

The Message Distribution System provides intelligent message distribution across multiple phone devices for companies with multi-device setups. It supports multiple distribution strategies and includes automatic failure handling.

## Features

- **Multiple Distribution Strategies**:
  - Round Robin: Sequential device selection with fair distribution
  - Load Balancing: Distribution based on current device load
  - Priority-based: Distribution based on device priority levels
  - Custom Rules: Administrator-defined distribution rules

- **Device Failure Handling**: Automatic exclusion of failed devices for 5 minutes
- **Thread-Safe Operations**: All strategies use mutexes for concurrent access
- **Flexible Configuration**: Per-tenant strategy configuration stored in database
- **Real-time Metrics**: Track distribution statistics per device

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   DistributionManager                        │
│  (Main entry point for message distribution)                │
└────────────┬────────────────────────────────────────────────┘
             │
             ├──► StrategySelector (Factory pattern)
             │    ├──► RoundRobinStrategy
             │    ├──► LoadBalancingStrategy
             │    ├──► PriorityStrategy
             │    └──► CustomRulesStrategy
             │
             ├──► DevicePoolManager (Database operations)
             │    ├──► GetAvailableDevices()
             │    ├──► UpdateDeviceLoad()
             │    └──► UpdateDeviceMetrics()
             │
             └──► DeviceFailureHandler (Failure management)
                  ├──► OnDeviceFailure()
                  ├──► IsDeviceExcluded()
                  └──► FilterAvailableDevices()
```

## Quick Start

### 1. Initialize Distribution Manager

```go
import (
    "github.com/yourusername/yourproject/internal/admin/distribution"
    "gorm.io/gorm"
)

// Create distribution manager
db := // your gorm.DB instance
manager := distribution.NewDistributionManager(db)
```

### 2. Configure Distribution Strategy

```go
ctx := context.Background()
tenantID := uuid.MustParse("your-tenant-id")

// Configure Round Robin strategy
err := manager.GetSelector().UpdateStrategyConfig(
    ctx,
    tenantID,
    distribution.StrategyRoundRobin,
    nil, // no additional config needed
    nil, // no custom rules
)
```

### 3. Select Device for Message

```go
// Select a device for message distribution
device, err := manager.SelectDeviceForMessage(ctx, tenantID)
if err != nil {
    log.Printf("Failed to select device: %v", err)
    return
}

log.Printf("Selected device: %s (ID: %s)", device.DeviceName, device.ID)
```

### 4. Update Device Load

```go
// After assigning message to device
err = manager.GetPoolManager().IncrementDeviceLoad(ctx, device.ID)
if err != nil {
    log.Printf("Failed to increment device load: %v", err)
}

// After message is processed
err = manager.GetPoolManager().DecrementDeviceLoad(ctx, device.ID)
if err != nil {
    log.Printf("Failed to decrement device load: %v", err)
}
```

### 5. Handle Device Failure

```go
// Mark device as failed (automatically excluded for 5 minutes)
err = manager.GetPoolManager().MarkDeviceAsFailed(ctx, device.ID)
if err != nil {
    log.Printf("Failed to mark device as failed: %v", err)
}
```

## Distribution Strategies

### Round Robin Strategy

Distributes messages sequentially across available devices. Ensures fair distribution within 5% variance over 1000 messages.

```go
// Configure Round Robin
err := selector.UpdateStrategyConfig(
    ctx,
    tenantID,
    distribution.StrategyRoundRobin,
    nil,
    nil,
)
```

**Use Case**: Equal distribution across all devices, suitable for devices with similar capabilities.

### Load Balancing Strategy

Selects the device with the lowest current load.

```go
// Configure Load Balancing
err := selector.UpdateStrategyConfig(
    ctx,
    tenantID,
    distribution.StrategyLoadBalancing,
    nil,
    nil,
)
```

**Use Case**: Optimal for varying device capabilities or when devices have different processing speeds.

### Priority-Based Strategy

Selects devices based on priority level (1-5, where 5 is highest). Among devices with equal priority, selects the one with lowest load.

```go
// Configure Priority-based
err := selector.UpdateStrategyConfig(
    ctx,
    tenantID,
    distribution.StrategyPriority,
    nil,
    nil,
)
```

**Use Case**: When certain devices should be preferred (e.g., newer devices, devices with better connectivity).

### Custom Rules Strategy

Supports administrator-defined rules for complex distribution logic.

#### Device Name Pattern

```go
customRules := map[string]interface{}{
    "type": "device_name_pattern",
    "pattern": "Premium-",
}

err := selector.UpdateStrategyConfig(
    ctx,
    tenantID,
    distribution.StrategyCustom,
    nil,
    customRules,
)
```

#### Priority Threshold

```go
customRules := map[string]interface{}{
    "type": "priority_threshold",
    "min_priority": 3.0,
}
```

#### Load Threshold

```go
customRules := map[string]interface{}{
    "type": "load_threshold",
    "max_load": 5.0,
}
```

#### Weighted Selection

```go
customRules := map[string]interface{}{
    "type": "weighted_selection",
    "weights": map[string]interface{}{
        "priority": 0.7,
        "load": 0.3,
    },
}
```

## Device Failure Handling

The system automatically handles device failures:

1. **Failure Detection**: When a device fails to process a message
2. **Automatic Exclusion**: Device is excluded from distribution for 5 minutes
3. **Auto-Reinclude**: Device is automatically re-included after exclusion period
4. **Manual Reinclude**: Administrators can manually re-include devices

```go
failureHandler := manager.GetFailureHandler()

// Check if device is excluded
isExcluded := failureHandler.IsDeviceExcluded(deviceID)

// Manually re-include device
failureHandler.ReincludeDevice(deviceID)

// Get excluded devices
excludedDevices := failureHandler.GetExcludedDevices()

// Cleanup expired exclusions
removed := failureHandler.CleanupExpiredExclusions()
```

## Device Pool Management

### Get Available Devices

```go
poolManager := manager.GetPoolManager()

// Get all available (online, non-excluded) devices
devices, err := poolManager.GetAvailableDevices(ctx, tenantID)

// Get all devices regardless of status
allDevices, err := poolManager.GetAllDevices(ctx, tenantID)

// Get specific device
device, err := poolManager.GetDeviceByID(ctx, deviceID)
```

### Update Device Metrics

```go
// Update device metrics
metrics := distribution.DeviceMetricsUpdate{
    TotalMessagesSent:       distribution.Int64Ptr(1),
    SuccessfulMessages:      distribution.Int64Ptr(1),
    AverageProcessingTimeMs: distribution.IntPtr(150),
}

err := poolManager.UpdateDeviceMetrics(ctx, deviceID, metrics)
```

### Device Status Management

```go
// Mark device as online
err := poolManager.MarkDeviceAsOnline(ctx, deviceID)

// Mark device as failed
err := poolManager.MarkDeviceAsFailed(ctx, deviceID)

// Update device status
err := poolManager.UpdateDeviceStatus(ctx, deviceID, distribution.DeviceStatusOffline)
```

## Distribution Metrics

Track distribution statistics per device:

```go
// Save distribution metrics
metrics := &distribution.DistributionMetrics{
    TenantID:                tenantID,
    DeviceID:                deviceID,
    PeriodStart:             startTime,
    PeriodEnd:               endTime,
    MessagesAssigned:        100,
    MessagesSent:            95,
    MessagesFailed:          5,
    AverageProcessingTimeMs: 150,
    MinProcessingTimeMs:     50,
    MaxProcessingTimeMs:     500,
}

err := poolManager.SaveDistributionMetrics(ctx, metrics)

// Retrieve metrics
metrics, err := poolManager.GetDistributionMetrics(ctx, deviceID, startTime, endTime)
```

## Database Schema

### phone_devices Table

Stores device information and status:

```sql
CREATE TABLE phone_devices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    device_name VARCHAR(255) NOT NULL,
    priority_level INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    current_load INTEGER NOT NULL DEFAULT 0,
    total_messages_sent BIGINT NOT NULL DEFAULT 0,
    successful_messages BIGINT NOT NULL DEFAULT 0,
    failed_messages BIGINT NOT NULL DEFAULT 0,
    -- ... other fields
);
```

### distribution_strategies Table

Stores distribution strategy configuration:

```sql
CREATE TABLE distribution_strategies (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL UNIQUE,
    strategy_type VARCHAR(50) NOT NULL,
    strategy_config JSONB,
    custom_rules JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- ... other fields
);
```

### distribution_metrics Table

Stores distribution statistics:

```sql
CREATE TABLE distribution_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    device_id UUID NOT NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    messages_assigned BIGINT NOT NULL DEFAULT 0,
    messages_sent BIGINT NOT NULL DEFAULT 0,
    messages_failed BIGINT NOT NULL DEFAULT 0,
    -- ... other fields
);
```

## Error Handling

The system provides specific error types:

```go
// Check for specific errors
if errors.Is(err, distribution.ErrNoDevicesAvailable) {
    // Handle no devices available
}

if errors.Is(err, distribution.ErrStrategyNotConfigured) {
    // Handle strategy not configured
}

if errors.Is(err, distribution.ErrDeviceNotFound) {
    // Handle device not found
}

if errors.Is(err, distribution.ErrInvalidCustomRule) {
    // Handle invalid custom rule
}
```

## Thread Safety

All components are thread-safe:

- **RoundRobinStrategy**: Uses `sync.Mutex` for counter protection
- **LoadBalancingStrategy**: Uses `sync.RWMutex` for device list access
- **PriorityStrategy**: Uses `sync.RWMutex` for device list access
- **CustomRulesStrategy**: Uses `sync.RWMutex` for device list and rules access
- **DeviceFailureHandler**: Uses `sync.RWMutex` for exclusion map access

## Performance Considerations

1. **Strategy Caching**: Strategies are cached per tenant to avoid repeated database queries
2. **Device Filtering**: Excluded devices are filtered in-memory for fast access
3. **Batch Operations**: Use batch operations for updating multiple device metrics
4. **Periodic Cleanup**: Run `CleanupExpiredExclusions()` periodically to prevent memory leaks

## Integration Example

Complete example of integrating the distribution system:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/google/uuid"
    "github.com/yourusername/yourproject/internal/admin/distribution"
    "gorm.io/gorm"
)

func main() {
    // Initialize
    db := // your gorm.DB instance
    manager := distribution.NewDistributionManager(db)
    ctx := context.Background()
    tenantID := uuid.MustParse("your-tenant-id")

    // Configure strategy
    err := manager.GetSelector().UpdateStrategyConfig(
        ctx,
        tenantID,
        distribution.StrategyLoadBalancing,
        nil,
        nil,
    )
    if err != nil {
        log.Fatalf("Failed to configure strategy: %v", err)
    }

    // Message distribution loop
    for i := 0; i < 100; i++ {
        // Select device
        device, err := manager.SelectDeviceForMessage(ctx, tenantID)
        if err != nil {
            log.Printf("Failed to select device: %v", err)
            continue
        }

        // Increment load
        err = manager.GetPoolManager().IncrementDeviceLoad(ctx, device.ID)
        if err != nil {
            log.Printf("Failed to increment load: %v", err)
        }

        // Send message (your implementation)
        err = sendMessage(device, "Hello, World!")
        if err != nil {
            // Handle failure
            manager.GetPoolManager().MarkDeviceAsFailed(ctx, device.ID)
            manager.GetPoolManager().DecrementDeviceLoad(ctx, device.ID)
            continue
        }

        // Decrement load after successful send
        err = manager.GetPoolManager().DecrementDeviceLoad(ctx, device.ID)
        if err != nil {
            log.Printf("Failed to decrement load: %v", err)
        }

        // Update metrics
        metrics := distribution.DeviceMetricsUpdate{
            TotalMessagesSent:  distribution.Int64Ptr(1),
            SuccessfulMessages: distribution.Int64Ptr(1),
        }
        manager.GetPoolManager().UpdateDeviceMetrics(ctx, device.ID, metrics)
    }
}

func sendMessage(device *distribution.Device, message string) error {
    // Your message sending implementation
    return nil
}
```

## Requirements Validation

This implementation validates the following requirements:

- **Requirement 3.1**: Multiple distribution strategies supported
- **Requirement 3.2**: Round Robin with fair distribution (5% variance)
- **Requirement 3.3**: Load Balancing based on current device load
- **Requirement 3.4**: Priority-based distribution
- **Requirement 3.5**: Custom Rules support
- **Requirement 3.6**: Per-company strategy configuration
- **Requirement 3.7**: Distribution metrics tracking
- **Requirement 3.8**: Device failure handling (5-minute exclusion)
- **Requirement 3.9**: Real-time distribution status
- **Requirement 3.10**: Fair distribution in Round Robin mode

## Testing

See the property tests in `distribution_property_test.go` for comprehensive testing of:
- Distribution strategy execution
- Round Robin fairness
- Load Balancing selection
- Priority-based selection
- Device failure exclusion

## Support

For issues or questions, please refer to the main project documentation or contact the development team.
