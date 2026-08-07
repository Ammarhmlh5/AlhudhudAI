package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Aggregator aggregates daily statistics for dedicated gateways
type Aggregator struct {
	db *sql.DB
}

// NewAggregator creates a new statistics aggregator
func NewAggregator(db *sql.DB) *Aggregator {
	return &Aggregator{
		db: db,
	}
}

// DailyStatistics represents daily statistics for a gateway
type DailyStatistics struct {
	GatewayID          uuid.UUID
	TenantID           uuid.UUID
	Date               time.Time
	MessagesSent       int
	MessagesDelivered  int
	MessagesFailed     int
	MessagesPending    int
	AvgResponseTimeMs  *int
	MaxResponseTimeMs  *int
	MinResponseTimeMs  *int
	UptimePercentage   float64
	DowntimeMinutes    int
	TotalCost          decimal.Decimal
	TotalRevenue       decimal.Decimal
	ProfitMargin       decimal.Decimal
}

// AggregateDaily aggregates daily statistics for a specific gateway and date
func (a *Aggregator) AggregateDaily(ctx context.Context, gatewayID uuid.UUID, date time.Time) error {
	// 1. Get tenant_id for the gateway
	var tenantID uuid.UUID
	err := a.db.QueryRowContext(ctx, "SELECT tenant_id FROM dedicated_gateways WHERE id = $1", gatewayID).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("gateway not found: %w", err)
	}

	// 2. Start transaction for atomicity
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 3. Calculate message statistics (Direct hit on messages.gateway_id - No N+1!)
	messageStats, err := a.calculateMessageStatistics(ctx, gatewayID, date, tx)
	if err != nil {
		return err
	}

	// 4. Calculate performance statistics from messages table
	perfStats, err := a.calculatePerformanceStatistics(ctx, gatewayID, date, tx)
	if err != nil {
		return err
	}

	// 5. Calculate uptime statistics
	uptimeStats, err := a.calculateUptimeStatistics(ctx, gatewayID, date, tx)
	if err != nil {
		return err
	}

	// 6. Calculate financial statistics (Direct hit on accounting_transactions.gateway_id)
	financialStats, err := a.calculateFinancialStatistics(ctx, gatewayID, date, tx)
	if err != nil {
		return err
	}

	// 7. Save results
	stats := &DailyStatistics{
		GatewayID:         gatewayID,
		TenantID:          tenantID,
		Date:              date,
		MessagesSent:      messageStats.Sent,
		MessagesDelivered: messageStats.Delivered,
		MessagesFailed:    messageStats.Failed,
		MessagesPending:   messageStats.Pending,
		AvgResponseTimeMs: perfStats.AvgResponseTime,
		MaxResponseTimeMs: perfStats.MaxResponseTime,
		MinResponseTimeMs: perfStats.MinResponseTime,
		UptimePercentage:  uptimeStats.UptimePercentage,
		DowntimeMinutes:   uptimeStats.DowntimeMinutes,
		TotalCost:         financialStats.TotalCost,
		TotalRevenue:      financialStats.TotalRevenue,
		ProfitMargin:      financialStats.TotalRevenue.Sub(financialStats.TotalCost),
	}

	if err := a.saveStatistics(ctx, stats, tx); err != nil {
		return err
	}

	return tx.Commit()
}

// AggregateAllDaily aggregates daily statistics for all active gateways
func (a *Aggregator) AggregateAllDaily(ctx context.Context, date time.Time) error {
	rows, err := a.db.QueryContext(ctx, "SELECT id FROM dedicated_gateways WHERE status = 'active' AND deleted_at IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if err := a.AggregateDaily(ctx, id, date); err != nil {
			fmt.Printf("Error aggregating gateway %s: %v\n", id, err)
			continue
		}
	}
	return nil
}

type MessageStatistics struct {
	Sent      int
	Delivered int
	Failed    int
	Pending   int
}

func (a *Aggregator) calculateMessageStatistics(ctx context.Context, gatewayID uuid.UUID, date time.Time, tx *sql.Tx) (*MessageStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'DELIVERED') as delivered,
			COUNT(*) FILTER (WHERE status = 'FAILED') as failed,
			COUNT(*) FILTER (WHERE status = 'PENDING' OR status = 'PROCESSING') as pending
		FROM messages
		WHERE gateway_id = $1
		  AND DATE(created_at) = $2
	`
	stats := &MessageStatistics{}
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02"))
	} else {
		row = a.db.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02"))
	}
	err := row.Scan(&stats.Sent, &stats.Delivered, &stats.Failed, &stats.Pending)
	return stats, err
}

type PerformanceStatistics struct {
	AvgResponseTime *int
	MaxResponseTime *int
	MinResponseTime *int
}

func (a *Aggregator) calculatePerformanceStatistics(ctx context.Context, gatewayID uuid.UUID, date time.Time, tx *sql.Tx) (*PerformanceStatistics, error) {
	query := `
		SELECT 
			AVG(EXTRACT(EPOCH FROM (updated_at - created_at)) * 1000)::INTEGER as avg_time,
			MAX(EXTRACT(EPOCH FROM (updated_at - created_at)) * 1000)::INTEGER as max_time,
			MIN(EXTRACT(EPOCH FROM (updated_at - created_at)) * 1000)::INTEGER as min_time
		FROM messages
		WHERE gateway_id = $1
		  AND DATE(created_at) = $2
		  AND status = 'DELIVERED'
	`
	stats := &PerformanceStatistics{}
	var avg, max, min sql.NullInt64
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02"))
	} else {
		row = a.db.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02"))
	}
	err := row.Scan(&avg, &max, &min)
	if err == nil {
		if avg.Valid { v := int(avg.Int64); stats.AvgResponseTime = &v }
		if max.Valid { v := int(max.Int64); stats.MaxResponseTime = &v }
		if min.Valid { v := int(min.Int64); stats.MinResponseTime = &v }
	}
	return stats, err
}

type UptimeStatistics struct {
	UptimePercentage float64
	DowntimeMinutes  int
}

func (a *Aggregator) calculateUptimeStatistics(ctx context.Context, gatewayID uuid.UUID, date time.Time, tx *sql.Tx) (*UptimeStatistics, error) {
	// Simple uptime calculation from health checks
	query := `
		SELECT 
			COALESCE(AVG(CASE WHEN status = 'ONLINE' THEN 100 ELSE 0 END), 100)::FLOAT as uptime
		FROM gateway_health_checks
		WHERE gateway_id = $1
		  AND DATE(checked_at) = $2
	`
	stats := &UptimeStatistics{UptimePercentage: 100.0}
	err := tx.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02")).Scan(&stats.UptimePercentage)
	return stats, err
}

type FinancialStatistics struct {
	TotalCost    decimal.Decimal
	TotalRevenue decimal.Decimal
}

func (a *Aggregator) calculateFinancialStatistics(ctx context.Context, gatewayID uuid.UUID, date time.Time, tx *sql.Tx) (*FinancialStatistics, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN transaction_type = 'COST' THEN amount ELSE 0 END), 0) as total_cost,
			COALESCE(SUM(CASE WHEN transaction_type = 'DEDUCTION' THEN amount WHEN transaction_type = 'REFUND' THEN -amount ELSE 0 END), 0) as total_revenue
		FROM accounting_transactions
		WHERE gateway_id = $1
		  AND DATE(created_at) = $2
	`
	stats := &FinancialStatistics{}
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02"))
	} else {
		row = a.db.QueryRowContext(ctx, query, gatewayID, date.Format("2006-01-02"))
	}
	err := row.Scan(&stats.TotalCost, &stats.TotalRevenue)
	return stats, err
}

func (a *Aggregator) saveStatistics(ctx context.Context, stats *DailyStatistics, tx *sql.Tx) error {
	query := `
		INSERT INTO gateway_statistics (
			tenant_id, gateway_id, date, 
			messages_sent, messages_delivered, messages_failed, messages_pending,
			avg_response_time_ms, max_response_time_ms, min_response_time_ms,
			uptime_percentage, downtime_minutes,
			total_cost, total_revenue, profit_margin
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (gateway_id, date) DO UPDATE SET
			messages_sent = EXCLUDED.messages_sent,
			messages_delivered = EXCLUDED.messages_delivered,
			messages_failed = EXCLUDED.messages_failed,
			messages_pending = EXCLUDED.messages_pending,
			avg_response_time_ms = EXCLUDED.avg_response_time_ms,
			max_response_time_ms = EXCLUDED.max_response_time_ms,
			min_response_time_ms = EXCLUDED.min_response_time_ms,
			uptime_percentage = EXCLUDED.uptime_percentage,
			downtime_minutes = EXCLUDED.downtime_minutes,
			total_cost = EXCLUDED.total_cost,
			total_revenue = EXCLUDED.total_revenue,
			profit_margin = EXCLUDED.profit_margin,
			updated_at = NOW()
	`
	_, err := tx.ExecContext(ctx, query,
		stats.TenantID, stats.GatewayID, stats.Date.Format("2006-01-02"),
		stats.MessagesSent, stats.MessagesDelivered, stats.MessagesFailed, stats.MessagesPending,
		stats.AvgResponseTimeMs, stats.MaxResponseTimeMs, stats.MinResponseTimeMs,
		stats.UptimePercentage, stats.DowntimeMinutes,
		stats.TotalCost, stats.TotalRevenue, stats.ProfitMargin,
	)
	return err
}

// GetStatistics retrieves statistics for a gateway and date range
func (a *Aggregator) GetStatistics(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID, startDate, endDate time.Time) ([]DailyStatistics, error) {
	query := `
		SELECT
			gateway_id, tenant_id, date,
			messages_sent, messages_delivered, messages_failed, messages_pending,
			avg_response_time_ms, max_response_time_ms, min_response_time_ms,
			uptime_percentage, downtime_minutes,
			total_cost, total_revenue, profit_margin
		FROM gateway_statistics
		WHERE gateway_id = $1 AND tenant_id = $2
		  AND date BETWEEN $3 AND $4
		ORDER BY date DESC
	`

	rows, err := a.db.QueryContext(ctx, query, gatewayID, tenantID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statistics []DailyStatistics
	for rows.Next() {
		var stat DailyStatistics
		err := rows.Scan(
			&stat.GatewayID, &stat.TenantID, &stat.Date,
			&stat.MessagesSent, &stat.MessagesDelivered, &stat.MessagesFailed, &stat.MessagesPending,
			&stat.AvgResponseTimeMs, &stat.MaxResponseTimeMs, &stat.MinResponseTimeMs,
			&stat.UptimePercentage, &stat.DowntimeMinutes,
			&stat.TotalCost, &stat.TotalRevenue, &stat.ProfitMargin,
		)
		if err != nil {
			return nil, err
		}
		statistics = append(statistics, stat)
	}
	return statistics, nil
}

// CalculateSuccessRate calculates the success rate for a gateway on a specific date
func (a *Aggregator) CalculateSuccessRate(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID, date time.Time) (float64, error) {
	query := `
		SELECT 
			CASE 
				WHEN messages_sent > 0 THEN 
					(messages_delivered::DECIMAL / messages_sent::DECIMAL * 100)::DECIMAL(5,2)
				ELSE 
					0.00
			END as success_rate
		FROM gateway_statistics
		WHERE gateway_id = $1 AND tenant_id = $2 AND date = $3
	`

	var successRate float64
	err := a.db.QueryRowContext(ctx, query, gatewayID, tenantID, date.Format("2006-01-02")).Scan(&successRate)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return successRate, err
}

// getGatewayTenantID retrieves the tenant_id for a gateway
func (a *Aggregator) getGatewayTenantID(ctx context.Context, gatewayID uuid.UUID) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := a.db.QueryRowContext(ctx, "SELECT tenant_id FROM dedicated_gateways WHERE id = $1", gatewayID).Scan(&tenantID)
	return tenantID, err
}

