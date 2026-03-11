// Package space provides the Space domain CRUD (handler → service → repository).
package space

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for Spaces.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Space repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new Space.
func (r *Repository) Create(ctx context.Context, space *models.Space) error {
	if err := r.db.WithContext(ctx).Create(space).Error; err != nil {
		return fmt.Errorf("creating space: %w", err)
	}
	return nil
}

// FindByIDOrSlug retrieves a Space by UUID or slug within an org.
func (r *Repository) FindByIDOrSlug(ctx context.Context, orgID, idOrSlug string) (*models.Space, error) {
	var space models.Space
	query := r.db.WithContext(ctx).Where("org_id = ?", orgID)

	if _, err := uuid.Parse(idOrSlug); err == nil {
		query = query.Where("id = ?", idOrSlug)
	} else {
		query = query.Where("slug = ?", idOrSlug)
	}

	if err := query.First(&space).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding space: %w", err)
	}
	return &space, nil
}

// List returns a paginated list of Spaces within an org.
func (r *Repository) List(ctx context.Context, orgID string, params pagination.Params) ([]models.Space, *pagination.PageInfo, error) {
	var spaces []models.Space
	query := r.db.WithContext(ctx).Where("org_id = ?", orgID).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&spaces).Error; err != nil {
		return nil, nil, fmt.Errorf("listing spaces: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(spaces) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(spaces[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		spaces = spaces[:params.Limit]
	}

	return spaces, pageInfo, nil
}

// Update saves changes to an existing Space.
func (r *Repository) Update(ctx context.Context, space *models.Space) error {
	if err := r.db.WithContext(ctx).Save(space).Error; err != nil {
		return fmt.Errorf("updating space: %w", err)
	}
	return nil
}

// Delete soft-deletes a Space.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Space{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting space: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SlugExistsInOrg checks if a slug is already taken within the org.
func (r *Repository) SlugExistsInOrg(ctx context.Context, orgID, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Space{}).
		Where("org_id = ? AND slug = ?", orgID, slug).Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug: %w", err)
	}
	return count > 0, nil
}
