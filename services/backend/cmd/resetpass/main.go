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
	// Load .env from the backend root
	godotenv.Load(".env")

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	newPassword := "Admin123456"
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash:", err)
	}

	result, err := db.Exec(`UPDATE users SET password_hash = $1 WHERE email = 'admin@alhudhud.sa' AND user_type = 'SUPER_ADMIN'`, string(hash))
	if err != nil {
		log.Fatal("Failed to update:", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		fmt.Println("⚠️  No SUPER_ADMIN found with email admin@alhudhud.sa")
		// List all users
		res, _ := db.Query(`SELECT email, user_type FROM users LIMIT 10`)
		defer res.Close()
		fmt.Println("Existing users:")
		for res.Next() {
			var email, utype string
			res.Scan(&email, &utype)
			fmt.Printf("  - %s (%s)\n", email, utype)
		}
	} else {
		fmt.Printf("✅ Password updated for admin@alhudhud.sa → Admin123456\n")
	}
}
