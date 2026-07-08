package db

import (
	"crypto/rand"
	"database/sql"
	"log"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func generateSecurePassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[num.Int64()]
	}
	return string(result)
}

// SeedDatabase populates the database with default data
func SeedDatabase() {
	log.Println("Starting database seeding...")

	// Seed Permissions
	seedPermissions()

	// Seed Roles
	seedRoles()

	// Seed Super Admin
	seedSuperAdmin()

	// Seed Test Tenant
	seedTestTenant()

	// Seed Staff Users
	seedStaff()

	log.Println("Database seeding completed successfully")
}

func seedPermissions() {
	permissions := []struct {
		name        string
		resource    string
		action      string
		description string
	}{
		// Tenant permissions
		{"view_tenants", "tenants", "view", "View tenant list and details"},
		{"create_tenant", "tenants", "create", "Create new tenants"},
		{"edit_tenant", "tenants", "edit", "Edit tenant information"},
		{"delete_tenant", "tenants", "delete", "Delete tenants"},
		{"manage_tenant_balance", "tenants", "manage_balance", "Add or deduct tenant balance"},

		// Finance permissions
		{"view_finances", "finance", "view", "View financial reports and transactions"},
		{"approve_payments", "finance", "approve", "Approve payment requests"},
		{"manage_payment_methods", "finance", "manage", "Manage payment methods and gateways"},

		// Support permissions
		{"view_tickets", "support", "view", "View support tickets"},
		{"respond_tickets", "support", "respond", "Respond to support tickets"},
		{"close_tickets", "support", "close", "Close support tickets"},

		// Gateway permissions
		{"view_gateways", "gateways", "view", "View SMS gateways"},
		{"manage_gateways", "gateways", "manage", "Manage SMS gateway configurations"},

		// Staff permissions
		{"view_staff", "staff", "view", "View staff members"},
		{"create_staff", "staff", "create", "Create new staff members"},
		{"edit_staff", "staff", "edit", "Edit staff information"},
		{"delete_staff", "staff", "delete", "Delete staff members"},
		{"manage_roles", "staff", "manage_roles", "Manage roles and permissions"},

		// Device permissions
		{"view_devices", "devices", "view", "View paired devices"},
		{"manage_devices", "devices", "manage", "Manage device pairings"},
	}

	for _, p := range permissions {
		_, err := DB.Exec(`
			INSERT INTO permissions (name, resource, action, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING
		`, p.name, p.resource, p.action, p.description)

		if err != nil {
			log.Printf("Error seeding permission %s: %v", p.name, err)
		}
	}

	log.Println("Permissions seeded successfully")
}

func seedRoles() {
	roles := []struct {
		name        string
		description string
		permissions []string
	}{
		{
			name:        "Support Agent",
			description: "Customer support staff with ticket management access",
			permissions: []string{
				"view_tenants",
				"view_tickets",
				"respond_tickets",
				"close_tickets",
			},
		},
		{
			name:        "Accountant",
			description: "Financial staff with payment and balance management access",
			permissions: []string{
				"view_tenants",
				"view_finances",
				"approve_payments",
				"manage_tenant_balance",
			},
		},
		{
			name:        "Technical Admin",
			description: "Technical staff with gateway and device management access",
			permissions: []string{
				"view_tenants",
				"view_gateways",
				"manage_gateways",
				"view_devices",
				"manage_devices",
			},
		},
		{
			name:        "Staff Manager",
			description: "HR staff with staff management access",
			permissions: []string{
				"view_staff",
				"create_staff",
				"edit_staff",
				"manage_roles",
			},
		},
		{
			name:        "Full Admin",
			description: "Staff with all permissions except super admin privileges",
			permissions: []string{
				"view_tenants", "create_tenant", "edit_tenant", "manage_tenant_balance",
				"view_finances", "approve_payments", "manage_payment_methods",
				"view_tickets", "respond_tickets", "close_tickets",
				"view_gateways", "manage_gateways",
				"view_staff", "create_staff", "edit_staff", "manage_roles",
				"view_devices", "manage_devices",
			},
		},
	}

	for _, role := range roles {
		// Insert role
		var roleID string
		err := DB.QueryRow(`
			INSERT INTO roles (name, description)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE SET description = $2
			RETURNING id
		`, role.name, role.description).Scan(&roleID)

		if err != nil {
			log.Printf("Error seeding role %s: %v", role.name, err)
			continue
		}

		// Assign permissions to role
		for _, permName := range role.permissions {
			_, err := DB.Exec(`
				INSERT INTO role_permissions (role_id, permission_id)
				SELECT $1, id FROM permissions WHERE name = $2
				ON CONFLICT (role_id, permission_id) DO NOTHING
			`, roleID, permName)

			if err != nil {
				log.Printf("Error assigning permission %s to role %s: %v", permName, role.name, err)
			}
		}
	}

	log.Println("Roles seeded successfully")
}

func seedSuperAdmin() {
	// Check if super admin already exists
	var exists bool
	err := DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM users WHERE user_type = 'SUPER_ADMIN')
	`).Scan(&exists)

	if err != nil {
		log.Printf("Error checking for super admin: %v", err)
		return
	}

	if exists {
		log.Println("Super Admin already exists, skipping creation")
		return
	}

	// Create super admin user
	email := "admin@alhudhud.sa"
	password := generateSecurePassword(16) // Securely generated password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing super admin password: %v", err)
		return
	}

	var userID string
	err = DB.QueryRow(`
		INSERT INTO users (email, password_hash, user_type, is_active)
		VALUES ($1, $2, 'SUPER_ADMIN', true)
		RETURNING id
	`, email, string(passwordHash)).Scan(&userID)

	if err != nil {
		log.Printf("Error creating super admin user: %v", err)
		return
	}

	log.Printf("Super Admin created successfully")
	log.Printf("Email: %s", email)
	log.Printf("Password: %s (CHANGE THIS IMMEDIATELY)", password)
}

func seedTestTenant() {
	// Create tenant user if not exists
	var userID string
	err := DB.QueryRow(`SELECT id FROM users WHERE email = 'tenant@test.sa'`).Scan(&userID)

	if err == sql.ErrNoRows {
		log.Println("Creating test tenant user...")
		email := "tenant@test.sa"
		password := generateSecurePassword(16)
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing test tenant password: %v", err)
			return
		}

		err = DB.QueryRow(`
			INSERT INTO users (email, password_hash, user_type, is_active)
			VALUES ($1, $2, 'TENANT', true)
			RETURNING id
		`, email, string(passwordHash)).Scan(&userID)

		if err != nil {
			log.Printf("Error creating test tenant user: %v", err)
			return
		}
		log.Printf("Test Tenant user created successfully with Password: %s", password)
	} else if err != nil {
		log.Printf("Error checking for test tenant user: %v", err)
		return
	}

	// Create tenant profile if not exists
	var profileID string
	err = DB.QueryRow(`SELECT id FROM tenants WHERE user_id = $1`, userID).Scan(&profileID)

	if err == sql.ErrNoRows {
		log.Println("Creating test tenant profile...")
		_, err = DB.Exec(`
			INSERT INTO tenants (user_id, business_name, contact_name, email, phone, balance, subscription_plan, message_limit, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, userID, "Test Company", "John Doe", "tenant@test.sa", "+966500000001", 1000.0, "Premium", 10000, true)

		if err != nil {
			log.Printf("Error creating test tenant profile: %v", err)
			return
		}
		log.Printf("Test Tenant profile created successfully")
	} else if err != nil {
		log.Printf("Error checking for test tenant profile: %v", err)
		return
	}

	log.Printf("Test Tenant Email: tenant@test.sa, Password: CHECK CONSOLE LOGS AT CREATION")
}

func seedStaff() {
	staffUsers := []struct {
		email    string
		password string
		fullName string
		roleName string
	}{
		{
			email:    "support@alhudhud.sa",
			fullName: "Support Agent One",
			roleName: "Support Agent",
		},
		{
			email:    "accountant@alhudhud.sa",
			fullName: "Accountant One",
			roleName: "Accountant",
		},
		{
			email:    "tech@alhudhud.sa",
			fullName: "Tech Admin One",
			roleName: "Technical Admin",
		},
	}

	for _, s := range staffUsers {
		// Generate secure password per user
		password := generateSecurePassword(16)
		
		// Check if user exists
		var userID string
		err := DB.QueryRow(`SELECT id FROM users WHERE email = $1`, s.email).Scan(&userID)

		if err == sql.ErrNoRows {
			log.Printf("Creating staff user: %s", s.email)
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Error hashing password for %s: %v", s.email, err)
				continue
			}

			err = DB.QueryRow(`
				INSERT INTO users (email, password_hash, user_type, is_active)
				VALUES ($1, $2, 'STAFF', true)
				RETURNING id
			`, s.email, string(passwordHash)).Scan(&userID)

			if err != nil {
				log.Printf("Error creating user %s: %v", s.email, err)
				continue
			}
		} else if err != nil {
			log.Printf("Error checking user %s: %v", s.email, err)
			continue
		}

		// Get role ID
		var roleID string
		err = DB.QueryRow(`SELECT id FROM roles WHERE name = $1`, s.roleName).Scan(&roleID)
		if err != nil {
			log.Printf("Error finding role %s for %s: %v", s.roleName, s.email, err)
			continue
		}

		// Create or update staff profile
		_, err = DB.Exec(`
			INSERT INTO staff_profiles (user_id, full_name, role_id, is_active)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (user_id) DO UPDATE SET 
				full_name = $2,
				role_id = $3
		`, userID, s.fullName, roleID)

		if err != nil {
			log.Printf("Error creating profile for %s: %v", s.email, err)
			continue
		}

		log.Printf("Staff user seeded: %s (%s) | Password: %s", s.email, s.roleName, password)
	}
}
