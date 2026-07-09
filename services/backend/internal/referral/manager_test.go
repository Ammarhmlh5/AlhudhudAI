package referral

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests require a test database connection
// Run with: go test -v ./github.com/al-hudhud/backend/internal/referral/...

func TestGenerateReferralCode(t *testing.T) {
	// Skip if no database connection
	t.Skip("Requires database connection")
	
	config := &ReferralConfig{
		CodePrefix: "TEST",
		CodeLength: 8,
		Enabled:    true,
	}
	
	manager := NewManager(nil, config)
	
	ctx := context.Background()
	tenantID := uuid.New()
	
	code, err := manager.GenerateReferralCode(ctx, tenantID)
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Contains(t, code, "TEST-")
	assert.Len(t, code, len("TEST-")+8)
}

func TestProcessReferral_Success(t *testing.T) {
	t.Skip("Requires database connection")
	
	// Setup
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	referrerID := uuid.New()
	refereeID := uuid.New()
	code := "TEST-REF-001"
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     code,
	}
	
	// Execute
	referral, err := manager.ProcessReferral(ctx, req)
	
	// Assert
	require.NoError(t, err)
	assert.NotNil(t, referral)
	assert.Equal(t, referrerID, referral.ReferrerTenantID)
	assert.Equal(t, refereeID, referral.RefereeTenantID)
	assert.Equal(t, code, referral.ReferralCode)
	assert.Equal(t, ReferralStatusPending, referral.Status)
	assert.False(t, referral.RewardGranted)
}

func TestProcessReferral_SelfReferral(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	tenantID := uuid.New()
	
	req := &CreateReferralRequest{
		ReferrerTenantID: tenantID,
		RefereeTenantID:  tenantID, // Same as referrer
		ReferralCode:     "TEST-REF-002",
	}
	
	_, err := manager.ProcessReferral(ctx, req)
	
	assert.ErrorIs(t, err, ErrSelfReferral)
}

func TestProcessReferral_ImmediateReward(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	referrerID := uuid.New()
	refereeID := uuid.New()
	condition := RewardConditionImmediate
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     "TEST-REF-003",
		RewardCondition:  &condition,
	}
	
	referral, err := manager.ProcessReferral(ctx, req)
	
	require.NoError(t, err)
	assert.True(t, referral.ConditionMet)
	assert.NotNil(t, referral.ConditionMetAt)
	assert.Equal(t, ReferralStatusCompleted, referral.Status)
}

func TestMarkConditionMet(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	// First create a referral
	referrerID := uuid.New()
	refereeID := uuid.New()
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     "TEST-REF-004",
	}
	
	referral, err := manager.ProcessReferral(ctx, req)
	require.NoError(t, err)
	
	// Mark condition as met
	err = manager.MarkConditionMet(ctx, referral.ID)
	require.NoError(t, err)
	
	// Verify
	updated, err := manager.GetReferralByID(ctx, referral.ID)
	require.NoError(t, err)
	assert.True(t, updated.ConditionMet)
	assert.NotNil(t, updated.ConditionMetAt)
	assert.Equal(t, ReferralStatusCompleted, updated.Status)
}

func TestGrantReward_Success(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	// Create a completed referral
	referrerID := uuid.New()
	refereeID := uuid.New()
	condition := RewardConditionImmediate
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     "TEST-REF-005",
		RewardCondition:  &condition,
	}
	
	referral, err := manager.ProcessReferral(ctx, req)
	require.NoError(t, err)
	
	// Grant reward
	grantReq := &GrantRewardRequest{
		ReferralID: referral.ID,
	}
	
	err = manager.GrantReward(ctx, grantReq)
	require.NoError(t, err)
	
	// Verify
	updated, err := manager.GetReferralByID(ctx, referral.ID)
	require.NoError(t, err)
	assert.True(t, updated.RewardGranted)
	assert.NotNil(t, updated.RewardGrantedAt)
	assert.Equal(t, ReferralStatusRewarded, updated.Status)
}

func TestGrantReward_ConditionNotMet(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	// Create a pending referral
	referrerID := uuid.New()
	refereeID := uuid.New()
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     "TEST-REF-006",
	}
	
	referral, err := manager.ProcessReferral(ctx, req)
	require.NoError(t, err)
	
	// Try to grant reward
	grantReq := &GrantRewardRequest{
		ReferralID: referral.ID,
	}
	
	err = manager.GrantReward(ctx, grantReq)
	assert.ErrorIs(t, err, ErrConditionNotMet)
}

func TestGrantReward_AlreadyGranted(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	// Create and grant reward
	referrerID := uuid.New()
	refereeID := uuid.New()
	condition := RewardConditionImmediate
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     "TEST-REF-007",
		RewardCondition:  &condition,
	}
	
	referral, err := manager.ProcessReferral(ctx, req)
	require.NoError(t, err)
	
	grantReq := &GrantRewardRequest{
		ReferralID: referral.ID,
	}
	
	err = manager.GrantReward(ctx, grantReq)
	require.NoError(t, err)
	
	// Try to grant again
	err = manager.GrantReward(ctx, grantReq)
	assert.ErrorIs(t, err, ErrRewardAlreadyGranted)
}

func TestGetReferralStats(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	referrerID := uuid.New()
	
	// Create multiple referrals
	for i := 0; i < 3; i++ {
		refereeID := uuid.New()
		condition := RewardConditionImmediate
		
		req := &CreateReferralRequest{
			ReferrerTenantID: referrerID,
			RefereeTenantID:  refereeID,
			ReferralCode:     uuid.New().String(),
			RewardCondition:  &condition,
		}
		
		referral, err := manager.ProcessReferral(ctx, req)
		require.NoError(t, err)
		
		// Grant reward for first two
		if i < 2 {
			grantReq := &GrantRewardRequest{
				ReferralID: referral.ID,
			}
			err = manager.GrantReward(ctx, grantReq)
			require.NoError(t, err)
		}
	}
	
	// Get stats
	stats, err := manager.GetReferralStats(ctx, referrerID)
	require.NoError(t, err)
	
	assert.Equal(t, 3, stats.TotalReferrals)
	assert.Equal(t, 3, stats.CompletedReferrals)
	assert.Equal(t, 2, stats.RewardedReferrals)
	assert.Equal(t, 0, stats.PendingReferrals)
}

func TestGetReferralPerformance(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	// Get performance
	perf, err := manager.GetReferralPerformance(ctx)
	require.NoError(t, err)
	
	assert.NotNil(t, perf)
	assert.GreaterOrEqual(t, perf.TotalReferrals, 0)
	assert.GreaterOrEqual(t, perf.ConversionRate, 0.0)
	assert.LessOrEqual(t, perf.ConversionRate, 100.0)
}

func TestProcessPendingRewards(t *testing.T) {
	t.Skip("Requires database connection")
	
	manager := NewManager(nil, nil)
	ctx := context.Background()
	
	// Create some completed but unrewarded referrals
	for i := 0; i < 3; i++ {
		referrerID := uuid.New()
		refereeID := uuid.New()
		
		req := &CreateReferralRequest{
			ReferrerTenantID: referrerID,
			RefereeTenantID:  refereeID,
			ReferralCode:     uuid.New().String(),
		}
		
		referral, err := manager.ProcessReferral(ctx, req)
		require.NoError(t, err)
		
		// Mark condition as met
		err = manager.MarkConditionMet(ctx, referral.ID)
		require.NoError(t, err)
	}
	
	// Process pending rewards
	processed, err := manager.ProcessPendingRewards(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, processed, 0)
}

// Benchmark tests
func BenchmarkGenerateReferralCode(b *testing.B) {
	config := &ReferralConfig{
		CodePrefix: "BENCH",
		CodeLength: 8,
		Enabled:    true,
	}
	
	manager := NewManager(nil, config)
	ctx := context.Background()
	tenantID := uuid.New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateReferralCode(ctx, tenantID)
	}
}

// Helper functions for testing
func createTestReferral(t *testing.T, manager *Manager, referrerID, refereeID uuid.UUID) *Referral {
	ctx := context.Background()
	
	req := &CreateReferralRequest{
		ReferrerTenantID: referrerID,
		RefereeTenantID:  refereeID,
		ReferralCode:     uuid.New().String(),
	}
	
	referral, err := manager.ProcessReferral(ctx, req)
	require.NoError(t, err)
	
	return referral
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
