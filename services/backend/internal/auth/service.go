package auth

import (
	"errors"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/jwt"
	"github.com/al-hudhud/backend/pkg/password"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotActive      = errors.New("user account is not active")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// Service handles authentication business logic
type Service struct{}

// NewService creates a new auth service
func NewService() *Service {
	return &Service{}
}

// Login authenticates a user and returns tokens
func (s *Service) Login(email, passwordStr, ipAddress, userAgent string) (*models.LoginResponse, error) {
	var user models.User
	result := db.GormDB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, result.Error
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	if !password.CheckPassword(passwordStr, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Get permissions
	permissions, err := db.GetUserPermissions(user.ID.String())
	if err != nil {
		return nil, err
	}

	var tenant *models.Tenant
	var staff *models.StaffProfile
	var tenantIDStr, staffIDStr string

	if user.UserType == models.TenantType {
		tenant = &models.Tenant{}
		if err := db.GormDB.Where("user_id = ?", user.ID).First(tenant).Error; err == nil {
			tenantIDStr = tenant.ID.String()
		}
	}

	if user.UserType == models.StaffType {
		staff = &models.StaffProfile{}
		if err := db.GormDB.Preload("Role").Where("user_id = ?", user.ID).First(staff).Error; err == nil {
			staffIDStr = staff.ID.String()
		}
	}

	token, err := jwt.GenerateToken(user.ID.String(), user.Email, string(user.UserType), tenantIDStr, staffIDStr, permissions)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Session management using GORM
	session := &models.Session{
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(jwt.RefreshTokenExpiry),
	}

	if err := db.GormDB.Create(session).Error; err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
		Tenant:       tenant,
		Staff:        staff,
		Permissions:  permissions,
	}, nil
}

// Logout invalidates a user's session
func (s *Service) Logout(token string) error {
	return db.GormDB.Where("token = ?", token).Delete(&models.Session{}).Error
}

// RefreshToken generates a new access token from a refresh token
func (s *Service) RefreshToken(refreshToken string) (string, error) {
	var session models.Session
	err := db.GormDB.Where("refresh_token = ?", refreshToken).First(&session).Error
	if err != nil {
		return "", ErrInvalidToken
	}

	if time.Now().After(session.ExpiresAt) {
		db.GormDB.Delete(&session)
		return "", ErrInvalidToken
	}

	var user models.User
	if err := db.GormDB.First(&user, session.UserID).Error; err != nil {
		return "", err
	}

	if !user.IsActive {
		return "", ErrUserNotActive
	}

	permissions, err := db.GetUserPermissions(user.ID.String())
	if err != nil {
		return "", err
	}

	var tenantIDStr, staffIDStr string
	if user.UserType == models.TenantType {
		var t models.Tenant
		if err := db.GormDB.Where("user_id = ?", user.ID).First(&t).Error; err == nil {
			tenantIDStr = t.ID.String()
		}
	}
	if user.UserType == models.StaffType {
		var sp models.StaffProfile
		if err := db.GormDB.Where("user_id = ?", user.ID).First(&sp).Error; err == nil {
			staffIDStr = sp.ID.String()
		}
	}

	newToken, err := jwt.GenerateToken(user.ID.String(), user.Email, string(user.UserType), tenantIDStr, staffIDStr, permissions)
	if err != nil {
		return "", err
	}

	db.GormDB.Model(&session).Update("token", newToken)

	return newToken, nil
}

// GetUserByToken retrieves user information from a token
func (s *Service) GetUserByToken(token string) (*models.User, *models.Tenant, *models.StaffProfile, []string, error) {
	claims, err := jwt.ValidateToken(token)
	if err != nil {
		return nil, nil, nil, nil, ErrInvalidToken
	}

	var user models.User
	if err := db.GormDB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, nil, nil, ErrUserNotActive
	}

	var tenant *models.Tenant
	var staff *models.StaffProfile

	if user.UserType == models.TenantType && claims.TenantID != "" {
		tenant = &models.Tenant{}
		db.GormDB.Where("id = ?", claims.TenantID).First(tenant)
	}

	if user.UserType == models.StaffType && claims.StaffID != "" {
		staff = &models.StaffProfile{}
		db.GormDB.Preload("Role").Where("id = ?", claims.StaffID).First(staff)
	}

	return &user, tenant, staff, claims.Permissions, nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(userID, oldPassword, newPassword string) error {
	if !password.ValidatePasswordStrength(newPassword) {
		return errors.New("password does not meet strength requirements")
	}

	var user models.User
	if err := db.GormDB.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	if !password.CheckPassword(oldPassword, user.PasswordHash) {
		return errors.New("old password is incorrect")
	}

	newHash, err := password.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return db.GormDB.Model(&user).Update("password_hash", newHash).Error
}
