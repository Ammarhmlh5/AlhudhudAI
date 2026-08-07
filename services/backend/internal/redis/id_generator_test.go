package redis

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return &Client{
		Client: client,
	}, mr
}

func TestGenerateEventID(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	tenantID := uuid.New()
	senderIP := "192.168.1.1"

	ctx := context.Background()
	eventID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)

	require.NoError(t, err)
	assert.NotEmpty(t, eventID)
	assert.Equal(t, 20, len(eventID), "Event ID should be 20 characters (including prefix)")
	assert.Equal(t, "EV", eventID[:2], "Event ID should start with EV")
}

func TestGenerateEventID_Format(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	tenantID := uuid.New()
	senderIP := "192.168.1.1"

	ctx := context.Background()
	eventID, err := idGen.GenerateEventID(ctx, tenantID, senderIP)

	require.NoError(t, err)

	// Parse the ID
	parsed, err := ParseEventID(eventID)
	require.NoError(t, err)

	// Verify timestamp is recent (within last minute)
	now := time.Now()
	assert.WithinDuration(t, now, parsed.Timestamp, time.Minute)

	// Verify tenant hash is 4 digits
	assert.Len(t, parsed.TenantHash, 4)

	// Verify sequence is 1
	assert.Equal(t, 1, parsed.Sequence, "First event should have sequence 1")
}

func TestGenerateEventID_Sequential(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	tenantID := uuid.New()
	senderIP := "192.168.1.1"

	ctx := context.Background()

	// Generate 3 events
	id1, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
	require.NoError(t, err)

	id2, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
	require.NoError(t, err)

	id3, err := idGen.GenerateEventID(ctx, tenantID, senderIP)
	require.NoError(t, err)

	// Parse sequences
	parsed1, _ := ParseEventID(id1)
	parsed2, _ := ParseEventID(id2)
	parsed3, _ := ParseEventID(id3)

	assert.Equal(t, 1, parsed1.Sequence)
	assert.Equal(t, 2, parsed2.Sequence)
	assert.Equal(t, 3, parsed3.Sequence)
}

func TestGenerateMessageID(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	eventCustomID := "EV260323120012340001"

	ctx := context.Background()
	messageID, err := idGen.GenerateMessageID(ctx, eventCustomID)

	require.NoError(t, err)
	assert.NotEmpty(t, messageID)
	assert.Equal(t, 23, len(messageID), "Message ID should be 23 characters (including prefix)")
	assert.Equal(t, "MS", messageID[:2], "Message ID should start with MS")
}

func TestGenerateMessageID_Format(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	eventCustomID, _ := idGen.GenerateEventID(context.Background(), uuid.New(), "127.0.0.1")

	ctx := context.Background()
	messageID, err := idGen.GenerateMessageID(ctx, eventCustomID)

	require.NoError(t, err)

	// Parse the ID
	parsed, err := ParseMessageID(messageID)
	require.NoError(t, err)

	// Verify timestamp is recent
	now := time.Now()
	assert.WithinDuration(t, now, parsed.Timestamp, time.Minute)

	// Verify tenant hash is 4 digits
	assert.Len(t, parsed.TenantHash, 4)

	// Verify event sequence is correct
	assert.Equal(t, 1, parsed.EventSeq)

	// Verify message sequence is 1
	assert.Equal(t, 1, parsed.MessageSeq, "First message should have sequence 1")
}

func TestGenerateMessageID_Sequential(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	eventCustomID, _ := idGen.GenerateEventID(context.Background(), uuid.New(), "127.0.0.1")

	ctx := context.Background()

	// Generate 3 messages
	id1, err := idGen.GenerateMessageID(ctx, eventCustomID)
	require.NoError(t, err)

	id2, err := idGen.GenerateMessageID(ctx, eventCustomID)
	require.NoError(t, err)

	id3, err := idGen.GenerateMessageID(ctx, eventCustomID)
	require.NoError(t, err)

	// Parse sequences
	parsed1, _ := ParseMessageID(id1)
	parsed2, _ := ParseMessageID(id2)
	parsed3, _ := ParseMessageID(id3)

	assert.Equal(t, 1, parsed1.MessageSeq)
	assert.Equal(t, 2, parsed2.MessageSeq)
	assert.Equal(t, 3, parsed3.MessageSeq)
}

func TestGenerateMessageID_Limit999(t *testing.T) {
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	idGen := NewIDGenerator(redisClient)
	eventCustomID := "EV260323120012340001"

	ctx := context.Background()

	// Set counter to 998 (next will be 999, then 1000 should fail)
	key := MessageCounterKey(eventCustomID)
	mr.Set(key, "998")

	// Generate message 999 (should succeed)
	id999, err := idGen.GenerateMessageID(ctx, eventCustomID)
	require.NoError(t, err)
	parsed999, _ := ParseMessageID(id999)
	assert.Equal(t, 999, parsed999.MessageSeq)

	// Generate message 1000 (should fail)
	_, err = idGen.GenerateMessageID(ctx, eventCustomID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message limit exceeded")
}

func TestParseEventID(t *testing.T) {
	tests := []struct {
		name      string
		eventID   string
		wantError bool
	}{
		{
			name:      "Valid event ID",
			eventID:   "EV260323120012340001",
			wantError: false,
		},
		{
			name:      "Invalid prefix",
			eventID:   "XX260323120012340001",
			wantError: true,
		},
		{
			name:      "Too short",
			eventID:   "EV26010123",
			wantError: true,
		},
		{
			name:      "Too long",
			eventID:   "EV26010123123450001001234",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseEventID(tt.eventID)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsed)
			}
		})
	}
}

func TestParseMessageID(t *testing.T) {
	tests := []struct {
		name      string
		messageID string
		wantError bool
	}{
		{
			name:      "Valid message ID",
			messageID: "MS260323120012340001001",
			wantError: false,
		},
		{
			name:      "Invalid prefix",
			messageID: "XX260323120012340001001",
			wantError: true,
		},
		{
			name:      "Too short",
			messageID: "MS26010123",
			wantError: true,
		},
		{
			name:      "Too long",
			messageID: "MS260101231234500010011234",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseMessageID(tt.messageID)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsed)
			}
		})
	}
}

func TestGenerateTenantHash(t *testing.T) {
	tenantID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	hash := generateTenantHash(tenantID)

	assert.Len(t, hash, 5, "Tenant hash should be 5 digits")
	assert.Regexp(t, "^[0-9]{5}$", hash, "Tenant hash should contain only digits")
}

func TestGenerateTenantHash_Consistency(t *testing.T) {
	tenantID := uuid.New()

	// Generate hash multiple times
	hash1 := generateTenantHash(tenantID)
	hash2 := generateTenantHash(tenantID)
	hash3 := generateTenantHash(tenantID)

	// Should be consistent
	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
}

func TestGenerateTenantHash_Uniqueness(t *testing.T) {
	// Generate hashes for different tenants. Use a deterministic pseudo-random
	// LCG seeded with a constant so the inputs look like real (high-entropy)
	// UUIDs yet the collision count is reproducible rather than flaky.
	hashes := make(map[string]bool)
	collisions := 0

	state := uint64(0x9E3779B97F4A7C15)
	for i := 0; i < 1000; i++ {
		state = state*6364136223846793005 + 1442695040888963407
		var raw [16]byte
		binary.BigEndian.PutUint64(raw[0:], state)
		tenantID := uuid.UUID(raw)
		hash := generateTenantHash(tenantID)

		if hashes[hash] {
			collisions++
		}
		hashes[hash] = true
	}

	// Some collisions are expected with 5 digits (100,000 possible values)
	// but should be rare for 1000 samples
	assert.Less(t, collisions, 10, "Too many hash collisions")
}

func BenchmarkGenerateEventID(b *testing.B) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	redisClient := &Client{Client: client}
	idGen := NewIDGenerator(redisClient)
	tenantID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idGen.GenerateEventID(ctx, tenantID, "192.168.1.1")
	}
}

func BenchmarkGenerateMessageID(b *testing.B) {
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		b.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	redisClient := &Client{Client: client}
	idGen := NewIDGenerator(redisClient)
	eventCustomID := "EV260323120012340001"
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = idGen.GenerateMessageID(ctx, eventCustomID)
	}
}
