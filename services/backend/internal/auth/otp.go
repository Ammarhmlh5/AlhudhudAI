package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"go.uber.org/zap"
)

const (
	OTPExpiry = 10 * time.Minute
)

// GenerateOTP creates a 6-digit numeric OTP
func GenerateOTP() string {
	max := big.NewInt(1000000)
	n, _ := rand.Int(rand.Reader, max)
	return fmt.Sprintf("%06d", n)
}

// SendOTP saves the OTP to the database and simulates sending it
func SendOTP(target, code string) error {
	log := logger.GetLogger()

	// Clean up old codes for this target
	db.GormDB.Where("target = ?", target).Delete(&models.VerificationCode{})

	otpRecord := models.VerificationCode{
		Target:    target,
		Code:      code,
		Type:      "REGISTRATION",
		ExpiresAt: time.Now().Add(OTPExpiry),
	}

	if err := db.GormDB.Create(&otpRecord).Error; err != nil {
		return err
	}

	// Simulation: Log the OTP
	log.Info("OTP Sent (SIMULATED)",
		zap.String("target", target),
		zap.String("code", code),
	)

	// Here you would integrate with an SMS/Email service provider
	return nil
}

// VerifyOTP checks if the provided code is valid and not expired
func VerifyOTP(target, code string) (bool, error) {
	var otpRecord models.VerificationCode
	err := db.GormDB.Where("target = ? AND code = ? AND type = ?", target, code, "REGISTRATION").First(&otpRecord).Error
	if err != nil {
		return false, nil // Not found
	}

	if time.Now().After(otpRecord.ExpiresAt) {
		db.GormDB.Delete(&otpRecord)
		return false, nil // Expired
	}

	// Success: Delete the code after use
	db.GormDB.Delete(&otpRecord)
	return true, nil
}
