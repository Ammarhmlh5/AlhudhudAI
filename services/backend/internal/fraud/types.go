package fraud

import "time"

// Severity represents the severity level of a fraud alert
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// AlertStatus represents the status of a fraud alert
type AlertStatus string

const (
	AlertStatusNew           AlertStatus = "new"
	AlertStatusInvestigating AlertStatus = "investigating"
	AlertStatusResolved      AlertStatus = "resolved"
	AlertStatusFalsePositive AlertStatus = "false_positive"
)

// FraudRule represents a fraud detection rule
type FraudRule struct {
	ID          int64    `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Description string   `json:"description" db:"description"`
	RuleType    string   `json:"rule_type" db:"rule_type"`
	Enabled     bool     `json:"enabled" db:"enabled"`
	Severity    Severity `json:"severity" db:"severity"`
	Threshold   float64  `json:"threshold" db:"threshold"`
	TimeWindow  int      `json:"time_window" db:"time_window"` // in minutes
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// FraudAlert represents a fraud alert
type FraudAlert struct {
	ID          int64       `json:"id" db:"id"`
	TenantID    int64       `json:"tenant_id" db:"tenant_id"`
	UserID      int64       `json:"user_id" db:"user_id"`
	RuleID      int64       `json:"rule_id" db:"rule_id"`
	Severity    Severity    `json:"severity" db:"severity"`
	Description string      `json:"description" db:"description"`
	Metadata    string      `json:"metadata" db:"metadata"` // JSON
	DetectedAt  time.Time   `json:"detected_at" db:"detected_at"`
	Status      AlertStatus `json:"status" db:"status"`
	ResolvedAt  *time.Time  `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy  *int64      `json:"resolved_by,omitempty" db:"resolved_by"`
	Notes       string      `json:"notes" db:"notes"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

// Transaction represents a transaction for fraud checking
type Transaction struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	UserID      int64     `json:"user_id"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	IPAddress   string    `json:"ip_address"`
	Country     string    `json:"country"`
	CreatedAt   time.Time `json:"created_at"`
}

// FraudCheckResult represents the result of a fraud check
type FraudCheckResult struct {
	IsFraud     bool          `json:"is_fraud"`
	Severity    Severity      `json:"severity"`
	TriggeredRules []string   `json:"triggered_rules"`
	Score       float64       `json:"score"` // 0-100
	Alerts      []FraudAlert  `json:"alerts"`
}
