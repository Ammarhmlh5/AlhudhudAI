// Package gateway provides management functionality for dedicated gateways
// Requirements: 8.1-8.15, 26.1, 28.5
package gateway

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Manager handles dedicated gateway operations
type Manager struct {
	db            *gorm.DB
	encryptionKey []byte // AES-256 key (32 bytes)
}

// NewManager creates a new gateway manager instance
func NewManager(db *gorm.DB, encryptionKey string) (*Manager, error) {
	if len(encryptionKey) != 32 {
		return nil, errors.New("encryption key must be 32 bytes for AES-256")
	}

	return &Manager{
		db:            db,
		encryptionKey: []byte(encryptionKey),
	}, nil
}

// CreateGatewayRequest represents the request to create a new dedicated gateway
type CreateGatewayRequest struct {
	TenantID             uuid.UUID              `json:"tenant_id" validate:"required"`
	GatewayName          string                 `json:"gateway_name" validate:"required,min=3,max=255"`
	GatewayCode          string                 `json:"gateway_code" validate:"required,min=3,max=50"`
	Description          string                 `json:"description"`
	TelecomProvider      string                 `json:"telecom_provider" validate:"required"`
	TelecomAPIEndpoint   string                 `json:"telecom_api_endpoint" validate:"required,url"`
	TelecomUsername      string                 `json:"telecom_username"`
	TelecomPassword      string                 `json:"telecom_password"` // Will be encrypted
	TelecomAPIKey        string                 `json:"telecom_api_key"`  // Will be encrypted
	ServerType           string                 `json:"server_type" validate:"required,oneof=dedicated vps container shared"`
	ServerIP             string                 `json:"server_ip"`
	ServerHostname       string                 `json:"server_hostname"`
	ServerSpecs          map[string]interface{} `json:"server_specs"`
	MaxThroughput        int                    `json:"max_throughput" validate:"required,min=1"`
	DailyLimit           *int                   `json:"daily_limit"`
	MonthlyLimit         *int                   `json:"monthly_limit"`
	SubscriptionType     string                 `json:"subscription_type" validate:"required,oneof=monthly yearly"`
	SubscriptionFee      float64                `json:"subscription_fee" validate:"min=0"`
	ServerRentalFee      float64                `json:"server_rental_fee" validate:"min=0"`
	SetupFee             float64                `json:"setup_fee" validate:"min=0"`
	BillingCycle         string                 `json:"billing_cycle" validate:"required,oneof=monthly yearly"`
	BalanceAlertThreshold *float64              `json:"balance_alert_threshold"`
	ContractStartDate    time.Time              `json:"contract_start_date" validate:"required"`
	ContractEndDate      *time.Time             `json:"contract_end_date"`
	AutoRenew            bool                   `json:"auto_renew"`
	
	// Resource Quotas (optional, will use defaults if not provided)
	ResourceQuotas *ResourceQuotasRequest `json:"resource_quotas"`
}

// ResourceQuotasRequest represents resource quota configuration
type ResourceQuotasRequest struct {
	CPUCores                 *float64 `json:"cpu_cores"`
	MemoryMB                 *int     `json:"memory_mb"`
	BandwidthMbps            *int     `json:"bandwidth_mbps"`
	MaxConcurrentConnections *int     `json:"max_concurrent_connections"`
	StorageGB                *int     `json:"storage_gb"`
	MaxDBConnections         *int     `json:"max_db_connections"`
	MaxQueriesPerSecond      *int     `json:"max_queries_per_second"`
	RedisMemoryMB            *int     `json:"redis_memory_mb"`
	RedisMaxKeys             *int     `json:"redis_max_keys"`
	MaxMessagesPerHour       *int     `json:"max_messages_per_hour"`
	MaxMessagesPerDay        *int     `json:"max_messages_per_day"`
	MaxQueueDepth            *int     `json:"max_queue_depth"`
	APIRateLimitPerMinute    *int     `json:"api_rate_limit_per_minute"`
	APIRateLimitPerHour      *int     `json:"api_rate_limit_per_hour"`
	MaxWorkerThreads         *int     `json:"max_worker_threads"`
	EnforcementMode          string   `json:"enforcement_mode" validate:"oneof=soft hard"`
}

// CreateGateway creates a new dedicated gateway with tenant validation and credential encryption
// Requirements: 8.1, 8.2, 8.3, 8.4, 26.1
func (m *Manager) CreateGateway(ctx context.Context, req *CreateGatewayRequest) (*DedicatedGateway, error) {
	// Validate tenant exists
	if err := m.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Check if gateway code is unique
	var existingGateway DedicatedGateway
	if err := m.db.WithContext(ctx).Where("gateway_code = ?", req.GatewayCode).First(&existingGateway).Error; err == nil {
		return nil, errors.New("gateway code already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check gateway code uniqueness: %w", err)
	}

	// Encrypt credentials
	encryptedPassword, err := m.encrypt(req.TelecomPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	encryptedAPIKey, err := m.encrypt(req.TelecomAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	// Calculate next billing date
	nextBillingDate := m.calculateNextBillingDate(req.ContractStartDate, req.BillingCycle)

	// Create gateway
	gateway := &DedicatedGateway{
		ID:                       uuid.New(),
		TenantID:                 req.TenantID,
		GatewayName:              req.GatewayName,
		GatewayCode:              req.GatewayCode,
		Description:              req.Description,
		TelecomProvider:          req.TelecomProvider,
		TelecomAPIEndpoint:       req.TelecomAPIEndpoint,
		TelecomUsername:          req.TelecomUsername,
		TelecomPasswordEncrypted: encryptedPassword,
		TelecomAPIKeyEncrypted:   encryptedAPIKey,
		ServerType:               req.ServerType,
		ServerIP:                 req.ServerIP,
		ServerHostname:           req.ServerHostname,
		ServerSpecs:              req.ServerSpecs,
		MaxThroughput:            req.MaxThroughput,
		DailyLimit:               req.DailyLimit,
		MonthlyLimit:             req.MonthlyLimit,
		SubscriptionType:         req.SubscriptionType,
		SubscriptionFee:          req.SubscriptionFee,
		ServerRentalFee:          req.ServerRentalFee,
		SetupFee:                 req.SetupFee,
		BillingCycle:             req.BillingCycle,
		NextBillingDate:          &nextBillingDate,
		BalanceAlertThreshold:    req.BalanceAlertThreshold,
		ContractStartDate:        req.ContractStartDate,
		ContractEndDate:          req.ContractEndDate,
		AutoRenew:                req.AutoRenew,
		Status:                   "active",
		HealthStatus:             "unknown",
		BalanceCheckEnabled:      true,
	}

	// Start transaction
	tx := m.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create gateway
	if err := tx.Create(gateway).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create gateway: %w", err)
	}

	// Create default resource quotas
	quotas := m.createDefaultResourceQuotas(gateway.ID, req.ResourceQuotas)
	if err := tx.Create(quotas).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create resource quotas: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return gateway, nil
}

// UpdateGatewayRequest represents the request to update gateway settings
type UpdateGatewayRequest struct {
	GatewayID             uuid.UUID              `json:"gateway_id" validate:"required"`
	TenantID              uuid.UUID              `json:"tenant_id" validate:"required"`
	GatewayName           *string                `json:"gateway_name"`
	Description           *string                `json:"description"`
	TelecomAPIEndpoint    *string                `json:"telecom_api_endpoint"`
	TelecomUsername       *string                `json:"telecom_username"`
	TelecomPassword       *string                `json:"telecom_password"` // Will be encrypted if provided
	TelecomAPIKey         *string                `json:"telecom_api_key"`  // Will be encrypted if provided
	ServerIP              *string                `json:"server_ip"`
	ServerHostname        *string                `json:"server_hostname"`
	ServerSpecs           map[string]interface{} `json:"server_specs"`
	MaxThroughput         *int                   `json:"max_throughput"`
	DailyLimit            *int                   `json:"daily_limit"`
	MonthlyLimit          *int                   `json:"monthly_limit"`
	SubscriptionFee       *float64               `json:"subscription_fee"`
	ServerRentalFee       *float64               `json:"server_rental_fee"`
	BalanceAlertThreshold *float64               `json:"balance_alert_threshold"`
	ContractEndDate       *time.Time             `json:"contract_end_date"`
	AutoRenew             *bool                  `json:"auto_renew"`
}

// UpdateGateway updates gateway settings with tenant validation
// Requirements: 8.1, 8.2, 8.3, 26.1
func (m *Manager) UpdateGateway(ctx context.Context, req *UpdateGatewayRequest) (*DedicatedGateway, error) {
	// Validate tenant
	if err := m.validateTenant(ctx, req.TenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get existing gateway
	var gateway DedicatedGateway
	if err := m.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", req.GatewayID, req.TenantID).First(&gateway).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("gateway not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch gateway: %w", err)
	}

	// Update fields
	updates := make(map[string]interface{})

	if req.GatewayName != nil {
		updates["gateway_name"] = *req.GatewayName
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.TelecomAPIEndpoint != nil {
		updates["telecom_api_endpoint"] = *req.TelecomAPIEndpoint
	}
	if req.TelecomUsername != nil {
		updates["telecom_username"] = *req.TelecomUsername
	}
	if req.TelecomPassword != nil && *req.TelecomPassword != "" {
		encrypted, err := m.encrypt(*req.TelecomPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		updates["telecom_password_encrypted"] = encrypted
	}
	if req.TelecomAPIKey != nil && *req.TelecomAPIKey != "" {
		encrypted, err := m.encrypt(*req.TelecomAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		updates["telecom_api_key_encrypted"] = encrypted
	}
	if req.ServerIP != nil {
		updates["server_ip"] = *req.ServerIP
	}
	if req.ServerHostname != nil {
		updates["server_hostname"] = *req.ServerHostname
	}
	if req.ServerSpecs != nil {
		updates["server_specs"] = req.ServerSpecs
	}
	if req.MaxThroughput != nil {
		updates["max_throughput"] = *req.MaxThroughput
	}
	if req.DailyLimit != nil {
		updates["daily_limit"] = *req.DailyLimit
	}
	if req.MonthlyLimit != nil {
		updates["monthly_limit"] = *req.MonthlyLimit
	}
	if req.SubscriptionFee != nil {
		updates["subscription_fee"] = *req.SubscriptionFee
	}
	if req.ServerRentalFee != nil {
		updates["server_rental_fee"] = *req.ServerRentalFee
	}
	if req.BalanceAlertThreshold != nil {
		updates["balance_alert_threshold"] = *req.BalanceAlertThreshold
	}
	if req.ContractEndDate != nil {
		updates["contract_end_date"] = *req.ContractEndDate
	}
	if req.AutoRenew != nil {
		updates["auto_renew"] = *req.AutoRenew
	}

	// Update gateway
	if err := m.db.WithContext(ctx).Model(&gateway).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update gateway: %w", err)
	}

	// Reload gateway
	if err := m.db.WithContext(ctx).Where("id = ?", gateway.ID).First(&gateway).Error; err != nil {
		return nil, fmt.Errorf("failed to reload gateway: %w", err)
	}

	return &gateway, nil
}

// SuspendGateway suspends a gateway and sends notification
// Requirements: 8.7, 17.1
func (m *Manager) SuspendGateway(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID, reason string) error {
	// Validate tenant
	if err := m.validateTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get gateway
	var gateway DedicatedGateway
	if err := m.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", gatewayID, tenantID).First(&gateway).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("gateway not found or access denied")
		}
		return fmt.Errorf("failed to fetch gateway: %w", err)
	}

	// Update status
	if err := m.db.WithContext(ctx).Model(&gateway).Updates(map[string]interface{}{
		"status": "suspended",
	}).Error; err != nil {
		return fmt.Errorf("failed to suspend gateway: %w", err)
	}

	// TODO: Send notification to enterprise client
	// This will be implemented in the notification service
	// notificationService.SendGatewaySuspendedNotification(ctx, tenantID, gatewayID, reason)

	return nil
}

// TerminateGateway terminates a gateway and sends notification
// Requirements: 8.7, 17.1
func (m *Manager) TerminateGateway(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID, reason string) error {
	// Validate tenant
	if err := m.validateTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get gateway
	var gateway DedicatedGateway
	if err := m.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", gatewayID, tenantID).First(&gateway).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("gateway not found or access denied")
		}
		return fmt.Errorf("failed to fetch gateway: %w", err)
	}

	// Update status
	if err := m.db.WithContext(ctx).Model(&gateway).Updates(map[string]interface{}{
		"status": "terminated",
	}).Error; err != nil {
		return fmt.Errorf("failed to terminate gateway: %w", err)
	}

	// TODO: Send notification to enterprise client
	// This will be implemented in the notification service
	// notificationService.SendGatewayTerminatedNotification(ctx, tenantID, gatewayID, reason)

	return nil
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	IsHealthy         bool      `json:"is_healthy"`
	Status            string    `json:"status"` // 'healthy', 'degraded', 'unhealthy'
	ResponseTime      int64     `json:"response_time_ms"`
	TelecomConnection bool      `json:"telecom_connection"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	CheckedAt         time.Time `json:"checked_at"`
}

// PerformHealthCheck performs a health check on the gateway
// Requirements: 28.5
func (m *Manager) PerformHealthCheck(ctx context.Context, gatewayID uuid.UUID, tenantID uuid.UUID) (*HealthCheckResult, error) {
	// Validate tenant
	if err := m.validateTenant(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("tenant validation failed: %w", err)
	}

	// Get gateway
	var gateway DedicatedGateway
	if err := m.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", gatewayID, tenantID).First(&gateway).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("gateway not found or access denied")
		}
		return nil, fmt.Errorf("failed to fetch gateway: %w", err)
	}

	// Perform health check
	startTime := time.Now()
	result := &HealthCheckResult{
		CheckedAt: startTime,
	}

	// TODO: Implement actual health check logic
	// This should:
	// 1. Check connection to telecom provider API
	// 2. Verify credentials are valid
	// 3. Check response time
	// 4. Determine overall health status
	
	// For now, simulate a basic check
	responseTime := time.Since(startTime).Milliseconds()
	result.ResponseTime = responseTime
	result.TelecomConnection = true // Simulated
	result.IsHealthy = true
	result.Status = "healthy"

	// Update gateway health status
	now := time.Now()
	if err := m.db.WithContext(ctx).Model(&gateway).Updates(map[string]interface{}{
		"last_health_check": now,
		"health_status":     result.Status,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to update health status: %w", err)
	}

	return result, nil
}

// Helper methods

// validateTenant checks if tenant exists
func (m *Manager) validateTenant(ctx context.Context, tenantID uuid.UUID) error {
	var count int64
	if err := m.db.WithContext(ctx).Table("tenants").Where("id = ?", tenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to validate tenant: %w", err)
	}
	if count == 0 {
		return errors.New("tenant not found")
	}
	return nil
}

// encrypt encrypts data using AES-256
func (m *Manager) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts data using AES-256
func (m *Manager) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// calculateNextBillingDate calculates the next billing date based on billing cycle
func (m *Manager) calculateNextBillingDate(startDate time.Time, billingCycle string) time.Time {
	switch billingCycle {
	case "monthly":
		return startDate.AddDate(0, 1, 0)
	case "yearly":
		return startDate.AddDate(1, 0, 0)
	default:
		return startDate.AddDate(0, 1, 0) // Default to monthly
	}
}

// createDefaultResourceQuotas creates default resource quotas for a gateway
func (m *Manager) createDefaultResourceQuotas(gatewayID uuid.UUID, customQuotas *ResourceQuotasRequest) *ResourceQuota {
	// Default quotas for medium gateway
	quotas := &ResourceQuota{
		ID:                       uuid.New(),
		GatewayID:                gatewayID,
		CPUCores:                 floatPtr(4.0),
		MemoryMB:                 intPtr(8192),
		BandwidthMbps:            intPtr(200),
		MaxConcurrentConnections: intPtr(1000),
		StorageGB:                intPtr(100),
		MaxDBConnections:         intPtr(50),
		MaxQueriesPerSecond:      intPtr(200),
		RedisMemoryMB:            intPtr(1024),
		RedisMaxKeys:             intPtr(500000),
		MaxMessagesPerHour:       intPtr(50000),
		MaxMessagesPerDay:        intPtr(500000),
		MaxQueueDepth:            intPtr(5000),
		APIRateLimitPerMinute:    intPtr(300),
		APIRateLimitPerHour:      intPtr(15000),
		MaxWorkerThreads:         intPtr(20),
		EnforcementMode:          "soft",
		IsActive:                 true,
	}

	// Override with custom quotas if provided
	if customQuotas != nil {
		if customQuotas.CPUCores != nil {
			quotas.CPUCores = customQuotas.CPUCores
		}
		if customQuotas.MemoryMB != nil {
			quotas.MemoryMB = customQuotas.MemoryMB
		}
		if customQuotas.BandwidthMbps != nil {
			quotas.BandwidthMbps = customQuotas.BandwidthMbps
		}
		if customQuotas.MaxConcurrentConnections != nil {
			quotas.MaxConcurrentConnections = customQuotas.MaxConcurrentConnections
		}
		if customQuotas.StorageGB != nil {
			quotas.StorageGB = customQuotas.StorageGB
		}
		if customQuotas.MaxDBConnections != nil {
			quotas.MaxDBConnections = customQuotas.MaxDBConnections
		}
		if customQuotas.MaxQueriesPerSecond != nil {
			quotas.MaxQueriesPerSecond = customQuotas.MaxQueriesPerSecond
		}
		if customQuotas.RedisMemoryMB != nil {
			quotas.RedisMemoryMB = customQuotas.RedisMemoryMB
		}
		if customQuotas.RedisMaxKeys != nil {
			quotas.RedisMaxKeys = customQuotas.RedisMaxKeys
		}
		if customQuotas.MaxMessagesPerHour != nil {
			quotas.MaxMessagesPerHour = customQuotas.MaxMessagesPerHour
		}
		if customQuotas.MaxMessagesPerDay != nil {
			quotas.MaxMessagesPerDay = customQuotas.MaxMessagesPerDay
		}
		if customQuotas.MaxQueueDepth != nil {
			quotas.MaxQueueDepth = customQuotas.MaxQueueDepth
		}
		if customQuotas.APIRateLimitPerMinute != nil {
			quotas.APIRateLimitPerMinute = customQuotas.APIRateLimitPerMinute
		}
		if customQuotas.APIRateLimitPerHour != nil {
			quotas.APIRateLimitPerHour = customQuotas.APIRateLimitPerHour
		}
		if customQuotas.MaxWorkerThreads != nil {
			quotas.MaxWorkerThreads = customQuotas.MaxWorkerThreads
		}
		if customQuotas.EnforcementMode != "" {
			quotas.EnforcementMode = customQuotas.EnforcementMode
		}
	}

	return quotas
}

// Helper functions for pointer conversion
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
