package thread

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service provides business logic for Thread operations.
type Service struct {
	repo *Repository
}

// NewService creates a new Thread service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the data needed to create a Thread.
type CreateInput struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Metadata string `json:"metadata"`
}

// Create validates input and creates a new Thread.
// Returns an error if the parent board is locked.
func (s *Service) Create(ctx context.Context, boardID, authorID string, boardLocked bool, input CreateInput) (*models.Thread, error) {
	if boardLocked {
		return nil, fmt.Errorf("board is locked")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, err
		}
	} else {
		input.Metadata = "{}"
	}

	threadSlug := slug.Generate(input.Title)
	baseSlug := threadSlug
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExistsInBoard(ctx, boardID, threadSlug)
		if err != nil {
			return nil, fmt.Errorf("checking slug: %w", err)
		}
		if !exists {
			break
		}
		threadSlug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	t := &models.Thread{
		BoardID:  boardID,
		Title:    input.Title,
		Body:     input.Body,
		Slug:     threadSlug,
		Metadata: input.Metadata,
		AuthorID: authorID,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Get retrieves a Thread by ID or slug within a board.
func (s *Service) Get(ctx context.Context, boardID, idOrSlug string) (*models.Thread, error) {
	return s.repo.FindByIDOrSlug(ctx, boardID, idOrSlug)
}

// List returns a paginated, filtered list of Threads within a board.
func (s *Service) List(ctx context.Context, boardID string, params ListParams) ([]models.Thread, *pagination.PageInfo, error) {
	return s.repo.List(ctx, boardID, params)
}

// UpdateInput holds partial update data for a Thread.
type UpdateInput struct {
	Title    *string `json:"title"`
	Body     *string `json:"body"`
	Metadata *string `json:"metadata"`
}

// Update applies partial updates to a Thread with revision tracking.
func (s *Service) Update(ctx context.Context, boardID, idOrSlug, editorID string, input UpdateInput) (*models.Thread, error) {
	t, err := s.repo.FindByIDOrSlug(ctx, boardID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}

	// Create revision before update.
	prevContent := map[string]string{
		"title":    t.Title,
		"body":     t.Body,
		"metadata": t.Metadata,
	}
	prevJSON, _ := json.Marshal(prevContent)

	if input.Title != nil && *input.Title != "" {
		t.Title = *input.Title
		newSlug := slug.Generate(*input.Title)
		if newSlug != t.Slug {
			baseSlug := newSlug
			for i := 2; ; i++ {
				exists, err := s.repo.SlugExistsInBoard(ctx, boardID, newSlug)
				if err != nil {
					return nil, fmt.Errorf("checking slug: %w", err)
				}
				if !exists || newSlug == t.Slug {
					break
				}
				newSlug = fmt.Sprintf("%s-%d", baseSlug, i)
			}
			t.Slug = newSlug
		}
	}
	if input.Body != nil {
		t.Body = *input.Body
	}
	if input.Metadata != nil {
		if err := metadata.Validate(*input.Metadata); err != nil {
			return nil, err
		}
		merged, err := metadata.DeepMerge(t.Metadata, *input.Metadata)
		if err != nil {
			return nil, err
		}
		t.Metadata = merged
	}

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}

	// Save revision.
	rev := &models.Revision{
		EntityType:      "thread",
		EntityID:        t.ID,
		PreviousContent: string(prevJSON),
		EditorID:        editorID,
	}
	_ = s.repo.CreateRevision(ctx, rev)

	return t, nil
}

// Delete soft-deletes a Thread.
func (s *Service) Delete(ctx context.Context, boardID, idOrSlug string) error {
	t, err := s.repo.FindByIDOrSlug(ctx, boardID, idOrSlug)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("not found")
	}
	return s.repo.Delete(ctx, t.ID)
}

// SetPin sets the pin state of a Thread.
func (s *Service) SetPin(ctx context.Context, boardID, idOrSlug string, pinned bool) (*models.Thread, error) {
	t, err := s.repo.FindByIDOrSlug(ctx, boardID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	t.IsPinned = pinned
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// SetLock sets the lock state of a Thread.
func (s *Service) SetLock(ctx context.Context, boardID, idOrSlug string, locked bool) (*models.Thread, error) {
	t, err := s.repo.FindByIDOrSlug(ctx, boardID, idOrSlug)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}
	t.IsLocked = locked
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
