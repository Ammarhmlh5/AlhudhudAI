package password

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a password with its hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength checks if password meets minimum requirements
func ValidatePasswordStrength(password string) bool {
	// Minimum 8 characters
	if len(password) < 8 {
		return false
	}

	// Could add more complex validation here:
	// - At least one uppercase letter
	// - At least one lowercase letter
	// - At least one number
	// - At least one special character

	return true
}
