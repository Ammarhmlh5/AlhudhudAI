// Package gateway provides API key management functionality for dedicated gateways
// Requirements: 12.1-12.11
package gateway

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// APIKeyManager handles gateway API key operations
type APIKeyManager struct {
	db *gorm.DB
}

// NewAPIKeyManager creates a new API key manager instance
func NewAPIKeyManager(db *gorm.DB) *APIKeyManager {
	return &APIKeyManager{
		db: db,
	}
}

// GatewayAPIKey represents an API key for a dedicated gateway
type GatewayAPIKey struct {
	ID            uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID      uuid.UUID              `gorm:"type:uuid;not null" json:"tenant_id"`
	GatewayID     uuid.UUID              `gorm:"type:uuid;not null" json:"gateway_id"`
	KeyName       string                 `gorm:"size:255;not null" json:"key_name"`
	Description   string                 `gorm:"type:text" json:"description"`
	APIKeyHash    string                 `gorm:"size:255;not null;uniqueIndex" json:"-"` // Hidden from JSON
	Permissions   map[string]interface{} `gorm:"type:jsonb;default:'{}';serializer:json" json:"permissions"`
	RateLimit     int                    `gorm:"default:60" json:"rate_limit"` // requests per minute
	AllowedIPs    []string               `gorm:"type:text[]" json:"allowed_ips"`
	IsActive      bool                   `gorm:"default:true" json:"is_active"`
	ExpiresAt     *time.Time             `json:"expires_at"`
	LastUsedAt    *time.Time             `json:"last_used_at"`
	UsageCount    int                    `gorm:"default:0" json:"usage_count"`
	CreatedAt     time.Time              `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time              `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for GatewayAPIKey
func (GatewayAPIKey) TableName() string {
	return "gateway_api_keys"
}

// CreateAPIKeyRequest represents the request to create an API key
type CreateAPIKeyRequest struct {
	TenantID    uuid.UUID              `json:"tenant_id" validate:"required"`
	GatewayID   uuid.UUID              `json:"gateway_id" validate:"required"`
	KeyName     string                 `json:"key_name" validate:"required"`
	Description string                 `json:"description"`
	Permissions map[string]interface{} `json:"permissions"`
	RateLimit   int                    `json:"rate_limit"` // requests per minute
	AllowedIPs  []string               `json:"allowed_ips"`
	ExpiresAt   *time.Time             `json:"expires_at"`
}

// CreateAPIKeyResponse represents the response after creating an API key
type CreateAPIKeyResponse struct {
	APIKey    string         `json:"api_key"` // Plain text key (only shown once)
	KeyInfo   *GatewayAPIKey `json:"key_info"`
	ExpiresAt *time.Time     `json:"expires_at"`
}

// CreateAPIKey creates a new API key for a gateway
// Requirements: 12.1, 12.2, 12.3
func (akm *APIKeyManager) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Validate tenant exists
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate gateway exists and belongs to tenant
	if err := akm.validateGateway(ctx, req.GatewayID, req.TenantID); err != nil {
		return nil, fmt.Errorf("gateway validation failed: %w", err)
	}

	// Validate key name is unique for this gateway
	if err := akm.validateKeyNameUnique(ctx, req.GatewayID, req.KeyName); err != nil {
		return nil, err
	}

	// Generate unique API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the API key using bcrypt
	keyHash, err := hashAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash API key: %w", err)
	}

	// Set default rate limit if not provided
	rateLimit := req.RateLimit
	if rateLimit <= 0 {
		rateLimit = 60 // Default: 60 requests per minute
	}

	// Set default permissions if not provided
	permissions := req.Permissions
	if permissions == nil || len(permissions) == 0 {
		permissions = getDefaultAPIKeyPermissions()
	}

	// Validate IP addresses if provided
	if len(req.AllowedIPs) > 0 {
		if err := validateIPAddresses(req.AllowedIPs); err != nil {
			return nil, fmt.Errorf("invalid IP addresses: %w", err)
		}
	}

	// Create API key record
	gatewayAPIKey := &GatewayAPIKey{
		ID:          uuid.New(),
		TenantID:    req.TenantID,
		GatewayID:   req.GatewayID,
		KeyName:     req.KeyName,
		Description: req.Description,
		APIKeyHash:  keyHash,
		Permissions: permissions,
		RateLimit:   rateLimit,
		AllowedIPs:  req.AllowedIPs,
		IsActive:    true,
		ExpiresAt:   req.ExpiresAt,
	}

	// Insert into database
	if err := akm.db.WithContext(ctx).Create(gatewayAPIKey).Error; err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Return response with plain text key (only shown once)
	return &CreateAPIKeyResponse{
		APIKey:    apiKey,
		KeyInfo:   gatewayAPIKey,
		ExpiresAt: req.ExpiresAt,
	}, nil
}

// SetPermissionsRequest represents the request to set API key permissions
type SetPermissionsRequest struct {
	TenantID    uuid.UUID              `json:"tenant_id" validate:"required"`
	GatewayID   uuid.UUID              `json:"gateway_id" validate:"required"`
	KeyID       uuid.UUID              `json:"key_id" validate:"required"`
	Permissions map[string]interface{} `json:"permissions" validate:"required"`
}

// SetPermissions sets custom permissions for an API key
// Requirements: 12.4
func (akm *APIKeyManager) SetPermissions(ctx context.Context, req *SetPermissionsRequest) (*GatewayAPIKey, error) {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get existing API key
	var apiKey GatewayAPIKey
	err := akm.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		First(&apiKey).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch API key: %w", err)
	}

	// Validate permissions
	if err := validateAPIKeyPermissions(req.Permissions); err != nil {
		return nil, fmt.Errorf("invalid permissions: %w", err)
	}

	// Update permissions
	// The raw column update bypasses the field's serializer, so the map must be
	// serialized to JSON explicitly (jsonb column in Postgres, text in SQLite).
	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize permissions: %w", err)
	}
	if err := akm.db.WithContext(ctx).Model(&apiKey).Update("permissions", string(permissionsJSON)).Error; err != nil {
		return nil, fmt.Errorf("failed to update permissions: %w", err)
	}

	// Reload API key
	if err := akm.db.WithContext(ctx).Where("id = ?", apiKey.ID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to reload API key: %w", err)
	}

	return &apiKey, nil
}

// SetRateLimitRequest represents the request to set API key rate limit
type SetRateLimitRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	KeyID     uuid.UUID `json:"key_id" validate:"required"`
	RateLimit int       `json:"rate_limit" validate:"required,gt=0"`
}

// SetRateLimit sets the rate limit for an API key
// Requirements: 12.5
func (akm *APIKeyManager) SetRateLimit(ctx context.Context, req *SetRateLimitRequest) (*GatewayAPIKey, error) {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate rate limit
	if req.RateLimit <= 0 {
		return nil, errors.New("rate limit must be greater than 0")
	}

	// Update rate limit
	var apiKey GatewayAPIKey
	result := akm.db.WithContext(ctx).
		Model(&apiKey).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		Update("rate_limit", req.RateLimit)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update rate limit: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("API key not found or access denied")
	}

	// Reload API key
	if err := akm.db.WithContext(ctx).Where("id = ?", req.KeyID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to reload API key: %w", err)
	}

	return &apiKey, nil
}

// AddAllowedIPRequest represents the request to add an allowed IP
type AddAllowedIPRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	KeyID     uuid.UUID `json:"key_id" validate:"required"`
	IPAddress string    `json:"ip_address" validate:"required"`
}

// AddAllowedIP adds an IP address to the whitelist
// Requirements: 12.6
func (akm *APIKeyManager) AddAllowedIP(ctx context.Context, req *AddAllowedIPRequest) (*GatewayAPIKey, error) {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate IP address
	if err := validateIPAddresses([]string{req.IPAddress}); err != nil {
		return nil, fmt.Errorf("invalid IP address: %w", err)
	}

	// Get existing API key
	var apiKey GatewayAPIKey
	err := akm.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		First(&apiKey).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch API key: %w", err)
	}

	// Check if IP already exists
	for _, ip := range apiKey.AllowedIPs {
		if ip == req.IPAddress {
			return nil, errors.New("IP address already in whitelist")
		}
	}

	// Add IP to whitelist
	apiKey.AllowedIPs = append(apiKey.AllowedIPs, req.IPAddress)

	// Update in database
	if err := akm.db.WithContext(ctx).Model(&apiKey).Update("allowed_ips", apiKey.AllowedIPs).Error; err != nil {
		return nil, fmt.Errorf("failed to add IP to whitelist: %w", err)
	}

	return &apiKey, nil
}

// RemoveAllowedIPRequest represents the request to remove an allowed IP
type RemoveAllowedIPRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	KeyID     uuid.UUID `json:"key_id" validate:"required"`
	IPAddress string    `json:"ip_address" validate:"required"`
}

// RemoveAllowedIP removes an IP address from the whitelist
// Requirements: 12.6
func (akm *APIKeyManager) RemoveAllowedIP(ctx context.Context, req *RemoveAllowedIPRequest) (*GatewayAPIKey, error) {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get existing API key
	var apiKey GatewayAPIKey
	err := akm.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		First(&apiKey).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API key not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch API key: %w", err)
	}

	// Remove IP from whitelist
	newAllowedIPs := make([]string, 0)
	found := false
	for _, ip := range apiKey.AllowedIPs {
		if ip != req.IPAddress {
			newAllowedIPs = append(newAllowedIPs, ip)
		} else {
			found = true
		}
	}

	if !found {
		return nil, errors.New("IP address not found in whitelist")
	}

	apiKey.AllowedIPs = newAllowedIPs

	// Update in database
	if err := akm.db.WithContext(ctx).Model(&apiKey).Update("allowed_ips", apiKey.AllowedIPs).Error; err != nil {
		return nil, fmt.Errorf("failed to remove IP from whitelist: %w", err)
	}

	return &apiKey, nil
}

// SetExpiryRequest represents the request to set API key expiry
type SetExpiryRequest struct {
	TenantID  uuid.UUID  `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID  `json:"gateway_id" validate:"required"`
	KeyID     uuid.UUID  `json:"key_id" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at"` // nil means no expiration
}

// SetExpiry sets the expiration date for an API key
// Requirements: 12.7
func (akm *APIKeyManager) SetExpiry(ctx context.Context, req *SetExpiryRequest) (*GatewayAPIKey, error) {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate expiry date is in the future
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("expiry date must be in the future")
	}

	// Update expiry date
	var apiKey GatewayAPIKey
	result := akm.db.WithContext(ctx).
		Model(&apiKey).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		Update("expires_at", req.ExpiresAt)
	
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update expiry date: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("API key not found or access denied")
	}

	// Reload API key
	if err := akm.db.WithContext(ctx).Where("id = ?", req.KeyID).First(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to reload API key: %w", err)
	}

	return &apiKey, nil
}

// Helper methods

// generateAPIKey generates a cryptographically secure random API key
func generateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64 URL-safe format
	apiKey := base64.URLEncoding.EncodeToString(bytes)
	
	// Add prefix for identification
	return "gw_" + apiKey, nil
}

// hashAPIKey hashes an API key using bcrypt
func hashAPIKey(apiKey string) (string, error) {
	// Use bcrypt with cost 12 (good balance between security and performance)
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// validateTenant checks if tenant exists
func (akm *APIKeyManager) validateTenant(ctx context.Context, tenantID uuid.UUID) error {
	var count int64
	if err := akm.db.WithContext(ctx).Table("tenants").Where("id = ?", tenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate tenant: %w", err)
	}
	if count == 0 {
		return errors.New("tenant not found")
	}
	return nil
}

// validateGateway checks if gateway exists and belongs to tenant
func (akm *APIKeyManager) validateGateway(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID) error {
	var count int64
	err := akm.db.WithContext(ctx).
		Table("dedicated_gateways").
		Where("id = ? AND tenant_id = ?", gatewayID, tenantID).
		Count(&count).Error
	
	if err != nil {
		return fmt.Errorf("failed to validate gateway: %w", err)
	}
	if count == 0 {
		return errors.New("gateway not found or does not belong to tenant")
	}
	return nil
}

// validateKeyNameUnique checks if key name is unique for the gateway
func (akm *APIKeyManager) validateKeyNameUnique(ctx context.Context, gatewayID uuid.UUID, keyName string) error {
	var count int64
	err := akm.db.WithContext(ctx).
		Table("gateway_api_keys").
		Where("gateway_id = ? AND key_name = ?", gatewayID, keyName).
		Count(&count).Error
	
	if err != nil {
		return fmt.Errorf("failed to check key name uniqueness: %w", err)
	}
	if count > 0 {
		return errors.New("API key with this name already exists for this gateway")
	}
	return nil
}

// getDefaultAPIKeyPermissions returns default permissions for API keys
func getDefaultAPIKeyPermissions() map[string]interface{} {
	return map[string]interface{}{
		"send_sms":     true,
		"view_stats":   true,
		"view_reports": true,
		"manage_gateway": false,
		"manage_users":   false,
		"manage_api_keys": false,
	}
}

// validateAPIKeyPermissions validates that all permissions are valid
func validateAPIKeyPermissions(permissions map[string]interface{}) error {
	validPermissions := map[string]bool{
		"send_sms":        true,
		"view_stats":      true,
		"view_reports":    true,
		"manage_gateway":  true,
		"manage_users":    true,
		"manage_api_keys": true,
	}

	for key := range permissions {
		if !validPermissions[key] {
			return fmt.Errorf("invalid permission: %s", key)
		}
	}

	return nil
}

// validateIPAddresses validates a list of IP addresses or CIDR notations
func validateIPAddresses(ips []string) error {
	for _, ipStr := range ips {
		// Try parsing as IP address
		if ip := net.ParseIP(ipStr); ip != nil {
			continue
		}

		// Try parsing as CIDR notation
		if _, _, err := net.ParseCIDR(ipStr); err == nil {
			continue
		}

		return fmt.Errorf("invalid IP address or CIDR notation: %s", ipStr)
	}

	return nil
}


// ValidateAPIKeyRequest represents the request to validate an API key
type ValidateAPIKeyRequest struct {
	APIKey    string    `json:"api_key" validate:"required"`
	ClientIP  string    `json:"client_ip" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id"`
}

// ValidateAPIKeyResponse represents the response after validating an API key
type ValidateAPIKeyResponse struct {
	IsValid     bool                   `json:"is_valid"`
	KeyID       uuid.UUID              `json:"key_id,omitempty"`
	GatewayID   uuid.UUID              `json:"gateway_id,omitempty"`
	TenantID    uuid.UUID              `json:"tenant_id,omitempty"`
	Permissions map[string]interface{} `json:"permissions,omitempty"`
	RateLimit   int                    `json:"rate_limit,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// ValidateAPIKey validates an API key and checks permissions, IP, and expiry
// Requirements: 12.8, 12.10, 12.11
func (akm *APIKeyManager) ValidateAPIKey(ctx context.Context, req *ValidateAPIKeyRequest) (*ValidateAPIKeyResponse, error) {
	// Parse client IP
	clientIP := net.ParseIP(req.ClientIP)
	if clientIP == nil {
		return &ValidateAPIKeyResponse{
			IsValid: false,
			Error:   "Invalid client IP address",
		}, nil
	}

	// Find all API keys and check against each hash
	var apiKeys []GatewayAPIKey
	query := akm.db.WithContext(ctx).Where("is_active = ?", true)
	
	// If gateway_id is provided, filter by it
	if req.GatewayID != uuid.Nil {
		query = query.Where("gateway_id = ?", req.GatewayID)
	}
	
	if err := query.Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch API keys: %w", err)
	}

	// Try to match the API key with stored hashes
	var matchedKey *GatewayAPIKey
	for i := range apiKeys {
		if err := bcrypt.CompareHashAndPassword([]byte(apiKeys[i].APIKeyHash), []byte(req.APIKey)); err == nil {
			matchedKey = &apiKeys[i]
			break
		}
	}

	if matchedKey == nil {
		return &ValidateAPIKeyResponse{
			IsValid: false,
			Error:   "Invalid API key",
		}, nil
	}

	// Check if key is active
	if !matchedKey.IsActive {
		return &ValidateAPIKeyResponse{
			IsValid: false,
			KeyID:   matchedKey.ID,
			Error:   "API key is not active",
		}, nil
	}

	// Check expiry date (Requirement 12.11)
	if matchedKey.ExpiresAt != nil && matchedKey.ExpiresAt.Before(time.Now()) {
		return &ValidateAPIKeyResponse{
			IsValid:   false,
			KeyID:     matchedKey.ID,
			GatewayID: matchedKey.GatewayID,
			TenantID:  matchedKey.TenantID,
			Error:     "API key has expired",
		}, nil
	}

	// Check IP whitelisting (Requirement 12.10)
	if len(matchedKey.AllowedIPs) > 0 {
		ipAllowed := false
		for _, allowedIP := range matchedKey.AllowedIPs {
			// Try parsing as single IP
			if ip := net.ParseIP(allowedIP); ip != nil {
				if ip.Equal(clientIP) {
					ipAllowed = true
					break
				}
				continue
			}

			// Try parsing as CIDR
			if _, ipNet, err := net.ParseCIDR(allowedIP); err == nil {
				if ipNet.Contains(clientIP) {
					ipAllowed = true
					break
				}
			}
		}

		if !ipAllowed {
			// Log unauthorized access attempt
			akm.logUnauthorizedAccess(ctx, matchedKey, req.ClientIP)
			
			return &ValidateAPIKeyResponse{
				IsValid:   false,
				KeyID:     matchedKey.ID,
				GatewayID: matchedKey.GatewayID,
				TenantID:  matchedKey.TenantID,
				Error:     "IP address not whitelisted",
			}, nil
		}
	}

	// Update last_used_at and usage_count (Requirement 12.8)
	now := time.Now()
	if err := akm.db.WithContext(ctx).Model(matchedKey).Updates(map[string]interface{}{
		"last_used_at": now,
		"usage_count":  gorm.Expr("usage_count + 1"),
	}).Error; err != nil {
		// Log error but don't fail validation
		fmt.Printf("Warning: failed to update API key usage: %v\n", err)
	}

	// API key is valid
	return &ValidateAPIKeyResponse{
		IsValid:     true,
		KeyID:       matchedKey.ID,
		GatewayID:   matchedKey.GatewayID,
		TenantID:    matchedKey.TenantID,
		Permissions: matchedKey.Permissions,
		RateLimit:   matchedKey.RateLimit,
	}, nil
}

// GetAPIKeysByGatewayRequest represents the request to get API keys for a gateway
type GetAPIKeysByGatewayRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
}

// GetAPIKeysByGateway retrieves all API keys for a specific gateway
func (akm *APIKeyManager) GetAPIKeysByGateway(ctx context.Context, req *GetAPIKeysByGatewayRequest) ([]GatewayAPIKey, error) {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Validate gateway
	if err := akm.validateGateway(ctx, req.GatewayID, req.TenantID); err != nil {
		return nil, fmt.Errorf("gateway validation failed: %w", err)
	}

	var apiKeys []GatewayAPIKey
	err := akm.db.WithContext(ctx).
		Where("tenant_id = ? AND gateway_id = ?", req.TenantID, req.GatewayID).
		Order("created_at DESC").
		Find(&apiKeys).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to fetch API keys: %w", err)
	}

	return apiKeys, nil
}

// UpdateAPIKeyStatusRequest represents the request to update API key status
type UpdateAPIKeyStatusRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	KeyID     uuid.UUID `json:"key_id" validate:"required"`
	IsActive  bool      `json:"is_active"`
}

// UpdateAPIKeyStatus activates or deactivates an API key
// Requirements: 12.9
func (akm *APIKeyManager) UpdateAPIKeyStatus(ctx context.Context, req *UpdateAPIKeyStatusRequest) error {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// Update status
	result := akm.db.WithContext(ctx).
		Model(&GatewayAPIKey{}).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		Update("is_active", req.IsActive)
	
	if result.Error != nil {
		return fmt.Errorf("failed to update API key status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("API key not found or access denied")
	}

	return nil
}

// DeleteAPIKeyRequest represents the request to delete an API key
type DeleteAPIKeyRequest struct {
	TenantID  uuid.UUID `json:"tenant_id" validate:"required"`
	GatewayID uuid.UUID `json:"gateway_id" validate:"required"`
	KeyID     uuid.UUID `json:"key_id" validate:"required"`
}

// DeleteAPIKey deletes an API key
func (akm *APIKeyManager) DeleteAPIKey(ctx context.Context, req *DeleteAPIKeyRequest) error {
	// Validate tenant
	if err := akm.validateTenant(ctx, req.TenantID); err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// Delete API key
	result := akm.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND gateway_id = ?", req.KeyID, req.TenantID, req.GatewayID).
		Delete(&GatewayAPIKey{})
	
	if result.Error != nil {
		return fmt.Errorf("failed to delete API key: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("API key not found or access denied")
	}

	return nil
}

// logUnauthorizedAccess logs an unauthorized API key access attempt
func (akm *APIKeyManager) logUnauthorizedAccess(ctx context.Context, apiKey *GatewayAPIKey, clientIP string) {
	// This would typically log to a security_logs table
	// For now, we'll just print to console
	fmt.Printf("SECURITY: Unauthorized API key access attempt - Key ID: %d, Gateway ID: %d, Client IP: %s\n",
		apiKey.ID, apiKey.GatewayID, clientIP)
	
	// In production, you would insert into security_logs table:
	// INSERT INTO security_logs (tenant_id, event_type, resource_type, resource_id, ip_address, details, created_at)
	// VALUES (apiKey.TenantID, 'UNAUTHORIZED_API_KEY_ACCESS', 'gateway_api_key', apiKey.ID, clientIP, {...}, NOW())
}
