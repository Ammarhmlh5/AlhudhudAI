package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAgentNotFound      = errors.New("agent not found")
	ErrEmployeeNotFound   = errors.New("employee not found")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrAlreadyExists      = errors.New("already exists")
)

const (
	AccessTokenExpiry  = time.Hour
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ============================================================================
// Authentication
// ============================================================================

// AgentLogin - تسجيل دخول الوكيل
func (s *Service) AgentLogin(ctx context.Context, email, password string) (*models.AgentLoginResponse, error) {
	var agent models.Agent
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !agent.IsActive {
		return nil, ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(agent.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, refreshToken, err := s.generateTokens(agent.ID.String(), agent.ID.String(), "agent", "admin")
	if err != nil {
		return nil, err
	}

	// Save session
	s.saveSession(ctx, agent.ID, "agent", token, refreshToken)

	return &models.AgentLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         agent,
		UserType:     "agent",
		AgentID:      int64(agent.ID.ID()),
	}, nil
}

// EmployeeLogin - تسجيل دخول الموظف
func (s *Service) EmployeeLogin(ctx context.Context, email, password string) (*models.AgentLoginResponse, error) {
	var employee models.AgentEmployee
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !employee.IsActive {
		return nil, ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	s.db.WithContext(ctx).Model(&employee).Update("last_login", now)

	token, refreshToken, err := s.generateTokens(employee.ID.String(), employee.AgentID.String(), "employee", employee.Role)
	if err != nil {
		return nil, err
	}

	// Save session
	s.saveSession(ctx, employee.ID, "employee", token, refreshToken)

	return &models.AgentLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         employee,
		UserType:     "employee",
		AgentID:      int64(employee.AgentID.ID()),
	}, nil
}

// ChangePassword - تغيير كلمة المرور
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, userType, currentPassword, newPassword string) error {
	var passwordHash string

	if userType == "agent" {
		var agent models.Agent
		if err := s.db.WithContext(ctx).First(&agent, userID).Error; err != nil {
			return ErrAgentNotFound
		}
		passwordHash = agent.PasswordHash
	} else {
		var employee models.AgentEmployee
		if err := s.db.WithContext(ctx).First(&employee, userID).Error; err != nil {
			return ErrEmployeeNotFound
		}
		passwordHash = employee.PasswordHash
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	if userType == "agent" {
		s.db.WithContext(ctx).Model(&models.Agent{}).Where("id = ?", userID).Update("password_hash", string(hash))
	} else {
		s.db.WithContext(ctx).Model(&models.AgentEmployee{}).Where("id = ?", userID).Update("password_hash", string(hash))
	}

	return nil
}

// ============================================================================
// Token Generation
// ============================================================================

func (s *Service) generateTokens(userID, agentID, userType, role string) (string, string, error) {
	now := time.Now()

	// Access token
	accessClaims := jwt.MapClaims{
		"user_id":   userID,
		"agent_id":  agentID,
		"user_type": userType,
		"role":      role,
		"type":      "access",
		"exp":       now.Add(AccessTokenExpiry).Unix(),
		"iat":       now.Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(JWTKey))
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshClaims := jwt.MapClaims{
		"user_id":   userID,
		"agent_id":  agentID,
		"user_type": userType,
		"type":      "refresh",
		"exp":       now.Add(RefreshTokenExpiry).Unix(),
		"iat":       now.Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(JWTKey))
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *Service) saveSession(ctx context.Context, userID uuid.UUID, userType, token, refreshToken string) {
	session := models.AgentSession{
		UserID:       userID,
		UserType:     userType,
		Token:        token,
		RefreshToken: &refreshToken,
		ExpiresAt:    time.Now().Add(AccessTokenExpiry),
	}
	s.db.WithContext(ctx).Create(&session)
}

// ValidateToken - التحقق من صلاحية التوكن
func (s *Service) ValidateToken(tokenString string) (*models.AgentJWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(JWTKey), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrUnauthorized
	}

	userID, _ := uuid.Parse(claims["user_id"].(string))
	agentID, _ := uuid.Parse(claims["agent_id"].(string))

	return &models.AgentJWTClaims{
		UserID:   int64(userID.ID()),
		AgentID:  int64(agentID.ID()),
		Role:     claims["role"].(string),
		UserType: claims["user_type"].(string),
		Type:     claims["type"].(string),
	}, nil
}

// ============================================================================
// Employee Management
// ============================================================================

// CreateEmployee - إنشاء موظف جديد
func (s *Service) CreateEmployee(ctx context.Context, agentID uuid.UUID, req *models.CreateAgentEmployeeRequest) (*models.AgentEmployee, error) {
	// Check if email exists
	var exists int64
	s.db.WithContext(ctx).Model(&models.AgentEmployee{}).Where("email = ?", req.Email).Count(&exists)
	if exists > 0 {
		return nil, ErrAlreadyExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	employee := &models.AgentEmployee{
		AgentID:      agentID,
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		IsActive:     true,
	}

	if err := s.db.WithContext(ctx).Create(employee).Error; err != nil {
		return nil, err
	}

	// Log activity
	s.logActivity(ctx, agentID, "agent", agentID, "create_employee", "employee", &employee.ID, map[string]interface{}{
		"name":  req.Name,
		"email": req.Email,
		"role":  req.Role,
	})

	return employee, nil
}

// GetEmployee - تفاصيل موظف (ضمن الوكيل نفسه فقط)
func (s *Service) GetEmployee(ctx context.Context, agentID, employeeID uuid.UUID) (*models.AgentEmployee, error) {
	var employee models.AgentEmployee
	err := s.db.WithContext(ctx).
		Where("id = ? AND agent_id = ?", employeeID, agentID).
		First(&employee).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}
	return &employee, nil
}

// ListEmployees - قائمة الموظفين
func (s *Service) ListEmployees(ctx context.Context, agentID uuid.UUID) ([]models.AgentEmployee, error) {
	var employees []models.AgentEmployee
	err := s.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("created_at DESC").
		Find(&employees).Error
	return employees, err
}

// DeleteEmployee - حذف موظف
func (s *Service) DeleteEmployee(ctx context.Context, agentID, employeeID uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND agent_id = ?", employeeID, agentID).
		Delete(&models.AgentEmployee{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrEmployeeNotFound
	}

	// Log activity
	s.logActivity(ctx, agentID, "agent", agentID, "delete_employee", "employee", &employeeID, nil)

	return nil
}

// ============================================================================
// Tenant Management
// ============================================================================

// CreateTenant - إنشاء مستأجر جديد وربطه بالوكيل
// ينشئ حساب مستأجر حقيقي (User + Tenant) ثم يربطه بالوكيل عبر agent_tenants
func (s *Service) CreateTenant(ctx context.Context, agentID, createdBy uuid.UUID, req *models.CreateAgentTenantRequest) (*models.Tenant, error) {
	// Check if a user with this email already exists in the main system
	var existingUser int64
	s.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", req.Email).Count(&existingUser)
	if existingUser > 0 {
		return nil, ErrAlreadyExists
	}

	// Generate a temporary password for the tenant's main-system account
	tempPassword, err := randomPassword()
	if err != nil {
		return nil, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var tenant *models.Tenant

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create main-system user
		user := &models.User{
			Email:        req.Email,
			PasswordHash: string(passwordHash),
			UserType:     models.TenantType,
			IsActive:     true,
			IsVerified:   true,
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// 2. Create the tenant record
		var count int64
		tx.Model(&models.Tenant{}).Count(&count)

		tenant = &models.Tenant{
			UserID:           user.ID,
			BusinessName:     req.Name,
			ContactName:      req.Name,
			Email:            req.Email,
			Phone:            req.Phone,
			Status:           models.TenantActive,
			IsActive:         true,
			CompanySerial:    fmt.Sprintf("HUD-%d-%05d", time.Now().Year(), count+1),
			APIKeyAccounting: generateAPIKey("ACC"),
			APIKeyPhone:      generateAPIKey("PHO"),
			APIKeyWeb:        generateAPIKey("WEB"),
		}
		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		// 3. Link the tenant to the agent
		agentTenant := &models.AgentTenant{
			AgentID:   agentID,
			TenantID:  tenant.ID,
			CreatedBy: &createdBy,
		}
		if err := tx.Create(agentTenant).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 4. Provision the tenant schema (best-effort, same as admin flow)
	if err := db.CreateTenantSchema(tenant.ID); err != nil {
		// Schema provisioning failure is logged but does not fail the request
		_ = err
	} else {
		_ = db.MigrateTenantSchema(tenant.ID)
	}

	// Log activity
	s.logActivity(ctx, agentID, "employee", agentID, "create_tenant", "tenant", &tenant.ID, map[string]interface{}{
		"name":  req.Name,
		"email": req.Email,
	})

	return tenant, nil
}

// GetTenant - تفاصيل مستأجر مرتبط بالوكيل
func (s *Service) GetTenant(ctx context.Context, agentID, tenantID uuid.UUID) (*models.Tenant, error) {
	// Ensure the tenant is linked to this agent
	var linkCount int64
	s.db.WithContext(ctx).Model(&models.AgentTenant{}).
		Where("agent_id = ? AND tenant_id = ?", agentID, tenantID).
		Count(&linkCount)
	if linkCount == 0 {
		return nil, ErrTenantNotFound
	}

	var tenant models.Tenant
	if err := s.db.WithContext(ctx).First(&tenant, tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// ListTenants - قائمة المستأجرين المرتبطين بالوكيل
func (s *Service) ListTenants(ctx context.Context, agentID uuid.UUID) ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := s.db.WithContext(ctx).
		Joins("JOIN agent_tenants at ON at.tenant_id = tenants.id").
		Where("at.agent_id = ?", agentID).
		Order("tenants.created_at DESC").
		Find(&tenants).Error
	return tenants, err
}

// ============================================================================
// Payment Management
// ============================================================================

// CreatePayment - تسجيل دفعة
func (s *Service) CreatePayment(ctx context.Context, agentID, employeeID uuid.UUID, req *models.AgentPaymentRequest) (*models.AgentPayment, error) {
	payment := &models.AgentPayment{
		InvoiceID:     &req.InvoiceID,
		EmployeeID:    &employeeID,
		AgentID:       agentID,
		Amount:        req.Amount,
		Currency:      "SAR",
		PaymentMethod: &req.PaymentMethod,
		TransactionID: &req.TransactionID,
		Status:        "completed",
	}

	if req.Notes != "" {
		payment.Notes = &req.Notes
	}

	if err := s.db.WithContext(ctx).Create(payment).Error; err != nil {
		return nil, err
	}

	// Log activity
	s.logActivity(ctx, agentID, "employee", agentID, "create_payment", "payment", &payment.ID, map[string]interface{}{
		"invoice_id":     req.InvoiceID,
		"amount":         req.Amount,
		"payment_method": req.PaymentMethod,
	})

	return payment, nil
}

// ListPayments - قائمة المدفوعات
func (s *Service) ListPayments(ctx context.Context, agentID uuid.UUID) ([]models.AgentPayment, error) {
	var payments []models.AgentPayment
	err := s.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("created_at DESC").
		Find(&payments).Error
	return payments, err
}

// ============================================================================
// Activity Logging
// ============================================================================

func (s *Service) logActivity(ctx context.Context, userID uuid.UUID, userType string, agentID uuid.UUID, action, entityType string, entityID *uuid.UUID, details map[string]interface{}) {
	log := models.AgentActivityLog{
		UserID:     userID,
		UserType:   userType,
		AgentID:    &agentID,
		Action:     action,
		EntityType: &entityType,
		EntityID:   entityID,
		Details:    details,
	}
	s.db.WithContext(ctx).Create(&log)
}

// ============================================================================
// Utility Functions
// ============================================================================

// GenerateAPIKey - إنشاء مفتاح API
func (s *Service) GenerateAPIKey() (string, string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", "", err
	}

	apiKey := "hud_" + hex.EncodeToString(key)[:32]
	apiSecret := hex.EncodeToString(key)

	return apiKey, apiSecret, nil
}

// generateAPIKey - مفتاح API بنمط بادئة مثل ACC/PHO/WEB
func generateAPIKey(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String())
}

// randomPassword - توليد كلمة مرور مؤقتة
func randomPassword() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
