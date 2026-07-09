package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/al-hudhud/backend/internal/admin/rating"
	"github.com/gin-gonic/gin"
)

// AdminRatingHandler handles admin rating API endpoints
type AdminRatingHandler struct {
	ratingEngine *rating.Calculator
}

// NewAdminRatingHandler creates a new admin rating handler
func NewAdminRatingHandler(ratingEngine *rating.Calculator) *AdminRatingHandler {
	return &AdminRatingHandler{
		ratingEngine: ratingEngine,
	}
}

// GetCompanyRating handles GET /api/v1/admin/companies/:tenant_id/rating
func (h *AdminRatingHandler) GetCompanyRating(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "invalid_tenant_id", "Invalid tenant ID format", nil)
		return
	}

	// In a real implementation, we would fetch metrics from a repository first.
	// Since the Calculator only provides calculations, we'll return a placeholder or 
	// we'd need to integrate with a metrics repository here.
	
	metrics := rating.PerformanceMetrics{
		TenantID: tenantID,
		// Mock data for now
		TotalMessages:      1000,
		SuccessfulMessages: 950,
		FailedMessages:     50,
	}

	ratingValue := h.ratingEngine.CalculateRating(&metrics)
	category := h.ratingEngine.ClassifyRating(ratingValue)

	result := map[string]interface{}{
		"tenant_id":       tenantID,
		"rating":          ratingValue,
		"category":        category,
		"metrics":         metrics,
	}

	RespondWithSuccess(c, http.StatusOK, result, "")
}
