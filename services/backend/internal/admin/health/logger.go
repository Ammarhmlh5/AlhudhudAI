package health

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HistoryLogger handles logging and querying server health check history
type HistoryLogger struct {
	db *sqlx.DB
}

// NewHistoryLogger creates a new history logger instance
func NewHistoryLogger(db *sqlx.DB) *HistoryLogger {
	return &HistoryLogger{
		db: db,
	}
}

// LogHealthCheck saves a health check result to the database
func (hl *HistoryLogger) LogHealthCheck(ctx context.Context, result *HealthCheckResult) error {
	query := `
		INSERT INTO server_health_logs (
			server_id,
			tenant_id,
			check_timestamp,
			response_time_ms,
			status_code,
			is_available,
			error_message,
			error_count,
			error_rate,
			health_status,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	
	_, err := hl.db.ExecContext(
		ctx,
		query,
		result.ServerID,
		result.TenantID,
		result.CheckTimestamp,
		result.ResponseTimeMs,
		result.StatusCode,
		result.IsAvailable,
		nullString(result.ErrorMessage),
		result.ErrorCount,
		result.ErrorRate,
		result.HealthStatus,
		time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to log health check: %w", err)
	}
	
	return nil
}

// GetHealthHistory retrieves health check history with optional filters
func (hl *HistoryLogger) GetHealthHistory(ctx context.Context, filter HealthHistoryFilter) ([]HealthCheckResult, error) {
	query := `
		SELECT 
			id,
			server_id,
			tenant_id,
			check_timestamp,
			response_time_ms,
			status_code,
			is_available,
			COALESCE(error_message, '') as error_message,
			error_count,
			error_rate,
			health_status
		FROM server_health_logs
		WHERE server_id = $1
	`
	
	args := []interface{}{filter.ServerID}
	argCount := 1
	
	// Add optional filters
	if filter.TenantID != nil {
		argCount++
		query += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
	}
	
	if filter.StartTime != nil {
		argCount++
		query += fmt.Sprintf(" AND check_timestamp >= $%d", argCount)
		args = append(args, *filter.StartTime)
	}
	
	if filter.EndTime != nil {
		argCount++
		query += fmt.Sprintf(" AND check_timestamp <= $%d", argCount)
		args = append(args, *filter.EndTime)
	}
	
	if filter.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND health_status = $%d", argCount)
		args = append(args, *filter.Status)
	}
	
	// Add ordering and pagination
	query += " ORDER BY check_timestamp DESC"
	
	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}
	
	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}
	
	var results []HealthCheckResult
	err := hl.db.SelectContext(ctx, &results, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get health history: %w", err)
	}
	
	return results, nil
}

// GetLatestHealthCheck retrieves the most recent health check for a server
func (hl *HistoryLogger) GetLatestHealthCheck(ctx context.Context, serverID uuid.UUID) (*HealthCheckResult, error) {
	query := `
		SELECT 
			id,
			server_id,
			tenant_id,
			check_timestamp,
			response_time_ms,
			status_code,
			is_available,
			COALESCE(error_message, '') as error_message,
			error_count,
			error_rate,
			health_status
		FROM server_health_logs
		WHERE server_id = $1
		ORDER BY check_timestamp DESC
		LIMIT 1
	`
	
	var result HealthCheckResult
	err := hl.db.GetContext(ctx, &result, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest health check: %w", err)
	}
	
	return &result, nil
}

// GetHealthSummary retrieves a summary of server health over a time period
func (hl *HistoryLogger) GetHealthSummary(ctx context.Context, serverID uuid.UUID, period time.Duration) (*HealthSummary, error) {
	query := `
		SELECT 
			server_id,
			tenant_id,
			(SELECT health_status FROM server_health_logs WHERE server_id = $1 ORDER BY check_timestamp DESC LIMIT 1) as current_status,
			(SELECT check_timestamp FROM server_health_logs WHERE server_id = $1 ORDER BY check_timestamp DESC LIMIT 1) as last_check_time,
			COALESCE(AVG(response_time_ms)::int, 0) as avg_response_time_ms,
			COALESCE(AVG(error_rate), 0) as current_error_rate,
			COALESCE((SUM(CASE WHEN is_available THEN 1 ELSE 0 END)::float / COUNT(*)::float) * 100, 0) as uptime_percentage,
			COUNT(*) as total_checks,
			SUM(CASE WHEN is_available = false THEN 1 ELSE 0 END) as failed_checks
		FROM server_health_logs
		WHERE server_id = $1
		AND check_timestamp > $2
		GROUP BY server_id, tenant_id
	`
	
	startTime := time.Now().Add(-period)
	
	var summary HealthSummary
	err := hl.db.GetContext(ctx, &summary, query, serverID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get health summary: %w", err)
	}
	
	return &summary, nil
}

// CleanupOldLogs deletes health check logs older than 30 days
func (hl *HistoryLogger) CleanupOldLogs(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM server_health_logs
		WHERE check_timestamp < NOW() - INTERVAL '30 days'
	`
	
	result, err := hl.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old logs: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}

// CleanupOldLogsForServer deletes health check logs older than specified duration for a specific server
func (hl *HistoryLogger) CleanupOldLogsForServer(ctx context.Context, serverID uuid.UUID, retentionPeriod time.Duration) (int64, error) {
	query := `
		DELETE FROM server_health_logs
		WHERE server_id = $1
		AND check_timestamp < $2
	`
	
	cutoffTime := time.Now().Add(-retentionPeriod)
	
	result, err := hl.db.ExecContext(ctx, query, serverID, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old logs for server: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}

// GetHealthCheckCount returns the total number of health checks for a server
func (hl *HistoryLogger) GetHealthCheckCount(ctx context.Context, serverID uuid.UUID) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM server_health_logs
		WHERE server_id = $1
	`
	
	var count int64
	err := hl.db.QueryRowContext(ctx, query, serverID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get health check count: %w", err)
	}
	
	return count, nil
}

// GetHealthCheckCountByStatus returns the count of health checks by status for a server
func (hl *HistoryLogger) GetHealthCheckCountByStatus(ctx context.Context, serverID uuid.UUID, period time.Duration) (map[HealthStatus]int64, error) {
	query := `
		SELECT 
			health_status,
			COUNT(*) as count
		FROM server_health_logs
		WHERE server_id = $1
		AND check_timestamp > $2
		GROUP BY health_status
	`
	
	startTime := time.Now().Add(-period)
	
	rows, err := hl.db.QueryContext(ctx, query, serverID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get health check count by status: %w", err)
	}
	defer rows.Close()
	
	counts := make(map[HealthStatus]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		counts[HealthStatus(status)] = count
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return counts, nil
}

// nullString returns a sql.NullString for the given string
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
