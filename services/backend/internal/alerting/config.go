package alerting

import (
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	adminAlerts "github.com/al-hudhud/backend/internal/admin/alerts"
)

// EmailConfig holds email channel configuration
type EmailConfig struct {
	Enabled  bool
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
	MinLevel AlertLevel
}

// WebhookConfig holds webhook channel configuration
type WebhookConfig struct {
	Enabled  bool
	URL      string
	Headers  map[string]string
	Timeout  time.Duration
	MinLevel AlertLevel
}

// Config holds the complete alerting system configuration
type Config struct {
	Email   EmailConfig
	Webhook WebhookConfig
	Manager ManagerConfig
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() Config {
	return Config{
		Email:   loadEmailConfig(),
		Webhook: loadWebhookConfig(),
		Manager: loadManagerConfig(),
	}
}

func loadEmailConfig() EmailConfig {
	enabled, _ := strconv.ParseBool(getEnv("ALERT_EMAIL_ENABLED", "false"))
	port, _ := strconv.Atoi(getEnv("ALERT_EMAIL_SMTP_PORT", "587"))
	
	toStr := getEnv("ALERT_EMAIL_TO", "")
	var recipients []string
	if toStr != "" {
		recipients = strings.Split(toStr, ",")
		for i := range recipients {
			recipients[i] = strings.TrimSpace(recipients[i])
		}
	}
	
	return EmailConfig{
		Enabled:  enabled,
		SMTPHost: getEnv("ALERT_EMAIL_SMTP_HOST", ""),
		SMTPPort: port,
		Username: getEnv("ALERT_EMAIL_USERNAME", ""),
		Password: getEnv("ALERT_EMAIL_PASSWORD", ""),
		From:     getEnv("ALERT_EMAIL_FROM", ""),
		To:       recipients,
		MinLevel: AlertLevel(getEnv("ALERT_EMAIL_MIN_LEVEL", "INFO")),
	}
}

func loadWebhookConfig() WebhookConfig {
	enabled, _ := strconv.ParseBool(getEnv("ALERT_WEBHOOK_ENABLED", "false"))
	timeout, _ := time.ParseDuration(getEnv("ALERT_WEBHOOK_TIMEOUT", "10s"))
	
	headers := make(map[string]string)
	headersStr := getEnv("ALERT_WEBHOOK_HEADERS", "")
	if headersStr != "" {
		pairs := strings.Split(headersStr, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	
	return WebhookConfig{
		Enabled:  enabled,
		URL:      getEnv("ALERT_WEBHOOK_URL", ""),
		Headers:  headers,
		Timeout:  timeout,
		MinLevel: AlertLevel(getEnv("ALERT_WEBHOOK_MIN_LEVEL", "INFO")),
	}
}

func loadManagerConfig() ManagerConfig {
	maxHistory, _ := strconv.Atoi(getEnv("ALERT_HISTORY_SIZE", "1000"))
	maxRetries, _ := strconv.Atoi(getEnv("ALERT_RETRIES", "3"))
	retryDelay, _ := time.ParseDuration(getEnv("ALERT_RETRY_DELAY", "5s"))
	
	return ManagerConfig{
		MaxHistory: maxHistory,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
	}
}

// Initialize initializes the complete alerting system
func Initialize(logger *zap.Logger, adminManager *adminAlerts.Manager) (*Manager, error) {
	config := LoadConfigFromEnv()
	
	// Create manager
	manager := NewManagerWithConfig(logger, config.Manager)
	
	// Register email channel if enabled
	if config.Email.Enabled {
		if config.Email.SMTPHost == "" || config.Email.From == "" || len(config.Email.To) == 0 {
			logger.Warn("Email alert channel enabled but configuration incomplete")
		} else {
			emailChannel := NewEmailChannel(
				config.Email.SMTPHost,
				config.Email.SMTPPort,
				config.Email.Username,
				config.Email.Password,
				config.Email.From,
				config.Email.To,
				config.Email.MinLevel,
			)
			manager.AddChannel(emailChannel)
			logger.Info("Email alert channel registered",
				zap.String("smtp_host", config.Email.SMTPHost),
				zap.Int("recipients", len(config.Email.To)),
			)
		}
	}
	
	// Register webhook channel if enabled
	if config.Webhook.Enabled {
		if config.Webhook.URL == "" {
			logger.Warn("Webhook alert channel enabled but URL not configured")
		} else {
			webhookChannel := NewWebhookChannel(
				config.Webhook.URL,
				config.Webhook.Headers,
				config.Webhook.Timeout,
				config.Webhook.MinLevel,
			)
			manager.AddChannel(webhookChannel)
			logger.Info("Webhook alert channel registered",
				zap.String("url", config.Webhook.URL),
			)
		}
	}
	
	// Register database channel if admin manager is provided
	if adminManager != nil {
		dbChannel := NewDatabaseChannel(adminManager)
		manager.AddChannel(dbChannel)
		logger.Info("Database alert channel registered (persistent storage enabled)")
	}
	
	logger.Info("New alerting system initialized successfully")
	return manager, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
