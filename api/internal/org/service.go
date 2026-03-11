package org

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service provides business logic for Org operations.
type Service struct {
	repo *Repository
}

// NewService creates a new Org service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the data needed to create an Org.
type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

// Create validates input and creates a new Org.
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.Org, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, err
		}
	} else {
		input.Metadata = "{}"
	}

	orgSlug := slug.Generate(input.Name)
	// Ensure uniqueness with suffix.
	baseSlug := orgSlug
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExists(ctx, orgSlug)
		if err != nil {
			return nil, fmt.Errorf("checking slug: %w", err)
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

// Get retrieves an Org by ID or slug.
func (s *Service) Get(ctx context.Context, idOrSlug string) (*models.Org, error) {
	return s.repo.FindByIDOrSlug(ctx, idOrSlug)
}

// List returns a paginated list of Orgs.
func (s *Service) List(ctx context.Context, params pagination.Params) ([]models.Org, *pagination.PageInfo, error) {
	return s.repo.List(ctx, params)
}

// UpdateInput holds partial update data for an Org.
type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Metadata    *string `json:"metadata"`
}

// Update applies partial updates to an Org using deep-merge for metadata.
func (s *Service) Update(ctx context.Context, idOrSlug string, input UpdateInput) (*models.Org, error) {
	org, err := s.repo.FindByIDOrSlug(ctx, idOrSlug)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, nil
	}

	if input.Name != nil && *input.Name != "" {
		org.Name = *input.Name
		// Regenerate slug on rename.
		newSlug := slug.Generate(*input.Name)
		if newSlug != org.Slug {
			baseSlug := newSlug
			for i := 2; ; i++ {
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
		if err := metadata.Validate(*input.Metadata); err != nil {
			return nil, err
		}
		merged, err := metadata.DeepMerge(org.Metadata, *input.Metadata)
		if err != nil {
			return nil, err
		}
		org.Metadata = merged
	}

	if err := s.repo.Update(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// Delete soft-deletes an Org.
func (s *Service) Delete(ctx context.Context, idOrSlug string) error {
	org, err := s.repo.FindByIDOrSlug(ctx, idOrSlug)
	if err != nil {
		return err
	}
	if org == nil {
		return fmt.Errorf("not found")
	}
	return s.repo.Delete(ctx, org.ID)
}
