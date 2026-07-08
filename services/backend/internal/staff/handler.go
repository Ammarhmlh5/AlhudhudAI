package staff

import (
	"net/http"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for staff management
type Handler struct {
	service *Service
}

// NewHandler creates a new staff handler
func NewHandler(svc *Service) *Handler {
	return &Handler{
		service: svc,
	}
}

// CreateStaff handles staff creation
func (h *Handler) CreateStaff(c *gin.Context) {
	var req struct {
		Email      string `json:"email" binding:"required,email"`
		FullName   string `json:"full_name" binding:"required"`
		RoleID     string `json:"role_id" binding:"required"`
		Department string `json:"department"`
		Password   string `json:"password" binding:"required,min=8"`
		TenantID   string `json:"tenant_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}

	if tenantID == uuid.Nil && userType != "SUPER_ADMIN" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID"})
		return
	}

	staff, err := h.service.CreateStaff(tenantID, req.Email, req.FullName, req.RoleID, req.Department, req.Password)
	if err != nil {
		logger.GetLogger().Error("Failed to create staff", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, staff)
}

// ListStaff handles listing all staff with filters
func (h *Handler) ListStaff(c *gin.Context) {
	search := c.Query("search")
	roleID := c.Query("role_id")
	department := c.Query("department")

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && c.Query("tenant_id") != "" {
		tenantID, _ = uuid.Parse(c.Query("tenant_id"))
	} else if userType != "SUPER_ADMIN" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context missing"})
		return
	}

	staff, err := h.service.ListStaff(tenantID, search, roleID, department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, staff)
}

// GetStaff handles getting a single staff member
func (h *Handler) GetStaff(c *gin.Context) {
	staffID := c.Query("id")
	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && c.Query("tenant_id") != "" {
		tenantID, _ = uuid.Parse(c.Query("tenant_id"))
	}

	staff, err := h.service.GetStaff(c.Request.Context(), tenantID, staffID)
	if err != nil {
		if err == ErrStaffNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, staff)
}

// UpdateStaff handles staff updates
func (h *Handler) UpdateStaff(c *gin.Context) {
	var req struct {
		ID         string `json:"id" binding:"required"`
		FullName   string `json:"full_name" binding:"required"`
		RoleID     string `json:"role_id" binding:"required"`
		Department string `json:"department"`
		IsActive   bool   `json:"is_active"`
		TenantID   string `json:"tenant_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}

	err := h.service.UpdateStaff(c.Request.Context(), tenantID, req.ID, req.FullName, req.RoleID, req.Department, req.IsActive)
	if err != nil {
		if err == ErrStaffNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff updated successfully"})
}

// DeleteStaff handles staff deletion
func (h *Handler) DeleteStaff(c *gin.Context) {
	staffID := c.Query("id")
	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && c.Query("tenant_id") != "" {
		tenantID, _ = uuid.Parse(c.Query("tenant_id"))
	}

	err := h.service.DeleteStaff(c.Request.Context(), tenantID, staffID)
	if err != nil {
		if err == ErrStaffNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff deleted successfully"})
}

// ListRoles handles listing all roles for the tenant
func (h *Handler) ListRoles(c *gin.Context) {
	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && c.Query("tenant_id") != "" {
		tenantID, _ = uuid.Parse(c.Query("tenant_id"))
	} else if userType != "SUPER_ADMIN" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context missing"})
		return
	}

	roles, err := h.service.ListRoles(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// CreateRole handles role creation
func (h *Handler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
		TenantID    string   `json:"tenant_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}

	role, err := h.service.CreateRole(tenantID, req.Name, req.Description, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// UpdateRole handles role updates
func (h *Handler) UpdateRole(c *gin.Context) {
	var req struct {
		ID          string   `json:"id" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
		TenantID    string   `json:"tenant_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}

	err := h.service.UpdateRole(tenantID, req.ID, req.Name, req.Description, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// DeleteRole handles role deletion
func (h *Handler) DeleteRole(c *gin.Context) {
	roleID := c.Query("id")
	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && c.Query("tenant_id") != "" {
		tenantID, _ = uuid.Parse(c.Query("tenant_id"))
	}

	err := h.service.DeleteRole(tenantID, roleID)
	if err != nil {
		if err == ErrRoleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

// ListPermissions handles listing all permissions
func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// BulkDeleteStaff handles bulk staff deletion
func (h *Handler) BulkDeleteStaff(c *gin.Context) {
	var req struct {
		IDs      []uuid.UUID `json:"ids" binding:"required"`
		TenantID string      `json:"tenant_id"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}

	if err := h.service.BulkDeleteStaff(c.Request.Context(), tenantID, req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff members deleted successfully"})
}

// BulkUpdateStaff handles bulk staff updates
func (h *Handler) BulkUpdateStaff(c *gin.Context) {
	var req struct {
		IDs      []uuid.UUID `json:"ids" binding:"required"`
		RoleID   uuid.UUID   `json:"role_id"`
		IsActive bool        `json:"is_active"`
		TenantID string      `json:"tenant_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantIDStr := c.GetString("tenant_id")
	userType := c.GetString("user_type")
	
	var tenantID uuid.UUID
	if tenantIDStr != "" {
		tenantID, _ = uuid.Parse(tenantIDStr)
	} else if userType == "SUPER_ADMIN" && req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}

	if err := h.service.BulkUpdateStaff(c.Request.Context(), tenantID, req.IDs, req.RoleID, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff members updated successfully"})
}
