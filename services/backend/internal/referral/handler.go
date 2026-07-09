package referral

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for referral operations
type Handler struct {
	manager *Manager
	logger  *zap.Logger
}

// NewHandler creates a new referral handler
func NewHandler(manager *Manager, logger *zap.Logger) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
	}
}

// GenerateReferralCode handles POST /api/v1/referrals/generate
func (h *Handler) GenerateReferralCode(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware)
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Generate referral code
	code, err := h.manager.GenerateReferralCode(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to generate referral code",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate referral code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       code,
		"tenant_id":  tenantID.String(),
		"message":    "Referral code generated successfully",
	})
}

// ValidateReferralCode handles POST /api/v1/referrals/validate
func (h *Handler) ValidateReferralCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: code is required"})
		return
	}

	// Validate referral code
	info, err := h.manager.ValidateReferralCode(c.Request.Context(), req.Code)
	if err != nil {
		if err == ErrInvalidReferralCode {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invalid referral code"})
			return
		}
		if err == ErrReferralCodeMaxUsage {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Referral code has reached maximum usage",
				"is_valid": false,
			})
			return
		}
		h.logger.Error("Failed to validate referral code",
			zap.String("code", req.Code),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate referral code"})
		return
	}

	c.JSON(http.StatusOK, info)
}

// ProcessReferral handles POST /api/v1/referrals/process
func (h *Handler) ProcessReferral(c *gin.Context) {
	var req CreateReferralRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Process referral
	referral, err := h.manager.ProcessReferral(c.Request.Context(), &req)
	if err != nil {
		if err == ErrSelfReferral {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot refer yourself"})
			return
		}
		if err == ErrAlreadyReferred {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User has already been referred"})
			return
		}
		h.logger.Error("Failed to process referral",
			zap.String("referrer_id", req.ReferrerTenantID.String()),
			zap.String("referee_id", req.RefereeTenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process referral"})
		return
	}

	c.JSON(http.StatusCreated, referral)
}

// GetReferralStats handles GET /api/v1/referrals/stats/:tenantID
func (h *Handler) GetReferralStats(c *gin.Context) {
	tenantIDStr := c.Param("tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Get referral stats
	stats, err := h.manager.GetReferralStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get referral stats",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get referral stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetReferralReport handles GET /api/v1/referrals/report/:tenantID
func (h *Handler) GetReferralReport(c *gin.Context) {
	tenantIDStr := c.Param("tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Get format from query parameter (default: json)
	format := c.DefaultQuery("format", "json")

	// Get referrals for the tenant
	referrals, err := h.manager.GetReferralsByReferrer(c.Request.Context(), tenantID, 1000, 0)
	if err != nil {
		h.logger.Error("Failed to get referrals for report",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}

	// Return report based on format
	switch format {
	case "json":
		c.JSON(http.StatusOK, gin.H{
			"tenant_id":  tenantID.String(),
			"referrals":  referrals,
			"total":      len(referrals),
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported format. Use 'json'"})
	}
}
