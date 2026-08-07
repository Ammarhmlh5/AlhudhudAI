package agent

import (
	"net/http"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================================
// Handler
// ============================================================================

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ============================================================================
// Authentication Handlers
// ============================================================================

// AgentLogin - تسجيل دخول الوكيل
// POST /api/v1/agents/auth/login
func (h *Handler) AgentLogin(c *gin.Context) {
	var req models.AgentLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response, err := h.service.AgentLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if err == ErrUnauthorized {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// EmployeeLogin - تسجيل دخول الموظف
// POST /api/v1/agents/auth/employees/login
func (h *Handler) EmployeeLogin(c *gin.Context) {
	var req models.AgentEmployeeLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	response, err := h.service.EmployeeLogin(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if err == ErrUnauthorized {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// ChangePassword - تغيير كلمة المرور
// POST /api/v1/agents/auth/change-password
func (h *Handler) ChangePassword(c *gin.Context) {
	var req models.AgentChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Get user info from context (set by auth middleware)
	userIDStr := c.GetString("user_id")
	userType := c.GetString("user_type")

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid user ID",
		})
		return
	}

	err = h.service.ChangePassword(c.Request.Context(), userID, userType, req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "تم تغيير كلمة المرور بنجاح",
	})
}

// ============================================================================
// Employee Management Handlers (Agent Only)
// ============================================================================

// CreateEmployee - إنشاء موظف جديد
// POST /api/v1/agents/employees
func (h *Handler) CreateEmployee(c *gin.Context) {
	// Verify user is agent (not employee)
	if c.GetString("user_type") != "agent" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "غير مسموح - هذا الإجراء للوكيل فقط",
		})
		return
	}

	var req models.CreateAgentEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	employee, err := h.service.CreateEmployee(c.Request.Context(), agentID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Don't return password hash
	employee.PasswordHash = ""

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    employee,
	})
}

// ListEmployees - قائمة الموظفين
// GET /api/v1/agents/employees
func (h *Handler) ListEmployees(c *gin.Context) {
	// Verify user is agent (not employee)
	if c.GetString("user_type") != "agent" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "غير مسموح - هذا الإجراء للوكيل فقط",
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	employees, err := h.service.ListEmployees(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Remove password hashes
	for i := range employees {
		employees[i].PasswordHash = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employees,
	})
}

// GetEmployee - تفاصيل موظف
// GET /api/v1/agents/employees/:id
func (h *Handler) GetEmployee(c *gin.Context) {
	// Verify user is agent
	if c.GetString("user_type") != "agent" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "غير مسموح - هذا الإجراء للوكيل فقط",
		})
		return
	}

	employeeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "معرف الموظف غير صالح",
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	employee, err := h.service.GetEmployee(c.Request.Context(), agentID, employeeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	employee.PasswordHash = ""

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    employee,
	})
}

// DeleteEmployee - حذف موظف
// DELETE /api/v1/agents/employees/:id
func (h *Handler) DeleteEmployee(c *gin.Context) {
	// Verify user is agent
	if c.GetString("user_type") != "agent" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "غير مسموح - هذا الإجراء للوكيل فقط",
		})
		return
	}

	employeeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "معرف الموظف غير صالح",
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	err = h.service.DeleteEmployee(c.Request.Context(), agentID, employeeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "تم حذف الموظف بنجاح",
	})
}

// ============================================================================
// Tenant Management Handlers
// ============================================================================

// CreateTenant - إنشاء مستأجر
// POST /api/v1/agents/tenants
func (h *Handler) CreateTenant(c *gin.Context) {
	var req models.CreateAgentTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))
	userID, _ := uuid.Parse(c.GetString("user_id"))

	tenant, err := h.service.CreateTenant(c.Request.Context(), agentID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// ListTenants - قائمة المستأجرين
// GET /api/v1/agents/tenants
func (h *Handler) ListTenants(c *gin.Context) {
	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	tenants, err := h.service.ListTenants(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenants,
		"meta": gin.H{
			"total": len(tenants),
		},
	})
}

// GetTenant - تفاصيل مستأجر
// GET /api/v1/agents/tenants/:id
func (h *Handler) GetTenant(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "معرف المستأجر غير صالح",
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	tenant, err := h.service.GetTenant(c.Request.Context(), agentID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tenant,
	})
}

// ============================================================================
// Payment Management Handlers
// ============================================================================

// CreatePayment - سداد فاتورة
// POST /api/v1/agents/payments
func (h *Handler) CreatePayment(c *gin.Context) {
	var req models.AgentPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	agentID, _ := uuid.Parse(c.GetString("agent_id"))
	userID, _ := uuid.Parse(c.GetString("user_id"))

	payment, err := h.service.CreatePayment(c.Request.Context(), agentID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    payment,
		"message": "تم السداد بنجاح",
	})
}

// ListPayments - قائمة المدفوعات
// GET /api/v1/agents/payments
func (h *Handler) ListPayments(c *gin.Context) {
	agentID, _ := uuid.Parse(c.GetString("agent_id"))

	payments, err := h.service.ListPayments(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    payments,
		"meta": gin.H{
			"total": len(payments),
		},
	})
}

// ============================================================================
// Blocked Actions Handlers
// ============================================================================

// BlockedDeleteTenant - حذف مستأجر (محظور)
// DELETE /api/v1/agents/tenants/:id
func (h *Handler) BlockedDeleteTenant(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"error":   "غير مسموح - هذا الإجراء يتم من المنصة الرئيسية",
	})
}

// BlockedSuspendTenant - إيقاف مستأجر (محظور)
// PUT /api/v1/agents/tenants/:id/suspend
func (h *Handler) BlockedSuspendTenant(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"error":   "غير مسموح - هذا الإجراء يتم من المنصة الرئيسية",
	})
}

// BlockedActivateTenant - تفعيل مستأجر (محظور)
// PUT /api/v1/agents/tenants/:id/activate
func (h *Handler) BlockedActivateTenant(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"error":   "غير مسموح - هذا الإجراء يتم من المنصة الرئيسية",
	})
}

// BlockedUpdateTenant - تحديث مستأجر (محظور)
// PUT /api/v1/agents/tenants/:id
func (h *Handler) BlockedUpdateTenant(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"error":   "غير مسموح - هذا الإجراء يتم من المنصة الرئيسية",
	})
}

// ============================================================================
// Health Check
// ============================================================================

// HealthCheck - فحص الحالة
// GET /api/v1/agents/health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "healthy",
		"service": "agents",
	})
}