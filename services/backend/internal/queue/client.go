package queue

import (
	"context"
	"fmt"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// InitTenantClient creates a River client specialized for a tenant's schema
func InitTenantClient(ctx context.Context, tenantID uuid.UUID, redisClient *redis.Client) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &SendSMSWorker{})
	river.AddWorker(workers, &GlobalEventWorker{})
	river.AddWorker(workers, &AnalysisWorker{})
	river.AddWorker(workers, &MessageActivationWorker{Redis: redisClient})
	river.AddWorker(workers, &DLRWorker{})
	river.AddWorker(workers, &StatisticsWorker{})

	driver := riverpgxv5.New(db.PgxPool)

	client, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 1},
		},
		Workers: workers,
		Schema:  db.GetSchemaName(tenantID),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create river client for tenant %s: %w", tenantID, err)
	}

	return client, nil
}
