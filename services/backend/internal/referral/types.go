package referral

import (
	"time"

	"github.com/google/uuid"
)

// ReferralStatus represents the status of a referral
type ReferralStatus string

const (
	// ReferralStatusPending indicates the referral is pending (referee registered but condition not met)
	ReferralStatusPending ReferralStatus = "pending"
	
	// ReferralStatusCompleted indicates the referral is completed (condition met)
	ReferralStatusCompleted ReferralStatus = "completed"
	
	// ReferralStatusRewarded indicates the reward has been granted
	ReferralStatusRewarded ReferralStatus = "rewarded"
	
	// ReferralStatusCancelled indicates the referral was cancelled
	ReferralStatusCancelled ReferralStatus = "cancelled"
)

// RewardType represents the type of reward
type RewardType string

const (
	// RewardTypeBalance indicates a balance reward (money)
	RewardTypeBalance RewardType = "balance"
	
	// RewardTypeMessages indicates free messages reward
	RewardTypeMessages RewardType = "messages"
	
	// RewardTypeDiscount indicates a discount reward (percentage)
	RewardTypeDiscount RewardType = "discount"
)

// RewardCondition represents the condition for granting the reward
type RewardCondition string

const (
	// RewardConditionImmediate grants reward immediately upon registration
	RewardConditionImmediate RewardCondition = "immediate"
	
	// RewardConditionFirstSubscription grants reward after first subscription
	RewardConditionFirstSubscription RewardCondition = "first_subscription"
	
	// RewardConditionFirstPayment grants reward after first payment
	RewardConditionFirstPayment RewardCondition = "first_payment"
)

// Referral represents a referral relationship
type Referral struct {
	ID                uuid.UUID        `json:"id" db:"id"`
	ReferrerTenantID  uuid.UUID        `json:"referrer_tenant_id" db:"referrer_tenant_id"`
	RefereeTenantID   uuid.UUID        `json:"referee_tenant_id" db:"referee_tenant_id"`
	ReferralCode      string           `json:"referral_code" db:"referral_code"`
	Status            ReferralStatus   `json:"status" db:"status"`
	RewardType        *RewardType      `json:"reward_type,omitempty" db:"reward_type"`
	RewardAmount      *float64         `json:"reward_amount,omitempty" db:"reward_amount"`
	RewardGranted     bool             `json:"reward_granted" db:"reward_granted"`
	RewardGrantedAt   *time.Time       `json:"reward_granted_at,omitempty" db:"reward_granted_at"`
	RewardCondition   *RewardCondition `json:"reward_condition,omitempty" db:"reward_condition"`
	ConditionMet      bool             `json:"condition_met" db:"condition_met"`
	ConditionMetAt    *time.Time       `json:"condition_met_at,omitempty" db:"condition_met_at"`
	Notes             *string          `json:"notes,omitempty" db:"notes"`
	Metadata          map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at" db:"updated_at"`
}

// ReferralStats represents statistics for a referrer
type ReferralStats struct {
	ReferrerTenantID     uuid.UUID  `json:"referrer_tenant_id" db:"referrer_tenant_id"`
	TotalReferrals       int        `json:"total_referrals" db:"total_referrals"`
	CompletedReferrals   int        `json:"completed_referrals" db:"completed_referrals"`
	RewardedReferrals    int        `json:"rewarded_referrals" db:"rewarded_referrals"`
	PendingReferrals     int        `json:"pending_referrals" db:"pending_referrals"`
	TotalRewardsGranted  float64    `json:"total_rewards_granted" db:"total_rewards_granted"`
	LastReferralDate     *time.Time `json:"last_referral_date,omitempty" db:"last_referral_date"`
}

// ReferralConfig represents the configuration for the referral system
type ReferralConfig struct {
	// Default reward settings
	DefaultRewardType      RewardType      `json:"default_reward_type"`
	DefaultRewardAmount    float64         `json:"default_reward_amount"`
	DefaultRewardCondition RewardCondition `json:"default_reward_condition"`
	
	// Code generation settings
	CodePrefix string `json:"code_prefix"`
	CodeLength int    `json:"code_length"`
	
	// Limits
	MaxReferralsPerUser int `json:"max_referrals_per_user"`
	
	// Enabled/Disabled
	Enabled bool `json:"enabled"`
}

// CreateReferralRequest represents a request to create a referral
type CreateReferralRequest struct {
	ReferrerTenantID uuid.UUID        `json:"referrer_tenant_id" validate:"required"`
	RefereeTenantID  uuid.UUID        `json:"referee_tenant_id" validate:"required"`
	ReferralCode     string           `json:"referral_code" validate:"required"`
	RewardType       *RewardType      `json:"reward_type,omitempty"`
	RewardAmount     *float64         `json:"reward_amount,omitempty"`
	RewardCondition  *RewardCondition `json:"reward_condition,omitempty"`
	Notes            *string          `json:"notes,omitempty"`
}

// GrantRewardRequest represents a request to grant a reward
type GrantRewardRequest struct {
	ReferralID uuid.UUID `json:"referral_id" validate:"required"`
	Notes      *string   `json:"notes,omitempty"`
}

// ReferralPerformance represents performance metrics for the referral program
type ReferralPerformance struct {
	TotalReferrals      int     `json:"total_referrals"`
	CompletedReferrals  int     `json:"completed_referrals"`
	RewardedReferrals   int     `json:"rewarded_referrals"`
	PendingReferrals    int     `json:"pending_referrals"`
	CancelledReferrals  int     `json:"cancelled_referrals"`
	TotalRewardsGranted float64 `json:"total_rewards_granted"`
	ConversionRate      float64 `json:"conversion_rate"` // completed / total
	RewardRate          float64 `json:"reward_rate"`     // rewarded / completed
	AverageReward       float64 `json:"average_reward"`
}

// ReferralCodeInfo represents information about a referral code
type ReferralCodeInfo struct {
	Code             string    `json:"code"`
	ReferrerTenantID uuid.UUID `json:"referrer_tenant_id"`
	IsValid          bool      `json:"is_valid"`
	UsageCount       int       `json:"usage_count"`
	MaxUsage         *int      `json:"max_usage,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}
