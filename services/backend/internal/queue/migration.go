package queue

import (
	"context"
	"fmt"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// MigrateSchema runs River migrations for a specific schema
func MigrateSchema(ctx context.Context, schemaName string) error {
	migrator, err := rivermigrate.New[pgx.Tx](riverpgxv5.New(db.PgxPool), &rivermigrate.Config{
		Schema: schemaName,
	})
	if err != nil {
		return fmt.Errorf("failed to create river migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})

	if err != nil {
		return fmt.Errorf("failed to run river migrations for schema %s: %w", schemaName, err)
	}

	return nil
}
