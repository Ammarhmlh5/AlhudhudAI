package api

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/referral"
)

// ReferralHandler handles referral-related HTTP requests
type ReferralHandler struct {
	manager *referral.Manager
}

// NewReferralHandler creates a new referral handler
func NewReferralHandler(manager *referral.Manager) *ReferralHandler {
	return &ReferralHandler{
		manager: manager,
	}
}

// RegisterRoutes registers referral routes
func (h *ReferralHandler) RegisterRoutes(r *gin.RouterGroup) {
	referrals := r.Group("/referrals")
	{
		// Public routes (for signup)
		referrals.POST("/validate", h.ValidateReferralCode)
		referrals.POST("/process", h.ProcessReferral)
		
		// Protected routes (require authentication)
		referrals.GET("/code", h.GetReferralCode)
		referrals.GET("/stats", h.GetReferralStats)
		referrals.GET("/list", h.ListReferrals)
		referrals.GET("/performance", h.GetReferralPerformance)
		
		// Admin routes
		referrals.POST("/:id/grant-reward", h.GrantReward)
		referrals.POST("/:id/mark-condition-met", h.MarkConditionMet)
	}
}

// GetReferralCode generates a referral code for the current user
// GET /api/referrals/code
// Requirement 33.1: THE النظام SHALL يولّد كود إحالة فريد لكل عميل
func (h *ReferralHandler) GetReferralCode(c *gin.Context) {
	// Get tenant ID from context (set by auth middleware)
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
		})
		return
	}
	
	tid, ok := tenantID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tenant ID",
		})
		return
	}
	
	// Generate code
	code, err := h.manager.GenerateReferralCode(c.Request.Context(), tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to generate referral code",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"code": code,
			"share_url": "https://example.com/signup?ref=" + code,
		},
	})
}

// ValidateReferralCode validates a referral code
// POST /api/referrals/validate
// Body: { "code": "REF-XXXXXXXX" }
// Requirement 33.2: THE النظام SHALL يسمح للعميل بمشاركة كود الإحالة
func (h *ReferralHandler) ValidateReferralCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}
	
	// Validate code
	info, err := h.manager.ValidateReferralCode(c.Request.Context(), req.Code)
	if err != nil {
		if err == referral.ErrInvalidReferralCode {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "invalid referral code",
			})
			return
		}
		
		if err == referral.ErrReferralCodeMaxUsage {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "referral code has reached maximum usage",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to validate referral code",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    info,
	})
}

// ProcessReferral processes a new referral
// POST /api/referrals/process
// Body: { "referrer_tenant_id": "uuid", "referee_tenant_id": "uuid", "referral_code": "REF-XXX" }
// Requirement 33.3: WHEN يشترك عميل جديد باستخدام كود إحالة، THE النظام SHALL يسجل العلاقة
func (h *ReferralHandler) ProcessReferral(c *gin.Context) {
	var req referral.CreateReferralRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}
	
	// Process referral
	ref, err := h.manager.ProcessReferral(c.Request.Context(), &req)
	if err != nil {
		if err == referral.ErrSelfReferral {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "cannot refer yourself",
			})
			return
		}
		
		if err == referral.ErrAlreadyReferred {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "user has already been referred",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to process referral",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    ref,
	})
}

// GetReferralStats retrieves referral statistics for the current user
// GET /api/referrals/stats
// Requirement 33.7: THE النظام SHALL يعرض للعميل عدد الإحالات الناجحة والمكافآت المكتسبة
func (h *ReferralHandler) GetReferralStats(c *gin.Context) {
	// Get tenant ID from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
		})
		return
	}
	
	tid, ok := tenantID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tenant ID",
		})
		return
	}
	
	// Get stats
	stats, err := h.manager.GetReferralStats(c.Request.Context(), tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get referral stats",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ListReferrals lists all referrals for the current user
// GET /api/referrals/list?limit=10&offset=0
// Requirement 33.7: THE النظام SHALL يعرض للعميل عدد الإحالات الناجحة والمكافآت المكتسبة
func (h *ReferralHandler) ListReferrals(c *gin.Context) {
	// Get tenant ID from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
		})
		return
	}
	
	tid, ok := tenantID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid tenant ID",
		})
		return
	}
	
	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	
	if offset < 0 {
		offset = 0
	}
	
	// Get referrals
	referrals, err := h.manager.GetReferralsByReferrer(c.Request.Context(), tid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get referrals",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"referrals": referrals,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// GetReferralPerformance retrieves overall referral program performance
// GET /api/referrals/performance
// Requirement 33.9: THE النظام SHALL يوفر تقرير بأداء برنامج الإحالة
func (h *ReferralHandler) GetReferralPerformance(c *gin.Context) {
	// Check if user is admin
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "admin access required",
		})
		return
	}
	
	// Get performance
	perf, err := h.manager.GetReferralPerformance(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to get referral performance",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    perf,
	})
}

// GrantReward grants a reward for a referral (admin only)
// POST /api/referrals/:id/grant-reward
// Requirement 33.4: THE النظام SHALL يمنح العميل المُحيل مكافأة عند اشتراك المُحال
func (h *ReferralHandler) GrantReward(c *gin.Context) {
	// Check if user is admin
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "admin access required",
		})
		return
	}
	
	// Parse referral ID
	idStr := c.Param("id")
	referralID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid referral ID",
		})
		return
	}
	
	// Parse request body
	var req struct {
		Notes *string `json:"notes"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		// Notes are optional, so ignore binding errors
	}
	
	// Grant reward
	grantReq := &referral.GrantRewardRequest{
		ReferralID: referralID,
		Notes:      req.Notes,
	}
	
	err = h.manager.GrantReward(c.Request.Context(), grantReq)
	if err != nil {
		if err == referral.ErrReferralNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "referral not found",
			})
			return
		}
		
		if err == referral.ErrConditionNotMet {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "reward condition not met",
			})
			return
		}
		
		if err == referral.ErrRewardAlreadyGranted {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "reward already granted",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to grant reward",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "reward granted successfully",
	})
}

// MarkConditionMet marks the referral condition as met (admin only)
// POST /api/referrals/:id/mark-condition-met
// Requirement 33.6: THE النظام SHALL يدعم تحديد شروط المكافأة
func (h *ReferralHandler) MarkConditionMet(c *gin.Context) {
	// Check if user is admin
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "admin access required",
		})
		return
	}
	
	// Parse referral ID
	idStr := c.Param("id")
	referralID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid referral ID",
		})
		return
	}
	
	// Mark condition as met
	err = h.manager.MarkConditionMet(c.Request.Context(), referralID)
	if err != nil {
		if err == referral.ErrReferralNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "referral not found",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to mark condition met",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "condition marked as met",
	})
}

// Helper function to get tenant ID from database by referral code
func (h *ReferralHandler) getTenantIDByReferralCode(ctx context.Context, code string) (uuid.UUID, error) {
	// This would query the database to find the tenant who owns this code
	// For now, we'll return an error
	return uuid.Nil, sql.ErrNoRows
}
