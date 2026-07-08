package rating

import (
	"time"

	"github.com/google/uuid"
)

// RatingCategory represents the classification of a company's rating
type RatingCategory string

const (
	// Excellent rating: >= 90%
	Excellent RatingCategory = "excellent"
	// Good rating: 75-89%
	Good RatingCategory = "good"
	// Average rating: 60-74%
	Average RatingCategory = "average"
	// Poor rating: < 60%
	Poor RatingCategory = "poor"
)

// CompanyRating represents a company's rating at a specific point in time
type CompanyRating struct {
	ID                   int64          `json:"id" db:"id"`
	TenantID             uuid.UUID      `json:"tenant_id" db:"tenant_id" validate:"required"`
	SuccessRate          float64        `json:"success_rate" db:"success_rate" validate:"gte=0,lte=100"`
	SendingVolume        int64          `json:"sending_volume" db:"sending_volume" validate:"gte=0"`
	PaymentCompliance    float64        `json:"payment_compliance" db:"payment_compliance" validate:"gte=0,lte=100"`
	ServiceQualityScore  float64        `json:"service_quality_score" db:"service_quality_score" validate:"gte=0,lte=100"`
	OverallRating        float64        `json:"overall_rating" db:"overall_rating" validate:"gte=0,lte=100"`
	RatingCategory       RatingCategory `json:"rating_category" db:"rating_category" validate:"required,oneof=excellent good average poor"`
	CalculationTimestamp time.Time      `json:"calculation_timestamp" db:"calculation_timestamp"`
	ValidFrom            time.Time      `json:"valid_from" db:"valid_from"`
	ValidTo              *time.Time     `json:"valid_to,omitempty" db:"valid_to"`
}

// PerformanceMetrics represents the performance metrics used to calculate ratings
type PerformanceMetrics struct {
	ID                     int64     `json:"id" db:"id"`
	TenantID               uuid.UUID `json:"tenant_id" db:"tenant_id" validate:"required"`
	PeriodStart            time.Time `json:"period_start" db:"period_start" validate:"required"`
	PeriodEnd              time.Time `json:"period_end" db:"period_end" validate:"required"`
	PeriodType             string    `json:"period_type" db:"period_type" validate:"required,oneof=hourly daily monthly"`
	TotalMessages          int64     `json:"total_messages" db:"total_messages" validate:"gte=0"`
	SuccessfulMessages     int64     `json:"successful_messages" db:"successful_messages" validate:"gte=0"`
	FailedMessages         int64     `json:"failed_messages" db:"failed_messages" validate:"gte=0"`
	SuccessRate            float64   `json:"success_rate" db:"success_rate"`
	AverageDeliveryTimeMs  *int      `json:"average_delivery_time_ms,omitempty" db:"average_delivery_time_ms"`
	P95DeliveryTimeMs      *int      `json:"p95_delivery_time_ms,omitempty" db:"p95_delivery_time_ms"`
	P99DeliveryTimeMs      *int      `json:"p99_delivery_time_ms,omitempty" db:"p99_delivery_time_ms"`
	ErrorCount             int       `json:"error_count" db:"error_count" validate:"gte=0"`
	ErrorRate              float64   `json:"error_rate" db:"error_rate"`
	TotalPayments          int       `json:"total_payments" db:"total_payments" validate:"gte=0"`
	OnTimePayments         int       `json:"on_time_payments" db:"on_time_payments" validate:"gte=0"`
	PaymentCompliance      float64   `json:"payment_compliance" db:"payment_compliance"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
}

// RatingWeights represents the weights used in rating calculation
type RatingWeights struct {
	SuccessRate       float64 `json:"success_rate" validate:"gte=0,lte=1"`
	SendingVolume     float64 `json:"sending_volume" validate:"gte=0,lte=1"`
	PaymentCompliance float64 `json:"payment_compliance" validate:"gte=0,lte=1"`
	ServiceQuality    float64 `json:"service_quality" validate:"gte=0,lte=1"`
}

// DefaultRatingWeights returns the default weights for rating calculation
func DefaultRatingWeights() RatingWeights {
	return RatingWeights{
		SuccessRate:       0.40, // 40%
		SendingVolume:     0.20, // 20%
		PaymentCompliance: 0.25, // 25%
		ServiceQuality:    0.15, // 15%
	}
}

// Validate checks if the weights sum to 1.0 (100%)
func (w RatingWeights) Validate() bool {
	sum := w.SuccessRate + w.SendingVolume + w.PaymentCompliance + w.ServiceQuality
	// Allow small floating point error (0.001)
	return sum >= 0.999 && sum <= 1.001
}

// RatingUpdateEvent represents an event that triggers a rating update
type RatingUpdateEvent struct {
	EventType string    `json:"event_type"` // "message_sent", "message_failed", "payment_received"
	TenantID  uuid.UUID `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
}

// RatingHistoryFilter represents filters for querying rating history
type RatingHistoryFilter struct {
	TenantID   uuid.UUID       `json:"tenant_id"`
	StartDate  *time.Time      `json:"start_date,omitempty"`
	EndDate    *time.Time      `json:"end_date,omitempty"`
	Category   *RatingCategory `json:"category,omitempty"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
}
