package messages

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Message struct {
	ID             uuid.UUID       `json:"id"`
	MessageID      string          `json:"message_id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	PhoneNumber    string          `json:"phone_number"`
	Content        string          `json:"content"`
	Status         string          `json:"status"`
	Priority       string          `json:"priority"`
	IdempotencyKey string          `json:"idempotency_key"`
	PartsCount     int             `json:"parts_count"`
	CharCount      int             `json:"char_count"`
	Encoding       string          `json:"encoding"`
	Cost           decimal.Decimal `json:"cost"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Repository interface {
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*Message, error)
	GetMessagesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Message, error)
	UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status string) error
	UpdateMessageDeliveryTime(ctx context.Context, messageID uuid.UUID, deliveredAt time.Time) error
	IncrementAttempts(ctx context.Context, messageID uuid.UUID) error
	SetMessageError(ctx context.Context, messageID uuid.UUID, errorMsg string) error
}

type Manager struct {
	repo       Repository
	createdIDs map[string]string // idempotency_key -> message_id
}

func NewManager(repo Repository) *Manager {
	return &Manager{
		repo:       repo,
		createdIDs: make(map[string]string),
	}
}

var phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

func (m *Manager) CreateMessage(ctx context.Context, msg *Message) (*Message, error) {
	if msg.IdempotencyKey != "" {
		if existingID, ok := m.createdIDs[msg.IdempotencyKey]; ok {
			return &Message{MessageID: existingID, Status: "QUEUED"}, nil
		}
	}

	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.MessageID == "" {
		msg.MessageID = uuid.New().String()
	}
	msg.Status = "QUEUED"
	msg.CreatedAt = time.Now()

	if m.repo != nil {
		if err := m.repo.CreateMessage(ctx, msg); err != nil {
			return nil, err
		}
	}

	if msg.IdempotencyKey != "" {
		m.createdIDs[msg.IdempotencyKey] = msg.MessageID
	}

	return msg, nil
}

func (m *Manager) ValidatePhoneNumber(phone string) error {
	if !phoneRegex.MatchString(phone) {
		return errors.New("invalid phone number format")
	}
	return nil
}

func (m *Manager) UpdateStatus(ctx context.Context, messageID uuid.UUID, status string) error {
	if m.repo != nil {
		return m.repo.UpdateMessageStatus(ctx, messageID, status)
	}
	return nil
}

func (m *Manager) MarkDelivered(ctx context.Context, messageID uuid.UUID) error {
	if m.repo != nil {
		return m.repo.UpdateMessageDeliveryTime(ctx, messageID, time.Now())
	}
	return nil
}

func (m *Manager) RecordFailure(ctx context.Context, messageID uuid.UUID, errorMsg string) error {
	if m.repo != nil {
		if err := m.repo.IncrementAttempts(ctx, messageID); err != nil {
			return err
		}
		return m.repo.SetMessageError(ctx, messageID, errorMsg)
	}
	return nil
}

func (m *Manager) GetMessagesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Message, error) {
	if m.repo != nil {
		return m.repo.GetMessagesByTenant(ctx, tenantID, limit, offset)
	}
	return nil, nil
}

func (m *Manager) CalculateCost(country string, parts int) (decimal.Decimal, error) {
	switch country {
	case "SA":
		return decimal.NewFromFloat(0.05 * float64(parts)), nil
	default:
		return decimal.Zero, errors.New("unknown country")
	}
}
