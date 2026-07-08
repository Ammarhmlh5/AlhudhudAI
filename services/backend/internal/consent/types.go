package consent

import "time"

// ConsentType represents the type of consent
type ConsentType string

const (
	ConsentTermsOfService ConsentType = "terms_of_service"
	ConsentPrivacyPolicy  ConsentType = "privacy_policy"
	ConsentMarketing      ConsentType = "marketing"
	ConsentDataProcessing ConsentType = "data_processing"
)

// Consent represents a user consent record
type Consent struct {
	ID         int64       `json:"id" db:"id"`
	TenantID   int64       `json:"tenant_id" db:"tenant_id"`
	UserID     int64       `json:"user_id" db:"user_id"`
	Type       ConsentType `json:"type" db:"consent_type"`
	Version    string      `json:"version" db:"version"`
	Granted    bool        `json:"granted" db:"granted"`
	GrantedAt  time.Time   `json:"granted_at" db:"granted_at"`
	RevokedAt  *time.Time  `json:"revoked_at,omitempty" db:"revoked_at"`
	IPAddress  string      `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  string      `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
}

// ConsentRequest represents a request to record consent
type ConsentRequest struct {
	UserID    int64       `json:"user_id"`
	Type      ConsentType `json:"type"`
	Version   string      `json:"version"`
	Granted   bool        `json:"granted"`
	IPAddress string      `json:"ip_address"`
	UserAgent string      `json:"user_agent"`
}
