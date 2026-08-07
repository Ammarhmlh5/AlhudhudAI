package agent

import (
	"github.com/gin-gonic/gin"
)

// ============================================================================
// Router - ربط المسارات بالـ Handlers
// ============================================================================

// SetupRoutes - إعداد جميع مسارات نظام الوكلاء
func SetupRoutes(router *gin.Engine, handler *Handler) {
	setupRoutes(router.Group("/api/v1/agents"), handler)
}

// setupRoutes - يسجل مسارات نظام الوكلاء على مجموعة مسار جذرية معطاة
func setupRoutes(base *gin.RouterGroup, handler *Handler) {
	// ========================================
	// المسارات العامة (بدون مصادقة)
	// ========================================
	public := base.Group("/auth")
	{
		public.POST("/login", handler.AgentLogin)
		public.POST("/employees/login", handler.EmployeeLogin)
	}

	// Health check
	base.GET("/health", handler.HealthCheck)

	// ========================================
	// المسارات المحمية (تتطلب مصادقة)
	// ========================================
	agent := base.Group("")
	agent.Use(AuthMiddleware())           // المصادقة
	agent.Use(ScopeGuardMiddleware())     // منع الإجراءات المحظورة

	{
		// ========================================
		// المصادقة والإعدادات
		// ========================================
		auth := agent.Group("/auth")
		{
			auth.POST("/change-password", handler.ChangePassword)
		}

		// ========================================
		// إدارة الموظفين (للوكيل فقط)
		// ========================================
		employees := agent.Group("/employees")
		employees.Use(AgentOnlyMiddleware()) // الوكيل فقط
		{
			employees.POST("", handler.CreateEmployee)
			employees.GET("", handler.ListEmployees)
			employees.GET("/:id", handler.GetEmployee)
			employees.DELETE("/:id", handler.DeleteEmployee)
		}

		// ========================================
		// إدارة المستأجرين
		// ========================================
		tenants := agent.Group("/tenants")
		{
			tenants.POST("", handler.CreateTenant)
			tenants.GET("", handler.ListTenants)
			tenants.GET("/:id", handler.GetTenant)

			// المحظورة - سيتم تحويلها إلى handlers ترد بـ 403
			tenants.DELETE("/:id", handler.BlockedDeleteTenant)
			tenants.PUT("/:id/suspend", handler.BlockedSuspendTenant)
			tenants.PUT("/:id/activate", handler.BlockedActivateTenant)
			tenants.PUT("/:id", handler.BlockedUpdateTenant)
			tenants.PATCH("/:id", handler.BlockedUpdateTenant)
		}

		// ========================================
		// المدفوعات
		// ========================================
		// الفواتير (قراءة فقط) - تحتاج تكامل مع نظام الفواتير
		agent.Group("/invoices")

		// المدفوعات
		payments := agent.Group("/payments")
		{
			payments.POST("", handler.CreatePayment)
			payments.GET("", handler.ListPayments)
		}
	}
}

// SetupRoutesWithConfig - إعداد المسارات مع إعدادات مخصصة
func SetupRoutesWithConfig(router *gin.Engine, handler *Handler, config *RouterConfig) {
	if config == nil {
		config = &DefaultRouterConfig
	}

	// Same as SetupRoutes but with custom config
	SetupRoutes(router, handler)
}

// ============================================================================
// Router Configuration
// ============================================================================

type RouterConfig struct {
	Prefix            string
	EnableRateLimit   bool
	EnableAuditLog    bool
	EnableCORS        bool
}

var DefaultRouterConfig = RouterConfig{
	Prefix:            "/api/v1/agents",
	EnableRateLimit:   true,
	EnableAuditLog:    true,
	EnableCORS:        false,
}

// ============================================================================
// Route Groups (for external use)
// ============================================================================

// GetAuthGroup - مجموعة المصادقة
func GetAuthGroup(router *gin.Engine, auth gin.HandlerFunc) *gin.RouterGroup {
	return router.Group("/api/v1/agents/auth")
}

// GetAgentGroup - مجموعة الوكيل (مع المصادقة)
func GetAgentGroup(router *gin.Engine, auth gin.HandlerFunc, guards ...gin.HandlerFunc) *gin.RouterGroup {
	group := router.Group("/api/v1/agents")
	group.Use(auth)
	for _, guard := range guards {
		group.Use(guard)
	}
	return group
}

// GetEmployeesGroup - مجموعة الموظفين
func GetEmployeesGroup(agent *gin.RouterGroup, agentOnly gin.HandlerFunc) *gin.RouterGroup {
	return agent.Group("/employees", agentOnly)
}

// GetTenantsGroup - مجموعة المستأجرين
func GetTenantsGroup(agent *gin.RouterGroup) *gin.RouterGroup {
	return agent.Group("/tenants")
}

// GetPaymentsGroup - مجموعة المدفوعات
func GetPaymentsGroup(agent *gin.RouterGroup) *gin.RouterGroup {
	return agent.Group("/payments")
}