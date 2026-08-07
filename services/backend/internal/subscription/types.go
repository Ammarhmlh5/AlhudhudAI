package subscription

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	StatusActive      SubscriptionStatus = "active"
	StatusGracePeriod SubscriptionStatus = "grace_period"
	StatusSuspended   SubscriptionStatus = "suspended"
	StatusCancelled   SubscriptionStatus = "cancelled"
	StatusExpired     SubscriptionStatus = "expired"
)

// Subscription represents a customer subscription to a package
type Subscription struct {
	ID                   int64              `json:"id"`
	TenantID             uuid.UUID          `json:"tenant_id"`
	PackageID            int64              `json:"package_id"`
	StartDate            time.Time          `json:"start_date"`
	EndDate              time.Time          `json:"end_date"`
	MessagesUsed         int64              `json:"messages_used"`
	MessagesRemaining    *int64             `json:"messages_remaining"` // NULL for unlimited
	OverageMessages      int64              `json:"overage_messages"`
	OverageCost          decimal.Decimal    `json:"overage_cost"`
	Status               SubscriptionStatus `json:"status"`
	AutoRenew            bool               `json:"auto_renew"`
	GracePeriodDays      int                `json:"grace_period_days"`
	LastRenewedAt        *time.Time         `json:"last_renewed_at"`
	NextRenewalDate      *time.Time         `json:"next_renewal_date"`
	RenewalAttempts      int                `json:"renewal_attempts"`
	LastRenewalAttempt   *time.Time         `json:"last_renewal_attempt"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

// Package represents a subscription package
type Package struct {
	ID                   int64           `json:"id"`
	TenantID             uuid.UUID       `json:"tenant_id"`
	ServiceID            int64           `json:"service_id"`
	Name                 string          `json:"name"`
	Code                 string          `json:"code"`
	PriceUSD             decimal.Decimal `json:"price_usd"`
	IncludedMessages     *int64          `json:"included_messages"` // NULL for unlimited
	OverageRate          decimal.Decimal `json:"overage_rate"`
	IsUnlimited          bool            `json:"is_unlimited"`
	BillingCycle         string          `json:"billing_cycle"`
	RolloverEnabled      bool            `json:"rollover_enabled"`
	RolloverLimit        *int64          `json:"rollover_limit"`
	RolloverExpiryMonths *int            `json:"rollover_expiry_months"`
	IsActive             bool            `json:"is_active"`
	DisplayOrder         int             `json:"display_order"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// SubscribeRequest represents a request to subscribe to a package
type SubscribeRequest struct {
	TenantID       uuid.UUID  `json:"tenant_id"`
	PackageID      int64      `json:"package_id"`
	AutoRenew      bool       `json:"auto_renew"`
	PromoCode      string     `json:"promo_code,omitempty"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	IPAddress      string     `json:"ip_address,omitempty"`
}

// SubscribeResult represents the result of a subscription operation
type SubscribeResult struct {
	Success        bool           `json:"success"`
	SubscriptionID int64          `json:"subscription_id,omitempty"`
	Message        string         `json:"message,omitempty"`
	Error          error          `json:"error,omitempty"`
	OldBalance     decimal.Decimal `json:"old_balance,omitempty"`
	NewBalance     decimal.Decimal `json:"new_balance,omitempty"`
}

// DeductMessagesRequest represents a request to deduct messages from subscription
type DeductMessagesRequest struct {
	SubscriptionID int64  `json:"subscription_id"`
	MessageCount   int64  `json:"message_count"`
	MessageID      string `json:"message_id"`
}

// DeductMessagesResult represents the result of message deduction
type DeductMessagesResult struct {
	Success           bool    `json:"success"`
	MessagesDeducted  int64   `json:"messages_deducted"`
	OverageMessages   int64   `json:"overage_messages"`
	OverageCost       decimal.Decimal `json:"overage_cost"`
	MessagesRemaining *int64  `json:"messages_remaining"` // NULL for unlimited
	Error             error   `json:"error,omitempty"`
}

// UsageInfo represents subscription usage information
type UsageInfo struct {
	SubscriptionID    int64              `json:"subscription_id"`
	PackageID         int64              `json:"package_id"`
	PackageName       string             `json:"package_name"`
	MessagesUsed      int64              `json:"messages_used"`
	MessagesRemaining *int64             `json:"messages_remaining"` // NULL for unlimited
	IncludedMessages  *int64             `json:"included_messages"`  // NULL for unlimited
	OverageMessages   int64              `json:"overage_messages"`
	OverageCost       decimal.Decimal    `json:"overage_cost"`
	OverageRate       decimal.Decimal    `json:"overage_rate"`
	Status            SubscriptionStatus `json:"status"`
	StartDate         time.Time          `json:"start_date"`
	EndDate           time.Time          `json:"end_date"`
	NextRenewalDate   *time.Time         `json:"next_renewal_date"`
}
