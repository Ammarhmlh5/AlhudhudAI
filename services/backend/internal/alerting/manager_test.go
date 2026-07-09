package alerting

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockChannel is a mock implementation of AlertChannel for testing
type MockChannel struct {
	mu           sync.Mutex
	sentAlerts   []Alert
	shouldFail   bool
	failCount    int
	failAttempts int
}

func NewMockChannel() *MockChannel {
	return &MockChannel{
		sentAlerts: make([]Alert, 0),
	}
}

func (mc *MockChannel) Send(alert Alert) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.shouldFail {
		if mc.failAttempts < mc.failCount {
			mc.failAttempts++
			return errors.New("mock send failure")
		}
	}

	mc.sentAlerts = append(mc.sentAlerts, alert)
	return nil
}

func (mc *MockChannel) GetSentAlerts() []Alert {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return append([]Alert{}, mc.sentAlerts...)
}

func (mc *MockChannel) Name() string {
	return "mock"
}

func (mc *MockChannel) IsEnabled() bool {
	return true
}

func (mc *MockChannel) SupportsLevel(level AlertLevel) bool {
	return true
}

func (mc *MockChannel) SetShouldFail(fail bool, count int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.shouldFail = fail
	mc.failCount = count
	mc.failAttempts = 0
}

func (mc *MockChannel) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.sentAlerts = make([]Alert, 0)
	mc.shouldFail = false
	mc.failCount = 0
	mc.failAttempts = 0
}

func TestNewManager(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.GetChannelCount() != 0 {
		t.Errorf("Expected 0 channels, got %d", manager.GetChannelCount())
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	logger := zap.NewNop()
	config := ManagerConfig{
		MaxHistory: 500,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	}

	manager := NewManagerWithConfig(logger, config)

	if manager == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}

	if manager.maxHistory != 500 {
		t.Errorf("Expected maxHistory 500, got %d", manager.maxHistory)
	}

	if manager.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", manager.maxRetries)
	}

	if manager.retryDelay != 2*time.Second {
		t.Errorf("Expected retryDelay 2s, got %v", manager.retryDelay)
	}
}

func TestAddChannel(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel1 := NewMockChannel()
	channel2 := NewMockChannel()

	manager.AddChannel(channel1)
	if manager.GetChannelCount() != 1 {
		t.Errorf("Expected 1 channel, got %d", manager.GetChannelCount())
	}

	manager.AddChannel(channel2)
	if manager.GetChannelCount() != 2 {
		t.Errorf("Expected 2 channels, got %d", manager.GetChannelCount())
	}
}

func TestSendAlert_Success(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel := NewMockChannel()
	manager.AddChannel(channel)

	metadata := map[string]interface{}{
		"queue_size": 1000,
		"threshold":  500,
	}

	manager.SendAlert(AlertLevelWarning, "Queue Backup", "Queue size exceeded threshold", metadata)

	// Wait for async send
	time.Sleep(100 * time.Millisecond)

	alerts := channel.GetSentAlerts()
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert sent, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.Level != AlertLevelWarning {
		t.Errorf("Expected level WARNING, got %s", alert.Level)
	}
	if alert.Title != "Queue Backup" {
		t.Errorf("Expected title 'Queue Backup', got '%s'", alert.Title)
	}
	if alert.Message != "Queue size exceeded threshold" {
		t.Errorf("Expected specific message, got '%s'", alert.Message)
	}
	if alert.Metadata["queue_size"] != 1000 {
		t.Errorf("Expected queue_size 1000, got %v", alert.Metadata["queue_size"])
	}
}

func TestSendAlert_MultipleChannels(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel1 := NewMockChannel()
	channel2 := NewMockChannel()
	channel3 := NewMockChannel()

	manager.AddChannel(channel1)
	manager.AddChannel(channel2)
	manager.AddChannel(channel3)

	manager.SendAlert(AlertLevelCritical, "System Down", "Critical system failure", nil)

	// Wait for async sends
	time.Sleep(100 * time.Millisecond)

	// All channels should receive the alert
	if len(channel1.GetSentAlerts()) != 1 {
		t.Errorf("Channel 1: expected 1 alert, got %d", len(channel1.GetSentAlerts()))
	}
	if len(channel2.GetSentAlerts()) != 1 {
		t.Errorf("Channel 2: expected 1 alert, got %d", len(channel2.GetSentAlerts()))
	}
	if len(channel3.GetSentAlerts()) != 1 {
		t.Errorf("Channel 3: expected 1 alert, got %d", len(channel3.GetSentAlerts()))
	}
}

func TestSendAlert_NoChannels(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	// Should not panic when no channels configured
	manager.SendAlert(AlertLevelInfo, "Test", "Test message", nil)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// No panic means success
}

func TestSendAlert_WithRetry(t *testing.T) {
	logger := zap.NewNop()
	config := ManagerConfig{
		MaxHistory: 100,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond, // Short delay for testing
	}
	manager := NewManagerWithConfig(logger, config)

	channel := NewMockChannel()
	// Fail first 2 attempts, succeed on 3rd
	channel.SetShouldFail(true, 2)
	manager.AddChannel(channel)

	manager.SendAlert(AlertLevelWarning, "Test Retry", "Testing retry logic", nil)

	// Wait for retries to complete
	time.Sleep(500 * time.Millisecond)

	alerts := channel.GetSentAlerts()
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert sent after retries, got %d", len(alerts))
	}

	// Check history
	history := manager.GetHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 history record, got %d", len(history))
	}

	if !history[0].Success {
		t.Error("Expected success in history")
	}

	if history[0].RetryCount != 2 {
		t.Errorf("Expected 2 retries, got %d", history[0].RetryCount)
	}
}

func TestSendAlert_RetryExhausted(t *testing.T) {
	logger := zap.NewNop()
	config := ManagerConfig{
		MaxHistory: 100,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}
	manager := NewManagerWithConfig(logger, config)

	channel := NewMockChannel()
	// Fail all attempts
	channel.SetShouldFail(true, 10)
	manager.AddChannel(channel)

	manager.SendAlert(AlertLevelCritical, "Test Failure", "Testing failure handling", nil)

	// Wait for all retries
	time.Sleep(500 * time.Millisecond)

	alerts := channel.GetSentAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts sent, got %d", len(alerts))
	}

	// Check history
	history := manager.GetHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 history record, got %d", len(history))
	}

	if history[0].Success {
		t.Error("Expected failure in history")
	}

	if history[0].Error == "" {
		t.Error("Expected error message in history")
	}

	if history[0].RetryCount != 2 {
		t.Errorf("Expected 2 retries, got %d", history[0].RetryCount)
	}
}

func TestSendAlertSync_Success(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel1 := NewMockChannel()
	channel2 := NewMockChannel()
	manager.AddChannel(channel1)
	manager.AddChannel(channel2)

	ctx := context.Background()
	err := manager.SendAlertSync(ctx, AlertLevelInfo, "Sync Test", "Testing sync send", nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(channel1.GetSentAlerts()) != 1 {
		t.Errorf("Channel 1: expected 1 alert, got %d", len(channel1.GetSentAlerts()))
	}
	if len(channel2.GetSentAlerts()) != 1 {
		t.Errorf("Channel 2: expected 1 alert, got %d", len(channel2.GetSentAlerts()))
	}
}

func TestSendAlertSync_NoChannels(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	ctx := context.Background()
	err := manager.SendAlertSync(ctx, AlertLevelInfo, "Test", "Test", nil)

	if err == nil {
		t.Error("Expected error when no channels configured")
	}
}

func TestSendAlertSync_PartialFailure(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel1 := NewMockChannel()
	channel2 := NewMockChannel()
	channel2.SetShouldFail(true, 10) // Always fail

	manager.AddChannel(channel1)
	manager.AddChannel(channel2)

	ctx := context.Background()
	err := manager.SendAlertSync(ctx, AlertLevelWarning, "Partial Fail", "Testing partial failure", nil)

	if err == nil {
		t.Error("Expected error when some channels fail")
	}

	// Channel 1 should succeed
	if len(channel1.GetSentAlerts()) != 1 {
		t.Errorf("Channel 1: expected 1 alert, got %d", len(channel1.GetSentAlerts()))
	}

	// Channel 2 should fail
	if len(channel2.GetSentAlerts()) != 0 {
		t.Errorf("Channel 2: expected 0 alerts, got %d", len(channel2.GetSentAlerts()))
	}
}

func TestAlertHistory(t *testing.T) {
	logger := zap.NewNop()
	config := ManagerConfig{
		MaxHistory: 5,
		MaxRetries: 1,
		RetryDelay: 10 * time.Millisecond,
	}
	manager := NewManagerWithConfig(logger, config)

	channel := NewMockChannel()
	manager.AddChannel(channel)

	// Send multiple alerts
	for i := 0; i < 10; i++ {
		manager.SendAlert(AlertLevelInfo, "Test", "Test message", nil)
	}

	// Wait for all to complete
	time.Sleep(200 * time.Millisecond)

	history := manager.GetHistory()

	// Should only keep last 5 due to maxHistory
	if len(history) != 5 {
		t.Errorf("Expected 5 history records, got %d", len(history))
	}

	// All should be successful
	for i, record := range history {
		if !record.Success {
			t.Errorf("Record %d: expected success", i)
		}
	}
}

func TestGetRecentHistory(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel := NewMockChannel()
	manager.AddChannel(channel)

	// Send 5 alerts
	for i := 0; i < 5; i++ {
		manager.SendAlert(AlertLevelInfo, "Test", "Test message", nil)
	}

	time.Sleep(200 * time.Millisecond)

	// Get recent 3
	recent := manager.GetRecentHistory(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent records, got %d", len(recent))
	}

	// Get more than available
	recent = manager.GetRecentHistory(10)
	if len(recent) != 5 {
		t.Errorf("Expected 5 recent records, got %d", len(recent))
	}

	// Get 0
	recent = manager.GetRecentHistory(0)
	if len(recent) != 0 {
		t.Errorf("Expected 0 recent records, got %d", len(recent))
	}
}

func TestClearHistory(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel := NewMockChannel()
	manager.AddChannel(channel)

	// Send alerts
	manager.SendAlert(AlertLevelInfo, "Test", "Test message", nil)
	time.Sleep(100 * time.Millisecond)

	history := manager.GetHistory()
	if len(history) == 0 {
		t.Fatal("Expected history to have records")
	}

	// Clear history
	manager.ClearHistory()

	history = manager.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected 0 history records after clear, got %d", len(history))
	}
}

func TestConcurrentSends(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel := NewMockChannel()
	manager.AddChannel(channel)

	// Send many alerts concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.SendAlert(AlertLevelInfo, "Concurrent Test", "Testing concurrency", nil)
		}()
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	alerts := channel.GetSentAlerts()
	if len(alerts) != 100 {
		t.Errorf("Expected 100 alerts, got %d", len(alerts))
	}

	history := manager.GetHistory()
	if len(history) != 100 {
		t.Errorf("Expected 100 history records, got %d", len(history))
	}
}

func TestAlertLevels(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	channel := NewMockChannel()
	manager.AddChannel(channel)

	// Test all alert levels
	manager.SendAlert(AlertLevelInfo, "Info Test", "Info message", nil)
	manager.SendAlert(AlertLevelWarning, "Warning Test", "Warning message", nil)
	manager.SendAlert(AlertLevelCritical, "Critical Test", "Critical message", nil)

	time.Sleep(200 * time.Millisecond)

	alerts := channel.GetSentAlerts()
	if len(alerts) != 3 {
		t.Fatalf("Expected 3 alerts, got %d", len(alerts))
	}

	levels := make(map[AlertLevel]bool)
	for _, alert := range alerts {
		levels[alert.Level] = true
	}

	if !levels[AlertLevelInfo] {
		t.Errorf("Expected alert with INFO level to be sent")
	}
	if !levels[AlertLevelWarning] {
		t.Errorf("Expected alert with WARNING level to be sent")
	}
	if !levels[AlertLevelCritical] {
		t.Errorf("Expected alert with CRITICAL level to be sent")
	}
}
