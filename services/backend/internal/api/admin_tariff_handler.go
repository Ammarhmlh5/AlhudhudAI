package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/tariff"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminTariffHandler handles global tariff management
type AdminTariffHandler struct {
	repo *tariff.Repository
}

// NewAdminTariffHandler creates a new admin tariff handler
func NewAdminTariffHandler(db *gorm.DB) *AdminTariffHandler {
	return &AdminTariffHandler{
		repo: tariff.NewRepository(db),
	}
}

// ListTariffs handles GET /api/v1/admin/tariffs
func (h *AdminTariffHandler) ListTariffs(c *gin.Context) {
	var tariffs []models.Tariff
	
	tenantID := c.Query("tenant_id")
	serviceID := c.Query("service_id")
	isActive := c.Query("is_active")

	query := db.GormDB.Model(&models.Tariff{}).Preload("Tenant").Preload("Service")

	if tenantID != "" {
		if uid, err := uuid.Parse(tenantID); err == nil {
			query = query.Where("tenant_id = ?", uid)
		}
	}

	if serviceID != "" {
		if sid, err := strconv.Atoi(serviceID); err == nil {
			query = query.Where("service_id = ?", sid)
		}
	}

	if isActive != "" {
		active, _ := strconv.ParseBool(isActive)
		query = query.Where("is_active = ?", active)
	}

	if err := query.Find(&tariffs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tariffs"})
		return
	}

	c.JSON(http.StatusOK, tariffs)
}

// CreateTariff handles POST /api/v1/admin/tariffs
func (h *AdminTariffHandler) CreateTariff(c *gin.Context) {
	var t models.Tariff
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if t.EffectiveFrom.IsZero() {
		t.EffectiveFrom = time.Now()
	}

	if err := db.GormDB.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tariff"})
		return
	}

	c.JSON(http.StatusCreated, t)
}

// UpdateTariff handles PUT /api/v1/admin/tariffs/:id
func (h *AdminTariffHandler) UpdateTariff(c *gin.Context) {
	id := c.Param("id")
	var t models.Tariff
	if err := db.GormDB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tariff not found"})
		return
	}

	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t.UpdatedAt = time.Now()
	if err := db.GormDB.Save(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tariff"})
		return
	}

	c.JSON(http.StatusOK, t)
}

// GetTariff handles GET /api/v1/admin/tariffs/:id
func (h *AdminTariffHandler) GetTariff(c *gin.Context) {
	id := c.Param("id")
	var t models.Tariff
	if err := db.GormDB.Preload("Tenant").Preload("Service").First(&t, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tariff not found"})
		return
	}

	c.JSON(http.StatusOK, t)
}

// DeleteTariff handles DELETE /api/v1/admin/tariffs/:id
func (h *AdminTariffHandler) DeleteTariff(c *gin.Context) {
	id := c.Param("id")
	if err := db.GormDB.Delete(&models.Tariff{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tariff"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tariff deleted successfully"})
}
