package contact

import (
	"context"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) ListContacts(ctx context.Context, tenantID uuid.UUID) ([]models.Contact, error) {
	var contacts []models.Contact
	if err := db.Get(ctx).Where("tenant_id = ?", tenantID).Find(&contacts).Error; err != nil {
		return nil, err
	}
	return contacts, nil
}

func (s *Service) CreateContact(ctx context.Context, tenantID uuid.UUID, contact *models.Contact) error {
	contact.ID = uuid.New()
	contact.TenantID = tenantID
	return db.Get(ctx).Create(contact).Error
}

func (s *Service) UpdateContact(ctx context.Context, tenantID uuid.UUID, contact *models.Contact) error {
	return db.Get(ctx).Where("id = ? AND tenant_id = ?", contact.ID, tenantID).Save(contact).Error
}

func (s *Service) DeleteContact(ctx context.Context, tenantID uuid.UUID, contactID uuid.UUID) error {
	return db.Get(ctx).Where("id = ? AND tenant_id = ?", contactID, tenantID).Delete(&models.Contact{}).Error
}
