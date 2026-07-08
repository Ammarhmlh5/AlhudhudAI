package distribution

import (
	"sort"
	"sync"
)

// LoadBalancingStrategy implements load-based device selection
// It selects the device with the lowest current load
type LoadBalancingStrategy struct {
	devices []*Device
	mu      sync.RWMutex
}

// NewLoadBalancingStrategy creates a new load balancing distribution strategy
func NewLoadBalancingStrategy(devices []*Device) *LoadBalancingStrategy {
	return &LoadBalancingStrategy{
		devices: devices,
	}
}

// SelectDevice selects the device with the lowest current load
// Thread-safe implementation using read-write mutex
func (lb *LoadBalancingStrategy) SelectDevice() (*Device, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.devices) == 0 {
		return nil, ErrNoDevicesAvailable
	}

	// Create a copy of devices slice to avoid modifying the original
	devicesCopy := make([]*Device, len(lb.devices))
	copy(devicesCopy, lb.devices)

	// Sort by current load (ascending)
	sort.Slice(devicesCopy, func(i, j int) bool {
		return devicesCopy[i].CurrentLoad < devicesCopy[j].CurrentLoad
	})

	// Return device with lowest load
	return devicesCopy[0], nil
}

// UpdateDevices updates the device pool
// Thread-safe operation
func (lb *LoadBalancingStrategy) UpdateDevices(devices []*Device) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.devices = devices
}

// GetDeviceCount returns the number of devices in the pool
func (lb *LoadBalancingStrategy) GetDeviceCount() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	return len(lb.devices)
}

// UpdateDeviceLoad updates the current load for a specific device
// This should be called after assigning a message to a device
func (lb *LoadBalancingStrategy) UpdateDeviceLoad(deviceID string, newLoad int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for _, device := range lb.devices {
		if device.ID.String() == deviceID {
			device.CurrentLoad = newLoad
			break
		}
	}
}
