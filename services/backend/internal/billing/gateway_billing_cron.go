package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// GatewayBillingCron handles scheduled gateway billing checks
type GatewayBillingCron struct {
	service       *GatewayBillingService
	checkInterval time.Duration
	logger        *zap.Logger
}

// NewGatewayBillingCron creates a new gateway billing cron job
func NewGatewayBillingCron(pool *pgxpool.Pool, logger *zap.Logger, checkInterval time.Duration) *GatewayBillingCron {
	if checkInterval == 0 {
		checkInterval = 24 * time.Hour // Default: check daily
	}

	return &GatewayBillingCron{
		service:       NewGatewayBillingService(pool, logger),
		checkInterval: checkInterval,
		logger:        logger,
	}
}

// Start starts the periodic gateway billing check
// This is the Cron Job implementation for automatic monthly/yearly billing
// Implements Requirements 8.10, 8.11: Automatic subscription and rental billing
func (gbc *GatewayBillingCron) Start(ctx context.Context) {
	ticker := time.NewTicker(gbc.checkInterval)
	defer ticker.Stop()

	gbc.logger.Info("Starting periodic gateway billing check",
		zap.Duration("interval", gbc.checkInterval),
	)

	// Run immediately on start
	if err := gbc.service.CheckDueGateways(ctx); err != nil {
		gbc.logger.Error("Error checking due gateways on startup", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			gbc.logger.Info("Stopping periodic gateway billing check")
			return
		case <-ticker.C:
			gbc.logger.Info("Running scheduled gateway billing check")
			if err := gbc.service.CheckDueGateways(ctx); err != nil {
				gbc.logger.Error("Error checking due gateways", zap.Error(err))
			}
		}
	}
}

// RunOnce runs the billing check once (useful for testing or manual triggers)
func (gbc *GatewayBillingCron) RunOnce(ctx context.Context) error {
	gbc.logger.Info("Running manual gateway billing check")
	return gbc.service.CheckDueGateways(ctx)
}

// GetUpcomingBillings returns gateways with upcoming billing dates
func (gbc *GatewayBillingCron) GetUpcomingBillings(ctx context.Context, daysAhead int) ([]GatewayBillingInfo, error) {
	return gbc.service.GetUpcomingBillings(ctx, daysAhead)
}

// SendBillingReminders sends reminders for upcoming billings
// This can be called before the actual billing date to notify enterprise companies
func (gbc *GatewayBillingCron) SendBillingReminders(ctx context.Context, daysAhead int) error {
	gbc.logger.Info("Sending billing reminders",
		zap.Int("days_ahead", daysAhead),
	)

	billings, err := gbc.service.GetUpcomingBillings(ctx, daysAhead)
	if err != nil {
		return fmt.Errorf("failed to get upcoming billings: %w", err)
	}

	gbc.logger.Info("Found upcoming billings",
		zap.Int("count", len(billings)),
	)

	for _, billing := range billings {
		// Calculate days until billing
		daysUntil := int(time.Until(billing.NextBillingDate).Hours() / 24)
		
		gbc.logger.Info("Upcoming billing",
			zap.Int64("gateway_id", billing.GatewayID),
			zap.String("gateway_name", billing.GatewayName),
			zap.String("tenant_id", billing.TenantID.String()),
			zap.Int("days_until", daysUntil),
			zap.String("amount", billing.SubscriptionFee.Add(billing.ServerRentalFee).String()),
		)

		// TODO: Send notification to enterprise company
		// This should integrate with the notification service (Task 13)
		// Example:
		// notificationService.SendBillingReminder(ctx, billing)
	}

	return nil
}
