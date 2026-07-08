package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookChannel sends alerts via HTTP webhook
type WebhookChannel struct {
	url      string
	headers  map[string]string
	timeout  time.Duration
	minLevel AlertLevel
	enabled  bool
}

// NewWebhookChannel creates a new webhook alert channel
func NewWebhookChannel(url string, headers map[string]string, timeout time.Duration, minLevel AlertLevel) *WebhookChannel {
	return &WebhookChannel{
		url:      url,
		headers:  headers,
		timeout:  timeout,
		minLevel: minLevel,
		enabled:  true,
	}
}

// Name returns the channel name
func (wc *WebhookChannel) Name() string {
	return "WebhookChannel"
}

// IsEnabled returns whether the channel is enabled
func (wc *WebhookChannel) IsEnabled() bool {
	return wc.enabled
}

// SupportsLevel returns whether the channel supports the alert level
func (wc *WebhookChannel) SupportsLevel(level AlertLevel) bool {
	if !wc.enabled {
		return false
	}
	
	levelPriority := map[AlertLevel]int{
		AlertLevelInfo:     1,
		AlertLevelWarning:  2,
		AlertLevelCritical: 3,
	}
	
	return levelPriority[level] >= levelPriority[wc.minLevel]
}

// Send sends an alert via webhook with JSON payload
func (wc *WebhookChannel) Send(alert Alert) error {
	if !wc.enabled {
		return nil
	}

	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	client := &http.Client{
		Timeout: wc.timeout,
	}

	req, err := http.NewRequest("POST", wc.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wc.headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
