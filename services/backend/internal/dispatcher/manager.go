package dispatcher

import (
	"sync"
	"time"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/al-hudhud/backend/internal/redis"
	"go.uber.org/zap"
)

// Manager manages all tenant dispatchers
type Manager struct {
	dispatchers map[uuid.UUID]*TenantDispatcher
	redisClient *redis.Client
	mu          sync.RWMutex
	stopChan    chan struct{}
}

var GlobalManager *Manager

// NewManager creates a new dispatcher manager
func NewManager(redisClient *redis.Client) *Manager {
	m := &Manager{
		dispatchers: make(map[uuid.UUID]*TenantDispatcher),
		redisClient: redisClient,
		stopChan:    make(chan struct{}),
	}
	GlobalManager = m
	return m
}

// Start initializes and starts dispatchers for all active tenants
func (m *Manager) Start() {
	log := logger.GetLogger()
	log.Info("TenantDispatcher Manager starting...")

	// Load all active tenants
	var tenants []models.Tenant
	if err := db.GormDB.Select("id", "business_name", "is_active").Where("is_active = ?", true).Find(&tenants).Error; err != nil {
		log.Error("Failed to load active tenants", zap.Error(err))
		return
	}

	// Start dispatcher for each tenant
	for _, tenant := range tenants {
		m.StartTenantDispatcher(tenant.ID)
	}

	// Start monitoring goroutine
	go m.monitorTenants()

	log.Info("TenantDispatcher Manager started",
		zap.Int("active_tenants", len(tenants)))
}

// Stop stops all dispatchers
func (m *Manager) Stop() {
	log := logger.GetLogger()
	log.Info("TenantDispatcher Manager stopping...")

	close(m.stopChan)

	m.mu.Lock()
	defer m.mu.Unlock()

	for tenantID, dispatcher := range m.dispatchers {
		dispatcher.Stop()
		delete(m.dispatchers, tenantID)
	}

	log.Info("TenantDispatcher Manager stopped")
}

// StartTenantDispatcher starts a dispatcher for a specific tenant
func (m *Manager) StartTenantDispatcher(tenantID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if _, exists := m.dispatchers[tenantID]; exists {
		return nil // Already running
	}

	// Create and start new dispatcher
	dispatcher := NewTenantDispatcher(tenantID, m.redisClient)
	if err := dispatcher.Start(); err != nil {
		return err
	}

	m.dispatchers[tenantID] = dispatcher
	return nil
}

// StopTenantDispatcher stops a dispatcher for a specific tenant
func (m *Manager) StopTenantDispatcher(tenantID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dispatcher, exists := m.dispatchers[tenantID]; exists {
		dispatcher.Stop()
		delete(m.dispatchers, tenantID)
	}
}

// monitorTenants periodically checks for new/removed tenants
func (m *Manager) monitorTenants() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.syncTenants()
		case <-m.stopChan:
			return
		}
	}
}

// syncTenants synchronizes dispatchers with active tenants
func (m *Manager) syncTenants() {
	log := logger.GetLogger()

	// Get all active tenants
	var tenants []models.Tenant
	if err := db.GormDB.Select("id", "business_name", "is_active").Where("is_active = ?", true).Find(&tenants).Error; err != nil {
		log.Error("Failed to sync tenants", zap.Error(err))
		return
	}

	activeTenantIDs := make(map[uuid.UUID]bool)
	for _, tenant := range tenants {
		activeTenantIDs[tenant.ID] = true

		// Start dispatcher if not running
		m.mu.Lock()
		_, exists := m.dispatchers[tenant.ID]
		if !exists {
			log.Info("Starting dispatcher for new tenant",
				zap.String("tenant_id", tenant.ID.String()))
			// Create and start new dispatcher (inline to keep lock)
			dispatcher := NewTenantDispatcher(tenant.ID, m.redisClient)
			if err := dispatcher.Start(); err == nil {
				m.dispatchers[tenant.ID] = dispatcher
			} else {
				log.Error("Failed to start dispatcher", zap.String("tenant_id", tenant.ID.String()), zap.Error(err))
			}
		}
		m.mu.Unlock()
	}

	// Stop dispatchers for inactive tenants
	m.mu.RLock()
	currentDispatchers := make([]uuid.UUID, 0, len(m.dispatchers))
	for tenantID := range m.dispatchers {
		currentDispatchers = append(currentDispatchers, tenantID)
	}
	m.mu.RUnlock()

	for _, tenantID := range currentDispatchers {
		if !activeTenantIDs[tenantID] {
			log.Info("Stopping dispatcher for inactive tenant",
				zap.String("tenant_id", tenantID.String()))
			m.StopTenantDispatcher(tenantID)
		}
	}
}

// GetStats returns statistics for all dispatchers
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_dispatchers"] = len(m.dispatchers)

	dispatcherStats := make([]map[string]interface{}, 0, len(m.dispatchers))
	for _, dispatcher := range m.dispatchers {
		dispatcherStats = append(dispatcherStats, dispatcher.GetStats())
	}
	stats["dispatchers"] = dispatcherStats

	return stats
}

// GetRiverClient returns the River client for a specific tenant
func (m *Manager) GetRiverClient(tenantID uuid.UUID) (*river.Client[pgx.Tx], error) {
	m.mu.RLock()
	dispatcher, exists := m.dispatchers[tenantID]
	m.mu.RUnlock()

	if !exists {
		// Try to start it if it's an active tenant
		if err := m.StartTenantDispatcher(tenantID); err != nil {
			return nil, err
		}

		m.mu.RLock()
		dispatcher = m.dispatchers[tenantID]
		m.mu.RUnlock()
	}

	return dispatcher.riverClient, nil
}
