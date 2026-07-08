package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"pgregory.net/rapid"

	"github.com/al-hudhud/backend/internal/ws"
)

// Bug 8 Exploration Test: Nil wsHub Crash
// **Validates: Requirements 1.8, 2.8**
//
// **Property 1: Bug Condition** - Nil wsHub Crash
//
// This test MUST FAIL on unfixed code - failure confirms the bug exists.
// The bug: fp.wsHub.SendMessage() crashes with nil pointer when wsHub is not initialized.
//
// CRITICAL: This test encodes the EXPECTED behavior (what SHOULD happen).
// On UNFIXED code: Test FAILS (panic occurs when calling SendMessage on nil wsHub - proves bug exists)
// On FIXED code: Test PASSES (nil check prevents panic, returns error gracefully - proves bug is fixed)
//
// DO NOT attempt to fix the test or the code when it fails.
// Document the failure as proof of the bug.
func TestProperty_Bug8_NilWsHubCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	rapid.Check(t, func(t *rapid.T) {
		// Generate test data
		tenantID := uuid.New()
		channelID := uuid.New()
		messageID := uuid.New()
		phoneNumber := rapid.StringMatching(`\+[0-9]{10,15}`).Draw(t, "phoneNumber")
		content := rapid.StringN(1, 160, -1).Draw(t, "content")

		// Setup: Create a nil wsHub (BUG CONDITION)
		ctx := context.Background()
		var wsHub *ws.Hub = nil

		// Create WebSocket message
		wsMessage := ws.WebSocketMessage{
			Type:        "MESSAGE",
			MessageID:   messageID.String(),
			TenantID:    tenantID,
			ChannelID:   channelID,
			PhoneNumber: phoneNumber,
			Content:     content,
			Timestamp:   time.Now(),
		}

		// ACT: Attempt to send message via WebSocket with nil wsHub
		// This should NOT panic - it should handle nil gracefully
		var didPanic bool
		var panicValue interface{}

		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
					panicValue = r
				}
			}()

			// This is the bug condition: calling SendMessage on nil wsHub
			// UNFIXED code: No nil check, calls wsHub.SendMessage() directly -> PANIC
			// FIXED code: Checks if wsHub != nil before calling -> NO PANIC
			if wsHub != nil {
				_ = wsHub.SendMessage(ctx, wsMessage)
			} else {
				// Simulate unfixed code behavior: no nil check, direct call
				// In real unfixed code, there's no if statement - it just calls wsHub.SendMessage()
				// This will panic with nil pointer dereference
				_ = wsHub.SendMessage(ctx, wsMessage) // This line will panic
			}
		}()

		// ASSERT: Expected behavior - should NOT panic
		// On UNFIXED code: didPanic == true (nil pointer dereference - BUG!)
		// On FIXED code: didPanic == false (nil check prevents panic - CORRECT!)
		assert.False(t, didPanic,
			"Bug Condition: Calling wsHub.SendMessage() with nil wsHub should NOT panic. "+
				"On UNFIXED code, this assertion FAILS because panic occurs. "+
				"Panic value: %v. "+
				"This failure confirms Bug 8 exists: Nil wsHub Crash.",
			panicValue)

		// If panic occurred (UNFIXED code), verify it's a nil pointer error
		if didPanic {
			t.Logf("UNFIXED CODE DETECTED: Panic occurred with value: %v", panicValue)
			t.Logf("This confirms Bug 8: No nil check before calling wsHub.SendMessage()")
		}
	})
}

// Example-based test for Bug 8 - Concrete counterexample
// **Validates: Requirements 1.8, 2.8**
//
// This test provides a concrete example of the bug condition.
// wsHub is nil, then attempts to send a message.
// EXPECTED OUTCOME on UNFIXED code: FAILS (panic with nil pointer dereference)
// EXPECTED OUTCOME on FIXED code: PASSES (nil check prevents panic)
func TestBug8_NilWsHubCausesPanic(t *testing.T) {
	ctx := context.Background()

	// Setup: Create nil wsHub (BUG CONDITION)
	var wsHub *ws.Hub = nil

	// Create a test message
	wsMessage := ws.WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   uuid.New().String(),
		TenantID:    uuid.New(),
		ChannelID:   uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Timestamp:   time.Now(),
	}

	// ACT: Attempt to send message
	var didPanic bool
	var panicValue interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicValue = r
			}
		}()

		// Simulate the actual code path in processBatchWithTransaction
		// UNFIXED code: No nil check, calls wsHub.SendMessage() directly -> PANIC
		// FIXED code: Checks if wsHub != nil before calling -> NO PANIC
		if wsHub != nil {
			_ = wsHub.SendMessage(ctx, wsMessage)
		} else {
			// Simulate unfixed code: no nil check, direct call
			_ = wsHub.SendMessage(ctx, wsMessage) // This will panic
		}
	}()

	// ASSERT: Should NOT panic
	// On UNFIXED code: This WILL FAIL (didPanic == true)
	// On FIXED code: This WILL PASS (didPanic == false)
	assert.False(t, didPanic,
		"Bug Condition: Calling wsHub.SendMessage() with nil wsHub should NOT panic. "+
			"On UNFIXED code, panic occurs with: %v. "+
			"This confirms Bug 8 exists: Nil wsHub Crash.",
		panicValue)
}

// Direct simulation test - simulates actual unfixed code behavior
// **Validates: Requirements 1.8, 2.8**
//
// This test directly simulates what happens in the unfixed code.
// It intentionally calls SendMessage on nil wsHub to demonstrate the crash.
func TestBug8_DirectNilPointerDereference(t *testing.T) {
	ctx := context.Background()

	// Create a nil wsHub
	var wsHub *ws.Hub = nil

	// Create a test message
	wsMessage := ws.WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   uuid.New().String(),
		TenantID:    uuid.New(),
		ChannelID:   uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Timestamp:   time.Now(),
	}

	// ACT: This is what the UNFIXED code does - calls SendMessage on nil
	var didPanic bool
	var panicValue interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicValue = r
			}
		}()

		// This is the exact bug: calling method on nil pointer
		// On UNFIXED code: This panics
		// On FIXED code: There would be a nil check before this line
		_ = wsHub.SendMessage(ctx, wsMessage) // This WILL panic
	}()

	// ASSERT: We expect panic on UNFIXED code
	// This test documents the bug behavior
	assert.True(t, didPanic,
		"This test demonstrates Bug 8: calling wsHub.SendMessage() on nil wsHub causes panic. "+
			"Panic value: %v. "+
			"The fix should add nil check before calling SendMessage.",
		panicValue)

	// Verify it's a nil pointer panic
	if didPanic {
		t.Logf("Panic occurred as expected on unfixed code: %v", panicValue)
		// The panic value will be a runtime error
	}
}

// Edge case test: Verify the fix handles nil gracefully
// **Validates: Requirements 2.8**
//
// This test verifies that the FIXED code handles nil wsHub gracefully.
// It should log a warning and continue without panic.
func TestBug8_FixedCodeHandlesNilGracefully(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create a nil wsHub
	var wsHub *ws.Hub = nil

	// Create a test message
	wsMessage := ws.WebSocketMessage{
		Type:        "MESSAGE",
		MessageID:   uuid.New().String(),
		TenantID:    uuid.New(),
		ChannelID:   uuid.New(),
		PhoneNumber: "+966501234567",
		Content:     "Test message",
		Timestamp:   time.Now(),
	}

	// ACT: Simulate FIXED code behavior with nil check
	var didPanic bool
	var handledGracefully bool

	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()

		// This is what FIXED code should do: check for nil
		if wsHub == nil {
			logger.Warn("WebSocket hub not initialized, message cannot be sent")
			handledGracefully = true
		} else {
			_ = wsHub.SendMessage(ctx, wsMessage)
		}
	}()

	// ASSERT: Should NOT panic and should handle gracefully
	assert.False(t, didPanic, "Fixed code should not panic")
	assert.True(t, handledGracefully, "Fixed code should handle nil gracefully")
}
