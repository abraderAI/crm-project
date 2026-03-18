package globalspace

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
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

// ThreadWithAuthor wraps a Thread with resolved author and org display names.
// These fields are populated by enriching the list/get results from user_shadows
// and orgs tables. They are omitted when the author or org cannot be resolved.
type ThreadWithAuthor struct {
	models.Thread
	AuthorEmail string `json:"author_email,omitempty"`
	AuthorName  string `json:"author_name,omitempty"`
	OrgName     string `json:"org_name,omitempty"`
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

// ListThreads returns paginated, author-enriched threads from the specified global space.
// If the space or its board does not exist, an empty page is returned.
func (s *Service) ListThreads(ctx context.Context, input ListInput) ([]ThreadWithAuthor, *pagination.PageInfo, error) {
	board, err := s.repo.FindDefaultBoard(ctx, input.SpaceSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("listing threads: %w", err)
	}
	if board == nil {
		return []ThreadWithAuthor{}, &pagination.PageInfo{}, nil
	}

	params := ListParams{Params: input.Params}
	if input.Mine && input.UserID != "" {
		params.AuthorID = input.UserID
	}
	if input.OrgID != "" {
		params.OrgID = input.OrgID
	}

	threads, pageInfo, err := s.repo.ListThreads(ctx, board.ID, params)
	if err != nil {
		return nil, nil, err
	}
	enriched, err := s.enrichThreads(ctx, threads)
	if err != nil {
		return nil, nil, err
	}
	return enriched, pageInfo, nil
}

// enrichThreads batch-resolves author and org display info for a slice of threads.
func (s *Service) enrichThreads(ctx context.Context, threads []models.Thread) ([]ThreadWithAuthor, error) {
	// Collect unique author IDs and org IDs.
	authorIDs := make([]string, 0, len(threads))
	orgIDs := make([]string, 0, len(threads))
	seenAuthors := map[string]bool{}
	seenOrgs := map[string]bool{}
	for _, t := range threads {
		if t.AuthorID != "" && !seenAuthors[t.AuthorID] {
			authorIDs = append(authorIDs, t.AuthorID)
			seenAuthors[t.AuthorID] = true
		}
		if t.OrgID != nil && *t.OrgID != "" && !seenOrgs[*t.OrgID] {
			orgIDs = append(orgIDs, *t.OrgID)
			seenOrgs[*t.OrgID] = true
		}
	}

	shadows, err := s.repo.GetUserShadowsByIDs(ctx, authorIDs)
	if err != nil {
		return nil, fmt.Errorf("enriching threads: %w", err)
	}
	orgNames, err := s.repo.GetOrgNamesByIDs(ctx, orgIDs)
	if err != nil {
		return nil, fmt.Errorf("enriching threads: %w", err)
	}

	result := make([]ThreadWithAuthor, len(threads))
	for i, t := range threads {
		rich := ThreadWithAuthor{Thread: t}
		if s, ok := shadows[t.AuthorID]; ok {
			rich.AuthorEmail = s.Email
			rich.AuthorName = s.DisplayName
		}
		if t.OrgID != nil {
			rich.OrgName = orgNames[*t.OrgID]
		}
		result[i] = rich
	}
	return result, nil
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

// GetThread returns a single author-enriched thread by slug from the given global space.
// Returns nil, nil when the thread does not exist.
func (s *Service) GetThread(ctx context.Context, spaceSlug, threadSlug string) (*ThreadWithAuthor, error) {
	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("getting thread: %w", err)
	}
	if board == nil {
		return nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, fmt.Errorf("getting thread: %w", err)
	}
	if t == nil {
		return nil, nil
	}

	enriched, err := s.enrichThreads(ctx, []models.Thread{*t})
	if err != nil {
		return nil, err
	}
	return &enriched[0], nil
}

// UpdateInput holds the fields that may be updated on a global space thread.
type UpdateInput struct {
	// Body replaces the thread body when non-nil.
	Body *string `json:"body"`
	// Status sets the status field in metadata when non-empty.
	Status *string `json:"status"`
}

// UpdateThread applies a partial update to a thread in the given global space.
// Status is stored via metadata deep-merge. A revision record is saved.
// Returns nil, nil when the thread does not exist.
func (s *Service) UpdateThread(ctx context.Context, spaceSlug, threadSlug, editorID string, input UpdateInput) (*ThreadWithAuthor, error) {
	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("updating thread: %w", err)
	}
	if board == nil {
		return nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, fmt.Errorf("updating thread: %w", err)
	}
	if t == nil {
		return nil, nil
	}

	// Snapshot previous state for revision.
	prevContent := map[string]string{"body": t.Body, "metadata": t.Metadata}
	prevJSON, _ := json.Marshal(prevContent)

	if input.Body != nil {
		t.Body = *input.Body
	}
	if input.Status != nil {
		patch := fmt.Sprintf(`{"status":%q}`, *input.Status)
		merged, mergeErr := metadata.DeepMerge(t.Metadata, patch)
		if mergeErr != nil {
			return nil, fmt.Errorf("updating status: %w", mergeErr)
		}
		t.Metadata = merged
	}

	if err := s.repo.UpdateThread(ctx, t); err != nil {
		return nil, err
	}

	// Save revision — best-effort (errors are intentionally ignored).
	rev := &models.Revision{
		EntityType:      "thread",
		EntityID:        t.ID,
		PreviousContent: string(prevJSON),
		EditorID:        editorID,
	}
	_ = s.repo.CreateRevision(ctx, rev)

	// Reload from DB so generated columns (status, priority, etc.) are fresh.
	reloaded, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil || reloaded == nil {
		// Fallback: return the in-memory version if reload fails.
		reloaded = t
	}

	enriched, err := s.enrichThreads(ctx, []models.Thread{*reloaded})
	if err != nil {
		return nil, err
	}
	return &enriched[0], nil
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
