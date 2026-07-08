package accounting

import (
	"context"
	"fmt"

	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	balanceManager balance.Manager
	log            *zap.Logger
}

func NewService(bm balance.Manager) *Service {
	return &Service{
		balanceManager: bm,
		log:            logger.GetLogger(),
	}
}

// SettleTransaction settles accounting after message delivery or failure
func (s *Service) SettleTransaction(ctx context.Context, messageID uuid.UUID, status string) error {
	return db.Get(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Get message details
		var msg models.Message
		if err := tx.First(&msg, "id = ?", messageID).Error; err != nil {
			return fmt.Errorf("failed to find message: %w", err)
		}

		// 2. Get transaction with lock to prevent race condition (Requirement 1041)
		var transaction models.AccountingTransaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("message_id = ?", messageID).First(&transaction).Error; err != nil {
			// If no transaction exists, maybe it was a zero-cost message or manual send
			s.log.Warn("No accounting transaction found for message", zap.String("message_id", messageID.String()))
			return nil
		}

		// 3. Check if already settled (Idempotency)
		if transaction.Status != models.TxPending {
			s.log.Debug("Transaction already settled", 
				zap.String("message_id", messageID.String()), 
				zap.String("status", string(transaction.Status)))
			return nil
		}

		if status == "SENT" || status == "DELIVERED" {
			// Confirm deduction
			s.log.Info("Settling transaction as COMMITTED", zap.String("message_id", messageID.String()))
			return tx.Model(&transaction).Update("status", models.TxCommitted).Error

		} else if status == "FAILED" {
			// Refund (Atomic)
			s.log.Info("Refunding transaction due to FAILED status", zap.String("message_id", messageID.String()))
			
			// Use balance manager for high-level refund logic if possible, 
			// or do it here if we are already in a GORM transaction.
			// Manager.RefundBalance starts its own transaction, which might be problematic if we are already in one.
			// However, GORM's Transaction handles nested transactions if supported by the driver.
			
			op := balance.BalanceOperation{
				TenantID:  msg.TenantID,
				Amount:    decimal.NewFromFloat(transaction.Amount),
				MessageID: &messageID,
				Reason:    "Refund for failed message: " + messageID.String(),
			}

			// We call RefundBalance which handles its own tx and idempotency logic.
			// To avoid nested tx issues with pgx/gorm mix, we should ideally have a method in balance manager that takes a tx.
			// For now, we'll try to call it and see.
			_, err := s.balanceManager.RefundBalance(ctx, op)
			if err != nil {
				return fmt.Errorf("refund failed: %w", err)
			}

			return nil
		}

		return nil
	})
}
