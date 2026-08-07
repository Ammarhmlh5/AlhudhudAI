package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HealthChecker performs periodic health checks on dedicated servers
type HealthChecker struct {
	db             *sqlx.DB
	httpClient     *http.Client
	checkInterval  time.Duration
	checkTimeout   time.Duration
	classifier     *StatusClassifier
	alertTrigger   *AlertTrigger
	logger         *HistoryLogger
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(db *sqlx.DB, classifier *StatusClassifier, alertTrigger *AlertTrigger, logger *HistoryLogger) *HealthChecker {
	return &HealthChecker{
		db:            db,
		httpClient:    &http.Client{
			Timeout: 10 * time.Second,
		},
		checkInterval: 30 * time.Second,
		checkTimeout:  10 * time.Second,
		classifier:    classifier,
		alertTrigger:  alertTrigger,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the periodic health check process
func (hc *HealthChecker) Start(ctx context.Context) error {
	hc.wg.Add(1)
	go hc.runPeriodicChecks(ctx)
	return nil
}

// Stop gracefully stops the health checker
func (hc *HealthChecker) Stop() error {
	close(hc.stopChan)
	hc.wg.Wait()
	return nil
}

// runPeriodicChecks runs health checks on all active servers every 30 seconds
func (hc *HealthChecker) runPeriodicChecks(ctx context.Context) {
	defer hc.wg.Done()
	
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()
	
	// Run initial check immediately
	hc.checkAllServers(ctx)
	
	for {
		select {
		case <-ticker.C:
			hc.checkAllServers(ctx)
		case <-hc.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAllServers retrieves all active servers and checks their health
func (hc *HealthChecker) checkAllServers(ctx context.Context) {
	servers, err := hc.getActiveServers(ctx)
	if err != nil {
		fmt.Printf("Error getting active servers: %v\n", err)
		return
	}
	
	// Check each server concurrently
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(s ServerInfo) {
			defer wg.Done()
			result := hc.CheckServerHealth(ctx, s)
			
			// Log the result
			if err := hc.logger.LogHealthCheck(ctx, result); err != nil {
				fmt.Printf("Error logging health check for server %s: %v\n", s.ID, err)
			}
			
			// Check for status changes and trigger alerts if needed
			if hc.alertTrigger != nil {
				if err := hc.alertTrigger.OnStatusChange(ctx, result); err != nil {
					fmt.Printf("Error triggering alert for server %s: %v\n", s.ID, err)
				}
			}
		}(server)
	}
	
	wg.Wait()
}

// CheckServerHealth performs a health check on a single server
func (hc *HealthChecker) CheckServerHealth(ctx context.Context, server ServerInfo) *HealthCheckResult {
	result := &HealthCheckResult{
		ServerID:       server.ID,
		TenantID:       server.TenantID,
		CheckTimestamp: time.Now(),
		IsAvailable:    false,
		ErrorCount:     0,
		ErrorRate:      0.0,
	}
	
	// Construct health check URL
	healthURL := server.ServerURL
	if server.HealthEndpoint != "" {
		healthURL = fmt.Sprintf("%s%s", server.ServerURL, server.HealthEndpoint)
	} else {
		healthURL = fmt.Sprintf("%s/health", server.ServerURL)
	}
	
	// Create request with timeout
	reqCtx, cancel := context.WithTimeout(ctx, hc.checkTimeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		result.ErrorCount = 1
		result.HealthStatus = Unhealthy
		return result
	}
	
	// Perform the health check
	startTime := time.Now()
	resp, err := hc.httpClient.Do(req)
	responseTime := time.Since(startTime)
	result.ResponseTimeMs = int(responseTime.Milliseconds())
	
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Health check failed: %v", err)
		result.ErrorCount = 1
		result.IsAvailable = false
	} else {
		defer resp.Body.Close()
		result.StatusCode = resp.StatusCode
		result.IsAvailable = resp.StatusCode >= 200 && resp.StatusCode < 300
		
		if !result.IsAvailable {
			result.ErrorMessage = fmt.Sprintf("Unhealthy status code: %d", resp.StatusCode)
			result.ErrorCount = 1
		}
	}
	
	// Calculate error rate from recent history
	errorRate, err := hc.calculateRecentErrorRate(ctx, server.ID)
	if err != nil {
		fmt.Printf("Error calculating error rate for server %s: %v\n", server.ID, err)
		errorRate = 0.0
	}
	result.ErrorRate = errorRate
	
	// Classify health status
	result.HealthStatus = hc.classifier.ClassifyHealthStatus(result.ResponseTimeMs, result.ErrorRate)
	
	return result
}

// getActiveServers retrieves all active dedicated servers from the database
func (hc *HealthChecker) getActiveServers(ctx context.Context) ([]ServerInfo, error) {
	query := `
		SELECT
			dg.id,
			dg.tenant_id,
			COALESCE(NULLIF(dg.server_hostname, ''), dg.server_ip, '') as server_url,
			'/health' as health_endpoint,
			true as is_active
		FROM dedicated_gateways dg
		WHERE dg.status = 'active'
	`

	var servers []ServerInfo
	if err := hc.db.SelectContext(ctx, &servers, query); err != nil {
		return nil, fmt.Errorf("failed to get active servers: %w", err)
	}

	return servers, nil
}

// calculateRecentErrorRate calculates the error rate from the last 10 health checks
func (hc *HealthChecker) calculateRecentErrorRate(ctx context.Context, serverID uuid.UUID) (float64, error) {
	query := `
		SELECT 
			COUNT(*) as total_checks,
			SUM(CASE WHEN is_available = false THEN 1 ELSE 0 END) as failed_checks
		FROM (
			SELECT is_available
			FROM server_health_logs
			WHERE server_id = $1
			ORDER BY check_timestamp DESC
			LIMIT 10
		) recent_checks
	`
	
	var totalChecks, failedChecks int
	err := hc.db.QueryRowContext(ctx, query, serverID).Scan(&totalChecks, &failedChecks)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0.0, nil
		}
		return 0.0, fmt.Errorf("failed to calculate error rate: %w", err)
	}
	
	if totalChecks == 0 {
		return 0.0, nil
	}
	
	errorRate := (float64(failedChecks) / float64(totalChecks)) * 100.0
	return errorRate, nil
}

// CheckServerHealthOnce performs a single health check on a specific server
func (hc *HealthChecker) CheckServerHealthOnce(ctx context.Context, serverID uuid.UUID) (*HealthCheckResult, error) {
	// Get server info
	query := `
		SELECT
			dg.id,
			dg.tenant_id,
			COALESCE(NULLIF(dg.server_hostname, ''), dg.server_ip, '') as server_url,
			'/health' as health_endpoint,
			dg.status = 'active' as is_active
		FROM dedicated_gateways dg
		WHERE dg.id = $1
	`
	
	var server ServerInfo
	err := hc.db.GetContext(ctx, &server, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	
	// Perform health check
	result := hc.CheckServerHealth(ctx, server)
	
	// Log the result
	if err := hc.logger.LogHealthCheck(ctx, result); err != nil {
		return result, fmt.Errorf("failed to log health check: %w", err)
	}
	
	return result, nil
}

// GetAllServers returns all servers matching the optional filters
func (hc *HealthChecker) GetAllServers(ctx context.Context, statusFilter *string, tenantFilter *uuid.UUID) ([]ServerInfo, error) {
	return hc.getActiveServers(ctx)
}

// GetServer returns a specific server
func (hc *HealthChecker) GetServer(ctx context.Context, serverID uuid.UUID) (*ServerInfo, error) {
	query := `
		SELECT
			dg.id,
			dg.tenant_id,
			COALESCE(NULLIF(dg.server_hostname, ''), dg.server_ip, '') as server_url,
			'/health' as health_endpoint,
			dg.status = 'active' as is_active
		FROM dedicated_gateways dg
		WHERE dg.id = $1
	`

	var server ServerInfo
	err := hc.db.GetContext(ctx, &server, query, serverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found")
		}
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	return &server, nil
}

// GetServerWithHealth returns a server merged with its latest health check result
func (hc *HealthChecker) GetServerWithHealth(ctx context.Context, serverID uuid.UUID) (*ServerHealthDetail, error) {
	server, err := hc.GetServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	detail := &ServerHealthDetail{
		ID:             server.ID,
		TenantID:       server.TenantID,
		ServerURL:      server.ServerURL,
		HealthEndpoint: server.HealthEndpoint,
		IsActive:       server.IsActive,
		HealthStatus:   "unknown",
	}

	latest, err := hc.logger.GetLatestHealthCheck(ctx, serverID)
	if err == nil && latest != nil {
		detail.HealthStatus = string(latest.HealthStatus)
		detail.ResponseTimeMs = latest.ResponseTimeMs
		detail.ErrorRate = latest.ErrorRate
		detail.LastCheck = &latest.CheckTimestamp
		detail.IsAvailable = latest.IsAvailable
	}

	return detail, nil
}
