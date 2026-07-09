package promo

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// UsageTracker متتبع استخدام أكواد الخصم
// Requirements: 32.9, 32.10
type UsageTracker struct {
	db *sqlx.DB
}

// NewUsageTracker ينشئ متتبع استخدام جديد
func NewUsageTracker(db *sqlx.DB) *UsageTracker {
	return &UsageTracker{
		db: db,
	}
}

// RecordUsage يسجل استخدام كود خصم
// Requirements: 32.9
func (ut *UsageTracker) RecordUsage(ctx context.Context, usage *PromoCodeUsage) error {
	query := `
		INSERT INTO promo_code_usage (
			tenant_id, promo_code_id, user_id, order_type, order_id,
			original_amount, discount_amount, final_amount
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, used_at
	`

	err := ut.db.QueryRowContext(
		ctx, query,
		usage.TenantID, usage.PromoCodeID, usage.UserID,
		usage.OrderType, usage.OrderID,
		usage.OriginalAmount, usage.DiscountAmount, usage.FinalAmount,
	).Scan(&usage.ID, &usage.UsedAt)

	if err != nil {
		return fmt.Errorf("failed to record promo code usage: %w", err)
	}

	return nil
}

// GetUserUsageCount يحصل على عدد مرات استخدام المستخدم لكود معين
// Requirements: 32.10
func (ut *UsageTracker) GetUserUsageCount(ctx context.Context, tenantID int64, promoCodeID int64, userID int64) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM promo_code_usage 
		WHERE tenant_id = $1 AND promo_code_id = $2 AND user_id = $3
	`

	var count int
	err := ut.db.GetContext(ctx, &count, query, tenantID, promoCodeID, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user usage count: %w", err)
	}

	return count, nil
}

// GetUserUsageHistory يحصل على سجل استخدام المستخدم لكود معين
// Requirements: 32.10
func (ut *UsageTracker) GetUserUsageHistory(ctx context.Context, tenantID int64, promoCodeID int64, userID int64) ([]*PromoCodeUsage, error) {
	query := `
		SELECT 
			id, tenant_id, promo_code_id, user_id, order_type, order_id,
			original_amount, discount_amount, final_amount, used_at
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2 AND user_id = $3
		ORDER BY used_at DESC
	`

	var usages []*PromoCodeUsage
	err := ut.db.SelectContext(ctx, &usages, query, tenantID, promoCodeID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user usage history: %w", err)
	}

	return usages, nil
}

// HasUserUsedPromoCode يتحقق من استخدام المستخدم لكود معين
// Requirements: 32.8, 32.10
func (ut *UsageTracker) HasUserUsedPromoCode(ctx context.Context, tenantID int64, promoCodeID int64, userID int64) (bool, error) {
	count, err := ut.GetUserUsageCount(ctx, tenantID, promoCodeID, userID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CanUserUsePromoCode يتحقق من إمكانية استخدام المستخدم لكود معين
// Requirements: 32.8, 32.10
func (ut *UsageTracker) CanUserUsePromoCode(ctx context.Context, tenantID int64, promoCodeID int64, userID int64, perUserLimit int) (bool, error) {
	count, err := ut.GetUserUsageCount(ctx, tenantID, promoCodeID, userID)
	if err != nil {
		return false, err
	}

	return count < perUserLimit, nil
}

// GetPromoCodeUsageStats يحصل على إحصائيات استخدام كود خصم
// Requirements: 32.11
func (ut *UsageTracker) GetPromoCodeUsageStats(ctx context.Context, tenantID int64, promoCodeID int64) (*PromoCodeUsageStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_uses,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(SUM(original_amount), 0) as total_original_amount,
			COALESCE(SUM(discount_amount), 0) as total_discount_given,
			COALESCE(SUM(final_amount), 0) as total_final_amount,
			COALESCE(AVG(discount_amount), 0) as avg_discount_amount,
			MIN(used_at) as first_used_at,
			MAX(used_at) as last_used_at
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2
	`

	var stats PromoCodeUsageStats
	err := ut.db.GetContext(ctx, &stats, query, tenantID, promoCodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get promo code usage stats: %w", err)
	}

	stats.TenantID = tenantID
	stats.PromoCodeID = promoCodeID

	return &stats, nil
}

// GetUsageByDateRange يحصل على استخدامات كود خصم في فترة زمنية
// Requirements: 32.11
func (ut *UsageTracker) GetUsageByDateRange(ctx context.Context, tenantID int64, promoCodeID int64, startDate, endDate time.Time) ([]*PromoCodeUsage, error) {
	query := `
		SELECT 
			id, tenant_id, promo_code_id, user_id, order_type, order_id,
			original_amount, discount_amount, final_amount, used_at
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2 
		  AND used_at >= $3 AND used_at <= $4
		ORDER BY used_at DESC
	`

	var usages []*PromoCodeUsage
	err := ut.db.SelectContext(ctx, &usages, query, tenantID, promoCodeID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage by date range: %w", err)
	}

	return usages, nil
}

// GetUsageByOrderType يحصل على استخدامات كود خصم حسب نوع الطلب
// Requirements: 32.11
func (ut *UsageTracker) GetUsageByOrderType(ctx context.Context, tenantID int64, promoCodeID int64, orderType OrderType) ([]*PromoCodeUsage, error) {
	query := `
		SELECT 
			id, tenant_id, promo_code_id, user_id, order_type, order_id,
			original_amount, discount_amount, final_amount, used_at
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2 AND order_type = $3
		ORDER BY used_at DESC
	`

	var usages []*PromoCodeUsage
	err := ut.db.SelectContext(ctx, &usages, query, tenantID, promoCodeID, orderType)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage by order type: %w", err)
	}

	return usages, nil
}

// GetTopUsers يحصل على أكثر المستخدمين استخداماً لكود خصم
// Requirements: 32.11
func (ut *UsageTracker) GetTopUsers(ctx context.Context, tenantID int64, promoCodeID int64, limit int) ([]*UserUsageSummary, error) {
	query := `
		SELECT 
			user_id,
			COUNT(*) as usage_count,
			COALESCE(SUM(discount_amount), 0) as total_discount_received,
			COALESCE(SUM(final_amount), 0) as total_amount_paid,
			MIN(used_at) as first_used_at,
			MAX(used_at) as last_used_at
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2
		GROUP BY user_id
		ORDER BY usage_count DESC, total_discount_received DESC
		LIMIT $3
	`

	var users []*UserUsageSummary
	err := ut.db.SelectContext(ctx, &users, query, tenantID, promoCodeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}

	return users, nil
}

// GetDailyUsageStats يحصل على إحصائيات الاستخدام اليومية
// Requirements: 32.11
func (ut *UsageTracker) GetDailyUsageStats(ctx context.Context, tenantID int64, promoCodeID int64, days int) ([]*DailyUsageStats, error) {
	query := `
		SELECT 
			DATE(used_at) as usage_date,
			COUNT(*) as usage_count,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(SUM(discount_amount), 0) as total_discount,
			COALESCE(SUM(final_amount), 0) as total_revenue
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2 
		  AND used_at >= NOW() - INTERVAL '1 day' * $3
		GROUP BY DATE(used_at)
		ORDER BY usage_date DESC
	`

	var stats []*DailyUsageStats
	err := ut.db.SelectContext(ctx, &stats, query, tenantID, promoCodeID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage stats: %w", err)
	}

	return stats, nil
}

// PreventDuplicateUsage يمنع الاستخدام المتكرر لنفس الطلب
// Requirements: 32.8
func (ut *UsageTracker) PreventDuplicateUsage(ctx context.Context, tenantID int64, promoCodeID int64, orderType OrderType, orderID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM promo_code_usage 
			WHERE tenant_id = $1 AND promo_code_id = $2 
			  AND order_type = $3 AND order_id = $4
		)
	`

	var exists bool
	err := ut.db.GetContext(ctx, &exists, query, tenantID, promoCodeID, orderType, orderID)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate usage: %w", err)
	}

	return !exists, nil // true = يمكن الاستخدام، false = مستخدم بالفعل
}

// PromoCodeUsageStats إحصائيات استخدام كود خصم
type PromoCodeUsageStats struct {
	TenantID            int64     `db:"tenant_id" json:"tenant_id"`
	PromoCodeID         int64     `db:"promo_code_id" json:"promo_code_id"`
	TotalUses           int       `db:"total_uses" json:"total_uses"`
	UniqueUsers         int       `db:"unique_users" json:"unique_users"`
	TotalOriginalAmount float64   `db:"total_original_amount" json:"total_original_amount"`
	TotalDiscountGiven  float64   `db:"total_discount_given" json:"total_discount_given"`
	TotalFinalAmount    float64   `db:"total_final_amount" json:"total_final_amount"`
	AvgDiscountAmount   float64   `db:"avg_discount_amount" json:"avg_discount_amount"`
	FirstUsedAt         time.Time `db:"first_used_at" json:"first_used_at"`
	LastUsedAt          time.Time `db:"last_used_at" json:"last_used_at"`
}

// UserUsageSummary ملخص استخدام المستخدم
type UserUsageSummary struct {
	UserID                int64     `db:"user_id" json:"user_id"`
	UsageCount            int       `db:"usage_count" json:"usage_count"`
	TotalDiscountReceived float64   `db:"total_discount_received" json:"total_discount_received"`
	TotalAmountPaid       float64   `db:"total_amount_paid" json:"total_amount_paid"`
	FirstUsedAt           time.Time `db:"first_used_at" json:"first_used_at"`
	LastUsedAt            time.Time `db:"last_used_at" json:"last_used_at"`
}

// DailyUsageStats إحصائيات الاستخدام اليومية
type DailyUsageStats struct {
	UsageDate     time.Time `db:"usage_date" json:"usage_date"`
	UsageCount    int       `db:"usage_count" json:"usage_count"`
	UniqueUsers   int       `db:"unique_users" json:"unique_users"`
	TotalDiscount float64   `db:"total_discount" json:"total_discount"`
	TotalRevenue  float64   `db:"total_revenue" json:"total_revenue"`
}
