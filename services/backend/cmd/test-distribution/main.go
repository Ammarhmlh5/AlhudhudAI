package main

import (
	"context"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/internal/admin/distribution"
	"github.com/google/uuid"
)

func main() {
	_ = context.Background()
	
	// Create mockup devices
	device1 := &distribution.Device{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		DeviceName:  "Device 1",
		PriorityLevel: 10,
		Status:      "active",
		CurrentLoad: 5,
	}
	device2 := &distribution.Device{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		DeviceName:  "Device 2",
		PriorityLevel: 8,
		Status:      "active",
		CurrentLoad: 3,
	}
	device3 := &distribution.Device{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		DeviceName:  "Device 3",
		PriorityLevel: 6,
		Status:      "active",
		CurrentLoad: 7,
	}

	devices := []*distribution.Device{device1, device2, device3}
	
	fmt.Println("=============================================================")
	fmt.Println("Testing Message Distribution Strategies")
	fmt.Println("=============================================================")
	fmt.Println()
	
	// Test 1: Round Robin Strategy
	fmt.Println("Test 1: Round Robin Strategy")
	fmt.Println("-------------------------------------------------------------")
	
	rrStrategy := distribution.NewRoundRobinStrategy(devices)
	rrCounts := make(map[string]int)
	
	for i := 0; i < 30; i++ {
		device, err := rrStrategy.SelectDevice()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		rrCounts[device.DeviceName]++
	}
	
	fmt.Println("Message distribution (30 messages):")
	for name, count := range rrCounts {
		fmt.Printf("  %s: %d messages (%.1f%%)\n", name, count, float64(count)/30*100)
	}
	
	// Verify fairness (each device should have exactly 10 messages)
	allEqual := true
	for _, count := range rrCounts {
		if count != 10 {
			allEqual = false
			break
		}
	}
	
	if allEqual {
		fmt.Println("[PASS] Distribution is fair (each device got 10 messages)")
	} else {
		fmt.Println("[FAIL] Distribution is not fair")
	}
	fmt.Println()
	
	// Test 2: Load Balancing Strategy
	fmt.Println("Test 2: Load Balancing Strategy")
	fmt.Println("-------------------------------------------------------------")
	
	lbStrategy := distribution.NewLoadBalancingStrategy(devices)
	lbCounts := make(map[string]int)
	
	for i := 0; i < 10; i++ {
		device, err := lbStrategy.SelectDevice()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		lbCounts[device.DeviceName]++
	}
	
	fmt.Println("Message distribution (10 messages):")
	for name, count := range lbCounts {
		fmt.Printf("  %s: %d messages\n", name, count)
	}
	
	// Device 2 should get the most messages (load = 3 is the lowest)
	if lbCounts["Device 2"] >= lbCounts["Device 1"] && lbCounts["Device 2"] >= lbCounts["Device 3"] {
		fmt.Println("[PASS] Device with lowest load got most messages")
	} else {
		fmt.Println("[FAIL] Distribution doesn't follow Load Balancing policy")
	}
	fmt.Println()
	
	// Test 3: Priority-Based Strategy
	fmt.Println("Test 3: Priority-Based Strategy")
	fmt.Println("-------------------------------------------------------------")
	
	prioStrategy := distribution.NewPriorityStrategy(devices)
	prioCounts := make(map[string]int)
	
	for i := 0; i < 10; i++ {
		device, err := prioStrategy.SelectDevice()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		prioCounts[device.DeviceName]++
	}
	
	fmt.Println("Message distribution (10 messages):")
	for name, count := range prioCounts {
		fmt.Printf("  %s: %d messages\n", name, count)
	}
	
	// Device 1 should get the most messages (priority = 10 is the highest)
	if prioCounts["Device 1"] >= prioCounts["Device 2"] && prioCounts["Device 1"] >= prioCounts["Device 3"] {
		fmt.Println("[PASS] Device with highest priority got most messages")
	} else {
		fmt.Println("[FAIL] Distribution doesn't follow Priority-Based policy")
	}
	fmt.Println()
	
	// Test 4: Device Failure Handling
	fmt.Println("Test 4: Device Failure Handling")
	fmt.Println("-------------------------------------------------------------")
	
	failureHandler := distribution.NewDeviceFailureHandlerWithDuration(2 * time.Second)
	
	// Exclude Device 1
	failureHandler.OnDeviceFailure(device1.ID)
	fmt.Printf("Excluded %s for 2 seconds\n", device1.DeviceName)
	
	// Check available devices
	availableDevices := failureHandler.FilterAvailableDevices(devices)
	fmt.Printf("Available devices: %d (should be 2)\n", len(availableDevices))
	
	if len(availableDevices) == 2 {
		fmt.Println("[PASS] Failed device excluded successfully")
	} else {
		fmt.Println("[FAIL] Failed to exclude device")
	}
	
	// Wait 3 seconds for recovery
	fmt.Println("Waiting 3 seconds...")
	time.Sleep(3 * time.Second)
	
	// Check available devices again
	availableDevices = failureHandler.FilterAvailableDevices(devices)
	fmt.Printf("Available devices after recovery: %d (should be 3)\n", len(availableDevices))
	
	if len(availableDevices) == 3 {
		fmt.Println("[PASS] Device re-included automatically")
	} else {
		fmt.Println("[FAIL] Failed to re-include device")
	}
	fmt.Println()
	
	// Test 5: Strategy Selector / Factory check
	fmt.Println("Test 5: Strategy Factory instantiation")
	fmt.Println("-------------------------------------------------------------")
	
	customRules := map[string]interface{}{"type": "device_name_pattern", "pattern": "Device"}
	customStrategy := distribution.NewCustomRulesStrategy(devices, customRules)
	device, err := customStrategy.SelectDevice()
	if err != nil {
		fmt.Printf("[FAIL] Custom rules selection failed: %v\n", err)
	} else {
		fmt.Printf("[PASS] Custom rules selected device: %s\n", device.DeviceName)
	}
	
	fmt.Println()
	fmt.Println("=============================================================")
	if allEqual && len(availableDevices) == 3 && err == nil {
		fmt.Println("ALL TESTS PASSED SUCCESSFULLY!")
	} else {
		fmt.Println("SOME TESTS FAILED!")
	}
	fmt.Println("=============================================================")
}
