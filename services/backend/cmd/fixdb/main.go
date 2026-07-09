package main

import (
	"log"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db.InitDB()

	queries := []string{
		"ALTER TABLE roles ADD CONSTRAINT uni_roles_name UNIQUE (name);",
		"ALTER TABLE permissions ADD CONSTRAINT uni_permissions_name UNIQUE (name);",
		"ALTER TABLE users ADD CONSTRAINT uni_users_email UNIQUE (email);",
		"ALTER TABLE tenants ADD CONSTRAINT uni_tenants_name UNIQUE (name);",
        "ALTER TABLE staff_profiles ADD CONSTRAINT uni_staff_profiles_user_id UNIQUE (user_id);",
	}

	for _, q := range queries {
		_, err := db.DB.Exec(q)
		if err != nil {
			log.Printf("Query %s failed (expected if exists or column changed): %v", q, err)
		} else {
			log.Printf("Successfully ran: %s", q)
		}
	}
}
