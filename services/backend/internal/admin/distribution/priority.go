package distribution

import (
	"sort"
	"sync"
)

// PriorityStrategy implements priority-based device selection
// It selects devices based on priority level (descending) and then by load (ascending)
type PriorityStrategy struct {
	devices []*Device
	mu      sync.RWMutex
}

// NewPriorityStrategy creates a new priority-based distribution strategy
func NewPriorityStrategy(devices []*Device) *PriorityStrategy {
	return &PriorityStrategy{
		devices: devices,
	}
}

// SelectDevice selects the device with the highest priority
// Among devices with equal priority, selects the one with lowest load
// Thread-safe implementation using read-write mutex
func (ps *PriorityStrategy) SelectDevice() (*Device, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if len(ps.devices) == 0 {
		return nil, ErrNoDevicesAvailable
	}

	// Create a copy of devices slice to avoid modifying the original
	devicesCopy := make([]*Device, len(ps.devices))
	copy(devicesCopy, ps.devices)

	// Sort by priority (descending), then by load (ascending)
	sort.Slice(devicesCopy, func(i, j int) bool {
		// First compare by priority (higher priority first)
		if devicesCopy[i].PriorityLevel != devicesCopy[j].PriorityLevel {
			return devicesCopy[i].PriorityLevel > devicesCopy[j].PriorityLevel
		}
		// If priorities are equal, compare by load (lower load first)
		return devicesCopy[i].CurrentLoad < devicesCopy[j].CurrentLoad
	})

	// Return device with highest priority and lowest load
	return devicesCopy[0], nil
}

// UpdateDevices updates the device pool
// Thread-safe operation
func (ps *PriorityStrategy) UpdateDevices(devices []*Device) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.devices = devices
}

// GetDeviceCount returns the number of devices in the pool
func (ps *PriorityStrategy) GetDeviceCount() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return len(ps.devices)
}

// UpdateDeviceLoad updates the current load for a specific device
// This should be called after assigning a message to a device
func (ps *PriorityStrategy) UpdateDeviceLoad(deviceID string, newLoad int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for _, device := range ps.devices {
		if device.ID.String() == deviceID {
			device.CurrentLoad = newLoad
			break
		}
	}
}
