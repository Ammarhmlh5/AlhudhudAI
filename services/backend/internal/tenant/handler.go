package tenant

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/ws"
	jwtpkg "github.com/al-hudhud/backend/pkg/jwt"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/al-hudhud/backend/pkg/password"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{
		service: svc,
	}
}

// ListTenants handles the GET /api/v1/admin/tenants request with pagination and filters
func (h *Handler) ListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	search := c.Query("search")
	status := c.Query("status")
	sector := c.Query("sector")
	minBalStr := c.Query("min_balance")
	maxBalStr := c.Query("max_balance")

	var minBal, maxBal *float64
	if minBalStr != "" {
		if f, err := strconv.ParseFloat(minBalStr, 64); err == nil {
			minBal = &f
		}
	}
	if maxBalStr != "" {
		if f, err := strconv.ParseFloat(maxBalStr, 64); err == nil {
			maxBal = &f
		}
	}

	resp, err := h.service.ListTenants(page, pageSize, search, status, sector, minBal, maxBal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tenants"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetTenantSubscription returns the subscription details for a specific tenant
func (h *Handler) GetTenantSubscription(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	data, err := h.service.GetTenantSubscription(tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetTenantTransactions returns the transaction history for a specific tenant
func (h *Handler) GetTenantTransactions(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	data, err := h.service.GetTenantTransactions(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// RegisterStep1 handles the first stage of registration (Email/Phone + Password)
func (h *Handler) RegisterStep1(c *gin.Context) {
	var req RegisterStep1Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	resp, err := h.service.RegisterTenantStep1(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// VerifyOTP handles the verification of the OTP code
func (h *Handler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.VerifyOTP(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification successful"})
}

// RegisterStep2 handles the final stage of registration (Company info + API Keys)
func (h *Handler) RegisterStep2(c *gin.Context) {
	var req RegisterStep2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tenant, err := h.service.RegisterTenantStep2(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Registration complete. Welcome to ALHudhudAI!",
		"tenant":  tenant,
	})
}

// Register handles the single-step registration
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	resp, err := h.service.RegisterTenant(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// BulkUpdateTenants handles bulk status updates
func (h *Handler) BulkUpdateTenants(c *gin.Context) {
	var req struct {
		IDs      []uuid.UUID `json:"ids" binding:"required"`
		IsActive bool        `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.BulkUpdateStatus(c.Request.Context(), req.IDs, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenants updated successfully"})
}

// BulkDeleteTenants handles bulk deletion
func (h *Handler) BulkDeleteTenants(c *gin.Context) {
	var req struct {
		IDs []uuid.UUID `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenants deleted successfully"})
}

// ImpersonateTenant generates a login token for the specified tenant, allowing admins to login as them
func (h *Handler) ImpersonateTenant(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	tenant, err := h.service.GetTenantByID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	var user models.User
	if err := db.GormDB.First(&user, "id = ?", tenant.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User for tenant not found"})
		return
	}

	permissions, _ := db.GetUserPermissions(user.ID.String())

	token, err := jwtpkg.GenerateToken(user.ID.String(), user.Email, string(models.TenantType), tenant.ID.String(), "", permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate impersonation token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"tenant": tenant,
	})
}

// AdminCreateTenant handles the direct creation of a tenant by an admin
func (h *Handler) AdminCreateTenant(c *gin.Context) {
	var req AdminCreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log := logger.GetLogger()
	log.Info("AdminCreateTenant request received", zap.String("email", req.Email), zap.String("business", req.BusinessName))

	tenant, err := h.service.AdminCreateTenant(req)
	if err != nil {
		log.Error("AdminCreateTenant service call failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info("AdminCreateTenant successful, sending response", zap.String("tenant_id", tenant.ID.String()))
	c.JSON(http.StatusCreated, tenant)
}



// RegenerateAPIKey generates a new secure API key for the isolated tenant
func (h *Handler) RegenerateAPIKey(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}

	// Verify the password
	var user models.User
	if err := db.GormDB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	// Make sure to import github.com/al-hudhud/backend/pkg/password at the top
	if !password.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "كلمة المرور غير صحيحة"})
		return
	}

	// Generate a secure API key using UUID (removing hyphens for cleanliness)
	newKey := strings.ReplaceAll(uuid.New().String(), "-", "")

	// Prefix it for identity checks (hh_live_)
	finalKey := "hh_live_" + newKey

	// Update the tenant's api_key_accounting field via GORM
	result := db.GormDB.Model(&models.Tenant{}).Where("id = ?", tenantID).Update("api_key_accounting", finalKey)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "تم تجديد مفتاح الـ API بنجاح",
		"api_key": finalKey,
	})
}

// GetDashboard returns aggregated dashboard metrics for the authenticated tenant
// GET /api/v1/tenant/dashboard
func (h *Handler) GetDashboard(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	// Message stats
	var totalSent, totalDelivered, totalFailed, totalWaiting int
	db.DB.QueryRow(`
		SELECT
			COUNT(*) FILTER (WHERE status IN ('SENT','DELIVERED')) AS total_sent,
			COUNT(*) FILTER (WHERE status = 'DELIVERED') AS total_delivered,
			COUNT(*) FILTER (WHERE status = 'FAILED') AS total_failed,
			COUNT(*) FILTER (WHERE status IN ('PENDING','QUEUED','PROCESSING')) AS total_waiting
		FROM messages WHERE tenant_id = $1
	`, tenantID).Scan(&totalSent, &totalDelivered, &totalFailed, &totalWaiting)

	var successRate float64
	if totalSent > 0 {
		successRate = float64(totalDelivered) / float64(totalSent) * 100
	}

	// Device stats
	var totalDevices int
	db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM paired_devices WHERE tenant_id = $1
	`, tenantID).Scan(&totalDevices)

	connectedDevices := ws.GlobalHub.GetConnectedDevicesCount(tenantID)

	c.JSON(http.StatusOK, gin.H{
		"messages": gin.H{
			"total_sent":       totalSent,
			"total_delivered":  totalDelivered,
			"total_failed":     totalFailed,
			"total_waiting":    totalWaiting,
			"success_rate":     successRate,
			"send_rate":        0,
			"process_rate":     0,
		},
		"devices": gin.H{
			"total":     totalDevices,
			"connected": connectedDevices,
		},
	})
}

// GetBalance returns the balance and quota info for the authenticated tenant
// GET /api/v1/finance/balance
func (h *Handler) GetBalance(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var balance float64
	var messageLimit, messageUsed int
	err := db.DB.QueryRow(`
		SELECT t.balance, t.message_limit,
		       COALESCE((SELECT COUNT(*) FROM messages m WHERE m.tenant_id = t.id AND m.created_at > NOW() - INTERVAL '30 days'), 0)
		FROM tenants t WHERE t.id = $1 AND t.deleted_at IS NULL
	`, tenantID).Scan(&balance, &messageLimit, &messageUsed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cash_balance":  balance,
		"quota_balance": messageLimit - messageUsed,
		"message_limit": messageLimit,
		"message_used":  messageUsed,
	})
}

