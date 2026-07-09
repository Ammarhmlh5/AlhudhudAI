package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetTenantIDOrAbort extracts and validates tenant_id from context
// Returns the tenant UUID and true if successful, or aborts the request and returns false
func GetTenantIDOrAbort(c *gin.Context) (uuid.UUID, bool) {
	tenantIDStr := c.GetString("tenant_id")
	if tenantIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context missing"})
		c.Abort()
		return uuid.Nil, false
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID in context"})
		c.Abort()
		return uuid.Nil, false
	}

	return tenantID, true
}

// GetTenantIDString extracts tenant_id as string from context
// Returns empty string if not found
func GetTenantIDString(c *gin.Context) string {
	return c.GetString("tenant_id")
}

// MustGetTenantID extracts tenant_id and panics if not found
// Use only in contexts where tenant_id is guaranteed to exist
func MustGetTenantID(c *gin.Context) uuid.UUID {
	tenantIDStr := c.GetString("tenant_id")
	if tenantIDStr == "" {
		panic("tenant_id not found in context")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		panic("invalid tenant_id format: " + err.Error())
	}

	return tenantID
}
