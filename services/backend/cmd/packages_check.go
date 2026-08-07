package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	pool, err := pgxpool.New(context.Background(), "postgres://postgres@localhost:5432/sms_gateway_test?sslmode=disable")
	if err != nil { panic(err) }
	defer pool.Close()

	rows, err := pool.Query(context.Background(),
		`SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns
		 WHERE table_name='packages' ORDER BY ordinal_position`)
	if err != nil { panic(err) }
	defer rows.Close()
	fmt.Println("packages columns:")
	for rows.Next() {
		var c, t, n, d string
		rows.Scan(&c, &t, &n, &d)
		fmt.Printf("  %-24s %-16s null=%s def=%s\n", c, t, n, d)
	}
}
