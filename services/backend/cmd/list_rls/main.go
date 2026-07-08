package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}
	defer pool.Close()

	fmt.Println("=== ALL TABLES IN PUBLIC SCHEMA ===")
	rows, _ := pool.Query(ctx, `
		SELECT 
			tablename, 
			rowsecurity 
		FROM pg_tables 
		WHERE schemaname = 'public'
		ORDER BY tablename;
	`)
	for rows.Next() {
		var name string
		var rls bool
		rows.Scan(&name, &rls)
		fmt.Printf("Table: %-30s | RLS: %v\n", name, rls)
	}
	rows.Close()
}
