# Statistics Aggregator - Quick Start Guide

## Overview

The Statistics Aggregator is a component of the Monitoring Service that calculates and stores daily statistics for dedicated gateways. It aggregates data from multiple sources to provide comprehensive insights into gateway performance, message delivery, uptime, and financial metrics.

## Features

### Core Functionality

1. **Daily Statistics Aggregation**
   - Automatically aggregates statistics for all active gateways
   - Calculates message statistics (sent, delivered, failed, pending)
   - Computes performance metrics (response times)
   - Tracks uptime and downtime
   - Summarizes financial data (cost, revenue, profit)

2. **Message Statistics** (Requirements 13.2, 13.3)
   - Messages sent
   - Messages delivered
   - Messages failed
   - Messages pending

3. **Performance Statistics** (Requirements 13.4, 13.5)
   - Average response time
   - Maximum response time
   - Minimum response time

4. **Uptime Statistics** (Requirement 13.6)
   - Uptime percentage
   - Downtime minutes

5. **Financial Statistics** (Requirement 13.7)
   - Total cost
   - Total revenue
   - Profit margin (automatically calculated)

6. **Success Rate Calculation** (Requirement 13.10)
   - Automatically calculates delivery success rate
   - Formula: (messages_delivered / messages_sent) * 100

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Statistics Aggregator                     │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌───────▼────────┐   ┌───────▼────────┐
│    Messages    │   │  Message Logs  │   │ Health Checks  │
│     Table      │   │     Table      │   │     Table      │
└────────────────┘   └────────────────┘   └────────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │ Gateway Statistics│
                    │      Table        │
                    └───────────────────┘
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "database/sql"
    "time"
    
    "your-project/internal/monitoring"
)

func main() {
    // Initialize database connection
    db, err := sql.Open("postgres", "your-connection-string")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Create aggregator
    aggregator := monitoring.NewAggregator(db)
    
    // Aggregate statistics for a specific gateway and date
    ctx := context.Background()
    gatewayID := 1
    date := time.Now().AddDate(0, 0, -1) // Yesterday
    
    err = aggregator.AggregateDaily(ctx, gatewayID, date)
    if err != nil {
        panic(err)
    }
}
```

### Aggregate All Gateways

```go
// Aggregate statistics for all active gateways
date := time.Now().AddDate(0, 0, -1) // Yesterday
err := aggregator.AggregateAllDaily(ctx, date)
if err != nil {
    log.Printf("Error aggregating statistics: %v", err)
}
```

### Start Daily Aggregation Job

```go
// Start a background job that runs daily
go aggregator.StartDailyAggregation(ctx)

// The job will:
// 1. Run immediately for yesterday's data
// 2. Run every 24 hours thereafter
// 3. Aggregate statistics for all active gateways
```

### Retrieve Statistics

```go
gatewayID := 1
tenantID := uuid.MustParse("your-tenant-id")
startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

statistics, err := aggregator.GetStatistics(ctx, gatewayID, tenantID, startDate, endDate)
if err != nil {
    log.Printf("Error retrieving statistics: %v", err)
    return
}

for _, stat := range statistics {
    fmt.Printf("Date: %s\n", stat.Date.Format("2006-01-02"))
    fmt.Printf("Messages Sent: %d\n", stat.MessagesSent)
    fmt.Printf("Messages Delivered: %d\n", stat.MessagesDelivered)
    fmt.Printf("Success Rate: %.2f%%\n", 
        float64(stat.MessagesDelivered)/float64(stat.MessagesSent)*100)
    fmt.Printf("Uptime: %.2f%%\n", stat.UptimePercentage)
    fmt.Printf("Profit: $%.2f\n\n", stat.ProfitMargin)
}
```

### Calculate Success Rate

```go
gatewayID := 1
tenantID := uuid.MustParse("your-tenant-id")
date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

successRate, err := aggregator.CalculateSuccessRate(ctx, gatewayID, tenantID, date)
if err != nil {
    log.Printf("Error calculating success rate: %v", err)
    return
}

fmt.Printf("Success Rate: %.2f%%\n", successRate)
```

## Data Sources

The aggregator collects data from multiple tables:

### 1. Messages Table
- **Purpose**: Message delivery statistics
- **Metrics**: sent, delivered, failed, pending counts
- **Query**: Filters by gateway_id and date

### 2. Message Logs Table (Optional)
- **Purpose**: Performance metrics
- **Metrics**: response_time_ms
- **Fallback**: If not available, calculates from messages table using (updated_at - created_at)

### 3. Gateway Health Checks Table (Optional)
- **Purpose**: Uptime statistics
- **Metrics**: is_healthy, check_interval_minutes
- **Default**: 100% uptime if no health checks exist

### 4. Accounting Transactions Table (Optional)
- **Purpose**: Financial statistics
- **Metrics**: cost and revenue transactions
- **Default**: $0.00 if no transactions exist

## Database Schema

### Gateway Statistics Table

```sql
CREATE TABLE gateway_statistics (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    gateway_id INTEGER NOT NULL,
    date DATE NOT NULL,
    
    -- Message Statistics
    messages_sent INTEGER DEFAULT 0,
    messages_delivered INTEGER DEFAULT 0,
    messages_failed INTEGER DEFAULT 0,
    messages_pending INTEGER DEFAULT 0,
    
    -- Performance Statistics
    avg_response_time_ms INTEGER,
    max_response_time_ms INTEGER,
    min_response_time_ms INTEGER,
    
    -- Uptime Statistics
    uptime_percentage DECIMAL(5, 2),
    downtime_minutes INTEGER DEFAULT 0,
    
    -- Financial Statistics
    total_cost DECIMAL(10, 2) DEFAULT 0.00,
    total_revenue DECIMAL(10, 2) DEFAULT 0.00,
    profit_margin DECIMAL(10, 2) DEFAULT 0.00,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(gateway_id, date)
);
```

## Tenant Isolation

The aggregator implements strict tenant isolation:

1. **RLS (Row Level Security)**
   - All queries respect tenant_id
   - Sets `app.current_tenant` before operations

2. **Automatic Tenant Detection**
   - Retrieves tenant_id from dedicated_gateways table
   - Validates tenant ownership

3. **Isolated Statistics**
   - Each tenant can only access their own statistics
   - Cross-tenant access is prevented

## Error Handling

The aggregator handles errors gracefully:

```go
// Continues with other gateways if one fails
err := aggregator.AggregateAllDaily(ctx, date)
if err != nil {
    // Some gateways failed, but others succeeded
    log.Printf("Some aggregations failed: %v", err)
}

// Missing tables are handled gracefully
// - message_logs: Falls back to messages table
// - gateway_health_checks: Defaults to 100% uptime
// - accounting_transactions: Defaults to $0.00
```

## Performance Considerations

1. **Batch Processing**
   - Aggregates all gateways in a single run
   - Processes one gateway at a time to avoid memory issues

2. **Upsert Operations**
   - Uses `ON CONFLICT` to update existing records
   - Prevents duplicate statistics

3. **Indexed Queries**
   - All queries use indexed columns (gateway_id, date)
   - Tenant isolation uses indexed tenant_id

4. **Scheduled Execution**
   - Runs once per day (typically at midnight)
   - Processes previous day's data

## Integration with Other Components

### Metrics Collector
- Provides real-time metrics
- Aggregator summarizes these into daily statistics

### Usage Logger
- Logs resource usage every minute
- Aggregator uses this data for performance metrics

### Alert Manager
- Can trigger alerts based on aggregated statistics
- Example: Low success rate, high downtime

## Testing

Run the comprehensive test suite:

```bash
cd services/backend
go test -v ./internal/monitoring/aggregator_test.go ./internal/monitoring/aggregator.go ./internal/monitoring/types.go
```

### Test Coverage

- ✅ Daily aggregation for single gateway
- ✅ Daily aggregation for all gateways
- ✅ Message statistics calculation
- ✅ Performance statistics calculation
- ✅ Uptime statistics calculation
- ✅ Financial statistics calculation
- ✅ Success rate calculation
- ✅ Statistics retrieval with date range
- ✅ Tenant isolation
- ✅ Error handling

## Monitoring

Monitor the aggregator's health:

```go
// Check if aggregation is running
// Look for daily log entries like:
// "Aggregating statistics for gateway 1"
// "Successfully aggregated statistics for 5 gateways"

// Check for errors
// "Error aggregating statistics for gateway 1: ..."
```

## Troubleshooting

### Issue: No statistics generated

**Solution**: Check if:
1. Gateways are marked as 'active'
2. Messages exist for the date
3. Tenant context is set correctly

### Issue: Performance metrics are NULL

**Solution**: 
- message_logs table might not exist
- Falls back to messages table
- Ensure messages have updated_at timestamps

### Issue: Uptime is always 100%

**Solution**:
- gateway_health_checks table might not exist
- This is expected if health checks aren't implemented
- Default behavior is to assume 100% uptime

## Next Steps

After implementing the Statistics Aggregator:

1. ✅ **Task 19.1**: Metrics Collector (Completed)
2. ✅ **Task 19.2**: Usage Logger (Completed)
3. ✅ **Task 19.3**: Alert Manager (Completed)
4. ✅ **Task 19.4**: Statistics Aggregator (Current - Completed)
5. ⏳ **Task 19.5**: Balance Monitor for dedicated gateways
6. ⏳ **Task 19.6**: Subscription Billing for dedicated gateways

## Requirements Validation

This implementation satisfies the following requirements:

- ✅ **Requirement 13.1**: Records daily statistics for each dedicated gateway
- ✅ **Requirement 13.2**: Records number of messages sent daily
- ✅ **Requirement 13.3**: Records delivered, failed, and pending messages
- ✅ **Requirement 13.4**: Records average response time daily
- ✅ **Requirement 13.5**: Records maximum response time daily
- ✅ **Requirement 13.6**: Records uptime percentage daily
- ✅ **Requirement 13.7**: Records cost and revenue daily
- ✅ **Requirement 13.10**: Automatically calculates success rate

## Summary

The Statistics Aggregator provides comprehensive daily statistics for dedicated gateways, enabling:

- Performance monitoring
- Financial tracking
- Uptime analysis
- Success rate calculation
- Historical trend analysis

All operations respect tenant isolation and handle missing data gracefully.
