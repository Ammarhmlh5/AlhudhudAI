package rating

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HistoryManager manages rating history
type HistoryManager struct {
	db *sqlx.DB
}

// NewHistoryManager creates a new HistoryManager
func NewHistoryManager(db *sqlx.DB) *HistoryManager {
	return &HistoryManager{
		db: db,
	}
}

// SaveRatingHistory saves a rating record to history
func (h *HistoryManager) SaveRatingHistory(ctx context.Context, rating *CompanyRating) error {
	// Invalidate previous rating (set valid_to)
	updateQuery := `
		UPDATE company_ratings
		SET valid_to = $1
		WHERE tenant_id = $2
		  AND valid_to IS NULL
	`

	_, err := h.db.ExecContext(ctx, updateQuery, rating.ValidFrom, rating.TenantID)
	if err != nil {
		return fmt.Errorf("failed to invalidate previous rating: %w", err)
	}

	// Insert new rating
	insertQuery := `
		INSERT INTO company_ratings (
			tenant_id, success_rate, sending_volume, payment_compliance,
			service_quality_score, overall_rating, rating_category,
			calculation_timestamp, valid_from
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING id
	`

	err = h.db.QueryRowContext(ctx, insertQuery,
		rating.TenantID,
		rating.SuccessRate,
		rating.SendingVolume,
		rating.PaymentCompliance,
		rating.ServiceQualityScore,
		rating.OverallRating,
		rating.RatingCategory,
		rating.CalculationTimestamp,
		rating.ValidFrom,
	).Scan(&rating.ID)

	if err != nil {
		return fmt.Errorf("failed to insert rating: %w", err)
	}

	return nil
}

// GetLatestRating gets the latest rating for a company
func (h *HistoryManager) GetLatestRating(ctx context.Context, tenantID uuid.UUID) (*CompanyRating, error) {
	query := `
		SELECT 
			id, tenant_id, success_rate, sending_volume, payment_compliance,
			service_quality_score, overall_rating, rating_category,
			calculation_timestamp, valid_from, valid_to
		FROM company_ratings
		WHERE tenant_id = $1
		  AND valid_to IS NULL
		ORDER BY calculation_timestamp DESC
		LIMIT 1
	`

	var rating CompanyRating
	err := h.db.QueryRowContext(ctx, query, tenantID).Scan(
		&rating.ID,
		&rating.TenantID,
		&rating.SuccessRate,
		&rating.SendingVolume,
		&rating.PaymentCompliance,
		&rating.ServiceQualityScore,
		&rating.OverallRating,
		&rating.RatingCategory,
		&rating.CalculationTimestamp,
		&rating.ValidFrom,
		&rating.ValidTo,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get latest rating: %w", err)
	}

	return &rating, nil
}

// GetRatingHistory gets rating history for a company with filters
func (h *HistoryManager) GetRatingHistory(ctx context.Context, filter *RatingHistoryFilter) ([]*CompanyRating, error) {
	// Build query with filters
	query := `
		SELECT 
			id, tenant_id, success_rate, sending_volume, payment_compliance,
			service_quality_score, overall_rating, rating_category,
			calculation_timestamp, valid_from, valid_to
		FROM company_ratings
		WHERE tenant_id = $1
	`

	args := []interface{}{filter.TenantID}
	argIndex := 2

	// Add date filters
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND calculation_timestamp >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND calculation_timestamp <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Add category filter
	if filter.Category != nil {
		query += fmt.Sprintf(" AND rating_category = $%d", argIndex)
		args = append(args, *filter.Category)
		argIndex++
	}

	// Add ordering and pagination
	query += " ORDER BY calculation_timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	// Execute query
	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rating history: %w", err)
	}
	defer rows.Close()

	// Scan results
	var ratings []*CompanyRating
	for rows.Next() {
		var rating CompanyRating
		err := rows.Scan(
			&rating.ID,
			&rating.TenantID,
			&rating.SuccessRate,
			&rating.SendingVolume,
			&rating.PaymentCompliance,
			&rating.ServiceQualityScore,
			&rating.OverallRating,
			&rating.RatingCategory,
			&rating.CalculationTimestamp,
			&rating.ValidFrom,
			&rating.ValidTo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rating: %w", err)
		}
		ratings = append(ratings, &rating)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return ratings, nil
}

// GetRatingHistoryCount gets the total count of rating history records
func (h *HistoryManager) GetRatingHistoryCount(ctx context.Context, filter *RatingHistoryFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM company_ratings
		WHERE tenant_id = $1
	`

	args := []interface{}{filter.TenantID}
	argIndex := 2

	// Add date filters
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND calculation_timestamp >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND calculation_timestamp <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	// Add category filter
	if filter.Category != nil {
		query += fmt.Sprintf(" AND rating_category = $%d", argIndex)
		args = append(args, *filter.Category)
	}

	var count int
	err := h.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rating history: %w", err)
	}

	return count, nil
}

// GetRatingAtTime gets the rating that was valid at a specific time
func (h *HistoryManager) GetRatingAtTime(ctx context.Context, tenantID uuid.UUID, timestamp time.Time) (*CompanyRating, error) {
	query := `
		SELECT 
			id, tenant_id, success_rate, sending_volume, payment_compliance,
			service_quality_score, overall_rating, rating_category,
			calculation_timestamp, valid_from, valid_to
		FROM company_ratings
		WHERE tenant_id = $1
		  AND valid_from <= $2
		  AND (valid_to IS NULL OR valid_to > $2)
		ORDER BY valid_from DESC
		LIMIT 1
	`

	var rating CompanyRating
	err := h.db.QueryRowContext(ctx, query, tenantID, timestamp).Scan(
		&rating.ID,
		&rating.TenantID,
		&rating.SuccessRate,
		&rating.SendingVolume,
		&rating.PaymentCompliance,
		&rating.ServiceQualityScore,
		&rating.OverallRating,
		&rating.RatingCategory,
		&rating.CalculationTimestamp,
		&rating.ValidFrom,
		&rating.ValidTo,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get rating at time: %w", err)
	}

	return &rating, nil
}

// CleanupOldRecords deletes rating records older than the retention period (12 months)
func (h *HistoryManager) CleanupOldRecords(ctx context.Context) (int64, error) {
	// Calculate cutoff date (12 months ago)
	cutoffDate := time.Now().AddDate(0, -12, 0)

	query := `
		DELETE FROM company_ratings
		WHERE calculation_timestamp < $1
	`

	result, err := h.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetRatingTrend gets rating trend over time
func (h *HistoryManager) GetRatingTrend(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]*CompanyRating, error) {
	query := `
		SELECT 
			id, tenant_id, success_rate, sending_volume, payment_compliance,
			service_quality_score, overall_rating, rating_category,
			calculation_timestamp, valid_from, valid_to
		FROM company_ratings
		WHERE tenant_id = $1
		  AND calculation_timestamp >= $2
		  AND calculation_timestamp <= $3
		ORDER BY calculation_timestamp ASC
	`

	rows, err := h.db.QueryContext(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query rating trend: %w", err)
	}
	defer rows.Close()

	var ratings []*CompanyRating
	for rows.Next() {
		var rating CompanyRating
		err := rows.Scan(
			&rating.ID,
			&rating.TenantID,
			&rating.SuccessRate,
			&rating.SendingVolume,
			&rating.PaymentCompliance,
			&rating.ServiceQualityScore,
			&rating.OverallRating,
			&rating.RatingCategory,
			&rating.CalculationTimestamp,
			&rating.ValidFrom,
			&rating.ValidTo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rating: %w", err)
		}
		ratings = append(ratings, &rating)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return ratings, nil
}

// GetCategoryDistribution gets the distribution of companies across rating categories
func (h *HistoryManager) GetCategoryDistribution(ctx context.Context) (map[RatingCategory]int, error) {
	query := `
		SELECT rating_category, COUNT(*) as count
		FROM company_ratings
		WHERE valid_to IS NULL
		GROUP BY rating_category
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query category distribution: %w", err)
	}
	defer rows.Close()

	distribution := make(map[RatingCategory]int)
	for rows.Next() {
		var category RatingCategory
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan category distribution: %w", err)
		}
		distribution[category] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return distribution, nil
}
