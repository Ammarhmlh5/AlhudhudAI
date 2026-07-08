package distribution

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrStrategyNotConfigured is returned when no strategy is configured for a tenant
	ErrStrategyNotConfigured = errors.New("distribution strategy not configured for tenant")
	// ErrInvalidStrategyType is returned when an unknown strategy type is encountered
	ErrInvalidStrategyType = errors.New("invalid distribution strategy type")
)

// strategyCacheEntry wraps a strategy with a creation timestamp so entries
// can be expired automatically.
type strategyCacheEntry struct {
	strategy DistributionStrategy
	cachedAt time.Time
}

// StrategySelector manages the selection and creation of distribution strategies.
// Thread-safe: all map accesses are protected by mu.
type StrategySelector struct {
	db             *gorm.DB
	poolManager    *DevicePoolManager
	failureHandler *DeviceFailureHandler
	// mu protects strategyCache and configCache from concurrent access.
	mu            sync.RWMutex
	strategyCache map[uuid.UUID]strategyCacheEntry
	configCache   map[uuid.UUID]*DistributionConfig
	// cacheTTL controls how long a cached strategy is considered fresh.
	// Default: 30 seconds — ensures device-pool changes are reflected quickly.
	cacheTTL time.Duration
}

// NewStrategySelector creates a new strategy selector
func NewStrategySelector(db *gorm.DB, poolManager *DevicePoolManager, failureHandler *DeviceFailureHandler) *StrategySelector {
	return &StrategySelector{
		db:             db,
		poolManager:    poolManager,
		failureHandler: failureHandler,
		strategyCache:  make(map[uuid.UUID]strategyCacheEntry),
		configCache:    make(map[uuid.UUID]*DistributionConfig),
		cacheTTL:       30 * time.Second,
	}
}

// GetStrategy retrieves the appropriate distribution strategy for a tenant.
// Uses factory pattern to create strategy instances based on configuration.
// Thread-safe. Automatically rebuilds the strategy after cacheTTL expires so
// that newly-online or offline devices are reflected without manual refresh.
func (ss *StrategySelector) GetStrategy(ctx context.Context, tenantID uuid.UUID) (DistributionStrategy, error) {
	// Fast path: read-lock and check cache freshness.
	ss.mu.RLock()
	entry, exists := ss.strategyCache[tenantID]
	ss.mu.RUnlock()

	if exists && time.Since(entry.cachedAt) < ss.cacheTTL {
		return entry.strategy, nil
	}

	// Cache miss or expired — build new strategy under write-lock.
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Re-check after acquiring write-lock (double-checked locking).
	if entry, exists = ss.strategyCache[tenantID]; exists && time.Since(entry.cachedAt) < ss.cacheTTL {
		return entry.strategy, nil
	}

	// Fetch configuration from database
	config, err := ss.getDistributionConfigLocked(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Fetch current live devices (always fresh — not cached here).
	devices, err := ss.poolManager.GetAvailableDevices(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available devices: %w", err)
	}

	// Create strategy based on configuration
	strategy, err := ss.createStrategy(config, devices)
	if err != nil {
		return nil, err
	}

	// Store in cache with timestamp.
	ss.strategyCache[tenantID] = strategyCacheEntry{strategy: strategy, cachedAt: time.Now()}
	ss.configCache[tenantID] = config

	return strategy, nil
}

// getDistributionConfigLocked fetches the distribution configuration for a tenant.
// Caller MUST hold ss.mu (write or read lock) before calling.
func (ss *StrategySelector) getDistributionConfigLocked(ctx context.Context, tenantID uuid.UUID) (*DistributionConfig, error) {
	// Check config cache (already under lock).
	if config, exists := ss.configCache[tenantID]; exists {
		return config, nil
	}

	var config DistributionConfig

	err := ss.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		First(&config).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStrategyNotConfigured
		}
		return nil, fmt.Errorf("failed to fetch distribution config: %w", err)
	}

	return &config, nil
}

// createStrategy creates a strategy instance based on configuration
// Implements factory pattern for strategy creation
func (ss *StrategySelector) createStrategy(config *DistributionConfig, devices []*Device) (DistributionStrategy, error) {
	switch config.StrategyType {
	case StrategyRoundRobin:
		return NewRoundRobinStrategy(devices), nil

	case StrategyLoadBalancing:
		return NewLoadBalancingStrategy(devices), nil

	case StrategyPriority:
		return NewPriorityStrategy(devices), nil

	case StrategyCustom:
		if config.CustomRules == nil {
			return nil, fmt.Errorf("%w: custom rules not defined", ErrInvalidCustomRule)
		}
		return NewCustomRulesStrategy(devices, config.CustomRules), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidStrategyType, config.StrategyType)
	}
}

// RefreshStrategy forces a cache eviction and rebuilds the strategy for a tenant.
// Safe to call concurrently. Should be called when device status changes (online/offline).
func (ss *StrategySelector) RefreshStrategy(ctx context.Context, tenantID uuid.UUID) error {
	ss.mu.Lock()
	delete(ss.strategyCache, tenantID)
	delete(ss.configCache, tenantID)
	ss.mu.Unlock()

	// Rebuild — GetStrategy will re-acquire the lock internally.
	_, err := ss.GetStrategy(ctx, tenantID)
	return err
}

// UpdateStrategyConfig updates the distribution strategy configuration for a tenant
func (ss *StrategySelector) UpdateStrategyConfig(ctx context.Context, tenantID uuid.UUID, strategyType StrategyType, strategyConfig, customRules map[string]interface{}) error {
	// Validate strategy type
	if !isValidStrategyType(strategyType) {
		return fmt.Errorf("%w: %s", ErrInvalidStrategyType, strategyType)
	}

	// Check if configuration exists
	var existingConfig DistributionConfig
	err := ss.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(&existingConfig).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing config: %w", err)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new configuration
		newConfig := &DistributionConfig{
			TenantID:       tenantID,
			StrategyType:   strategyType,
			StrategyConfig: strategyConfig,
			CustomRules:    customRules,
			IsActive:       true,
		}

		err = ss.db.WithContext(ctx).Create(newConfig).Error
		if err != nil {
			return fmt.Errorf("failed to create distribution config: %w", err)
		}
	} else {
		// Update existing configuration
		updates := map[string]interface{}{
			"strategy_type":   strategyType,
			"strategy_config": strategyConfig,
			"custom_rules":    customRules,
		}

		err = ss.db.WithContext(ctx).
			Model(&DistributionConfig{}).
			Where("tenant_id = ?", tenantID).
			Updates(updates).Error

		if err != nil {
			return fmt.Errorf("failed to update distribution config: %w", err)
		}
	}

	// Refresh strategy cache
	return ss.RefreshStrategy(ctx, tenantID)
}

// GetStrategyConfig retrieves the distribution strategy configuration for a tenant.
func (ss *StrategySelector) GetStrategyConfig(ctx context.Context, tenantID uuid.UUID) (*DistributionConfig, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.getDistributionConfigLocked(ctx, tenantID)
}

// DisableStrategy disables the distribution strategy for a tenant
func (ss *StrategySelector) DisableStrategy(ctx context.Context, tenantID uuid.UUID) error {
	err := ss.db.WithContext(ctx).
		Model(&DistributionConfig{}).
		Where("tenant_id = ?", tenantID).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to disable strategy: %w", err)
	}

	// Remove from cache (thread-safe).
	ss.mu.Lock()
	delete(ss.strategyCache, tenantID)
	delete(ss.configCache, tenantID)
	ss.mu.Unlock()

	return nil
}

// EnableStrategy enables the distribution strategy for a tenant
func (ss *StrategySelector) EnableStrategy(ctx context.Context, tenantID uuid.UUID) error {
	err := ss.db.WithContext(ctx).
		Model(&DistributionConfig{}).
		Where("tenant_id = ?", tenantID).
		Update("is_active", true).Error

	if err != nil {
		return fmt.Errorf("failed to enable strategy: %w", err)
	}

	// Refresh strategy
	return ss.RefreshStrategy(ctx, tenantID)
}

// ClearCache clears the strategy and configuration cache. Thread-safe.
func (ss *StrategySelector) ClearCache() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.strategyCache = make(map[uuid.UUID]strategyCacheEntry)
	ss.configCache = make(map[uuid.UUID]*DistributionConfig)
}

// ClearCacheForTenant clears the cache for a specific tenant. Thread-safe.
func (ss *StrategySelector) ClearCacheForTenant(tenantID uuid.UUID) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.strategyCache, tenantID)
	delete(ss.configCache, tenantID)
}

// GetCachedStrategyCount returns the number of cached strategies. Thread-safe.
func (ss *StrategySelector) GetCachedStrategyCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return len(ss.strategyCache)
}

// isValidStrategyType checks if a strategy type is valid
func isValidStrategyType(strategyType StrategyType) bool {
	switch strategyType {
	case StrategyRoundRobin, StrategyLoadBalancing, StrategyPriority, StrategyCustom:
		return true
	default:
		return false
	}
}

// DistributionManager combines all distribution components
// This is the main entry point for message distribution
type DistributionManager struct {
	selector       *StrategySelector
	poolManager    *DevicePoolManager
	failureHandler *DeviceFailureHandler
}

// NewDistributionManager creates a new distribution manager
func NewDistributionManager(db *gorm.DB) *DistributionManager {
	failureHandler := NewDeviceFailureHandler()
	poolManager := NewDevicePoolManager(db, failureHandler)
	selector := NewStrategySelector(db, poolManager, failureHandler)

	return &DistributionManager{
		selector:       selector,
		poolManager:    poolManager,
		failureHandler: failureHandler,
	}
}

// SelectDeviceForMessage selects a device for message distribution
func (dm *DistributionManager) SelectDeviceForMessage(ctx context.Context, tenantID uuid.UUID) (*Device, error) {
	// Get strategy for tenant
	strategy, err := dm.selector.GetStrategy(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Select device using strategy
	device, err := strategy.SelectDevice()
	if err != nil {
		return nil, err
	}

	return device, nil
}

// GetPoolManager returns the device pool manager
func (dm *DistributionManager) GetPoolManager() *DevicePoolManager {
	return dm.poolManager
}

// GetFailureHandler returns the device failure handler
func (dm *DistributionManager) GetFailureHandler() *DeviceFailureHandler {
	return dm.failureHandler
}

// GetSelector returns the strategy selector
func (dm *DistributionManager) GetSelector() *StrategySelector {
	return dm.selector
}
