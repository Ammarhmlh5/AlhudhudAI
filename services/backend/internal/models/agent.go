package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================================
// Agent - شركة البرمجة (الوكيل)
// ============================================================================

type Agent struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CompanyName  string         `gorm:"type:varchar(255);not null" json:"company_name"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Phone        *string        `gorm:"type:varchar(50)" json:"phone"`
	APIKey       *string        `gorm:"uniqueIndex" json:"api_key"`
	APISecret    *string        `gorm:"type:varchar(255)" json:"-"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Employees []AgentEmployee `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"employees,omitempty"`
	Tenants   []AgentTenant   `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"tenants,omitempty"`
}

func (Agent) TableName() string {
	return "agents"
}

// ============================================================================
// AgentEmployee - موظف الوكيل
// ============================================================================

type AgentEmployee struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AgentID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"agent_id"`
	Agent        Agent          `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"-"`
	Name         string         `gorm:"type:varchar(255);not null" json:"name"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Role         string         `gorm:"type:varchar(20);default:'staff'" json:"role"` // 'admin', 'staff'
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	LastLogin    *time.Time     `json:"last_login"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Payments []AgentPayment `gorm:"foreignKey:EmployeeID" json:"payments,omitempty"`
}

func (AgentEmployee) TableName() string {
	return "agent_employees"
}

// ============================================================================
// AgentTenant - ربط الوكيل بمستأجريه
// ============================================================================

type AgentTenant struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AgentID    uuid.UUID `gorm:"type:uuid;not null;index" json:"agent_id"`
	Agent      Agent     `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"-"`
	TenantID   uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Tenant     Tenant    `gorm:"foreignKey:TenantID" json:"-"`
	CreatedBy  *uuid.UUID `gorm:"type:uuid" json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AgentTenant) TableName() string {
	return "agent_tenants"
}

// ============================================================================
// AgentPayment - سجل المدفوعات
// ============================================================================

type AgentPayment struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	InvoiceID      *uuid.UUID `gorm:"type:uuid;index" json:"invoice_id"`
	EmployeeID     *uuid.UUID `gorm:"type:uuid;index" json:"employee_id"`
	Employee       AgentEmployee `gorm:"foreignKey:EmployeeID" json:"-"`
	AgentID        uuid.UUID   `gorm:"type:uuid;not null;index" json:"agent_id"`
	Agent          Agent        `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"-"`
	TenantID       *uuid.UUID  `gorm:"type:uuid;index" json:"tenant_id"`
	Amount         float64     `gorm:"type:decimal(10,2);not null" json:"amount"`
	Currency       string      `gorm:"type:varchar(3);default:'SAR'" json:"currency"`
	PaymentMethod  *string     `gorm:"type:varchar(50)" json:"payment_method"`
	TransactionID  *string     `gorm:"type:varchar(255)" json:"transaction_id"`
	Status         string      `gorm:"type:varchar(20);default:'pending'" json:"status"` // pending, completed, failed
	Notes          *string     `gorm:"type:text" json:"notes"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func (AgentPayment) TableName() string {
	return "agent_payments"
}

// ============================================================================
// AgentSession - جلسة تسجيل الدخول
// ============================================================================

type AgentSession struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"` // agent_id or employee_id
	UserType      string    `gorm:"type:varchar(20);not null" json:"user_type"` // 'agent' or 'employee'
	Token         string    `gorm:"type:varchar(500);uniqueIndex;not null" json:"-"`
	RefreshToken  *string   `gorm:"type:varchar(500)" json:"-"`
	ExpiresAt     time.Time `gorm:"not null;index" json:"expires_at"`
	IPAddress     *string   `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent     *string   `gorm:"type:text" json:"user_agent"`
	CreatedAt     time.Time `json:"created_at"`
}

func (AgentSession) TableName() string {
	return "agent_sessions"
}

// ============================================================================
// AgentActivityLog - سجل الأنشطة
// ============================================================================

type AgentActivityLog struct {
	ID          uuid.UUID              `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID              `gorm:"type:uuid;not null;index" json:"user_id"`
	UserType    string                 `gorm:"type:varchar(20);not null" json:"user_type"` // 'agent' or 'employee'
	AgentID     *uuid.UUID             `gorm:"type:uuid;index" json:"agent_id"`
	Action      string                 `gorm:"type:varchar(100);not null" json:"action"`
	EntityType  *string                `gorm:"type:varchar(50)" json:"entity_type"` // tenant, payment, employee
	EntityID    *uuid.UUID             `gorm:"type:uuid" json:"entity_id"`
	Details     map[string]interface{} `gorm:"type:jsonb" json:"details"`
	IPAddress   *string                `gorm:"type:varchar(45)" json:"ip_address"`
	CreatedAt   time.Time              `json:"created_at"`
}

func (AgentActivityLog) TableName() string {
	return "agent_activity_log"
}

// ============================================================================
// Request/Response Types
// ============================================================================

// Agent Login Request
type AgentLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Employee Login Request
type AgentEmployeeLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login Response
type AgentLoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         any    `json:"user"` // Agent or AgentEmployee
	UserType     string `json:"user_type"` // 'agent' or 'employee'
	AgentID      int64  `json:"agent_id"`
}

// Change Password Request
type AgentChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// Create Employee Request
type CreateAgentEmployeeRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=admin staff"` // admin or staff
}

// Create Tenant Request
type CreateAgentTenantRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Phone string `json:"phone"`
}

// Payment Request
type AgentPaymentRequest struct {
	InvoiceID     uuid.UUID `json:"invoice_id" binding:"required"`
	Amount        float64   `json:"amount" binding:"required"`
	PaymentMethod string    `json:"payment_method" binding:"required"`
	TransactionID string    `json:"transaction_id"`
	Notes         string    `json:"notes"`
}

// JWT Claims
type AgentJWTClaims struct {
	UserID   int64  `json:"user_id"`
	AgentID  int64  `json:"agent_id"`
	Role     string `json:"role"` // 'agent', 'employee_admin', 'employee_staff'
	UserType string `json:"user_type"` // 'agent' or 'employee'
	Type     string `json:"type"` // 'access' or 'refresh'
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}