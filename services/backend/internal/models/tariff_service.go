package models

import (
	"time"

	"github.com/google/uuid"
)

// ServiceType represents the type of service
type ServiceType string

const (
	ServiceTypeGateway   ServiceType = "gateway"   // Hudud Gateway (pay-as-you-go)
	ServiceTypeMobile    ServiceType = "mobile"    // Mobile Gateway (subscription)
	ServiceTypeDedicated ServiceType = "dedicated" // Dedicated Gateway (enterprise)
)

// ServiceCode represents the unique code for each service
type ServiceCode string

const (
	ServiceCodeHududGateway     ServiceCode = "HUDUD_GATEWAY"
	ServiceCodeMobileGateway    ServiceCode = "MOBILE_GATEWAY"
	ServiceCodeDedicatedGateway ServiceCode = "DEDICATED_GATEWAY"
)

// Service represents a service in the tariff and pricing system
// Supports three service models:
// 1. Hudud Gateway Service (pay-as-you-go)
// 2. Mobile Gateway Service (monthly subscription)
// 3. Dedicated Gateway Service (gateway-as-a-service for enterprises)
type Service struct {
	ID          int         `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID    uuid.UUID   `gorm:"type:uuid;not null;index:idx_services_tenant_id" json:"tenant_id"`
	Name        string      `gorm:"type:varchar(255);not null" json:"name"`
	Code        ServiceCode `gorm:"type:varchar(50);not null" json:"code"`
	Type        ServiceType `gorm:"type:varchar(50);not null" json:"type"`
	Description string      `gorm:"type:text" json:"description"`
	IsActive    bool        `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time   `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	Tenant *Tenant `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"tenant,omitempty"`
}

// TableName specifies the table name for the Service model
func (Service) TableName() string {
	return "services"
}

// IsGatewayService checks if the service is a gateway service (pay-as-you-go)
func (s *Service) IsGatewayService() bool {
	return s.Type == ServiceTypeGateway
}

// IsMobileService checks if the service is a mobile service (subscription)
func (s *Service) IsMobileService() bool {
	return s.Type == ServiceTypeMobile
}

// IsDedicatedService checks if the service is a dedicated service (enterprise)
func (s *Service) IsDedicatedService() bool {
	return s.Type == ServiceTypeDedicated
}

// Validate validates the service data
func (s *Service) Validate() error {
	if s.TenantID == uuid.Nil {
		return ErrInvalidTenantID
	}

	if s.Name == "" {
		return ErrInvalidServiceName
	}

	if s.Code == "" {
		return ErrInvalidServiceCode
	}

	// Validate service type
	switch s.Type {
	case ServiceTypeGateway, ServiceTypeMobile, ServiceTypeDedicated:
		// Valid type
	default:
		return ErrInvalidServiceType
	}

	return nil
}

// Custom errors for service validation
var (
	ErrInvalidTenantID     = &ValidationError{Field: "tenant_id", Message: "tenant_id is required"}
	ErrInvalidServiceName  = &ValidationError{Field: "name", Message: "service name is required"}
	ErrInvalidServiceCode  = &ValidationError{Field: "code", Message: "service code is required"}
	ErrInvalidServiceType  = &ValidationError{Field: "type", Message: "invalid service type"}
	ErrServiceNotFound     = &NotFoundError{Resource: "service"}
	ErrServiceAlreadyExists = &ConflictError{Resource: "service", Message: "service with this code already exists for this tenant"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NotFoundError represents a not found error
type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string {
	return e.Resource + " not found"
}

// ConflictError represents a conflict error
type ConflictError struct {
	Resource string
	Message  string
}

func (e *ConflictError) Error() string {
	return e.Message
}
