package subscription

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	repo interface {
		CreateSubscription(ctx context.Context, sub *Subscription) error
		GetSubscriptionByID(ctx context.Context, subID uuid.UUID) (*Subscription, error)
		GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)
		UpdateSubscriptionStatus(ctx context.Context, subID uuid.UUID, status string) error
		CancelSubscription(ctx context.Context, subID uuid.UUID, reason string) error
		GetSubscriptionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]Subscription, error)
	}
}

func NewService(repo interface {
	CreateSubscription(ctx context.Context, sub *Subscription) error
	GetSubscriptionByID(ctx context.Context, subID uuid.UUID) (*Subscription, error)
	GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error)
	UpdateSubscriptionStatus(ctx context.Context, subID uuid.UUID, status string) error
	CancelSubscription(ctx context.Context, subID uuid.UUID, reason string) error
	GetSubscriptionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]Subscription, error)
}) *Service {
	return &Service{repo: repo}
}

var ErrNoActiveSubscription = &SubscriptionError{
	Type:    ErrorSubscriptionNotFound,
	Message: "no active subscription found",
}

func (s *Service) CreateSubscription(ctx context.Context, sub *Subscription) (*Subscription, error) {
	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) ActivateSubscription(ctx context.Context, subID uuid.UUID) error {
	return s.repo.UpdateSubscriptionStatus(ctx, subID, "ACTIVE")
}

func (s *Service) GetActiveSubscription(ctx context.Context, tenantID uuid.UUID) (*Subscription, error) {
	return s.repo.GetActiveSubscription(ctx, tenantID)
}

func (s *Service) CancelSubscription(ctx context.Context, subID uuid.UUID, reason string) error {
	return s.repo.CancelSubscription(ctx, subID, reason)
}

func (s *Service) RenewSubscription(ctx context.Context, subID uuid.UUID, renewalPeriod time.Duration) error {
	sub, err := s.repo.GetSubscriptionByID(ctx, subID)
	if err != nil {
		return err
	}
	return s.repo.UpdateSubscriptionStatus(ctx, subID, string(sub.Status))
}

func (s *Service) CheckAutoRenewal(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, tenantID)
	if err != nil {
		return false, err
	}
	return sub.AutoRenew && sub.EndDate.Before(time.Now().Add(7*24*time.Hour)), nil
}

func (s *Service) UpgradePackage(ctx context.Context, subID uuid.UUID, newPackageID int64, upgradePrice decimal.Decimal) error {
	return s.ActivateSubscription(ctx, subID)
}

func (s *Service) GetSubscriptionHistory(ctx context.Context, tenantID uuid.UUID) ([]Subscription, error) {
	return s.repo.GetSubscriptionsByTenant(ctx, tenantID)
}

func (s *Service) ValidateSubscription(sub *Subscription) error {
	if sub.Status == StatusCancelled {
		return ErrNoActiveSubscription
	}
	if time.Now().After(sub.EndDate) {
		return ErrNoActiveSubscription
	}
	return nil
}
