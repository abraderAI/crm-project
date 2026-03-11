// Package message provides the Message domain CRUD (handler → service → repository).
package message

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for Messages.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Message repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new Message.
func (r *Repository) Create(ctx context.Context, msg *models.Message) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("creating message: %w", err)
	}
	return nil
}

// FindByID retrieves a Message by its ID within a thread.
func (r *Repository) FindByID(ctx context.Context, threadID, id string) (*models.Message, error) {
	var msg models.Message
	if err := r.db.WithContext(ctx).
		Where("thread_id = ? AND id = ?", threadID, id).
		First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding message: %w", err)
	}
	return &msg, nil
}

// List returns a paginated list of Messages within a thread.
func (r *Repository) List(ctx context.Context, threadID string, params pagination.Params) ([]models.Message, *pagination.PageInfo, error) {
	var messages []models.Message
	query := r.db.WithContext(ctx).Where("thread_id = ?", threadID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&messages).Error; err != nil {
		return nil, nil, fmt.Errorf("listing messages: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(messages) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(messages[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		messages = messages[:params.Limit]
	}

	return messages, pageInfo, nil
}

// Update saves changes to an existing Message.
func (r *Repository) Update(ctx context.Context, msg *models.Message) error {
	if err := r.db.WithContext(ctx).Save(msg).Error; err != nil {
		return fmt.Errorf("updating message: %w", err)
	}
	return nil
}

// Delete soft-deletes a Message.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Message{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CreateRevision stores a revision record.
func (r *Repository) CreateRevision(ctx context.Context, rev *models.Revision) error {
	if err := r.db.WithContext(ctx).Create(rev).Error; err != nil {
		return fmt.Errorf("creating revision: %w", err)
	}
	return nil
}
