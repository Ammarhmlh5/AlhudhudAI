package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Entity Type Constants
const (
	EntityTypeEvent       = "EV"
	EntityTypeMessage     = "MS"
	EntityTypeTenant      = "TN"
	EntityTypeChannel     = "CH"
	EntityTypeGateway     = "GW"
	EntityTypeTransaction = "TR"
	EntityTypeInvoice     = "IN"
	EntityTypePackage     = "PK"
	EntityTypeSubscription = "SB"
)

// IDGenerator generates unique IDs using Redis
type IDGenerator struct {
	client *Client
}

// NewIDGenerator creates a new ID generator
func NewIDGenerator(client *Client) *IDGenerator {
	return &IDGenerator{
		client: client,
	}
}

// GenerateEventID generates a unique event ID
// Format: EV + YYMMDDHHmm (10) + TenantHash (4) + Sequence (4)
// Example: EV260101231234560001
func (g *IDGenerator) GenerateEventID(ctx context.Context, tenantID uuid.UUID, senderIP string) (string, error) {
	now := time.Now()
	timestamp := formatTimestamp(now)
	tenantHash := generateTenantHash(tenantID)[:4] // Use 4 digits of hash
	
	// Redis key for event counter
	counterKey := fmt.Sprintf("%s:event:counter:%s:%s", AppNamespace, timestamp, tenantHash)
	
	// Increment counter atomically
	sequence, err := g.client.Client.Incr(ctx, counterKey).Result()
	if err != nil {
		return "", fmt.Errorf("failed to generate event sequence: %w", err)
	}
	
	// Set expiry for counter
	g.client.Client.Expire(ctx, counterKey, 1*time.Hour)
	
	// Check if sequence exceeds limit (9999 per minute)
	if sequence > 9999 {
		return "", fmt.Errorf("event sequence limit exceeded for timestamp %s", timestamp)
	}
	
	// Generate Event ID: EV + timestamp(10) + tenantHash(4) + sequence(4) = 20 chars
	eventID := fmt.Sprintf("%s%s%s%04d", EntityTypeEvent, timestamp, tenantHash, sequence)
	
	// Store event metadata in Redis
	eventKey := fmt.Sprintf("%s:event:%s", AppNamespace, eventID)
	pipe := g.client.Client.Pipeline()
	pipe.HSet(ctx, eventKey, map[string]interface{}{
		"tenant_id":  tenantID.String(),
		"sender_ip":  senderIP,
		"created_at": now.Unix(),
		"msg_count":  0,
	})
	pipe.Expire(ctx, eventKey, 7*24*time.Hour)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to store event metadata: %w", err)
	}
	
	return eventID, nil
}

// GenerateMessageID generates a unique message ID within an event
// Format: MS + YYMMDDHHmm (10) + TenantHash (4) + EventSeq (4) + MsgSeq (3)
// Example: MS260101231234560001001
func (g *IDGenerator) GenerateMessageID(ctx context.Context, eventID string) (string, error) {
	// Validate event ID format
	if len(eventID) != 20 || eventID[:2] != EntityTypeEvent {
		return "", fmt.Errorf("invalid event ID format: %s", eventID)
	}
	
	// Extract parts from event ID
	timestamp := eventID[2:12]
	tenantHash := eventID[12:16]
	eventSeq := eventID[16:20]
	
	// Redis key for message counter within event
	counterKey := MessageCounterKey(eventID)
	
	// Increment message counter atomically
	msgSequence, err := g.client.Client.Incr(ctx, counterKey).Result()
	if err != nil {
		return "", fmt.Errorf("failed to generate message sequence: %w", err)
	}
	
	// Set expiry for counter
	g.client.Client.Expire(ctx, counterKey, 7*24*time.Hour)
	
	// Check if message limit exceeded (999 messages per event)
	if msgSequence > 999 {
		return "", fmt.Errorf("message limit exceeded for event %s (max 999)", eventID)
	}
	
	// Generate Message ID: MS + timestamp(10) + tenantHash(4) + eventSeq(4) + msgSeq(3) = 23 chars
	messageID := fmt.Sprintf("%s%s%s%s%03d", EntityTypeMessage, timestamp, tenantHash, eventSeq, msgSequence)
	
	// Update event message count
	eventKey := fmt.Sprintf("%s:event:%s", AppNamespace, eventID)
	g.client.Client.HIncrBy(ctx, eventKey, "msg_count", 1)
	
	return messageID, nil
}

// ParseEventID extracts information from an event ID
func ParseEventID(eventID string) (*EventIDInfo, error) {
	if len(eventID) != 20 || eventID[:2] != EntityTypeEvent {
		return nil, fmt.Errorf("invalid event ID format")
	}
	
	timestamp := eventID[2:12]
	tenantHash := eventID[12:16]
	sequence := eventID[16:20]
	
	// Parse timestamp: YYMMDDHHmm
	year, _ := strconv.Atoi("20" + timestamp[0:2])
	month, _ := strconv.Atoi(timestamp[2:4])
	day, _ := strconv.Atoi(timestamp[4:6])
	hour, _ := strconv.Atoi(timestamp[6:8])
	minute, _ := strconv.Atoi(timestamp[8:10])
	
	seq, _ := strconv.Atoi(sequence)
	ts := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
	
	return &EventIDInfo{
		EntityType:  EntityTypeEvent,
		Timestamp:   ts,
		Year:        year,
		Month:       month,
		Day:         day,
		Hour:        hour,
		TenantHash:  tenantHash,
		Sequence:    seq,
		OriginalID:  eventID,
	}, nil
}

// ParseMessageID extracts information from a message ID
func ParseMessageID(messageID string) (*MessageIDInfo, error) {
	if len(messageID) != 23 || messageID[:2] != EntityTypeMessage {
		return nil, fmt.Errorf("invalid message ID format")
	}
	
	timestamp := messageID[2:12]
	tenantHash := messageID[12:16]
	eventSeq := messageID[16:20]
	msgSeq := messageID[20:23]
	
	// Parse timestamp
	year, _ := strconv.Atoi("20" + timestamp[0:2])
	month, _ := strconv.Atoi(timestamp[2:4])
	day, _ := strconv.Atoi(timestamp[4:6])
	hour, _ := strconv.Atoi(timestamp[6:8])
	minute, _ := strconv.Atoi(timestamp[8:10])
	
	eSeq, _ := strconv.Atoi(eventSeq)
	mSeq, _ := strconv.Atoi(msgSeq)
	ts := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
	
	// Reconstruct event ID
	eventID := fmt.Sprintf("%s%s%s%s", EntityTypeEvent, timestamp, tenantHash, eventSeq)
	
	return &MessageIDInfo{
		EntityType:  EntityTypeMessage,
		Timestamp:   ts,
		Year:        year,
		Month:       month,
		Day:         day,
		Hour:        hour,
		TenantHash:  tenantHash,
		EventSeq:    eSeq,
		MessageSeq:  mSeq,
		EventID:     eventID,
		OriginalID:  messageID,
	}, nil
}

// GetEventMetadata retrieves event metadata from Redis
func (g *IDGenerator) GetEventMetadata(ctx context.Context, eventID string) (map[string]string, error) {
	eventKey := fmt.Sprintf("%s:event:%s", AppNamespace, eventID)
	return g.client.Client.HGetAll(ctx, eventKey).Result()
}

// GetEventMessageCount returns the number of messages in an event
func (g *IDGenerator) GetEventMessageCount(ctx context.Context, eventID string) (int64, error) {
	counterKey := MessageCounterKey(eventID)
	count, err := g.client.Client.Get(ctx, counterKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// EventIDInfo holds parsed event ID information
type EventIDInfo struct {
	EntityType string
	Timestamp  time.Time
	Year       int
	Month      int
	Day        int
	Hour       int
	TenantHash string
	Sequence   int
	OriginalID string
}

// MessageIDInfo holds parsed message ID information
type MessageIDInfo struct {
	EntityType string
	Timestamp  time.Time
	Year       int
	Month      int
	Day        int
	Hour       int
	TenantHash string
	EventSeq   int
	MessageSeq int
	EventID    string
	OriginalID string
}

// Helper functions

// generateTenantHash generates a 5-digit hash from tenant UUID
func generateTenantHash(tenantID uuid.UUID) string {
	// Use first 8 bytes of UUID to generate hash
	bytes := tenantID[0:8]
	var hash uint32
	for _, b := range bytes {
		hash = hash*31 + uint32(b)
	}
	// Modulo to get 5 digits (00000-99999)
	return fmt.Sprintf("%05d", hash%100000)
}
