package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/pkg/jwt"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuthMiddleware validates JWT tokens and adds user info to Gin context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// If already authenticated via API Key, skip JWT check
		if id, _ := c.Get("tenant_id"); id != nil && id != "" {
			c.Next()
			return
		}

		token := extractToken(c.Request)
		if token == "" {
			logger.GetLogger().Warn("Authentication failed: No token provided", 
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
			c.Abort()
			return
		}

		claims, err := jwt.ValidateToken(token)
		if err != nil {
			logger.GetLogger().Warn("Authentication failed: Invalid token", 
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
			c.Abort()
			return
		}

		// Add user info to Gin context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("user_type", claims.UserType)
		c.Set("tenant_id", claims.TenantID)
		c.Set("staff_id", claims.StaffID)
		c.Set("permissions", claims.Permissions)
		c.Set("request_source", "WEB_UI")

		c.Next()
	}
}

// RequireUserType middleware ensures the user has one of the allowed user types
func RequireUserType(allowedTypes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")
		if userType == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		allowed := false
		for _, t := range allowedTypes {
			if userType == t {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient privileges"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission middleware ensures the user has a specific permission
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")

		// Super admin has all permissions
		if userType == "SUPER_ADMIN" {
			c.Next()
			return
		}

		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: no permissions found"})
			c.Abort()
			return
		}

		permsList, ok := permissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: invalid permissions format"})
			c.Abort()
			return
		}

		hasPermission := false
		for _, p := range permsList {
			if p == permission || p == "*" {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TenantIsolationMiddleware ensures tenants can only access their own data
func TenantIsolationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")

		// Only apply to tenant and staff users
		if userType != "TENANT" && userType != "STAFF" {
			c.Next()
			return
		}

		tenantID := c.GetString("tenant_id")
		if tenantID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: tenant ID not found"})
			c.Abort()
			return
		}

		// Add tenant_id to context for use in handlers
		c.Set("isolated_tenant_id", tenantID)

		// Parse Tenant UUID
		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: invalid tenant ID"})
			c.Abort()
			return
		}

		// Start a transaction for this request with the correct search_path
		// This ensures strict schema isolation for all DB operations in this request
		tx := db.GormDB.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			c.Abort()
			return
		}

		schemaName := db.GetSchemaName(tenantUUID)
		// Set search path first thing - Use SET LOCAL for transaction scope
		if err := tx.Exec(fmt.Sprintf("SET LOCAL search_path TO %s, public", schemaName)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database schema error"})
			c.Abort()
			return
		}

		// Set tenant context for Row Level Security (RLS)
		// Using true for the last parameter (is_local) makes it transaction-scoped
		if err := tx.Exec("SELECT set_config('app.current_tenant', $1, true)", tenantID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database context error"})
			c.Abort()
			return
		}

		// Set super admin context if applicable - transaction-scoped
		isSuperAdmin := userType == "SUPER_ADMIN"
		if err := tx.Exec("SELECT set_config('app.is_super_admin', $1, true)", fmt.Sprintf("%t", isSuperAdmin)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database security error"})
			c.Abort()
			return
		}

		// Store the transactional DB in context
		c.Set("db", tx)

		// Process request
		c.Next()

		// Commit or Rollback based on status
		if c.Writer.Status() >= 400 || len(c.Errors) > 0 {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}
}

// extractToken extracts the JWT token from the Authorization header or cookies
func extractToken(r *http.Request) string {
	// Try header first
	bearerToken := r.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}

	// Try cookie
	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}

	return ""
}
