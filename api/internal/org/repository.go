// Package org provides the Org domain CRUD (handler → service → repository).
package org

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Repository handles database operations for Orgs.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new Org repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new Org.
func (r *Repository) Create(ctx context.Context, org *models.Org) error {
	if err := r.db.WithContext(ctx).Create(org).Error; err != nil {
		return fmt.Errorf("creating org: %w", err)
	}
	return nil
}

// FindByIDOrSlug retrieves an Org by its UUID or slug.
func (r *Repository) FindByIDOrSlug(ctx context.Context, idOrSlug string) (*models.Org, error) {
	var org models.Org
	query := r.db.WithContext(ctx)

	// Try UUID first.
	if _, err := uuid.Parse(idOrSlug); err == nil {
		query = query.Where("id = ?", idOrSlug)
	} else {
		query = query.Where("slug = ?", idOrSlug)
	}

	if err := query.First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("finding org: %w", err)
	}
	return &org, nil
}

// List returns a paginated list of Orgs.
func (r *Repository) List(ctx context.Context, params pagination.Params) ([]models.Org, *pagination.PageInfo, error) {
	var orgs []models.Org
	query := r.db.WithContext(ctx).Order("id ASC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id > ?", cursorID.String())
	}

	// Fetch one extra to determine if there are more results.
	if err := query.Limit(params.Limit + 1).Find(&orgs).Error; err != nil {
		return nil, nil, fmt.Errorf("listing orgs: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(orgs) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(orgs[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		orgs = orgs[:params.Limit]
	}

	return orgs, pageInfo, nil
}

// Update saves changes to an existing Org.
func (r *Repository) Update(ctx context.Context, org *models.Org) error {
	if err := r.db.WithContext(ctx).Save(org).Error; err != nil {
		return fmt.Errorf("updating org: %w", err)
	}
	return nil
}

// Delete soft-deletes an Org.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Org{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("deleting org: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SlugExists checks if a slug is already taken.
func (r *Repository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Org{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		return false, fmt.Errorf("checking slug: %w", err)
	}
	return count > 0, nil
}
