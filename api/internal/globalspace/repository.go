// Package globalspace provides HTTP handlers for the /global-spaces API endpoints.
// These endpoints give authenticated users access to platform-wide spaces such as
// global-support and global-forum without needing to know the underlying org/board IDs.
package globalspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for global space threads.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new global space Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// FindDefaultBoard returns the first non-deleted board in the global space with the
// given slug. Returns nil, nil when no board exists.
func (r *Repository) FindDefaultBoard(ctx context.Context, spaceSlug string) (*models.Board, error) {
	var board models.Board
	err := r.db.WithContext(ctx).
		Joins("JOIN spaces ON spaces.id = boards.space_id AND spaces.deleted_at IS NULL").
		Where("spaces.slug = ? AND boards.deleted_at IS NULL", spaceSlug).
		First(&board).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding default board for %s: %w", spaceSlug, err)
	}
	return &board, nil
}

// ListParams holds pagination and scoping options for listing global space threads.
type ListParams struct {
	pagination.Params
	// AuthorID, when non-empty, restricts results to threads authored by this user.
	AuthorID string
	// OrgID, when non-empty, restricts results to threads belonging to this org.
	OrgID string
}

// ListThreads returns a paginated list of threads in boardID, filtered by ListParams.
func (r *Repository) ListThreads(ctx context.Context, boardID string, params ListParams) ([]models.Thread, *pagination.PageInfo, error) {
	var threads []models.Thread
	query := r.db.WithContext(ctx).Where("board_id = ?", boardID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}
	if params.AuthorID != "" {
		query = query.Where("author_id = ?", params.AuthorID)
	}
	if params.OrgID != "" {
		query = query.Where("org_id = ?", params.OrgID)
	}

	if err := query.Limit(params.Limit + 1).Find(&threads).Error; err != nil {
		return nil, nil, fmt.Errorf("listing global space threads: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(threads) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(threads[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		threads = threads[:params.Limit]
	}
	return threads, pageInfo, nil
}

// CreateThread inserts a new thread record.
func (r *Repository) CreateThread(ctx context.Context, thread *models.Thread) error {
	if err := r.db.WithContext(ctx).Create(thread).Error; err != nil {
		return fmt.Errorf("creating global space thread: %w", err)
	}
	return nil
}

// SlugExistsInBoard reports whether a thread slug already exists in boardID.
func (r *Repository) SlugExistsInBoard(ctx context.Context, boardID, threadSlug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Thread{}).
		Where("board_id = ? AND slug = ?", boardID, threadSlug).Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug: %w", err)
	}
	return count > 0, nil
}
