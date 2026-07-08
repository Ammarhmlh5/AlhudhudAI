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

	fmt.Println("--- ACCOUNTING_TRANSACTIONS COLUMNS ---")
	cols, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'accounting_transactions'`)
	if err != nil {
		log.Fatal(err)
	}
	defer cols.Close()
	for cols.Next() {
		var name, dtype string
		cols.Scan(&name, &dtype)
		fmt.Printf("Column: %s (%s)\n", name, dtype)
	}
}
