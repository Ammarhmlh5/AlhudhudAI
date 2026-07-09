package auth

import (
	"net/http"
	"strings"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for authentication
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler() *Handler {
	return &Handler{
		service: NewService(),
	}
}

// Login handles user login
func (h *Handler) Login(c *gin.Context) {
	log := logger.GetLogger()
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	email := strings.ToLower(strings.TrimSpace(req.Email))
	response, err := h.service.Login(email, req.Password, ipAddress, userAgent)
	if err != nil {
		log.Warn("Login failed", zap.String("email", req.Email), zap.String("ip", ipAddress), zap.Error(err))
		if err == ErrInvalidCredentials {
			db.LogActivity("", "FAILED_LOGIN", "auth", map[string]string{"email": req.Email}, ipAddress)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		if err == ErrUserNotActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "User account is not active"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	db.LogActivity(response.User.ID.String(), "SUCCESSFUL_LOGIN", "auth", map[string]string{"email": req.Email}, ipAddress)

	// Set httpOnly cookies
	h.setAuthCookies(c, response.Token, response.RefreshToken)

	// Restore tokens in response body to allow localStorage support for frontends relying on Bearer standard
	// response.Token = ""
	// response.RefreshToken = ""

	c.JSON(http.StatusOK, response)
}

func (h *Handler) setAuthCookies(c *gin.Context, token, refreshToken string) {
	// Determine if we're in production (HTTPS)
	isProduction := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// In development, use SameSite=Lax to allow cross-port requests (e.g. 5173 -> 8085)
	// In production, use SameSite=Strict for stronger CSRF protection
	if isProduction {
		c.SetSameSite(http.SameSiteStrictMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}

	// Set access token cookie (1 hour)
	c.SetCookie("access_token", token, 3600, "/", "", isProduction, true)

	// Set refresh token cookie (7 days)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/api/v1/auth/refresh", "", isProduction, true)
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	token := extractToken(c.Request)
	if token == "" {
		// Also try to get from cookie if header is missing
		if t, err := c.Cookie("access_token"); err == nil {
			token = t
		}
	}

	if token != "" {
		if err := h.service.Logout(token); err != nil {
			logger.GetLogger().Error("Failed to logout from service", zap.Error(err))
		}
	}

	// Clear cookies regardless of service success
	h.clearAuthCookies(c)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth/refresh", "", false, true)
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var refreshToken string

	// Try to get from cookie first
	if t, err := c.Cookie("refresh_token"); err == nil {
		refreshToken = t
	} else {
		// Fallback to request body for backward compatibility
		var req models.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}

	newToken, err := h.service.RefreshToken(refreshToken)
	if err != nil {
		if err == ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Update access token cookie
	isProduction := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("access_token", newToken, 3600, "/", "", isProduction, true)

	// Don't send token in response body for security
	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed successfully"})
}

// Me returns current user information
func (h *Handler) Me(c *gin.Context) {
	token := extractToken(c.Request)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token provided"})
		return
	}

	user, tenant, staff, permissions, err := h.service.GetUserByToken(token)
	if err != nil {
		if err == ErrInvalidToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"tenant":      tenant,
		"staff":       staff,
		"permissions": permissions,
	})
}

// ChangePassword handles password change
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id") // Should be set by middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}


