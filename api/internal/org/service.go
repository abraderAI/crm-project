package org

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Common service errors.
var (
	ErrNotFound     = errors.New("org not found")
	ErrSlugConflict = errors.New("slug already exists")
	ErrNameRequired = errors.New("name is required")
	ErrInvalidMeta  = errors.New("invalid metadata JSON")
	ErrForbidden    = errors.New("insufficient permissions")
)

// CreateInput holds parameters for creating an org.
type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

// UpdateInput holds parameters for updating an org.
type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata    *string `json:"metadata,omitempty"`
}

// Service provides business logic for org operations.
type Service struct {
	repo *Repository
}

// NewService creates a new org service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new org with the given input.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Org, error) {
	if input.Name == "" {
		return nil, ErrNameRequired
	}

	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, ErrInvalidMeta
		}
	} else {
		input.Metadata = "{}"
	}

	orgSlug := slug.Generate(input.Name)
	// Deduplicate slug if needed.
	baseSlug := orgSlug
	for i := 1; ; i++ {
		exists, err := s.repo.SlugExists(ctx, orgSlug)
		if err != nil {
			return nil, fmt.Errorf("checking slug uniqueness: %w", err)
		}
		if !exists {
			break
		}
		orgSlug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	org := &models.Org{
		Name:        input.Name,
		Slug:        orgSlug,
		Description: input.Description,
		Metadata:    input.Metadata,
	}
	if err := s.repo.Create(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// GetByRef retrieves an org by ID or slug.
func (s *Service) GetByRef(ctx context.Context, ref string) (*models.Org, error) {
	org, err := s.repo.GetByIDOrSlug(ctx, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return org, nil
}

// List returns paginated orgs, optionally filtered by user membership.
func (s *Service) List(ctx context.Context, cursor string, limit int, userID string) ([]models.Org, bool, error) {
	orgs, err := s.repo.List(ctx, cursor, limit, userID)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(orgs) > limit
	if hasMore {
		orgs = orgs[:limit]
	}
	return orgs, hasMore, nil
}

// Update updates an org with deep-merge for metadata.
func (s *Service) Update(ctx context.Context, ref string, input UpdateInput) (*models.Org, error) {
	org, err := s.repo.GetByIDOrSlug(ctx, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, ErrNameRequired
		}
		org.Name = *input.Name
		// Regenerate slug only if name changed.
		newSlug := slug.Generate(*input.Name)
		if newSlug != org.Slug {
			baseSlug := newSlug
			for i := 1; ; i++ {
				exists, err := s.repo.SlugExists(ctx, newSlug)
				if err != nil {
					return nil, fmt.Errorf("checking slug: %w", err)
				}
				if !exists || newSlug == org.Slug {
					break
				}
				newSlug = fmt.Sprintf("%s-%d", baseSlug, i)
			}
			org.Slug = newSlug
		}
	}

	if input.Description != nil {
		org.Description = *input.Description
	}

	if input.Metadata != nil {
		merged, err := metadata.DeepMerge(org.Metadata, *input.Metadata)
		if err != nil {
			return nil, ErrInvalidMeta
		}
		org.Metadata = merged
	}

	if err := s.repo.Update(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// Delete soft-deletes an org.
func (s *Service) Delete(ctx context.Context, ref string) error {
	org, err := s.repo.GetByIDOrSlug(ctx, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, org.ID)
}
