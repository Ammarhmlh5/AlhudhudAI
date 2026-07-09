package worker

import (
	"context"
	"time"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/routing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// WorkerPoolConfig represents worker pool configuration
// Implements Requirement 13.1: 50 Workers متزامنة
type WorkerPoolConfig struct {
	// WorkerCount is the number of concurrent workers
	// Default: 50 (as per Requirement 13.1)
	WorkerCount int

	// QueueName is the River queue name
	// Default: "default"
	QueueName string

	// MaxWorkers is the maximum number of workers per queue
	// Default: 50
	MaxWorkers int

	// PollInterval is how often to poll for new jobs
	// Default: 100ms
	PollInterval time.Duration

	// JobTimeout is the maximum time for a job to complete
	// Default: 5 minutes
	JobTimeout time.Duration

	// RetryPolicy defines retry behavior
	RetryPolicy RetryPolicy
}

// RetryPolicy defines retry behavior for failed jobs
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts
	// Default: 3 (as per Requirement 17.4)
	MaxAttempts int

	// InitialInterval is the initial retry delay
	// Default: 1 second
	InitialInterval time.Duration

	// MaxInterval is the maximum retry delay
	// Default: 1 minute
	MaxInterval time.Duration

	// Multiplier is the backoff multiplier
	// Default: 2.0 (exponential backoff)
	Multiplier float64
}

// DefaultWorkerPoolConfig returns default configuration
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		WorkerCount:  50,
		QueueName:    river.QueueDefault,
		MaxWorkers:   50,
		PollInterval: 100 * time.Millisecond,
		JobTimeout:   5 * time.Minute,
		RetryPolicy: RetryPolicy{
			MaxAttempts:     3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     1 * time.Minute,
			Multiplier:      2.0,
		},
	}
}

// WorkerPool represents a pool of River workers
type WorkerPool interface {
	// Start starts the worker pool
	Start(ctx context.Context) error

	// Stop stops the worker pool gracefully
	Stop(ctx context.Context) error

	// GetStatus returns the current status of the worker pool
	GetStatus(ctx context.Context) (*WorkerPoolStatus, error)

	// GetClient returns the River client
	GetClient() *river.Client[pgx.Tx]
}

// WorkerPoolStatus represents the status of the worker pool
type WorkerPoolStatus struct {
	// WorkerCount is the number of active workers
	WorkerCount int

	// QueueSize is the number of pending jobs
	QueueSize int64

	// ProcessedCount is the total number of processed jobs
	ProcessedCount int64

	// FailedCount is the total number of failed jobs
	FailedCount int64

	// AverageProcessingTime is the average job processing time
	AverageProcessingTime time.Duration

	// LastProcessedAt is the timestamp of the last processed job
	LastProcessedAt time.Time

	// IsRunning indicates if the worker pool is running
	IsRunning bool
}

type MessageWorkerDependencies struct {
	// BalanceManager handles balance operations
	BalanceManager balance.Manager

	// RoutingEngine handles channel selection
	RoutingEngine interface {
		SelectChannel(ctx context.Context, tenantID uuid.UUID) (*routing.Channel, error)
		IncrementChannelLoad(ctx context.Context, channelID uuid.UUID, tenantID uuid.UUID) error
	}

	// WebSocketHub handles WebSocket communication
	WebSocketHub interface {
		SendMessage(ctx context.Context, msg WebSocketMessage) error
	}

	// Logger for structured logging
	Logger *zap.Logger
}

// WebSocketMessage represents message sent via WebSocket
type WebSocketMessage struct {
	Type        string
	MessageID   string
	TenantID    uuid.UUID
	ChannelID   uuid.UUID
	PhoneNumber string
	Content     string
	Timestamp   time.Time
}

// WorkerMetrics represents metrics for a single worker
type WorkerMetrics struct {
	// WorkerID is the unique identifier for the worker
	WorkerID int

	// ProcessedCount is the number of messages processed
	ProcessedCount int64

	// FailedCount is the number of failed messages
	FailedCount int64

	// SuccessRate is the success rate (0-1)
	SuccessRate float64

	// AverageTime is the average processing time
	AverageTime time.Duration

	// LastProcessedAt is the timestamp of the last processed message
	LastProcessedAt time.Time
}

// WorkerHealth represents the health status of a worker
// Implements Requirement 13.4: مراقبة صحة Workers
type WorkerHealth struct {
	// WorkerID is the unique identifier for the worker
	WorkerID int

	// IsHealthy indicates if the worker is healthy
	IsHealthy bool

	// LastHealthCheck is the timestamp of the last health check
	LastHealthCheck time.Time

	// ConsecutiveFailures is the number of consecutive failures
	ConsecutiveFailures int

	// LastError is the last error encountered
	LastError string

	// RestartCount is the number of times the worker has been restarted
	RestartCount int
}

// HealthMonitorConfig represents health monitoring configuration
// Implements Requirement 13.4: مراقبة صحة Workers كل 10 ثوانٍ
type HealthMonitorConfig struct {
	// CheckInterval is how often to check worker health
	// Default: 10 seconds (as per Requirement 13.4)
	CheckInterval time.Duration

	// FailureThreshold is the number of consecutive failures before marking unhealthy
	// Default: 3
	FailureThreshold int

	// RestartEnabled indicates if automatic restart is enabled
	// Default: true (as per Requirement 13.5)
	RestartEnabled bool

	// MaxRestarts is the maximum number of restarts per worker
	// Default: 5
	MaxRestarts int
}

// DefaultHealthMonitorConfig returns default health monitoring configuration
func DefaultHealthMonitorConfig() *HealthMonitorConfig {
	return &HealthMonitorConfig{
		CheckInterval:    10 * time.Second,
		FailureThreshold: 3,
		RestartEnabled:   true,
		MaxRestarts:      5,
	}
}
