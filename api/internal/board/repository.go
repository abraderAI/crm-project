// Package board provides the Board domain CRUD (handler → service → repository).
package board

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for Boards.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Board repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new Board.
func (r *Repository) Create(ctx context.Context, board *models.Board) error {
	if err := r.db.WithContext(ctx).Create(board).Error; err != nil {
		return fmt.Errorf("creating board: %w", err)
	}
	return nil
}

// FindByIDOrSlug retrieves a Board by UUID or slug within a space.
func (r *Repository) FindByIDOrSlug(ctx context.Context, spaceID, idOrSlug string) (*models.Board, error) {
	var board models.Board
	query := r.db.WithContext(ctx).Where("space_id = ?", spaceID)

	if _, err := uuid.Parse(idOrSlug); err == nil {
		query = query.Where("id = ?", idOrSlug)
	} else {
		query = query.Where("slug = ?", idOrSlug)
	}

	if err := query.First(&board).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding board: %w", err)
	}
	return &board, nil
}

// List returns a paginated list of Boards within a space.
func (r *Repository) List(ctx context.Context, spaceID string, params pagination.Params) ([]models.Board, *pagination.PageInfo, error) {
	var boards []models.Board
	query := r.db.WithContext(ctx).Where("space_id = ?", spaceID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&boards).Error; err != nil {
		return nil, nil, fmt.Errorf("listing boards: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(boards) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(boards[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		boards = boards[:params.Limit]
	}

	return boards, pageInfo, nil
}

// Update saves changes to an existing Board.
func (r *Repository) Update(ctx context.Context, board *models.Board) error {
	if err := r.db.WithContext(ctx).Save(board).Error; err != nil {
		return fmt.Errorf("updating board: %w", err)
	}
	return nil
}

// Delete soft-deletes a Board.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Board{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting board: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SlugExistsInSpace checks if a slug is already taken within the space.
func (r *Repository) SlugExistsInSpace(ctx context.Context, spaceID, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Board{}).
		Where("space_id = ? AND slug = ?", spaceID, slug).Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug: %w", err)
	}
	return count > 0, nil
}
