package channel

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// EmailInboxRepository handles database operations for EmailInbox records.
type EmailInboxRepository struct {
	db *gorm.DB
}

// NewEmailInboxRepository creates a new EmailInboxRepository.
func NewEmailInboxRepository(db *gorm.DB) *EmailInboxRepository {
	return &EmailInboxRepository{db: db}
}

// Create inserts a new EmailInbox and populates its generated ID.
func (r *EmailInboxRepository) Create(ctx context.Context, inbox *models.EmailInbox) error {
	if err := r.db.WithContext(ctx).Create(inbox).Error; err != nil {
		return fmt.Errorf("creating email inbox: %w", err)
	}
	return nil
}

// FindByID retrieves an EmailInbox by primary key.
// Returns nil, nil when no record exists.
func (r *EmailInboxRepository) FindByID(ctx context.Context, id string) (*models.EmailInbox, error) {
	var inbox models.EmailInbox
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&inbox).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding email inbox %s: %w", id, err)
	}
	return &inbox, nil
}

// ListByOrg returns all non-deleted EmailInbox records for the given org,
// ordered by creation time ascending.
func (r *EmailInboxRepository) ListByOrg(ctx context.Context, orgID string) ([]models.EmailInbox, error) {
	var inboxes []models.EmailInbox
	if err := r.db.WithContext(ctx).
		Where("org_id = ? AND deleted_at IS NULL", orgID).
		Order("created_at ASC").
		Find(&inboxes).Error; err != nil {
		return nil, fmt.Errorf("listing email inboxes for org %s: %w", orgID, err)
	}
	return inboxes, nil
}

// Save persists all fields of an existing EmailInbox (upsert-style full save).
func (r *EmailInboxRepository) Save(ctx context.Context, inbox *models.EmailInbox) error {
	if err := r.db.WithContext(ctx).Save(inbox).Error; err != nil {
		return fmt.Errorf("saving email inbox %s: %w", inbox.ID, err)
	}
	return nil
}

// SoftDelete marks an EmailInbox as deleted (GORM soft-delete).
func (r *EmailInboxRepository) SoftDelete(ctx context.Context, inbox *models.EmailInbox) error {
	if err := r.db.WithContext(ctx).Delete(inbox).Error; err != nil {
		return fmt.Errorf("deleting email inbox %s: %w", inbox.ID, err)
	}
	return nil
}
