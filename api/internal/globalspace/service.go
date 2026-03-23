package globalspace

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
	"github.com/abraderAI/crm-project/api/internal/vote"
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
	Email  string   // populated from user_shadows for ContactEmail matching
}

// Service provides business logic for global space thread operations.
type Service struct {
	repo      *Repository
	bus       *eventbus.Bus
	uploadSvc *upload.Service
	voteSvc   VoteToggler
}

// VoteToggler abstracts the vote toggle operation. Satisfied by *vote.Service.
type VoteToggler interface {
	Toggle(ctx context.Context, threadID, userID string, role models.Role, billingTier string) (*vote.VoteResult, error)
}

// NewService creates a new global space Service.
// bus, uploadSvc, and voteSvc may be nil.
func NewService(repo *Repository, bus *eventbus.Bus, uploadSvc *upload.Service) *Service {
	return &Service{repo: repo, bus: bus, uploadSvc: uploadSvc}
}

// SetVoteService injects the vote service dependency. Called during server startup.
func (s *Service) SetVoteService(v VoteToggler) {
	s.voteSvc = v
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

	// Resolve email for ContactEmail matching on owner-scoped tickets.
	var email string
	shadows, shadowErr := s.repo.GetUserShadowsByIDs(ctx, []string{userID})
	if shadowErr == nil {
		if s, ok := shadows[userID]; ok {
			email = s.Email
		}
	}

	orgIDs, err := s.repo.FindUserOrgIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(orgIDs) > 0 {
		return &CallerVisibility{Scope: ScopeOrg, UserID: userID, OrgIDs: orgIDs, Email: email}, nil
	}

	return &CallerVisibility{Scope: ScopeOwner, UserID: userID, Email: email}, nil
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
		if t.AuthorID == cv.UserID || t.AssignedTo == cv.UserID {
			return true
		}
		// Allow access when the ticket's ContactEmail matches the caller's email.
		if t.ContactEmail != "" && cv.Email != "" && t.ContactEmail == cv.Email {
			return true
		}
		return false
	default:
		return false
	}
}

// ThreadWithAuthor wraps a Thread with resolved author and org display names.
// These fields are populated by enriching the list/get results from user_shadows
// and orgs tables. They are omitted when the author or org cannot be resolved.
type ThreadWithAuthor struct {
	models.Thread
	AuthorEmail        string `json:"author_email,omitempty"`
	AuthorName         string `json:"author_name,omitempty"`
	OrgName            string `json:"org_name,omitempty"`
	RegistrationStatus string `json:"registration_status,omitempty"`
}

// MessageWithAuthor wraps a Message with resolved author display info.
type MessageWithAuthor struct {
	models.Message
	AuthorName string `json:"author_name,omitempty"`
	AuthorOrg  string `json:"author_org,omitempty"`
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
	// IncludeHidden, when true, returns hidden threads (admin view).
	IncludeHidden bool
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

	// Support and leads sort newest-first; forum sorts oldest-first.
	sortDesc := input.SpaceSlug == "global-support" || input.SpaceSlug == "global-leads"
	params := ListParams{Params: input.Params, IncludeHidden: input.IncludeHidden, SortDesc: sortDesc}
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
			params.VisibleUserEmail = input.Visibility.Email
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

	// Privacy: strip ContactEmail for non-DEFT callers.
	if input.Visibility != nil && input.Visibility.Scope != ScopeAll {
		stripContactEmail(enriched)
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

	// Resolve author org memberships.
	authorOrgNames, err := s.repo.GetUserPrimaryOrgNames(ctx, authorIDs)
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
		// If thread has no explicit org_name, fall back to the author's primary org.
		// For email-assigned tickets only do this when the contact has registered
		// (author_id == the contact's clerk ID). When the contact is unregistered
		// author_id is still the DEFT agent, so we must not use their org.
		if rich.OrgName == "" {
			authorIsContact := t.ContactEmail == ""
			if !authorIsContact {
				if s, found := shadows[t.AuthorID]; found && strings.EqualFold(s.Email, t.ContactEmail) {
					authorIsContact = true
				}
			}
			if authorIsContact {
				rich.OrgName = authorOrgNames[t.AuthorID]
			}
		}

		// Derive registration status for the ticket creator.
		if t.ContactEmail != "" {
			// Email-assigned ticket: determine whether the contact has registered.
			// When the contact was found at creation time, author_id was set to
			// their clerk ID and their shadow email matches contact_email.
			// Otherwise author_id is the DEFT agent — the contact is unregistered.
			if s, found := shadows[t.AuthorID]; found && strings.EqualFold(s.Email, t.ContactEmail) {
				// Contact is a registered user who owns this ticket.
				if rich.OrgName == "" {
					rich.RegistrationStatus = "registered"
				}
			} else {
				// Author is a DEFT agent — contact has not yet registered.
				rich.RegistrationStatus = "unregistered"
			}
		} else if t.OrgID == nil || (t.OrgID != nil && *t.OrgID == "") {
			if _, found := shadows[t.AuthorID]; found {
				// Registered user with no org.
				if rich.OrgName == "" {
					rich.RegistrationStatus = "registered"
				}
			}
		}
		// When OrgName is set, registration_status stays empty (org badge shown instead).
		result[i] = rich
	}
	return result, nil
}

// stripContactEmail removes ContactEmail from enriched results.
// Used to prevent non-DEFT callers from seeing PII.
func stripContactEmail(threads []ThreadWithAuthor) {
	for i := range threads {
		threads[i].ContactEmail = ""
	}
}

// CreateInput holds data needed to create a thread in a global space.
type CreateInput struct {
	// Title is the thread title (required).
	Title string
	// Body is the optional thread body.
	Body string
	// OrgID associates the thread with an org for scoping (optional).
	OrgID *string
	// ContactEmail, when set, assigns the ticket to this email address.
	// Only DEFT members may use this field.
	ContactEmail *string
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

	// Privacy: strip ContactEmail for non-DEFT callers.
	if cv != nil && cv.Scope != ScopeAll {
		stripContactEmail(enriched)
	}

	return &enriched[0], nil
}

// UpdateInput holds the fields that may be updated on a global space thread.
type UpdateInput struct {
	// Body replaces the thread body when non-nil.
	Body *string `json:"body"`
	// Status sets the status field in metadata when non-empty.
	Status *string `json:"status"`
	// AssignedTo sets the assigned_to field in metadata. Must be a DEFT org member.
	AssignedTo *string `json:"assigned_to"`
	// IsPinned sets the thread pin state when non-nil.
	IsPinned *bool `json:"is_pinned"`
	// IsHidden sets the thread hidden state when non-nil.
	IsHidden *bool `json:"is_hidden"`
	// IsLocked sets the thread lock state when non-nil.
	IsLocked *bool `json:"is_locked"`
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
	if input.IsPinned != nil {
		t.IsPinned = *input.IsPinned
	}
	if input.IsHidden != nil {
		t.IsHidden = *input.IsHidden
	}
	if input.IsLocked != nil {
		t.IsLocked = *input.IsLocked
	}
	if input.Status != nil {
		patch := fmt.Sprintf(`{"status":%q}`, *input.Status)
		merged, mergeErr := metadata.DeepMerge(t.Metadata, patch)
		if mergeErr != nil {
			return nil, fmt.Errorf("updating status: %w", mergeErr)
		}
		t.Metadata = merged
	}
	if input.AssignedTo != nil {
		// Validate the target user is a DEFT org member (empty string = unassign).
		if *input.AssignedTo != "" {
			isDeft, deftErr := s.repo.IsDeftOrAdmin(ctx, *input.AssignedTo)
			if deftErr != nil {
				return nil, fmt.Errorf("checking assignee: %w", deftErr)
			}
			if !isDeft {
				return nil, fmt.Errorf("assignee must be a DEFT org member")
			}
		}
		patch := fmt.Sprintf(`{"assigned_to":%q}`, *input.AssignedTo)
		merged, mergeErr := metadata.DeepMerge(t.Metadata, patch)
		if mergeErr != nil {
			return nil, fmt.Errorf("updating assigned_to: %w", mergeErr)
		}
		t.Metadata = merged

		// Auto-transition status: open → assigned on assign, assigned → open on unassign.
		// Only applies when the caller did not explicitly set a status in the same request.
		if input.Status == nil {
			curStatus := t.Status // generated column from metadata
			if *input.AssignedTo != "" && (curStatus == "" || curStatus == "open") {
				statusPatch := `{"status":"assigned"}`
				t.Metadata, _ = metadata.DeepMerge(t.Metadata, statusPatch)
			} else if *input.AssignedTo == "" && curStatus == "assigned" {
				statusPatch := `{"status":"open"}`
				t.Metadata, _ = metadata.DeepMerge(t.Metadata, statusPatch)
			}
		}
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
		payload := map[string]any{
			"title":        reloaded.Title,
			"source":       "global-support",
			"participants": []string{reloaded.AuthorID},
		}
		if input.AssignedTo != nil && *input.AssignedTo != "" {
			payload["assigned_to"] = *input.AssignedTo
		}
		s.bus.Publish(eventbus.Event{
			Type:       "thread.updated",
			EntityType: "thread",
			EntityID:   reloaded.ID,
			UserID:     editorID,
			Timestamp:  time.Now(),
			Payload:    payload,
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

// ListMessages returns paginated, author-enriched messages for a thread.
// Returns nil, nil, nil when the space or thread does not exist.
func (s *Service) ListMessages(ctx context.Context, spaceSlug, threadSlug string, params pagination.Params) ([]MessageWithAuthor, *pagination.PageInfo, error) {
	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("listing messages: %w", err)
	}
	if board == nil {
		return nil, nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("listing messages: %w", err)
	}
	if t == nil {
		return nil, nil, nil
	}

	messages, pageInfo, err := s.repo.ListMessages(ctx, t.ID, params)
	if err != nil {
		return nil, nil, err
	}

	enriched, err := s.enrichMessages(ctx, messages)
	if err != nil {
		return nil, nil, err
	}
	return enriched, pageInfo, nil
}

// enrichMessages batch-resolves author display info for a slice of messages.
func (s *Service) enrichMessages(ctx context.Context, messages []models.Message) ([]MessageWithAuthor, error) {
	authorIDs := make([]string, 0, len(messages))
	seen := map[string]bool{}
	for _, m := range messages {
		if m.AuthorID != "" && !seen[m.AuthorID] {
			authorIDs = append(authorIDs, m.AuthorID)
			seen[m.AuthorID] = true
		}
	}

	shadows, err := s.repo.GetUserShadowsByIDs(ctx, authorIDs)
	if err != nil {
		return nil, fmt.Errorf("enriching messages: %w", err)
	}
	orgNames, err := s.repo.GetUserPrimaryOrgNames(ctx, authorIDs)
	if err != nil {
		return nil, fmt.Errorf("enriching messages: %w", err)
	}

	result := make([]MessageWithAuthor, len(messages))
	for i, m := range messages {
		rich := MessageWithAuthor{Message: m}
		if s, ok := shadows[m.AuthorID]; ok {
			rich.AuthorName = s.DisplayName
		}
		rich.AuthorOrg = orgNames[m.AuthorID]
		result[i] = rich
	}
	return result, nil
}

// CreateMessageInput holds data needed to create a message in a global space thread.
type CreateMessageInput struct {
	Body string `json:"body"`
}

// CreateMessage adds a message to a thread in the given global space.
// Returns nil, nil when the space or thread does not exist.
func (s *Service) CreateMessage(ctx context.Context, spaceSlug, threadSlug, authorID string, input CreateMessageInput) (*models.Message, error) {
	if input.Body == "" {
		return nil, fmt.Errorf("body is required")
	}

	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}
	if board == nil {
		return nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}
	if t == nil {
		return nil, nil
	}
	if t.IsLocked {
		return nil, fmt.Errorf("thread is locked")
	}

	msg := &models.Message{
		ThreadID: t.ID,
		Body:     input.Body,
		AuthorID: authorID,
		Metadata: "{}",
		Type:     models.MessageTypeComment,
	}
	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// ToggleVote toggles the authenticated user's vote on a forum thread.
// Returns nil, nil when the space or thread does not exist.
func (s *Service) ToggleVote(ctx context.Context, spaceSlug, threadSlug, userID string) (*vote.VoteResult, error) {
	if s.voteSvc == nil {
		return nil, fmt.Errorf("vote service not available")
	}

	board, err := s.repo.FindDefaultBoard(ctx, spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("toggling vote: %w", err)
	}
	if board == nil {
		return nil, nil
	}

	t, err := s.repo.FindThreadBySlug(ctx, board.ID, threadSlug)
	if err != nil {
		return nil, fmt.Errorf("toggling vote: %w", err)
	}
	if t == nil {
		return nil, nil
	}

	// Use default viewer role / no tier bonus for forum votes.
	result, err := s.voteSvc.Toggle(ctx, t.ID, userID, models.RoleViewer, "free")
	if err != nil {
		return nil, fmt.Errorf("toggling vote: %w", err)
	}
	return result, nil
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

	// When contact_email is provided, resolve the target user or leave orphaned.
	effectiveAuthorID := authorID
	contactEmail := ""
	if input.ContactEmail != nil && *input.ContactEmail != "" {
		// Only DEFT members may assign tickets by email.
		isDeft, deftErr := s.repo.IsDeftOrAdmin(ctx, authorID)
		if deftErr != nil {
			return nil, fmt.Errorf("checking DEFT membership: %w", deftErr)
		}
		if !isDeft {
			return nil, fmt.Errorf("only DEFT members may use contact_email")
		}
		contactEmail = *input.ContactEmail

		// Look up registered user by email.
		shadow, lookupErr := s.repo.FindUserShadowByEmail(ctx, contactEmail)
		if lookupErr != nil {
			return nil, fmt.Errorf("looking up contact email: %w", lookupErr)
		}
		if shadow != nil {
			// Known user — set them as the ticket author.
			effectiveAuthorID = shadow.ClerkUserID
		}
		// If shadow == nil the ticket is orphaned: authorID stays as the DEFT
		// member and contactEmail bridges the gap until the user registers.
	}

	t := &models.Thread{
		BoardID:      board.ID,
		Title:        input.Title,
		Body:         input.Body,
		Slug:         threadSlug,
		Metadata:     "{}",
		AuthorID:     effectiveAuthorID,
		ContactEmail: contactEmail,
		ThreadType:   threadTypeForSpace(spaceSlug),
		Visibility:   models.ThreadVisibilityOrgOnly,
		OrgID:        input.OrgID,
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
