package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")
	
	if host == "" { host = "localhost" }
	if port == "" { port = "5433" }
	if user == "" { user = "postgres" }
	if password == "" { password = "Ammar1983hmlh##" }
	if dbname == "" { dbname = "al_hudhud_new" }
	if sslmode == "" { sslmode = "disable" }
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Reading migrations/059_link_gateways_to_accounting.sql...")
	sqlBytes, err := os.ReadFile("migrations/059_link_gateways_to_accounting.sql")
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	fmt.Println("Executing migration...")
	_, err = db.Exec(string(sqlBytes))
	if err != nil {
		log.Fatal("Failed to execute migration SQL:", err)
	}

	fmt.Println("[SUCCESS] gateway_id added to accounting_transactions successfully!")
}
