package db

import (
	"fmt"
	"log"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedTenantGateways creates default dispatch channels for a tenant
func SeedTenantGateways(tenantID uuid.UUID) error {
	log.Printf("Seeding default gateways for tenant: %s", tenantID)

	return RunInTenantScope(tenantID, true, func(tx *gorm.DB) error {
		// Check if gateways already exist
		var count int64
		if err := tx.Model(&models.DispatchChannel{}).
			Where("tenant_id = ?", tenantID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check existing gateways: %w", err)
		}

		if count > 0 {
			log.Printf("Gateways already exist for tenant %s, skipping", tenantID)
			return nil
		}

		// Create default gateways
		defaultGateways := []models.DispatchChannel{
			{
				TenantID:   tenantID,
				Type:       models.ChannelGatewayHudhud,
				ChannelKey: "hudhud_global",
				Config:     `{"endpoint": "https://api.alhudhud.com/v1/sms", "description": "ALHudhudAI Global Gateway"}`,
				Status:     "ONLINE",
				Priority:   10,
				IsDefault:  true,
			},
		}

		for _, gateway := range defaultGateways {
			if err := tx.Create(&gateway).Error; err != nil {
				return fmt.Errorf("failed to create gateway %s: %w", gateway.ChannelKey, err)
			}
			log.Printf("Created gateway: %s for tenant %s", gateway.ChannelKey, tenantID)
		}

		return nil
	})
}

// SeedAllTenantsGateways seeds gateways for all existing tenants
func SeedAllTenantsGateways() {
	log.Println("Seeding gateways for all tenants...")

	var tenants []models.Tenant
	if err := GormDB.Find(&tenants).Error; err != nil {
		log.Printf("Error fetching tenants: %v", err)
		return
	}

	for _, tenant := range tenants {
		if err := SeedTenantGateways(tenant.ID); err != nil {
			log.Printf("Error seeding gateways for tenant %s: %v", tenant.ID, err)
		}
	}

	log.Println("Finished seeding gateways for all tenants")
}
