package dispatcher

import (
	"github.com/google/uuid"
)

// Helper functions

// mustParseTenantID parses a tenant ID string and panics if invalid
func mustParseTenantID(tenantIDStr string) uuid.UUID {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		panic("invalid tenant ID: " + tenantIDStr)
	}
	return tenantID
}
