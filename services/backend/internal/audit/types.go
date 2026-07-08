package audit

import (
	"time"

	"github.com/google/uuid"
)

// Action types for audit logging
const (
	// Balance Operations
	ActionBalanceDeduct     = "BALANCE_DEDUCT"
	ActionBalanceRefund     = "BALANCE_REFUND"
	ActionBalanceCharge     = "BALANCE_CHARGE"
	ActionBalanceAdjustment = "BALANCE_ADJUSTMENT"

	// Message Operations
	ActionMessageCreate = "MESSAGE_CREATE"
	ActionMessageSend   = "MESSAGE_SEND"
	ActionMessageFail   = "MESSAGE_FAIL"
	ActionMessageCancel = "MESSAGE_CANCEL"

	// Status Operations
	ActionStatusUpdate = "STATUS_UPDATE"
	ActionStatusRefund = "STATUS_REFUND"

	// User Operations
	ActionUserLogin        = "USER_LOGIN"
	ActionUserLogout       = "USER_LOGOUT"
	ActionUserCreate       = "USER_CREATE"
	ActionUserUpdate       = "USER_UPDATE"
	ActionUserDelete       = "USER_DELETE"
	ActionPasswordChange   = "PASSWORD_CHANGE"
	ActionPasswordReset    = "PASSWORD_RESET"
	Action2FAEnable        = "2FA_ENABLE"
	Action2FADisable       = "2FA_DISABLE"

	// Tenant Operations
	ActionTenantCreate = "TENANT_CREATE"
	ActionTenantUpdate = "TENANT_UPDATE"
	ActionTenantDelete = "TENANT_DELETE"

	// Configuration Operations
	ActionConfigUpdate = "CONFIG_UPDATE"
	ActionRateLimit    = "RATE_LIMIT_UPDATE"

	// Channel Operations
	ActionChannelCreate = "CHANNEL_CREATE"
	ActionChannelUpdate = "CHANNEL_UPDATE"
	ActionChannelDelete = "CHANNEL_DELETE"

	// Access Operations
	ActionDataAccess      = "DATA_ACCESS"
	ActionSensitiveAccess = "SENSITIVE_ACCESS"
)

// Resource types
const (
	ResourceBalance  = "BALANCE"
	ResourceMessage  = "MESSAGE"
	ResourceUser     = "USER"
	ResourceTenant   = "TENANT"
	ResourceChannel  = "CHANNEL"
	ResourceConfig   = "CONFIG"
	ResourceSession  = "SESSION"
	ResourceAPI      = "API"
)

// Result types
const (
	ResultSuccess = "SUCCESS"
	ResultFailure = "FAILURE"
	ResultPartial = "PARTIAL"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID          int64                  `json:"id" db:"id"`
	UserID      *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	TenantID    *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	Action      string                 `json:"action" db:"action"`
	Resource    string                 `json:"resource" db:"resource"`
	ResourceID  string                 `json:"resource_id,omitempty" db:"resource_id"`
	IPAddress   string                 `json:"ip_address" db:"ip_address"`
	UserAgent   string                 `json:"user_agent,omitempty" db:"user_agent"`
	Result      string                 `json:"result" db:"result"`
	ErrorMessage string                `json:"error_message,omitempty" db:"error_message"`
	Details     map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// AuditLogger interface defines methods for audit logging
type AuditLogger interface {
	// Log logs an audit entry
	Log(entry AuditEntry) error

	// LogSuccess logs a successful operation
	LogSuccess(action, resource, resourceID string, userID, tenantID *uuid.UUID, ipAddress string, details map[string]interface{}) error

	// LogFailure logs a failed operation
	LogFailure(action, resource, resourceID string, userID, tenantID *uuid.UUID, ipAddress string, errorMsg string, details map[string]interface{}) error

	// Query queries audit logs with filters
	Query(filters AuditQueryFilters) ([]AuditEntry, error)
}

// AuditQueryFilters defines filters for querying audit logs
type AuditQueryFilters struct {
	UserID     *uuid.UUID
	TenantID   *uuid.UUID
	Action     string
	Resource   string
	ResourceID string
	Result     string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

// LegalEvent represents a legal compliance event
// Requirements: 36.10, 36.11
type LegalEvent struct {
	ID         int64                  `json:"id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	TenantID   *uuid.UUID             `json:"tenant_id,omitempty"`
	EventType  string                 `json:"event_type"`
	ResourceID string                 `json:"resource_id,omitempty"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
