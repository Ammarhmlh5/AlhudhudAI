package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/al-hudhud/backend/internal/rollover"
	"github.com/al-hudhud/backend/internal/subscription"
)

// SubscriptionHandler handles subscription management APIs
type SubscriptionHandler struct {
	subManager      subscription.Manager
	rolloverManager rollover.Manager
	pool            *pgxpool.Pool
	logger          *zap.Logger
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(
	subManager subscription.Manager,
	rolloverManager rollover.Manager,
	pool *pgxpool.Pool,
	logger *zap.Logger,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subManager:      subManager,
		rolloverManager: rolloverManager,
		pool:            pool,
		logger:          logger,
	}
}

// SubscribeRequest represents a request to subscribe to a package
type SubscribeRequest struct {
	PackageID int64 `json:"package_id" binding:"required"`
	AutoRenew bool  `json:"auto_renew"`
}

// SubscribeResponse represents the response for subscription
type SubscribeResponse struct {
	Success        bool    `json:"success"`
	SubscriptionID int64   `json:"subscription_id,omitempty"`
	Message        string  `json:"message"`
	OldBalance     string  `json:"old_balance,omitempty"`
	NewBalance     string  `json:"new_balance,omitempty"`
}

// Subscribe subscribes a tenant to a package
// Implements Requirements 4.1, 4.2, 21.3
// - Tenant validation
// - Balance check
// - Atomic subscription creation
// - Error handling for insufficient balance and inactive packages
//
// POST /api/v1/subscriptions
func (h *SubscriptionHandler) Subscribe(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context (set by middleware)
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		h.logger.Error("tenant_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		h.logger.Error("Invalid tenant_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Parse request body
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Validate package_id
	if req.PackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid package_id",
		})
		return
	}

	// Call subscription manager
	subReq := subscription.SubscribeRequest{
		TenantID:  tenantID,
		PackageID: req.PackageID,
		AutoRenew: req.AutoRenew,
	}

	result, err := h.subManager.Subscribe(ctx, subReq)
	if err != nil {
		h.logger.Error("Failed to subscribe",
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

	// Handle subscription result
	if !result.Success {
		// Business logic failure (insufficient balance, inactive package, etc.)
		statusCode := http.StatusBadRequest
		if result.Message == "Insufficient balance" {
			statusCode = http.StatusPaymentRequired
		} else if result.Message == "Package is not active" {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, SubscribeResponse{
			Success:    false,
			Message:    result.Message,
			OldBalance: result.OldBalance.String(),
			NewBalance: result.NewBalance.String(),
		})
		return
	}

	// Success
	h.logger.Info("Subscription created successfully",
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

// GetUsageResponse represents the response for usage information
type GetUsageResponse struct {
	Success           bool                  `json:"success"`
	Usage             *UsageInfo            `json:"usage,omitempty"`
	RolloverMessages  []RolloverMessageInfo `json:"rollover_messages,omitempty"`
}

// UsageInfo represents subscription usage information
type UsageInfo struct {
	SubscriptionID    int64  `json:"subscription_id"`
	PackageID         int64  `json:"package_id"`
	PackageName       string `json:"package_name"`
	MessagesUsed      int64  `json:"messages_used"`
	MessagesRemaining *int64 `json:"messages_remaining"` // NULL for unlimited
	IncludedMessages  *int64 `json:"included_messages"`  // NULL for unlimited
	OverageMessages   int64  `json:"overage_messages"`
	OverageCost       string `json:"overage_cost"`
	OverageRate       string `json:"overage_rate"`
	Status            string `json:"status"`
	StartDate         string `json:"start_date"`
	EndDate           string `json:"end_date"`
	NextRenewalDate   string `json:"next_renewal_date,omitempty"`
}

// RolloverMessageInfo represents rollover message information
type RolloverMessageInfo struct {
	ID            int64  `json:"id"`
	MessagesCount int64  `json:"messages_count"`
	SourceMonth   string `json:"source_month"`
	ExpiryDate    string `json:"expiry_date"`
	CreatedAt     string `json:"created_at"`
}

// GetUsage returns subscription usage information
// Implements Requirements 4.6, 21.5, 21.6, 21.9
// - Tenant isolation
// - Displays messages used and remaining
// - Displays overage messages and cost
// - Displays rollover messages with expiry dates
//
// GET /api/v1/subscriptions/usage
func (h *SubscriptionHandler) GetUsage(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		h.logger.Error("tenant_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		h.logger.Error("Invalid tenant_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Get usage information
	usage, err := h.subManager.GetUsage(ctx, tenantID)
	if err != nil {
		h.logger.Error("Failed to get usage",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No active subscription found",
		})
		return
	}

	// Convert to response format
	usageInfo := &UsageInfo{
		SubscriptionID:    usage.SubscriptionID,
		PackageID:         usage.PackageID,
		PackageName:       usage.PackageName,
		MessagesUsed:      usage.MessagesUsed,
		MessagesRemaining: usage.MessagesRemaining,
		IncludedMessages:  usage.IncludedMessages,
		OverageMessages:   usage.OverageMessages,
		OverageCost:       usage.OverageCost.String(),
		OverageRate:       usage.OverageRate.String(),
		Status:            string(usage.Status),
		StartDate:         usage.StartDate.Format("2006-01-02"),
		EndDate:           usage.EndDate.Format("2006-01-02"),
	}

	if usage.NextRenewalDate != nil {
		usageInfo.NextRenewalDate = usage.NextRenewalDate.Format("2006-01-02")
	}

	// Get rollover messages (Requirement 21.9)
	rolloverMessages, err := h.rolloverManager.GetRolloverMessages(ctx, usage.SubscriptionID)
	if err != nil {
		h.logger.Warn("Failed to get rollover messages",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("subscription_id", usage.SubscriptionID),
			zap.Error(err),
		)
		// Continue without rollover messages
		rolloverMessages = []rollover.RolloverMessage{}
	}

	// Convert rollover messages to response format
	rolloverInfo := make([]RolloverMessageInfo, 0, len(rolloverMessages))
	for _, rm := range rolloverMessages {
		rolloverInfo = append(rolloverInfo, RolloverMessageInfo{
			ID:            rm.ID,
			MessagesCount: rm.MessagesCount,
			SourceMonth:   rm.SourceMonth.Format("2006-01"),
			ExpiryDate:    rm.ExpiryDate.Format("2006-01-02"),
			CreatedAt:     rm.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, GetUsageResponse{
		Success:          true,
		Usage:            usageInfo,
		RolloverMessages: rolloverInfo,
	})
}

// GetSubscription returns the active subscription for a tenant
// GET /api/v1/subscriptions/active
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Get active subscription
	sub, err := h.subManager.GetActiveSubscription(ctx, tenantID)
	if err != nil {
		h.logger.Error("Failed to get active subscription",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "No active subscription found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"subscription": sub,
	})
}

// CancelSubscription cancels auto-renewal for a subscription
// PUT /api/v1/subscriptions/:id/cancel
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Parse subscription ID
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid subscription ID",
		})
		return
	}

	// Update auto_renew to false
	query := `
		UPDATE subscriptions
		SET auto_renew = false, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id
	`

	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"

	var updatedID int64
	err = db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, subscriptionID, tenantID).Scan(&updatedID)
	})
	if err != nil {
		h.logger.Error("Failed to cancel subscription",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("subscription_id", subscriptionID),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Subscription not found",
		})
		return
	}

	h.logger.Info("Subscription auto-renewal cancelled",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("subscription_id", subscriptionID),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto-renewal cancelled successfully",
	})
}

// EnableAutoRenew enables auto-renewal for a subscription
// PUT /api/v1/subscriptions/:id/enable-auto-renew
func (h *SubscriptionHandler) EnableAutoRenew(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Parse subscription ID
	subscriptionIDStr := c.Param("id")
	subscriptionID, err := strconv.ParseInt(subscriptionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid subscription ID",
		})
		return
	}

	// Update auto_renew to true
	query := `
		UPDATE subscriptions
		SET auto_renew = true, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id
	`

	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"

	var updatedID int64
	err = db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, subscriptionID, tenantID).Scan(&updatedID)
	})
	if err != nil {
		h.logger.Error("Failed to enable auto-renewal",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("subscription_id", subscriptionID),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Subscription not found",
		})
		return
	}

	h.logger.Info("Subscription auto-renewal enabled",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("subscription_id", subscriptionID),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Auto-renewal enabled successfully",
	})
}
