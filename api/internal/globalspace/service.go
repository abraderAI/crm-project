package globalspace

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// VisibilityScope controls which support tickets a caller may see.
type VisibilityScope int

const (
	// ScopeAll allows access to all tickets (DEFT employees / platform admins).
	ScopeAll VisibilityScope = iota
	// ScopeOrg allows access to tickets belonging to the caller's org(s).
	ScopeOrg
	// ScopeOwner allows access only to tickets authored by or assigned to the caller.
	ScopeOwner
)

// CallerVisibility holds the resolved visibility tier and associated data for
// the authenticated caller.
type CallerVisibility struct {
	Scope  VisibilityScope
	UserID string
	OrgIDs []string // populated when Scope == ScopeOrg
}

// Service provides business logic for global space thread operations.
type Service struct {
	repo      *Repository
	bus       *eventbus.Bus
	uploadSvc *upload.Service
}

// NewService creates a new global space Service.
// bus and uploadSvc may be nil; when provided, events are published and file
// attachments are supported respectively.
func NewService(repo *Repository, bus *eventbus.Bus, uploadSvc *upload.Service) *Service {
	return &Service{repo: repo, bus: bus, uploadSvc: uploadSvc}
}

// ResolveVisibility determines the caller's visibility tier for support tickets.
// Must be called with a valid userID.
func (s *Service) ResolveVisibility(ctx context.Context, userID string) (*CallerVisibility, error) {
	isDeft, err := s.repo.IsDeftOrAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isDeft {
		return &CallerVisibility{Scope: ScopeAll, UserID: userID}, nil
	}

	orgIDs, err := s.repo.FindUserOrgIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(orgIDs) > 0 {
		return &CallerVisibility{Scope: ScopeOrg, UserID: userID, OrgIDs: orgIDs}, nil
	}

	return &CallerVisibility{Scope: ScopeOwner, UserID: userID}, nil
}

// canSeeThread checks whether the caller's visibility tier permits access to
// the given thread. Returns true when the caller may view the ticket.
func canSeeThread(cv *CallerVisibility, t *models.Thread) bool {
	switch cv.Scope {
	case ScopeAll:
		return true
	case ScopeOrg:
		if t.OrgID == nil {
			return false
		}
		for _, oid := range cv.OrgIDs {
			if oid == *t.OrgID {
				return true
			}
		}
		return false
	case ScopeOwner:
		return t.AuthorID == cv.UserID || t.AssignedTo == cv.UserID
	default:
		return false
	}
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
	// Visibility is the resolved visibility tier for the caller. When non-nil
	// and the space is global-support, the service enforces scoping.
	Visibility *CallerVisibility
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

	// Enforce visibility scoping for support tickets.
	if input.SpaceSlug == "global-support" && input.Visibility != nil {
		switch input.Visibility.Scope {
		case ScopeOrg:
			params.VisibleOrgIDs = input.Visibility.OrgIDs
		case ScopeOwner:
			params.VisibleUserID = input.Visibility.UserID
		case ScopeAll:
			// No additional filter.
		}
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
// When cv is non-nil and the space is global-support, visibility scoping is enforced.
// Returns nil, nil when the thread does not exist or the caller lacks access.
func (s *Service) GetThread(ctx context.Context, spaceSlug, threadSlug string, cv *CallerVisibility) (*ThreadWithAuthor, error) {
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

	// Enforce visibility for support tickets.
	if spaceSlug == "global-support" && cv != nil && !canSeeThread(cv, t) {
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
// Returns nil, nil when the thread does not exist or the caller lacks access.
func (s *Service) UpdateThread(ctx context.Context, spaceSlug, threadSlug, editorID string, input UpdateInput, cv *CallerVisibility) (*ThreadWithAuthor, error) {
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

	// Enforce visibility for support tickets.
	if spaceSlug == "global-support" && cv != nil && !canSeeThread(cv, t) {
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

	// Publish thread.updated event for notification routing — best-effort.
	if s.bus != nil && spaceSlug == "global-support" {
		s.bus.Publish(eventbus.Event{
			Type:       "thread.updated",
			EntityType: "thread",
			EntityID:   reloaded.ID,
			UserID:     editorID,
			Timestamp:  time.Now(),
			Payload: map[string]any{
				"title":        reloaded.Title,
				"source":       "global-support",
				"participants": []string{reloaded.AuthorID},
			},
		})
	}

	enriched, err := s.enrichThreads(ctx, []models.Thread{*reloaded})
	if err != nil {
		return nil, err
	}
	return &enriched[0], nil
}

// ListAttachments returns all uploads attached to the specified thread.
// Returns nil, nil when the space or thread does not exist or the caller lacks access.
func (s *Service) ListAttachments(ctx context.Context, spaceSlug, threadSlug string, cv *CallerVisibility) ([]models.Upload, error) {
	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("listing attachments: %w", err)
	}
	if board == nil {
		return nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, fmt.Errorf("listing attachments: %w", err)
	}
	if t == nil {
		return nil, nil
	}

	// Enforce visibility for support tickets.
	if spaceSlug == "global-support" && cv != nil && !canSeeThread(cv, t) {
		return nil, nil
	}

	uploads, err := s.repo.ListUploadsByThread(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	return uploads, nil
}

// UploadAttachment stores a file and associates it with the given thread.
// The org_id for the upload is taken from the thread when available, falling
// back to the _system org so that tickets from users without an org are accepted.
// Returns nil, nil when the space or thread does not exist or the caller lacks access.
func (s *Service) UploadAttachment(ctx context.Context, spaceSlug, threadSlug, uploaderID, filename string, size int64, file multipart.File, cv *CallerVisibility) (*models.Upload, error) {
	if s.uploadSvc == nil {
		return nil, fmt.Errorf("upload service not available")
	}

	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("uploading attachment: %w", err)
	}
	if board == nil {
		return nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, fmt.Errorf("uploading attachment: %w", err)
	}
	if t == nil {
		return nil, nil
	}

	// Enforce visibility for support tickets.
	if spaceSlug == "global-support" && cv != nil && !canSeeThread(cv, t) {
		return nil, nil
	}

	// Resolve org: use the ticket's own org when set, otherwise scope to _system.
	orgID := ""
	if t.OrgID != nil && *t.OrgID != "" {
		orgID = *t.OrgID
	} else {
		orgID, err = s.repo.FindSystemOrgID(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving org for upload: %w", err)
		}
	}

	return s.uploadSvc.Create(ctx, orgID, "thread", t.ID, uploaderID, filename, size, file)
}

// TicketNumberer can assign sequential ticket numbers to support threads.
// It is satisfied by the support.Repository type.
type TicketNumberer interface {
	AssignTicketNumber(ctx context.Context, t *models.Thread, orgID string) error
}

// ticketNumberer is an optional injected dependency for assigning ticket numbers.
var ticketNumberer TicketNumberer

// SetTicketNumberer injects the ticket numbering implementation. Called from
// server-services during startup.
func SetTicketNumberer(tn TicketNumberer) {
	ticketNumberer = tn
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
	var initialEntry *models.Message
	if spaceSlug == "global-support" && input.Body != "" {
		now := time.Now()
		initialEntry = &models.Message{
			Body:        input.Body,
			AuthorID:    authorID,
			Metadata:    "{}",
			Type:        models.MessageTypeCustomer,
			IsDeftOnly:  false,
			IsPublished: true,
			IsImmutable: true,
			PublishedAt: &now,
		}
	}
	if err := s.repo.CreateThreadWithInitialEntry(ctx, t, initialEntry); err != nil {
		return nil, err
	}

	// Assign a sequential ticket number for support threads — best effort.
	if spaceSlug == "global-support" && ticketNumberer != nil {
		orgID := "_system"
		if t.OrgID != nil && *t.OrgID != "" {
			orgID = *t.OrgID
		}
		_ = ticketNumberer.AssignTicketNumber(ctx, t, orgID)
	}

	return t, nil
}
