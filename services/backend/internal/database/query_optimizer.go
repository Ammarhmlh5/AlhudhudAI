package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// QueryOptimizer provides query optimization and analysis capabilities
type QueryOptimizer struct {
	db *sqlx.DB
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *sqlx.DB) *QueryOptimizer {
	return &QueryOptimizer{
		db: db,
	}
}

// QueryPlan represents the execution plan for a query
type QueryPlan struct {
	Query         string
	Plan          []PlanNode
	TotalCost     float64
	ExecutionTime time.Duration
	RowsReturned  int64
}

// PlanNode represents a node in the query execution plan
type PlanNode struct {
	NodeType      string
	RelationName  string
	StartupCost   float64
	TotalCost     float64
	PlanRows      int64
	PlanWidth     int
	ActualTime    float64
	ActualRows    int64
	Filter        string
	IndexName     string
}

// AnalyzeQuery runs EXPLAIN ANALYZE on a query and returns the execution plan
func (qo *QueryOptimizer) AnalyzeQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	// Prepare EXPLAIN ANALYZE query
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) %s", query)
	
	start := time.Now()
	rows, err := qo.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}
	defer rows.Close()

	var planJSON string
	if rows.Next() {
		if err := rows.Scan(&planJSON); err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
	}

	executionTime := time.Since(start)

	// Parse the plan (simplified - in production, parse the JSON properly)
	plan := &QueryPlan{
		Query:         query,
		ExecutionTime: executionTime,
	}

	return plan, nil
}

// OptimizeQuery analyzes a query and suggests optimizations
func (qo *QueryOptimizer) OptimizeQuery(ctx context.Context, query string, args ...interface{}) (string, []string, error) {
	plan, err := qo.AnalyzeQuery(ctx, query, args...)
	if err != nil {
		return query, nil, err
	}

	suggestions := []string{}
	optimizedQuery := query

	// Check for missing indexes
	if strings.Contains(strings.ToLower(query), "where") && !strings.Contains(strings.ToLower(query), "index") {
		suggestions = append(suggestions, "Consider adding indexes on WHERE clause columns")
	}

	// Check for SELECT *
	if strings.Contains(strings.ToLower(query), "select *") {
		suggestions = append(suggestions, "Avoid SELECT *, specify only needed columns")
		optimizedQuery = strings.Replace(optimizedQuery, "SELECT *", "SELECT /* specify columns */", 1)
	}

	// Check for missing tenant_id in WHERE clause
	if !strings.Contains(strings.ToLower(query), "tenant_id") {
		suggestions = append(suggestions, "Add tenant_id to WHERE clause for better index usage")
	}

	// Check execution time
	if plan.ExecutionTime > 100*time.Millisecond {
		suggestions = append(suggestions, fmt.Sprintf("Query is slow (%.2fms), consider optimization", float64(plan.ExecutionTime.Milliseconds())))
	}

	return optimizedQuery, suggestions, nil
}

// PreparedStatementCache manages prepared statements
type PreparedStatementCache struct {
	db         *sqlx.DB
	statements map[string]*sqlx.Stmt
}

// NewPreparedStatementCache creates a new prepared statement cache
func NewPreparedStatementCache(db *sqlx.DB) *PreparedStatementCache {
	return &PreparedStatementCache{
		db:         db,
		statements: make(map[string]*sqlx.Stmt),
	}
}

// GetOrPrepare gets a prepared statement from cache or prepares it
func (psc *PreparedStatementCache) GetOrPrepare(ctx context.Context, name, query string) (*sqlx.Stmt, error) {
	if stmt, exists := psc.statements[name]; exists {
		return stmt, nil
	}

	stmt, err := psc.db.PreparexContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement %s: %w", name, err)
	}

	psc.statements[name] = stmt
	return stmt, nil
}

// Close closes all prepared statements
func (psc *PreparedStatementCache) Close() error {
	for _, stmt := range psc.statements {
		if err := stmt.Close(); err != nil {
			return err
		}
	}
	return nil
}

// OptimizedQueries contains optimized versions of common queries
type OptimizedQueries struct {
	stmtCache *PreparedStatementCache
}

// NewOptimizedQueries creates a new optimized queries instance
func NewOptimizedQueries(db *sqlx.DB) *OptimizedQueries {
	return &OptimizedQueries{
		stmtCache: NewPreparedStatementCache(db),
	}
}

// GetActiveTariff retrieves active tariff with optimized query
func (oq *OptimizedQueries) GetActiveTariff(ctx context.Context, tenantID uuid.UUID, serviceID int, countryCode, operator string) (*Tariff, error) {
	// Optimized query with proper index usage
	query := `
		SELECT id, tenant_id, service_id, country_code, network_operator, 
		       message_type, cost_per_message, currency, effective_from, 
		       effective_to, is_active, created_at, updated_at
		FROM tariffs
		WHERE tenant_id = $1 
		  AND service_id = $2 
		  AND country_code = $3 
		  AND network_operator = $4 
		  AND is_active = true
		  AND effective_from <= CURRENT_DATE
		  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
		ORDER BY effective_from DESC
		LIMIT 1
	`

	stmt, err := oq.stmtCache.GetOrPrepare(ctx, "get_active_tariff", query)
	if err != nil {
		return nil, err
	}

	var tariff Tariff
	err = stmt.GetContext(ctx, &tariff, tenantID, serviceID, countryCode, operator)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active tariff found")
		}
		return nil, err
	}

	return &tariff, nil
}

// GetActiveSubscription retrieves active subscription with optimized query
func (oq *OptimizedQueries) GetActiveSubscription(ctx context.Context, tenantID, userID uuid.UUID) (*Subscription, error) {
	// Optimized query with index hints
	query := `
		SELECT id, tenant_id, user_id, package_id, status, 
		       start_date, end_date, messages_included, messages_used, 
		       messages_remaining, overage_messages, overage_cost, 
		       auto_renewal, next_renewal_date, created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1 
		  AND user_id = $2 
		  AND status IN ('active', 'grace_period')
		  AND end_date >= CURRENT_DATE
		ORDER BY end_date DESC
		LIMIT 1
	`

	stmt, err := oq.stmtCache.GetOrPrepare(ctx, "get_active_subscription", query)
	if err != nil {
		return nil, err
	}

	var subscription Subscription
	err = stmt.GetContext(ctx, &subscription, tenantID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active subscription found")
		}
		return nil, err
	}

	return &subscription, nil
}

// GetRolloverMessages retrieves rollover messages with optimized query
func (oq *OptimizedQueries) GetRolloverMessages(ctx context.Context, tenantID, subscriptionID uuid.UUID) ([]RolloverMessage, error) {
	// Optimized query with proper ordering for FIFO
	query := `
		SELECT id, tenant_id, subscription_id, source_month, 
		       messages_count, messages_used, messages_remaining, 
		       expiry_date, status, payment_status, created_at
		FROM rollover_messages
		WHERE tenant_id = $1 
		  AND subscription_id = $2 
		  AND status = 'active'
		  AND messages_remaining > 0
		  AND expiry_date >= CURRENT_DATE
		ORDER BY created_at ASC
	`

	stmt, err := oq.stmtCache.GetOrPrepare(ctx, "get_rollover_messages", query)
	if err != nil {
		return nil, err
	}

	var messages []RolloverMessage
	err = stmt.SelectContext(ctx, &messages, tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// GetGatewayStatistics retrieves gateway statistics with optimized aggregation
func (oq *OptimizedQueries) GetGatewayStatistics(ctx context.Context, tenantID uuid.UUID, gatewayID int, startDate, endDate time.Time) ([]GatewayStatistic, error) {
	// Optimized query with date range index usage
	query := `
		SELECT gateway_id, date, messages_sent, messages_delivered, 
		       messages_failed, messages_pending, avg_response_time, 
		       max_response_time, uptime_percentage, cost, revenue
		FROM gateway_statistics
		WHERE tenant_id = $1 
		  AND gateway_id = $2 
		  AND date BETWEEN $3 AND $4
		ORDER BY date DESC
	`

	stmt, err := oq.stmtCache.GetOrPrepare(ctx, "get_gateway_statistics", query)
	if err != nil {
		return nil, err
	}

	var stats []GatewayStatistic
	err = stmt.SelectContext(ctx, &stats, tenantID, gatewayID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetResourceUsage retrieves resource usage with optimized time-series query
func (oq *OptimizedQueries) GetResourceUsage(ctx context.Context, tenantID uuid.UUID, gatewayID int, startTime, endTime time.Time) ([]ResourceUsage, error) {
	// Optimized query with time-series index
	query := `
		SELECT gateway_id, timestamp, cpu_usage, memory_usage, 
		       bandwidth_usage, active_connections, queries_per_second, 
		       redis_memory_usage, redis_keys_count, messages_per_hour, 
		       queue_depth, api_requests_per_minute, active_workers
		FROM resource_usage
		WHERE tenant_id = $1 
		  AND gateway_id = $2 
		  AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp DESC
	`

	stmt, err := oq.stmtCache.GetOrPrepare(ctx, "get_resource_usage", query)
	if err != nil {
		return nil, err
	}

	var usage []ResourceUsage
	err = stmt.SelectContext(ctx, &usage, tenantID, gatewayID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return usage, nil
}

// GetAccountingTransactions retrieves transactions with optimized pagination
func (oq *OptimizedQueries) GetAccountingTransactions(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]AccountingTransaction, error) {
	// Optimized query with index-based pagination
	query := `
		SELECT id, tenant_id, transaction_type, amount, currency, 
		       balance_before, balance_after, description, status, 
		       user_id, package_id, gateway_id, created_at
		FROM accounting_transactions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	stmt, err := oq.stmtCache.GetOrPrepare(ctx, "get_accounting_transactions", query)
	if err != nil {
		return nil, err
	}

	var transactions []AccountingTransaction
	err = stmt.SelectContext(ctx, &transactions, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// Close closes all prepared statements
func (oq *OptimizedQueries) Close() error {
	return oq.stmtCache.Close()
}

// Temporary types for compilation (these should match your actual models)
type Tariff struct {
	ID              int       `db:"id"`
	TenantID        uuid.UUID `db:"tenant_id"`
	ServiceID       int       `db:"service_id"`
	CountryCode     string    `db:"country_code"`
	NetworkOperator string    `db:"network_operator"`
	MessageType     string    `db:"message_type"`
	CostPerMessage  float64   `db:"cost_per_message"`
	Currency        string    `db:"currency"`
	EffectiveFrom   time.Time `db:"effective_from"`
	EffectiveTo     *time.Time `db:"effective_to"`
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type Subscription struct {
	ID               int       `db:"id"`
	TenantID         uuid.UUID `db:"tenant_id"`
	UserID           uuid.UUID `db:"user_id"`
	PackageID        int       `db:"package_id"`
	Status           string    `db:"status"`
	StartDate        time.Time `db:"start_date"`
	EndDate          time.Time `db:"end_date"`
	MessagesIncluded int       `db:"messages_included"`
	MessagesUsed     int       `db:"messages_used"`
	MessagesRemaining int      `db:"messages_remaining"`
	OverageMessages  int       `db:"overage_messages"`
	OverageCost      float64   `db:"overage_cost"`
	AutoRenewal      bool      `db:"auto_renewal"`
	NextRenewalDate  *time.Time `db:"next_renewal_date"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

type RolloverMessage struct {
	ID                int       `db:"id"`
	TenantID          uuid.UUID `db:"tenant_id"`
	SubscriptionID    uuid.UUID `db:"subscription_id"`
	SourceMonth       string    `db:"source_month"`
	MessagesCount     int       `db:"messages_count"`
	MessagesUsed      int       `db:"messages_used"`
	MessagesRemaining int       `db:"messages_remaining"`
	ExpiryDate        time.Time `db:"expiry_date"`
	Status            string    `db:"status"`
	PaymentStatus     string    `db:"payment_status"`
	CreatedAt         time.Time `db:"created_at"`
}

type GatewayStatistic struct {
	GatewayID         int       `db:"gateway_id"`
	Date              time.Time `db:"date"`
	MessagesSent      int       `db:"messages_sent"`
	MessagesDelivered int       `db:"messages_delivered"`
	MessagesFailed    int       `db:"messages_failed"`
	MessagesPending   int       `db:"messages_pending"`
	AvgResponseTime   float64   `db:"avg_response_time"`
	MaxResponseTime   float64   `db:"max_response_time"`
	UptimePercentage  float64   `db:"uptime_percentage"`
	Cost              float64   `db:"cost"`
	Revenue           float64   `db:"revenue"`
}

type ResourceUsage struct {
	GatewayID            int       `db:"gateway_id"`
	Timestamp            time.Time `db:"timestamp"`
	CPUUsage             float64   `db:"cpu_usage"`
	MemoryUsage          int64     `db:"memory_usage"`
	BandwidthUsage       int64     `db:"bandwidth_usage"`
	ActiveConnections    int       `db:"active_connections"`
	QueriesPerSecond     int       `db:"queries_per_second"`
	RedisMemoryUsage     int64     `db:"redis_memory_usage"`
	RedisKeysCount       int       `db:"redis_keys_count"`
	MessagesPerHour      int       `db:"messages_per_hour"`
	QueueDepth           int       `db:"queue_depth"`
	APIRequestsPerMinute int       `db:"api_requests_per_minute"`
	ActiveWorkers        int       `db:"active_workers"`
}

type AccountingTransaction struct {
	ID            int       `db:"id"`
	TenantID      uuid.UUID `db:"tenant_id"`
	TransactionType string  `db:"transaction_type"`
	Amount        float64   `db:"amount"`
	Currency      string    `db:"currency"`
	BalanceBefore float64   `db:"balance_before"`
	BalanceAfter  float64   `db:"balance_after"`
	Description   string    `db:"description"`
	Status        string    `db:"status"`
	UserID        *uuid.UUID `db:"user_id"`
	PackageID     *int      `db:"package_id"`
	GatewayID     *int      `db:"gateway_id"`
	CreatedAt     time.Time `db:"created_at"`
}
