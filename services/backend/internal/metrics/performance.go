package metrics

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// PerformanceMetrics tracks system performance metrics
type PerformanceMetrics struct {
	mu sync.RWMutex

	// Message metrics
	TotalMessagesSent      int64
	TotalMessagesDelivered int64
	TotalMessagesFailed    int64
	TotalMessagesWaiting   int64

	// Channel metrics
	ChannelUsage map[string]int64 // channel_type -> count

	// Timing metrics
	AverageProcessingTime time.Duration
	AverageDeliveryTime   time.Duration

	// Financial metrics
	TotalRevenue          float64
	TotalRefunds          float64
	PendingTransactions   int64
	CommittedTransactions int64

	// Tenant metrics
	ActiveTenants     int64
	TenantsInWaitMode int64

	// System health
	LastUpdated time.Time
}

var (
	globalMetrics *PerformanceMetrics
	once          sync.Once
)

// GetMetrics returns the global metrics instance
func GetMetrics() *PerformanceMetrics {
	once.Do(func() {
		globalMetrics = &PerformanceMetrics{
			ChannelUsage: make(map[string]int64),
			LastUpdated:  time.Now(),
		}
	})
	return globalMetrics
}

// RecordMessageSent increments the sent message counter
func (m *PerformanceMetrics) RecordMessageSent(channelType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalMessagesSent++
	m.ChannelUsage[channelType]++
	m.LastUpdated = time.Now()
}

// RecordMessageDelivered increments the delivered message counter
func (m *PerformanceMetrics) RecordMessageDelivered(processingTime, deliveryTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalMessagesDelivered++

	// Update average times (simple moving average)
	if m.TotalMessagesDelivered == 1 {
		m.AverageProcessingTime = processingTime
		m.AverageDeliveryTime = deliveryTime
	} else {
		m.AverageProcessingTime = (m.AverageProcessingTime + processingTime) / 2
		m.AverageDeliveryTime = (m.AverageDeliveryTime + deliveryTime) / 2
	}

	m.LastUpdated = time.Now()
}

// RecordMessageFailed increments the failed message counter
func (m *PerformanceMetrics) RecordMessageFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalMessagesFailed++
	m.LastUpdated = time.Now()
}

// RecordMessageWaiting increments the waiting message counter
func (m *PerformanceMetrics) RecordMessageWaiting() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalMessagesWaiting++
	m.LastUpdated = time.Now()
}

// RecordMessageResumed decrements the waiting message counter
func (m *PerformanceMetrics) RecordMessageResumed() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalMessagesWaiting > 0 {
		m.TotalMessagesWaiting--
	}
	m.LastUpdated = time.Now()
}

// RecordTransaction records a financial transaction
func (m *PerformanceMetrics) RecordTransaction(amount float64, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch status {
	case "PENDING":
		m.PendingTransactions++
	case "COMMITTED":
		m.CommittedTransactions++
		m.TotalRevenue += amount
		if m.PendingTransactions > 0 {
			m.PendingTransactions--
		}
	case "REFUNDED":
		m.TotalRefunds += amount
		if m.PendingTransactions > 0 {
			m.PendingTransactions--
		}
	}

	m.LastUpdated = time.Now()
}

// RecordTenantStatus updates tenant status metrics
func (m *PerformanceMetrics) RecordTenantStatus(tenantID uuid.UUID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// This is a simplified version - in production, you'd track individual tenants
	if status == "WAIT_MODE" {
		m.TenantsInWaitMode++
	} else if status == "ACTIVE" {
		if m.TenantsInWaitMode > 0 {
			m.TenantsInWaitMode--
		}
		m.ActiveTenants++
	}

	m.LastUpdated = time.Now()
}

// GetSnapshot returns a read-only snapshot of current metrics
func (m *PerformanceMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channelUsageCopy := make(map[string]int64)
	for k, v := range m.ChannelUsage {
		channelUsageCopy[k] = v
	}

	return MetricsSnapshot{
		TotalMessagesSent:      m.TotalMessagesSent,
		TotalMessagesDelivered: m.TotalMessagesDelivered,
		TotalMessagesFailed:    m.TotalMessagesFailed,
		TotalMessagesWaiting:   m.TotalMessagesWaiting,
		ChannelUsage:           channelUsageCopy,
		AverageProcessingTime:  m.AverageProcessingTime,
		AverageDeliveryTime:    m.AverageDeliveryTime,
		TotalRevenue:           m.TotalRevenue,
		TotalRefunds:           m.TotalRefunds,
		PendingTransactions:    m.PendingTransactions,
		CommittedTransactions:  m.CommittedTransactions,
		ActiveTenants:          m.ActiveTenants,
		TenantsInWaitMode:      m.TenantsInWaitMode,
		LastUpdated:            m.LastUpdated,
	}
}

// MetricsSnapshot is a read-only snapshot of metrics
type MetricsSnapshot struct {
	TotalMessagesSent      int64
	TotalMessagesDelivered int64
	TotalMessagesFailed    int64
	TotalMessagesWaiting   int64
	ChannelUsage           map[string]int64
	AverageProcessingTime  time.Duration
	AverageDeliveryTime    time.Duration
	TotalRevenue           float64
	TotalRefunds           float64
	PendingTransactions    int64
	CommittedTransactions  int64
	ActiveTenants          int64
	TenantsInWaitMode      int64
	LastUpdated            time.Time
}

// SuccessRate calculates the message success rate
func (s MetricsSnapshot) SuccessRate() float64 {
	total := s.TotalMessagesSent
	if total == 0 {
		return 0
	}
	return float64(s.TotalMessagesDelivered) / float64(total) * 100
}

// FailureRate calculates the message failure rate
func (s MetricsSnapshot) FailureRate() float64 {
	total := s.TotalMessagesSent
	if total == 0 {
		return 0
	}
	return float64(s.TotalMessagesFailed) / float64(total) * 100
}

// NetRevenue calculates net revenue (revenue - refunds)
func (s MetricsSnapshot) NetRevenue() float64 {
	return s.TotalRevenue - s.TotalRefunds
}
