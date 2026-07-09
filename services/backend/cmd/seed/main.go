package main

import (
	"encoding/json"
	"log"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	// Load .env if it exists
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Note: .env file not found in current directory, using system env")
	}

	// Initialize DB
	db.InitDB()
	db.InitGormDB()

	log.Println("Starting Gateway Seeding for all tenants...")

	var tenants []models.Tenant
	if err := db.GormDB.Find(&tenants).Error; err != nil {
		log.Fatalf("Failed to fetch tenants: %v", err)
	}

	log.Printf("Found %d tenants.", len(tenants))

	for _, tenant := range tenants {
		log.Printf("Processing tenant: %s (%s)", tenant.BusinessName, tenant.ID)

		// Ensure tenant schema is fully migrated before proceeding
		if err := db.MigrateTenantSchema(tenant.ID); err != nil {
			log.Printf("Warning: Failed to migrate tenant schema prior to seed: %v", err)
		}

		err := db.RunInTenantScope(tenant.ID, true, func(tx *gorm.DB) error {
			// Check if they already have a default gateway
			var count int64
			tx.Model(&models.DispatchChannel{}).
				Where("tenant_id = ? AND channel_key = ?", tenant.ID, "hudhud_global").
				Count(&count)

			if count > 0 {
				log.Println(" - Default gateway already exists. Skipping.")
				return nil
			}

			// Create default gateway
			config := map[string]string{
				"endpoint": "https://api.alhudhud.com/v1/sms",
			}
			configJSON, _ := json.Marshal(config)

			gateway := models.DispatchChannel{
				ID:         uuid.New(),
				TenantID:   tenant.ID,
				Type:       models.ChannelGatewayHudhud,
				ChannelKey: "hudhud_global",
				Status:     "ONLINE",
				Priority:   10,
				IsDefault:  true,
				Config:     string(configJSON),
			}

			if err := tx.Create(&gateway).Error; err != nil {
				log.Printf(" - Failed to create gateway: %v", err)
				return err
			}
			
			log.Println(" - Default gateway created successfully.")
			return nil
		})

		if err != nil {
			log.Printf("Tenant processing error: %v", err)
		}
	}

	log.Println("Seeding completed.")
}
