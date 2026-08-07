package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	msgID := os.Args[1]
	seconds, _ := time.ParseDuration(os.Args[2])
	connStr := "postgres://postgres:Ammar1983hmlh%23%23@localhost:5433/al_hudhud_new?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect: %v\n", err)
	}
	defer conn.Close(context.Background())

	last := ""
	start := time.Now()
	fmt.Println("Monitoring message", msgID, "for", seconds)
	for time.Since(start) < seconds {
		var status, deviceID, errorMsg, customID string
		var channelID *string
		var sentAt, deliveredAt *time.Time
		var updatedAt time.Time
		err := conn.QueryRow(context.Background(), `
			SELECT status, COALESCE(device_id::text,''), COALESCE(error_details,''),
			       COALESCE(custom_id,''), COALESCE(channel_id::text,''),
			       sent_at, delivered_at, updated_at
			FROM messages WHERE id = $1`, msgID).Scan(
			&status, &deviceID, &errorMsg, &customID, &channelID, &sentAt, &deliveredAt, &updatedAt)
		if err != nil {
			fmt.Println("error:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		line := fmt.Sprintf("status=%s updated=%s elapsed=%s", status, updatedAt.Format("15:04:05.000"), time.Since(start).Round(time.Millisecond))
		if deviceID != "" {
			line += " device=" + deviceID[:8]
		}
		if channelID != nil && *channelID != "" {
			line += " channel=" + (*channelID)[:8]
		}
		if sentAt != nil {
			line += " sent_at=" + sentAt.Format("15:04:05.000")
		}
		if deliveredAt != nil {
			line += " delivered_at=" + deliveredAt.Format("15:04:05.000")
		}
		if errorMsg != "" {
			line += " error=" + errorMsg
		}
		if line != last {
			fmt.Println("[MSG]", line)
			last = line
		}
		time.Sleep(300 * time.Millisecond)
	}
}
