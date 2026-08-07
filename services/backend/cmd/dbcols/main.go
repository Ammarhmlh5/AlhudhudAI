package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://postgres:Ammar1983hmlh%23%23@localhost:5433/al_hudhud_new?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect: %v\n", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("=== LAST 15 DELIVERY LOGS ===")
	rows, err := conn.Query(context.Background(), `
		SELECT message_id, device_id, channel_id, delivery_channel, delivered, delivered_at
		FROM message_delivery_logs ORDER BY delivered_at DESC LIMIT 15`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		found = true
		var msgID, deviceID, channelID, channel string
		var delivered bool
		var deliveredAt *string
		if err := rows.Scan(&msgID, &deviceID, &channelID, &channel, &delivered, &deliveredAt); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("msg=%s device=%s ch=%s type=%s delivered=%v at=%v\n",
			msgID[:8], deviceID, channelID[:8], channel, delivered, ptrs(deliveredAt))
	}
	if !found {
		fmt.Println("(لا توجد سجلات تسليم)")
	}

	fmt.Println("\n=== LAST 10 MESSAGES + DELIVERY FLAG ===")
	rows2, err := conn.Query(context.Background(), `
		SELECT m.id, m.status, m.channel_id, m.device_id,
		       (SELECT count(*) FROM message_delivery_logs dl WHERE dl.message_id = m.id) AS dl_count
		FROM messages m ORDER BY m.created_at DESC LIMIT 10`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, status string
		var channelID, deviceID *string
		var dlCount int64
		if err := rows2.Scan(&id, &status, &channelID, &deviceID, &dlCount); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("msg=%s status=%s channel=%v device=%v delivery_logs=%d\n",
			id[:8], status, nv(channelID), nv(deviceID), dlCount)
	}
}

func ptrs(s *string) string {
	if s == nil { return "NULL" }
	return *s
}

func nv(s *string) string {
	if s == nil { return "NULL" }
	return (*s)[:8]
}
