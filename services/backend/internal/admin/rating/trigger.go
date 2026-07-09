package rating

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Trigger handles rating update events with debouncing
type Trigger struct {
	db         *sqlx.DB
	collector  *Collector
	calculator *Calculator
	history    *HistoryManager
	
	// Debouncing state
	lastUpdate map[uuid.UUID]time.Time
	mu         sync.RWMutex
	
	// Configuration
	debounceInterval time.Duration // 5 seconds
}

// NewTrigger creates a new Trigger
func NewTrigger(db *sqlx.DB, collector *Collector, calculator *Calculator, history *HistoryManager) *Trigger {
	return &Trigger{
		db:               db,
		collector:        collector,
		calculator:       calculator,
		history:          history,
		lastUpdate:       make(map[uuid.UUID]time.Time),
		debounceInterval: 5 * time.Second,
	}
}

// OnEvent handles a rating update event
func (t *Trigger) OnEvent(ctx context.Context, event *RatingUpdateEvent) error {
	// Check if we should update (debouncing)
	if !t.shouldUpdate(event.TenantID) {
		return nil
	}

	// Update rating asynchronously
	go func() {
		if err := t.UpdateRating(context.Background(), event.TenantID); err != nil {
			// Log error (in production, use proper logging)
			fmt.Printf("Error updating rating for tenant %s: %v\n", event.TenantID, err)
		}
	}()

	return nil
}

// OnMessageSent handles message sent event
func (t *Trigger) OnMessageSent(ctx context.Context, tenantID uuid.UUID) error {
	return t.OnEvent(ctx, &RatingUpdateEvent{
		EventType: "message_sent",
		TenantID:  tenantID,
		Timestamp: time.Now(),
	})
}

// OnMessageFailed handles message failed event
func (t *Trigger) OnMessageFailed(ctx context.Context, tenantID uuid.UUID) error {
	return t.OnEvent(ctx, &RatingUpdateEvent{
		EventType: "message_failed",
		TenantID:  tenantID,
		Timestamp: time.Now(),
	})
}

// OnPaymentReceived handles payment received event
func (t *Trigger) OnPaymentReceived(ctx context.Context, tenantID uuid.UUID) error {
	return t.OnEvent(ctx, &RatingUpdateEvent{
		EventType: "payment_received",
		TenantID:  tenantID,
		Timestamp: time.Now(),
	})
}

// shouldUpdate checks if we should update the rating (debouncing)
func (t *Trigger) shouldUpdate(tenantID uuid.UUID) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	lastUpdate, exists := t.lastUpdate[tenantID]
	if !exists {
		t.lastUpdate[tenantID] = time.Now()
		return true
	}

	// Check if enough time has passed since last update
	if time.Since(lastUpdate) >= t.debounceInterval {
		t.lastUpdate[tenantID] = time.Now()
		return true
	}

	return false
}

// UpdateRating updates the rating for a company
func (t *Trigger) UpdateRating(ctx context.Context, tenantID uuid.UUID) error {
	// Collect current metrics
	metrics, err := t.collector.CollectCurrentMetrics(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to collect metrics: %w", err)
	}

	// Calculate rating
	rating := t.calculator.CalculateRating(metrics)
	category := t.calculator.ClassifyRating(rating)

	// Get previous rating to check for category change
	previousRating, err := t.history.GetLatestRating(ctx, tenantID)
	categoryChanged := false
	if err == nil && previousRating != nil {
		categoryChanged = previousRating.RatingCategory != category
	}

	// Create new rating record
	now := time.Now()
	newRating := &CompanyRating{
		TenantID:             tenantID,
		SuccessRate:          metrics.SuccessRate,
		SendingVolume:        metrics.TotalMessages,
		PaymentCompliance:    metrics.PaymentCompliance,
		ServiceQualityScore:  t.calculator.CalculateQualityScore(metrics),
		OverallRating:        rating,
		RatingCategory:       category,
		CalculationTimestamp: now,
		ValidFrom:            now,
	}

	// Save rating history
	if err := t.history.SaveRatingHistory(ctx, newRating); err != nil {
		return fmt.Errorf("failed to save rating history: %w", err)
	}

	// If category changed, trigger alert
	if categoryChanged {
		if err := t.triggerCategoryChangeAlert(ctx, tenantID, previousRating.RatingCategory, category, rating); err != nil {
			// Log error but don't fail the update
			fmt.Printf("Error triggering category change alert: %v\n", err)
		}
	}

	return nil
}

// triggerCategoryChangeAlert triggers an alert when rating category changes
func (t *Trigger) triggerCategoryChangeAlert(ctx context.Context, tenantID uuid.UUID, oldCategory, newCategory RatingCategory, newRating float64) error {
	// Determine severity based on category change
	severity := "info"
	if newCategory == Poor {
		severity = "critical"
	} else if newCategory == Average && oldCategory == Good {
		severity = "warning"
	} else if newCategory == Good && oldCategory == Excellent {
		severity = "warning"
	}

	// Create alert
	query := `
		INSERT INTO alerts (
			tenant_id, alert_type, severity, title, message, context, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	title := fmt.Sprintf("Company Rating Category Changed: %s → %s", oldCategory, newCategory)
	message := fmt.Sprintf("Company rating has changed from %s to %s (rating: %.2f)", oldCategory, newCategory, newRating)
	context := fmt.Sprintf(`{"old_category": "%s", "new_category": "%s", "rating": %.2f}`, oldCategory, newCategory, newRating)

	_, err := t.db.ExecContext(ctx, query,
		tenantID,
		"rating_change",
		severity,
		title,
		message,
		context,
		"active",
	)

	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}

// UpdateAllRatings updates ratings for all companies (for batch processing)
func (t *Trigger) UpdateAllRatings(ctx context.Context) error {
	// Get all tenant IDs
	query := `SELECT DISTINCT tenant_id FROM tenants WHERE is_active = true`
	
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	var tenantIDs []uuid.UUID
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return fmt.Errorf("failed to scan tenant ID: %w", err)
		}
		tenantIDs = append(tenantIDs, tenantID)
	}

	// Update ratings for all tenants
	for _, tenantID := range tenantIDs {
		if err := t.UpdateRating(ctx, tenantID); err != nil {
			// Log error but continue with other tenants
			fmt.Printf("Error updating rating for tenant %s: %v\n", tenantID, err)
		}
	}

	return nil
}

// StartPeriodicUpdates starts periodic rating updates (e.g., every hour)
func (t *Trigger) StartPeriodicUpdates(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := t.UpdateAllRatings(ctx); err != nil {
				fmt.Printf("Error in periodic rating update: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
