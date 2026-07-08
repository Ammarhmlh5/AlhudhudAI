package referral

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
)

// ReferralReport represents a detailed referral report
type ReferralReport struct {
	Period           string                   `json:"period"`
	StartDate        time.Time                `json:"start_date"`
	EndDate          time.Time                `json:"end_date"`
	Performance      *ReferralPerformance     `json:"performance"`
	TopReferrers     []*TopReferrer           `json:"top_referrers"`
	RewardBreakdown  *RewardBreakdown         `json:"reward_breakdown"`
	ConversionFunnel *ConversionFunnel        `json:"conversion_funnel"`
	Trends           *ReferralTrends          `json:"trends"`
}

// TopReferrer represents a top referrer
type TopReferrer struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	TenantName       string    `json:"tenant_name"`
	TotalReferrals   int       `json:"total_referrals"`
	RewardedReferrals int      `json:"rewarded_referrals"`
	TotalRewards     float64   `json:"total_rewards"`
	ConversionRate   float64   `json:"conversion_rate"`
}

// RewardBreakdown represents reward distribution by type
type RewardBreakdown struct {
	BalanceRewards   RewardTypeStats `json:"balance_rewards"`
	MessageRewards   RewardTypeStats `json:"message_rewards"`
	DiscountRewards  RewardTypeStats `json:"discount_rewards"`
}

// RewardTypeStats represents statistics for a reward type
type RewardTypeStats struct {
	Count       int     `json:"count"`
	TotalAmount float64 `json:"total_amount"`
	Percentage  float64 `json:"percentage"`
}

// ConversionFunnel represents the referral conversion funnel
type ConversionFunnel struct {
	TotalReferrals     int     `json:"total_referrals"`
	PendingReferrals   int     `json:"pending_referrals"`
	CompletedReferrals int     `json:"completed_referrals"`
	RewardedReferrals  int     `json:"rewarded_referrals"`
	CancelledReferrals int     `json:"cancelled_referrals"`
	PendingRate        float64 `json:"pending_rate"`
	CompletionRate     float64 `json:"completion_rate"`
	RewardRate         float64 `json:"reward_rate"`
	CancellationRate   float64 `json:"cancellation_rate"`
}

// ReferralTrends represents referral trends over time
type ReferralTrends struct {
	Daily   []*TrendPoint `json:"daily"`
	Weekly  []*TrendPoint `json:"weekly"`
	Monthly []*TrendPoint `json:"monthly"`
}

// TrendPoint represents a single point in a trend
type TrendPoint struct {
	Date              time.Time `json:"date"`
	NewReferrals      int       `json:"new_referrals"`
	CompletedReferrals int      `json:"completed_referrals"`
	RewardsGranted    int       `json:"rewards_granted"`
	TotalRewardAmount float64   `json:"total_reward_amount"`
}

// GenerateReferralReport generates a comprehensive referral report
// Requirement 33.9: THE النظام SHALL يوفر تقرير بأداء برنامج الإحالة
func (m *Manager) GenerateReferralReport(ctx context.Context, startDate, endDate time.Time) (*ReferralReport, error) {
	report := &ReferralReport{
		Period:    fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		StartDate: startDate,
		EndDate:   endDate,
	}
	
	// Get overall performance
	perf, err := m.GetReferralPerformanceForPeriod(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get performance: %w", err)
	}
	report.Performance = perf
	
	// Get top referrers
	topReferrers, err := m.GetTopReferrers(ctx, startDate, endDate, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top referrers: %w", err)
	}
	report.TopReferrers = topReferrers
	
	// Get reward breakdown
	breakdown, err := m.GetRewardBreakdown(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get reward breakdown: %w", err)
	}
	report.RewardBreakdown = breakdown
	
	// Get conversion funnel
	funnel, err := m.GetConversionFunnel(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion funnel: %w", err)
	}
	report.ConversionFunnel = funnel
	
	// Get trends
	trends, err := m.GetReferralTrends(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}
	report.Trends = trends
	
	return report, nil
}

// GetReferralPerformanceForPeriod retrieves performance metrics for a specific period
func (m *Manager) GetReferralPerformanceForPeriod(ctx context.Context, startDate, endDate time.Time) (*ReferralPerformance, error) {
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
		WHERE created_at >= $1 AND created_at <= $2
	`
	
	err := m.db.GetContext(ctx, &perf, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get performance: %w", err)
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

// GetTopReferrers retrieves the top referrers for a period
func (m *Manager) GetTopReferrers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopReferrer, error) {
	query := `
		SELECT 
			r.referrer_tenant_id as tenant_id,
			t.name as tenant_name,
			COUNT(*) as total_referrals,
			COUNT(*) FILTER (WHERE r.status = 'rewarded') as rewarded_referrals,
			COALESCE(SUM(CASE WHEN r.reward_granted THEN r.reward_amount ELSE 0 END), 0) as total_rewards,
			CASE 
				WHEN COUNT(*) > 0 THEN 
					(COUNT(*) FILTER (WHERE r.status IN ('completed', 'rewarded'))::float / COUNT(*)::float * 100)
				ELSE 0 
			END as conversion_rate
		FROM referrals r
		LEFT JOIN tenants t ON r.referrer_tenant_id = t.id
		WHERE r.created_at >= $1 AND r.created_at <= $2
		GROUP BY r.referrer_tenant_id, t.name
		ORDER BY total_referrals DESC, total_rewards DESC
		LIMIT $3
	`
	
	var topReferrers []*TopReferrer
	err := m.db.SelectContext(ctx, &topReferrers, query, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top referrers: %w", err)
	}
	
	return topReferrers, nil
}

// GetRewardBreakdown retrieves reward distribution by type
func (m *Manager) GetRewardBreakdown(ctx context.Context, startDate, endDate time.Time) (*RewardBreakdown, error) {
	query := `
		SELECT 
			reward_type,
			COUNT(*) as count,
			COALESCE(SUM(reward_amount), 0) as total_amount
		FROM referrals
		WHERE created_at >= $1 AND created_at <= $2
		  AND reward_granted = true
		GROUP BY reward_type
	`
	
	type rewardStat struct {
		RewardType  *RewardType `db:"reward_type"`
		Count       int         `db:"count"`
		TotalAmount float64     `db:"total_amount"`
	}
	
	var stats []rewardStat
	err := m.db.SelectContext(ctx, &stats, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get reward breakdown: %w", err)
	}
	
	breakdown := &RewardBreakdown{}
	totalCount := 0
	
	for _, stat := range stats {
		totalCount += stat.Count
	}
	
	for _, stat := range stats {
		percentage := 0.0
		if totalCount > 0 {
			percentage = float64(stat.Count) / float64(totalCount) * 100
		}
		
		rewardStats := RewardTypeStats{
			Count:       stat.Count,
			TotalAmount: stat.TotalAmount,
			Percentage:  percentage,
		}
		
		if stat.RewardType != nil {
			switch *stat.RewardType {
			case RewardTypeBalance:
				breakdown.BalanceRewards = rewardStats
			case RewardTypeMessages:
				breakdown.MessageRewards = rewardStats
			case RewardTypeDiscount:
				breakdown.DiscountRewards = rewardStats
			}
		}
	}
	
	return breakdown, nil
}

// GetConversionFunnel retrieves the referral conversion funnel
func (m *Manager) GetConversionFunnel(ctx context.Context, startDate, endDate time.Time) (*ConversionFunnel, error) {
	var funnel ConversionFunnel
	
	query := `
		SELECT 
			COUNT(*) as total_referrals,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_referrals,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_referrals,
			COUNT(*) FILTER (WHERE status = 'rewarded') as rewarded_referrals,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_referrals
		FROM referrals
		WHERE created_at >= $1 AND created_at <= $2
	`
	
	err := m.db.GetContext(ctx, &funnel, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversion funnel: %w", err)
	}
	
	// Calculate rates
	if funnel.TotalReferrals > 0 {
		funnel.PendingRate = float64(funnel.PendingReferrals) / float64(funnel.TotalReferrals) * 100
		funnel.CompletionRate = float64(funnel.CompletedReferrals) / float64(funnel.TotalReferrals) * 100
		funnel.RewardRate = float64(funnel.RewardedReferrals) / float64(funnel.TotalReferrals) * 100
		funnel.CancellationRate = float64(funnel.CancelledReferrals) / float64(funnel.TotalReferrals) * 100
	}
	
	return &funnel, nil
}

// GetReferralTrends retrieves referral trends over time
func (m *Manager) GetReferralTrends(ctx context.Context, startDate, endDate time.Time) (*ReferralTrends, error) {
	trends := &ReferralTrends{}
	
	// Get daily trends
	dailyTrends, err := m.getTrendsForInterval(ctx, startDate, endDate, "day")
	if err != nil {
		return nil, fmt.Errorf("failed to get daily trends: %w", err)
	}
	trends.Daily = dailyTrends
	
	// Get weekly trends
	weeklyTrends, err := m.getTrendsForInterval(ctx, startDate, endDate, "week")
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly trends: %w", err)
	}
	trends.Weekly = weeklyTrends
	
	// Get monthly trends
	monthlyTrends, err := m.getTrendsForInterval(ctx, startDate, endDate, "month")
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly trends: %w", err)
	}
	trends.Monthly = monthlyTrends
	
	return trends, nil
}

// getTrendsForInterval retrieves trends for a specific interval (day, week, month)
func (m *Manager) getTrendsForInterval(ctx context.Context, startDate, endDate time.Time, interval string) ([]*TrendPoint, error) {
	query := fmt.Sprintf(`
		SELECT 
			DATE_TRUNC('%s', created_at) as date,
			COUNT(*) as new_referrals,
			COUNT(*) FILTER (WHERE status IN ('completed', 'rewarded')) as completed_referrals,
			COUNT(*) FILTER (WHERE reward_granted = true) as rewards_granted,
			COALESCE(SUM(CASE WHEN reward_granted THEN reward_amount ELSE 0 END), 0) as total_reward_amount
		FROM referrals
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY DATE_TRUNC('%s', created_at)
		ORDER BY date ASC
	`, interval, interval)
	
	var trends []*TrendPoint
	err := m.db.SelectContext(ctx, &trends, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}
	
	return trends, nil
}

// ExportReferralReport exports a referral report to various formats
func (m *Manager) ExportReferralReport(ctx context.Context, report *ReferralReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return m.exportToJSON(report)
	case "csv":
		return m.exportToCSV(report)
	case "pdf":
		return m.exportToPDF(report)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// exportToJSON exports report to JSON format
func (m *Manager) exportToJSON(report *ReferralReport) ([]byte, error) {
	// For large reports (> 1000 records), we could implement streaming
	// For now, we use MarshalIndent for readable output
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal report to JSON: %w", err)
	}
	return data, nil
}

// exportToCSV exports report to CSV format
func (m *Manager) exportToCSV(report *ReferralReport) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	
	// Write header
	header := []string{"Period", "Metric", "Value"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Write performance metrics
	if report.Performance != nil {
		perf := report.Performance
		rows := [][]string{
			{report.Period, "Total Referrals", fmt.Sprintf("%d", perf.TotalReferrals)},
			{report.Period, "Completed Referrals", fmt.Sprintf("%d", perf.CompletedReferrals)},
			{report.Period, "Rewarded Referrals", fmt.Sprintf("%d", perf.RewardedReferrals)},
			{report.Period, "Pending Referrals", fmt.Sprintf("%d", perf.PendingReferrals)},
			{report.Period, "Cancelled Referrals", fmt.Sprintf("%d", perf.CancelledReferrals)},
			{report.Period, "Conversion Rate", fmt.Sprintf("%.2f%%", perf.ConversionRate)},
			{report.Period, "Reward Rate", fmt.Sprintf("%.2f%%", perf.RewardRate)},
			{report.Period, "Total Rewards Granted", fmt.Sprintf("%.2f", perf.TotalRewardsGranted)},
			{report.Period, "Average Reward", fmt.Sprintf("%.2f", perf.AverageReward)},
		}
		
		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				return nil, fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
	}
	
	// Write empty row separator
	writer.Write([]string{})
	
	// Write top referrers section
	if len(report.TopReferrers) > 0 {
		writer.Write([]string{"Top Referrers"})
		referrerHeader := []string{"Tenant ID", "Tenant Name", "Total Referrals", "Rewarded Referrals", "Total Rewards", "Conversion Rate"}
		if err := writer.Write(referrerHeader); err != nil {
			return nil, fmt.Errorf("failed to write referrer header: %w", err)
		}
		
		for _, ref := range report.TopReferrers {
			row := []string{
				ref.TenantID.String(),
				ref.TenantName,
				fmt.Sprintf("%d", ref.TotalReferrals),
				fmt.Sprintf("%d", ref.RewardedReferrals),
				fmt.Sprintf("%.2f", ref.TotalRewards),
				fmt.Sprintf("%.2f%%", ref.ConversionRate),
			}
			if err := writer.Write(row); err != nil {
				return nil, fmt.Errorf("failed to write referrer row: %w", err)
			}
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}
	
	return buf.Bytes(), nil
}

// exportToPDF exports report to PDF format
func (m *Manager) exportToPDF(report *ReferralReport) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	
	// Set font for header
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Referral Program Report")
	pdf.Ln(12)
	
	// Period
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 8, fmt.Sprintf("Period: %s", report.Period))
	pdf.Ln(10)
	
	// Summary Statistics
	if report.Performance != nil {
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 8, "Performance Summary")
		pdf.Ln(8)
		
		pdf.SetFont("Arial", "", 11)
		perf := report.Performance
		
		stats := []struct {
			label string
			value string
		}{
			{"Total Referrals:", fmt.Sprintf("%d", perf.TotalReferrals)},
			{"Completed Referrals:", fmt.Sprintf("%d", perf.CompletedReferrals)},
			{"Rewarded Referrals:", fmt.Sprintf("%d", perf.RewardedReferrals)},
			{"Pending Referrals:", fmt.Sprintf("%d", perf.PendingReferrals)},
			{"Cancelled Referrals:", fmt.Sprintf("%d", perf.CancelledReferrals)},
			{"Conversion Rate:", fmt.Sprintf("%.2f%%", perf.ConversionRate)},
			{"Reward Rate:", fmt.Sprintf("%.2f%%", perf.RewardRate)},
			{"Total Rewards Granted:", fmt.Sprintf("%.2f", perf.TotalRewardsGranted)},
			{"Average Reward:", fmt.Sprintf("%.2f", perf.AverageReward)},
		}
		
		for _, stat := range stats {
			pdf.Cell(80, 6, stat.label)
			pdf.Cell(0, 6, stat.value)
			pdf.Ln(6)
		}
		pdf.Ln(5)
	}
	
	// Top Referrers Table
	if len(report.TopReferrers) > 0 {
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 8, "Top Referrers")
		pdf.Ln(8)
		
		// Table header
		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(200, 200, 200)
		pdf.CellFormat(60, 7, "Tenant Name", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, "Referrals", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, "Rewarded", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, "Total Rewards", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, "Conv. Rate", "1", 0, "C", true, 0, "")
		pdf.Ln(-1)
		
		// Table rows (limit to 50 per page for pagination)
		pdf.SetFont("Arial", "", 9)
		rowCount := 0
		for _, ref := range report.TopReferrers {
			// Add new page if needed (50 records per page)
			if rowCount > 0 && rowCount%50 == 0 {
				pdf.AddPage()
				pdf.SetFont("Arial", "B", 14)
				pdf.Cell(0, 8, "Top Referrers (continued)")
				pdf.Ln(8)
				
				// Repeat header
				pdf.SetFont("Arial", "B", 10)
				pdf.SetFillColor(200, 200, 200)
				pdf.CellFormat(60, 7, "Tenant Name", "1", 0, "C", true, 0, "")
				pdf.CellFormat(30, 7, "Referrals", "1", 0, "C", true, 0, "")
				pdf.CellFormat(30, 7, "Rewarded", "1", 0, "C", true, 0, "")
				pdf.CellFormat(35, 7, "Total Rewards", "1", 0, "C", true, 0, "")
				pdf.CellFormat(35, 7, "Conv. Rate", "1", 0, "C", true, 0, "")
				pdf.Ln(-1)
				pdf.SetFont("Arial", "", 9)
			}
			
			pdf.CellFormat(60, 6, ref.TenantName, "1", 0, "L", false, 0, "")
			pdf.CellFormat(30, 6, fmt.Sprintf("%d", ref.TotalReferrals), "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 6, fmt.Sprintf("%d", ref.RewardedReferrals), "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 6, fmt.Sprintf("%.2f", ref.TotalRewards), "1", 0, "R", false, 0, "")
			pdf.CellFormat(35, 6, fmt.Sprintf("%.2f%%", ref.ConversionRate), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
			rowCount++
		}
	}
	
	// Footer
	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 5, fmt.Sprintf("Generated on %s", time.Now().Format("2006-01-02 15:04:05")))
	
	// Output to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	
	return buf.Bytes(), nil
}

// GetReferralInsights provides insights and recommendations
func (m *Manager) GetReferralInsights(ctx context.Context) (*ReferralInsights, error) {
	insights := &ReferralInsights{}
	
	// Get current performance
	perf, err := m.GetReferralPerformance(ctx)
	if err != nil {
		return nil, err
	}
	
	// Analyze conversion rate
	if perf.ConversionRate < 50 {
		insights.Recommendations = append(insights.Recommendations, 
			"Conversion rate is low. Consider improving the onboarding experience for referred users.")
	}
	
	// Analyze reward rate
	if perf.RewardRate < 80 {
		insights.Recommendations = append(insights.Recommendations, 
			"Many completed referrals haven't received rewards. Review the reward granting process.")
	}
	
	// Analyze average reward
	if perf.AverageReward < 5 {
		insights.Recommendations = append(insights.Recommendations, 
			"Average reward is low. Consider increasing rewards to incentivize more referrals.")
	}
	
	insights.OverallHealth = m.calculateOverallHealth(perf)
	
	return insights, nil
}

// ReferralInsights represents insights and recommendations
type ReferralInsights struct {
	OverallHealth   string   `json:"overall_health"`
	Recommendations []string `json:"recommendations"`
}

// calculateOverallHealth calculates the overall health of the referral program
func (m *Manager) calculateOverallHealth(perf *ReferralPerformance) string {
	score := 0
	
	if perf.ConversionRate >= 70 {
		score += 3
	} else if perf.ConversionRate >= 50 {
		score += 2
	} else if perf.ConversionRate >= 30 {
		score += 1
	}
	
	if perf.RewardRate >= 90 {
		score += 3
	} else if perf.RewardRate >= 70 {
		score += 2
	} else if perf.RewardRate >= 50 {
		score += 1
	}
	
	if perf.TotalReferrals >= 100 {
		score += 2
	} else if perf.TotalReferrals >= 50 {
		score += 1
	}
	
	switch {
	case score >= 7:
		return "Excellent"
	case score >= 5:
		return "Good"
	case score >= 3:
		return "Fair"
	default:
		return "Needs Improvement"
	}
}
