// Package middleware provides HTTP middleware for the application
// Requirements: 11.7, 11.8, 11.9
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GatewayPermissionChecker interface for checking gateway permissions
type GatewayPermissionChecker interface {
	CheckGatewayPermission(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error)
}

// CheckGatewayPermissionRequest represents the request to check user permission
type CheckGatewayPermissionRequest struct {
	TenantID   uuid.UUID
	GatewayID  int
	UserID     uuid.UUID
	Permission string
}

// GatewayPermissionMiddleware creates a middleware that checks gateway permissions
type GatewayPermissionMiddleware struct {
	checker GatewayPermissionChecker
}

// NewGatewayPermissionMiddleware creates a new gateway permission middleware
func NewGatewayPermissionMiddleware(checker GatewayPermissionChecker) *GatewayPermissionMiddleware {
	return &GatewayPermissionMiddleware{
		checker: checker,
	}
}

// RequirePermission returns a middleware that checks if the user has the required permission
// Requirements: 11.7, 11.8, 11.9
func (gpm *GatewayPermissionMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant_id from context (set by tenant isolation middleware)
		tenantIDValue, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TENANT_NOT_FOUND",
					"message": "Tenant ID not found in context",
				},
			})
			c.Abort()
			return
		}

		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TENANT_ID",
					"message": "Invalid tenant ID format",
				},
			})
			c.Abort()
			return
		}

		// Extract user_id from context (set by authentication middleware)
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USER_NOT_FOUND",
					"message": "User ID not found in context",
				},
			})
			c.Abort()
			return
		}

		userID, ok := userIDValue.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_USER_ID",
					"message": "Invalid user ID format",
				},
			})
			c.Abort()
			return
		}

		// Extract gateway_id from URL parameter
		gatewayIDStr := c.Param("gateway_id")
		if gatewayIDStr == "" {
			// Try to get from query parameter
			gatewayIDStr = c.Query("gateway_id")
		}

		if gatewayIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "GATEWAY_ID_REQUIRED",
					"message": "Gateway ID is required",
				},
			})
			c.Abort()
			return
		}

		gatewayID, err := strconv.Atoi(gatewayIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_GATEWAY_ID",
					"message": "Invalid gateway ID format",
				},
			})
			c.Abort()
			return
		}

		// Check permission
		req := &CheckGatewayPermissionRequest{
			TenantID:   tenantID,
			GatewayID:  gatewayID,
			UserID:     userID,
			Permission: permission,
		}

		hasPermission, err := gpm.checker.CheckGatewayPermission(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_CHECK_FAILED",
					"message": "Failed to check permission",
					"details": err.Error(),
				},
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PERMISSION_DENIED",
					"message": "You do not have permission to perform this action",
					"details": gin.H{
						"required_permission": permission,
						"gateway_id":          gatewayID,
					},
				},
			})
			c.Abort()
			return
		}

		// Permission granted, continue
		c.Set("gateway_id", gatewayID)
		c.Next()
	}
}

// RequireRole returns a middleware that checks if the user has the required role
// Requirements: 11.7, 11.8, 11.9
func (gpm *GatewayPermissionMiddleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant_id from context
		tenantIDValue, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TENANT_NOT_FOUND",
					"message": "Tenant ID not found in context",
				},
			})
			c.Abort()
			return
		}

		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TENANT_ID",
					"message": "Invalid tenant ID format",
				},
			})
			c.Abort()
			return
		}

		// Extract user_id from context
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USER_NOT_FOUND",
					"message": "User ID not found in context",
				},
			})
			c.Abort()
			return
		}

		userID, ok := userIDValue.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_USER_ID",
					"message": "Invalid user ID format",
				},
			})
			c.Abort()
			return
		}

		// Extract gateway_id from URL parameter
		gatewayIDStr := c.Param("gateway_id")
		if gatewayIDStr == "" {
			gatewayIDStr = c.Query("gateway_id")
		}

		if gatewayIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "GATEWAY_ID_REQUIRED",
					"message": "Gateway ID is required",
				},
			})
			c.Abort()
			return
		}

		gatewayID, err := strconv.Atoi(gatewayIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_GATEWAY_ID",
					"message": "Invalid gateway ID format",
				},
			})
			c.Abort()
			return
		}

		// Get user's role from database
		role, err := gpm.getUserRole(c.Request.Context(), tenantID, gatewayID, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "USER_NOT_AUTHORIZED",
						"message": "User is not authorized for this gateway",
					},
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "ROLE_CHECK_FAILED",
						"message": "Failed to check user role",
						"details": err.Error(),
					},
				})
			}
			c.Abort()
			return
		}

		// Check if user has one of the allowed roles
		hasRole := false
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ROLE_DENIED",
					"message": "Your role does not have access to this resource",
					"details": gin.H{
						"your_role":     role,
						"required_roles": allowedRoles,
					},
				},
			})
			c.Abort()
			return
		}

		// Role check passed
		c.Set("gateway_id", gatewayID)
		c.Set("user_role", role)
		c.Next()
	}
}

// getUserRole retrieves the user's role for a specific gateway
func (gpm *GatewayPermissionMiddleware) getUserRole(ctx context.Context, tenantID uuid.UUID, gatewayID int, userID uuid.UUID) (string, error) {
	// This is a simplified implementation
	// In a real implementation, this would query the database through the checker
	// For now, we'll use a type assertion to access the underlying database
	
	type roleGetter interface {
		GetUserRole(ctx context.Context, tenantID uuid.UUID, gatewayID int, userID uuid.UUID) (string, error)
	}

	if rg, ok := gpm.checker.(roleGetter); ok {
		return rg.GetUserRole(ctx, tenantID, gatewayID, userID)
	}

	return "", errors.New("role getter not implemented")
}

// RequireAdminRole is a convenience middleware that requires admin role
func (gpm *GatewayPermissionMiddleware) RequireAdminRole() gin.HandlerFunc {
	return gpm.RequireRole("admin")
}

// RequireOperatorOrAdmin is a convenience middleware that requires operator or admin role
func (gpm *GatewayPermissionMiddleware) RequireOperatorOrAdmin() gin.HandlerFunc {
	return gpm.RequireRole("admin", "operator")
}

// RequireAnyRole is a convenience middleware that requires any valid role (admin, operator, or viewer)
func (gpm *GatewayPermissionMiddleware) RequireAnyRole() gin.HandlerFunc {
	return gpm.RequireRole("admin", "operator", "viewer")
}

