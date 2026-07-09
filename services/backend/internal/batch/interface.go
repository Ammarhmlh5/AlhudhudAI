package batch

import "context"

// Processor defines the interface for batch processing
type Processor interface {
	// ProcessBatch processes a batch of messages for a specific tenant
	// Returns BatchResult containing processing statistics
	ProcessBatch(ctx context.Context, tenantID int64) (*BatchResult, error)

	// ProcessAllTenants processes batches for all active tenants
	// Iterates through all tenants with pending messages
	ProcessAllTenants(ctx context.Context) error

	// AdaptBatchSize adjusts batch size based on buffer pressure
	// Returns optimal batch size for current load
	AdaptBatchSize(ctx context.Context, tenantID int64) int

	// Start starts the batch processor
	// Runs continuously until stopped or context cancelled
	Start(ctx context.Context) error

	// Stop stops the batch processor gracefully
	// Waits for current batch to complete
	Stop(ctx context.Context) error
}
