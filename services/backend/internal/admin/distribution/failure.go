package distribution

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeviceFailureHandler manages device failures and exclusions
// It excludes failed devices for a configured duration and automatically re-includes them
type DeviceFailureHandler struct {
	excludedDevices map[uuid.UUID]time.Time
	excludeDuration time.Duration
	mu              sync.RWMutex
}

// NewDeviceFailureHandler creates a new device failure handler
// Default exclusion duration is 5 minutes
func NewDeviceFailureHandler() *DeviceFailureHandler {
	return &DeviceFailureHandler{
		excludedDevices: make(map[uuid.UUID]time.Time),
		excludeDuration: 5 * time.Minute,
	}
}

// NewDeviceFailureHandlerWithDuration creates a new device failure handler with custom duration
func NewDeviceFailureHandlerWithDuration(duration time.Duration) *DeviceFailureHandler {
	return &DeviceFailureHandler{
		excludedDevices: make(map[uuid.UUID]time.Time),
		excludeDuration: duration,
	}
}

// OnDeviceFailure marks a device as failed and excludes it from distribution
// The device will be excluded for the configured duration (default: 5 minutes)
func (dfh *DeviceFailureHandler) OnDeviceFailure(deviceID uuid.UUID) {
	dfh.mu.Lock()
	defer dfh.mu.Unlock()

	excludeUntil := time.Now().Add(dfh.excludeDuration)
	dfh.excludedDevices[deviceID] = excludeUntil
}

// IsDeviceExcluded checks if a device is currently excluded
// Automatically removes the device from exclusion list if the exclusion period has expired
func (dfh *DeviceFailureHandler) IsDeviceExcluded(deviceID uuid.UUID) bool {
	dfh.mu.RLock()
	excludeUntil, exists := dfh.excludedDevices[deviceID]
	dfh.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if exclusion period has expired
	if time.Now().After(excludeUntil) {
		// Remove from exclusion list (auto-reinclude)
		dfh.mu.Lock()
		delete(dfh.excludedDevices, deviceID)
		dfh.mu.Unlock()
		return false
	}

	return true
}

// GetExcludedDevices returns a list of currently excluded device IDs
func (dfh *DeviceFailureHandler) GetExcludedDevices() []uuid.UUID {
	dfh.mu.RLock()
	defer dfh.mu.RUnlock()

	excluded := make([]uuid.UUID, 0, len(dfh.excludedDevices))
	now := time.Now()

	for deviceID, excludeUntil := range dfh.excludedDevices {
		if now.Before(excludeUntil) {
			excluded = append(excluded, deviceID)
		}
	}

	return excluded
}

// GetExclusionExpiry returns the expiry time for a device's exclusion
// Returns nil if the device is not excluded
func (dfh *DeviceFailureHandler) GetExclusionExpiry(deviceID uuid.UUID) *time.Time {
	dfh.mu.RLock()
	defer dfh.mu.RUnlock()

	if excludeUntil, exists := dfh.excludedDevices[deviceID]; exists {
		if time.Now().Before(excludeUntil) {
			return &excludeUntil
		}
	}

	return nil
}

// ReincludeDevice manually re-includes a device before its exclusion period expires
func (dfh *DeviceFailureHandler) ReincludeDevice(deviceID uuid.UUID) {
	dfh.mu.Lock()
	defer dfh.mu.Unlock()

	delete(dfh.excludedDevices, deviceID)
}

// CleanupExpiredExclusions removes all expired exclusions from the map
// This should be called periodically to prevent memory leaks
func (dfh *DeviceFailureHandler) CleanupExpiredExclusions() int {
	dfh.mu.Lock()
	defer dfh.mu.Unlock()

	now := time.Now()
	removed := 0

	for deviceID, excludeUntil := range dfh.excludedDevices {
		if now.After(excludeUntil) {
			delete(dfh.excludedDevices, deviceID)
			removed++
		}
	}

	return removed
}

// GetExcludedDeviceCount returns the number of currently excluded devices
func (dfh *DeviceFailureHandler) GetExcludedDeviceCount() int {
	dfh.mu.RLock()
	defer dfh.mu.RUnlock()

	count := 0
	now := time.Now()

	for _, excludeUntil := range dfh.excludedDevices {
		if now.Before(excludeUntil) {
			count++
		}
	}

	return count
}

// SetExcludeDuration updates the exclusion duration
// This affects only new exclusions, not existing ones
func (dfh *DeviceFailureHandler) SetExcludeDuration(duration time.Duration) {
	dfh.mu.Lock()
	defer dfh.mu.Unlock()

	dfh.excludeDuration = duration
}

// GetExcludeDuration returns the current exclusion duration
func (dfh *DeviceFailureHandler) GetExcludeDuration() time.Duration {
	dfh.mu.RLock()
	defer dfh.mu.RUnlock()

	return dfh.excludeDuration
}

// FilterAvailableDevices filters out excluded devices from a device list
// Returns only devices that are not currently excluded
func (dfh *DeviceFailureHandler) FilterAvailableDevices(devices []*Device) []*Device {
	available := make([]*Device, 0, len(devices))

	for _, device := range devices {
		if !dfh.IsDeviceExcluded(device.ID) {
			available = append(available, device)
		}
	}

	return available
}
