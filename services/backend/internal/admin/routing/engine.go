package routing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	// ErrServerUnavailable is returned when a dedicated server is unavailable
	ErrServerUnavailable = errors.New("dedicated server unavailable")
	// ErrNoRoutingConfig is returned when no routing configuration is found
	ErrNoRoutingConfig = errors.New("no routing configuration found")
	// ErrInvalidRoutingType is returned when routing type is invalid
	ErrInvalidRoutingType = errors.New("invalid routing type")
)

// RoutingEngine handles intelligent message routing to dedicated or shared servers
type RoutingEngine struct {
	db            *sqlx.DB
	configManager *RoutingConfigManager
	metricsTracker *MetricsTracker
}

// NewRoutingEngine creates a new routing engine instance
func NewRoutingEngine(db *sqlx.DB, configManager *RoutingConfigManager, metricsTracker *MetricsTracker) *RoutingEngine {
	return &RoutingEngine{
		db:            db,
		configManager: configManager,
		metricsTracker: metricsTracker,
	}
}

// RouteMessage performs intelligent routing for a message
// Requirement 2.1: Route messages to dedicated server when configured
// Requirement 2.2: Detect company type automatically within 100ms
// Requirement 2.4: Return error if dedicated server is unavailable (no fallback)
func (re *RoutingEngine) RouteMessage(ctx context.Context, message *Message) (*RoutingDecision, error) {
	startTime := time.Now()
	
	// Step 1: Get routing configuration for the tenant
	config, err := re.configManager.GetRoutingConfig(ctx, message.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing config: %w", err)
	}
	
	decision := &RoutingDecision{
		TenantID:    message.TenantID,
		RoutingType: config.RoutingType,
		ShouldRoute: false,
	}
	
	// Step 2: Check routing type
	if config.RoutingType == "shared" || !config.HasDedicatedServer() {
		// Shared infrastructure - no special routing needed
		decision.ShouldRoute = true
		
		// Track metrics
		latency := time.Since(startTime).Milliseconds()
		re.metricsTracker.TrackRoutingDecision(ctx, message.TenantID, true, int(latency))
		
		return decision, nil
	}
	
	// Step 3: Dedicated server routing
	if config.RoutingType == "dedicated" {
		decision.DedicatedServerID = config.DedicatedServerID
		decision.DedicatedServerURL = config.DedicatedServerURL
		decision.HealthStatus = config.HealthStatus
		
		// Step 4: Check server health
		// Requirement 2.4: If server is unhealthy, return error (no fallback)
		if config.HealthStatus != nil && *config.HealthStatus != "healthy" {
			// Log the failure
			re.logRoutingFailure(ctx, message.TenantID, *config.DedicatedServerID, 
				fmt.Sprintf("Server unhealthy: %s", *config.HealthStatus))
			
			// Track failed routing
			latency := time.Since(startTime).Milliseconds()
			re.metricsTracker.TrackRoutingDecision(ctx, message.TenantID, false, int(latency))
			
			decision.Error = ErrServerUnavailable
			return decision, ErrServerUnavailable
		}
		
		// Server is healthy, allow routing
		decision.ShouldRoute = true
		
		// Track successful routing
		latency := time.Since(startTime).Milliseconds()
		re.metricsTracker.TrackRoutingDecision(ctx, message.TenantID, true, int(latency))
		
		return decision, nil
	}
	
	// Unknown routing type
	latency := time.Since(startTime).Milliseconds()
	re.metricsTracker.TrackRoutingDecision(ctx, message.TenantID, false, int(latency))
	
	decision.Error = ErrInvalidRoutingType
	return decision, ErrInvalidRoutingType
}

// GetRoutingDecision returns the routing decision for a tenant without tracking metrics
// This is useful for checking routing configuration without side effects
func (re *RoutingEngine) GetRoutingDecision(ctx context.Context, tenantID uuid.UUID) (*RoutingDecision, error) {
	// Get routing configuration
	config, err := re.configManager.GetRoutingConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing config: %w", err)
	}
	
	decision := &RoutingDecision{
		TenantID:    tenantID,
		RoutingType: config.RoutingType,
		ShouldRoute: false,
	}
	
	// Check routing type
	if config.RoutingType == "shared" || !config.HasDedicatedServer() {
		decision.ShouldRoute = true
		return decision, nil
	}
	
	// Dedicated server routing
	if config.RoutingType == "dedicated" {
		decision.DedicatedServerID = config.DedicatedServerID
		decision.DedicatedServerURL = config.DedicatedServerURL
		decision.HealthStatus = config.HealthStatus
		
		// Check server health
		if config.HealthStatus != nil && *config.HealthStatus != "healthy" {
			decision.Error = ErrServerUnavailable
			return decision, ErrServerUnavailable
		}
		
		decision.ShouldRoute = true
		return decision, nil
	}
	
	decision.Error = ErrInvalidRoutingType
	return decision, ErrInvalidRoutingType
}

// CheckServerHealth checks if a dedicated server is healthy for a tenant
func (re *RoutingEngine) CheckServerHealth(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	config, err := re.configManager.GetRoutingConfig(ctx, tenantID)
	if err != nil {
		return false, fmt.Errorf("failed to get routing config: %w", err)
	}
	
	// If not dedicated, always return true
	if config.RoutingType != "dedicated" || !config.HasDedicatedServer() {
		return true, nil
	}
	
	// Check health status
	if config.HealthStatus == nil {
		// No health check performed yet, assume unhealthy
		return false, nil
	}
	
	return *config.HealthStatus == "healthy", nil
}

// logRoutingFailure logs a routing failure to the database
func (re *RoutingEngine) logRoutingFailure(ctx context.Context, tenantID, serverID uuid.UUID, reason string) {
	query := `
		INSERT INTO server_health_logs (
			server_id, tenant_id, check_timestamp, is_available, 
			error_message, health_status
		) VALUES ($1, $2, NOW(), false, $3, 'unhealthy')
	`
	
	_, err := re.db.ExecContext(ctx, query, serverID, tenantID, reason)
	if err != nil {
		// Log error but don't fail the routing decision
		fmt.Printf("Failed to log routing failure: %v\n", err)
	}
}

// RouteToSharedInfrastructure is a helper function for routing to shared infrastructure
// This is called when a company doesn't have a dedicated server
func (re *RoutingEngine) RouteToSharedInfrastructure(ctx context.Context, message *Message) error {
	// This is a placeholder - actual implementation would integrate with
	// the existing message routing system
	return nil
}

// SendToDedicatedServer sends a message to a dedicated server
// This is a placeholder for the actual HTTP client implementation
func (re *RoutingEngine) SendToDedicatedServer(ctx context.Context, serverURL string, apiKey string, message *Message) error {
	// This is a placeholder - actual implementation would:
	// 1. Create HTTP client with timeout
	// 2. Add authentication headers (API key)
	// 3. Send POST request to server
	// 4. Handle response and errors
	// 5. Track metrics
	return nil
}

// GetRoutingStats returns routing statistics for a tenant
func (re *RoutingEngine) GetRoutingStats(ctx context.Context, tenantID uuid.UUID) (*RoutingStats, error) {
	config, err := re.configManager.GetRoutingConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing config: %w", err)
	}
	
	stats := &RoutingStats{
		TenantID:           tenantID,
		RoutingType:        config.RoutingType,
		TotalRequests:      config.TotalRequests,
		SuccessfulRequests: config.SuccessfulRequests,
		FailedRequests:     config.FailedRequests,
		AverageLatencyMs:   config.AverageLatencyMs,
		HealthStatus:       config.HealthStatus,
		LastHealthCheck:    config.LastHealthCheck,
	}
	
	// Calculate success rate
	if config.TotalRequests > 0 {
		successRate := (float64(config.SuccessfulRequests) / float64(config.TotalRequests)) * 100.0
		stats.SuccessRate = &successRate
	}
	
	return stats, nil
}
