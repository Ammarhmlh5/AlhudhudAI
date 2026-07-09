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

	fmt.Println("=== ALL TABLES ===")
	tables, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'`)
	if err != nil {
		log.Fatal(err)
	}
	defer tables.Close()
	
	for tables.Next() {
		var tname string
		tables.Scan(&tname)
		fmt.Printf("Table: %s\n", tname)
		
		cols, err := db.Query(fmt.Sprintf(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = '%s'`, tname))
		if err == nil {
			for cols.Next() {
				var cname, dtype string
				cols.Scan(&cname, &dtype)
				fmt.Printf("  - %s (%s)\n", cname, dtype)
			}
			cols.Close()
		}
	}
}
