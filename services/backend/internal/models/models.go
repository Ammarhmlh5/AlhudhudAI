package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type UserType string

const (
	SuperAdmin   UserType = "SUPER_ADMIN"
	TenantType   UserType = "TENANT"
	ResellerType UserType = "RESELLER"
	StaffType    UserType = "STAFF"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:public.uuid_generate_v4()" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	PhoneNumber  *string        `gorm:"uniqueIndex" json:"phone_number"`
	GoogleID     *string        `gorm:"uniqueIndex" json:"google_id,omitempty"`
	UserType     UserType       `gorm:"not null" json:"user_type"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	IsVerified   bool           `gorm:"default:false" json:"is_verified"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type TenantStatus string

const (
	TenantActive   TenantStatus = "ACTIVE"
	TenantWaitMode TenantStatus = "WAIT_MODE"
)

type Tenant struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey;default:public.uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID    `gorm:"uniqueIndex;type:uuid" json:"user_id"`
	User             User         `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user"`
	BusinessName     string       `gorm:"not null" json:"business_name"`
	ContactName      string       `gorm:"not null" json:"contact_name"`
	Email            string       `gorm:"not null" json:"email"`
	Phone            string       `json:"phone"`
	CountryCode      string       `json:"country_code"`
	Address          string       `json:"address"`
	Sector           string       `json:"sector"`
	Balance          float64      `gorm:"type:decimal(12,2);default:0" json:"balance"`
	Status           TenantStatus `gorm:"default:'ACTIVE'" json:"status"`
	IsActive         bool         `gorm:"default:true" json:"is_active"`
	CompanySerial    string       `gorm:"uniqueIndex" json:"company_serial"`
	APIKeyAccounting string       `gorm:"uniqueIndex" json:"api_key_accounting"`
	APIKeyPhone      string       `gorm:"uniqueIndex" json:"api_key_phone"`
	APIKeyWeb        string       `gorm:"uniqueIndex" json:"api_key_web"`
	IsReseller       bool         `gorm:"default:false" json:"is_reseller"`
	ParentID         *uuid.UUID   `gorm:"type:uuid;index" json:"parent_id"` // Link to Reseller Tenant
	Parent           *Tenant      `gorm:"foreignKey:ParentID" json:"-"`
	AllowGateways    bool         `gorm:"default:false" json:"allow_gateways"` // Control access to Gateway tab
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// CompanyStatus tracks the high-level active/inactive state of a company and its unified channel routing config
type CompanyStatus struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID         uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"tenant_id"`
	IsActive         bool       `gorm:"default:true;not null" json:"is_active"`
	ChannelType      string     `gorm:"type:varchar(50);default:'PHONES';not null" json:"channel_type"` // 'PHONES', 'ALHUDHUD', 'PRIVATE'
	PrivateServerURL string     `gorm:"type:varchar(255)" json:"private_server_url"`
	DeviceID         *uuid.UUID `gorm:"type:uuid" json:"device_id"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type VerificationCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Target    string    `gorm:"index" json:"target"` // Email or Phone
	Code      string    `json:"code"`
	Type      string    `json:"type"` // "REGISTRATION", "PASSWORD_RESET"
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:public.uuid_generate_v4()" json:"id"`
	TenantID    *uuid.UUID   `gorm:"type:uuid;index" json:"tenant_id,omitempty"` // NULL for system roles
	Name        string       `gorm:"uniqueIndex:idx_role_tenant_name;not null" json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:public.uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Resource    string    `gorm:"index;not null" json:"resource"`
	Action      string    `gorm:"not null" json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type StaffProfile struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:public.uuid_generate_v4()" json:"id"`
	TenantID   uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	UserID     uuid.UUID `gorm:"uniqueIndex;type:uuid" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user"`
	FullName   string    `gorm:"not null" json:"full_name"`
	RoleID     uuid.UUID `gorm:"type:uuid" json:"role_id"`
	Role       Role      `gorm:"foreignKey:RoleID" json:"role"`
	Department string    `json:"department"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type MessageStatus string

const (
	MessagePending    MessageStatus = "PENDING"
	MessageQueued     MessageStatus = "QUEUED"
	MessageWait       MessageStatus = "WAITING"
	MessageProcessing MessageStatus = "PROCESSING"
	MessageSent       MessageStatus = "SENT"
	MessageDelivered  MessageStatus = "DELIVERED"
	MessageFailed     MessageStatus = "FAILED"
)

// MessageCategory represents the business type of the message (for cost tracking & filtering)
type MessageCategory string

const (
	MsgCatInvoice  MessageCategory = "INVOICE"   // فاتورة
	MsgCatPayment  MessageCategory = "PAYMENT"   // سداد
	MsgCatAlert    MessageCategory = "ALERT"     // تنبيه
	MsgCatReversal MessageCategory = "REVERSAL"  // عكس تسديد
	MsgCatPromo    MessageCategory = "PROMO"     // إعلان
	MsgCatOTP      MessageCategory = "OTP"       // رمز تحقق
	MsgCatReminder MessageCategory = "REMINDER"  // تذكير
	MsgCatOther    MessageCategory = "OTHER"     // أخرى
)

type Message struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey;default:public.uuid_generate_v4()" json:"id"` // Internal counter for admin stats only
	CustomID          string          `gorm:"type:varchar(23);uniqueIndex:idx_messages_custom_id" json:"custom_id"` // Redis-generated Message ID (MS...)
	EventCustomID     string          `gorm:"type:varchar(20);index" json:"event_custom_id"` // Reference to Event custom_id
	TenantID          uuid.UUID       `gorm:"type:uuid;index" json:"tenant_id"`
	Tenant            Tenant          `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	DeviceID          *uuid.UUID      `gorm:"type:uuid;index" json:"device_id"`            // LEGACY: Use ChannelID
	ChannelID         *uuid.UUID      `gorm:"type:uuid;index" json:"channel_id"`           // The chosen dispatch channel
	IncomingMessageID *uuid.UUID      `gorm:"type:uuid;index" json:"incoming_message_id"`  // Link to raw ingestion
	GlobalEventID     *int64          `gorm:"index" json:"global_event_id"`               // Phase 1: Raw event reference
	EventID           *uuid.UUID      `gorm:"type:uuid;index" json:"event_id"`            // Phase 4: Structured MessageEvent
	Source            string          `gorm:"index" json:"source"`                        // Origin: WEB_UI, WEB_API, ACCOUNTING, MOBILE

	// Sender & Context
	SenderName     string          `json:"sender_name"`
	SenderUserID   *uuid.UUID      `gorm:"type:uuid;index" json:"sender_user_id"`
	SenderDeviceID *uuid.UUID      `gorm:"type:uuid;index" json:"sender_device_id"`

	// Message Content
	PhoneNumber string `gorm:"not null" json:"phone_number"`
	Content     string `gorm:"type:text;not null" json:"content"`

	// Phase 2: Message Type & Pricing
	MessageType MessageCategory `gorm:"type:message_category;default:'OTHER'" json:"message_type"`
	CharCount   int             `gorm:"default:0" json:"char_count"`   // Character count
	PartsCount  int             `gorm:"default:1" json:"parts_count"`  // Segments to deduct
	Encoding    string          `gorm:"default:'GSM7'" json:"encoding"` // GSM7 or UNICODE
	Cost        float64         `gorm:"type:decimal(12,4);default:0" json:"cost"` // Cost = PartsCount * price_per_part

	// Status & Routing
	Status         MessageStatus `gorm:"default:'PENDING';index" json:"status"`
	RetryCount     int           `gorm:"default:0" json:"retry_count"`
	IdempotencyKey string        `gorm:"index" json:"idempotency_key"`
	ProviderJobID  string        `json:"provider_job_id"` // External ID from gateway
	WaitReason     string        `json:"wait_reason"`
	ErrorDetails   string        `json:"error_details"`   // Error code/reason from gateway DLR

	// Timestamps
	ScheduledAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PairedDevice struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID      uuid.UUID  `gorm:"type:uuid;index" json:"tenant_id"`
	Tenant        Tenant     `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	ChannelID     *uuid.UUID `gorm:"type:uuid;index" json:"channel_id"` // Link to the DispatchChannel
	DeviceName    string     `json:"device_name"`
	HardwareID    string     `gorm:"uniqueIndex;not null" json:"hardware_id"` // Globally unique phone identifier
	DevicePairKey string     `gorm:"not null" json:"device_pair_key"`
	PhoneNumber   string     `json:"phone_number"`
	Status        string     `gorm:"default:'OFFLINE'" json:"status"`
	LastSeen      *time.Time `json:"last_seen"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ChannelType string

const (
	ChannelPhone          ChannelType = "PHONE"
	ChannelGatewayHudhud  ChannelType = "GATEWAY_HUDHUD"
	ChannelGatewayPrivate ChannelType = "GATEWAY_PRIVATE"
)

type DispatchChannel struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID    uuid.UUID   `gorm:"type:uuid;uniqueIndex:idx_tenant_channel_key;index:idx_channel_routing,priority:1" json:"tenant_id"`
	Type        ChannelType `gorm:"not null" json:"type"`
	ChannelKey  string      `gorm:"uniqueIndex:idx_tenant_channel_key;not null" json:"channel_key"`      // Key for API and routing
	Config      string      `gorm:"type:jsonb" json:"config"`                                            // API keys, Device ID, etc.
	GatewayID   *uuid.UUID  `gorm:"type:uuid;index" json:"gateway_id"`                                   // Link to dedicated gateway
	CurrentLoad int         `gorm:"default:0" json:"current_load"`                                       // Current number of messages being processed
	Status      string      `gorm:"default:'ONLINE';index:idx_channel_routing,priority:2" json:"status"` // ONLINE, OFFLINE, WAIT_MODE
	Priority    int         `gorm:"default:0;index:idx_channel_routing,priority:3" json:"priority"`      // Higher is more preferred
	IsDefault   bool        `gorm:"default:false" json:"is_default"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// AlhudhudGateway stores configurations for the official Alhudhud gateways (like Bawabaty)
type AlhudhudGateway struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ProviderName string    `gorm:"not null;default:'bawabaty'" json:"provider_name"` // e.g., 'bawabaty'
	BaseURL      string    `gorm:"not null" json:"base_url"`
	APIKey       string    `gorm:"not null" json:"api_key"`
	SenderID     string    `gorm:"not null" json:"sender_id"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TransactionStatus string

const (
	TxPending   TransactionStatus = "PENDING"
	TxCommitted TransactionStatus = "COMMITTED"
	TxRefunded  TransactionStatus = "REFUNDED"
)

type AccountingTransaction struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID  uuid.UUID         `gorm:"type:uuid;index" json:"tenant_id"`
	MessageID uuid.UUID         `gorm:"type:uuid;index" json:"message_id"`
	Amount    float64           `gorm:"type:decimal(12,2)" json:"amount"`
	Status    TransactionStatus `gorm:"default:'PENDING'" json:"status"`
	Type      string            `gorm:"default:'DEDUCTION'" json:"type"` // DEDUCTION, REFUND
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID       uuid.UUID `gorm:"type:uuid;index"`
	User         User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Token        string    `gorm:"uniqueIndex;not null"`
	RefreshToken string    `gorm:"uniqueIndex"`
	IPAddress    string
	UserAgent    string
	ExpiresAt    time.Time `gorm:"index"`
	CreatedAt    time.Time
}

type StaffActivity struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID  uuid.UUID      `gorm:"type:uuid;index" json:"tenant_id"`
	StaffID   *uuid.UUID     `gorm:"type:uuid;index" json:"staff_id"`
	Staff     *StaffProfile  `gorm:"foreignKey:StaffID;constraint:OnDelete:CASCADE" json:"staff,omitempty"`
	Action    string         `gorm:"not null" json:"action"`
	Resource  string         `json:"resource"`
	RawData   string         `gorm:"type:jsonb" json:"details"`
	IPAddress string         `json:"ip_address"`
	CreatedAt time.Time      `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token        string        `json:"token"`
	RefreshToken string        `json:"refresh_token"`
	User         User          `json:"user"`
	Tenant       *Tenant       `json:"tenant,omitempty"`
	Staff        *StaffProfile `json:"staff,omitempty"`
	Permissions  []string      `json:"permissions"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID        uuid.UUID       `gorm:"type:uuid;index" json:"tenant_id"`
	InvoiceNumber   string          `gorm:"uniqueIndex;not null" json:"invoice_number"`
	SubscriptionID  *int64          `json:"subscription_id,omitempty"`
	TransactionID   *int64          `json:"transaction_id,omitempty"`
	Subtotal        decimal.Decimal `gorm:"type:decimal(12,4)" json:"subtotal"`
	TaxRate         decimal.Decimal `gorm:"type:decimal(12,4)" json:"tax_rate"`
	TaxAmount       decimal.Decimal `gorm:"type:decimal(12,4)" json:"tax_amount"`
	TotalAmount     decimal.Decimal `gorm:"type:decimal(12,4)" json:"total_amount"`
	Currency        string          `gorm:"default:'USD'" json:"currency"`
	Description     string          `json:"description"`
	ServiceName     string          `json:"service_name"`
	IssueDate       time.Time       `json:"issue_date"`
	DueDate         *time.Time      `json:"due_date,omitempty"`
	PaidDate        *time.Time      `json:"paid_date,omitempty"`
	Status          string          `gorm:"default:'UNPAID'" json:"status"`
	PDFGenerated    bool            `gorm:"default:false" json:"pdf_generated"`
	PDFPath         string          `json:"pdf_path,omitempty"`
	EmailSent       bool            `gorm:"default:false" json:"email_sent"`
	EmailSentAt     *time.Time      `json:"email_sent_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (Invoice) TableName() string {
	return "billing_invoices"
}

// MessageDeliveryLog tracks message delivery to different channels (mobile, SMS, etc.)
type MessageDeliveryLog struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MessageID       uuid.UUID `gorm:"type:uuid;index" json:"message_id"`
	TenantID        uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	DeviceID        uuid.UUID `gorm:"type:uuid;index" json:"device_id"`
	ChannelID       uuid.UUID `gorm:"type:uuid;index" json:"channel_id"`
	DeliveryChannel string    `gorm:"type:varchar(20);not null" json:"delivery_channel"` // MOBILE, SMS, etc.
	Delivered       bool      `gorm:"default:false" json:"delivered"`
	DeliveredAt     time.Time `json:"delivered_at"`
	ErrorMessage    string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func (MessageDeliveryLog) TableName() string {
	return "message_delivery_logs"
}
// NotificationStats represents notification statistics for a tenant
type NotificationStats struct {
	Total      int64 `json:"total"`
	Unread     int64 `json:"unread"`
	Messages   int64 `json:"messages"`
	Deliveries int64 `json:"deliveries"`
	Statuses   int64 `json:"statuses"`
	Systems    int64 `json:"systems"`
	Alerts     int64 `json:"alerts"`
}