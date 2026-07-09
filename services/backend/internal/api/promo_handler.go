package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	
	"github.com/al-hudhud/backend/internal/promo"
)

// PromoHandler معالج API لأكواد الخصم
type PromoHandler struct {
	manager   *promo.Manager
	tracker   *promo.UsageTracker
	reporter  *promo.ReportGenerator
}

// NewPromoHandler ينشئ معالج API جديد
func NewPromoHandler(db *sqlx.DB) *PromoHandler {
	return &PromoHandler{
		manager:  promo.NewManager(db),
		tracker:  promo.NewUsageTracker(db),
		reporter: promo.NewReportGenerator(db),
	}
}

// CreatePromoCode ينشئ كود خصم جديد
// POST /api/admin/promo-codes
// Requirements: 32.1
func (h *PromoHandler) CreatePromoCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req promo.CreatePromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	promoCode, err := h.manager.CreatePromoCode(c.Request.Context(), tenantID, &req)
	if err != nil {
		if err == promo.ErrPromoCodeAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create promo code"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    promoCode,
	})
}

// ListPromoCodes يعرض قائمة أكواد الخصم
// GET /api/admin/promo-codes?active_only=true
// Requirements: 32.1
func (h *PromoHandler) ListPromoCodes(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	activeOnly := c.Query("active_only") == "true"

	promoCodes, err := h.manager.ListPromoCodes(c.Request.Context(), tenantID, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list promo codes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    promoCodes,
		"count":   len(promoCodes),
	})
}

// GetPromoCode يحصل على تفاصيل كود خصم
// GET /api/admin/promo-codes/:id
func (h *PromoHandler) GetPromoCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	// الحصول على الكود من القائمة (يمكن تحسينه بإضافة دالة GetByID)
	promoCodes, err := h.manager.ListPromoCodes(c.Request.Context(), tenantID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get promo code"})
		return
	}

	for _, pc := range promoCodes {
		if pc.ID == promoCodeID {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    pc,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "promo code not found"})
}

// UpdatePromoCode يحدث كود خصم
// PUT /api/admin/promo-codes/:id
// Requirements: 32.1
func (h *PromoHandler) UpdatePromoCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.manager.UpdatePromoCode(c.Request.Context(), tenantID, promoCodeID, updates)
	if err != nil {
		if err == promo.ErrPromoCodeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update promo code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "promo code updated successfully",
	})
}

// DeletePromoCode يحذف كود خصم (soft delete)
// DELETE /api/admin/promo-codes/:id
func (h *PromoHandler) DeletePromoCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	err = h.manager.DeletePromoCode(c.Request.Context(), tenantID, promoCodeID)
	if err != nil {
		if err == promo.ErrPromoCodeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete promo code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "promo code deleted successfully",
	})
}

// ValidatePromoCode يتحقق من صلاحية كود خصم
// POST /api/promo-codes/validate
// Requirements: 32.3, 32.4, 32.8
func (h *PromoHandler) ValidatePromoCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req promo.ApplyPromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	promoCode, err := h.manager.ValidatePromoCode(c.Request.Context(), tenantID, &req)
	if err != nil {
		// إرجاع رسالة خطأ واضحة للمستخدم
		if validationErr, ok := err.(*promo.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   validationErr.Message,
				"code":    validationErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate promo code"})
		return
	}

	// حساب الخصم المتوقع
	var discountAmount float64
	if promoCode.DiscountType == promo.DiscountTypePercentage {
		discountAmount = req.Amount * (promoCode.DiscountValue / 100.0)
		if promoCode.MaxDiscountAmount != nil && discountAmount > *promoCode.MaxDiscountAmount {
			discountAmount = *promoCode.MaxDiscountAmount
		}
	} else {
		discountAmount = promoCode.DiscountValue
		if discountAmount > req.Amount {
			discountAmount = req.Amount
		}
	}

	finalAmount := req.Amount - discountAmount

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"valid":   true,
		"data": gin.H{
			"promo_code_id":    promoCode.ID,
			"code":             promoCode.Code,
			"description":      promoCode.Description,
			"original_amount":  req.Amount,
			"discount_amount":  discountAmount,
			"final_amount":     finalAmount,
			"discount_type":    promoCode.DiscountType,
			"discount_value":   promoCode.DiscountValue,
		},
	})
}

// ApplyPromoCode يطبق كود خصم
// POST /api/promo-codes/apply
// Requirements: 32.8
func (h *PromoHandler) ApplyPromoCode(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req promo.ApplyPromoCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.manager.ApplyPromoCode(c.Request.Context(), tenantID, &req)
	if err != nil {
		// إرجاع رسالة خطأ واضحة للمستخدم
		if validationErr, ok := err.(*promo.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   validationErr.Message,
				"code":    validationErr.Code,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply promo code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetPromoPerformance يحصل على أداء كود خصم
// GET /api/admin/promo-codes/:id/performance
// Requirements: 32.11
func (h *PromoHandler) GetPromoPerformance(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	performance, err := h.manager.GetPromoPerformance(c.Request.Context(), tenantID, promoCodeID)
	if err != nil {
		if err == promo.ErrPromoCodeNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get promo code performance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    performance,
	})
}

// GetPromoUsageStats يحصل على إحصائيات استخدام كود خصم
// GET /api/admin/promo-codes/:id/stats
// Requirements: 32.11
func (h *PromoHandler) GetPromoUsageStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	stats, err := h.tracker.GetPromoCodeUsageStats(c.Request.Context(), tenantID, promoCodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetTopUsers يحصل على أكثر المستخدمين استخداماً لكود خصم
// GET /api/admin/promo-codes/:id/top-users?limit=10
// Requirements: 32.11
func (h *PromoHandler) GetTopUsers(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	topUsers, err := h.tracker.GetTopUsers(c.Request.Context(), tenantID, promoCodeID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get top users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    topUsers,
		"count":   len(topUsers),
	})
}

// GetDailyUsageStats يحصل على إحصائيات الاستخدام اليومية
// GET /api/admin/promo-codes/:id/daily-stats?days=30
// Requirements: 32.11
func (h *PromoHandler) GetDailyUsageStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	dailyStats, err := h.tracker.GetDailyUsageStats(c.Request.Context(), tenantID, promoCodeID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get daily stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dailyStats,
		"count":   len(dailyStats),
	})
}

// GeneratePerformanceReport يولد تقرير أداء شامل
// GET /api/admin/promo-codes/:id/report?days=30
// Requirements: 32.11
func (h *PromoHandler) GeneratePerformanceReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	report, err := h.reporter.GeneratePromoPerformanceReport(c.Request.Context(), tenantID, promoCodeID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

// GenerateAllPromosReport يولد تقرير أداء لجميع الأكواد
// GET /api/admin/promo-codes/reports/all
// Requirements: 32.11
func (h *PromoHandler) GenerateAllPromosReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	report, err := h.reporter.GenerateAllPromosPerformanceReport(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

// ComparePromos يقارن أداء عدة أكواد
// POST /api/admin/promo-codes/reports/compare
// Requirements: 32.11
func (h *PromoHandler) ComparePromos(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		PromoCodeIDs []int64 `json:"promo_code_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	performances, err := h.reporter.ComparePromoPerformance(c.Request.Context(), tenantID, req.PromoCodeIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compare promos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    performances,
		"count":   len(performances),
	})
}

// ExportPerformanceCSV يصدر تقرير الأداء بصيغة CSV
// GET /api/admin/promo-codes/:id/export/csv
// Requirements: 32.11
func (h *PromoHandler) ExportPerformanceCSV(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	promoCodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promo code id"})
		return
	}

	csv, err := h.reporter.ExportPromoPerformanceCSV(c.Request.Context(), tenantID, promoCodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export CSV"})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=promo_%d_performance.csv", promoCodeID))
	c.String(http.StatusOK, csv)
}

// GetPromoTrends يحصل على اتجاهات استخدام الأكواد
// GET /api/admin/promo-codes/reports/trends?days=30
// Requirements: 32.11
func (h *PromoHandler) GetPromoTrends(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	trends, err := h.reporter.GetPromoTrends(c.Request.Context(), tenantID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trends"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trends,
		"count":   len(trends),
	})
}

// RegisterPromoRoutes يسجل مسارات API لأكواد الخصم
func RegisterPromoRoutes(router *gin.Engine, db *sqlx.DB) {
	handler := NewPromoHandler(db)

	// مسارات الإدارة (تحتاج صلاحيات admin)
	admin := router.Group("/api/admin/promo-codes")
	{
		admin.POST("", handler.CreatePromoCode)
		admin.GET("", handler.ListPromoCodes)
		admin.GET("/:id", handler.GetPromoCode)
		admin.PUT("/:id", handler.UpdatePromoCode)
		admin.DELETE("/:id", handler.DeletePromoCode)
		admin.GET("/:id/performance", handler.GetPromoPerformance)
		admin.GET("/:id/stats", handler.GetPromoUsageStats)
		admin.GET("/:id/top-users", handler.GetTopUsers)
		admin.GET("/:id/daily-stats", handler.GetDailyUsageStats)
		
		// مسارات التقارير
		admin.GET("/:id/report", handler.GeneratePerformanceReport)
		admin.GET("/:id/export/csv", handler.ExportPerformanceCSV)
		admin.GET("/reports/all", handler.GenerateAllPromosReport)
		admin.POST("/reports/compare", handler.ComparePromos)
		admin.GET("/reports/trends", handler.GetPromoTrends)
	}

	// مسارات العملاء
	client := router.Group("/api/promo-codes")
	{
		client.POST("/validate", handler.ValidatePromoCode)
		client.POST("/apply", handler.ApplyPromoCode)
	}
}

// getTenantID يحصل على tenant_id من السياق
func getTenantID(c *gin.Context) (int64, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return 0, gin.Error{Err: http.ErrNotSupported}
	}

	switch v := tenantID.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, gin.Error{Err: http.ErrNotSupported}
	}
}
