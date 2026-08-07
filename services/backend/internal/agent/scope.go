package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Scope Guard - عزل العمليات والصلاحيات
// ============================================================================

// ScopeGuard - يمنع الإجراءات المحظورة على الوكلاء والموظفين
var blockedActions = map[string]bool{
	// Tenant management - blocked
	"DELETE:/api/v1/agents/tenants/:id":       true,
	"PUT:/api/v1/agents/tenants/:id/suspend":  true,
	"PUT:/api/v1/agents/tenants/:id/activate": true,
	"PUT:/api/v1/agents/tenants/:id":          true,
	"PATCH:/api/v1/agents/tenants/:id":        true,

	// Employee management - only agent can do this
	"POST:/api/v1/agents/employees":       true, // Handled by AgentOnlyMiddleware
	"PUT:/api/v1/agents/employees/:id":    true, // Handled by AgentOnlyMiddleware
	"DELETE:/api/v1/agents/employees/:id": true, // Handled by AgentOnlyMiddleware
}

// ScopeGuardMiddleware - middleware لمنع الإجراءات المحظورة
func ScopeGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build the key
		key := c.Request.Method + ":" + c.FullPath()

		// Check if this action is blocked
		if isBlocked(key) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "غير مسموح - هذا الإجراء يتم من المنصة الرئيسية",
				"code":    "FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isBlocked - يتحقق مما إذا كان الإجراء محظوراً
func isBlocked(key string) bool {
	// Check exact match
	if blockedActions[key] {
		return true
	}

	// Check pattern matching (wildcard)
	// For example, /api/v1/agents/tenants/:id matches /api/v1/agents/tenants/123
	for pattern := range blockedActions {
		if matchPattern(key, pattern) {
			return true
		}
	}

	return false
}

// matchPattern - يتحقق من تطابق المسار مع النمط
func matchPattern(path, pattern string) bool {
	// Convert pattern to regex
	// :id -> matches any string
	parts := splitPath(pattern)
	pathParts := splitPath(path)

	if len(parts) != len(pathParts) {
		return false
	}

	for i, part := range parts {
		if part == pathParts[i] {
			continue // Exact match
		}
		if isWildcard(part) {
			continue // Wildcard matches anything
		}
		return false
	}

	return true
}

// splitPath - يقسم المسار إلى أجزاء
func splitPath(path string) []string {
	var parts []string
	current := ""

	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// isWildcard - يتحقق مما إذا كان الجزء يحتوي على متغير
func isWildcard(part string) bool {
	return len(part) > 0 && part[0] == ':'
}

// ============================================================================
// Tenant Isolation Middleware - عزل المستأجرين
// ============================================================================

// TenantIsolationMiddleware - يضمن أن المستخدم يرى فقط مستأجرينه
// يضاف agent_id إلى الاستعلامات تلقائياً
func TenantIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get agent_id from JWT (set by auth middleware)
		agentID := c.GetString("agent_id")

		// If no agent_id, reject request
		if agentID == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "غير مصرح لك بالوصول",
			})
			c.Abort()
			return
		}

		// Set agent_id for use in handlers
		c.Set("tenant_filter_agent_id", agentID)

		c.Next()
	}
}

// ============================================================================
// Rate Limiting Middleware - تحديد الطلبات
// ============================================================================

type rateLimitConfig struct {
	Login   rateLimitRule `json:"login"`
	API     rateLimitRule `json:"api"`
	Payment rateLimitRule `json:"payment"`
}

type rateLimitRule struct {
	Requests      int `json:"requests"`
	WindowSeconds int `json:"window_seconds"`
	BlockDuration int `json:"block_duration_seconds"`
}

var defaultRateLimitConfig = rateLimitConfig{
	Login: rateLimitRule{
		Requests:      5,
		WindowSeconds: 60,
		BlockDuration: 900, // 15 minutes
	},
	API: rateLimitRule{
		Requests:      100,
		WindowSeconds: 60,
		BlockDuration: 60,
	},
	Payment: rateLimitRule{
		Requests:      10,
		WindowSeconds: 60,
		BlockDuration: 300, // 5 minutes
	},
}

// GetRateLimitConfig - إعدادات تحديد السرعة
func GetRateLimitConfig() rateLimitConfig {
	return defaultRateLimitConfig
}

// ============================================================================
// Permission Checker - فحص الصلاحيات
// ============================================================================

// Permission constants
const (
	PermCreateTenant   = "tenant:create"
	PermViewTenant     = "tenant:view"
	PermDeleteTenant   = "tenant:delete"   // Always blocked
	PermSuspendTenant  = "tenant:suspend"  // Always blocked
	PermActivateTenant = "tenant:activate" // Always blocked

	PermCreatePayment = "payment:create"
	PermViewPayment   = "payment:view"

	PermManageEmployees = "employee:manage" // Agent only
	PermViewEmployees   = "employee:view"   // Agent only
)

// CheckPermission - يتحقق من صلاحية المستخدم لإجراء معين
func CheckPermission(userType, role, permission string) bool {
	// Agent has all permissions
	if userType == "agent" {
		return true
	}

	// Employee permissions
	if userType == "employee" {
		switch permission {
		case PermCreateTenant:
			return true
		case PermViewTenant:
			return true
		case PermCreatePayment:
			return true
		case PermViewPayment:
			return true
		case PermManageEmployees:
			return false // Only agent
		case PermViewEmployees:
			return false // Only agent
		case PermDeleteTenant:
			return false // Always blocked
		case PermSuspendTenant:
			return false // Always blocked
		case PermActivateTenant:
			return false // Always blocked
		}
	}

	return false
}

// PermissionMiddleware - middleware لفحص الصلاحيات
func PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")
		role := c.GetString("role")

		if !CheckPermission(userType, role, requiredPermission) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "ليس لديك صلاحية لهذا الإجراء",
				"code":    "PERMISSION_DENIED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================================================
// Admin Only Middleware - للعمليات الإدارية
// ============================================================================

// AdminOnlyMiddleware - يضمن أن المستخدم هو مدير (وكيل أو موظف مدير)
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")
		role := c.GetString("role")

		// Agent is always admin
		if userType == "agent" {
			c.Next()
			return
		}

		// Employee with admin role
		if userType == "employee" && role == "admin" {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "غير مسموح - يتطلب صلاحيات مدير",
		})
		c.Abort()
		return
	}
}

// ============================================================================
// Audit Log Middleware - تسجيل الأنشطة
// ============================================================================

// AuditLogMiddleware - يسجل جميع الأنشطة
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for public endpoints
		publicPaths := []string{
			"/api/v1/agents/auth/login",
			"/api/v1/agents/auth/employees/login",
			"/api/v1/agents/health",
		}

		isPublic := false
		for _, p := range publicPaths {
			if c.FullPath() == p {
				isPublic = true
				break
			}
		}

		if !isPublic {
			// Log would be created in handler after action completes
			// This is just a placeholder for audit logging
		}

		c.Next()
	}
}

// ============================================================================
// Error Response Helpers
// ============================================================================

// ForbiddenResponse - response for forbidden actions
func ForbiddenResponse(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "FORBIDDEN",
			"message": message,
		},
	})
}

// UnauthorizedResponse - response for unauthorized requests
func UnauthorizedResponse(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
	})
}

// NotFoundResponse - response for not found
func NotFoundResponse(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "NOT_FOUND",
			"message": message,
		},
	})
}

// BadRequestResponse - response for bad requests
func BadRequestResponse(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "BAD_REQUEST",
			"message": message,
		},
	})
}
