package api

import (
	"fmt"
	"net/http"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AdminSubscriptionHandler manages tenant subscriptions on behalf of super admins.
// Unlike the tenant-facing SubscriptionHandler, the target tenant_id comes from
// the URL path (not the JWT context) so admins can manage any tenant.
type AdminSubscriptionHandler struct {
	subManager subscription.Manager
	pool       *pgxpool.Pool
	logger     *zap.Logger
}

// NewAdminSubscriptionHandler creates a new admin subscription handler
func NewAdminSubscriptionHandler(
	subManager subscription.Manager,
	pool *pgxpool.Pool,
	logger *zap.Logger,
) *AdminSubscriptionHandler {
	return &AdminSubscriptionHandler{
		subManager: subManager,
		pool:       pool,
		logger:     logger,
	}
}

// Subscribe assigns a package to a tenant (create or change package).
// The tenant's balance is charged exactly as if the tenant subscribed themselves.
//
// POST /api/v1/admin/tenants/:id/subscribe
func (h *AdminSubscriptionHandler) Subscribe(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant ID",
		})
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if req.PackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid package_id",
		})
		return
	}

	result, err := h.subManager.Subscribe(ctx, subscription.SubscribeRequest{
		TenantID:  tenantID,
		PackageID: req.PackageID,
		AutoRenew: req.AutoRenew,
	})
	if err != nil {
		h.logger.Error("Admin subscribe failed",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("package_id", req.PackageID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to process subscription",
		})
		return
	}

	if !result.Success {
		statusCode := http.StatusBadRequest
		if result.Message == "Insufficient balance" {
			statusCode = http.StatusPaymentRequired
		}
		h.logger.Warn("Admin subscribe rejected",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("package_id", req.PackageID),
			zap.String("message", result.Message),
		)
		c.JSON(statusCode, SubscribeResponse{
			Success:    false,
			Message:    result.Message,
			OldBalance: result.OldBalance.String(),
			NewBalance: result.NewBalance.String(),
		})
		return
	}

	// Deactivate any previous active/grace subscriptions so only the new one counts.
	err = db.RunInTenantPgxTx(ctx, tenantID, true, h.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE subscriptions
			 SET status = 'expired', updated_at = NOW()
			 WHERE tenant_id = $1 AND id != $2 AND status IN ('active', 'grace_period')`,
			tenantID, result.SubscriptionID,
		)
		return err
	})
	if err != nil {
		h.logger.Warn("Failed to deactivate previous subscriptions",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
	}

	h.logger.Info("Admin changed tenant subscription",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("subscription_id", result.SubscriptionID),
		zap.Int64("package_id", req.PackageID),
	)

	c.JSON(http.StatusCreated, SubscribeResponse{
		Success:        true,
		SubscriptionID: result.SubscriptionID,
		Message:        result.Message,
		OldBalance:     result.OldBalance.String(),
		NewBalance:     result.NewBalance.String(),
	})
}

// SetAutoRenew enables or disables auto-renewal for a tenant's active subscription.
//
// PUT /api/v1/admin/tenants/:id/subscription/auto-renew
func (h *AdminSubscriptionHandler) SetAutoRenew(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant ID",
		})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	var updatedID int64
	err = db.RunInTenantPgxTx(ctx, tenantID, true, h.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`UPDATE subscriptions
			 SET auto_renew = $1, updated_at = NOW()
			 WHERE tenant_id = $2 AND status IN ('active', 'grace_period')
			 RETURNING id`,
			req.Enabled, tenantID,
		).Scan(&updatedID)
	})
	if err != nil {
		h.logger.Error("Failed to update auto-renew",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No active subscription found",
		})
		return
	}

	state := "disabled"
	if req.Enabled {
		state = "enabled"
	}

	h.logger.Info("Admin toggled tenant auto-renewal",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("subscription_id", updatedID),
		zap.Bool("enabled", req.Enabled),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Auto-renewal %s successfully", state),
	})
}
