package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *server) tenantFromCtx(c *gin.Context) (uuid.UUID, bool) {
	tenantID, err := uuid.Parse(c.GetString(ctxTenantID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid tenant context"})
		return uuid.Nil, false
	}
	return tenantID, true
}

// GET /api/v1/packages
func (s *server) handleListPackages(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}

	onlyActive := c.DefaultQuery("is_active", "true") != "false"

	type pkg struct {
		ID                   int64           `json:"id"`
		TenantID             uuid.UUID       `json:"tenant_id"`
		ServiceID            int64           `json:"service_id"`
		Name                 string          `json:"name"`
		Code                 string          `json:"code"`
		PriceUSD             decimal.Decimal `json:"price_usd"`
		IncludedMessages     *int64          `json:"included_messages"`
		OverageRate          decimal.Decimal `json:"overage_rate"`
		IsUnlimited          bool            `json:"is_unlimited"`
		BillingCycle         string          `json:"billing_cycle"`
		RolloverEnabled      bool            `json:"rollover_enabled"`
		RolloverLimit        *int64          `json:"rollover_limit"`
		RolloverExpiryMonths *int            `json:"rollover_expiry_months"`
		IsActive             bool            `json:"is_active"`
		DisplayOrder         int             `json:"display_order"`
		CreatedAt            interface{}     `json:"created_at"`
		UpdatedAt            interface{}     `json:"updated_at"`
	}

	var packages []pkg
	err := s.withTenantTx(c.Request.Context(), tenantID, func(tx pgx.Tx) error {
		query := `
			SELECT id, tenant_id, service_id, name, code, price_usd,
			       included_messages, overage_rate, is_unlimited, billing_cycle,
			       rollover_enabled, rollover_limit, rollover_expiry_months,
			       is_active, display_order, created_at, updated_at
			FROM packages
			WHERE tenant_id = $1
		`
		if onlyActive {
			query += " AND is_active = TRUE"
		}
		query += " ORDER BY display_order ASC, created_at DESC"

		rows, err := tx.Query(c.Request.Context(), query, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var p pkg
			if err := rows.Scan(
				&p.ID, &p.TenantID, &p.ServiceID, &p.Name, &p.Code, &p.PriceUSD,
				&p.IncludedMessages, &p.OverageRate, &p.IsUnlimited, &p.BillingCycle,
				&p.RolloverEnabled, &p.RolloverLimit, &p.RolloverExpiryMonths,
				&p.IsActive, &p.DisplayOrder, &p.CreatedAt, &p.UpdatedAt,
			); err != nil {
				return err
			}
			packages = append(packages, p)
		}
		return rows.Err()
	})
	if err != nil {
		s.logger.Error("failed to list packages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list packages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"packages": packages})
}

// POST /api/v1/subscriptions
func (s *server) handleSubscribe(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}

	var req struct {
		PackageID int64  `json:"package_id"`
		AutoRenew bool   `json:"auto_renew"`
		PromoCode string `json:"promo_code,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.PackageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid package_id"})
		return
	}

	result, err := s.subManager.Subscribe(c.Request.Context(), subscription.SubscribeRequest{
		TenantID:  tenantID,
		PackageID: req.PackageID,
		AutoRenew: req.AutoRenew,
		PromoCode: req.PromoCode,
	})
	if err != nil {
		s.logger.Error("company subscribe failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process subscription"})
		return
	}

	if !result.Success {
		status := http.StatusBadRequest
		if result.Message == "Insufficient balance" {
			status = http.StatusPaymentRequired
		}
		c.JSON(status, gin.H{
			"success":     false,
			"message":     result.Message,
			"old_balance": result.OldBalance.String(),
			"new_balance": result.NewBalance.String(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":         true,
		"subscription_id": result.SubscriptionID,
		"message":         result.Message,
		"old_balance":     result.OldBalance.String(),
		"new_balance":     result.NewBalance.String(),
	})
}

// GET /api/v1/subscriptions/active
func (s *server) handleActiveSubscription(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}

	type sub struct {
		ID                int64      `json:"id"`
		PackageID         int64      `json:"package_id"`
		StartDate         interface{} `json:"start_date"`
		EndDate           interface{} `json:"end_date"`
		MessagesUsed      int64      `json:"messages_used"`
		MessagesRemaining *int64     `json:"messages_remaining"`
		OverageMessages   int64      `json:"overage_messages"`
		OverageCost       decimal.Decimal `json:"overage_cost"`
		Status            string     `json:"status"`
		AutoRenew         bool       `json:"auto_renew"`
		GracePeriodDays   int        `json:"grace_period_days"`
		NextRenewalDate   *interface{} `json:"next_renewal_date,omitempty"`
	}

	var result *sub
	err := s.withTenantTx(c.Request.Context(), tenantID, func(tx pgx.Tx) error {
		var s sub
		err := tx.QueryRow(c.Request.Context(), `
			SELECT id, package_id, start_date, end_date,
			       messages_used, messages_remaining, overage_messages, overage_cost,
			       status, auto_renew, grace_period_days, next_renewal_date
			FROM subscriptions
			WHERE tenant_id = $1
			  AND status = 'active'
			  AND end_date >= CURRENT_DATE
			ORDER BY created_at DESC
			LIMIT 1
		`, tenantID).Scan(
			&s.ID, &s.PackageID, &s.StartDate, &s.EndDate,
			&s.MessagesUsed, &s.MessagesRemaining, &s.OverageMessages, &s.OverageCost,
			&s.Status, &s.AutoRenew, &s.GracePeriodDays, &s.NextRenewalDate,
		)
		if err != nil {
			return err
		}
		result = &s
		return nil
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "No active subscription found"})
			return
		}
		s.logger.Error("failed to get active subscription", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "subscription": result})
}

// GET /api/v1/subscriptions/usage
func (s *server) handleUsage(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}

	type usage struct {
		SubscriptionID    int64       `json:"subscription_id"`
		PackageID         int64       `json:"package_id"`
		PackageName       string      `json:"package_name"`
		MessagesUsed      int64       `json:"messages_used"`
		MessagesRemaining *int64      `json:"messages_remaining"`
		IncludedMessages  *int64      `json:"included_messages"`
		OverageMessages   int64       `json:"overage_messages"`
		OverageCost       decimal.Decimal `json:"overage_cost"`
		OverageRate       decimal.Decimal `json:"overage_rate"`
		Status            string      `json:"status"`
		StartDate         interface{} `json:"start_date"`
		EndDate           interface{} `json:"end_date"`
	}

	var result *usage
	err := s.withTenantTx(c.Request.Context(), tenantID, func(tx pgx.Tx) error {
		var u usage
		err := tx.QueryRow(c.Request.Context(), `
			SELECT s.id, s.package_id, s.start_date, s.end_date,
			       s.messages_used, s.messages_remaining, s.overage_messages, s.overage_cost,
			       s.status,
			       p.name, p.included_messages, p.overage_rate
			FROM subscriptions s
			JOIN packages p ON s.package_id = p.id
			WHERE s.tenant_id = $1
			  AND s.status = 'active'
			  AND s.end_date >= CURRENT_DATE
			ORDER BY s.created_at DESC
			LIMIT 1
		`, tenantID).Scan(
			&u.SubscriptionID, &u.PackageID, &u.StartDate, &u.EndDate,
			&u.MessagesUsed, &u.MessagesRemaining, &u.OverageMessages, &u.OverageCost,
			&u.Status,
			&u.PackageName, &u.IncludedMessages, &u.OverageRate,
		)
		if err != nil {
			return err
		}
		result = &u
		return nil
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "No active subscription found"})
			return
		}
		s.logger.Error("failed to get usage", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "usage": result})
}

// PUT /api/v1/subscriptions/:id/cancel
func (s *server) handleCancelAutoRenew(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}
	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || subID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	err = s.updateAutoRenew(c.Request.Context(), tenantID, subID, false)
	if err != nil {
		s.logger.Error("failed to disable auto-renew", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Auto-renewal cancelled successfully"})
}

// PUT /api/v1/subscriptions/:id/enable-auto-renew
func (s *server) handleEnableAutoRenew(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}
	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || subID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	err = s.updateAutoRenew(c.Request.Context(), tenantID, subID, true)
	if err != nil {
		s.logger.Error("failed to enable auto-renew", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Subscription not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Auto-renewal enabled successfully"})
}

func (s *server) updateAutoRenew(ctx context.Context, tenantID uuid.UUID, subID int64, enabled bool) error {
	return s.withTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE subscriptions
			SET auto_renew = $1, updated_at = NOW()
			WHERE id = $2 AND tenant_id = $3
			RETURNING id
		`, enabled, subID, tenantID).Scan(&subID)
	})
}

// GET /api/v1/balance
func (s *server) handleBalance(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}

	var balance decimal.Decimal
	var businessName, email string
	err := s.withTenantTx(c.Request.Context(), tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(c.Request.Context(), `
			SELECT balance, COALESCE(business_name, ''), COALESCE(email, '')
			FROM tenants WHERE id = $1
		`, tenantID).Scan(&balance, &businessName, &email)
	})
	if err != nil {
		s.logger.Error("failed to get balance", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"balance":       balance,
		"business_name": businessName,
		"email":         email,
	})
}

// GET /api/v1/transactions
func (s *server) handleTransactions(c *gin.Context) {
	tenantID, ok := s.tenantFromCtx(c)
	if !ok {
		return
	}

	type txRow struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Amount    decimal.Decimal `json:"amount"`
		Status    string          `json:"status"`
		CreatedAt interface{}     `json:"created_at"`
	}

	var rows_ []txRow
	err := s.withTenantTx(c.Request.Context(), tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(c.Request.Context(), `
			SELECT id, type, amount, status, created_at
			FROM accounting_transactions
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT 50
		`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var r txRow
			if err := rows.Scan(&r.ID, &r.Type, &r.Amount, &r.Status, &r.CreatedAt); err != nil {
				return err
			}
			rows_ = append(rows_, r)
		}
		return rows.Err()
	})
	if err != nil {
		s.logger.Error("failed to get transactions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "transactions": rows_})
}
