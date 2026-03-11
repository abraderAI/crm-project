package board

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service provides business logic for Board operations.
type Service struct {
	repo *Repository
}

// NewService creates a new Board service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the data needed to create a Board.
type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

// Create validates input and creates a new Board.
func (s *Service) Create(ctx context.Context, spaceID string, input CreateInput) (*models.Board, error) {
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

	boardSlug := slug.Generate(input.Name)
	baseSlug := boardSlug
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExistsInSpace(ctx, spaceID, boardSlug)
		if err != nil {
			return nil, fmt.Errorf("checking slug: %w", err)
		}
		if !exists {
			break
		}
		boardSlug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	b := &models.Board{
		SpaceID:     spaceID,
		Name:        input.Name,
		Slug:        boardSlug,
		Description: input.Description,
		Metadata:    input.Metadata,
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// Get retrieves a Board by ID or slug within a space.
func (s *Service) Get(ctx context.Context, spaceID, idOrSlug string) (*models.Board, error) {
	return s.repo.FindByIDOrSlug(ctx, spaceID, idOrSlug)
}

// List returns a paginated list of Boards within a space.
func (s *Service) List(ctx context.Context, spaceID string, params pagination.Params) ([]models.Board, *pagination.PageInfo, error) {
	return s.repo.List(ctx, spaceID, params)
}

// UpdateInput holds partial update data for a Board.
type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Metadata    *string `json:"metadata"`
}

// Update applies partial updates to a Board.
func (s *Service) Update(ctx context.Context, spaceID, idOrSlug string, input UpdateInput) (*models.Board, error) {
	b, err := s.repo.FindByIDOrSlug(ctx, spaceID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}

	if input.Name != nil && *input.Name != "" {
		b.Name = *input.Name
		newSlug := slug.Generate(*input.Name)
		if newSlug != b.Slug {
			baseSlug := newSlug
			for i := 2; ; i++ {
				exists, err := s.repo.SlugExistsInSpace(ctx, spaceID, newSlug)
				if err != nil {
					return nil, fmt.Errorf("checking slug: %w", err)
				}
				if !exists || newSlug == b.Slug {
					break
				}
				newSlug = fmt.Sprintf("%s-%d", baseSlug, i)
			}
			b.Slug = newSlug
		}
	}
	if input.Description != nil {
		b.Description = *input.Description
	}
	if input.Metadata != nil {
		if err := metadata.Validate(*input.Metadata); err != nil {
			return nil, err
		}
		merged, err := metadata.DeepMerge(b.Metadata, *input.Metadata)
		if err != nil {
			return nil, err
		}
		b.Metadata = merged
	}

	if err := s.repo.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// Delete soft-deletes a Board.
func (s *Service) Delete(ctx context.Context, spaceID, idOrSlug string) error {
	b, err := s.repo.FindByIDOrSlug(ctx, spaceID, idOrSlug)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("not found")
	}
	return s.repo.Delete(ctx, b.ID)
}

// SetLock sets the lock state of a Board.
func (s *Service) SetLock(ctx context.Context, spaceID, idOrSlug string, locked bool) (*models.Board, error) {
	b, err := s.repo.FindByIDOrSlug(ctx, spaceID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, nil
	}
	b.IsLocked = locked
	if err := s.repo.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}
