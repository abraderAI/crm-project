// Package space provides CRUD operations for spaces nested under orgs.
package space

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
	ErrNotFound     = errors.New("space not found")
	ErrOrgNotFound  = errors.New("org not found")
	ErrNameRequired = errors.New("name is required")
	ErrInvalidMeta  = errors.New("invalid metadata JSON")
	ErrInvalidType  = errors.New("invalid space type")
)

// --- Repository ---

// Repository handles database operations for spaces.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new space repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new space.
func (r *Repository) Create(ctx context.Context, space *models.Space) error {
	return r.db.WithContext(ctx).Create(space).Error
}

// GetByIDOrSlug retrieves a space by ID or slug within an org.
func (r *Repository) GetByIDOrSlug(ctx context.Context, orgID, ref string) (*models.Space, error) {
	var space models.Space
	q := r.db.WithContext(ctx).Where("org_id = ?", orgID)
	if isUUID(ref) {
		q = q.Where("id = ?", ref)
	} else {
		q = q.Where("slug = ?", ref)
	}
	if err := q.First(&space).Error; err != nil {
		return nil, err
	}
	return &space, nil
}

// List retrieves spaces for an org with cursor pagination.
func (r *Repository) List(ctx context.Context, orgID, cursor string, limit int) ([]models.Space, error) {
	q := r.db.WithContext(ctx).Where("org_id = ?", orgID).Order("id ASC")
	if cursor != "" {
		q = q.Where("id > ?", cursor)
	}
	q = q.Limit(limit + 1)
	var spaces []models.Space
	return spaces, q.Find(&spaces).Error
}

// Update saves changes to a space.
func (r *Repository) Update(ctx context.Context, space *models.Space) error {
	return r.db.WithContext(ctx).Save(space).Error
}

// Delete soft-deletes a space.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Space{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// SlugExists checks if a slug exists within an org.
func (r *Repository) SlugExists(ctx context.Context, orgID, s string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Space{}).
		Where("org_id = ? AND slug = ?", orgID, s).Count(&count).Error
	return count > 0, err
}

// --- Service ---

// CreateInput holds parameters for creating a space.
type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
	Type        string `json:"type"`
}

// UpdateInput holds parameters for updating a space.
type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Metadata    *string `json:"metadata,omitempty"`
	Type        *string `json:"type,omitempty"`
}

// Service provides business logic for space operations.
type Service struct {
	repo *Repository
}

// NewService creates a new space service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new space in an org.
func (s *Service) Create(ctx context.Context, orgID string, input CreateInput) (*models.Space, error) {
	if input.Name == "" {
		return nil, ErrNameRequired
	}
	spaceType := models.SpaceType(input.Type)
	if input.Type == "" {
		spaceType = models.SpaceTypeGeneral
	} else if !spaceType.IsValid() {
		return nil, ErrInvalidType
	}
	if input.Metadata != "" {
		if err := metadata.Validate(input.Metadata); err != nil {
			return nil, ErrInvalidMeta
		}
	} else {
		input.Metadata = "{}"
	}

	spaceSlug := slug.Generate(input.Name)
	base := spaceSlug
	for i := 1; ; i++ {
		exists, err := s.repo.SlugExists(ctx, orgID, spaceSlug)
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
		spaceSlug = fmt.Sprintf("%s-%d", base, i)
	}

	sp := &models.Space{
		OrgID:       orgID,
		Name:        input.Name,
		Slug:        spaceSlug,
		Description: input.Description,
		Metadata:    input.Metadata,
		Type:        spaceType,
	}
	if err := s.repo.Create(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// GetByRef retrieves a space by ID or slug within an org.
func (s *Service) GetByRef(ctx context.Context, orgID, ref string) (*models.Space, error) {
	sp, err := s.repo.GetByIDOrSlug(ctx, orgID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return sp, nil
}

// List returns paginated spaces for an org.
func (s *Service) List(ctx context.Context, orgID, cursor string, limit int) ([]models.Space, bool, error) {
	spaces, err := s.repo.List(ctx, orgID, cursor, limit)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(spaces) > limit
	if hasMore {
		spaces = spaces[:limit]
	}
	return spaces, hasMore, nil
}

// Update updates a space with deep-merge for metadata.
func (s *Service) Update(ctx context.Context, orgID, ref string, input UpdateInput) (*models.Space, error) {
	sp, err := s.repo.GetByIDOrSlug(ctx, orgID, ref)
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
		sp.Name = *input.Name
		newSlug := slug.Generate(*input.Name)
		if newSlug != sp.Slug {
			base := newSlug
			for i := 1; ; i++ {
				exists, err := s.repo.SlugExists(ctx, orgID, newSlug)
				if err != nil {
					return nil, err
				}
				if !exists || newSlug == sp.Slug {
					break
				}
				newSlug = fmt.Sprintf("%s-%d", base, i)
			}
			sp.Slug = newSlug
		}
	}
	if input.Description != nil {
		sp.Description = *input.Description
	}
	if input.Type != nil {
		t := models.SpaceType(*input.Type)
		if !t.IsValid() {
			return nil, ErrInvalidType
		}
		sp.Type = t
	}
	if input.Metadata != nil {
		merged, err := metadata.DeepMerge(sp.Metadata, *input.Metadata)
		if err != nil {
			return nil, ErrInvalidMeta
		}
		sp.Metadata = merged
	}
	if err := s.repo.Update(ctx, sp); err != nil {
		return nil, err
	}
	return sp, nil
}

// Delete soft-deletes a space.
func (s *Service) Delete(ctx context.Context, orgID, ref string) error {
	sp, err := s.repo.GetByIDOrSlug(ctx, orgID, ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, sp.ID)
}

// --- Handler ---

// Handler provides HTTP handlers for space endpoints.
type Handler struct {
	service   *Service
	orgGetter OrgGetter
}

// OrgGetter resolves org ID from ref (ID or slug).
type OrgGetter interface {
	ResolveOrgID(ctx context.Context, ref string) (string, error)
}

// NewHandler creates a new space handler.
func NewHandler(service *Service, orgGetter OrgGetter) *Handler {
	return &Handler{service: service, orgGetter: orgGetter}
}

// Create handles POST /v1/orgs/{org}/spaces.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.resolveOrg(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	sp, err := h.service.Create(r.Context(), orgID, input)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sp)
}

// List handles GET /v1/orgs/{org}/spaces.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.resolveOrg(w, r)
	if !ok {
		return
	}
	params := pagination.Parse(r)
	cursorID := decodeCursorID(params.Cursor)
	spaces, hasMore, err := h.service.List(r.Context(), orgID, cursorID, params.Limit)
	if err != nil {
		apierrors.InternalError(w, "failed to list spaces")
		return
	}
	pageInfo := pagination.PageInfo{HasMore: hasMore}
	if hasMore && len(spaces) > 0 {
		lastID, _ := uuid.Parse(spaces[len(spaces)-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": spaces, "page_info": pageInfo,
	})
}

// Get handles GET /v1/orgs/{org}/spaces/{space}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.resolveOrg(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "space")
	sp, err := h.service.GetByRef(r.Context(), orgID, ref)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sp)
}

// Update handles PATCH /v1/orgs/{org}/spaces/{space}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.resolveOrg(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "space")
	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	sp, err := h.service.Update(r.Context(), orgID, ref, input)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sp)
}

// Delete handles DELETE /v1/orgs/{org}/spaces/{space}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, ok := h.resolveOrg(w, r)
	if !ok {
		return
	}
	ref := chi.URLParam(r, "space")
	if err := h.service.Delete(r.Context(), orgID, ref); err != nil {
		writeSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveOrg(w http.ResponseWriter, r *http.Request) (string, bool) {
	ref := chi.URLParam(r, "org")
	if ref == "" {
		apierrors.BadRequest(w, "org identifier is required")
		return "", false
	}
	orgID, err := h.orgGetter.ResolveOrgID(r.Context(), ref)
	if err != nil {
		apierrors.NotFound(w, "org not found")
		return "", false
	}
	return orgID, true
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

func writeSpaceError(w http.ResponseWriter, err error) {
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
	case errors.Is(err, ErrInvalidType):
		apierrors.ValidationError(w, "validation failed", []apierrors.FieldError{
			{Field: "type", Message: err.Error()},
		})
	default:
		apierrors.InternalError(w, "internal server error")
	}
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
