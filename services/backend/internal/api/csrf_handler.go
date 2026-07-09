package api

import (
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// CSRFHandler handles CSRF token generation
type CSRFHandler struct{}

// NewCSRFHandler creates a new CSRF handler
func NewCSRFHandler() *CSRFHandler {
	return &CSRFHandler{}
}

// GetCSRFToken generates and returns a CSRF token
// The token is set in both a cookie and the response body (double-submit pattern)
//
// GET /api/csrf-token
//
// Response:
//
//	{
//	  "success": true,
//	  "csrf_token": "base64-encoded-token"
//	}
//
// The token is also set in a cookie named "csrf_token"
func (h *CSRFHandler) GetCSRFToken(c *gin.Context) {
	// Generate new CSRF token
	token, err := middleware.GenerateCSRFToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CSRF_TOKEN_GENERATION_FAILED",
				"message": "Failed to generate CSRF token",
			},
		})
		return
	}

	// Set token in cookie (HttpOnly for security, but accessible by JavaScript for header)
	// Note: We don't use HttpOnly because the frontend needs to read it to send in headers
	c.SetCookie(
		"csrf_token",           // name
		token,                  // value
		24*60*60,               // maxAge (24 hours in seconds)
		"/",                    // path
		"",                     // domain (empty = current domain)
		false,                  // secure (set to true in production with HTTPS)
		false,                  // httpOnly (false so JavaScript can read it)
	)

	// Also return token in response body
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"csrf_token": token,
		"expires_at": time.Now().Add(24 * time.Hour).Unix(),
	})
}
