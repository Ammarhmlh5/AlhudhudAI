package gateway

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

var testTenantID = uuid.New()

func TestValidatePhoneNumber(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		phoneNumber string
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "valid phone number - 10 digits",
			phoneNumber: "+1234567890",
			wantError:   false,
		},
		{
			name:        "valid phone number - 15 digits",
			phoneNumber: "+123456789012345",
			wantError:   false,
		},
		{
			name:        "valid phone number - 12 digits",
			phoneNumber: "+966501234567",
			wantError:   false,
		},
		{
			name:        "empty phone number",
			phoneNumber: "",
			wantError:   true,
			errorMsg:    "phone_number is required",
		},
		{
			name:        "phone number without plus",
			phoneNumber: "1234567890",
			wantError:   true,
			errorMsg:    "E.164 format",
		},
		{
			name:        "phone number too short - 9 digits",
			phoneNumber: "+123456789",
			wantError:   true,
			errorMsg:    "10-15 digits",
		},
		{
			name:        "phone number too long - 16 digits",
			phoneNumber: "+1234567890123456",
			wantError:   true,
			errorMsg:    "10-15 digits",
		},
		{
			name:        "phone number with letters",
			phoneNumber: "+12345abc890",
			wantError:   true,
			errorMsg:    "E.164 format",
		},
		{
			name:        "phone number with spaces",
			phoneNumber: "+123 456 7890",
			wantError:   true,
			errorMsg:    "E.164 format",
		},
		{
			name:        "phone number starting with zero",
			phoneNumber: "+0123456789",
			wantError:   true,
			errorMsg:    "E.164 format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePhoneNumber(tt.phoneNumber)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Message, tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got '%s'", tt.errorMsg, err.Message)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Message)
				}
			}
		})
	}
}

func TestValidateContent(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		content   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid content - short",
			content:   "Hello, World!",
			wantError: false,
		},
		{
			name:      "valid content - exactly 1600 chars",
			content:   strings.Repeat("a", 1600),
			wantError: false,
		},
		{
			name:      "valid content - with Arabic",
			content:   "مرحبا بك في نظام الرسائل",
			wantError: false,
		},
		{
			name:      "empty content",
			content:   "",
			wantError: true,
			errorMsg:  "content is required",
		},
		{
			name:      "whitespace only content",
			content:   "   ",
			wantError: true,
			errorMsg:  "content is required",
		},
		{
			name:      "content too long - 1601 chars",
			content:   strings.Repeat("a", 1601),
			wantError: true,
			errorMsg:  "must not exceed 1600 characters",
		},
		{
			name:      "content too long - 2000 chars",
			content:   strings.Repeat("x", 2000),
			wantError: true,
			errorMsg:  "must not exceed 1600 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.validateContent(tt.content)
			
			if tt.wantError {
				if len(errs) == 0 {
					t.Errorf("expected error but got none")
					return
				}
				found := false
				for _, err := range errs {
					if strings.Contains(err.Message, tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error message to contain '%s', got %v", tt.errorMsg, errs)
				}
			} else {
				if len(errs) > 0 {
					t.Errorf("expected no error but got: %v", errs)
				}
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		priority  string
		wantError bool
	}{
		{
			name:      "valid priority - HIGH",
			priority:  "HIGH",
			wantError: false,
		},
		{
			name:      "valid priority - NORMAL",
			priority:  "NORMAL",
			wantError: false,
		},
		{
			name:      "valid priority - LOW",
			priority:  "LOW",
			wantError: false,
		},
		{
			name:      "invalid priority - lowercase",
			priority:  "high",
			wantError: true,
		},
		{
			name:      "invalid priority - unknown",
			priority:  "URGENT",
			wantError: true,
		},
		{
			name:      "invalid priority - empty",
			priority:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePriority(tt.priority)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Message)
				}
			}
		})
	}
}

func TestValidateMessageRequest(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		request    MessageRequest
		wantErrors int
	}{
		{
			name: "valid request - all fields",
			request: MessageRequest{
				TenantID:    testTenantID,
				PhoneNumber: "+1234567890",
				Content:     "Hello, World!",
				Priority:    "NORMAL",
			},
			wantErrors: 0,
		},
		{
			name: "valid request - no priority",
			request: MessageRequest{
				TenantID:    testTenantID,
				PhoneNumber: "+966501234567",
				Content:     "مرحبا",
			},
			wantErrors: 0,
		},
		{
			name: "invalid request - missing tenant_id",
			request: MessageRequest{
				TenantID:    uuid.Nil,
				PhoneNumber: "+1234567890",
				Content:     "Hello",
			},
			wantErrors: 1,
		},
		{
			name: "invalid request - invalid phone",
			request: MessageRequest{
				TenantID:    testTenantID,
				PhoneNumber: "invalid",
				Content:     "Hello",
			},
			wantErrors: 1,
		},
		{
			name: "invalid request - empty content",
			request: MessageRequest{
				TenantID:    testTenantID,
				PhoneNumber: "+1234567890",
				Content:     "",
			},
			wantErrors: 1,
		},
		{
			name: "invalid request - content too long",
			request: MessageRequest{
				TenantID:    testTenantID,
				PhoneNumber: "+1234567890",
				Content:     strings.Repeat("a", 1601),
			},
			wantErrors: 1,
		},
		{
			name: "invalid request - multiple errors",
			request: MessageRequest{
				TenantID:    uuid.Nil,
				PhoneNumber: "invalid",
				Content:     "",
				Priority:    "INVALID",
			},
			wantErrors: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateMessageRequest(tt.request)
			
			if len(errors) != tt.wantErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrors, len(errors), errors)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		request MessageRequest
		want    bool
	}{
		{
			name: "valid request",
			request: MessageRequest{
				TenantID:    testTenantID,
				PhoneNumber: "+1234567890",
				Content:     "Hello",
			},
			want: true,
		},
		{
			name: "invalid request",
			request: MessageRequest{
				TenantID:    uuid.Nil,
				PhoneNumber: "+1234567890",
				Content:     "Hello",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.IsValid(tt.request)
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateBulkRequest(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name       string
		request    BulkMessageRequest
		wantErrors int
	}{
		{
			name: "valid bulk request",
			request: BulkMessageRequest{
				TenantID: testTenantID,
				Messages: []MessageRequest{
					{
						TenantID:    testTenantID,
						PhoneNumber: "+1234567890",
						Content:     "Message 1",
					},
				},
			},
			wantErrors: 0,
		},
		{
			name: "invalid - missing tenant_id",
			request: BulkMessageRequest{
				TenantID: uuid.Nil,
				Messages: []MessageRequest{
					{
						TenantID:    testTenantID,
						PhoneNumber: "+1234567890",
						Content:     "Message 1",
					},
				},
			},
			wantErrors: 1,
		},
		{
			name: "invalid - empty messages",
			request: BulkMessageRequest{
				TenantID: testTenantID,
				Messages: []MessageRequest{},
			},
			wantErrors: 1,
		},
		{
			name: "invalid - too many messages",
			request: BulkMessageRequest{
				TenantID: testTenantID,
				Messages: make([]MessageRequest, 10001),
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateBulkRequest(tt.request)
			
			if len(errors) != tt.wantErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrors, len(errors), errors)
			}
		})
	}
}
