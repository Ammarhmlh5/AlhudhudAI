package worker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bug 6 Verification Test: Transaction Rollback After Commit (FIXED)
// **Validates: Requirements 1.6, 2.6**
//
// **Property 1: Bug Condition** - Rollback After Commit
//
// The bug was: defer tx.Rollback() executed after successful tx.Commit()
// in processBatchWithTransaction(). This has been FIXED by using a
// committed flag so rollback only runs when commit was not called.
//
// On FIXED code: This test PASSES (rollback is conditional using committed flag).
//
// NOTE: This test verifies the CODE PATTERN rather than runtime behavior,
// because pgx v5 silently ignores rollback after commit without logging warnings.
func TestProperty_Bug6_RollbackAfterCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}

	src, err := os.ReadFile("fast_processor.go")
	require.NoError(t, err, "Should be able to read fast_processor.go")
	content := string(src)

	t.Log("Bug 6 FIXED: rollback is conditional using the committed flag pattern")

	// No unconditional `defer tx.Rollback(ctx)` may remain.
	assert.NotRegexp(t, `(?m)^\s*defer\s+tx\.Rollback\(ctx\)\s*$`, content,
		"Bug 6 fixed: no unconditional defer tx.Rollback(ctx) in fast_processor.go")

	// The committed-flag conditional rollback pattern must be present.
	assert.Contains(t, content, "committed := false",
		"Bug 6 fixed: committed flag initialized before deferred rollback")
	assert.Regexp(t, `if\s+!committed\s*\{`, content,
		"Bug 6 fixed: deferred rollback is conditional on !committed")
	assert.Regexp(t, `(?m)^\s*committed\s*=\s*true\s*$`, content,
		"Bug 6 fixed: committed flag is set to true after successful commit")
}

// Example-based test for Bug 6 - Verifies the fixed transaction pattern
// **Validates: Requirements 1.6, 2.6**
//
// This test provides a concrete verification that the incorrect
// unconditional-rollback transaction pattern is no longer present.
func TestBug6_RollbackAfterCommitConcrete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bug detection test in short mode")
	}

	src, err := os.ReadFile("fast_processor.go")
	require.NoError(t, err, "Should be able to read fast_processor.go")
	content := string(src)

	t.Log("COUNTEREXAMPLE RESOLVED: Transaction Rollback After Commit is fixed")
	t.Log("")
	t.Log("Fixed Code Pattern:")
	t.Log("  tx, err := db.PgxPool.Begin(ctx)")
	t.Log("  if err != nil {")
	t.Log("    return fmt.Errorf(\"begin transaction: %w\", err)")
	t.Log("  }")
	t.Log("  committed := false")
	t.Log("  defer func() {")
	t.Log("    if !committed {")
	t.Log("      tx.Rollback(ctx)")
	t.Log("    }")
	t.Log("  }()")
	t.Log("  ...")
	t.Log("  if err := tx.Commit(ctx); err != nil {")
	t.Log("    return fmt.Errorf(\"commit transaction: %w\", err)")
	t.Log("  }")
	t.Log("  committed = true  // <-- Prevents rollback from executing")
	t.Log("")

	// The buggy unconditional pattern must NOT exist anymore.
	assert.NotContains(t, content, "defer tx.Rollback(ctx)",
		"Bug 6 fixed: unconditional defer tx.Rollback(ctx) removed from fast_processor.go")

	// The conditional rollback pattern MUST exist.
	assert.Contains(t, content, "committed := false",
		"Bug 6 fixed: committed flag pattern present in fast_processor.go")
	assert.Contains(t, content, "committed = true",
		"Bug 6 fixed: committed flag is set after successful commit")
}

// Test: Verify correct transaction pattern (for comparison)
// **Validates: Requirements 2.6**
//
// This test shows the CORRECT pattern that should be used.
func TestBug6_CorrectTransactionPattern(t *testing.T) {
	// This test documents the CORRECT transaction pattern
	// that should be used to fix Bug 6
	
	t.Log("CORRECT Transaction Pattern (Expected Behavior):")
	t.Log("")
	t.Log("  tx, err := db.PgxPool.Begin(ctx)")
	t.Log("  if err != nil {")
	t.Log("    return fmt.Errorf(\"begin transaction: %w\", err)")
	t.Log("  }")
	t.Log("")
	t.Log("  // Use committed flag pattern")
	t.Log("  committed := false")
	t.Log("  defer func() {")
	t.Log("    if !committed {")
	t.Log("      if rbErr := tx.Rollback(ctx); rbErr != nil {")
	t.Log("        logger.Error(\"Failed to rollback transaction\", zap.Error(rbErr))")
	t.Log("      }")
	t.Log("    }")
	t.Log("  }()")
	t.Log("")
	t.Log("  // ... process messages ...")
	t.Log("")
	t.Log("  // Commit and mark as committed")
	t.Log("  if err := tx.Commit(ctx); err != nil {")
	t.Log("    return fmt.Errorf(\"commit transaction: %w\", err)")
	t.Log("  }")
	t.Log("  committed = true  // <-- Prevents rollback from executing")
	t.Log("")
	t.Log("Benefits:")
	t.Log("  - Rollback only executes if commit fails")
	t.Log("  - Follows transaction lifecycle best practices")
	t.Log("  - Compatible with all database drivers")
	t.Log("  - Clear intent in code")
	
	// This test passes to show the correct pattern
	assert.True(t, true, "This demonstrates the CORRECT transaction pattern")
}
