package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRowLevelSecurity(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("Skipping RLS test: DB environment variables not set")
	}

	InitDB()
	ctx := context.Background()

	// 1. Create two separate tenants
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()

	// 2. Clear existing test data (optional but safer for isolation tests)
	_, _ = DB.Exec("DELETE FROM messages WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)

	// 3. Insert data for both tenants as superuser (Direct SQL)
	_, err := DB.Exec(`
		INSERT INTO messages (id, tenant_id, phone_number, content, status) 
		VALUES ($1, $2, '123456789', 'Tenant 1 Message', 'PENDING'),
		       ($3, $4, '987654321', 'Tenant 2 Message', 'PENDING')
	`, uuid.New(), tenant1ID, uuid.New(), tenant2ID)
	require.NoError(t, err)

	// 4. Test Isolation for Tenant 1
	t.Run("Tenant 1 Visibility", func(t *testing.T) {
		tx, err := DB.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		// Set tenant 1 context
		_, err = tx.Exec("SELECT set_config('app.current_tenant', $1, true)", tenant1ID.String())
		require.NoError(t, err)

		// Query messages
		rows, err := tx.Query("SELECT content FROM messages")
		require.NoError(t, err)
		defer rows.Close()

		var contents []string
		for rows.Next() {
			var content string
			rows.Scan(&content)
			contents = append(contents, content)
		}

		assert.Len(t, contents, 1)
		assert.Equal(t, "Tenant 1 Message", contents[0])
	})

	// 5. Test Isolation for Tenant 2
	t.Run("Tenant 2 Visibility", func(t *testing.T) {
		tx, err := DB.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		// Set tenant 2 context
		_, err = tx.Exec("SELECT set_config('app.current_tenant', $1, true)", tenant2ID.String())
		require.NoError(t, err)

		// Query messages
		rows, err := tx.Query("SELECT content FROM messages")
		require.NoError(t, err)
		defer rows.Close()

		var contents []string
		for rows.Next() {
			var content string
			rows.Scan(&content)
			contents = append(contents, content)
		}

		assert.Len(t, contents, 1)
		assert.Equal(t, "Tenant 2 Message", contents[0])
	})

	// 6. Test Super Admin Override
	t.Run("Super Admin Visibility", func(t *testing.T) {
		tx, err := DB.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		// Set super admin context
		_, err = tx.Exec("SELECT set_config('app.is_super_admin', 'true', true)")
		require.NoError(t, err)

		// Query messages
		rows, err := tx.Query("SELECT content FROM messages WHERE tenant_id IN ($1, $2)", tenant1ID, tenant2ID)
		require.NoError(t, err)
		defer rows.Close()

		var contents []string
		for rows.Next() {
			var content string
			rows.Scan(&content)
			contents = append(contents, content)
		}

		assert.Len(t, contents, 2)
	})
}
