package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// tenantScope describes the tenant context of an authenticated request.
type tenantScope struct {
	TenantID     uuid.UUID
	IsSuperAdmin bool
	Exists       bool // tenant_id key present in auth context
	Valid        bool // tenant_id is a parseable UUID
}

// resolveTenantScope extracts the tenant scope from the auth context.
// AuthMiddleware always sets tenant_id (possibly empty for SUPER_ADMIN),
// so handlers use this helper to distinguish a real tenant from a super admin.
func resolveTenantScope(c *gin.Context) tenantScope {
	scope := tenantScope{IsSuperAdmin: c.GetString("user_type") == "SUPER_ADMIN"}
	raw, exists := c.Get("tenant_id")
	if !exists {
		return scope
	}
	scope.Exists = true

	str, ok := raw.(string)
	if !ok || str == "" {
		return scope
	}

	tenantID, err := uuid.Parse(str)
	if err != nil {
		return scope
	}
	scope.TenantID = tenantID
	scope.Valid = true
	return scope
}
