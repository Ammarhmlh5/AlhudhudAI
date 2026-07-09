package distribution

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrInvalidCustomRule is returned when a custom rule is invalid
	ErrInvalidCustomRule = errors.New("invalid custom rule")
)

// CustomRulesStrategy implements custom rule-based device selection
// It supports administrator-defined rules from JSONB configuration
type CustomRulesStrategy struct {
	devices     []*Device
	customRules map[string]interface{}
	mu          sync.RWMutex
}

// NewCustomRulesStrategy creates a new custom rules distribution strategy
func NewCustomRulesStrategy(devices []*Device, customRules map[string]interface{}) *CustomRulesStrategy {
	return &CustomRulesStrategy{
		devices:     devices,
		customRules: customRules,
	}
}

// SelectDevice selects a device based on custom rules
// Thread-safe implementation using read-write mutex
func (crs *CustomRulesStrategy) SelectDevice() (*Device, error) {
	crs.mu.RLock()
	defer crs.mu.RUnlock()

	if len(crs.devices) == 0 {
		return nil, ErrNoDevicesAvailable
	}

	// Apply custom rules to filter and select device
	selectedDevice, err := crs.applyCustomRules()
	if err != nil {
		return nil, err
	}

	return selectedDevice, nil
}

// applyCustomRules applies the custom rules to select a device
// This is a flexible implementation that can be extended based on rule types
func (crs *CustomRulesStrategy) applyCustomRules() (*Device, error) {
	if crs.customRules == nil || len(crs.customRules) == 0 {
		// No custom rules defined, fallback to first available device
		return crs.devices[0], nil
	}

	// Check for rule type
	ruleType, ok := crs.customRules["type"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing or invalid rule type", ErrInvalidCustomRule)
	}

	switch ruleType {
	case "device_name_pattern":
		return crs.selectByDeviceNamePattern()
	case "priority_threshold":
		return crs.selectByPriorityThreshold()
	case "load_threshold":
		return crs.selectByLoadThreshold()
	case "weighted_selection":
		return crs.selectByWeightedRules()
	default:
		return nil, fmt.Errorf("%w: unknown rule type: %s", ErrInvalidCustomRule, ruleType)
	}
}

// selectByDeviceNamePattern selects devices matching a name pattern
func (crs *CustomRulesStrategy) selectByDeviceNamePattern() (*Device, error) {
	pattern, ok := crs.customRules["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing pattern for device_name_pattern rule", ErrInvalidCustomRule)
	}

	for _, device := range crs.devices {
		// Simple pattern matching (can be enhanced with regex)
		if matchesPattern(device.DeviceName, pattern) {
			return device, nil
		}
	}

	return nil, fmt.Errorf("no device matches pattern: %s", pattern)
}

// selectByPriorityThreshold selects devices above a priority threshold
func (crs *CustomRulesStrategy) selectByPriorityThreshold() (*Device, error) {
	threshold, ok := crs.customRules["min_priority"].(float64)
	if !ok {
		return nil, fmt.Errorf("%w: missing min_priority for priority_threshold rule", ErrInvalidCustomRule)
	}

	minPriority := int(threshold)

	for _, device := range crs.devices {
		if device.PriorityLevel >= minPriority {
			return device, nil
		}
	}

	return nil, fmt.Errorf("no device with priority >= %d", minPriority)
}

// selectByLoadThreshold selects devices below a load threshold
func (crs *CustomRulesStrategy) selectByLoadThreshold() (*Device, error) {
	threshold, ok := crs.customRules["max_load"].(float64)
	if !ok {
		return nil, fmt.Errorf("%w: missing max_load for load_threshold rule", ErrInvalidCustomRule)
	}

	maxLoad := int(threshold)

	for _, device := range crs.devices {
		if device.CurrentLoad <= maxLoad {
			return device, nil
		}
	}

	return nil, fmt.Errorf("no device with load <= %d", maxLoad)
}

// selectByWeightedRules selects devices using weighted criteria
func (crs *CustomRulesStrategy) selectByWeightedRules() (*Device, error) {
	weights, ok := crs.customRules["weights"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: missing weights for weighted_selection rule", ErrInvalidCustomRule)
	}

	priorityWeight := getFloatValue(weights, "priority", 0.5)
	loadWeight := getFloatValue(weights, "load", 0.5)

	var bestDevice *Device
	var bestScore float64

	for _, device := range crs.devices {
		// Calculate weighted score (higher is better)
		// Normalize priority (0-5) to 0-1 range
		normalizedPriority := float64(device.PriorityLevel) / 5.0
		// Normalize load (inverse, lower load is better)
		normalizedLoad := 1.0 - (float64(device.CurrentLoad) / float64(device.MaxConcurrentMessages))

		score := (normalizedPriority * priorityWeight) + (normalizedLoad * loadWeight)

		if bestDevice == nil || score > bestScore {
			bestDevice = device
			bestScore = score
		}
	}

	if bestDevice == nil {
		return nil, ErrNoDevicesAvailable
	}

	return bestDevice, nil
}

// UpdateDevices updates the device pool
// Thread-safe operation
func (crs *CustomRulesStrategy) UpdateDevices(devices []*Device) {
	crs.mu.Lock()
	defer crs.mu.Unlock()

	crs.devices = devices
}

// UpdateCustomRules updates the custom rules
// Thread-safe operation
func (crs *CustomRulesStrategy) UpdateCustomRules(rules map[string]interface{}) {
	crs.mu.Lock()
	defer crs.mu.Unlock()

	crs.customRules = rules
}

// GetDeviceCount returns the number of devices in the pool
func (crs *CustomRulesStrategy) GetDeviceCount() int {
	crs.mu.RLock()
	defer crs.mu.RUnlock()

	return len(crs.devices)
}

// Helper functions

// matchesPattern performs simple pattern matching
// Can be enhanced with regex for more complex patterns
func matchesPattern(deviceName, pattern string) bool {
	// Simple contains check for now
	// TODO: Implement regex pattern matching if needed
	return len(pattern) > 0 && len(deviceName) >= len(pattern) && 
		deviceName[:len(pattern)] == pattern
}

// getFloatValue safely extracts a float value from a map
func getFloatValue(m map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return defaultValue
}
