// Package board provides CRUD operations for boards nested under spaces.
package board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/metadata"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// Service errors.
var (
	ErrNotFound     = errors.New("board not found")
	ErrNameRequired = errors.New("name is required")
	ErrInvalidMeta  = errors.New("invalid metadata JSON")
	ErrBoardLocked  = errors.New("board is locked")
)

// --- Repository ---

// Repository handles database operations for boards.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new board repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new board.
func (r *Repository) Create(ctx context.Context, b *models.Board) error {
	return r.db.WithContext(ctx).Create(b).Error
}

// GetByIDOrSlug retrieves a board by ID or slug within a space.
func (r *Repository) GetByIDOrSlug(ctx context.Context, spaceID, ref string) (*models.Board, error) {
	var b models.Board
	q := r.db.WithContext(ctx).Where("space_id = ?", spaceID)
	if isUUID(ref) {
		q = q.Where("id = ?", ref)
	} else {
		q = q.Where("slug = ?", ref)
	}
	if err := q.First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// GetByID retrieves a board by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*models.Board, error) {
	var b models.Board
	if err := r.db.WithContext(ctx).First(&b, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// List retrieves boards for a space with cursor pagination.
func (r *Repository) List(ctx context.Context, spaceID, cursor string, limit int) ([]models.Board, error) {
	q := r.db.WithContext(ctx).Where("space_id = ?", spaceID).Order("id ASC")
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	q = q.Limit(limit + 1)
	var boards []models.Board
	return boards, q.Find(&boards).Error
}

// Update saves changes to a board.
func (r *Repository) Update(ctx context.Context, b *models.Board) error {
	return r.db.WithContext(ctx).Save(b).Error
}

// Delete soft-deletes a board.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Board{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// SlugExists checks if a slug exists within a space.
func (r *Repository) SlugExists(ctx context.Context, spaceID, s string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Board{}).
		Where("space_id = ? AND slug = ?", spaceID, s).Count(&count).Error
	return count > 0, err
}

// --- Service ---

// CreateInput holds parameters for creating a board.
type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

// UpdateInput holds parameters for updating a board.
type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata    *string `json:"metadata,omitempty"`
}

// Service provides business logic for board operations.
type Service struct {
	repo *Repository
}

// NewService creates a new board service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new board in a space.
func (s *Service) Create(ctx context.Context, spaceID string, input CreateInput) (*models.Board, error) {
	if input.Name == "" {
		return nil, ErrNameRequired
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, ErrInvalidMeta
		}
	} else {
		input.Metadata = "{}"
	}

	boardSlug := slug.Generate(input.Name)
	base := boardSlug
	for i := 1; ; i++ {
		exists, err := s.repo.SlugExists(ctx, spaceID, boardSlug)
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
		boardSlug = fmt.Sprintf("%s-%d", base, i)
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

// GetByRef retrieves a board by ID or slug within a space.
func (s *Service) GetByRef(ctx context.Context, spaceID, ref string) (*models.Board, error) {
	b, err := s.repo.GetByIDOrSlug(ctx, spaceID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

// List returns paginated boards for a space.
func (s *Service) List(ctx context.Context, spaceID, cursor string, limit int) ([]models.Board, bool, error) {
	boards, err := s.repo.List(ctx, spaceID, cursor, limit)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(boards) > limit
	if hasMore {
		boards = boards[:limit]
	}
	return boards, hasMore, nil
}

// Update updates a board with deep-merge for metadata.
func (s *Service) Update(ctx context.Context, spaceID, ref string, input UpdateInput) (*models.Board, error) {
	b, err := s.repo.GetByIDOrSlug(ctx, spaceID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if input.Name != nil {
		if *input.Name == "" {
			return nil, ErrNameRequired
		}
		b.Name = *input.Name
		newSlug := slug.Generate(*input.Name)
		if newSlug != b.Slug {
			base := newSlug
			for i := 1; ; i++ {
				exists, err := s.repo.SlugExists(ctx, spaceID, newSlug)
				if err != nil {
					return nil, err
				}
				if !exists || newSlug == b.Slug {
					break
				}
				newSlug = fmt.Sprintf("%s-%d", base, i)
			}
			b.Slug = newSlug
		}
	}
	if input.Description != nil {
		b.Description = *input.Description
	}
	if input.Metadata != nil {
		merged, err := metadata.DeepMerge(b.Metadata, *input.Metadata)
		if err != nil {
			return nil, ErrInvalidMeta
		}
		b.Metadata = merged
	}
	if err := s.repo.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// SetLock sets the lock state of a board.
func (s *Service) SetLock(ctx context.Context, spaceID, ref string, locked bool) (*models.Board, error) {
	b, err := s.repo.GetByIDOrSlug(ctx, spaceID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	b.IsLocked = locked
	if err := s.repo.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

// Delete soft-deletes a board.
func (s *Service) Delete(ctx context.Context, spaceID, ref string) error {
	b, err := s.repo.GetByIDOrSlug(ctx, spaceID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, b.ID)
}

// --- Handler ---

// Handler provides HTTP handlers for board endpoints.
type Handler struct {
	service     *Service
	spaceGetter SpaceGetter
}

// SpaceGetter resolves space ID from URL params.
type SpaceGetter interface {
	ResolveSpaceID(ctx context.Context, orgRef, spaceRef string) (string, error)
}

// NewHandler creates a new board handler.
func NewHandler(service *Service, spaceGetter SpaceGetter) *Handler {
	return &Handler{service: service, spaceGetter: spaceGetter}
}

// Create handles POST .../boards.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	b, err := h.service.Create(r.Context(), spaceID, input)
	if err != nil {
		writeBoardError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(b)
}

// List handles GET .../boards.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	params := pagination.Parse(r)
	cursorID := decodeCursorID(params.Cursor)
	boards, hasMore, err := h.service.List(r.Context(), spaceID, cursorID, params.Limit)
	if err != nil {
		apierrors.InternalError(w, "failed to list boards")
		return
	}
	pageInfo := pagination.PageInfo{HasMore: hasMore}
	if hasMore && len(boards) > 0 {
		lastID, _ := uuid.Parse(boards[len(boards)-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": boards, "page_info": pageInfo,
	})
}

// Get handles GET .../boards/{board}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "board")
	b, err := h.service.GetByRef(r.Context(), spaceID, ref)
	if err != nil {
		writeBoardError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// Update handles PATCH .../boards/{board}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "board")
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	b, err := h.service.Update(r.Context(), spaceID, ref, input)
	if err != nil {
		writeBoardError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// Lock handles POST .../boards/{board}/lock.
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "board")
	b, err := h.service.SetLock(r.Context(), spaceID, ref, true)
	if err != nil {
		writeBoardError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// Unlock handles POST .../boards/{board}/unlock.
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "board")
	b, err := h.service.SetLock(r.Context(), spaceID, ref, false)
	if err != nil {
		writeBoardError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// Delete handles DELETE .../boards/{board}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := h.resolveSpace(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "board")
	if err := h.service.Delete(r.Context(), spaceID, ref); err != nil {
		writeBoardError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveSpace(w http.ResponseWriter, r *http.Request) (string, bool) {
	orgRef := chi.URLParam(r, "org")
	spaceRef := chi.URLParam(r, "space")
	if orgRef == "" || spaceRef == "" {
		apierrors.BadRequest(w, "org and space identifiers are required")
		return "", false
	}
	spaceID, err := h.spaceGetter.ResolveSpaceID(r.Context(), orgRef, spaceRef)
	if err != nil {
		apierrors.NotFound(w, "space not found")
		return "", false
	}
	return spaceID, true
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

func writeBoardError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierrors.NotFound(w, err.Error())
	case errors.Is(err, ErrNameRequired):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "name", Message: err.Error()},
		})
	case errors.Is(err, ErrInvalidMeta):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "metadata", Message: err.Error()},
		})
	case errors.Is(err, ErrBoardLocked):
		apierrors.Conflict(w, err.Error())
	default:
		apierrors.InternalError(w, "internal server error")
	}
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
