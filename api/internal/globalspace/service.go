package globalspace

import (
	"context"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service provides business logic for global space thread operations.
type Service struct {
	repo *Repository
}

// NewService creates a new global space Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListInput holds parameters for listing threads in a global space.
type ListInput struct {
	SpaceSlug string
	pagination.Params
	// UserID is used when Mine is true to restrict results to the caller's threads.
	UserID string
	// Mine, when true, restricts results to threads authored by UserID.
	Mine bool
	// OrgID, when non-empty, restricts results to threads belonging to this org.
	OrgID string
}

// ListThreads returns paginated threads from the specified global space.
// If the space or its board does not exist, an empty page is returned.
func (s *Service) ListThreads(ctx context.Context, input ListInput) ([]models.Thread, *pagination.PageInfo, error) {
	board, err := s.repo.FindDefaultBoard(ctx, input.SpaceSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("listing threads: %w", err)
	}
	if board == nil {
		return []models.Thread{}, &pagination.PageInfo{}, nil
	}

	params := ListParams{Params: input.Params}
	if input.Mine && input.UserID != "" {
		params.AuthorID = input.UserID
	}
	if input.OrgID != "" {
		params.OrgID = input.OrgID
	}

	return s.repo.ListThreads(ctx, board.ID, params)
}

// CreateInput holds data needed to create a thread in a global space.
type CreateInput struct {
	// Title is the thread title (required).
	Title string
	// Body is the optional thread body.
	Body string
	// OrgID associates the thread with an org for scoping (optional).
	OrgID *string
}

// threadTypeForSpace maps a global space slug to the appropriate ThreadType.
func threadTypeForSpace(spaceSlug string) models.ThreadType {
	switch spaceSlug {
	case "global-support":
		return models.ThreadTypeSupport
	case "global-leads":
		return models.ThreadTypeLead
	default:
		return models.ThreadTypeForum
	}
}

// CreateThread creates a new thread in the specified global space.
// Returns an error if the title is empty, the board does not exist, or the board is locked.
func (s *Service) CreateThread(ctx context.Context, spaceSlug, authorID string, input CreateInput) (*models.Thread, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("creating thread: %w", err)
	}
	if board == nil {
		return nil, fmt.Errorf("global space %q not found", spaceSlug)
	}
	if board.IsLocked {
		return nil, fmt.Errorf("board is locked")
	}

	threadSlug := slug.Generate(input.Title)
	baseSlug := threadSlug
	for i := 2; ; i++ {
		exists, err := s.repo.SlugExistsInBoard(ctx, board.ID, threadSlug)
		if err != nil {
			return nil, fmt.Errorf("checking slug: %w", err)
		}
		if !exists {
			break
		}
		threadSlug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	t := &models.Thread{
		BoardID:    board.ID,
		Title:      input.Title,
		Body:       input.Body,
		Slug:       threadSlug,
		Metadata:   "{}",
		AuthorID:   authorID,
		ThreadType: threadTypeForSpace(spaceSlug),
		Visibility: models.ThreadVisibilityOrgOnly,
		OrgID:      input.OrgID,
	}
	if err := s.repo.CreateThread(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
