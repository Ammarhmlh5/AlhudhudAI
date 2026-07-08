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
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var users []models.User
	if err := db.Order("created_at desc").Limit(5).Find(&users).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Println("--- Last 5 Users ---")
	for _, u := range users {
		fmt.Printf("ID: %s | Email: %s | Active: %v | Verified: %v\n", u.ID, u.Email, u.IsActive, u.IsVerified)
	}

	var pending []models.PendingRegistration
	if err := db.Order("created_at desc").Limit(5).Find(&pending).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n--- Last 5 Pending Registrations ---")
	for _, p := range pending {
		fmt.Printf("ID: %s | Email: %s | Step: %d | Verified: %v | Expires: %s\n", p.ID, p.Email, p.Step, p.IsVerified, p.ExpiresAt)
	}

	type VerificationCode struct {
		ID        string
		Target    string
		Code      string
		ExpiresAt string
	}
	var codes []VerificationCode
	// Note: checking table 'verification_codes' assuming standard GORM naming
	if err := db.Table("verification_codes").Order("created_at desc").Limit(5).Find(&codes).Error; err != nil {
		fmt.Println("Error reading verification_codes:", err)
	} else {
		fmt.Println("\n--- Last 5 Verification Codes ---")
		for _, c := range codes {
			fmt.Printf("Target: %s | Code: %s | Expires: %s\n", c.Target, c.Code, c.ExpiresAt)
		}
	}
}
