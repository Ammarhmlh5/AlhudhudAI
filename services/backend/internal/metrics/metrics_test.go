package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordMessageReceived(t *testing.T) {
	// Reset metrics
	MessagesReceived.Reset()

	// Record a message
	RecordMessageReceived("1", "HIGH")

	// Verify counter increased
	count := testutil.ToFloat64(MessagesReceived.WithLabelValues("1", "HIGH"))
	if count != 1.0 {
		t.Errorf("Expected count 1.0, got %f", count)
	}

	// Record another message
	RecordMessageReceived("1", "HIGH")
	count = testutil.ToFloat64(MessagesReceived.WithLabelValues("1", "HIGH"))
	if count != 2.0 {
		t.Errorf("Expected count 2.0, got %f", count)
	}
}

func TestRecordMessageSent(t *testing.T) {
	// Reset metrics
	MessagesSent.Reset()

	// Record a sent message
	RecordMessageSent("1", "100")

	// Verify counter increased
	count := testutil.ToFloat64(MessagesSent.WithLabelValues("1", "100"))
	if count != 1.0 {
		t.Errorf("Expected count 1.0, got %f", count)
	}
}

func TestRecordMessageFailed(t *testing.T) {
	// Reset metrics
	MessagesFailed.Reset()

	// Record a failed message
	RecordMessageFailed("1", "NETWORK_ERROR")

	// Verify counter increased
	count := testutil.ToFloat64(MessagesFailed.WithLabelValues("1", "NETWORK_ERROR"))
	if count != 1.0 {
		t.Errorf("Expected count 1.0, got %f", count)
	}
}

func TestRecordBalanceOperation(t *testing.T) {
	// Reset metrics
	BalanceOperations.Reset()

	// Record a deduct operation
	RecordBalanceOperation("1", "DEDUCT", "success")

	// Verify counter increased
	count := testutil.ToFloat64(BalanceOperations.WithLabelValues("1", "DEDUCT", "success"))
	if count != 1.0 {
		t.Errorf("Expected count 1.0, got %f", count)
	}

	// Record a refund operation
	RecordBalanceOperation("1", "REFUND", "success")
	count = testutil.ToFloat64(BalanceOperations.WithLabelValues("1", "REFUND", "success"))
	if count != 1.0 {
		t.Errorf("Expected count 1.0, got %f", count)
	}
}

func TestSetQueueSize(t *testing.T) {
	// Set queue size
	SetQueueSize("1", "pending", 100.0)

	// Verify gauge value
	value := testutil.ToFloat64(QueueSize.WithLabelValues("1", "pending"))
	if value != 100.0 {
		t.Errorf("Expected value 100.0, got %f", value)
	}

	// Update queue size
	SetQueueSize("1", "pending", 50.0)
	value = testutil.ToFloat64(QueueSize.WithLabelValues("1", "pending"))
	if value != 50.0 {
		t.Errorf("Expected value 50.0, got %f", value)
	}
}

func TestSetActiveWorkers(t *testing.T) {
	// Set active workers
	SetActiveWorkers(10.0)

	// Verify gauge value
	value := testutil.ToFloat64(ActiveWorkers)
	if value != 10.0 {
		t.Errorf("Expected value 10.0, got %f", value)
	}
}

func TestSetDatabaseConnections(t *testing.T) {
	// Set database connections
	SetDatabaseConnections("idle", 5.0)
	SetDatabaseConnections("in_use", 10.0)

	// Verify gauge values
	idle := testutil.ToFloat64(DatabaseConnections.WithLabelValues("idle"))
	if idle != 5.0 {
		t.Errorf("Expected idle 5.0, got %f", idle)
	}

	inUse := testutil.ToFloat64(DatabaseConnections.WithLabelValues("in_use"))
	if inUse != 10.0 {
		t.Errorf("Expected in_use 10.0, got %f", inUse)
	}
}

func TestObserveMessageLatency(t *testing.T) {
	// Observe latencies
	ObserveMessageLatency("api_gateway", 0.005) // 5ms
	ObserveMessageLatency("api_gateway", 0.010) // 10ms
	ObserveMessageLatency("api_gateway", 0.015) // 15ms

	// Note: testutil.ToFloat64 doesn't work on Histogram observers.
	// This test just ensures the function doesn't panic.
}

func TestObserveBatchSize(t *testing.T) {
	// Observe batch sizes
	ObserveBatchSize(50.0)
	ObserveBatchSize(100.0)

	// Note: We can't easily verify histogram values without accessing internal state
	// This test just ensures the function doesn't panic
}

func TestSetSuccessRate(t *testing.T) {
	// Set success rate
	SetSuccessRate("1", 95.5)

	// Verify gauge value
	value := testutil.ToFloat64(SuccessRate.WithLabelValues("1"))
	if value != 95.5 {
		t.Errorf("Expected value 95.5, got %f", value)
	}
}

func TestSetCacheHitRate(t *testing.T) {
	// Set cache hit rate
	SetCacheHitRate("channels", 98.5)

	// Verify gauge value
	value := testutil.ToFloat64(CacheHitRate.WithLabelValues("channels"))
	if value != 98.5 {
		t.Errorf("Expected value 98.5, got %f", value)
	}
}

func TestRecordCacheHitAndMiss(t *testing.T) {
	// Reset metrics
	CacheHits.Reset()
	CacheMisses.Reset()

	// Record cache hits and misses
	RecordCacheHit("channels")
	RecordCacheHit("channels")
	RecordCacheMiss("channels")

	// Verify counters
	hits := testutil.ToFloat64(CacheHits.WithLabelValues("channels"))
	if hits != 2.0 {
		t.Errorf("Expected hits 2.0, got %f", hits)
	}

	misses := testutil.ToFloat64(CacheMisses.WithLabelValues("channels"))
	if misses != 1.0 {
		t.Errorf("Expected misses 1.0, got %f", misses)
	}
}

func TestSetTenantBalance(t *testing.T) {
	// Set tenant balance
	SetTenantBalance("1", 1000.50)

	// Verify gauge value
	value := testutil.ToFloat64(TenantBalance.WithLabelValues("1"))
	if value != 1000.50 {
		t.Errorf("Expected value 1000.50, got %f", value)
	}
}

func TestMultipleTenants(t *testing.T) {
	// Reset metrics
	MessagesReceived.Reset()
	QueueSize.Reset()

	// Record metrics for multiple tenants
	RecordMessageReceived("1", "NORMAL")
	RecordMessageReceived("2", "HIGH")
	RecordMessageReceived("1", "NORMAL")

	SetQueueSize("1", "pending", 50.0)
	SetQueueSize("2", "pending", 100.0)

	// Verify tenant 1
	count1 := testutil.ToFloat64(MessagesReceived.WithLabelValues("1", "NORMAL"))
	if count1 != 2.0 {
		t.Errorf("Expected tenant 1 count 2.0, got %f", count1)
	}

	queue1 := testutil.ToFloat64(QueueSize.WithLabelValues("1", "pending"))
	if queue1 != 50.0 {
		t.Errorf("Expected tenant 1 queue 50.0, got %f", queue1)
	}

	// Verify tenant 2
	count2 := testutil.ToFloat64(MessagesReceived.WithLabelValues("2", "HIGH"))
	if count2 != 1.0 {
		t.Errorf("Expected tenant 2 count 1.0, got %f", count2)
	}

	queue2 := testutil.ToFloat64(QueueSize.WithLabelValues("2", "pending"))
	if queue2 != 100.0 {
		t.Errorf("Expected tenant 2 queue 100.0, got %f", queue2)
	}
}
