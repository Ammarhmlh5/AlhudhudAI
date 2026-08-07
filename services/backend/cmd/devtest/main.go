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

	fmt.Println("=== PAIRED DEVICES ===")
	rows, err := conn.Query(context.Background(), `
		SELECT pd.id, pd.hardware_id, pd.device_name, pd.phone_number, pd.status,
		       pd.channel_id, pd.tenant_id, t.business_name, t.email
		FROM paired_devices pd
		LEFT JOIN tenants t ON t.id = pd.tenant_id
		ORDER BY pd.created_at DESC`)
	if err != nil {
		log.Fatalf("paired_devices query failed: %v\n", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		found = true
		var id, hardwareID, deviceName, phoneNumber, status string
		var channelID, tenantID, businessName, email *string
		if err := rows.Scan(&id, &hardwareID, &deviceName, &phoneNumber, &status, &channelID, &tenantID, &businessName, &email); err != nil {
			log.Fatal(err)
		}
		ch := "NULL"
		if channelID != nil {
			ch = *channelID
		}
		tid := "NULL"
		if tenantID != nil {
			tid = *tenantID
		}
		bn := "NULL"
		if businessName != nil {
			bn = *businessName
		}
		em := "NULL"
		if email != nil {
			em = *email
		}
		fmt.Printf("device_id=%s\n  hardware_id=%s\n  name=%s phone=%s status=%s\n  channel_id=%s tenant=%s\n  business=%s email=%s\n\n",
			id, hardwareID, deviceName, phoneNumber, status, ch, tid, bn, em)
	}
	if !found {
		fmt.Println("(لا توجد أجهزة مزدوجة مسجلة)")
	}

	fmt.Println("=== DISPATCH CHANNELS ===")
	chrows, err := conn.Query(context.Background(), `
		SELECT id, tenant_id, channel_key, type, status, is_default, priority
		FROM dispatch_channels
		ORDER BY created_at DESC`)
	if err != nil {
		log.Fatalf("dispatch_channels query failed: %v\n", err)
	}
	defer chrows.Close()
	cfound := false
	for chrows.Next() {
		cfound = true
		var id, tenantID, channelKey, ctype, status string
		var isDefault bool
		var priority int
		if err := chrows.Scan(&id, &tenantID, &channelKey, &ctype, &status, &isDefault, &priority); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("channel=%s tenant=%s key=%s type=%s status=%s default=%v priority=%d\n", id, tenantID, channelKey, ctype, status, isDefault, priority)
	}
	if !cfound {
		fmt.Println("(لا توجد قنوات إرسال)")
	}
}
