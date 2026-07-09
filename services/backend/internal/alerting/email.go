package alerting

import (
	"fmt"
	"net/smtp"
	"time"
)

// EmailChannel sends alerts via email using SMTP
type EmailChannel struct {
	smtpHost     string
	smtpPort     int
	smtpUser     string
	smtpPassword string
	from         string
	recipients   []string
	minLevel     AlertLevel
	enabled      bool
}

// NewEmailChannel creates a new email alert channel
func NewEmailChannel(host string, port int, user, password, from string, recipients []string, minLevel AlertLevel) *EmailChannel {
	return &EmailChannel{
		smtpHost:     host,
		smtpPort:     port,
		smtpUser:     user,
		smtpPassword: password,
		from:         from,
		recipients:   recipients,
		minLevel:     minLevel,
		enabled:      true,
	}
}

// Name returns the channel name
func (ec *EmailChannel) Name() string {
	return "EmailChannel"
}

// IsEnabled returns whether the channel is enabled
func (ec *EmailChannel) IsEnabled() bool {
	return ec.enabled
}

// SupportsLevel returns whether the channel supports the alert level
func (ec *EmailChannel) SupportsLevel(level AlertLevel) bool {
	if !ec.enabled {
		return false
	}
	
	levelPriority := map[AlertLevel]int{
		AlertLevelInfo:     1,
		AlertLevelWarning:  2,
		AlertLevelCritical: 3,
	}
	
	return levelPriority[level] >= levelPriority[ec.minLevel]
}

// Send sends an alert via email
func (ec *EmailChannel) Send(alert Alert) error {
	if !ec.enabled {
		return nil
	}
	
	subject := fmt.Sprintf("[%s] %s", alert.Level, alert.Title)
	body := fmt.Sprintf("Time: %s\nLevel: %s\nSource: %s\n\nMessage: %s\n\nMetadata: %v",
		alert.Timestamp.Format(time.RFC3339),
		alert.Level,
		alert.Source,
		alert.Message,
		alert.Metadata,
	)

	message := fmt.Sprintf("From: %s\r\nSubject: %s\r\n\r\n%s", ec.from, subject, body)

	auth := smtp.PlainAuth("", ec.smtpUser, ec.smtpPassword, ec.smtpHost)
	addr := fmt.Sprintf("%s:%d", ec.smtpHost, ec.smtpPort)

	return smtp.SendMail(addr, auth, ec.from, ec.recipients, []byte(message))
}
