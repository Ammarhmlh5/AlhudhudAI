package promo

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ReportGenerator مولد تقارير أداء العروض
// Requirements: 32.11
type ReportGenerator struct {
	db *sqlx.DB
}

// NewReportGenerator ينشئ مولد تقارير جديد
func NewReportGenerator(db *sqlx.DB) *ReportGenerator {
	return &ReportGenerator{
		db: db,
	}
}

// PromoPerformanceReport تقرير أداء شامل لكود خصم
type PromoPerformanceReport struct {
	PromoCode           *PromoCode              `json:"promo_code"`
	UsageStats          *PromoCodeUsageStats    `json:"usage_stats"`
	TopUsers            []*UserUsageSummary     `json:"top_users"`
	DailyStats          []*DailyUsageStats      `json:"daily_stats"`
	UsageByOrderType    map[OrderType]int       `json:"usage_by_order_type"`
	ConversionRate      float64                 `json:"conversion_rate"`
	AverageOrderValue   float64                 `json:"average_order_value"`
	TotalRevenueLost    float64                 `json:"total_revenue_lost"`
	ROI                 float64                 `json:"roi"`
}

// AllPromosPerformanceReport تقرير أداء جميع الأكواد
type AllPromosPerformanceReport struct {
	TotalPromoCodes     int                     `json:"total_promo_codes"`
	ActivePromoCodes    int                     `json:"active_promo_codes"`
	TotalUsages         int                     `json:"total_usages"`
	TotalDiscountGiven  float64                 `json:"total_discount_given"`
	TotalRevenue        float64                 `json:"total_revenue"`
	UniqueUsers         int                     `json:"unique_users"`
	TopPerformingPromos []*PromoCodePerformance `json:"top_performing_promos"`
	RecentUsages        []*PromoCodeUsage       `json:"recent_usages"`
}

// GeneratePromoPerformanceReport يولد تقرير أداء شامل لكود خصم
// Requirements: 32.11
func (rg *ReportGenerator) GeneratePromoPerformanceReport(ctx context.Context, tenantID int64, promoCodeID int64, days int) (*PromoPerformanceReport, error) {
	// الحصول على تفاصيل الكود
	promoCode, err := rg.getPromoCode(ctx, tenantID, promoCodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}

	// الحصول على إحصائيات الاستخدام
	tracker := NewUsageTracker(rg.db)
	usageStats, err := tracker.GetPromoCodeUsageStats(ctx, tenantID, promoCodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// الحصول على أكثر المستخدمين استخداماً
	topUsers, err := tracker.GetTopUsers(ctx, tenantID, promoCodeID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}

	// الحصول على الإحصائيات اليومية
	dailyStats, err := tracker.GetDailyUsageStats(ctx, tenantID, promoCodeID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	// الحصول على الاستخدام حسب نوع الطلب
	usageByOrderType, err := rg.getUsageByOrderType(ctx, tenantID, promoCodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage by order type: %w", err)
	}

	// حساب معدل التحويل (Conversion Rate)
	conversionRate := rg.calculateConversionRate(usageStats)

	// حساب متوسط قيمة الطلب
	averageOrderValue := 0.0
	if usageStats.TotalUses > 0 {
		averageOrderValue = usageStats.TotalFinalAmount / float64(usageStats.TotalUses)
	}

	// حساب إجمالي الإيرادات المفقودة (الخصم المعطى)
	totalRevenueLost := usageStats.TotalDiscountGiven

	// حساب العائد على الاستثمار (ROI)
	roi := rg.calculateROI(usageStats)

	return &PromoPerformanceReport{
		PromoCode:         promoCode,
		UsageStats:        usageStats,
		TopUsers:          topUsers,
		DailyStats:        dailyStats,
		UsageByOrderType:  usageByOrderType,
		ConversionRate:    conversionRate,
		AverageOrderValue: averageOrderValue,
		TotalRevenueLost:  totalRevenueLost,
		ROI:               roi,
	}, nil
}

// GenerateAllPromosPerformanceReport يولد تقرير أداء لجميع الأكواد
// Requirements: 32.11
func (rg *ReportGenerator) GenerateAllPromosPerformanceReport(ctx context.Context, tenantID int64) (*AllPromosPerformanceReport, error) {
	// عدد الأكواد الإجمالي
	totalPromoCodes, err := rg.getTotalPromoCodesCount(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total promo codes count: %w", err)
	}

	// عدد الأكواد النشطة
	activePromoCodes, err := rg.getActivePromoCodesCount(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active promo codes count: %w", err)
	}

	// إحصائيات الاستخدام الإجمالية
	totalStats, err := rg.getTotalUsageStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total usage stats: %w", err)
	}

	// أفضل الأكواد أداءً
	topPerformingPromos, err := rg.getTopPerformingPromos(ctx, tenantID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top performing promos: %w", err)
	}

	// الاستخدامات الأخيرة
	recentUsages, err := rg.getRecentUsages(ctx, tenantID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent usages: %w", err)
	}

	return &AllPromosPerformanceReport{
		TotalPromoCodes:     totalPromoCodes,
		ActivePromoCodes:    activePromoCodes,
		TotalUsages:         totalStats.TotalUsages,
		TotalDiscountGiven:  totalStats.TotalDiscountGiven,
		TotalRevenue:        totalStats.TotalRevenue,
		UniqueUsers:         totalStats.UniqueUsers,
		TopPerformingPromos: topPerformingPromos,
		RecentUsages:        recentUsages,
	}, nil
}

// ComparePromoPerformance يقارن أداء عدة أكواد
// Requirements: 32.11
func (rg *ReportGenerator) ComparePromoPerformance(ctx context.Context, tenantID int64, promoCodeIDs []int64) ([]*PromoCodePerformance, error) {
	performances := make([]*PromoCodePerformance, 0, len(promoCodeIDs))

	manager := NewManager(rg.db)

	for _, promoCodeID := range promoCodeIDs {
		performance, err := manager.GetPromoPerformance(ctx, tenantID, promoCodeID)
		if err != nil {
			continue // تخطي الأكواد غير الموجودة
		}
		performances = append(performances, performance)
	}

	return performances, nil
}

// ExportPromoPerformanceCSV يصدر تقرير الأداء بصيغة CSV
// Requirements: 32.11
func (rg *ReportGenerator) ExportPromoPerformanceCSV(ctx context.Context, tenantID int64, promoCodeID int64) (string, error) {
	report, err := rg.GeneratePromoPerformanceReport(ctx, tenantID, promoCodeID, 30)
	if err != nil {
		return "", err
	}

	csv := "التاريخ,عدد الاستخدامات,المستخدمين الفريدين,إجمالي الخصم,إجمالي الإيرادات\n"

	for _, daily := range report.DailyStats {
		csv += fmt.Sprintf("%s,%d,%d,%.2f,%.2f\n",
			daily.UsageDate.Format("2006-01-02"),
			daily.UsageCount,
			daily.UniqueUsers,
			daily.TotalDiscount,
			daily.TotalRevenue,
		)
	}

	return csv, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// getPromoCode يحصل على تفاصيل كود خصم
func (rg *ReportGenerator) getPromoCode(ctx context.Context, tenantID int64, promoCodeID int64) (*PromoCode, error) {
	query := `
		SELECT 
			id, tenant_id, code, description, discount_type, discount_value,
			max_discount_amount, min_purchase_amount, valid_from, valid_until,
			usage_limit, usage_count, per_user_limit, applicable_to,
			applicable_package_ids, is_active, created_at, updated_at, created_by
		FROM promo_codes
		WHERE tenant_id = $1 AND id = $2
	`

	var promoCode PromoCode
	err := rg.db.GetContext(ctx, &promoCode, query, tenantID, promoCodeID)
	if err != nil {
		return nil, err
	}

	return &promoCode, nil
}

// getUsageByOrderType يحصل على الاستخدام حسب نوع الطلب
func (rg *ReportGenerator) getUsageByOrderType(ctx context.Context, tenantID int64, promoCodeID int64) (map[OrderType]int, error) {
	query := `
		SELECT order_type, COUNT(*) as count
		FROM promo_code_usage
		WHERE tenant_id = $1 AND promo_code_id = $2
		GROUP BY order_type
	`

	rows, err := rg.db.QueryContext(ctx, query, tenantID, promoCodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[OrderType]int)
	for rows.Next() {
		var orderType OrderType
		var count int
		if err := rows.Scan(&orderType, &count); err != nil {
			return nil, err
		}
		result[orderType] = count
	}

	return result, nil
}

// calculateConversionRate يحسب معدل التحويل
func (rg *ReportGenerator) calculateConversionRate(stats *PromoCodeUsageStats) float64 {
	// معدل التحويل = (عدد الاستخدامات / عدد المستخدمين الفريدين) * 100
	if stats.UniqueUsers == 0 {
		return 0
	}
	return (float64(stats.TotalUses) / float64(stats.UniqueUsers)) * 100
}

// calculateROI يحسب العائد على الاستثمار
func (rg *ReportGenerator) calculateROI(stats *PromoCodeUsageStats) float64 {
	// ROI = ((الإيرادات - الخصم) / الخصم) * 100
	if stats.TotalDiscountGiven == 0 {
		return 0
	}
	return ((stats.TotalFinalAmount - stats.TotalDiscountGiven) / stats.TotalDiscountGiven) * 100
}

// getTotalPromoCodesCount يحصل على عدد الأكواد الإجمالي
func (rg *ReportGenerator) getTotalPromoCodesCount(ctx context.Context, tenantID int64) (int, error) {
	query := `SELECT COUNT(*) FROM promo_codes WHERE tenant_id = $1`
	var count int
	err := rg.db.GetContext(ctx, &count, query, tenantID)
	return count, err
}

// getActivePromoCodesCount يحصل على عدد الأكواد النشطة
func (rg *ReportGenerator) getActivePromoCodesCount(ctx context.Context, tenantID int64) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM promo_codes 
		WHERE tenant_id = $1 AND is_active = TRUE AND valid_until > NOW()
	`
	var count int
	err := rg.db.GetContext(ctx, &count, query, tenantID)
	return count, err
}

// TotalUsageStats إحصائيات الاستخدام الإجمالية
type TotalUsageStats struct {
	TotalUsages        int     `db:"total_usages"`
	TotalDiscountGiven float64 `db:"total_discount_given"`
	TotalRevenue       float64 `db:"total_revenue"`
	UniqueUsers        int     `db:"unique_users"`
}

// getTotalUsageStats يحصل على إحصائيات الاستخدام الإجمالية
func (rg *ReportGenerator) getTotalUsageStats(ctx context.Context, tenantID int64) (*TotalUsageStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_usages,
			COALESCE(SUM(discount_amount), 0) as total_discount_given,
			COALESCE(SUM(final_amount), 0) as total_revenue,
			COUNT(DISTINCT user_id) as unique_users
		FROM promo_code_usage
		WHERE tenant_id = $1
	`

	var stats TotalUsageStats
	err := rg.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// getTopPerformingPromos يحصل على أفضل الأكواد أداءً
func (rg *ReportGenerator) getTopPerformingPromos(ctx context.Context, tenantID int64, limit int) ([]*PromoCodePerformance, error) {
	query := `
		SELECT 
			pc.id as promo_code_id,
			pc.code,
			pc.description,
			pc.usage_count as total_usage_count,
			COALESCE(SUM(pcu.discount_amount), 0) as total_discount_given,
			COALESCE(SUM(pcu.final_amount), 0) as total_revenue,
			COUNT(DISTINCT pcu.user_id) as unique_users
		FROM promo_codes pc
		LEFT JOIN promo_code_usage pcu ON pc.id = pcu.promo_code_id AND pc.tenant_id = pcu.tenant_id
		WHERE pc.tenant_id = $1
		GROUP BY pc.id, pc.code, pc.description, pc.usage_count
		ORDER BY total_revenue DESC
		LIMIT $2
	`

	var performances []*PromoCodePerformance
	err := rg.db.SelectContext(ctx, &performances, query, tenantID, limit)
	if err != nil {
		return nil, err
	}

	return performances, nil
}

// getRecentUsages يحصل على الاستخدامات الأخيرة
func (rg *ReportGenerator) getRecentUsages(ctx context.Context, tenantID int64, limit int) ([]*PromoCodeUsage, error) {
	query := `
		SELECT 
			id, tenant_id, promo_code_id, user_id, order_type, order_id,
			original_amount, discount_amount, final_amount, used_at
		FROM promo_code_usage
		WHERE tenant_id = $1
		ORDER BY used_at DESC
		LIMIT $2
	`

	var usages []*PromoCodeUsage
	err := rg.db.SelectContext(ctx, &usages, query, tenantID, limit)
	if err != nil {
		return nil, err
	}

	return usages, nil
}

// GetPromoTrends يحصل على اتجاهات استخدام الأكواد
// Requirements: 32.11
func (rg *ReportGenerator) GetPromoTrends(ctx context.Context, tenantID int64, days int) ([]*DailyTrend, error) {
	query := `
		SELECT 
			DATE(used_at) as trend_date,
			COUNT(*) as total_usages,
			COUNT(DISTINCT promo_code_id) as unique_promos_used,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(SUM(discount_amount), 0) as total_discount,
			COALESCE(SUM(final_amount), 0) as total_revenue
		FROM promo_code_usage
		WHERE tenant_id = $1 AND used_at >= NOW() - INTERVAL '1 day' * $2
		GROUP BY DATE(used_at)
		ORDER BY trend_date DESC
	`

	var trends []*DailyTrend
	err := rg.db.SelectContext(ctx, &trends, query, tenantID, days)
	if err != nil {
		return nil, err
	}

	return trends, nil
}

// DailyTrend اتجاه يومي
type DailyTrend struct {
	TrendDate         time.Time `db:"trend_date" json:"trend_date"`
	TotalUsages       int       `db:"total_usages" json:"total_usages"`
	UniquePromosUsed  int       `db:"unique_promos_used" json:"unique_promos_used"`
	UniqueUsers       int       `db:"unique_users" json:"unique_users"`
	TotalDiscount     float64   `db:"total_discount" json:"total_discount"`
	TotalRevenue      float64   `db:"total_revenue" json:"total_revenue"`
}
