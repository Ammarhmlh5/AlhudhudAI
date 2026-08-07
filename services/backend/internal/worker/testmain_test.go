package worker

import (
	"os"
	"testing"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", "localhost")
	}
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", "5432")
	}
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", "postgres")
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", "sms_gateway_test")
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", os.Getenv("PGPASSWORD"))
	}
	if os.Getenv("DB_SSLMODE") == "" {
		os.Setenv("DB_SSLMODE", "disable")
	}
	db.InitGormDB()
	os.Exit(m.Run())
}

func createTestUser(t *testing.T) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	err := db.GormDB.Exec(`
		INSERT INTO users (id, email, password_hash, user_type, is_active, is_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, "worker-"+userID.String()+"@test.com", "hash", "ADMIN", true, true, time.Now(), time.Now()).Error
	require.NoError(t, err, "Failed to create test user")
	t.Cleanup(func() {
		db.GormDB.Exec("DELETE FROM tenants WHERE user_id = ?", userID)
		db.GormDB.Exec("DELETE FROM users WHERE id = ?", userID)
	})
	return userID
}
