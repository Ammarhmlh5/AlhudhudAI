package support

import "time"

// Priority represents ticket priority levels
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// Status represents ticket status
type Status string

const (
	StatusOpen            Status = "open"
	StatusInProgress      Status = "in_progress"
	StatusWaitingCustomer Status = "waiting_customer"
	StatusResolved        Status = "resolved"
	StatusClosed          Status = "closed"
)

// Category represents ticket category
type Category string

const (
	CategoryBilling        Category = "billing"
	CategoryTechnical      Category = "technical"
	CategoryFeatureRequest Category = "feature_request"
	CategoryBugReport      Category = "bug_report"
)

// Ticket represents a support ticket
type Ticket struct {
	ID              int64      `json:"id" db:"id"`
	TenantID        int64      `json:"tenant_id" db:"tenant_id"`
	UserID          int64      `json:"user_id" db:"user_id"`
	TicketNumber    string     `json:"ticket_number" db:"ticket_number"`
	Subject         string     `json:"subject" db:"subject"`
	Description     string     `json:"description" db:"description"`
	Priority        Priority   `json:"priority" db:"priority"`
	Status          Status     `json:"status" db:"status"`
	Category        Category   `json:"category" db:"category"`
	AssignedTo      *int64     `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	FirstResponseAt *time.Time `json:"first_response_at,omitempty" db:"first_response_at"`
	SLADueAt        *time.Time `json:"sla_due_at,omitempty" db:"sla_due_at"`
}

// Comment represents a ticket comment
type Comment struct {
	ID        int64     `json:"id" db:"id"`
	TicketID  int64     `json:"ticket_id" db:"ticket_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	IsStaff   bool      `json:"is_staff" db:"is_staff"`
	Comment   string    `json:"comment" db:"comment"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Attachment represents a ticket attachment
type Attachment struct {
	ID         int64     `json:"id" db:"id"`
	TicketID   int64     `json:"ticket_id" db:"ticket_id"`
	Filename   string    `json:"filename" db:"filename"`
	FilePath   string    `json:"file_path" db:"file_path"`
	FileSize   int64     `json:"file_size" db:"file_size"`
	MimeType   string    `json:"mime_type" db:"mime_type"`
	UploadedBy int64     `json:"uploaded_by" db:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// CreateTicketRequest represents request to create a ticket
type CreateTicketRequest struct {
	Subject     string   `json:"subject" validate:"required,min=5,max=255"`
	Description string   `json:"description" validate:"required,min=10"`
	Priority    Priority `json:"priority" validate:"required,oneof=low medium high urgent"`
	Category    Category `json:"category" validate:"required"`
}

// UpdateTicketRequest represents request to update a ticket
type UpdateTicketRequest struct {
	Status     *Status   `json:"status,omitempty"`
	Priority   *Priority `json:"priority,omitempty"`
	AssignedTo *int64    `json:"assigned_to,omitempty"`
}

// AddCommentRequest represents request to add a comment
type AddCommentRequest struct {
	Comment string `json:"comment" validate:"required,min=1"`
}

// TicketFilters represents filters for listing tickets
type TicketFilters struct {
	Status     *Status   `json:"status,omitempty"`
	Priority   *Priority `json:"priority,omitempty"`
	Category   *Category `json:"category,omitempty"`
	AssignedTo *int64    `json:"assigned_to,omitempty"`
	UserID     *int64    `json:"user_id,omitempty"`
	Limit      int       `json:"limit"`
	Offset     int       `json:"offset"`
}
