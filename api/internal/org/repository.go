// Package org provides the handler, service, and repository for Org CRUD operations.
package org

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// Repository handles database operations for orgs.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new org repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new org into the database.
func (r *Repository) Create(ctx context.Context, org *models.Org) error {
	if err := r.db.WithContext(ctx).Create(org).Error; err != nil {
		return fmt.Errorf("creating org: %w", err)
	}
	return nil
}

// GetByIDOrSlug retrieves an org by its ID (UUID) or slug.
func (r *Repository) GetByIDOrSlug(ctx context.Context, ref string) (*models.Org, error) {
	var org models.Org
	query := r.db.WithContext(ctx)
	if isUUID(ref) {
		query = query.Where("id = ?", ref)
	} else {
		query = query.Where("slug = ?", ref)
	}
	if err := query.First(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

// List retrieves orgs with cursor pagination, optionally filtered by user membership.
func (r *Repository) List(ctx context.Context, cursor string, limit int, userID string) ([]models.Org, error) {
	query := r.db.WithContext(ctx).Order("id ASC")

	// Filter by user membership if userID provided.
	if userID != "" {
		query = query.Where("id IN (SELECT org_id FROM org_memberships WHERE user_id = ? AND deleted_at IS NULL)", userID)
	}

	if cursor != "" {
		query = query.Where("id > ?", cursor)
	}
	query = query.Limit(limit + 1)

	var orgs []models.Org
	if err := query.Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("listing orgs: %w", err)
	}
	return orgs, nil
}

// Update saves changes to an existing org.
func (r *Repository) Update(ctx context.Context, org *models.Org) error {
	if err := r.db.WithContext(ctx).Save(org).Error; err != nil {
		return fmt.Errorf("updating org: %w", err)
	}
	return nil
}

// Delete soft-deletes an org.
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

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
