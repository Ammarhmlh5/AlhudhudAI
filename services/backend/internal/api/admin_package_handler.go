package api

import (
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/gin-gonic/gin"
)

// AdminPackageHandler handles global package management
type AdminPackageHandler struct{}

// NewAdminPackageHandler creates a new admin package handler
func NewAdminPackageHandler() *AdminPackageHandler {
	return &AdminPackageHandler{}
}

// ListPackages handles GET /api/v1/admin/packages
func (h *AdminPackageHandler) ListPackages(c *gin.Context) {
	// Query all packages across all tenants for admin
	var pkgs []struct {
		ID               int64   `json:"id"`
		TenantID         string  `json:"tenant_id"`
		Name             string  `json:"name"`
		Code             string  `json:"code"`
		PriceUSD         float64 `json:"price_usd"`
		IsActive         bool    `json:"is_active"`
		IncludedMessages int64   `json:"included_messages"`
	}

	if err := db.GormDB.Table("packages").Find(&pkgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list packages"})
		return
	}

	c.JSON(http.StatusOK, pkgs)
}

// GetPackage handles GET /api/v1/admin/packages/:id
func (h *AdminPackageHandler) GetPackage(c *gin.Context) {
	id := c.Param("id")
	var pkg struct {
		ID               int64   `json:"id"`
		TenantID         string  `json:"tenant_id"`
		Name             string  `json:"name"`
		Code             string  `json:"code"`
		PriceUSD         float64 `json:"price_usd"`
		IsActive         bool    `json:"is_active"`
		IncludedMessages int64   `json:"included_messages"`
	}

	if err := db.GormDB.Table("packages").Where("id = ?", id).First(&pkg).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	c.JSON(http.StatusOK, pkg)
}

// CreatePackage handles POST /api/v1/admin/packages
func (h *AdminPackageHandler) CreatePackage(c *gin.Context) {
	// Generic package creation logic
	var pkg map[string]interface{}
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pkg["created_at"] = time.Now()
	pkg["updated_at"] = time.Now()

	if err := db.GormDB.Table("packages").Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create package"})
		return
	}

	c.JSON(http.StatusCreated, pkg)
}

// UpdatePackage handles PUT /api/v1/admin/packages/:id
func (h *AdminPackageHandler) UpdatePackage(c *gin.Context) {
	id := c.Param("id")
	var pkg map[string]interface{}
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pkg["updated_at"] = time.Now()
	if err := db.GormDB.Table("packages").Where("id = ?", id).Updates(pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update package"})
		return
	}

	c.JSON(http.StatusOK, pkg)
}

// DeletePackage handles DELETE /api/v1/admin/packages/:id
func (h *AdminPackageHandler) DeletePackage(c *gin.Context) {
	id := c.Param("id")
	if err := db.GormDB.Table("packages").Where("id = ?", id).Delete(nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete package"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Package deleted successfully"})
}
