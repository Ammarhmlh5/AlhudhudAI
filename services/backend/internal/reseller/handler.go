package reseller

import (
	"net/http"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetTenants(c *gin.Context) {
	resellerID, _ := c.Get("tenant_id")
	id, ok := resellerID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid reseller identity"})
		return
	}

	tenants, err := h.service.ListTenants(c.Request.Context(), id)
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
	resellerID, _ := c.Get("tenant_id")
	rid, ok := resellerID.(uuid.UUID)
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

func (h *Handler) UpdateAPIKey(c *gin.Context) {
	resellerID, _ := c.Get("tenant_id")
	rid, ok := resellerID.(uuid.UUID)
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
	resellerID, _ := c.Get("tenant_id")
	rid, ok := resellerID.(uuid.UUID)
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
