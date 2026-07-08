package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("  Create Super Admin User")
	fmt.Println("========================================")
	fmt.Println()

	// Load environment variables
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Get database connection string
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "alhudhudai")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✓ Connected to database successfully")
	fmt.Println()

	// Check if super admin already exists
	var exists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = 'admin@alhudhud.sa')`).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check for existing user: %v", err)
	}

	if exists {
		fmt.Println("⚠ Super Admin user already exists!")
		fmt.Println()
		
		// Show existing user info
		var id, email, userType string
		var isActive bool
		err = db.QueryRow(`
			SELECT id, email, user_type, is_active 
			FROM users 
			WHERE email = 'admin@alhudhud.sa'
		`).Scan(&id, &email, &userType, &isActive)
		
		if err == nil {
			fmt.Println("Existing User Info:")
			fmt.Printf("  ID: %s\n", id)
			fmt.Printf("  Email: %s\n", email)
			fmt.Printf("  Type: %s\n", userType)
			fmt.Printf("  Active: %v\n", isActive)
		}
		
		fmt.Println()
		fmt.Println("If you want to reset the password, delete the user first:")
		fmt.Println("  DELETE FROM users WHERE email = 'admin@alhudhud.sa';")
		fmt.Println()
		return
	}

	// Create super admin user
	email := "admin@alhudhud.sa"
	password := "Admin@123456"
	
	fmt.Println("Creating Super Admin user...")
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	fmt.Println()

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Insert user
	var userID string
	err = db.QueryRow(`
		INSERT INTO users (email, password_hash, user_type, is_active, created_at, updated_at)
		VALUES ($1, $2, 'SUPER_ADMIN', true, NOW(), NOW())
		RETURNING id
	`, email, string(passwordHash)).Scan(&userID)

	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Println("✓ Super Admin user created successfully!")
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  Login Credentials")
	fmt.Println("========================================")
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	fmt.Println()
	fmt.Println("⚠ IMPORTANT: Change this password immediately after first login!")
	fmt.Println()
	fmt.Println("You can now login at:")
	fmt.Println("  Admin Panel: http://localhost:5173/login")
	fmt.Println()
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
