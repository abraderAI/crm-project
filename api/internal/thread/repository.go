// Package thread provides the Thread domain CRUD (handler → service → repository).
package thread

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for Threads.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Thread repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new Thread.
func (r *Repository) Create(ctx context.Context, thread *models.Thread) error {
	if err := r.db.WithContext(ctx).Create(thread).Error; err != nil {
		return fmt.Errorf("creating thread: %w", err)
	}
	return nil
}

// FindByIDOrSlug retrieves a Thread by UUID or slug within a board.
func (r *Repository) FindByIDOrSlug(ctx context.Context, boardID, idOrSlug string) (*models.Thread, error) {
	var thread models.Thread
	query := r.db.WithContext(ctx).Where("board_id = ?", boardID)

	if _, err := uuid.Parse(idOrSlug); err == nil {
		query = query.Where("id = ?", idOrSlug)
	} else {
		query = query.Where("slug = ?", idOrSlug)
	}

	if err := query.First(&thread).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding thread: %w", err)
	}
	return &thread, nil
}

// FindByID retrieves a Thread by its ID (regardless of board).
func (r *Repository) FindByID(ctx context.Context, id string) (*models.Thread, error) {
	var thread models.Thread
	if err := r.db.WithContext(ctx).First(&thread, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding thread by id: %w", err)
	}
	return &thread, nil
}

// MetadataFilter defines a single metadata filter condition.
type MetadataFilter struct {
	Path     string // e.g., "$.status"
	Operator string // "eq", "gt", "gte", "lt", "lte"
	Value    string
}

// ListParams extends pagination with metadata filtering.
type ListParams struct {
	pagination.Params
	Filters []MetadataFilter
}

// List returns a paginated, filtered list of Threads within a board.
func (r *Repository) List(ctx context.Context, boardID string, params ListParams) ([]models.Thread, *pagination.PageInfo, error) {
	var threads []models.Thread
	query := r.db.WithContext(ctx).Where("board_id = ?", boardID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	// Apply metadata filters.
	for _, f := range params.Filters {
		jsonPath := fmt.Sprintf("json_extract(metadata, '%s')", f.Path)
		switch strings.ToLower(f.Operator) {
		case "eq", "":
			query = query.Where(fmt.Sprintf("%s = ?", jsonPath), f.Value)
		case "gt":
			query = query.Where(fmt.Sprintf("CAST(%s AS REAL) > ?", jsonPath), f.Value)
		case "gte":
			query = query.Where(fmt.Sprintf("CAST(%s AS REAL) >= ?", jsonPath), f.Value)
		case "lt":
			query = query.Where(fmt.Sprintf("CAST(%s AS REAL) < ?", jsonPath), f.Value)
		case "lte":
			query = query.Where(fmt.Sprintf("CAST(%s AS REAL) <= ?", jsonPath), f.Value)
		}
	}

	if err := query.Limit(params.Limit + 1).Find(&threads).Error; err != nil {
		return nil, nil, fmt.Errorf("listing threads: %w", err)
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

// Update saves changes to an existing Thread.
func (r *Repository) Update(ctx context.Context, thread *models.Thread) error {
	if err := r.db.WithContext(ctx).Save(thread).Error; err != nil {
		return fmt.Errorf("updating thread: %w", err)
	}
	return nil
}

// Delete soft-deletes a Thread.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Thread{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting thread: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SlugExistsInBoard checks if a slug exists within the board.
func (r *Repository) SlugExistsInBoard(ctx context.Context, boardID, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Thread{}).
		Where("board_id = ? AND slug = ?", boardID, slug).Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug: %w", err)
	}
	return count > 0, nil
}

// CreateRevision stores a revision record.
func (r *Repository) CreateRevision(ctx context.Context, rev *models.Revision) error {
	if err := r.db.WithContext(ctx).Create(rev).Error; err != nil {
		return fmt.Errorf("creating revision: %w", err)
	}
	return nil
}
