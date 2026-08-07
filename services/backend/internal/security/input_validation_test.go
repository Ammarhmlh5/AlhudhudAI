package security

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestRequest builds a request whose target may contain characters that
// url.ParseRequestURI rejects (spaces, quotes, control bytes). It bypasses
// URL parsing so the raw path reaches the router for validation.
func newTestRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, "/", body)
	req.URL.Path = target
	req.RequestURI = target
	return req
}

// TestSQLInjectionPrevention tests SQL injection prevention
func TestSQLInjectionPrevention(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	sqlInjectionPayloads := []string{
		"' OR '1'='1",
		"'; DROP TABLE companies; --",
		"' UNION SELECT * FROM users --",
		"admin'--",
		"' OR 1=1--",
		"1' AND '1'='1",
	}

	endpoints := []string{
		"/api/admin/companies/%s/rating",
		"/api/admin/distribution/%s",
		"/api/admin/routing/%s",
		"/api/admin/signalpush/%s/statistics",
	}

	for _, payload := range sqlInjectionPayloads {
		for _, endpointTemplate := range endpoints {
			endpoint := fmt.Sprintf(endpointTemplate, payload)
			
			t.Run("SQL Injection - "+endpoint, func(t *testing.T) {
				req := newTestRequest("GET", endpoint, nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				w := httptest.NewRecorder()
				
				handler.ServeHTTP(w, req)
				
				// Should return 400 Bad Request or 404 Not Found, not 500
				assert.NotEqual(t, http.StatusInternalServerError, w.Code,
					"SQL injection should not cause internal server error")
				
				// Response should not contain SQL error messages
				assert.NotContains(t, w.Body.String(), "SQL",
					"Response should not leak SQL error details")
				assert.NotContains(t, w.Body.String(), "syntax error",
					"Response should not leak SQL syntax errors")
			})
		}
	}
}

// TestXSSPrevention tests XSS prevention
func TestXSSPrevention(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"<svg onload=alert('XSS')>",
		"javascript:alert('XSS')",
		"<iframe src='javascript:alert(\"XSS\")'></iframe>",
	}

	for _, payload := range xssPayloads {
		t.Run("XSS Prevention - "+payload, func(t *testing.T) {
			// Try to create alert with XSS payload in title
			body := fmt.Sprintf(`{"title":"%s","message":"test","severity":"info"}`, payload)
			req := httptest.NewRequest("POST", "/api/admin/alerts", bytes.NewBufferString(body))
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should either reject the input or sanitize it
			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				// If accepted, response should not contain unsanitized script tags
				assert.NotContains(t, w.Body.String(), "<script>",
					"Response should not contain unsanitized script tags")
				assert.NotContains(t, w.Body.String(), "onerror=",
					"Response should not contain unsanitized event handlers")
			}
		})
	}
}

// TestPathTraversalPrevention tests path traversal prevention
func TestPathTraversalPrevention(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
	}

	for _, payload := range pathTraversalPayloads {
		t.Run("Path Traversal - "+payload, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/admin/companies/"+payload+"/rating", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should return 400 Bad Request or 404 Not Found
			assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
				"Path traversal should be rejected")
			
			// Response should not contain file system paths
			assert.NotContains(t, w.Body.String(), "/etc/",
				"Response should not leak file system paths")
			assert.NotContains(t, w.Body.String(), "\\windows\\",
				"Response should not leak file system paths")
		})
	}
}

// TestInputLengthValidation tests input length validation
func TestInputLengthValidation(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	t.Run("Extremely long tenant ID", func(t *testing.T) {
		longTenantID := string(make([]byte, 10000)) // 10KB tenant ID
		req := newTestRequest("GET", "/api/admin/companies/"+longTenantID+"/rating", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code,
			"Extremely long input should be rejected")
	})
	
	t.Run("Extremely long JSON payload", func(t *testing.T) {
		longMessage := string(make([]byte, 1000000)) // 1MB message
		body := fmt.Sprintf(`{"title":"test","message":"%s","severity":"info"}`, longMessage)
		req := httptest.NewRequest("POST", "/api/admin/alerts", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		// Should return 400 Bad Request or 413 Payload Too Large
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusRequestEntityTooLarge,
			"Extremely long payload should be rejected")
	})
}

// TestSpecialCharacterHandling tests special character handling
func TestSpecialCharacterHandling(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	specialCharPayloads := []string{
		"tenant\x00id", // Null byte
		"tenant\nid",   // Newline
		"tenant\rid",   // Carriage return
		"tenant\tid",   // Tab
		"tenant%00id",  // URL encoded null byte
	}

	for _, payload := range specialCharPayloads {
		t.Run("Special char - "+payload, func(t *testing.T) {
			req := newTestRequest("GET", "/api/admin/companies/"+payload+"/rating", nil)
			req.Header.Set("Authorization", "Bearer "+validToken)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should handle gracefully (400 or 404, not 500)
			assert.NotEqual(t, http.StatusInternalServerError, w.Code,
				"Special characters should be handled gracefully")
		})
	}
}

// TestJSONInjection tests JSON injection prevention
func TestJSONInjection(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	jsonInjectionPayloads := []string{
		`{"title":"test","message":"test","severity":"info","admin":true}`,
		`{"title":"test","message":"test","severity":"info","role":"SUPER_ADMIN"}`,
		`{"title":"test","message":"test","severity":"info","__proto__":{"isAdmin":true}}`,
	}

	for _, payload := range jsonInjectionPayloads {
		t.Run("JSON Injection", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/admin/alerts", bytes.NewBufferString(payload))
			req.Header.Set("Authorization", "Bearer "+validToken)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			// Should either reject extra fields or ignore them
			// Should not grant elevated privileges
			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				assert.NotContains(t, w.Body.String(), `"admin":true`,
					"Extra fields should not be processed")
				assert.NotContains(t, w.Body.String(), `"role":"SUPER_ADMIN"`,
					"Extra fields should not grant privileges")
			}
		})
	}
}

// TestContentTypeValidation tests content type validation
func TestContentTypeValidation(t *testing.T) {
	handler := NewAdminHandler(nil)
	validToken := generateValidToken(t, "admin", []string{"SUPER_ADMIN"})
	
	t.Run("Missing Content-Type", func(t *testing.T) {
		body := `{"title":"test","message":"test","severity":"info"}`
		req := httptest.NewRequest("POST", "/api/admin/alerts", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		// No Content-Type header
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		// Should return 415 Unsupported Media Type or 400 Bad Request
		assert.True(t, w.Code == http.StatusUnsupportedMediaType || w.Code == http.StatusBadRequest,
			"Missing Content-Type should be rejected")
	})
	
	t.Run("Wrong Content-Type", func(t *testing.T) {
		body := `{"title":"test","message":"test","severity":"info"}`
		req := httptest.NewRequest("POST", "/api/admin/alerts", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+validToken)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		// Should return 415 Unsupported Media Type
		assert.Equal(t, http.StatusUnsupportedMediaType, w.Code,
			"Wrong Content-Type should be rejected")
	})
}

