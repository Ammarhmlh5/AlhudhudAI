package gateway

import (
	"time"

	"github.com/google/uuid"
)

// MessageRequest represents incoming message request from client
type MessageRequest struct {
	TenantID    uuid.UUID  `json:"tenant_id" validate:"required"`
	PhoneNumber string     `json:"phone_number" validate:"required"`
	Content     string     `json:"content" validate:"required"`
	Priority    string     `json:"priority,omitempty" validate:"omitempty,oneof=HIGH NORMAL LOW"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

// MessageResponse represents API response after queuing message
type MessageResponse struct {
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"` // QUEUED
	QueuedAt  time.Time `json:"queued_at"`
}

// BulkMessageRequest represents bulk message request
type BulkMessageRequest struct {
	TenantID uuid.UUID        `json:"tenant_id" validate:"required"`
	Messages []MessageRequest `json:"messages" validate:"required,max=10000,dive"`
}

// BulkResponse represents bulk operation response
type BulkResponse struct {
	BatchID      string    `json:"batch_id"`
	Status       string    `json:"status"` // PROCESSING
	TotalCount   int       `json:"total_count"`
	AcceptedCount int      `json:"accepted_count"`
	RejectedCount int      `json:"rejected_count"`
	QueuedAt     time.Time `json:"queued_at"`
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// BufferedMessage represents message in Redis buffer
type BufferedMessage struct {
	MessageID   string    `json:"message_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	PhoneNumber string    `json:"phone_number"`
	Content     string    `json:"content"`
	Priority    string    `json:"priority"`
	QueuedAt    time.Time `json:"queued_at"`
}

// NewBufferedMessage creates a new buffered message from request
func NewBufferedMessage(req MessageRequest) BufferedMessage {
	priority := req.Priority
	if priority == "" {
		priority = "NORMAL"
	}

	return BufferedMessage{
		MessageID:   uuid.New().String(),
		TenantID:    req.TenantID,
		PhoneNumber: req.PhoneNumber,
		Content:     req.Content,
		Priority:    priority,
		QueuedAt:    time.Now(),
	}
}

// DedicatedGateway represents a dedicated gateway for enterprise clients
type DedicatedGateway struct {
	ID                       uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID                 uuid.UUID              `gorm:"type:uuid;not null" json:"tenant_id"`
	GatewayName              string                 `gorm:"size:255;not null" json:"gateway_name"`
	GatewayCode              string                 `gorm:"size:50;uniqueIndex;not null" json:"gateway_code"`
	Description              string                 `gorm:"type:text" json:"description"`
	TelecomProvider          string                 `gorm:"size:100;not null" json:"telecom_provider"`
	TelecomAPIEndpoint       string                 `gorm:"type:text;not null" json:"telecom_api_endpoint"`
	TelecomUsername          string                 `gorm:"size:255" json:"telecom_username"`
	TelecomPasswordEncrypted string                 `gorm:"type:text" json:"-"` // Hidden from JSON
	TelecomAPIKeyEncrypted   string                 `gorm:"type:text" json:"-"` // Hidden from JSON
	ServerType               string                 `gorm:"size:50;not null;default:shared" json:"server_type"`
	ServerIP                 string                 `gorm:"size:45" json:"server_ip"`
	ServerHostname           string                 `gorm:"size:255" json:"server_hostname"`
	ServerSpecs              map[string]interface{} `gorm:"type:jsonb" json:"server_specs"`
	MaxThroughput            int                    `gorm:"default:100" json:"max_throughput"`
	DailyLimit               *int                   `json:"daily_limit"`
	MonthlyLimit             *int                   `json:"monthly_limit"`
	SubscriptionType         string                 `gorm:"size:20;not null;default:monthly" json:"subscription_type"`
	SubscriptionFee          float64                `gorm:"type:decimal(10,2);not null;default:0.00" json:"subscription_fee"`
	ServerRentalFee          float64                `gorm:"type:decimal(10,2);not null;default:0.00" json:"server_rental_fee"`
	SetupFee                 float64                `gorm:"type:decimal(10,2);default:0.00" json:"setup_fee"`
	BillingCycle             string                 `gorm:"size:20;not null;default:monthly" json:"billing_cycle"`
	NextBillingDate          *time.Time             `json:"next_billing_date"`
	TelecomBalance           *float64               `gorm:"type:decimal(10,2)" json:"telecom_balance"`
	BalanceAlertThreshold    *float64               `gorm:"type:decimal(10,2)" json:"balance_alert_threshold"`
	LastBalanceCheck         *time.Time             `json:"last_balance_check"`
	BalanceCheckEnabled      bool                   `gorm:"default:true" json:"balance_check_enabled"`
	ContractStartDate        time.Time              `gorm:"not null" json:"contract_start_date"`
	ContractEndDate          *time.Time             `json:"contract_end_date"`
	AutoRenew                bool                   `gorm:"default:false" json:"auto_renew"`
	Status                   string                 `gorm:"size:20;not null;default:active" json:"status"`
	LastHealthCheck          *time.Time             `json:"last_health_check"`
	HealthStatus             string                 `gorm:"size:20;default:unknown" json:"health_status"`
	CreatedAt                time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt                time.Time              `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for DedicatedGateway
func (DedicatedGateway) TableName() string {
	return "dedicated_gateways"
}

// ResourceQuota represents resource quotas for a dedicated gateway
type ResourceQuota struct {
	ID                       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GatewayID                uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"gateway_id"`
	CPUCores                 *float64  `gorm:"type:decimal(4,2)" json:"cpu_cores"`
	MemoryMB                 *int      `json:"memory_mb"`
	BandwidthMbps            *int      `json:"bandwidth_mbps"`
	MaxConcurrentConnections *int      `json:"max_concurrent_connections"`
	StorageGB                *int      `json:"storage_gb"`
	MaxDBConnections         *int      `json:"max_db_connections"`
	MaxQueriesPerSecond      *int      `json:"max_queries_per_second"`
	RedisMemoryMB            *int      `json:"redis_memory_mb"`
	RedisMaxKeys             *int      `json:"redis_max_keys"`
	MaxMessagesPerHour       *int      `json:"max_messages_per_hour"`
	MaxMessagesPerDay        *int      `json:"max_messages_per_day"`
	MaxQueueDepth            *int      `json:"max_queue_depth"`
	APIRateLimitPerMinute    *int      `json:"api_rate_limit_per_minute"`
	APIRateLimitPerHour      *int      `json:"api_rate_limit_per_hour"`
	MaxWorkerThreads         *int      `json:"max_worker_threads"`
	IsActive                 bool      `gorm:"default:true" json:"is_active"`
	EnforcementMode          string    `gorm:"size:20;default:soft" json:"enforcement_mode"`
	CreatedAt                time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt                time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name for ResourceQuota
func (ResourceQuota) TableName() string {
	return "resource_quotas"
}
