# Alerting System

The alerting system provides a flexible way to send alerts through multiple channels (email, webhook, SMS) with retry logic and failure handling.

## Features

- **Multiple Channels**: Support for email, webhook, and extensible to SMS
- **Asynchronous Sending**: Non-blocking alert delivery
- **Retry Logic**: Automatic retry with exponential backoff
- **Alert History**: Track sent alerts and their status
- **Graceful Failure Handling**: Continues operation even if some channels fail
- **Thread-Safe**: Safe for concurrent use

## Usage

### Basic Setup

```go
import (
    "github.com/yourusername/alhudhud/services/backend/internal/alerting"
    "go.uber.org/zap"
)

// Create logger
logger := zap.NewProduction()

// Create alert manager
manager := alerting.NewManager(logger)

// Add email channel
emailChannel := alerting.NewEmailChannel(
    "smtp.gmail.com",
    587,
    "alerts@example.com",
    "password",
    []string{"admin@example.com", "ops@example.com"},
)
manager.AddChannel(emailChannel)

// Add webhook channel
webhookChannel := alerting.NewWebhookChannel("https://hooks.slack.com/services/YOUR/WEBHOOK/URL")
manager.AddChannel(webhookChannel)
```

### Sending Alerts

```go
// Send async alert (non-blocking)
manager.SendAlert(
    alerting.AlertLevelCritical,
    "Database Connection Lost",
    "Unable to connect to primary database",
    map[string]interface{}{
        "host": "db.example.com",
        "port": 5432,
        "error": "connection timeout",
    },
)

// Send sync alert (waits for all channels)
ctx := context.Background()
err := manager.SendAlertSync(
    ctx,
    alerting.AlertLevelWarning,
    "High Queue Size",
    "Message queue size exceeded threshold",
    map[string]interface{}{
        "queue_size": 10000,
        "threshold": 5000,
    },
)
if err != nil {
    log.Printf("Failed to send alert: %v", err)
}
```

### Alert Levels

- `AlertLevelInfo`: Informational messages
- `AlertLevelWarning`: Warning conditions that need attention
- `AlertLevelCritical`: Critical issues requiring immediate action

### Custom Configuration

```go
config := alerting.ManagerConfig{
    MaxHistory: 500,              // Keep last 500 alerts in history
    MaxRetries: 5,                // Retry up to 5 times
    RetryDelay: 3 * time.Second,  // Initial retry delay (exponential backoff)
}

manager := alerting.NewManagerWithConfig(logger, config)
```

### Viewing Alert History

```go
// Get all history
history := manager.GetHistory()
for _, record := range history {
    fmt.Printf("Alert: %s, Success: %v, Retries: %d\n",
        record.Alert.Title,
        record.Success,
        record.RetryCount,
    )
}

// Get recent 10 alerts
recent := manager.GetRecentHistory(10)

// Clear history
manager.ClearHistory()
```

## Integration Examples

### Monitor Queue Backup

```go
func monitorQueue(manager *alerting.Manager, queue *redis.MessageQueue) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        size := queue.GetSize()
        if size > 5000 {
            manager.SendAlert(
                alerting.AlertLevelWarning,
                "Queue Backup",
                fmt.Sprintf("Queue size is %d, exceeding threshold of 5000", size),
                map[string]interface{}{
                    "queue_size": size,
                    "threshold": 5000,
                },
            )
        }
    }
}
```

### Monitor Error Rate

```go
func monitorErrorRate(manager *alerting.Manager, metrics *Metrics) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        errorRate := metrics.GetErrorRate()
        if errorRate > 0.1 { // 10% error rate
            manager.SendAlert(
                alerting.AlertLevelCritical,
                "High Error Rate",
                fmt.Sprintf("Error rate is %.2f%%, exceeding threshold of 10%%", errorRate*100),
                map[string]interface{}{
                    "error_rate": errorRate,
                    "threshold": 0.1,
                },
            )
        }
    }
}
```

### Monitor Low Balance

```go
func monitorBalance(manager *alerting.Manager, balanceManager *balance.Manager, tenantID uuid.UUID) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        balance, err := balanceManager.GetBalance(context.Background(), tenantID)
        if err != nil {
            continue
        }

        if balance.LessThan(decimal.NewFromInt(100)) {
            manager.SendAlert(
                alerting.AlertLevelWarning,
                "Low Balance Alert",
                fmt.Sprintf("Tenant %s balance is %s SAR", tenantID, balance.String()),
                map[string]interface{}{
                    "tenant_id": tenantID.String(),
                    "balance": balance.String(),
                    "threshold": "100",
                },
            )
        }
    }
}
```

## Implementing Custom Channels

To implement a custom alert channel, implement the `AlertChannel` interface:

```go
type SMSChannel struct {
    apiKey      string
    phoneNumbers []string
}

func NewSMSChannel(apiKey string, phoneNumbers []string) *SMSChannel {
    return &SMSChannel{
        apiKey:      apiKey,
        phoneNumbers: phoneNumbers,
    }
}

func (sc *SMSChannel) Send(alert alerting.Alert) error {
    message := fmt.Sprintf("[%s] %s: %s", alert.Level, alert.Title, alert.Message)
    
    for _, phone := range sc.phoneNumbers {
        err := sendSMS(sc.apiKey, phone, message)
        if err != nil {
            return fmt.Errorf("send SMS to %s: %w", phone, err)
        }
    }
    
    return nil
}

// Add to manager
smsChannel := NewSMSChannel("your-api-key", []string{"+1234567890"})
manager.AddChannel(smsChannel)
```

## Testing

The package includes comprehensive tests. Run them with:

```bash
go test -v ./internal/alerting/...
```

## Thread Safety

All methods are thread-safe and can be called concurrently from multiple goroutines.
