package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContactSyncItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	PhoneNumber  string   `json:"phone_number"`
	PhoneNumbers []string `json:"phone_numbers"`
}

type SyncContactsRequest struct {
	DeviceID uuid.UUID          `json:"device_id" binding:"required"`
	Contacts []ContactSyncItem  `json:"contacts"`
}

type SyncContactsResponse struct {
	Synced  int    `json:"synced"`
	Message string `json:"message"`
}

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

// SyncContacts upserts a mobile device's phone directory into the tenant's contacts.
// Each distinct phone number becomes a Contact row keyed by (tenant_id, phone_number).
func (s *Service) SyncContacts(ctx context.Context, tenantID uuid.UUID, req SyncContactsRequest) (int, error) {
	if req.DeviceID != uuid.Nil {
		var device models.PairedDevice
		err := db.Get(ctx).Where("id = ? AND tenant_id = ?", req.DeviceID, tenantID).First(&device).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("device not found for this tenant")
		}
		if err != nil {
			return 0, err
		}
	}

	synced := 0
	for _, item := range req.Contacts {
		name := strings.TrimSpace(item.Name)
		for _, number := range uniqueNumbers(item) {
			if number == "" {
				continue
			}
			if name == "" {
				name = number
			}

			var existing models.Contact
			err := db.Get(ctx).Where("tenant_id = ? AND phone_number = ?", tenantID, number).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				contact := models.Contact{
					ID:          uuid.New(),
					TenantID:    tenantID,
					Name:        name,
					PhoneNumber: number,
				}
				if err := db.Get(ctx).Create(&contact).Error; err != nil {
					return synced, err
				}
				synced++
			case err != nil:
				return synced, err
			default:
				if existing.Name != name {
					existing.Name = name
					if err := db.Get(ctx).Save(&existing).Error; err != nil {
						return synced, err
					}
				}
				synced++
			}
		}
	}
	return synced, nil
}

func uniqueNumbers(item ContactSyncItem) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		digits := normalizePhone(raw)
		if digits == "" {
			return
		}
		if _, ok := seen[digits]; ok {
			return
		}
		seen[digits] = struct{}{}
		out = append(out, digits)
	}
	add(item.PhoneNumber)
	for _, n := range item.PhoneNumbers {
		add(n)
	}
	return out
}

func normalizePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
