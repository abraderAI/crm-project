// Package moderation provides community moderation features (flags, move, merge, hide).
package moderation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for moderation.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Moderation repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateFlag inserts a new moderation flag.
func (r *Repository) CreateFlag(ctx context.Context, flag *models.Flag) error {
	if err := r.db.WithContext(ctx).Create(flag).Error; err != nil {
		return fmt.Errorf("creating flag: %w", err)
	}
	return nil
}

// FindFlagByID retrieves a flag by ID.
func (r *Repository) FindFlagByID(ctx context.Context, id string) (*models.Flag, error) {
	var flag models.Flag
	if err := r.db.WithContext(ctx).First(&flag, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding flag: %w", err)
	}
	return &flag, nil
}

// UpdateFlag saves changes to a flag.
func (r *Repository) UpdateFlag(ctx context.Context, flag *models.Flag) error {
	if err := r.db.WithContext(ctx).Save(flag).Error; err != nil {
		return fmt.Errorf("updating flag: %w", err)
	}
	return nil
}

// ListOrgFlags returns paginated open flags for threads within an org.
// It joins through the hierarchy: flag → thread → board → space → org.
func (r *Repository) ListOrgFlags(ctx context.Context, orgID string, status models.FlagStatus, params pagination.Params) ([]models.Flag, *pagination.PageInfo, error) {
	var flags []models.Flag

	query := r.db.WithContext(ctx).
		Joins("JOIN threads ON threads.id = flags.thread_id AND threads.deleted_at IS NULL").
		Joins("JOIN boards ON boards.id = threads.board_id AND boards.deleted_at IS NULL").
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.org_id = ?", orgID).
		Where("flags.status = ?", status).
		Order("flags.id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("flags.id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&flags).Error; err != nil {
		return nil, nil, fmt.Errorf("listing flags: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(flags) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(flags[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		flags = flags[:params.Limit]
	}

	return flags, pageInfo, nil
}

// FindThreadByID retrieves a thread by ID.
func (r *Repository) FindThreadByID(ctx context.Context, id string) (*models.Thread, error) {
	var thread models.Thread
	if err := r.db.WithContext(ctx).First(&thread, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}
	return &thread, nil
}

// UpdateThread saves changes to a thread.
func (r *Repository) UpdateThread(ctx context.Context, thread *models.Thread) error {
	if err := r.db.WithContext(ctx).Save(thread).Error; err != nil {
		return fmt.Errorf("updating thread: %w", err)
	}
	return nil
}

// FindBoardByID retrieves a board by ID.
func (r *Repository) FindBoardByID(ctx context.Context, id string) (*models.Board, error) {
	var b models.Board
	if err := r.db.WithContext(ctx).First(&b, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding board: %w", err)
	}
	return &b, nil
}

// MoveMessages updates all messages from one thread to another.
func (r *Repository) MoveMessages(ctx context.Context, fromThreadID, toThreadID string) error {
	err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("thread_id = ?", fromThreadID).
		Update("thread_id", toThreadID).Error
	if err != nil {
		return fmt.Errorf("moving messages: %w", err)
	}
	return nil
}

// SoftDeleteThread marks a thread as deleted.
func (r *Repository) SoftDeleteThread(ctx context.Context, threadID string) error {
	if err := r.db.WithContext(ctx).Delete(&models.Thread{}, "id = ?", threadID).Error; err != nil {
		return fmt.Errorf("soft-deleting thread: %w", err)
	}
	return nil
}

// CreateAuditLog inserts an audit log entry.
func (r *Repository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("creating audit log: %w", err)
	}
	return nil
}
