package referral

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// ErrReferralNotFound is returned when a referral is not found
	ErrReferralNotFound = errors.New("referral not found")
	
	// ErrInvalidReferralCode is returned when a referral code is invalid
	ErrInvalidReferralCode = errors.New("invalid referral code")
	
	// ErrReferralCodeExpired is returned when a referral code has expired
	ErrReferralCodeExpired = errors.New("referral code has expired")
	
	// ErrReferralCodeMaxUsage is returned when a referral code has reached max usage
	ErrReferralCodeMaxUsage = errors.New("referral code has reached maximum usage")
	
	// ErrSelfReferral is returned when trying to refer oneself
	ErrSelfReferral = errors.New("cannot refer yourself")
	
	// ErrAlreadyReferred is returned when a user has already been referred
	ErrAlreadyReferred = errors.New("user has already been referred")
	
	// ErrConditionNotMet is returned when trying to grant reward before condition is met
	ErrConditionNotMet = errors.New("reward condition not met")
	
	// ErrRewardAlreadyGranted is returned when trying to grant reward that was already granted
	ErrRewardAlreadyGranted = errors.New("reward already granted")
)

// Manager handles referral operations
type Manager struct {
	db          *sqlx.DB
	config      *ReferralConfig
	redisClient *goredis.Client
	cacheTTL    time.Duration
}

// NewManager creates a new referral manager
func NewManager(db *sqlx.DB, config *ReferralConfig) *Manager {
	if config == nil {
		config = &ReferralConfig{
			DefaultRewardType:      RewardTypeBalance,
			DefaultRewardAmount:    10.0,
			DefaultRewardCondition: RewardConditionFirstPayment,
			CodePrefix:             "REF",
			CodeLength:             8,
			MaxReferralsPerUser:    100,
			Enabled:                true,
		}
	}
	
	return &Manager{
		db:          db,
		config:      config,
		redisClient: nil, // Will be set via SetRedisClient if available
		cacheTTL:    5 * time.Minute, // Default 5 minutes cache TTL
	}
}

// SetRedisClient sets the Redis client for caching (optional)
func (m *Manager) SetRedisClient(client *goredis.Client) {
	m.redisClient = client
}

// GenerateReferralCode generates a unique referral code for a tenant
// Requirement 33.1: THE النظام SHALL يولّد كود إحالة فريد لكل عميل
func (m *Manager) GenerateReferralCode(ctx context.Context, tenantID uuid.UUID) (string, error) {
	if !m.config.Enabled {
		return "", errors.New("referral system is disabled")
	}
	
	// Generate random bytes
	randomBytes := make([]byte, m.config.CodeLength/2)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random code: %w", err)
	}
	
	// Convert to hex and uppercase
	randomStr := strings.ToUpper(hex.EncodeToString(randomBytes))
	
	// Create code with prefix
	code := fmt.Sprintf("%s-%s", m.config.CodePrefix, randomStr)
	
	return code, nil
}

// ValidateReferralCode validates a referral code
// Requirement 33.2: THE النظام SHALL يسمح للعميل بمشاركة كود الإحالة
// Requirement 2.3: Cache valid codes for 5 minutes to reduce database queries
func (m *Manager) ValidateReferralCode(ctx context.Context, code string) (*ReferralCodeInfo, error) {
	if !m.config.Enabled {
		return nil, errors.New("referral system is disabled")
	}
	
	// Basic validation
	if code == "" {
		return nil, ErrInvalidReferralCode
	}
	
	// Try to get from cache first (if Redis is available)
	cacheKey := fmt.Sprintf("referral:code:%s", code)
	if m.redisClient != nil {
		cachedData, err := m.redisClient.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			// Cache hit - parse and return
			var info ReferralCodeInfo
			if err := json.Unmarshal([]byte(cachedData), &info); err == nil {
				return &info, nil
			}
			// If unmarshal fails, continue to database query
		}
	}
	
	// Cache miss or Redis unavailable - query database
	query := `
		SELECT 
			r.referrer_tenant_id,
			COUNT(*) as usage_count
		FROM referrals r
		WHERE r.referral_code = $1
		GROUP BY r.referrer_tenant_id
	`
	
	var referrerID uuid.UUID
	var usageCount int
	
	err := m.db.QueryRowContext(ctx, query, code).Scan(&referrerID, &usageCount)
	if err != nil {
		if err == sql.ErrNoRows {
			// Code doesn't exist yet - it's valid for first use
			// We need to find the tenant who owns this code
			// For now, we'll return an error - the code should be generated first
			return nil, ErrInvalidReferralCode
		}
		return nil, fmt.Errorf("failed to validate referral code: %w", err)
	}
	
	info := &ReferralCodeInfo{
		Code:             code,
		ReferrerTenantID: referrerID,
		IsValid:          true,
		UsageCount:       usageCount,
	}
	
	// Check max usage if configured
	if m.config.MaxReferralsPerUser > 0 && usageCount >= m.config.MaxReferralsPerUser {
		info.IsValid = false
		
		// Cache the invalid result (shorter TTL)
		if m.redisClient != nil {
			if data, err := json.Marshal(info); err == nil {
				m.redisClient.Set(ctx, cacheKey, data, 1*time.Minute)
			}
		}
		
		return info, ErrReferralCodeMaxUsage
	}
	
	// Cache the valid result for 5 minutes
	if m.redisClient != nil {
		if data, err := json.Marshal(info); err == nil {
			m.redisClient.Set(ctx, cacheKey, data, m.cacheTTL)
		}
	}
	
	return info, nil
}

// InvalidateReferralCodeCache invalidates the cache for a referral code
// Called when code is used or expires
func (m *Manager) InvalidateReferralCodeCache(ctx context.Context, code string) error {
	if m.redisClient == nil {
		return nil // No cache to invalidate
	}
	
	cacheKey := fmt.Sprintf("referral:code:%s", code)
	return m.redisClient.Del(ctx, cacheKey).Err()
}

// ProcessReferral processes a new referral when a user signs up with a referral code
// Requirement 33.3: WHEN يشترك عميل جديد باستخدام كود إحالة، THE النظام SHALL يسجل العلاقة
func (m *Manager) ProcessReferral(ctx context.Context, req *CreateReferralRequest) (*Referral, error) {
	if !m.config.Enabled {
		return nil, errors.New("referral system is disabled")
	}
	
	// Validate request
	if req.ReferrerTenantID == uuid.Nil || req.RefereeTenantID == uuid.Nil {
		return nil, errors.New("invalid tenant IDs")
	}
	
	// Check self-referral
	if req.ReferrerTenantID == req.RefereeTenantID {
		return nil, ErrSelfReferral
	}
	
	// Check if referee has already been referred
	var existingCount int
	err := m.db.GetContext(ctx, &existingCount, 
		"SELECT COUNT(*) FROM referrals WHERE referee_tenant_id = $1", 
		req.RefereeTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing referrals: %w", err)
	}
	if existingCount > 0 {
		return nil, ErrAlreadyReferred
	}
	
	// Use default values if not provided
	rewardType := m.config.DefaultRewardType
	if req.RewardType != nil {
		rewardType = *req.RewardType
	}
	
	rewardAmount := m.config.DefaultRewardAmount
	if req.RewardAmount != nil {
		rewardAmount = *req.RewardAmount
	}
	
	rewardCondition := m.config.DefaultRewardCondition
	if req.RewardCondition != nil {
		rewardCondition = *req.RewardCondition
	}
	
	// Create referral
	referral := &Referral{
		ReferrerTenantID: req.ReferrerTenantID,
		RefereeTenantID:  req.RefereeTenantID,
		ReferralCode:     req.ReferralCode,
		Status:           ReferralStatusPending,
		RewardType:       &rewardType,
		RewardAmount:     &rewardAmount,
		RewardCondition:  &rewardCondition,
		RewardGranted:    false,
		ConditionMet:     false,
		Notes:            req.Notes,
	}
	
	// Check if condition is immediate
	if rewardCondition == RewardConditionImmediate {
		referral.ConditionMet = true
		now := time.Now()
		referral.ConditionMetAt = &now
		referral.Status = ReferralStatusCompleted
	}
	
	// Insert into database
	query := `
		INSERT INTO referrals (
			referrer_tenant_id,
			referee_tenant_id,
			referral_code,
			status,
			reward_type,
			reward_amount,
			reward_condition,
			condition_met,
			condition_met_at,
			notes
		) VALUES (
			:referrer_tenant_id,
			:referee_tenant_id,
			:referral_code,
			:status,
			:reward_type,
			:reward_amount,
			:reward_condition,
			:condition_met,
			:condition_met_at,
			:notes
		) RETURNING id, created_at, updated_at
	`
	
	rows, err := m.db.NamedQueryContext(ctx, query, referral)
	if err != nil {
		return nil, fmt.Errorf("failed to create referral: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&referral.ID, &referral.CreatedAt, &referral.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan referral: %w", err)
		}
	}
	
	// Invalidate cache for this referral code since usage count changed
	if err := m.InvalidateReferralCodeCache(ctx, req.ReferralCode); err != nil {
		// Log error but don't fail the operation
		logger.GetLogger().Warn("Failed to invalidate referral code cache",
			zap.String("code", req.ReferralCode),
			zap.Error(err),
		)
	}
	
	return referral, nil
}

// MarkConditionMet marks the referral condition as met
// Requirement 33.6: THE النظام SHALL يدعم تحديد شروط المكافأة
func (m *Manager) MarkConditionMet(ctx context.Context, referralID uuid.UUID) error {
	query := `
		UPDATE referrals
		SET 
			condition_met = true,
			condition_met_at = CURRENT_TIMESTAMP,
			status = 'completed',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND condition_met = false
	`
	
	result, err := m.db.ExecContext(ctx, query, referralID)
	if err != nil {
		return fmt.Errorf("failed to mark condition met: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return ErrReferralNotFound
	}
	
	return nil
}

// GrantReward grants the reward to the referrer
// Requirement 33.4: THE النظام SHALL يمنح العميل المُحيل مكافأة عند اشتراك المُحال
// Requirement 33.5: THE النظام SHALL يدعم تخصيص نوع المكافأة
func (m *Manager) GrantReward(ctx context.Context, req *GrantRewardRequest) error {
	if !m.config.Enabled {
		return errors.New("referral system is disabled")
	}
	
	// Get referral
	referral, err := m.GetReferralByID(ctx, req.ReferralID)
	if err != nil {
		return err
	}
	
	// Check if condition is met
	if !referral.ConditionMet {
		return ErrConditionNotMet
	}
	
	// Check if reward already granted
	if referral.RewardGranted {
		return ErrRewardAlreadyGranted
	}
	
	// Start transaction
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Grant reward based on type
	if referral.RewardType != nil && referral.RewardAmount != nil {
		switch *referral.RewardType {
		case RewardTypeBalance:
			// Add balance to referrer
			err = m.grantBalanceReward(ctx, tx, referral.ReferrerTenantID, *referral.RewardAmount)
			if err != nil {
				return fmt.Errorf("failed to grant balance reward: %w", err)
			}
			
		case RewardTypeMessages:
			// Add free messages to referrer
			err = m.grantMessagesReward(ctx, tx, referral.ReferrerTenantID, int(*referral.RewardAmount))
			if err != nil {
				return fmt.Errorf("failed to grant messages reward: %w", err)
			}
			
		case RewardTypeDiscount:
			// Create discount code for referrer
			err = m.grantDiscountReward(ctx, tx, referral.ReferrerTenantID, *referral.RewardAmount)
			if err != nil {
				return fmt.Errorf("failed to grant discount reward: %w", err)
			}
		}
	}
	
	// Update referral status
	query := `
		UPDATE referrals
		SET 
			reward_granted = true,
			reward_granted_at = CURRENT_TIMESTAMP,
			status = 'rewarded',
			notes = COALESCE(notes, '') || COALESCE($2, ''),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	
	_, err = tx.ExecContext(ctx, query, req.ReferralID, req.Notes)
	if err != nil {
		return fmt.Errorf("failed to update referral: %w", err)
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// grantBalanceReward adds balance to the referrer's account
func (m *Manager) grantBalanceReward(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, amount float64) error {
	// Update tenant balance
	query := `
		UPDATE tenants
		SET balance = balance + $1
		WHERE id = $2
	`
	
	_, err := tx.ExecContext(ctx, query, amount, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	
	// Record in accounting_transactions
	transQuery := `
		INSERT INTO accounting_transactions (
			tenant_id,
			transaction_type,
			amount,
			currency,
			description,
			status
		) VALUES (
			$1, 'REFERRAL_REWARD', $2, 'USD', 'Referral reward', 'completed'
		)
	`
	
	_, err = tx.ExecContext(ctx, transQuery, tenantID, amount)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}
	
	return nil
}

// grantMessagesReward adds free messages to the referrer's account
func (m *Manager) grantMessagesReward(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, messageCount int) error {
	// This would integrate with the subscription system to add free messages
	// For now, we'll just record it in the metadata
	
	// TODO: Integrate with subscription system
	// For example: add to messages_remaining in subscriptions table
	
	return nil
}

// grantDiscountReward creates a discount code for the referrer
func (m *Manager) grantDiscountReward(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, discountPercent float64) error {
	// This would integrate with the promo code system
	// For now, we'll just record it
	
	// TODO: Integrate with promo code system
	// For example: create a promo code with the discount percentage
	
	return nil
}

// GetReferralByID retrieves a referral by ID
func (m *Manager) GetReferralByID(ctx context.Context, id uuid.UUID) (*Referral, error) {
	var referral Referral
	
	query := `
		SELECT * FROM referrals WHERE id = $1
	`
	
	err := m.db.GetContext(ctx, &referral, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrReferralNotFound
		}
		return nil, fmt.Errorf("failed to get referral: %w", err)
	}
	
	return &referral, nil
}

// GetReferralsByReferrer retrieves all referrals for a referrer
// Requirement 33.7: THE النظام SHALL يعرض للعميل عدد الإحالات الناجحة والمكافآت المكتسبة
func (m *Manager) GetReferralsByReferrer(ctx context.Context, referrerID uuid.UUID, limit, offset int) ([]*Referral, error) {
	var referrals []*Referral
	
	query := `
		SELECT * FROM referrals 
		WHERE referrer_tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	err := m.db.SelectContext(ctx, &referrals, query, referrerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get referrals: %w", err)
	}
	
	return referrals, nil
}

// GetReferralStats retrieves statistics for a referrer
// Requirement 33.7: THE النظام SHALL يعرض للعميل عدد الإحالات الناجحة والمكافآت المكتسبة
func (m *Manager) GetReferralStats(ctx context.Context, referrerID uuid.UUID) (*ReferralStats, error) {
	var stats ReferralStats
	
	query := `
		SELECT * FROM referral_stats WHERE referrer_tenant_id = $1
	`
	
	err := m.db.GetContext(ctx, &stats, query, referrerID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No referrals yet
			return &ReferralStats{
				ReferrerTenantID: referrerID,
			}, nil
		}
		return nil, fmt.Errorf("failed to get referral stats: %w", err)
	}
	
	return &stats, nil
}

// GetReferralPerformance retrieves overall performance metrics
// Requirement 33.9: THE النظام SHALL يوفر تقرير بأداء برنامج الإحالة
func (m *Manager) GetReferralPerformance(ctx context.Context) (*ReferralPerformance, error) {
	var perf ReferralPerformance
	
	query := `
		SELECT 
			COUNT(*) as total_referrals,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_referrals,
			COUNT(*) FILTER (WHERE status = 'rewarded') as rewarded_referrals,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_referrals,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_referrals,
			COALESCE(SUM(CASE WHEN reward_granted THEN reward_amount ELSE 0 END), 0) as total_rewards_granted
		FROM referrals
	`
	
	err := m.db.GetContext(ctx, &perf, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get referral performance: %w", err)
	}
	
	// Calculate rates
	if perf.TotalReferrals > 0 {
		perf.ConversionRate = float64(perf.CompletedReferrals) / float64(perf.TotalReferrals) * 100
	}
	
	if perf.CompletedReferrals > 0 {
		perf.RewardRate = float64(perf.RewardedReferrals) / float64(perf.CompletedReferrals) * 100
	}
	
	if perf.RewardedReferrals > 0 {
		perf.AverageReward = perf.TotalRewardsGranted / float64(perf.RewardedReferrals)
	}
	
	return &perf, nil
}

// ProcessPendingRewards processes all pending rewards (for cron job)
// Requirement 33.8: THE النظام SHALL يسجل جميع عمليات الإحالة والمكافآت
func (m *Manager) ProcessPendingRewards(ctx context.Context) (int, error) {
	// Get all referrals where condition is met but reward not granted
	query := `
		SELECT id FROM referrals
		WHERE condition_met = true 
		  AND reward_granted = false
		  AND status = 'completed'
		ORDER BY condition_met_at ASC
		LIMIT 100
	`
	
	var referralIDs []uuid.UUID
	err := m.db.SelectContext(ctx, &referralIDs, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending rewards: %w", err)
	}
	
	processed := 0
	for _, id := range referralIDs {
		err = m.GrantReward(ctx, &GrantRewardRequest{ReferralID: id})
		if err != nil {
			// Log error but continue processing
			logger.GetLogger().Error("Failed to grant reward for referral",
				zap.String("referral_id", id.String()),
				zap.Error(err),
			)

			continue
		}
		processed++
	}
	
	return processed, nil
}
