package testmode

import "time"

// Message represents a test message
type Message struct {
	ID          string    `json:"id"`
	To          string    `json:"to"`
	From        string    `json:"from"`
	Body        string    `json:"body"`
	TenantID    int64     `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// MessageResult represents the result of sending a test message
type MessageResult struct {
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// TestOperation represents a test mode operation
type TestOperation struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	APIKey      string    `json:"api_key"`
	Operation   string    `json:"operation"`
	TestNumber  string    `json:"test_number"`
	Result      string    `json:"result"`
	RequestData string    `json:"request_data"`
	CreatedAt   time.Time `json:"created_at"`
}

// TestStatus represents possible test message statuses
type TestStatus string

const (
	TestStatusDelivered       TestStatus = "delivered"
	TestStatusFailed          TestStatus = "failed"
	TestStatusPending         TestStatus = "pending"
	TestStatusInvalidNumber   TestStatus = "invalid_number"
	TestStatusInsufficientBalance TestStatus = "insufficient_balance"
	TestStatusRateLimited     TestStatus = "rate_limited"
)
