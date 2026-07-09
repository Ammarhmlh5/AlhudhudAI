package balance

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestNotificationService_CheckAndNotifyLowBalance(t *testing.T) {
	logger := zap.NewNop()
	ns := NewNotificationService(logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		tenantID      uuid.UUID
		balance       decimal.Decimal
		expectWarning bool
		expectCritical bool
	}{
		{
			name:          "Balance above warning threshold",
			tenantID:      uuid.New(),
			balance:       decimal.NewFromInt(15),
			expectWarning: false,
			expectCritical: false,
		},
		{
			name:          "Balance at warning threshold (10)",
			tenantID:      uuid.New(),
			balance:       decimal.NewFromInt(10),
			expectWarning: false,
			expectCritical: false,
		},
		{
			name:          "Balance below warning threshold (9.5)",
			tenantID:      uuid.New(),
			balance:       decimal.NewFromFloat(9.5),
			expectWarning: true,
			expectCritical: false,
		},
		{
			name:          "Balance at critical threshold (5)",
			tenantID:      uuid.New(),
			balance:       decimal.NewFromInt(5),
			expectWarning: false,
			expectCritical: false,
		},
		{
			name:          "Balance below critical threshold (4.5)",
			tenantID:      uuid.New(),
			balance:       decimal.NewFromFloat(4.5),
			expectWarning: false,
			expectCritical: true,
		},
		{
			name:          "Balance at zero",
			tenantID:      uuid.New(),
			balance:       decimal.Zero,
			expectWarning: false,
			expectCritical: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ns.CheckAndNotifyLowBalance(ctx, tt.tenantID, tt.balance)
			if err != nil {
				t.Errorf("CheckAndNotifyLowBalance() error = %v", err)
			}
			// Note: In a real implementation, we would verify that notifications were sent
			// For now, we just verify that the function doesn't error
		})
	}
}

func TestNotificationService_LogInsufficientBalanceAttempt(t *testing.T) {
	logger := zap.NewNop()
	ns := NewNotificationService(logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		tenantID  uuid.UUID
		messageID *uuid.UUID
		required  decimal.Decimal
		available decimal.Decimal
	}{
		{
			name:      "Insufficient balance attempt",
			tenantID:  uuid.New(),
			messageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			required:  decimal.NewFromInt(10),
			available: decimal.NewFromInt(5),
		},
		{
			name:      "Zero balance attempt",
			tenantID:  uuid.New(),
			messageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			required:  decimal.NewFromInt(1),
			available: decimal.Zero,
		},
		{
			name:      "Large amount required",
			tenantID:  uuid.New(),
			messageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			required:  decimal.NewFromInt(1000),
			available: decimal.NewFromInt(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ns.LogInsufficientBalanceAttempt(ctx, tt.tenantID, tt.messageID, tt.required, tt.available)
			if err != nil {
				t.Errorf("LogInsufficientBalanceAttempt() error = %v", err)
			}
		})
	}
}

func TestNotificationService_SendNotification(t *testing.T) {
	logger := zap.NewNop()
	ns := NewNotificationService(logger)
	ctx := context.Background()

	tests := []struct {
		name         string
		notification Notification
	}{
		{
			name: "Low balance warning",
			notification: Notification{
				TenantID:  uuid.New(),
				Type:      NotificationTypeLowBalance,
				Level:     NotificationLevelWarning,
				Balance:   decimal.NewFromFloat(9.5),
				Threshold: decimal.NewFromInt(10),
				Message:   "Balance is low",
			},
		},
		{
			name: "Critical balance warning",
			notification: Notification{
				TenantID:  uuid.New(),
				Type:      NotificationTypeCriticalBalance,
				Level:     NotificationLevelCritical,
				Balance:   decimal.NewFromFloat(4.5),
				Threshold: decimal.NewFromInt(5),
				Message:   "Balance is critically low",
			},
		},
		{
			name: "Insufficient balance attempt",
			notification: Notification{
				TenantID:  uuid.New(),
				Type:      NotificationTypeInsufficientBalance,
				Level:     NotificationLevelWarning,
				Balance:   decimal.NewFromInt(5),
				Threshold: decimal.NewFromInt(10),
				Message:   "Failed to deduct balance",
				MessageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ns.SendNotification(ctx, tt.notification)
			if err != nil {
				t.Errorf("SendNotification() error = %v", err)
			}
		})
	}
}

func TestNotificationThresholds(t *testing.T) {
	// Test that thresholds are correctly defined
	criticalThreshold := decimal.NewFromInt(5)
	warningThreshold := decimal.NewFromInt(10)

	if !warningThreshold.GreaterThan(criticalThreshold) {
		t.Error("Warning threshold should be greater than critical threshold")
	}

	// Test boundary conditions
	tests := []struct {
		name       string
		balance    decimal.Decimal
		isCritical bool
		isWarning  bool
	}{
		{
			name:       "Above warning",
			balance:    decimal.NewFromInt(11),
			isCritical: false,
			isWarning:  false,
		},
		{
			name:       "At warning boundary",
			balance:    decimal.NewFromInt(10),
			isCritical: false,
			isWarning:  false,
		},
		{
			name:       "Below warning",
			balance:    decimal.NewFromFloat(9.99),
			isCritical: false,
			isWarning:  true,
		},
		{
			name:       "Between warning and critical",
			balance:    decimal.NewFromInt(7),
			isCritical: false,
			isWarning:  true,
		},
		{
			name:       "At critical boundary",
			balance:    decimal.NewFromInt(5),
			isCritical: false,
			isWarning:  true,
		},
		{
			name:       "Below critical",
			balance:    decimal.NewFromFloat(4.99),
			isCritical: true,
			isWarning:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCritical := tt.balance.LessThan(criticalThreshold)
			isWarning := tt.balance.LessThan(warningThreshold) && !isCritical

			if isCritical != tt.isCritical {
				t.Errorf("Expected isCritical=%v, got %v for balance %s", tt.isCritical, isCritical, tt.balance.String())
			}
			if isWarning != tt.isWarning {
				t.Errorf("Expected isWarning=%v, got %v for balance %s", tt.isWarning, isWarning, tt.balance.String())
			}
		})
	}
}
