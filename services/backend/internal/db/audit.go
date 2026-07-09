package db

import (
	"encoding/json"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LogActivity records an action in the audit trail using GORM
func LogActivity(userID string, action string, resource string, details interface{}, ipAddress string) {
	log := logger.GetLogger()

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Error("Error marshaling audit details", zap.Error(err))
		detailsJSON = []byte("{}")
	}

	var staffID *uuid.UUID
	if userID != "" {
		parsedUserID, err := uuid.Parse(userID)
		if err == nil {
			var profile models.StaffProfile
			if err := GormDB.Where("user_id = ?", parsedUserID).First(&profile).Error; err == nil {
				staffID = &profile.ID
			} else if !errorsIs(err, gorm.ErrRecordNotFound) {
				log.Error("Error resolving staff_id for audit", zap.Error(err))
			}
		}
	}

	activity := &models.StaffActivity{
		StaffID:   staffID,
		Action:    action,
		Resource:  resource,
		RawData:   string(detailsJSON),
		IPAddress: ipAddress,
	}

	if err := GormDB.Create(activity).Error; err != nil {
		log.Error("Error recording audit log", zap.Error(err))
	}
}

// errorsIs helper to avoid circular imports or direct errors comparison issues
func errorsIs(err, target error) bool {
	return err == target
}
