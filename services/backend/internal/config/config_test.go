package config

import (
	"testing"
)

func TestValidateJWTSecret(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		shouldErr bool
	}{
		{
			name:      "insecure default value 1",
			secret:    "change-me-in-production",
			shouldErr: true,
		},
		{
			name:      "insecure default value 2",
			secret:    "change-me-in-production-to-a-secure-random-string",
			shouldErr: true,
		},
		{
			name:      "insecure default value 3",
			secret:    "your_secure_jwt_secret_here_minimum_32_characters",
			shouldErr: true,
		},
		{
			name:      "too short - less than 48 chars",
			secret:    "short_secret_only_30_chars_xx",
			shouldErr: true,
		},
		{
			name:      "secure - exactly 48 chars",
			secret:    "7K9mP2xR5vN8qT4wY6zB3jL1hF0dS9gA7mK5nP8rV2xQ4wT6",
			shouldErr: false,
		},
		{
			name:      "secure - more than 48 chars",
			secret:    "7K9mP2xR5vN8qT4wY6zB3jL1hF0dS9gA7mK5nP8rV2xQ4wT6yZ3bJ1hL0fD9sG8a",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error for secret %q, but got none", tt.secret)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error for secret %q, but got: %v", tt.secret, err)
			}
		})
	}
}
