package monitoring

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAggregator(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	assert.NotNil(t, aggregator)
	assert.Equal(t, db, aggregator.db)
}

func TestAggregateDaily(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	tenantID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	// Mock getting tenant_id
	mock.ExpectQuery("SELECT tenant_id FROM dedicated_gateways").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(tenantID))

	// Mock start transaction
	mock.ExpectBegin()

	// Mock message statistics query
	mock.ExpectQuery("SELECT .* FROM messages").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"sent", "delivered", "failed", "pending"}).
			AddRow(100, 95, 3, 2))

	// Mock performance statistics query
	mock.ExpectQuery("SELECT .* FROM messages").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"avg_time", "max_time", "min_time"}).
			AddRow(150, 500, 50))

	// Mock uptime statistics query
	mock.ExpectQuery("SELECT .* FROM gateway_health_checks").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"uptime"}).
			AddRow(99.5))

	// Mock financial statistics query
	mock.ExpectQuery("SELECT .* FROM accounting_transactions").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"total_cost", "total_revenue"}).
			AddRow(decimal.NewFromFloat(50.00), decimal.NewFromFloat(100.00)))

	// Mock insert/update statistics
	mock.ExpectExec("INSERT INTO gateway_statistics").
		WithArgs(
			tenantID,
			gatewayID,
			date.Format("2006-01-02"),
			100, 95, 3, 2,
			150, 500, 50,
			99.5, 0, // downtime_minutes defaults to 0 in my new calc
			decimal.NewFromFloat(50.00), decimal.NewFromFloat(100.00), decimal.NewFromFloat(50.00),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	mock.ExpectCommit()

	err = aggregator.AggregateDaily(ctx, gatewayID, date)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateMessageStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		mockRows *sqlmock.Rows
		expected *MessageStatistics
	}{
		{
			name: "successful calculation",
			mockRows: sqlmock.NewRows([]string{"sent", "delivered", "failed", "pending"}).
				AddRow(100, 95, 3, 2),
			expected: &MessageStatistics{
				Sent:      100,
				Delivered: 95,
				Failed:    3,
				Pending:   2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery("SELECT .* FROM messages").
				WithArgs(gatewayID, date.Format("2006-01-02")).
				WillReturnRows(tt.mockRows)

			stats, err := aggregator.calculateMessageStatistics(ctx, gatewayID, date, nil) // tx is optional in my code
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, stats)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculatePerformanceStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("from messages table", func(t *testing.T) {
		avgTime := 150
		maxTime := 500
		minTime := 50

		mock.ExpectQuery("SELECT .* FROM messages").
			WithArgs(gatewayID, date.Format("2006-01-02")).
			WillReturnRows(sqlmock.NewRows([]string{"avg_time", "max_time", "min_time"}).
				AddRow(avgTime, maxTime, minTime))

		stats, err := aggregator.calculatePerformanceStatistics(ctx, gatewayID, date, nil)
		assert.NoError(t, err)
		assert.NotNil(t, stats.AvgResponseTime)
		assert.Equal(t, avgTime, *stats.AvgResponseTime)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateUptimeStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		mockRows *sqlmock.Rows
		mockErr  error
		expected *UptimeStatistics
	}{
		{
			name: "high uptime",
			mockRows: sqlmock.NewRows([]string{"uptime"}).
				AddRow(99.5),
			expected: &UptimeStatistics{
				UptimePercentage: 99.5,
				DowntimeMinutes:  0,
			},
		},
		{
			name: "perfect uptime",
			mockRows: sqlmock.NewRows([]string{"uptime"}).
				AddRow(100.0),
			expected: &UptimeStatistics{
				UptimePercentage: 100.0,
				DowntimeMinutes:  0,
			},
		},
		{
			name:    "no health checks - default to 100%",
			mockErr: sql.ErrNoRows,
			expected: &UptimeStatistics{
				UptimePercentage: 100.0,
				DowntimeMinutes:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectBegin()
			tx, _ := db.BeginTx(ctx, nil)

			query := mock.ExpectQuery("SELECT .* FROM gateway_health_checks").
				WithArgs(gatewayID, date.Format("2006-01-02"))

			if tt.mockErr != nil {
				query.WillReturnError(tt.mockErr)
			} else {
				query.WillReturnRows(tt.mockRows)
			}

			stats, err := aggregator.calculateUptimeStatistics(ctx, gatewayID, date, tx)
			if tt.mockErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected.UptimePercentage, stats.UptimePercentage)
			assert.Equal(t, tt.expected.DowntimeMinutes, stats.DowntimeMinutes)
			tx.Rollback()
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateFinancialStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		mockRows *sqlmock.Rows
		expected *FinancialStatistics
	}{
		{
			name: "with cost and revenue",
			mockRows: sqlmock.NewRows([]string{"total_cost", "total_revenue"}).
				AddRow(decimal.NewFromFloat(50.00), decimal.NewFromFloat(100.00)),
			expected: &FinancialStatistics{
				TotalCost:    decimal.NewFromFloat(50.00),
				TotalRevenue: decimal.NewFromFloat(100.00),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery("SELECT .* FROM accounting_transactions").
				WithArgs(gatewayID, date.Format("2006-01-02")).
				WillReturnRows(tt.mockRows)

			stats, err := aggregator.calculateFinancialStatistics(ctx, gatewayID, date, nil)
			assert.NoError(t, err)
			assert.True(t, tt.expected.TotalCost.Equal(stats.TotalCost))
			assert.True(t, tt.expected.TotalRevenue.Equal(stats.TotalRevenue))
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatistics(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	tenantID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	// Mock query
	rows := sqlmock.NewRows([]string{
		"gateway_id", "tenant_id", "date",
		"messages_sent", "messages_delivered", "messages_failed", "messages_pending",
		"avg_response_time_ms", "max_response_time_ms", "min_response_time_ms",
		"uptime_percentage", "downtime_minutes",
		"total_cost", "total_revenue", "profit_margin",
	}).
		AddRow(
			gatewayID, tenantID, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			100, 95, 3, 2,
			150, 500, 50,
			99.5, 7,
			decimal.NewFromFloat(50.00), decimal.NewFromFloat(100.00), decimal.NewFromFloat(50.00),
		)

	mock.ExpectQuery("SELECT .* FROM gateway_statistics").
		WithArgs(gatewayID, tenantID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		WillReturnRows(rows)

	stats, err := aggregator.GetStatistics(ctx, gatewayID, tenantID, startDate, endDate)
	assert.NoError(t, err)
	assert.Len(t, stats, 1)

	// Verify record
	assert.Equal(t, gatewayID, stats[0].GatewayID)
	assert.Equal(t, tenantID, stats[0].TenantID)
	assert.Equal(t, 100, stats[0].MessagesSent)
	assert.Equal(t, 95, stats[0].MessagesDelivered)
	assert.NotNil(t, stats[0].AvgResponseTimeMs)
	assert.Equal(t, 150, *stats[0].AvgResponseTimeMs)
	assert.Equal(t, 99.5, stats[0].UptimePercentage)
	assert.True(t, decimal.NewFromFloat(50.00).Equal(stats[0].TotalCost))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateSuccessRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	tenantID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		mockRows     *sqlmock.Rows
		expectedRate float64
	}{
		{
			name: "95% success rate",
			mockRows: sqlmock.NewRows([]string{"success_rate"}).
				AddRow(95.00),
			expectedRate: 95.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery("SELECT .* FROM gateway_statistics").
				WithArgs(gatewayID, tenantID, date.Format("2006-01-02")).
				WillReturnRows(tt.mockRows)

			rate, err := aggregator.CalculateSuccessRate(ctx, gatewayID, tenantID, date)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRate, rate)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAggregateAllDaily(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	gateway1ID := uuid.New()
	gateway1TenantID := uuid.New()
	gateway2ID := uuid.New()
	gateway2TenantID := uuid.New()

	// Mock getting all active gateways
	mock.ExpectQuery("SELECT id FROM dedicated_gateways WHERE status = 'active'").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(gateway1ID).
			AddRow(gateway2ID))

	// Mock aggregation for gateway 1
	mockGatewayAggregation(mock, gateway1ID, gateway1TenantID, date)

	// Mock aggregation for gateway 2
	mockGatewayAggregation(mock, gateway2ID, gateway2TenantID, date)

	err = aggregator.AggregateAllDaily(ctx, date)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Helper function to mock gateway aggregation
func mockGatewayAggregation(mock sqlmock.Sqlmock, gatewayID uuid.UUID, tenantID uuid.UUID, date time.Time) {
	// Mock getting tenant_id
	mock.ExpectQuery("SELECT tenant_id FROM dedicated_gateways").
		WithArgs(gatewayID).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(tenantID))

	mock.ExpectBegin()

	// Mock message statistics
	mock.ExpectQuery("SELECT .* FROM messages").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"sent", "delivered", "failed", "pending"}).
			AddRow(100, 95, 3, 2))

	// Mock performance statistics
	mock.ExpectQuery("SELECT .* FROM messages").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"avg_time", "max_time", "min_time"}).
			AddRow(150, 500, 50))

	// Mock uptime statistics
	mock.ExpectQuery("SELECT .* FROM gateway_health_checks").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"uptime"}).
			AddRow(99.5))

	// Mock financial statistics
	mock.ExpectQuery("SELECT .* FROM accounting_transactions").
		WithArgs(gatewayID, date.Format("2006-01-02")).
		WillReturnRows(sqlmock.NewRows([]string{"total_cost", "total_revenue"}).
			AddRow(decimal.NewFromFloat(50.00), decimal.NewFromFloat(100.00)))

	// Mock insert/update
	mock.ExpectExec("INSERT INTO gateway_statistics").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()
}

func TestGetGatewayTenantID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	gatewayID := uuid.New()
	expectedTenantID := uuid.New()

	t.Run("successful retrieval", func(t *testing.T) {
		mock.ExpectQuery("SELECT tenant_id FROM dedicated_gateways").
			WithArgs(gatewayID).
			WillReturnRows(sqlmock.NewRows([]string{"tenant_id"}).AddRow(expectedTenantID))

		tenantID, err := aggregator.getGatewayTenantID(ctx, gatewayID)
		assert.NoError(t, err)
		assert.Equal(t, expectedTenantID, tenantID)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}
