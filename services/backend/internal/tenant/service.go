package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/cache"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/finance"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/subscription"
	jwtpkg "github.com/al-hudhud/backend/pkg/jwt"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/al-hudhud/backend/pkg/password"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	cache *cache.QueryCache
}

func NewService(qc *cache.QueryCache) *Service {
	return &Service{cache: qc}
}

// TenantSummary defines the output for the admin tenants list
type TenantSummary struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"business_name"`
	ContactName  string    `json:"contact_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Status       string    `json:"status"`
	Plan         string    `json:"plan"`
	Balance      float64   `json:"balance"`
	Devices      int       `json:"devices"`
	Joined       string    `json:"joined"`
	IsActive     bool      `json:"is_active"`
}

type PaginatedTenants struct {
	Tenants    []TenantSummary `json:"tenants"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
}

// ListTenants retrieves a paginated summarized list of all tenants for the admin dashboard with advanced filters
func (s *Service) ListTenants(page, pageSize int, search, status, sector string, minBalance, maxBalance *float64) (*PaginatedTenants, error) {
	var tenants []models.Tenant
	var totalCount int64

	query := db.GormDB.Model(&models.Tenant{})

	// Exclude reseller companies from the regular companies list (G8)
	query = query.Where("is_reseller = ? OR is_reseller IS NULL", false)

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("business_name ILIKE ? OR contact_name ILIKE ? OR email ILIKE ?", searchTerm, searchTerm, searchTerm)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if sector != "" {
		query = query.Where("sector = ?", sector)
	}

	if minBalance != nil {
		query = query.Where("balance >= ?", *minBalance)
	}

	if maxBalance != nil {
		query = query.Where("balance <= ?", *maxBalance)
	}

	query.Count(&totalCount)

	offset := (page - 1) * pageSize
	if err := query.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, err
	}

	// Resolve the active subscription plan per tenant from the real subscription
	// system instead of the legacy tenants.subscription_plan column.
	planByTenant := map[uuid.UUID]string{}
	rows, err := db.DB.Query(`
		SELECT s.tenant_id::text, p.name
		FROM subscriptions s
		JOIN packages p ON p.id = s.package_id
		WHERE s.status IN ('active', 'grace_period')
		  AND s.end_date >= CURRENT_DATE
	`)
	if err == nil {
		for rows.Next() {
			var tenantIDStr, planName string
			if err := rows.Scan(&tenantIDStr, &planName); err == nil {
				if tid, perr := uuid.Parse(tenantIDStr); perr == nil {
					planByTenant[tid] = planName
				}
			}
		}
		rows.Close()
	}

	var summaries []TenantSummary
	for _, t := range tenants {
		var deviceCount int64
		db.GormDB.Model(&models.PairedDevice{}).Where("tenant_id = ?", t.ID).Count(&deviceCount)

		summaries = append(summaries, TenantSummary{
			ID:           t.ID,
			BusinessName: t.BusinessName,
			ContactName:  t.ContactName,
			Email:        t.Email,
			Phone:        t.Phone,
			Status:       string(t.Status),
			Plan:         planByTenant[t.ID],
			Balance:      t.Balance,
			Devices:      int(deviceCount),
			Joined:       t.CreatedAt.Format("2006-01-02"),
			IsActive:     t.IsActive,
		})
	}

	return &PaginatedTenants{
		Tenants:    summaries,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// RegisterStep1Request defines the input for the first stage of registration
type RegisterStep1Request struct {
	Email       string `json:"email" binding:"required_without=Phone,omitempty,email"`
	Phone       string `json:"phone" binding:"required_without=Email"`
	CountryCode string `json:"country_code"`
	Password    string `json:"password" binding:"required,min=8"`
	GoogleID    string `json:"google_id"`
}

// VerifyOTPRequest defines the input for OTP verification
type VerifyOTPRequest struct {
	Target string `json:"target" binding:"required"` // Email or Phone
	Code   string `json:"code" binding:"required"`
}

// RegisterStep2Request defines the input for completing tenant data
type RegisterStep2Request struct {
	UserID       uuid.UUID `json:"user_id" binding:"required"`
	BusinessName string    `json:"business_name" binding:"required"`
	ContactName  string    `json:"contact_name" binding:"required"`
	Address      string    `json:"address"`
	Sector       string    `json:"sector"`
	Captcha      string    `json:"captcha" binding:"required"` // Simulated for now
}

// RegisterResponse defines the output for tenant registration
type RegisterResponse struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Message  string    `json:"message"`
	DebugOTP string    `json:"debug_otp,omitempty"` // Only for development
}

// RegisterTenantStep1 handles the initial registration and OTP sending
func (s *Service) RegisterTenantStep1(req RegisterStep1Request) (*RegisterResponse, error) {
	log := logger.GetLogger()

	// If Google Login
	if req.GoogleID != "" {
		// logic for google login (simplified)
		return s.handleGoogleRegistration(req)
	}

	passwordHash, err := password.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	target := req.Email
	if target == "" {
		target = req.Phone
	}

	var phonePtr *string
	if req.Phone != "" {
		phonePtr = &req.Phone
	}

	// Create pending registration instead of user
	pending := &models.PendingRegistration{
		Email:        req.Email,
		PhoneNumber:  phonePtr,
		PasswordHash: passwordHash,
		CountryCode:  req.CountryCode,
		Step:         1,
		IsVerified:   false,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // Expires in 24 hours
	}

	err = db.GormDB.Transaction(func(tx *gorm.DB) error {
		// Check if email/phone already exists in Users table (completed registrations)
		var existingUser models.User
		if req.Email != "" {
			if err := tx.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
				return errors.New("email already registered")
			}
		}
		if req.Phone != "" {
			if err := tx.Where("phone_number = ?", req.Phone).First(&existingUser).Error; err == nil {
				return errors.New("phone number already registered")
			}
		}

		// Check if there's an existing pending registration and delete it
		var existingPending models.PendingRegistration
		if req.Email != "" {
			if err := tx.Where("email = ?", req.Email).First(&existingPending).Error; err == nil {
				// Delete old pending registration
				tx.Delete(&existingPending)
				log.Info("Deleted old pending registration", zap.String("email", req.Email))
			}
		}
		if req.Phone != "" {
			if err := tx.Where("phone_number = ?", req.Phone).First(&existingPending).Error; err == nil {
				tx.Delete(&existingPending)
				log.Info("Deleted old pending registration", zap.String("phone", req.Phone))
			}
		}

		// Create new pending registration
		if err := tx.Create(pending).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Generate and Send OTP
	otp := auth.GenerateOTP()
	if err := auth.SendOTP(target, otp); err != nil {
		return nil, err
	}

	log.Info("Pending registration created and OTP sent",
		zap.String("pending_id", pending.ID.String()),
		zap.String("target", target),
	)

	response := &RegisterResponse{
		TenantID: pending.ID, // Return pending ID as tenant_id for now
		Email:    req.Email,
		Message:  "Verification code sent successfully",
	}

	// Developer Convenience: Return OTP in response if in development mode
	appEnv := os.Getenv("APP_ENV")
	log.Info("Checking APP_ENV for Developer Mode", zap.String("app_env", appEnv))
	if appEnv == "development" {
		response.DebugOTP = otp
	}

	return response, nil
}

// VerifyOTP verifies the OTP code and marks the pending registration as verified
func (s *Service) VerifyOTP(req VerifyOTPRequest) error {
	log := logger.GetLogger()

	// Verify OTP
	valid, err := auth.VerifyOTP(req.Target, req.Code)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("invalid or expired verification code")
	}

	// Update pending registration to mark as verified
	result := db.GormDB.Model(&models.PendingRegistration{}).
		Where("email = ? OR phone_number = ?", req.Target, req.Target).
		Updates(map[string]interface{}{
			"is_verified": true,
			"step":        2,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("pending registration not found")
	}

	log.Info("OTP verified successfully",
		zap.String("target", req.Target),
	)

	return nil
}

// RegisterTenantStep2 completes registration by creating User and Tenant from PendingRegistration
func (s *Service) RegisterTenantStep2(req RegisterStep2Request) (*models.Tenant, error) {
	log := logger.GetLogger()

	// Simulate CAPTCHA verification
	if req.Captcha != "HUDHUD_OK" {
		return nil, errors.New("captcha verification failed")
	}

	// Get the verified pending registration
	var pending models.PendingRegistration
	if err := db.GormDB.Where("id = ? AND is_verified = true", req.UserID).First(&pending).Error; err != nil {
		return nil, errors.New("pending registration not found or not verified")
	}

	var tenant *models.Tenant

	// Create User and Tenant in a transaction
	err := db.GormDB.Transaction(func(tx *gorm.DB) error {
		// Create the actual User
		user := &models.User{
			Email:        pending.Email,
			PhoneNumber:  pending.PhoneNumber,
			PasswordHash: pending.PasswordHash,
			UserType:     models.TenantType,
			IsActive:     true,
			IsVerified:   true, // Already verified via OTP
		}

		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// Create the Tenant with API keys
		tenant = &models.Tenant{
			UserID:       user.ID,
			BusinessName: req.BusinessName,
			ContactName:  req.ContactName,
			Email:        user.Email,
			Phone:        "",
			Address:      req.Address,
			Sector:       req.Sector,
			CountryCode:  pending.CountryCode,
			IsActive:     true,
		}

		if user.PhoneNumber != nil {
			tenant.Phone = *user.PhoneNumber
		}

		// Generate API Keys
		tenant.CompanySerial = s.generateCompanySerial()
		tenant.APIKeyAccounting = s.generateAPIKey("ACC")
		tenant.APIKeyPhone = s.generateAPIKey("TEL")
		tenant.APIKeyWeb = s.generateAPIKey("WEB")

		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		// Delete the pending registration after successful creation
		if err := tx.Delete(&pending).Error; err != nil {
			log.Warn("Failed to delete pending registration", zap.Error(err))
			// Don't fail the transaction, just log the warning
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Info("Registration completed successfully",
		zap.String("user_id", tenant.UserID.String()),
		zap.String("tenant_id", tenant.ID.String()),
		zap.String("company_serial", tenant.CompanySerial),
	)

	// [Finance] Auto-create accounting wallet for the new tenant
	go func() {
		if bridge := finance.Bridge(); bridge != nil {
			bridge.OnTenantRegistered(tenant.ID, tenant.BusinessName)
		} else {
			log.Warn("Finance bridge not initialized; skipping wallet account creation for new tenant", zap.String("tenant_id", tenant.ID.String()))
		}
	}()

	return tenant, nil
}


// AdminCreateTenant allows an admin to create a new tenant directly
func (s *Service) AdminCreateTenant(req AdminCreateTenantRequest) (*models.Tenant, error) {
	log := logger.GetLogger()

	// 1. Check if user exists
	var existingUser models.User
	if err := db.GormDB.Where("email = ?", req.Email).Or("phone_number = ?", req.Phone).First(&existingUser).Error; err == nil {
		return nil, errors.New("email or phone number already registered")
	}

	// 2. Hash Password
	passwordHash, err := password.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	phone := req.Phone
	var tenant *models.Tenant

	// 3. Create User and Tenant in Transaction
	log.Info("Starting transaction for user and tenant creation")
	err = db.GormDB.Transaction(func(tx *gorm.DB) error {
		user := &models.User{
			Email:        req.Email,
			PhoneNumber:  &phone,
			PasswordHash: passwordHash,
			UserType:     models.TenantType,
			IsActive:     true,
			IsVerified:   true,
		}

		if req.IsReseller {
			user.UserType = models.ResellerType
		}

		if err := tx.Create(user).Error; err != nil {
			log.Error("Failed to create user in transaction", zap.Error(err))
			return err
		}
		log.Info("User created successfully", zap.String("user_id", user.ID.String()))

		tenant = &models.Tenant{
			UserID:        user.ID,
			BusinessName:  req.BusinessName,
			ContactName:   req.ContactName,
			Email:         user.Email,
			Phone:         phone,
			Address:       "Not Provided",
			Sector:        "Other",
			CountryCode:   req.CountryCode,
			Status:        models.TenantActive,
			IsActive:      true,
			IsReseller:    req.IsReseller,
			CompanySerial: s.generateCompanySerial(),
		}

		tenant.APIKeyAccounting = s.generateAPIKey("ACC")
		tenant.APIKeyPhone = s.generateAPIKey("PHO")
		tenant.APIKeyWeb = s.generateAPIKey("WEB")

		if err := tx.Create(tenant).Error; err != nil {
			log.Error("Failed to create tenant in transaction", zap.Error(err))
			return err
		}
		log.Info("Tenant created successfully", zap.String("tenant_id", tenant.ID.String()))

		return nil
	})

	if err != nil {
		log.Error("Admin tenant creation transaction failed", zap.Error(err))
		return nil, err
	}

	// 4. Create and Migrate Tenant Schema
	log.Info("Creating tenant schema", zap.String("tenant_id", tenant.ID.String()))
	if err := db.CreateTenantSchema(tenant.ID); err != nil {
		log.Error("Failed to create tenant schema", zap.Error(err))
	} else {
		log.Info("Tenant schema created, running migrations")
		if err := db.MigrateTenantSchema(tenant.ID); err != nil {
			log.Error("Failed to migrate tenant schema", zap.Error(err))
		}
	}

	// [Finance] Auto-create accounting wallet for the new tenant
	refType := "TENANT"
	if req.IsReseller {
		refType = "RESELLER"
	}
	if bridge := finance.Bridge(); bridge != nil {
		if refType == "RESELLER" {
			go bridge.OnResellerRegistered(tenant.ID, tenant.BusinessName)
		} else {
			go bridge.OnTenantRegistered(tenant.ID, tenant.BusinessName)
		}
	} else {
		log.Warn("Finance bridge not initialized; skipping wallet account creation for admin tenant", zap.String("tenant_id", tenant.ID.String()), zap.String("ref_type", refType))
	}

	log.Info("AdminCreateTenant service completed successfully")
	return tenant, nil
}


func (s *Service) generateCompanySerial() string {
	var count int64
	db.GormDB.Model(&models.Tenant{}).Count(&count)
	return fmt.Sprintf("HUD-%d-%05d", time.Now().Year(), count+1)
}

func (s *Service) generateAPIKey(prefix string) string {
	u := uuid.New().String()
	return fmt.Sprintf("%s-%s", prefix, u)
}

func (s *Service) sendAPIKeyByEmail(email, key string) {
	log := logger.GetLogger()
	log.Info("Accounting API Key Sent",
		zap.String("email", email),
		zap.String("key", key),
	)
}

func (s *Service) handleGoogleRegistration(req RegisterStep1Request) (*RegisterResponse, error) {
	// Simplified Google registration logic
	var googleIDPtr *string
	if req.GoogleID != "" {
		googleIDPtr = &req.GoogleID
	}

	user := &models.User{
		Email:      req.Email,
		GoogleID:   googleIDPtr,
		UserType:   models.TenantType,
		IsActive:   true,
		IsVerified: true, // Google login is pre-verified
	}

	if err := db.GormDB.Create(user).Error; err != nil {
		return nil, err
	}

	return &RegisterResponse{
		TenantID: user.ID,
		Email:    user.Email,
		Message:  "Google account linked. Please complete your company data.",
	}, nil
}

// GetTenantByID retrieves tenant info by ID with caching
func (s *Service) GetTenantByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	if s.cache == nil {
		var tenant models.Tenant
		if err := db.GormDB.First(&tenant, "id = ?", id).Error; err != nil {
			return nil, err
		}
		return &tenant, nil
	}

	key := redis.TenantCacheKey(id)
	val, err := s.cache.GetOrSet(ctx, key, s.cache.GetTTL("tenant"), func() (interface{}, error) {
		var tenant models.Tenant
		if err := db.GormDB.First(&tenant, "id = ?", id).Error; err != nil {
			return nil, err
		}
		return &tenant, nil
	})

	if err != nil {
		return nil, err
	}

	// Because GetOrSet returns interface{}, we need to handle the conversion
	// and potentially re-unmarshal if it came from cache as raw JSON
	if tenant, ok := val.(*models.Tenant); ok {
		return tenant, nil
	}

	// Handle map[string]interface{} from JSON unmarshaling in GetOrSet
	data, _ := json.Marshal(val)
	var tenant models.Tenant
	if err := json.Unmarshal(data, &tenant); err != nil {
		return nil, err
	}
	return &tenant, nil
}

// GetTenantByUserID retrieves tenant info for a user with caching
func (s *Service) GetTenantByUserID(ctx context.Context, userID uuid.UUID) (*models.Tenant, error) {
	if s.cache == nil {
		var tenant models.Tenant
		if err := db.GormDB.Where("user_id = ?", userID).First(&tenant).Error; err != nil {
			return nil, err
		}
		return &tenant, nil
	}

	key := redis.TenantUserCacheKey(userID)
	val, err := s.cache.GetOrSet(ctx, key, s.cache.GetTTL("tenant"), func() (interface{}, error) {
		var tenant models.Tenant
		if err := db.GormDB.Where("user_id = ?", userID).First(&tenant).Error; err != nil {
			return nil, err
		}
		return &tenant, nil
	})

	if err != nil {
		return nil, err
	}

	if tenant, ok := val.(*models.Tenant); ok {
		return tenant, nil
	}

	data, _ := json.Marshal(val)
	var tenant models.Tenant
	if err := json.Unmarshal(data, &tenant); err != nil {
		return nil, err
	}
	return &tenant, nil
}

// InvalidateCache clears the cache for a tenant
func (s *Service) InvalidateCache(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) {
	if s.cache == nil {
		return
	}
	s.cache.Delete(ctx, fmt.Sprintf("tenant:%s", tenantID.String()))
	if userID != uuid.Nil {
		s.cache.Delete(ctx, fmt.Sprintf("tenant:user:%s", userID.String()))
	}
}

// CleanupExpiredRegistrations removes expired pending registrations from the database
func (s *Service) CleanupExpiredRegistrations() error {
	log := logger.GetLogger()
	now := time.Now()

	result := db.GormDB.Unscoped().Where("expires_at < ?", now).Delete(&models.PendingRegistration{})
	if result.Error != nil {
		log.Error("Failed to cleanup expired registrations", zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.With(zap.Int64("count", result.RowsAffected)).Info("Cleaned up expired registrations")
	}

	return nil
}

// RegisterRequest defines the input for single-step registration
type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Phone           string `json:"phone" binding:"required"`
	CountryCode     string `json:"country_code" binding:"required"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

type AdminCreateTenantRequest struct {
	BusinessName string `json:"business_name" binding:"required"`
	ContactName  string `json:"contact_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone" binding:"required"`
	CountryCode  string `json:"country_code" binding:"required"`
	Password     string `json:"password" binding:"required,min=8"`
	IsReseller   bool   `json:"is_reseller"`
}


// RegisterTenant handles the single-step registration
func (s *Service) RegisterTenant(req RegisterRequest) (*models.LoginResponse, error) {
	log := logger.GetLogger()

	// 1. Check if user exists
	var existingUser models.User
	if err := db.GormDB.Where("email = ?", req.Email).Or("phone_number = ?", req.Phone).First(&existingUser).Error; err == nil {
		return nil, errors.New("email or phone number already registered")
	}

	// 2. Hash Password
	passwordHash, err := password.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	phone := req.Phone

	var user *models.User
	var tenant *models.Tenant

	// 3. Create User and Tenant in Transaction
	err = db.GormDB.Transaction(func(tx *gorm.DB) error {
		user = &models.User{
			Email:        req.Email,
			PhoneNumber:  &phone,
			PasswordHash: passwordHash,
			UserType:     models.TenantType,
			IsActive:     true,
			IsVerified:   true, // Auto-verified for single step flow as requested
		}

		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// Use Email username as default Business Name if not provided (which it isn't in this flow)
		// implementation_plan.md: "Initialize the Tenant with default/placeholder values"
		defaultName := "Tenant " + req.Email

		tenant = &models.Tenant{
			UserID:        user.ID,
			BusinessName:  defaultName,
			ContactName:   "Admin", // Default contact name
			Email:         user.Email,
			Phone:         phone,
			Address:       "Not Provided",
			Sector:        "Other",
			CountryCode:   req.CountryCode,
			IsActive:      true,
			AllowGateways: true,
		}

		// Generate API Keys
		tenant.CompanySerial = s.generateCompanySerial()
		tenant.APIKeyAccounting = s.generateAPIKey("ACC")
		tenant.APIKeyPhone = s.generateAPIKey("TEL")
		tenant.APIKeyWeb = s.generateAPIKey("WEB")

		if err := tx.Create(tenant).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("Registration transaction failed", zap.Error(err))
		return nil, err
	}

	// 4. Generate Tokens (Auto-Login)
	permissions, _ := db.GetUserPermissions(user.ID.String()) // Likely empty for new tenant unless default roles exist

	token, err := jwtpkg.GenerateToken(user.ID.String(), user.Email, string(user.UserType), tenant.ID.String(), "", permissions)
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwtpkg.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Create Session
	session := &models.Session{
		UserID:       user.ID,
		Token:        token,
		RefreshToken: refreshToken,
		IPAddress:    "Registration", // Could pass context info if available
		UserAgent:    "Registration",
		ExpiresAt:    time.Now().Add(jwtpkg.RefreshTokenExpiry),
	}
	db.GormDB.Create(session)

	// Create and Migrate Tenant Schema
	if err := db.CreateTenantSchema(tenant.ID); err != nil {
		log.Error("Failed to create tenant schema", zap.Error(err))
	} else {
		if err := db.MigrateTenantSchema(tenant.ID); err != nil {
			log.Error("Failed to migrate tenant schema", zap.Error(err))
		}
	}

	// [Finance] Auto-create accounting wallet for the new tenant
	go func() {
		if bridge := finance.Bridge(); bridge != nil {
			bridge.OnTenantRegistered(tenant.ID, tenant.BusinessName)
		} else {
			log.Warn("Finance bridge not initialized; skipping wallet account creation for registration tenant", zap.String("tenant_id", tenant.ID.String()))
		}
	}()

	return &models.LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         *user,
		Tenant:       tenant,
		Permissions:  permissions,
	}, nil
}

// GetTenantSubscription returns subscription details for a specific tenant
// Requirements: 2.5
func (s *Service) GetTenantSubscription(tenantID uuid.UUID) (interface{}, error) {
	var sub subscription.Subscription
	if err := db.GormDB.Where("tenant_id = ?", tenantID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, err
	}

	// Fetch package info
	var pkg subscription.Package
	db.GormDB.First(&pkg, sub.PackageID)

	return gin.H{
		"subscription": sub,
		"package":      pkg,
	}, nil
}

// GetTenantTransactions returns transaction history for a specific tenant
// Requirements: 2.5
func (s *Service) GetTenantTransactions(tenantID uuid.UUID) (interface{}, error) {
	var transactions []models.AccountingTransaction
	if err := db.GormDB.Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(50).Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}

// BulkUpdateStatus updates the status for multiple tenants
func (s *Service) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, isActive bool) error {
	return db.GormDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Tenant{}).Where("id IN ?", ids).Update("is_active", isActive).Error; err != nil {
			return err
		}

		// Invalidate cache for all updated tenants
		for _, id := range ids {
			s.InvalidateCache(ctx, id, uuid.Nil)
		}
		return nil
	})
}

// BulkDelete deletes multiple tenants (soft-delete)
func (s *Service) BulkDelete(ctx context.Context, ids []uuid.UUID) error {
	return db.GormDB.Transaction(func(tx *gorm.DB) error {
		// Get UserIDs first
		var tenants []models.Tenant
		if err := tx.Select("id, user_id").Where("id IN ?", ids).Find(&tenants).Error; err != nil {
			return err
		}

		userIDs := make([]uuid.UUID, len(tenants))
		for i, t := range tenants {
			userIDs[i] = t.UserID
			// Invalidate cache
			s.InvalidateCache(ctx, t.ID, t.UserID)
		}

		// Explicitly soft-delete tenants
		if err := tx.Delete(&models.Tenant{}, "id IN ?", ids).Error; err != nil {
			return err
		}

		// Soft-delete users
		if err := tx.Delete(&models.User{}, "id IN ?", userIDs).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetTenantByAPIKey retrieves tenant info by API Key with caching
func (s *Service) GetTenantByAPIKey(ctx context.Context, key string, isAccounting bool) (*models.Tenant, error) {
	prefix := "acc"
	field := "api_key_accounting"
	if key[:3] == "TEL" {
		prefix = "tel"
		field = "api_key_phone"
	} else if key[:3] == "WEB" {
		prefix = "web"
		field = "api_key_web"
	}

	if s.cache == nil {
		var tenant models.Tenant
		if err := db.GormDB.Where(field+" = ?", key).First(&tenant).Error; err != nil {
			return nil, err
		}
		return &tenant, nil
	}

	cacheKey := redis.TenantAPIKeyCacheKey(prefix, key)
	val, err := s.cache.GetOrSet(ctx, cacheKey, s.cache.GetTTL("tenant"), func() (interface{}, error) {
		var tenant models.Tenant
		if err := db.GormDB.Where(field+" = ?", key).First(&tenant).Error; err != nil {
			return nil, err
		}
		return &tenant, nil
	})

	if err != nil {
		return nil, err
	}

	if tenant, ok := val.(*models.Tenant); ok {
		return tenant, nil
	}

	data, _ := json.Marshal(val)
	var tenant models.Tenant
	if err := json.Unmarshal(data, &tenant); err != nil {
		return nil, err
	}
	return &tenant, nil
}
