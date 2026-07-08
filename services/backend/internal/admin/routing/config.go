package routing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RoutingConfigManager manages routing configurations for companies
type RoutingConfigManager struct {
	db *sqlx.DB
}

// NewRoutingConfigManager creates a new routing configuration manager
func NewRoutingConfigManager(db *sqlx.DB) *RoutingConfigManager {
	return &RoutingConfigManager{
		db: db,
	}
}

// HasDedicatedServer checks if the configuration has a dedicated server
func (rc *RoutingConfig) HasDedicatedServer() bool {
	return rc.RoutingType == "dedicated" && rc.DedicatedServerID != nil && rc.DedicatedServerURL != nil
}

// GetRoutingConfig retrieves the routing configuration for a tenant
// Requirement 2.1, 2.2: Detect company type and retrieve routing settings within 100ms
func (rcm *RoutingConfigManager) GetRoutingConfig(ctx context.Context, tenantID uuid.UUID) (*RoutingConfig, error) {
	query := `
		SELECT 
			id, tenant_id, routing_type, dedicated_server_id, dedicated_server_url,
			dedicated_server_api_key, total_requests, successful_requests, failed_requests,
			average_latency_ms, is_active, last_health_check, health_status,
			created_at, updated_at
		FROM routing_configurations
		WHERE tenant_id = $1 AND is_active = true
	`
	
	var config RoutingConfig
	err := rcm.db.GetContext(ctx, &config, query, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No configuration found, return default shared configuration
			return &RoutingConfig{
				TenantID:    tenantID,
				RoutingType: "shared",
				IsActive:    true,
			}, nil
		}
		return nil, fmt.Errorf("failed to get routing config: %w", err)
	}
	
	return &config, nil
}

// Validate validates the update input
func (input *UpdateRoutingConfigInput) Validate() error {
	// Validate routing type
	if input.RoutingType != "shared" && input.RoutingType != "dedicated" {
		return fmt.Errorf("invalid routing_type: must be 'shared' or 'dedicated'")
	}
	
	// If dedicated, ensure required fields are present
	if input.RoutingType == "dedicated" {
		if input.DedicatedServerID == nil {
			return fmt.Errorf("dedicated_server_id is required for dedicated routing")
		}
		if input.DedicatedServerURL == nil || *input.DedicatedServerURL == "" {
			return fmt.Errorf("dedicated_server_url is required for dedicated routing")
		}
	}
	
	return nil
}

// UpdateRoutingConfig updates the routing configuration for a tenant
// Requirement 2.1, 2.2: Update routing settings with validation
func (rcm *RoutingConfigManager) UpdateRoutingConfig(ctx context.Context, input *UpdateRoutingConfigInput) (*RoutingConfig, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Check if configuration exists
	existingConfig, err := rcm.GetRoutingConfig(ctx, input.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing config: %w", err)
	}
	
	// If dedicated server is specified, verify it exists
	if input.DedicatedServerID != nil {
		exists, err := rcm.verifyDedicatedServerExists(ctx, *input.DedicatedServerID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify dedicated server: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("dedicated server not found: %s", input.DedicatedServerID)
		}
	}
	
	// Update or insert configuration
	if existingConfig.ID > 0 {
		// Update existing configuration
		return rcm.updateExistingConfig(ctx, existingConfig.ID, input)
	} else {
		// Insert new configuration
		return rcm.insertNewConfig(ctx, input)
	}
}

// updateExistingConfig updates an existing routing configuration
func (rcm *RoutingConfigManager) updateExistingConfig(ctx context.Context, configID int64, input *UpdateRoutingConfigInput) (*RoutingConfig, error) {
	query := `
		UPDATE routing_configurations
		SET 
			routing_type = $1,
			dedicated_server_id = $2,
			dedicated_server_url = $3,
			dedicated_server_api_key = $4,
			is_active = COALESCE($5, is_active),
			updated_at = NOW()
		WHERE id = $6
		RETURNING 
			id, tenant_id, routing_type, dedicated_server_id, dedicated_server_url,
			dedicated_server_api_key, total_requests, successful_requests, failed_requests,
			average_latency_ms, is_active, last_health_check, health_status,
			created_at, updated_at
	`
	
	var config RoutingConfig
	err := rcm.db.GetContext(ctx, &config, query,
		input.RoutingType,
		input.DedicatedServerID,
		input.DedicatedServerURL,
		input.DedicatedServerAPIKey,
		input.IsActive,
		configID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update routing config: %w", err)
	}
	
	return &config, nil
}

// insertNewConfig inserts a new routing configuration
func (rcm *RoutingConfigManager) insertNewConfig(ctx context.Context, input *UpdateRoutingConfigInput) (*RoutingConfig, error) {
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	
	query := `
		INSERT INTO routing_configurations (
			tenant_id, routing_type, dedicated_server_id, dedicated_server_url,
			dedicated_server_api_key, is_active
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING 
			id, tenant_id, routing_type, dedicated_server_id, dedicated_server_url,
			dedicated_server_api_key, total_requests, successful_requests, failed_requests,
			average_latency_ms, is_active, last_health_check, health_status,
			created_at, updated_at
	`
	
	var config RoutingConfig
	err := rcm.db.GetContext(ctx, &config, query,
		input.TenantID,
		input.RoutingType,
		input.DedicatedServerID,
		input.DedicatedServerURL,
		input.DedicatedServerAPIKey,
		isActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert routing config: %w", err)
	}
	
	return &config, nil
}

// verifyDedicatedServerExists checks if a dedicated server exists
func (rcm *RoutingConfigManager) verifyDedicatedServerExists(ctx context.Context, serverID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM dedicated_gateways WHERE id = $1)`
	
	var exists bool
	err := rcm.db.GetContext(ctx, &exists, query, serverID)
	if err != nil {
		return false, fmt.Errorf("failed to check server existence: %w", err)
	}
	
	return exists, nil
}

// UpdateHealthStatus updates the health status of a routing configuration
func (rcm *RoutingConfigManager) UpdateHealthStatus(ctx context.Context, tenantID uuid.UUID, healthStatus string, lastCheckTime time.Time) error {
	query := `
		UPDATE routing_configurations
		SET 
			health_status = $1,
			last_health_check = $2,
			updated_at = NOW()
		WHERE tenant_id = $3
	`
	
	_, err := rcm.db.ExecContext(ctx, query, healthStatus, lastCheckTime, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update health status: %w", err)
	}
	
	return nil
}
