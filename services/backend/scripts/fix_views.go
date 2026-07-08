//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s timezone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	log.Println("1. Dropping dependent views and conflicting constraints...")
	gdb.Exec(`DROP VIEW IF EXISTS v_messages_with_custom_ids CASCADE;`)
	gdb.Exec(`DROP VIEW IF EXISTS v_event_load_analytics CASCADE;`)
	// Drop the explicit UNIQUE constraint created via manual SQL because GORM handles uniques with indexes (uniqueIndex)
	gdb.Exec(`ALTER TABLE global_incoming_events DROP CONSTRAINT IF EXISTS global_incoming_events_custom_id_key;`)
	gdb.Exec(`ALTER TABLE global_incoming_events DROP CONSTRAINT IF EXISTS uni_global_incoming_events_custom_id;`)

	log.Println("2. Running exact AutoMigrate to align schema...")
	if err := gdb.AutoMigrate(&models.GlobalIncomingEvent{}); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	log.Println("3. Recreating views...")
	err = gdb.Exec(`
CREATE OR REPLACE VIEW v_messages_with_custom_ids AS
SELECT 
    m.custom_id as message_id,
    m.event_custom_id as event_id,
    m.tenant_id,
    m.phone_number,
    m.content,
    m.status,
    m.cost,
    m.created_at,
    m.updated_at,
    ge.custom_id as event_custom_id_full,
    ge.sender_ip as event_sender_ip,
    ge.created_at as event_created_at
FROM messages m
LEFT JOIN global_incoming_events ge ON m.event_custom_id = ge.custom_id
WHERE m.custom_id IS NOT NULL;

CREATE OR REPLACE VIEW v_event_load_analytics AS
SELECT 
    extract_timestamp_from_custom_id(custom_id) as event_hour,
    extract_tenant_hash_from_custom_id(custom_id) as tenant_hash,
    COUNT(*) as event_count,
    SUM(total_items) as total_messages,
    AVG(total_items) as avg_messages_per_event,
    MAX(total_items) as max_messages_per_event,
    COUNT(CASE WHEN total_items >= 900 THEN 1 END) as near_limit_events
FROM global_incoming_events
WHERE custom_id IS NOT NULL
GROUP BY event_hour, tenant_hash
ORDER BY event_hour DESC;
	`).Error

	if err != nil {
		log.Fatalf("Failed to recreate views: %v", err)
	}

	log.Println("Success! Schema is now perfectly aligned with GORM.")
}
