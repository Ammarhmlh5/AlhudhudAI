package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// MockGatewayPermissionChecker is a mock implementation of GatewayPermissionChecker
type MockGatewayPermissionChecker struct {
	CheckPermissionFunc func(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error)
	GetUserRoleFunc     func(ctx context.Context, tenantID uuid.UUID, gatewayID int, userID uuid.UUID) (string, error)
}

func (m *MockGatewayPermissionChecker) CheckGatewayPermission(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error) {
	if m.CheckPermissionFunc != nil {
		return m.CheckPermissionFunc(ctx, req)
	}
	return false, nil
}

func (m *MockGatewayPermissionChecker) GetUserRole(ctx context.Context, tenantID uuid.UUID, gatewayID int, userID uuid.UUID) (string, error) {
	if m.GetUserRoleFunc != nil {
		return m.GetUserRoleFunc(ctx, tenantID, gatewayID, userID)
	}
	return "", errors.New("not implemented")
}

func setupGatewayPermissionTest() (*gin.Engine, *MockGatewayPermissionChecker) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	mockChecker := &MockGatewayPermissionChecker{}
	
	return router, mockChecker
}

func TestRequirePermission(t *testing.T) {
	router, mockChecker := setupGatewayPermissionTest()
	middleware := NewGatewayPermissionMiddleware(mockChecker)

	tenantID := uuid.New()
	userID := uuid.New()
	gatewayID := 1

	t.Run("Permission granted", func(t *testing.T) {
		mockChecker.CheckPermissionFunc = func(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error) {
			assert.Equal(t, tenantID, req.TenantID)
			assert.Equal(t, userID, req.UserID)
			assert.Equal(t, gatewayID, req.GatewayID)
			assert.Equal(t, "send_messages", req.Permission)
			return true, nil
		}

		router.GET("/gateway/:gateway_id/messages", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequirePermission("send_messages"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/messages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Permission denied", func(t *testing.T) {
		mockChecker.CheckPermissionFunc = func(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error) {
			return false, nil
		}

		router.GET("/gateway/:gateway_id/users", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequirePermission("manage_users"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Missing tenant_id", func(t *testing.T) {
		router.GET("/gateway/:gateway_id/test1", 
			middleware.RequirePermission("send_messages"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/test1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing user_id", func(t *testing.T) {
		router.GET("/gateway/:gateway_id/test2", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
			},
			middleware.RequirePermission("send_messages"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/test2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing gateway_id", func(t *testing.T) {
		router.GET("/test3", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequirePermission("send_messages"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/test3", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Permission check error", func(t *testing.T) {
		mockChecker.CheckPermissionFunc = func(ctx context.Context, req *CheckGatewayPermissionRequest) (bool, error) {
			return false, errors.New("database error")
		}

		router.GET("/gateway/:gateway_id/test4", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequirePermission("send_messages"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/test4", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRequireRole(t *testing.T) {
	router, mockChecker := setupGatewayPermissionTest()
	middleware := NewGatewayPermissionMiddleware(mockChecker)

	tenantID := uuid.New()
	userID := uuid.New()
	gatewayID := 1

	t.Run("Admin role granted", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			assert.Equal(t, tenantID, tid)
			assert.Equal(t, userID, uid)
			assert.Equal(t, gatewayID, gid)
			return "admin", nil
		}

		router.GET("/gateway/:gateway_id/admin", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireRole("admin"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Operator role granted for operator or admin", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			return "operator", nil
		}

		router.GET("/gateway/:gateway_id/operate", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireRole("admin", "operator"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/operate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Viewer role denied for admin-only", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			return "viewer", nil
		}

		router.GET("/gateway/:gateway_id/admin-only", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireRole("admin"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/admin-only", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("User not found", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			return "", gorm.ErrRecordNotFound
		}

		router.GET("/gateway/:gateway_id/not-found", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireRole("admin"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/not-found", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestConvenienceMiddlewares(t *testing.T) {
	router, mockChecker := setupGatewayPermissionTest()
	middleware := NewGatewayPermissionMiddleware(mockChecker)

	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("RequireAdminRole", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			return "admin", nil
		}

		router.GET("/gateway/:gateway_id/admin-conv", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireAdminRole(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/admin-conv", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RequireOperatorOrAdmin", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			return "operator", nil
		}

		router.GET("/gateway/:gateway_id/op-admin", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireOperatorOrAdmin(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/op-admin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RequireAnyRole", func(t *testing.T) {
		mockChecker.GetUserRoleFunc = func(ctx context.Context, tid uuid.UUID, gid int, uid uuid.UUID) (string, error) {
			return "viewer", nil
		}

		router.GET("/gateway/:gateway_id/any-role", 
			func(c *gin.Context) {
				c.Set("tenant_id", tenantID)
				c.Set("user_id", userID)
			},
			middleware.RequireAnyRole(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/gateway/1/any-role", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

