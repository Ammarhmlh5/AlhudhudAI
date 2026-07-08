package gateway

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Validator handles message validation
type Validator struct {
	phoneRegex *regexp.Regexp
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	// E.164 format: +[country code][number] (10-15 digits total)
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{9,14}$`)
	
	return &Validator{
		phoneRegex: phoneRegex,
	}
}

// ValidateMessageRequest validates a message request
// Returns list of validation errors
func (v *Validator) ValidateMessageRequest(req MessageRequest) []ValidationError {
	var errors []ValidationError

	// Validate tenant_id
	if req.TenantID == uuid.Nil {
		errors = append(errors, ValidationError{
			Field:   "tenant_id",
			Message: "tenant_id is required",
		})
	}

	// Validate phone_number
	phoneErr := v.validatePhoneNumber(req.PhoneNumber)
	if phoneErr != nil {
		errors = append(errors, *phoneErr)
	}

	// Validate content
	contentErrs := v.validateContent(req.Content)
	errors = append(errors, contentErrs...)

	// Validate priority (if provided)
	if req.Priority != "" {
		priorityErr := v.validatePriority(req.Priority)
		if priorityErr != nil {
			errors = append(errors, *priorityErr)
		}
	}

	return errors
}

// validatePhoneNumber validates phone number format
// Requirements: 10-15 digits in E.164 format (+[country][number])
func (v *Validator) validatePhoneNumber(phone string) *ValidationError {
	// Check if empty
	if strings.TrimSpace(phone) == "" {
		return &ValidationError{
			Field:   "phone_number",
			Message: "phone_number is required",
		}
	}

	// Check format (E.164)
	if !v.phoneRegex.MatchString(phone) {
		return &ValidationError{
			Field:   "phone_number",
			Message: "phone_number must be in E.164 format (+[country code][number], 10-15 digits total)",
		}
	}

	// Additional validation: check length
	// Remove '+' and count digits
	digits := strings.TrimPrefix(phone, "+")
	digitCount := len(digits)
	
	if digitCount < 10 || digitCount > 15 {
		return &ValidationError{
			Field:   "phone_number",
			Message: fmt.Sprintf("phone_number must have 10-15 digits (found %d)", digitCount),
		}
	}

	return nil
}

// validateContent validates message content
// Requirements: 
// - Must not be empty
// - Must not exceed 1600 characters
func (v *Validator) validateContent(content string) []ValidationError {
	var errors []ValidationError

	// Check if empty
	if strings.TrimSpace(content) == "" {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: "content is required and cannot be empty",
		})
		return errors
	}

	// Check length (1600 characters max)
	charCount := utf8.RuneCountInString(content)
	if charCount > 1600 {
		errors = append(errors, ValidationError{
			Field:   "content",
			Message: fmt.Sprintf("content must not exceed 1600 characters (found %d)", charCount),
		})
	}

	return errors
}

// validatePriority validates priority value
func (v *Validator) validatePriority(priority string) *ValidationError {
	validPriorities := map[string]bool{
		"HIGH":   true,
		"NORMAL": true,
		"LOW":    true,
	}

	if !validPriorities[priority] {
		return &ValidationError{
			Field:   "priority",
			Message: "priority must be one of: HIGH, NORMAL, LOW",
		}
	}

	return nil
}

// IsValid checks if request is valid (no errors)
func (v *Validator) IsValid(req MessageRequest) bool {
	errors := v.ValidateMessageRequest(req)
	return len(errors) == 0
}

// ValidateBulkRequest validates bulk message request
func (v *Validator) ValidateBulkRequest(req BulkMessageRequest) []ValidationError {
	var errors []ValidationError

	// Validate tenant_id
	if req.TenantID == uuid.Nil {
		errors = append(errors, ValidationError{
			Field:   "tenant_id",
			Message: "tenant_id is required",
		})
	}

	// Validate messages array
	if len(req.Messages) == 0 {
		errors = append(errors, ValidationError{
			Field:   "messages",
			Message: "messages array cannot be empty",
		})
	}

	if len(req.Messages) > 10000 {
		errors = append(errors, ValidationError{
			Field:   "messages",
			Message: fmt.Sprintf("messages array cannot exceed 10000 items (found %d)", len(req.Messages)),
		})
	}

	return errors
}
