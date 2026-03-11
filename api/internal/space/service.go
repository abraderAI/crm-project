package space

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service provides business logic for Space operations.
type Service struct {
	repo *Repository
}

// NewService creates a new Space service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the data needed to create a Space.
type CreateInput struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Metadata    string           `json:"metadata"`
	Type        models.SpaceType `json:"type"`
}

// Create validates input and creates a new Space.
func (s *Service) Create(ctx context.Context, orgID string, input CreateInput) (*models.Space, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if input.Type == "" {
		input.Type = models.SpaceTypeGeneral
	}
	if !input.Type.IsValid() {
		return nil, fmt.Errorf("invalid space type: %s", input.Type)
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, err
		}
	} else {
		input.Metadata = "{}"
	}

	spaceSlug := slug.Generate(input.Name)
	baseSlug := spaceSlug
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExistsInOrg(ctx, orgID, spaceSlug)
		if err != nil {
			return nil, fmt.Errorf("checking slug: %w", err)
		}
		if !exists {
			break
		}
		spaceSlug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	sp := &models.Space{
		OrgID:       orgID,
		Name:        input.Name,
		Slug:        spaceSlug,
		Description: input.Description,
		Metadata:    input.Metadata,
		Type:        input.Type,
	}
	if err := s.repo.Create(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// Get retrieves a Space by ID or slug within an org.
func (s *Service) Get(ctx context.Context, orgID, idOrSlug string) (*models.Space, error) {
	return s.repo.FindByIDOrSlug(ctx, orgID, idOrSlug)
}

// List returns a paginated list of Spaces within an org.
func (s *Service) List(ctx context.Context, orgID string, params pagination.Params) ([]models.Space, *pagination.PageInfo, error) {
	return s.repo.List(ctx, orgID, params)
}

// UpdateInput holds partial update data for a Space.
type UpdateInput struct {
	Name        *string           `json:"name"`
	Description *string           `json:"description"`
	Metadata    *string           `json:"metadata"`
	Type        *models.SpaceType `json:"type"`
}

// Update applies partial updates to a Space using deep-merge for metadata.
func (s *Service) Update(ctx context.Context, orgID, idOrSlug string, input UpdateInput) (*models.Space, error) {
	sp, err := s.repo.FindByIDOrSlug(ctx, orgID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, nil
	}

	if input.Name != nil && *input.Name != "" {
		sp.Name = *input.Name
		newSlug := slug.Generate(*input.Name)
		if newSlug != sp.Slug {
			baseSlug := newSlug
			for i := 2; ; i++ {
				exists, err := s.repo.SlugExistsInOrg(ctx, orgID, newSlug)
				if err != nil {
					return nil, fmt.Errorf("checking slug: %w", err)
				}
				if !exists || newSlug == sp.Slug {
					break
				}
				newSlug = fmt.Sprintf("%s-%d", baseSlug, i)
			}
			sp.Slug = newSlug
		}
	}
	if input.Description != nil {
		sp.Description = *input.Description
	}
	if input.Type != nil {
		if !input.Type.IsValid() {
			return nil, fmt.Errorf("invalid space type: %s", *input.Type)
		}
		sp.Type = *input.Type
	}
	if input.Metadata != nil {
		if err := metadata.Validate(*input.Metadata); err != nil {
			return nil, err
		}
		merged, err := metadata.DeepMerge(sp.Metadata, *input.Metadata)
		if err != nil {
			return nil, err
		}
		sp.Metadata = merged
	}

	if err := s.repo.Update(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// Delete soft-deletes a Space.
func (s *Service) Delete(ctx context.Context, orgID, idOrSlug string) error {
	sp, err := s.repo.FindByIDOrSlug(ctx, orgID, idOrSlug)
	if err != nil {
		return err
	}
	if sp == nil {
		return fmt.Errorf("not found")
	}
	return s.repo.Delete(ctx, sp.ID)
}
