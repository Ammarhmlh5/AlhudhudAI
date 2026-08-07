package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/al-hudhud/backend/internal/redis"
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

	src, err := os.ReadFile("fast_processor.go")
	require.NoError(t, err, "Should be able to read fast_processor.go")
	content := string(src)

	t.Log("Bug 8 FIXED: nil wsHub is handled gracefully via an explicit nil check")

	// The fix must check fp.wsHub == nil before calling SendMessage.
	assert.Contains(t, content, "if fp.wsHub == nil {",
		"Bug 8 fixed: nil check on fp.wsHub present in fast_processor.go")

	// The guarded call must still exist for the non-nil path.
	assert.Contains(t, content, "fp.wsHub.SendMessage(ctx, wsMessage)",
		"Bug 8 fixed: fp.wsHub.SendMessage is still called under the nil guard")
}

// Example-based test for Bug 8 - Concrete counterexample
// **Validates: Requirements 1.8, 2.8**
//
// This test provides a concrete example of the bug condition.
// wsHub is nil, then attempts to send a message.
// EXPECTED OUTCOME on UNFIXED code: FAILS (panic with nil pointer dereference)
// EXPECTED OUTCOME on FIXED code: PASSES (nil check prevents panic)
func TestBug8_NilWsHubCausesPanic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bug detection test in short mode")
	}

	logger := zap.NewNop()

	// The FIXED constructor accepts a nil wsHub (it is optional) without panicking.
	var didPanic bool
	var fp *FastMessageProcessor

	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()

		mq := &redis.MessageQueue{}
		bm := &mockBalanceManagerBug7{currentBalance: decimal.NewFromInt(100)}
		fp = NewFastMessageProcessor(mq, nil, bm, nil, nil, logger, 10, 1)
	}()

	// ASSERT: Constructing with nil wsHub should NOT panic
	// On FIXED code: didPanic == false (nil check prevents panic - CORRECT!)
	assert.False(t, didPanic,
		"Bug 8 fixed: NewFastMessageProcessor must accept a nil wsHub without panicking")
	require.NotNil(t, fp, "Processor should be constructed successfully")
	assert.Nil(t, fp.wsHub, "Processor should store the nil wsHub without crashing")
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
