//go:build ignore

package main

import (
	"log"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load(".env")
	db.InitDB()
	db.InitGormDB()

	email := "test@test.com"
	password := "test1234"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
		UserType:     models.TenantType,
		IsActive:     true,
	}

	if err := db.GormDB.Where("email = ?", email).FirstOrCreate(&user).Error; err != nil {
		log.Fatalf("Failed to create test user: %v", err)
	}

	tenant := models.Tenant{
		UserID:       user.ID,
		BusinessName: "Test Corp",
		ContactName:  "Test User",
		Email:        email,
		IsActive:     true,
	}

	if err := db.GormDB.Where("user_id = ?", user.ID).FirstOrCreate(&tenant).Error; err != nil {
		log.Fatalf("Failed to create test tenant: %v", err)
	}

	log.Printf("Test user created: %s / %s", email, password)
}
