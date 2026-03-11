// Package message provides CRUD operations for messages nested under threads.
package message

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// Service errors.
var (
	ErrNotFound     = errors.New("message not found")
	ErrBodyRequired = errors.New("body is required")
	ErrInvalidMeta  = errors.New("invalid metadata JSON")
	ErrInvalidType  = errors.New("invalid message type")
	ErrThreadLocked = errors.New("thread is locked")
	ErrNotAuthor    = errors.New("only the author can update this message")
)

// --- Repository ---

// Repository handles database operations for messages.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new message repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create inserts a new message.
func (r *Repository) Create(ctx context.Context, m *models.Message) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// GetByID retrieves a message by ID within a thread.
func (r *Repository) GetByID(ctx context.Context, threadID, id string) (*models.Message, error) {
	var m models.Message
	if err := r.db.WithContext(ctx).Where("thread_id = ? AND id = ?", threadID, id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// List retrieves messages for a thread with cursor pagination.
func (r *Repository) List(ctx context.Context, threadID, cursor string, limit int) ([]models.Message, error) {
	q := r.db.WithContext(ctx).Where("thread_id = ?", threadID).Order("id ASC")
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	q = q.Limit(limit + 1)
	var messages []models.Message
	return messages, q.Find(&messages).Error
}

// Update saves changes to a message.
func (r *Repository) Update(ctx context.Context, m *models.Message) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Delete soft-deletes a message.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Message{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// CreateRevision creates a revision record.
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

// CreateInput holds parameters for creating a message.
type CreateInput struct {
	Body     string `json:"body"`
	Metadata string `json:"metadata"`
	Type     string `json:"type"`
}

// UpdateInput holds parameters for updating a message.
type UpdateInput struct {
	Body     *string `json:"body,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

// ThreadChecker checks thread lock state.
type ThreadChecker interface {
	IsLocked(ctx context.Context, threadID string) (bool, error)
}

// Service provides business logic for message operations.
type Service struct {
	repo          *Repository
	threadChecker ThreadChecker
}

// NewService creates a new message service.
func NewService(repo *Repository, threadChecker ThreadChecker) *Service {
	return &Service{repo: repo, threadChecker: threadChecker}
}

// Create creates a new message in a thread.
func (s *Service) Create(ctx context.Context, threadID, authorID string, input CreateInput) (*models.Message, error) {
	if input.Body == "" {
		return nil, ErrBodyRequired
	}
	// Check if thread is locked.
	if s.threadChecker != nil {
		locked, err := s.threadChecker.IsLocked(ctx, threadID)
		if err != nil {
			return nil, err
		}
		if locked {
			return nil, ErrThreadLocked
		}
	}
	msgType := models.MessageType(input.Type)
	if input.Type == "" {
		msgType = models.MessageTypeComment
	} else if !msgType.IsValid() {
		return nil, ErrInvalidType
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, ErrInvalidMeta
		}
	} else {
		input.Metadata = "{}"
	}
	m := &models.Message{
		ThreadID: threadID,
		Body:     input.Body,
		AuthorID: authorID,
		Metadata: input.Metadata,
		Type:     msgType,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetByID retrieves a message by ID within a thread.
func (s *Service) GetByID(ctx context.Context, threadID, id string) (*models.Message, error) {
	m, err := s.repo.GetByID(ctx, threadID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

// List returns paginated messages for a thread.
func (s *Service) List(ctx context.Context, threadID, cursor string, limit int) ([]models.Message, bool, error) {
	messages, err := s.repo.List(ctx, threadID, cursor, limit)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

// Update updates a message (author-only) and creates a revision.
func (s *Service) Update(ctx context.Context, threadID, msgID, editorID string, input UpdateInput) (*models.Message, error) {
	m, err := s.repo.GetByID(ctx, threadID, msgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	// Author-only check.
	if m.AuthorID != editorID {
		return nil, ErrNotAuthor
	}
	// Capture previous state for revision.
	prevContent, _ := json.Marshal(map[string]interface{}{
		"body": m.Body, "metadata": m.Metadata,
	})
	if input.Body != nil {
		if *input.Body == "" {
			return nil, ErrBodyRequired
		}
		m.Body = *input.Body
	}
	if input.Metadata != nil {
		merged, err := metadata.DeepMerge(m.Metadata, *input.Metadata)
		if err != nil {
			return nil, ErrInvalidMeta
		}
		m.Metadata = merged
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	version, _ := s.repo.CountRevisions(ctx, "message", m.ID)
	_ = s.repo.CreateRevision(ctx, &models.Revision{
		EntityType:      "message",
		EntityID:        m.ID,
		Version:         version + 1,
		PreviousContent: string(prevContent),
		EditorID:        editorID,
	})
	return m, nil
}

// Delete soft-deletes a message.
func (s *Service) Delete(ctx context.Context, threadID, id string) error {
	m, err := s.repo.GetByID(ctx, threadID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, m.ID)
}

// --- Handler ---

// Handler provides HTTP handlers for message endpoints.
type Handler struct {
	service      *Service
	threadGetter ThreadGetter
}

// ThreadGetter resolves thread ID from URL params.
type ThreadGetter interface {
	ResolveThreadID(ctx context.Context, orgRef, spaceRef, boardRef, threadRef string) (string, error)
}

// NewHandler creates a new message handler.
func NewHandler(service *Service, threadGetter ThreadGetter) *Handler {
	return &Handler{service: service, threadGetter: threadGetter}
}

// Create handles POST .../messages.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	threadID, ok := h.resolveThread(w, r)
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
	m, err := h.service.Create(r.Context(), threadID, uc.UserID, input)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(m)
}

// List handles GET .../messages.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	threadID, ok := h.resolveThread(w, r)
	if !ok {
		return
	}
	params := pagination.Parse(r)
	cursorID := decodeCursorID(params.Cursor)
	messages, hasMore, err := h.service.List(r.Context(), threadID, cursorID, params.Limit)
	if err != nil {
		apierrors.InternalError(w, "failed to list messages")
		return
	}
	pageInfo := pagination.PageInfo{HasMore: hasMore}
	if hasMore && len(messages) > 0 {
		lastID, _ := uuid.Parse(messages[len(messages)-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": messages, "page_info": pageInfo,
	})
}

// Get handles GET .../messages/{message}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	threadID, ok := h.resolveThread(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "message")
	m, err := h.service.GetByID(r.Context(), threadID, id)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// Update handles PATCH .../messages/{message}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	threadID, ok := h.resolveThread(w, r)
	if !ok {
		return
	}
	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}
	id := chi.URLParam(r, "message")
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	m, err := h.service.Update(r.Context(), threadID, id, uc.UserID, input)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// Delete handles DELETE .../messages/{message}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	threadID, ok := h.resolveThread(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "message")
	if err := h.service.Delete(r.Context(), threadID, id); err != nil {
		writeMessageError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveThread(w http.ResponseWriter, r *http.Request) (string, bool) {
	orgRef := chi.URLParam(r, "org")
	spaceRef := chi.URLParam(r, "space")
	boardRef := chi.URLParam(r, "board")
	threadRef := chi.URLParam(r, "thread")
	if orgRef == "" || spaceRef == "" || boardRef == "" || threadRef == "" {
		apierrors.BadRequest(w, "org, space, board, and thread identifiers are required")
		return "", false
	}
	threadID, err := h.threadGetter.ResolveThreadID(r.Context(), orgRef, spaceRef, boardRef, threadRef)
	if err != nil {
		apierrors.NotFound(w, "thread not found")
		return "", false
	}
	return threadID, true
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

func writeMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierrors.NotFound(w, err.Error())
	case errors.Is(err, ErrBodyRequired):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "body", Message: err.Error()},
		})
	case errors.Is(err, ErrInvalidMeta):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "metadata", Message: err.Error()},
		})
	case errors.Is(err, ErrInvalidType):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "type", Message: err.Error()},
		})
	case errors.Is(err, ErrThreadLocked):
		apierrors.Conflict(w, err.Error())
	case errors.Is(err, ErrNotAuthor):
		apierrors.Forbidden(w, err.Error())
	default:
		apierrors.InternalError(w, "internal server error")
	}
}
