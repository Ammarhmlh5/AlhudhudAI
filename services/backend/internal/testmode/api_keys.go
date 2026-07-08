package testmode

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// API Key Prefixes
const (
	LivePublicKeyPrefix  = "pk_live_"
	LiveSecretKeyPrefix  = "sk_live_"
	TestPublicKeyPrefix  = "pk_test_"
	TestSecretKeyPrefix  = "sk_test_"
)

// GenerateTestAPIKey generates a new test API key
func GenerateTestAPIKey(isPublic bool) (string, error) {
	// Generate random bytes
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// Convert to hex string
	keyData := hex.EncodeToString(b)
	
	// Add appropriate prefix
	prefix := TestSecretKeyPrefix
	if isPublic {
		prefix = TestPublicKeyPrefix
	}
	
	return prefix + keyData, nil
}

// GenerateLiveAPIKey generates a new live API key
func GenerateLiveAPIKey(isPublic bool) (string, error) {
	// Generate random bytes
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// Convert to hex string
	keyData := hex.EncodeToString(b)
	
	// Add appropriate prefix
	prefix := LiveSecretKeyPrefix
	if isPublic {
		prefix = LivePublicKeyPrefix
	}
	
	return prefix + keyData, nil
}

// IsTestAPIKey checks if API key is for test mode
func IsTestAPIKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, TestPublicKeyPrefix) || 
	       strings.HasPrefix(apiKey, TestSecretKeyPrefix)
}

// IsLiveAPIKey checks if API key is for live mode
func IsLiveAPIKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, LivePublicKeyPrefix) || 
	       strings.HasPrefix(apiKey, LiveSecretKeyPrefix)
}

// IsPublicKey checks if API key is a public key
func IsPublicKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, LivePublicKeyPrefix) || 
	       strings.HasPrefix(apiKey, TestPublicKeyPrefix)
}

// IsSecretKey checks if API key is a secret key
func IsSecretKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, LiveSecretKeyPrefix) || 
	       strings.HasPrefix(apiKey, TestSecretKeyPrefix)
}

// GetKeyMode returns the mode of the API key (test or live)
func GetKeyMode(apiKey string) string {
	if IsTestAPIKey(apiKey) {
		return "test"
	}
	if IsLiveAPIKey(apiKey) {
		return "live"
	}
	return "unknown"
}

// GetKeyType returns the type of the API key (public or secret)
func GetKeyType(apiKey string) string {
	if IsPublicKey(apiKey) {
		return "public"
	}
	if IsSecretKey(apiKey) {
		return "secret"
	}
	return "unknown"
}

// ValidateAPIKeyFormat validates the format of an API key
func ValidateAPIKeyFormat(apiKey string) error {
	if len(apiKey) < 10 {
		return fmt.Errorf("API key too short")
	}
	
	if !IsTestAPIKey(apiKey) && !IsLiveAPIKey(apiKey) {
		return fmt.Errorf("invalid API key prefix")
	}
	
	// Check if key has valid hex characters after prefix
	var prefix string
	if strings.HasPrefix(apiKey, TestPublicKeyPrefix) {
		prefix = TestPublicKeyPrefix
	} else if strings.HasPrefix(apiKey, TestSecretKeyPrefix) {
		prefix = TestSecretKeyPrefix
	} else if strings.HasPrefix(apiKey, LivePublicKeyPrefix) {
		prefix = LivePublicKeyPrefix
	} else if strings.HasPrefix(apiKey, LiveSecretKeyPrefix) {
		prefix = LiveSecretKeyPrefix
	}
	
	keyData := strings.TrimPrefix(apiKey, prefix)
	if len(keyData) != 64 { // 32 bytes = 64 hex characters
		return fmt.Errorf("invalid API key length")
	}
	
	// Validate hex characters
	_, err := hex.DecodeString(keyData)
	if err != nil {
		return fmt.Errorf("invalid API key format: %w", err)
	}
	
	return nil
}

// MaskAPIKey masks an API key for display (shows only first and last 4 chars)
func MaskAPIKey(apiKey string) string {
	if len(apiKey) < 16 {
		return "****"
	}
	
	// Find prefix length
	var prefixLen int
	if strings.HasPrefix(apiKey, TestPublicKeyPrefix) {
		prefixLen = len(TestPublicKeyPrefix)
	} else if strings.HasPrefix(apiKey, TestSecretKeyPrefix) {
		prefixLen = len(TestSecretKeyPrefix)
	} else if strings.HasPrefix(apiKey, LivePublicKeyPrefix) {
		prefixLen = len(LivePublicKeyPrefix)
	} else if strings.HasPrefix(apiKey, LiveSecretKeyPrefix) {
		prefixLen = len(LiveSecretKeyPrefix)
	}
	
	prefix := apiKey[:prefixLen]
	first4 := apiKey[prefixLen:prefixLen+4]
	last4 := apiKey[len(apiKey)-4:]
	
	return fmt.Sprintf("%s%s****%s", prefix, first4, last4)
}

// APIKeyInfo provides information about an API key
type APIKeyInfo struct {
	Mode        string `json:"mode"`         // test or live
	Type        string `json:"type"`         // public or secret
	Masked      string `json:"masked"`       // masked key for display
	IsValid     bool   `json:"is_valid"`     // whether format is valid
	Description string `json:"description"`  // human-readable description
}

// GetAPIKeyInfo returns information about an API key
func GetAPIKeyInfo(apiKey string) *APIKeyInfo {
	info := &APIKeyInfo{
		Mode:    GetKeyMode(apiKey),
		Type:    GetKeyType(apiKey),
		Masked:  MaskAPIKey(apiKey),
		IsValid: ValidateAPIKeyFormat(apiKey) == nil,
	}
	
	// Generate description
	if info.IsValid {
		info.Description = fmt.Sprintf("%s %s API key", 
			strings.Title(info.Mode), 
			strings.Title(info.Type))
	} else {
		info.Description = "Invalid API key format"
	}
	
	return info
}

// GetAPIKeyDocumentation returns documentation for API keys
func GetAPIKeyDocumentation() string {
	return `
API Keys Documentation
=====================

API Key Types:
--------------

1. Public Keys (pk_):
   - Can be safely used in client-side code
   - Limited permissions
   - Used for read-only operations

2. Secret Keys (sk_):
   - Must be kept secure on server-side
   - Full permissions
   - Used for all operations

API Key Modes:
--------------

1. Test Mode (pk_test_ / sk_test_):
   - No real messages sent
   - No balance deducted
   - Use test phone numbers
   - Operations logged separately
   - Safe for development and testing

2. Live Mode (pk_live_ / sk_live_):
   - Real messages sent
   - Balance deducted
   - Production environment
   - All operations are real

Key Format:
-----------
- Test Public:  pk_test_[64 hex characters]
- Test Secret:  sk_test_[64 hex characters]
- Live Public:  pk_live_[64 hex characters]
- Live Secret:  sk_live_[64 hex characters]

Best Practices:
---------------
1. Never commit secret keys to version control
2. Use environment variables for keys
3. Rotate keys regularly
4. Use test keys for development
5. Use live keys only in production
6. Monitor key usage for suspicious activity
7. Revoke compromised keys immediately

Example Usage:
--------------
// Test mode
curl -H "Authorization: Bearer pk_test_abc123..." \
     https://api.example.com/messages

// Live mode
curl -H "Authorization: Bearer pk_live_xyz789..." \
     https://api.example.com/messages
`
}
