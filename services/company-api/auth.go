package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	cfg         *Config
	pool        *pgxpool.Pool
	logger      *zap.Logger
	subManager  subscription.Manager
	jwtSecret   []byte
	jwtLifetime time.Duration
}

type CompanyUser struct {
	ID           int64      `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	Email        string     `json:"email"`
	FullName     string     `json:"full_name,omitempty"`
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AuthClaims is the JWT payload for a company user.
type AuthClaims struct {
	UserID   int64     `json:"uid"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	jwt.RegisteredClaims
}

const (
	ctxUserID   = "company_user_id"
	ctxTenantID = "company_tenant_id"
	ctxEmail    = "company_email"
)

func (s *server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			for _, allowed := range s.cfg.AllowedOrigins {
				if strings.EqualFold(allowed, origin) {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
			if c.GetHeader("Access-Control-Allow-Origin") != "" {
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *server) signToken(user *CompanyUser) (string, error) {
	now := time.Now()
	claims := AuthClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtLifetime)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.ParseWithClaims(tokenStr, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return s.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*AuthClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxTenantID, claims.TenantID.String())
		c.Set(ctxEmail, claims.Email)
		c.Next()
	}
}

func (s *server) handleRegister(c *gin.Context) {
	var req struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		FullName         string `json:"full_name"`
		TenantID         string `json:"tenant_id"`
		RegistrationCode string `json:"registration_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if s.cfg.RegistrationCode == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "registration is disabled on this server"})
		return
	}
	if req.RegistrationCode != s.cfg.RegistrationCode {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid registration code"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(email, "@") || len(email) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email"})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	ctx := c.Request.Context()

	var exists bool
	err = s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1 AND deleted_at IS NULL)`,
		tenantID,
	).Scan(&exists)
	if err != nil {
		s.logger.Error("failed to check tenant", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify tenant"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	var user CompanyUser
	err = s.pool.QueryRow(ctx, `
		INSERT INTO company_users (tenant_id, email, password_hash, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, email, full_name, is_active, last_login_at, created_at, updated_at
	`, tenantID, email, string(hash), req.FullName).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.FullName,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		s.logger.Error("failed to create company user", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	token, err := s.signToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"token":   token,
		"user":    user,
	})
}

func (s *server) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	ctx := c.Request.Context()

	var user CompanyUser
	var passwordHash string
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, COALESCE(full_name, ''), is_active, last_login_at, created_at, updated_at
		FROM company_users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.TenantID, &user.Email, &passwordHash, &user.FullName,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		s.logger.Error("login query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is disabled"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	_, _ = s.pool.Exec(ctx, `UPDATE company_users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	token, err := s.signToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	tenant := s.fetchTenant(ctx, user.TenantID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
		"user":    user,
		"tenant":  tenant,
	})
}

func (s *server) handleMe(c *gin.Context) {
	userID := c.GetInt64(ctxUserID)
	tenantIDStr := c.GetString(ctxTenantID)
	tenantID, _ := uuid.Parse(tenantIDStr)

	var user CompanyUser
	err := s.pool.QueryRow(c.Request.Context(), `
		SELECT id, tenant_id, email, COALESCE(full_name, ''), is_active, last_login_at, created_at, updated_at
		FROM company_users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.TenantID, &user.Email, &user.FullName,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user":    user,
		"tenant":  s.fetchTenant(c.Request.Context(), tenantID),
	})
}

func (s *server) fetchTenant(ctx context.Context, tenantID uuid.UUID) gin.H {
	var tenant gin.H
	err := s.withTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var balance, status, email, businessName, phone string
		err := tx.QueryRow(ctx, `
			SELECT balance::text, status, COALESCE(email, ''), COALESCE(business_name, ''), COALESCE(phone, '')
			FROM tenants WHERE id = $1
		`, tenantID).Scan(&balance, &status, &email, &businessName, &phone)
		if err != nil {
			return err
		}
		tenant = gin.H{
			"id":            tenantID,
			"balance":       balance,
			"status":        status,
			"email":         email,
			"business_name": businessName,
			"phone":         phone,
		}
		return nil
	})
	if err != nil {
		s.logger.Warn("failed to fetch tenant", zap.String("tenant_id", tenantID.String()), zap.Error(err))
		return gin.H{"id": tenantID, "balance": "0"}
	}
	return tenant
}
