package batch

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BatchConfig represents batch processing configuration
type BatchConfig struct {
	DefaultBatchSize int           // 100
	MinBatchSize     int           // 10
	MaxBatchSize     int           // 1000
	ProcessInterval  time.Duration // 100ms
	AdaptiveMode     bool          // true
}

// DefaultBatchConfig returns default configuration
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		DefaultBatchSize: 100,
		MinBatchSize:     10,
		MaxBatchSize:     1000,
		ProcessInterval:  100 * time.Millisecond,
		AdaptiveMode:     true,
	}
}

// MessageForInsert represents a message ready for database insertion
type MessageForInsert struct {
	TenantID    string          `db:"tenant_id"`
	MessageID   string          `db:"message_id"`
	PhoneNumber string          `db:"phone_number"`
	Content     string          `db:"content"`
	Status      string          `db:"status"`
	Priority    string          `db:"priority"`
	Cost        decimal.Decimal `db:"cost"`
	PartsCount  int             `db:"parts_count"`
	QueuedAt    time.Time       `db:"queued_at"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

// BatchResult represents the result of batch processing
type BatchResult struct {
	TenantID         uuid.UUID
	ProcessedCount   int
	SuccessCount     int
	FailedCount      int
	ProcessingTimeMs int64
	Error            error
}
