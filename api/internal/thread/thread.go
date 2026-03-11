// Package thread provides CRUD operations for threads nested under boards.
package thread

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service errors.
var (
	ErrNotFound      = errors.New("thread not found")
	ErrTitleRequired = errors.New("title is required")
	ErrInvalidMeta   = errors.New("invalid metadata JSON")
	ErrBoardLocked   = errors.New("board is locked")
	ErrThreadLocked  = errors.New("thread is locked")
)

// --- Repository ---

// Repository handles database operations for threads.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new thread repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create inserts a new thread.
func (r *Repository) Create(ctx context.Context, t *models.Thread) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByIDOrSlug retrieves a thread by ID or slug within a board.
func (r *Repository) GetByIDOrSlug(ctx context.Context, boardID, ref string) (*models.Thread, error) {
	var t models.Thread
	q := r.db.WithContext(ctx).Where("board_id = ?", boardID)
	if isUUID(ref) {
		q = q.Where("id = ?", ref)
	} else {
		q = q.Where("slug = ?", ref)
	}
	if err := q.First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByID retrieves a thread by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*models.Thread, error) {
	var t models.Thread
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// List retrieves threads for a board with cursor pagination and metadata filters.
func (r *Repository) List(ctx context.Context, boardID, cursor string, limit int, filters []metadata.Filter) ([]models.Thread, error) {
	q := r.db.WithContext(ctx).Where("board_id = ?", boardID).Order("id ASC")
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	// Apply metadata filters.
	conditions, args := metadata.ToSQLConditions(filters)
	for i, cond := range conditions {
		q = q.Where(cond, args[i])
	}
	q = q.Limit(limit + 1)
	var threads []models.Thread
	return threads, q.Find(&threads).Error
}

// Update saves changes to a thread.
func (r *Repository) Update(ctx context.Context, t *models.Thread) error {
	return r.db.WithContext(ctx).Save(t).Error
}

// Delete soft-deletes a thread.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Thread{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// SlugExists checks if a slug exists within a board.
func (r *Repository) SlugExists(ctx context.Context, boardID, s string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Thread{}).
		Where("board_id = ? AND slug = ?", boardID, s).Count(&count).Error
	return count > 0, err
}

// CreateRevision creates a revision record for the thread.
func (r *Repository) CreateRevision(ctx context.Context, rev *models.Revision) error {
	return r.db.WithContext(ctx).Create(rev).Error
}

// CountRevisions returns the number of revisions for an entity.
func (r *Repository) CountRevisions(ctx context.Context, entityType, entityID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Revision{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).Count(&count).Error
	return int(count), err
}

// --- Service ---

// CreateInput holds parameters for creating a thread.
type CreateInput struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Metadata string `json:"metadata"`
}

// UpdateInput holds parameters for updating a thread.
type UpdateInput struct {
	Title    *string `json:"title,omitempty"`
	Body     *string `json:"body,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

// BoardChecker checks board lock state.
type BoardChecker interface {
	IsLocked(ctx context.Context, boardID string) (bool, error)
}

// Service provides business logic for thread operations.
type Service struct {
	repo         *Repository
	boardChecker BoardChecker
}

// NewService creates a new thread service.
func NewService(repo *Repository, boardChecker BoardChecker) *Service {
	return &Service{repo: repo, boardChecker: boardChecker}
}

// Create creates a new thread in a board.
func (s *Service) Create(ctx context.Context, boardID, authorID string, input CreateInput) (*models.Thread, error) {
	if input.Title == "" {
		return nil, ErrTitleRequired
	}
	// Check if board is locked.
	if s.boardChecker != nil {
		locked, err := s.boardChecker.IsLocked(ctx, boardID)
		if err != nil {
			return nil, err
		}
		if locked {
			return nil, ErrBoardLocked
		}
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, ErrInvalidMeta
		}
	} else {
		input.Metadata = "{}"
	}
	threadSlug := slug.Generate(input.Title)
	base := threadSlug
	for i := 1; ; i++ {
		exists, err := s.repo.SlugExists(ctx, boardID, threadSlug)
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
		threadSlug = fmt.Sprintf("%s-%d", base, i)
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

// GetByRef retrieves a thread by ID or slug within a board.
func (s *Service) GetByRef(ctx context.Context, boardID, ref string) (*models.Thread, error) {
	t, err := s.repo.GetByIDOrSlug(ctx, boardID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

// List returns paginated threads for a board with metadata filters.
func (s *Service) List(ctx context.Context, boardID, cursor string, limit int, filters []metadata.Filter) ([]models.Thread, bool, error) {
	threads, err := s.repo.List(ctx, boardID, cursor, limit, filters)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(threads) > limit
	if hasMore {
		threads = threads[:limit]
	}
	return threads, hasMore, nil
}

// Update updates a thread with deep-merge for metadata and creates a revision.
func (s *Service) Update(ctx context.Context, boardID, ref, editorID string, input UpdateInput) (*models.Thread, error) {
	t, err := s.repo.GetByIDOrSlug(ctx, boardID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Capture previous state for revision.
	prevContent, _ := json.Marshal(map[string]interface{}{
		"title": t.Title, "body": t.Body, "metadata": t.Metadata,
	})

	if input.Title != nil {
		if *input.Title == "" {
			return nil, ErrTitleRequired
		}
		t.Title = *input.Title
	}
	if input.Body != nil {
		t.Body = *input.Body
	}
	if input.Metadata != nil {
		merged, err := metadata.DeepMerge(t.Metadata, *input.Metadata)
		if err != nil {
			return nil, ErrInvalidMeta
		}
		t.Metadata = merged
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}

	// Create revision.
	version, _ := s.repo.CountRevisions(ctx, "thread", t.ID)
	_ = s.repo.CreateRevision(ctx, &models.Revision{
		EntityType:      "thread",
		EntityID:        t.ID,
		Version:         version + 1,
		PreviousContent: string(prevContent),
		EditorID:        editorID,
	})
	return t, nil
}

// SetPin sets the pin state of a thread.
func (s *Service) SetPin(ctx context.Context, boardID, ref string, pinned bool) (*models.Thread, error) {
	t, err := s.repo.GetByIDOrSlug(ctx, boardID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.IsPinned = pinned
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// SetLock sets the lock state of a thread.
func (s *Service) SetLock(ctx context.Context, boardID, ref string, locked bool) (*models.Thread, error) {
	t, err := s.repo.GetByIDOrSlug(ctx, boardID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	t.IsLocked = locked
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete soft-deletes a thread.
func (s *Service) Delete(ctx context.Context, boardID, ref string) error {
	t, err := s.repo.GetByIDOrSlug(ctx, boardID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, t.ID)
}

// --- Handler ---

// Handler provides HTTP handlers for thread endpoints.
type Handler struct {
	service     *Service
	boardGetter BoardGetter
}

// BoardGetter resolves board ID from URL params.
type BoardGetter interface {
	ResolveBoardID(ctx context.Context, orgRef, spaceRef, boardRef string) (string, error)
}

// NewHandler creates a new thread handler.
func NewHandler(service *Service, boardGetter BoardGetter) *Handler {
	return &Handler{service: service, boardGetter: boardGetter}
}

// Create handles POST .../threads.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	t, err := h.service.Create(r.Context(), boardID, uc.UserID, input)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

// List handles GET .../threads.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	params := pagination.Parse(r)
	cursorID := decodeCursorID(params.Cursor)
	filters := metadata.ParseFilters(r)
	threads, hasMore, err := h.service.List(r.Context(), boardID, cursorID, params.Limit, filters)
	if err != nil {
		apierrors.InternalError(w, "failed to list threads")
		return
	}
	pageInfo := pagination.PageInfo{HasMore: hasMore}
	if hasMore && len(threads) > 0 {
		lastID, _ := uuid.Parse(threads[len(threads)-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": threads, "page_info": pageInfo,
	})
}

// Get handles GET .../threads/{thread}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "thread")
	t, err := h.service.GetByRef(r.Context(), boardID, ref)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

// Update handles PATCH .../threads/{thread}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}
	ref := chi.URLParam(r, "thread")
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	t, err := h.service.Update(r.Context(), boardID, ref, uc.UserID, input)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

// Pin handles POST .../threads/{thread}/pin.
func (h *Handler) Pin(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "thread")
	t, err := h.service.SetPin(r.Context(), boardID, ref, true)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

// Unpin handles POST .../threads/{thread}/unpin.
func (h *Handler) Unpin(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "thread")
	t, err := h.service.SetPin(r.Context(), boardID, ref, false)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

// Lock handles POST .../threads/{thread}/lock.
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "thread")
	t, err := h.service.SetLock(r.Context(), boardID, ref, true)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

// Unlock handles POST .../threads/{thread}/unlock.
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "thread")
	t, err := h.service.SetLock(r.Context(), boardID, ref, false)
	if err != nil {
		writeThreadError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}

// Delete handles DELETE .../threads/{thread}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	boardID, ok := h.resolveBoard(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "thread")
	if err := h.service.Delete(r.Context(), boardID, ref); err != nil {
		writeThreadError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveBoard(w http.ResponseWriter, r *http.Request) (string, bool) {
	orgRef := chi.URLParam(r, "org")
	spaceRef := chi.URLParam(r, "space")
	boardRef := chi.URLParam(r, "board")
	if orgRef == "" || spaceRef == "" || boardRef == "" {
		apierrors.BadRequest(w, "org, space, and board identifiers are required")
		return "", false
	}
	boardID, err := h.boardGetter.ResolveBoardID(r.Context(), orgRef, spaceRef, boardRef)
	if err != nil {
		apierrors.NotFound(w, "board not found")
		return "", false
	}
	return boardID, true
}

func decodeCursorID(cursor string) string {
	if cursor == "" {
		return ""
	}
	decoded, err := pagination.DecodeCursor(cursor)
	if err != nil || decoded == uuid.Nil {
		return ""
	}
	return decoded.String()
}

func writeThreadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierrors.NotFound(w, err.Error())
	case errors.Is(err, ErrTitleRequired):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "title", Message: err.Error()},
		})
	case errors.Is(err, ErrInvalidMeta):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "metadata", Message: err.Error()},
		})
	case errors.Is(err, ErrBoardLocked):
		apierrors.Conflict(w, err.Error())
	case errors.Is(err, ErrThreadLocked):
		apierrors.Conflict(w, err.Error())
	default:
		apierrors.InternalError(w, "internal server error")
	}
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
