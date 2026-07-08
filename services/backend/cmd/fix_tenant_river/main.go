package main

import (
	"context"
	"fmt"
	"log"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	db.InitDB()
	db.InitGormDB()

	tenantIDStr := "f05ae2fe-af10-4acf-8e98-01104c8cd1c8"
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		log.Fatalf("Invalid tenant ID: %v", err)
	}

	schemaName := db.GetSchemaName(tenantID)
	fmt.Printf("Migrating schema: %s\n", schemaName)

	ctx := context.Background()
	err = queue.MigrateSchema(ctx, schemaName)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration successful!")
}
