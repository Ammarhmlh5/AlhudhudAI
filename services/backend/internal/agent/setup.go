package agent

import (
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ============================================================================
// Setup - ربط نظام الوكلاء بـ main.go
// ============================================================================

// Initialize - تهيئة نظام الوكلاء وإضافة المسارات
func Initialize(router *gin.Engine, db *gorm.DB) {
	// تأكد من وجود الجداول (إضافة/ترقية تلقائية)
	if err := RegisterModels(db); err != nil {
		logger.GetLogger().Warn("Failed to auto-migrate agent tables", zap.Error(err))
	}

	// تهيئة الـ Service والـ Handler
	service := NewService(db)
	handler := NewHandler(service)

	// إضافة المسارات
	SetupRoutes(router, handler)
}

// InitializeWithConfig - تهيئة مع إعدادات مخصصة
func InitializeWithConfig(router *gin.Engine, db *gorm.DB, config *RouterConfig) {
	if err := RegisterModels(db); err != nil {
		logger.GetLogger().Warn("Failed to auto-migrate agent tables", zap.Error(err))
	}

	service := NewService(db)
	handler := NewHandler(service)

	if config != nil {
		SetupRoutesWithConfig(router, handler, config)
	} else {
		SetupRoutes(router, handler)
	}
}

// RegisterModels - تسجيل النماذج في GORM
func RegisterModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Agent{},
		&models.AgentEmployee{},
		&models.AgentTenant{},
		&models.AgentPayment{},
		&models.AgentSession{},
		&models.AgentActivityLog{},
	)
}

// ============================================================================
// Helper to add routes from main.go
// ============================================================================

/*
Usage in main.go:

import (
    "github.com/al-hudhud/backend/internal/agent"
)

func main() {
    // ... existing setup ...
    
    // Add agent routes
    agent.Initialize(router, db)
    
    // ... rest of main.go ...
}

Or with custom config:

func main() {
    // ... existing setup ...
    
    agent.InitializeWithConfig(router, db, &agent.RouterConfig{
        Prefix: "/api/v1/agents",
        EnableRateLimit: true,
        EnableAuditLog: true,
    })
    
    // ... rest of main.go ...
}
*/