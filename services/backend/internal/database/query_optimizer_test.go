package database

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryOptimizer_AnalyzeQuery(t *testing.T) {
	t.Run("analyze simple query", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		optimizer := NewQueryOptimizer(sqlxDB)

		query := "SELECT * FROM tariffs WHERE tenant_id = $1"
		tenantID := uuid.New()
		
		mock.ExpectQuery("EXPLAIN \\(ANALYZE, BUFFERS, FORMAT JSON\\)").
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).
				AddRow(`[{"Plan": {"Node Type": "Seq Scan"}}]`))

		plan, err := optimizer.AnalyzeQuery(context.Background(), query, tenantID)
		require.NoError(t, err)
		assert.NotNil(t, plan)
		assert.Equal(t, query, plan.Query)
	})

	t.Run("handle query error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		optimizer := NewQueryOptimizer(sqlxDB)

		query := "SELECT * FROM invalid_table"
		
		mock.ExpectQuery("EXPLAIN \\(ANALYZE, BUFFERS, FORMAT JSON\\)").
			WillReturnError(assert.AnError)

		_, err = optimizer.AnalyzeQuery(context.Background(), query)
		assert.Error(t, err)
	})
}

func TestQueryOptimizer_OptimizeQuery(t *testing.T) {
	t.Run("suggest optimization for SELECT *", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		optimizer := NewQueryOptimizer(sqlxDB)

		query := "SELECT * FROM tariffs WHERE tenant_id = $1"
		tenantID := uuid.New()
		
		mock.ExpectQuery("EXPLAIN \\(ANALYZE, BUFFERS, FORMAT JSON\\)").
			WithArgs(tenantID).
			WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).
				AddRow(`[{"Plan": {"Node Type": "Seq Scan"}}]`))

		optimized, suggestions, err := optimizer.OptimizeQuery(context.Background(), query, tenantID)
		require.NoError(t, err)
		assert.NotEqual(t, query, optimized)
		assert.Contains(t, suggestions, "Avoid SELECT *, specify only needed columns")
	})

	t.Run("suggest index for WHERE clause", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		optimizer := NewQueryOptimizer(sqlxDB)

		query := "SELECT id FROM tariffs WHERE country_code = $1"
		
		mock.ExpectQuery("EXPLAIN \\(ANALYZE, BUFFERS, FORMAT JSON\\)").
			WithArgs("YE").
			WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).
				AddRow(`[{"Plan": {"Node Type": "Seq Scan"}}]`))

		_, suggestions, err := optimizer.OptimizeQuery(context.Background(), query, "YE")
		require.NoError(t, err)
		assert.Contains(t, suggestions, "Consider adding indexes on WHERE clause columns")
	})

	t.Run("suggest tenant_id in WHERE clause", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		optimizer := NewQueryOptimizer(sqlxDB)

		query := "SELECT id FROM tariffs WHERE country_code = $1"
		
		mock.ExpectQuery("EXPLAIN \\(ANALYZE, BUFFERS, FORMAT JSON\\)").
			WithArgs("YE").
			WillReturnRows(sqlmock.NewRows([]string{"QUERY PLAN"}).
				AddRow(`[{"Plan": {"Node Type": "Seq Scan"}}]`))

		_, suggestions, err := optimizer.OptimizeQuery(context.Background(), query, "YE")
		require.NoError(t, err)
		assert.Contains(t, suggestions, "Add tenant_id to WHERE clause for better index usage")
	})
}

func TestPreparedStatementCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	cache := NewPreparedStatementCache(sqlxDB)
	defer cache.Close()

	t.Run("prepare and cache statement", func(t *testing.T) {
		query := "SELECT * FROM tariffs WHERE id = $1"
		
		mock.ExpectPrepare(regexp.QuoteMeta(query))

		stmt1, err := cache.GetOrPrepare(context.Background(), "test_query", query)
		require.NoError(t, err)
		assert.NotNil(t, stmt1)

		// Second call should return cached statement
		stmt2, err := cache.GetOrPrepare(context.Background(), "test_query", query)
		require.NoError(t, err)
		assert.Equal(t, stmt1, stmt2)
	})

	t.Run("handle prepare error", func(t *testing.T) {
		query := "INVALID SQL"
		
		mock.ExpectPrepare(regexp.QuoteMeta(query)).WillReturnError(assert.AnError)

		_, err := cache.GetOrPrepare(context.Background(), "invalid_query", query)
		assert.Error(t, err)
	})
}

func TestOptimizedQueries_GetActiveTariff(t *testing.T) {
	tenantID := uuid.New()
	serviceID := 1
	countryCode := "YE"
	operator := "Yemen Mobile"

	t.Run("get active tariff successfully", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		oq := NewOptimizedQueries(sqlxDB)
		defer oq.Close()

		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, serviceID, countryCode, operator).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "service_id", "country_code", "network_operator",
				"message_type", "cost_per_message", "currency", "effective_from",
				"effective_to", "is_active", "created_at", "updated_at",
			}).AddRow(
				1, tenantID, serviceID, countryCode, operator,
				"standard", 9.0, "YER", time.Now(),
				nil, true, time.Now(), time.Now(),
			))

		tariff, err := oq.GetActiveTariff(context.Background(), tenantID, serviceID, countryCode, operator)
		require.NoError(t, err)
		assert.NotNil(t, tariff)
		assert.Equal(t, tenantID, tariff.TenantID)
		assert.Equal(t, 9.0, tariff.CostPerMessage)
	})

	t.Run("no active tariff found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		oq := NewOptimizedQueries(sqlxDB)
		defer oq.Close()

		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, serviceID, countryCode, operator).
			WillReturnError(sqlmock.ErrCancelled)

		_, err = oq.GetActiveTariff(context.Background(), tenantID, serviceID, countryCode, operator)
		assert.Error(t, err)
	})
}

func TestOptimizedQueries_GetActiveSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	oq := NewOptimizedQueries(sqlxDB)
	defer oq.Close()

	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("get active subscription successfully", func(t *testing.T) {
		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "user_id", "package_id", "status",
				"start_date", "end_date", "messages_included", "messages_used",
				"messages_remaining", "overage_messages", "overage_cost",
				"auto_renewal", "next_renewal_date", "created_at", "updated_at",
			}).AddRow(
				1, tenantID, userID, 1, "active",
				time.Now(), time.Now().AddDate(0, 1, 0), 1000, 100,
				900, 0, 0.0,
				true, time.Now().AddDate(0, 1, 0), time.Now(), time.Now(),
			))

		subscription, err := oq.GetActiveSubscription(context.Background(), tenantID, userID)
		require.NoError(t, err)
		assert.NotNil(t, subscription)
		assert.Equal(t, "active", subscription.Status)
		assert.Equal(t, 1000, subscription.MessagesIncluded)
	})
}

func TestOptimizedQueries_GetRolloverMessages(t *testing.T) {
	tenantID := uuid.New()
	subscriptionID := uuid.New()

	t.Run("get rollover messages successfully", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		oq := NewOptimizedQueries(sqlxDB)
		defer oq.Close()

		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, subscriptionID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "subscription_id", "source_month",
				"messages_count", "messages_used", "messages_remaining",
				"expiry_date", "status", "payment_status", "created_at",
			}).AddRow(
				1, tenantID, subscriptionID, "2024-01",
				100, 20, 80,
				time.Now().AddDate(0, 1, 0), "active", "success", time.Now(),
			))

		messages, err := oq.GetRolloverMessages(context.Background(), tenantID, subscriptionID)
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, 80, messages[0].MessagesRemaining)
	})

	t.Run("no rollover messages found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "sqlmock")
		oq := NewOptimizedQueries(sqlxDB)
		defer oq.Close()

		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, subscriptionID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "subscription_id", "source_month",
				"messages_count", "messages_used", "messages_remaining",
				"expiry_date", "status", "payment_status", "created_at",
			}))

		messages, err := oq.GetRolloverMessages(context.Background(), tenantID, subscriptionID)
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})
}

func TestOptimizedQueries_GetAccountingTransactions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	oq := NewOptimizedQueries(sqlxDB)
	defer oq.Close()

	tenantID := uuid.New()

	t.Run("get transactions with pagination", func(t *testing.T) {
		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, 10, 0).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "transaction_type", "amount", "currency",
				"balance_before", "balance_after", "description", "status",
				"user_id", "package_id", "gateway_id", "created_at",
			}).AddRow(
				1, tenantID, "subscription", 10.0, "USD",
				100.0, 90.0, "Monthly subscription", "completed",
				nil, nil, nil, time.Now(),
			))

		transactions, err := oq.GetAccountingTransactions(context.Background(), tenantID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, "subscription", transactions[0].TransactionType)
	})
}

func BenchmarkOptimizedQueries_GetActiveTariff(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	oq := NewOptimizedQueries(sqlxDB)
	defer oq.Close()

	tenantID := uuid.New()
	serviceID := 1
	countryCode := "YE"
	operator := "Yemen Mobile"

	for i := 0; i < b.N; i++ {
		mock.ExpectPrepare("SELECT id, tenant_id").
			ExpectQuery().
			WithArgs(tenantID, serviceID, countryCode, operator).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tenant_id", "service_id", "country_code", "network_operator",
				"message_type", "cost_per_message", "currency", "effective_from",
				"effective_to", "is_active", "created_at", "updated_at",
			}).AddRow(
				1, tenantID, serviceID, countryCode, operator,
				"standard", 9.0, "YER", time.Now(),
				nil, true, time.Now(), time.Now(),
			))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = oq.GetActiveTariff(context.Background(), tenantID, serviceID, countryCode, operator)
	}
}
