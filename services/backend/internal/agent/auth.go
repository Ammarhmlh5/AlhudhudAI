package agent

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	JWTKey = "hudhud-agent-jwt-secret-key" // يجب تغييره في الإنتاج
)

// ============================================================================
// Auth Middleware
// ============================================================================

// AuthMiddleware - مصادقة الوكلاء والموظفين
// يتحقق من صحة التوكن ويستخرج المعلومات من него
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "يرجى تسجيل الدخول",
			})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "تنسيق التوكن غير صحيح",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "توكن غير صالح أو منتهي الصلاحية",
			})
			c.Abort()
			return
		}

		// Set context values
		c.Set("user_id", claims.UserID)
		c.Set("agent_id", claims.AgentID)
		c.Set("user_type", claims.UserType)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AgentOnlyMiddleware - يضمن أن المستخدم هو وكيل وليس موظف
func AgentOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")
		if userType != "agent" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "غير مسموح - هذا الإجراء للوكيل فقط",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// EmployeeOrAgentMiddleware - ي允许 الوكيل أو الموظف
// This is the default for most endpoints
func EmployeeOrAgentMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")
		if userType != "agent" && userType != "employee" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "غير مصرح لك بالوصول",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ============================================================================
// Token Validation
// ============================================================================

type TokenClaims struct {
	UserID   string `json:"user_id"`
	AgentID  string `json:"agent_id"`
	Role     string `json:"role"`
	UserType string `json:"user_type"`
	Type     string `json:"type"`
	jwt.RegisteredClaims
}

func validateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(JWTKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		// Check if token is expired
		if claims.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("token expired")
		}

		// Verify it's an access token
		if claims.Type != "access" {
			return nil, errors.New("invalid token type")
		}

		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ============================================================================
// Token Generation (for service use)
// ============================================================================

// GenerateAccessToken - إنشاء توكن وصول
func GenerateAccessToken(userID, agentID uuid.UUID, userType, role string) (string, error) {
	now := time.Now()

	claims := TokenClaims{
		UserID:   userID.String(),
		AgentID:  agentID.String(),
		Role:     role,
		UserType: userType,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTKey))
}

// GenerateRefreshToken - إنشاء توكن تحديث
func GenerateRefreshToken(userID uuid.UUID, userType string) (string, error) {
	now := time.Now()

	claims := TokenClaims{
		UserID:   userID.String(),
		AgentID:  userID.String(), // Same as userID for refresh
		Role:     "",
		UserType: userType,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTKey))
}

// ValidateRefreshToken - التحقق من توكن التحديث
func ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		if claims.Type != "refresh" {
			return nil, errors.New("not a refresh token")
		}
		if claims.ExpiresAt.Before(time.Now()) {
			return nil, errors.New("refresh token expired")
		}
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// ============================================================================
// Helper Functions
// ============================================================================

// GetUserIDFromContext - استخراج معرف المستخدم من السياق
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		return uuid.Nil, errors.New("user_id not found in context")
	}
	return uuid.Parse(userIDStr)
}

// GetAgentIDFromContext - استخراج معرف الوكيل من السياق
func GetAgentIDFromContext(c *gin.Context) (uuid.UUID, error) {
	agentIDStr := c.GetString("agent_id")
	if agentIDStr == "" {
		return uuid.Nil, errors.New("agent_id not found in context")
	}
	return uuid.Parse(agentIDStr)
}

// GetUserTypeFromContext - استخراج نوع المستخدم من السياق
func GetUserTypeFromContext(c *gin.Context) string {
	return c.GetString("user_type")
}

// GetRoleFromContext - استخراج الدور من السياق
func GetRoleFromContext(c *gin.Context) string {
	return c.GetString("role")
}

// IsAgent - هل المستخدم وكيل
func IsAgent(c *gin.Context) bool {
	return c.GetString("user_type") == "agent"
}

// IsEmployee - هل المستخدم موظف
func IsEmployee(c *gin.Context) bool {
	return c.GetString("user_type") == "employee"
}

// IsAgentAdmin - هل المستخدم مدير (وكيل أو موظف مدير)
func IsAgentAdmin(c *gin.Context) bool {
	role := c.GetString("role")
	return c.GetString("user_type") == "agent" || role == "admin"
}