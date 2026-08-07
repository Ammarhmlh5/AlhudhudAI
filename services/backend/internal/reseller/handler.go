package reseller

import (
	"errors"
	"net/http"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// resellerIDFromContext extracts the reseller tenant ID from the auth context.
// AuthMiddleware stores tenant_id as a string, so we parse it into a uuid.UUID.
// Super admins have no tenant scope and are represented by uuid.Nil.
func resellerIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	if isSuperAdmin(c) {
		return uuid.Nil, true
	}
	id, err := uuid.Parse(c.GetString("tenant_id"))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func isSuperAdmin(c *gin.Context) bool {
	return c.GetString("user_type") == "SUPER_ADMIN"
}

// actingUser returns the email of the authenticated user for audit purposes.
func actingUser(c *gin.Context) string {
	if email := c.GetString("email"); email != "" {
		return email
	}
	return c.GetString("user_id")
}

// ListResellers returns all software companies (resellers) for the admin interface.
// Requires SUPER_ADMIN access (wired under /api/v1/admin/resellers).
func (h *Handler) ListResellers(c *gin.Context) {
	resellers, err := h.service.ListResellers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resellers)
}

// GetResellerDetail returns a single reseller along with its customer companies.
// GET /api/v1/admin/resellers/:id
func (h *Handler) GetResellerDetail(c *gin.Context) {
	resellerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reseller id"})
		return
	}

	var reseller models.Tenant
	if err := h.service.db.WithContext(c.Request.Context()).
		Where("id = ? AND is_reseller = ?", resellerID, true).
		First(&reseller).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "reseller not found"})
		return
	}

	companies, err := h.service.ListTenantsWithDetails(c.Request.Context(), resellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reseller":  reseller,
		"companies": companies,
	})
}

type CreditResellerRequest struct {
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Reason string          `json:"reason"`
}

// CreditReseller funds a reseller's wallet.
// POST /api/v1/admin/resellers/:id/credit
func (h *Handler) CreditReseller(c *gin.Context) {
	resellerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reseller id"})
		return
	}

	var req CreditResellerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreditReseller(
		c.Request.Context(),
		resellerID,
		req.Amount,
		req.Reason,
		actingUser(c),
		c.ClientIP(),
	)
	if err != nil {
		if errors.Is(err, ErrCreditLimitExceeded) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"old_balance": result.OldBalance.String(),
		"new_balance": result.NewBalance.String(),
	})
}

func (h *Handler) GetTenants(c *gin.Context) {
	id, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	tenants, err := h.service.ListTenantsWithDetails(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenants)
}

type CreateTenantRequest struct {
	BusinessName string `json:"business_name" binding:"required"`
	ContactName  string `json:"contact_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone"`
	Password     string `json:"password" binding:"required,min=6"`
}

func (h *Handler) CreateTenant(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		UserType:     models.TenantType,
		IsActive:     true,
		IsVerified:   true,
	}

	tenant := &models.Tenant{
		BusinessName: req.BusinessName,
		ContactName:  req.ContactName,
		Email:        req.Email,
		Phone:        req.Phone,
		IsActive:     true,
	}

	if err := h.service.CreateTenant(c.Request.Context(), rid, tenant, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// GetTenantBalance returns the wallet balance of a customer company.
// GET /api/v1/reseller/tenants/:id/balance
func (h *Handler) GetTenantBalance(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	balance, err := h.service.GetTenantBalance(c.Request.Context(), rid, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"balance":   balance.String(),
	})
}

// GetWalletBalance returns the authenticated reseller's own wallet balance.
// GET /api/v1/reseller/balance
func (h *Handler) GetWalletBalance(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	bal, err := h.service.GetWalletBalance(c.Request.Context(), rid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": bal.String()})
}

type TransferRequest struct {
	Amount decimal.Decimal `json:"amount" binding:"required"`
	Reason string          `json:"reason"`
}

// TransferToTenant moves funds from the reseller wallet to a customer company.
// POST /api/v1/reseller/tenants/:id/transfer
func (h *Handler) TransferToTenant(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.TransferToTenant(
		c.Request.Context(),
		rid,
		tenantID,
		req.Amount,
		req.Reason,
		actingUser(c),
		c.ClientIP(),
	)
	if err != nil {
		var balErr *balance.BalanceError
		if errors.As(err, &balErr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": balErr.Error(), "type": balErr.Type})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type SubscribeTenantRequest struct {
	PackageID int64 `json:"package_id" binding:"required"`
	AutoRenew bool  `json:"auto_renew"`
}

// SubscribeTenant subscribes a customer company to a package (charged to the
// customer company's wallet).
// POST /api/v1/reseller/tenants/:id/subscribe
func (h *Handler) SubscribeTenant(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	var req SubscribeTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.SubscribeTenant(c.Request.Context(), rid, tenantID, req.PackageID, req.AutoRenew)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !result.Success {
		status := http.StatusBadRequest
		if result.Message == "Insufficient balance" {
			status = http.StatusPaymentRequired
		}
		c.JSON(status, gin.H{
			"success":    false,
			"message":    result.Message,
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

func (h *Handler) UpdateAPIKey(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	tenantIDStr := c.Param("id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateTenantAPIKeys(c.Request.Context(), rid, tenantID, req.APIKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
}

func (h *Handler) GetStaff(c *gin.Context) {
	rid, ok := resellerIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	staff, err := h.service.ListStaff(c.Request.Context(), rid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, staff)
}
