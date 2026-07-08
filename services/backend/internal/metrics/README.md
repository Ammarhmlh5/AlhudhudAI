# Prometheus Metrics Package

## Overview

This package implements comprehensive Prometheus metrics for the SMS Gateway system, fulfilling Requirements 16.1, 16.2, 16.3, and 16.4.

## Features

### 1. Counter Metrics

- **messages_received_total**: Total messages received by API Gateway (labels: tenant_id, priority)
- **messages_sent_total**: Total messages successfully sent (labels: tenant_id, channel_id)
- **messages_failed_total**: Total messages that failed (labels: tenant_id, reason)
- **balance_operations_total**: Total balance operations (labels: tenant_id, type, result)
- **status_reports_processed_total**: Total status reports processed (labels: tenant_id, status)
- **batches_processed_total**: Total batches processed (labels: tenant_id, result)
- **cache_hits_total**: Total cache hits (labels: cache_type)
- **cache_misses_total**: Total cache misses (labels: cache_type)

### 2. Gauge Metrics

- **queue_size**: Current queue size (labels: tenant_id, queue_type)
- **active_workers**: Current number of active workers
- **database_connections**: Database connection pool stats (labels: state)
- **websocket_connections_active**: Active WebSocket connections (labels: tenant_id)
- **tenant_balance**: Current tenant balance (labels: tenant_id)
- **success_rate**: Message success rate percentage (labels: tenant_id)
- **cache_hit_rate**: Cache hit rate percentage (labels: cache_type)

### 3. Histogram Metrics

- **message_latency_seconds**: Message processing latency (labels: stage)
- **api_response_time_seconds**: API response time (labels: endpoint, method, status)
- **batch_size**: Batch processing size
- **balance_operation_duration_seconds**: Balance operation duration (labels: operation_type)

## Usage

### Recording Metrics

```go
import "github.com/al-hudhud/backend/internal/metrics"

// Record message received
metrics.RecordMessageReceived("1", "HIGH")

// Record message sent
metrics.RecordMessageSent("1", "100")

// Record message failed
metrics.RecordMessageFailed("1", "NETWORK_ERROR")

// Record balance operation
metrics.RecordBalanceOperation("1", "DEDUCT", "success")

// Set queue size
metrics.SetQueueSize("1", "pending", 100.0)

// Observe latency
metrics.ObserveMessageLatency("api_gateway", 0.005)
```

### HTTP Middleware

The package provides HTTP middleware for automatic metrics collection:

```go
import "github.com/al-hudhud/backend/internal/metrics"

// Wrap your handler
handler := metrics.HTTPMetricsMiddleware(yourHandler)
```

### Metrics Collector

The collector automatically gathers metrics from database and Redis:

```go
import "github.com/al-hudhud/backend/internal/metrics"

// Create collector
collector := metrics.NewCollector(db, redisClient)

// Start collection (runs every 30 seconds)
go collector.Start(context.Background(), 30*time.Second)
```

## Endpoints

### Prometheus Metrics Endpoint

```
GET /api/v1/metrics/prometheus
```

Returns metrics in Prometheus format for scraping.

### Legacy Metrics Endpoint

```
GET /api/v1/metrics
```

Returns metrics in JSON format (legacy endpoint, maintained for backward compatibility).

## Integration

### API Gateway

Metrics are automatically recorded in the gateway handler:
- Message received count
- API response time
- Message latency

### Batch Processor

Metrics are recorded during batch processing:
- Batch size
- Batch processing result
- Processing latency

### Balance Manager

Metrics are recorded for all balance operations:
- Balance operation count (deduct, refund, charge)
- Operation duration
- Operation result (success/failure)
- Current tenant balance

### Status Handler

Metrics are recorded when processing status reports:
- Status report count
- Message sent/failed count
- Status type distribution

### Routing Engine

Metrics are recorded for cache operations:
- Cache hits
- Cache misses
- Cache hit rate

## Testing

Run tests with:

```bash
go test ./internal/metrics/... -v
```

## Monitoring

### Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'sms-gateway'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/api/v1/metrics/prometheus'
    scrape_interval: 15s
```

### Grafana Dashboards

Key metrics to monitor:

1. **Message Throughput**
   - messages_received_total (rate)
   - messages_sent_total (rate)
   - messages_failed_total (rate)

2. **Performance**
   - message_latency_seconds (histogram)
   - api_response_time_seconds (histogram)
   - success_rate (gauge)

3. **Resources**
   - queue_size (gauge)
   - database_connections (gauge)
   - active_workers (gauge)

4. **Financial**
   - balance_operations_total (counter)
   - tenant_balance (gauge)

5. **Cache Performance**
   - cache_hit_rate (gauge)
   - cache_hits_total (counter)
   - cache_misses_total (counter)

## Alert Rules

Recommended Prometheus alert rules:

```yaml
groups:
  - name: sms_gateway_alerts
    rules:
      - alert: HighQueueSize
        expr: queue_size > 1000
        for: 5m
        annotations:
          summary: "Queue size is too high"
          
      - alert: HighLatency
        expr: histogram_quantile(0.95, message_latency_seconds) > 2
        for: 5m
        annotations:
          summary: "Message latency is too high"
          
      - alert: LowSuccessRate
        expr: success_rate < 95
        for: 10m
        annotations:
          summary: "Success rate is below 95%"
```

## Requirements Mapping

- **Requirement 16.1**: ✅ Prometheus metrics for all operations
- **Requirement 16.2**: ✅ Messages received, sent, failed, average latency
- **Requirement 16.3**: ✅ Database connections, cache hit rate, queue size
- **Requirement 16.4**: ✅ /metrics endpoint for Prometheus

## Files

- `metrics.go`: Core metrics definitions and helper functions
- `middleware.go`: HTTP middleware for automatic metrics collection
- `collector.go`: Background collector for database and Redis metrics
- `handler.go`: HTTP handlers for metrics endpoints
- `metrics_test.go`: Unit tests for metrics
- `middleware_test.go`: Unit tests for middleware
- `README.md`: This documentation

## Performance Considerations

- Metrics are recorded using Prometheus client library which is highly optimized
- Counter and gauge operations are O(1)
- Histogram observations are O(1) with pre-allocated buckets
- Collector runs in background goroutine to avoid blocking main operations
- Cache metrics are updated on every cache operation with minimal overhead

## Future Enhancements

1. Add more granular latency tracking per tenant
2. Implement custom metrics for business KPIs
3. Add metrics for webhook delivery
4. Track message retry attempts
5. Monitor Dead Letter Queue size
