package gdpr

import "time"

// DeletionStatus represents the status of a data deletion request
type DeletionStatus string

const (
	DeletionStatusPending    DeletionStatus = "pending"
	DeletionStatusProcessing DeletionStatus = "processing"
	DeletionStatusCompleted  DeletionStatus = "completed"
	DeletionStatusFailed     DeletionStatus = "failed"
)

// DeletionRequest represents a data deletion request
type DeletionRequest struct {
	ID          int64          `json:"id" db:"id"`
	TenantID    int64          `json:"tenant_id" db:"tenant_id"`
	UserID      int64          `json:"user_id" db:"user_id"`
	RequestedAt time.Time      `json:"requested_at" db:"requested_at"`
	ProcessedAt *time.Time     `json:"processed_at,omitempty" db:"processed_at"`
	Status      DeletionStatus `json:"status" db:"status"`
	Reason      string         `json:"reason" db:"reason"`
	ErrorMsg    string         `json:"error_msg,omitempty" db:"error_msg"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// DeletionSummary represents a summary of deleted data
type DeletionSummary struct {
	UserID              int64     `json:"user_id"`
	PersonalDataDeleted bool      `json:"personal_data_deleted"`
	RecordsAnonymized   int       `json:"records_anonymized"`
	TablesAffected      []string  `json:"tables_affected"`
	CompletedAt         time.Time `json:"completed_at"`
}
