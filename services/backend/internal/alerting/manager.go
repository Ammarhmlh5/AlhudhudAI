package alerting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// AlertHistory represents a historical alert record
type AlertHistory struct {
	Alert      Alert
	SentAt     time.Time
	Success    bool
	Error      string
	RetryCount int
}

// Manager manages multiple alert channels and sends alerts asynchronously
type Manager struct {
	channels    []AlertChannel
	logger      *zap.Logger
	mu          sync.RWMutex
	history     []AlertHistory
	historyMu   sync.RWMutex
	maxHistory  int
	maxRetries  int
	retryDelay  time.Duration
	wg          sync.WaitGroup
}

// ManagerConfig holds configuration for the alert manager
type ManagerConfig struct {
	MaxHistory int
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		MaxHistory: 1000,
		MaxRetries: 3,
		RetryDelay: 5 * time.Second,
	}
}

// NewManager creates a new alert manager with default configuration
func NewManager(logger *zap.Logger) *Manager {
	return NewManagerWithConfig(logger, DefaultManagerConfig())
}

// NewManagerWithConfig creates a new alert manager with custom configuration
func NewManagerWithConfig(logger *zap.Logger, config ManagerConfig) *Manager {
	return &Manager{
		channels:   make([]AlertChannel, 0),
		logger:     logger,
		history:    make([]AlertHistory, 0),
		maxHistory: config.MaxHistory,
		maxRetries: config.MaxRetries,
		retryDelay: config.RetryDelay,
	}
}

// AddChannel adds an alert channel to the manager
func (m *Manager) AddChannel(channel AlertChannel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels = append(m.channels, channel)
}

// GetChannelCount returns the number of registered channels
func (m *Manager) GetChannelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// SendAlert sends an alert to all configured channels asynchronously
func (m *Manager) SendAlert(level AlertLevel, title, message string, metadata map[string]interface{}) {
	alert := Alert{
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	m.mu.RLock()
	channels := m.channels
	m.mu.RUnlock()

	if len(channels) == 0 {
		m.logger.Warn("No alert channels configured, alert not sent",
			zap.String("title", title),
			zap.String("level", string(level)),
		)
		return
	}

	// Send to all channels asynchronously
	for _, channel := range channels {
		m.wg.Add(1)
		go func(ch AlertChannel, a Alert) {
			defer m.wg.Done()
			m.sendWithRetry(ch, a)
		}(channel, alert)
	}
}

// SendAlertAsync is an alias for SendAlert to match the worker's expectation
func (m *Manager) SendAlertAsync(alert Alert) {
	m.mu.RLock()
	channels := m.channels
	m.mu.RUnlock()

	if len(channels) == 0 {
		m.logger.Warn("No alert channels configured, alert not sent",
			zap.String("title", alert.Title),
			zap.String("level", string(alert.Level)),
		)
		return
	}

	// Send to all channels asynchronously
	for _, channel := range channels {
		m.wg.Add(1)
		go func(ch AlertChannel, a Alert) {
			defer m.wg.Done()
			m.sendWithRetry(ch, a)
		}(channel, alert)
	}
}

// sendWithRetry sends an alert with retry logic
func (m *Manager) sendWithRetry(channel AlertChannel, alert Alert) {
	var lastErr error
	retryCount := 0

	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			delay := m.retryDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(delay)
			retryCount++
			m.logger.Info("Retrying alert send",
				zap.String("title", alert.Title),
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", m.maxRetries),
			)
		}

		err := channel.Send(alert)
		if err == nil {
			// Success
			m.logger.Info("Alert sent successfully",
				zap.String("title", alert.Title),
				zap.String("level", string(alert.Level)),
				zap.Int("retry_count", retryCount),
			)
			m.addToHistory(alert, true, "", retryCount)
			return
		}

		lastErr = err
		m.logger.Warn("Failed to send alert",
			zap.String("title", alert.Title),
			zap.String("level", string(alert.Level)),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	// All retries exhausted
	m.logger.Error("Failed to send alert after all retries",
		zap.String("title", alert.Title),
		zap.String("level", string(alert.Level)),
		zap.Int("retry_count", retryCount),
		zap.Error(lastErr),
	)
	m.addToHistory(alert, false, lastErr.Error(), retryCount)
}

// SendAlertSync sends an alert synchronously and waits for all channels to complete
func (m *Manager) SendAlertSync(ctx context.Context, level AlertLevel, title, message string, metadata map[string]interface{}) error {
	alert := Alert{
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	m.mu.RLock()
	channels := m.channels
	m.mu.RUnlock()

	if len(channels) == 0 {
		return fmt.Errorf("no alert channels configured")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(channels))

	for _, channel := range channels {
		wg.Add(1)
		go func(ch AlertChannel) {
			defer wg.Done()
			if err := ch.Send(alert); err != nil {
				errChan <- err
				m.addToHistory(alert, false, err.Error(), 0)
			} else {
				m.addToHistory(alert, true, "", 0)
			}
		}(channel)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send alert to %d/%d channels", len(errors), len(channels))
	}

	return nil
}

// addToHistory adds an alert to the history
func (m *Manager) addToHistory(alert Alert, success bool, errorMsg string, retryCount int) {
	m.historyMu.Lock()
	defer m.historyMu.Unlock()

	record := AlertHistory{
		Alert:      alert,
		SentAt:     time.Now(),
		Success:    success,
		Error:      errorMsg,
		RetryCount: retryCount,
	}

	m.history = append(m.history, record)

	// Trim history if it exceeds max size
	if len(m.history) > m.maxHistory {
		m.history = m.history[len(m.history)-m.maxHistory:]
	}
}

// GetHistory returns the alert history
func (m *Manager) GetHistory() []AlertHistory {
	m.historyMu.RLock()
	defer m.historyMu.RUnlock()

	// Return a copy to prevent external modification
	historyCopy := make([]AlertHistory, len(m.history))
	copy(historyCopy, m.history)
	return historyCopy
}

// GetRecentHistory returns the most recent N alerts from history
func (m *Manager) GetRecentHistory(n int) []AlertHistory {
	m.historyMu.RLock()
	defer m.historyMu.RUnlock()

	if n > len(m.history) {
		n = len(m.history)
	}

	if n == 0 {
		return []AlertHistory{}
	}

	// Return the last n items
	historyCopy := make([]AlertHistory, n)
	copy(historyCopy, m.history[len(m.history)-n:])
	return historyCopy
}

// ClearHistory clears the alert history
func (m *Manager) ClearHistory() {
	m.historyMu.Lock()
	defer m.historyMu.Unlock()
	m.history = make([]AlertHistory, 0)
}

// Stop waits for all pending alerts to be sent
func (m *Manager) Stop() {
	m.logger.Info("Stopping alert manager, waiting for pending alerts...")
	m.wg.Wait()
	m.logger.Info("Alert manager stopped")
}

