package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCrossTenantDataAccess tests that tenants cannot access other tenants' data
func TestCrossTenantDataAccess(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	// Generate token for tenant-1
	tenant1Token := generateValidToken(t, "user-1", []string{"TENANT_ADMIN"})
	
	// Try to access tenant-2's data
	endpoints := []struct {
		path   string
		method string
	}{
		{"/api/admin/companies/tenant-2/rating", "GET"},
		{"/api/admin/companies/tenant-2/metrics", "GET"},
		{"/api/admin/distribution/tenant-2", "GET"},
		{"/api/admin/signalpush/tenant-2/statistics", "GET"},
		{"/api/admin/routing/tenant-2", "GET"},
	}

	for _, endpoint := range endpoints {
		t.Run("Cross-tenant access - "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			req.Header.Set("Authorization", "Bearer "+tenant1Token)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should return 403 Forbidden
			assert.Equal(t, http.StatusForbidden, w.Code,
				"Should not allow cross-tenant data access")
		})
	}
}

// TestRoleBasedPermissions tests role-based access control
func TestRoleBasedPermissions(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	testCases := []struct {
		role           string
		endpoint       string
		method         string
		expectedStatus int
		description    string
	}{
		{
			role:           "SUPER_ADMIN",
			endpoint:       "/api/admin/companies/ratings",
			method:         "GET",
			expectedStatus: http.StatusOK,
			description:    "Super admin can view all companies",
		},
		{
			role:           "STAFF",
			endpoint:       "/api/admin/companies/ratings",
			method:         "GET",
			expectedStatus: http.StatusOK,
			description:    "Staff can view companies",
		},
		{
			role:           "TENANT_ADMIN",
			endpoint:       "/api/admin/companies/ratings",
			method:         "GET",
			expectedStatus: http.StatusForbidden,
			description:    "Tenant admin cannot view all companies",
		},
		{
			role:           "SUPER_ADMIN",
			endpoint:       "/api/admin/routing/tenant-1",
			method:         "PUT",
			expectedStatus: http.StatusOK,
			description:    "Super admin can update routing config",
		},
		{
			role:           "STAFF",
			endpoint:       "/api/admin/routing/tenant-1",
			method:         "PUT",
			expectedStatus: http.StatusForbidden,
			description:    "Staff cannot update routing config",
		},
		{
			role:           "TENANT_ADMIN",
			endpoint:       "/api/admin/routing/tenant-1",
			method:         "PUT",
			expectedStatus: http.StatusForbidden,
			description:    "Tenant admin cannot update routing config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			token := generateValidToken(t, "test-user", []string{tc.role})
			
			req := httptest.NewRequest(tc.method, tc.endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedStatus, w.Code, tc.description)
		})
	}
}

// TestSuperAdminAccess tests that super admin has access to all resources
func TestSuperAdminAccess(t *testing.T) {
	handler := NewAdminHandler(nil)
	superAdminToken := generateValidToken(t, "super-admin", []string{"SUPER_ADMIN"})
	
	endpoints := []struct {
		path   string
		method string
	}{
		{"/api/admin/dashboard/overview", "GET"},
		{"/api/admin/companies/ratings", "GET"},
		{"/api/admin/companies/tenant-1/rating", "GET"},
		{"/api/admin/servers", "GET"},
		{"/api/admin/alerts", "GET"},
		{"/api/admin/alerts/config", "POST"},
		{"/api/admin/distribution/tenant-1", "GET"},
		{"/api/admin/distribution/tenant-1", "PUT"},
		{"/api/admin/routing/tenant-1", "GET"},
		{"/api/admin/routing/tenant-1", "PUT"},
		{"/api/admin/signalpush/tenant-1/statistics", "GET"},
		{"/api/admin/signalpush/tenant-1/force-mode", "POST"},
	}

	for _, endpoint := range endpoints {
		t.Run("Super admin access - "+endpoint.method+" "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			req.Header.Set("Authorization", "Bearer "+superAdminToken)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should not return 403 Forbidden
			assert.NotEqual(t, http.StatusForbidden, w.Code,
				"Super admin should have access to all endpoints")
		})
	}
}

// TestStaffLimitedAccess tests that staff has limited access
func TestStaffLimitedAccess(t *testing.T) {
	handler := NewAdminHandler(nil)
	staffToken := generateValidToken(t, "staff-user", []string{"STAFF"})
	
	allowedEndpoints := []struct {
		path   string
		method string
	}{
		{"/api/admin/dashboard/overview", "GET"},
		{"/api/admin/companies/ratings", "GET"},
		{"/api/admin/companies/tenant-1/rating", "GET"},
		{"/api/admin/servers", "GET"},
		{"/api/admin/alerts", "GET"},
	}

	forbiddenEndpoints := []struct {
		path   string
		method string
	}{
		{"/api/admin/alerts/config", "POST"},
		{"/api/admin/distribution/tenant-1", "PUT"},
		{"/api/admin/routing/tenant-1", "PUT"},
		{"/api/admin/signalpush/tenant-1/force-mode", "POST"},
	}

	for _, endpoint := range allowedEndpoints {
		t.Run("Staff allowed - "+endpoint.method+" "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			req.Header.Set("Authorization", "Bearer "+staffToken)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			assert.NotEqual(t, http.StatusForbidden, w.Code,
				"Staff should have read access")
		})
	}

	for _, endpoint := range forbiddenEndpoints {
		t.Run("Staff forbidden - "+endpoint.method+" "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			req.Header.Set("Authorization", "Bearer "+staffToken)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusForbidden, w.Code,
				"Staff should not have write access to critical endpoints")
		})
	}
}

// TestTenantIsolation tests that tenant data is properly isolated
func TestTenantIsolation(t *testing.T) {
	handler := NewAdminHandler(nil)
	
	// Create two tenant tokens
	tenant1Token := generateValidToken(t, "user-1", []string{"TENANT_ADMIN"})
	tenant2Token := generateValidToken(t, "user-2", []string{"TENANT_ADMIN"})
	
	// Tenant 1 tries to access their own data - should succeed
	req1 := httptest.NewRequest("GET", "/api/admin/companies/tenant-1/rating", nil)
	req1.Header.Set("Authorization", "Bearer "+tenant1Token)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.NotEqual(t, http.StatusForbidden, w1.Code,
		"Tenant should access their own data")
	
	// Tenant 1 tries to access Tenant 2's data - should fail
	req2 := httptest.NewRequest("GET", "/api/admin/companies/tenant-2/rating", nil)
	req2.Header.Set("Authorization", "Bearer "+tenant1Token)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code,
		"Tenant should not access other tenant's data")
	
	// Tenant 2 tries to access their own data - should succeed
	req3 := httptest.NewRequest("GET", "/api/admin/companies/tenant-2/rating", nil)
	req3.Header.Set("Authorization", "Bearer "+tenant2Token)
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	assert.NotEqual(t, http.StatusForbidden, w3.Code,
		"Tenant should access their own data")
}

// TestPermissionEscalation tests that users cannot escalate their permissions
func TestPermissionEscalation(t *testing.T) {
	handler := NewAdminHandler(nil)
	staffToken := generateValidToken(t, "staff-user", []string{"STAFF"})
	
	// Try to update own role to SUPER_ADMIN
	req := httptest.NewRequest("PUT", "/api/admin/users/staff-user/role", nil)
	req.Header.Set("Authorization", "Bearer "+staffToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	// Should return 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code,
		"Users should not be able to escalate their own permissions")
}
