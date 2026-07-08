package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// LoadAnalytics provides analytics for event and message load
type LoadAnalytics struct {
	client *Client
}

// NewLoadAnalytics creates a new load analytics instance
func NewLoadAnalytics(client *Client) *LoadAnalytics {
	return &LoadAnalytics{
		client: client,
	}
}

// LoadMetrics holds load metrics for a time period
type LoadMetrics struct {
	Timestamp      time.Time
	TenantHash     string
	EventCount     int64
	MessageCount   int64
	AvgMsgPerEvent float64
	MaxMsgPerEvent int64
	NearLimitCount int64 // Events with >900 messages
}

// GetCurrentLoad returns current load metrics
func (la *LoadAnalytics) GetCurrentLoad(ctx context.Context) (*LoadMetrics, error) {
	now := time.Now()
	timestamp := formatTimestamp(now)
	
	// Pattern for all event counters in current timestamp
	pattern := fmt.Sprintf("%s:event:counter:%s:*", AppNamespace, timestamp)
	
	var totalEvents int64
	var totalMessages int64
	var maxMessages int64
	var nearLimit int64
	
	// Scan all event counters with retry
	var iter *goredis.ScanIterator
	retryErr := la.client.WithRetry(ctx, func() error {
		iter = la.client.Client.Scan(ctx, 0, pattern, 100).Iterator()
		return nil
	})
	if retryErr != nil {
		return nil, retryErr
	}
	
	for iter.Next(ctx) {
		key := iter.Val()
		var count int64
		var err error
		
		// Use retry for each counter get
		retryErr := la.client.WithRetry(ctx, func() error {
			count, err = la.client.Client.Get(ctx, key).Int64()
			return err
		})
		if retryErr != nil {
			continue
		}
		totalEvents += count
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}
	
	// 1. Count messages from event metadata
	// Scan for all event metadata keys with current timestamp
	metadataPattern := fmt.Sprintf("%s:event:metadata:EV%s*", AppNamespace, timestamp)
	var metaIter *goredis.ScanIterator
	retryErr = la.client.WithRetry(ctx, func() error {
		metaIter = la.client.Client.Scan(ctx, 0, metadataPattern, 100).Iterator()
		return nil
	})
	if retryErr != nil {
		return nil, retryErr
	}
	
	for metaIter.Next(ctx) {
		key := metaIter.Val()
		// Get msg_count from event metadata hash with retry
		var msgCountStr string
		var err error
		
		retryErr := la.client.WithRetry(ctx, func() error {
			msgCountStr, err = la.client.Client.HGet(ctx, key, "msg_count").Result()
			return err
		})
		if retryErr != nil {
			continue
		}
		
		var msgCount int64
		_, err = fmt.Sscanf(msgCountStr, "%d", &msgCount)
		if err != nil {
			continue
		}
		
		totalMessages += msgCount
		if msgCount > maxMessages {
			maxMessages = msgCount
		}
		if msgCount >= 900 {
			nearLimit++
		}
	}
	
	if err := metaIter.Err(); err != nil {
		return nil, err
	}
	
	// 2. Count messages in queues (pending messages)
	// Scan for all queue keys: alhudhud:queue:messages:*
	queuePattern := fmt.Sprintf("%s:queue:messages:*", AppNamespace)
	var queueIter *goredis.ScanIterator
	retryErr = la.client.WithRetry(ctx, func() error {
		queueIter = la.client.Client.Scan(ctx, 0, queuePattern, 100).Iterator()
		return nil
	})
	if retryErr != nil {
		return nil, retryErr
	}
	
	for queueIter.Next(ctx) {
		key := queueIter.Val()
		var queueSize int64
		var err error
		
		retryErr := la.client.WithRetry(ctx, func() error {
			queueSize, err = la.client.Client.LLen(ctx, key).Result()
			return err
		})
		if retryErr != nil {
			continue
		}
		totalMessages += queueSize
	}
	
	if err := queueIter.Err(); err != nil {
		return nil, err
	}
	
	avgMsgPerEvent := float64(0)
	if totalEvents > 0 {
		avgMsgPerEvent = float64(totalMessages) / float64(totalEvents)
	}
	
	return &LoadMetrics{
		Timestamp:      now,
		EventCount:     totalEvents,
		MessageCount:   totalMessages,
		AvgMsgPerEvent: avgMsgPerEvent,
		MaxMsgPerEvent: maxMessages,
		NearLimitCount: nearLimit,
	}, nil
}

// GetLoadByTenant returns load metrics for a specific tenant
func (la *LoadAnalytics) GetLoadByTenant(ctx context.Context, tenantHash string, duration time.Duration) ([]*LoadMetrics, error) {
	var metrics []*LoadMetrics
	
	// Get metrics for each hour in the duration
	now := time.Now()
	for i := 0; i < int(duration.Hours()); i++ {
		t := now.Add(-time.Duration(i) * time.Hour)
		timestamp := formatTimestamp(t)
		
		counterKey := fmt.Sprintf("%s:event:counter:%s:%s", AppNamespace, timestamp, tenantHash)
		var eventCount int64
		var err error
		
		// Use retry for counter get
		retryErr := la.client.WithRetry(ctx, func() error {
			eventCount, err = la.client.Client.Get(ctx, counterKey).Int64()
			return err
		})
		if retryErr != nil {
			eventCount = 0
		}
		
		metrics = append(metrics, &LoadMetrics{
			Timestamp:  t,
			TenantHash: tenantHash,
			EventCount: eventCount,
		})
	}
	
	return metrics, nil
}

// DetectPressure detects if system is under pressure
func (la *LoadAnalytics) DetectPressure(ctx context.Context) (*PressureInfo, error) {
	metrics, err := la.GetCurrentLoad(ctx)
	if err != nil {
		return nil, err
	}
	
	pressure := &PressureInfo{
		Timestamp: metrics.Timestamp,
		Level:     PressureLevelNormal,
	}
	
	// Detect pressure based on metrics
	if metrics.EventCount > 5000 {
		pressure.Level = PressureLevelHigh
		pressure.Reason = fmt.Sprintf("High event count: %d events/hour", metrics.EventCount)
	} else if metrics.EventCount > 2000 {
		pressure.Level = PressureLevelMedium
		pressure.Reason = fmt.Sprintf("Medium event count: %d events/hour", metrics.EventCount)
	}
	
	if metrics.NearLimitCount > 10 {
		pressure.Level = PressureLevelHigh
		pressure.Reason = fmt.Sprintf("Many events near limit: %d events with >900 messages", metrics.NearLimitCount)
	}
	
	if metrics.AvgMsgPerEvent > 800 {
		if pressure.Level < PressureLevelMedium {
			pressure.Level = PressureLevelMedium
		}
		pressure.Reason += fmt.Sprintf("; High avg messages per event: %.0f", metrics.AvgMsgPerEvent)
	}
	
	return pressure, nil
}

// PressureLevel indicates system pressure level
type PressureLevel int

const (
	PressureLevelNormal PressureLevel = iota
	PressureLevelMedium
	PressureLevelHigh
	PressureLevelCritical
)

// PressureInfo holds pressure detection information
type PressureInfo struct {
	Timestamp time.Time
	Level     PressureLevel
	Reason    string
}

// String returns string representation of pressure level
func (pl PressureLevel) String() string {
	switch pl {
	case PressureLevelNormal:
		return "NORMAL"
	case PressureLevelMedium:
		return "MEDIUM"
	case PressureLevelHigh:
		return "HIGH"
	case PressureLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// GetEventStats returns statistics for a specific event
func (la *LoadAnalytics) GetEventStats(ctx context.Context, eventID string) (*EventStats, error) {
	metadata, err := NewIDGenerator(la.client).GetEventMetadata(ctx, eventID)
	if err != nil {
		return nil, err
	}
	
	msgCount, err := NewIDGenerator(la.client).GetEventMessageCount(ctx, eventID)
	if err != nil {
		return nil, err
	}
	
	return &EventStats{
		EventID:      eventID,
		TenantID:     metadata["tenant_id"],
		SenderIP:     metadata["sender_ip"],
		MessageCount: msgCount,
		CreatedAt:    time.Unix(parseInt64(metadata["created_at"]), 0),
	}, nil
}

// EventStats holds statistics for an event
type EventStats struct {
	EventID      string
	TenantID     string
	SenderIP     string
	MessageCount int64
	CreatedAt    time.Time
}

// Helper function
func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
