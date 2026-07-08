package rollover

import (
	"time"

	"github.com/google/uuid"
)

// RolloverMessage represents a rollover message record
type RolloverMessage struct {
	ID                int64      `json:"id"`
	SubscriptionID    int64      `json:"subscription_id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	MessagesCount     int64      `json:"messages_count"`
	SourceMonth       time.Time  `json:"source_month"`
	ExpiryDate        time.Time  `json:"expiry_date"`
	Status            string     `json:"status"`
	PaymentStatus     string     `json:"payment_status"`
	PaymentVerifiedAt *time.Time `json:"payment_verified_at"`
	ForfeitedReason   string     `json:"forfeited_reason"`
	ForfeitedAt       *time.Time `json:"forfeited_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
