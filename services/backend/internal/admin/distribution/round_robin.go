package distribution

import (
	"errors"
	"sync"
)

var (
	// ErrNoDevicesAvailable is returned when no devices are available for distribution
	ErrNoDevicesAvailable = errors.New("no devices available for distribution")
)

// RoundRobinStrategy implements round-robin device selection
// It distributes messages sequentially across available devices
type RoundRobinStrategy struct {
	devices      []*Device
	currentIndex int
	mu           sync.Mutex
}

// NewRoundRobinStrategy creates a new round-robin distribution strategy
func NewRoundRobinStrategy(devices []*Device) *RoundRobinStrategy {
	return &RoundRobinStrategy{
		devices:      devices,
		currentIndex: 0,
	}
}

// SelectDevice selects the next device using round-robin algorithm
// Thread-safe implementation using mutex to prevent race conditions
func (rr *RoundRobinStrategy) SelectDevice() (*Device, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.devices) == 0 {
		return nil, ErrNoDevicesAvailable
	}

	// Select current device
	device := rr.devices[rr.currentIndex]

	// Move to next device (circular)
	rr.currentIndex = (rr.currentIndex + 1) % len(rr.devices)

	return device, nil
}

// UpdateDevices updates the device pool
// Thread-safe operation
func (rr *RoundRobinStrategy) UpdateDevices(devices []*Device) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.devices = devices
	// Reset index if it's out of bounds
	if rr.currentIndex >= len(rr.devices) {
		rr.currentIndex = 0
	}
}

// GetDeviceCount returns the number of devices in the pool
func (rr *RoundRobinStrategy) GetDeviceCount() int {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	return len(rr.devices)
}
