// Package revision provides revision history endpoints for threads and messages.
package revision

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Repository handles database operations for revisions.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new revision repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// List returns revisions for an entity ordered by version descending.
func (r *Repository) List(ctx context.Context, entityType, entityID string, params pagination.Params) ([]models.Revision, *pagination.PageInfo, error) {
	var revisions []models.Revision
	query := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("id DESC")

	if params.Cursor != "" {
		cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("id < ?", cursorID.String())
	}

	if err := query.Limit(params.Limit + 1).Find(&revisions).Error; err != nil {
		return nil, nil, fmt.Errorf("listing revisions: %w", err)
	}

	pageInfo := &pagination.PageInfo{}
	if len(revisions) > params.Limit {
		pageInfo.HasMore = true
		lastID, _ := uuid.Parse(revisions[params.Limit-1].ID)
		pageInfo.NextCursor = pagination.EncodeCursor(lastID)
		revisions = revisions[:params.Limit]
	}

	return revisions, pageInfo, nil
}

// Get retrieves a single revision by ID.
func (r *Repository) Get(ctx context.Context, id string) (*models.Revision, error) {
	var rev models.Revision
	if err := r.db.WithContext(ctx).First(&rev, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("finding revision: %w", err)
	}
	return &rev, nil
}

// Handler provides HTTP handlers for revision history.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new revision handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// List handles GET /v1/revisions/{entityType}/{entityID}.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityID := chi.URLParam(r, "entityID")

	if entityType == "" || entityID == "" {
		apierrors.BadRequest(w, "entity type and ID are required")
		return
	}

	// Validate entity type.
	if entityType != "thread" && entityType != "message" {
		apierrors.BadRequest(w, "entity type must be 'thread' or 'message'")
		return
	}

	params := pagination.Parse(r)
	revisions, pageInfo, err := h.repo.List(r.Context(), entityType, entityID, params)
	if err != nil {
		apierrors.InternalError(w, "failed to list revisions")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     revisions,
		PageInfo: pageInfo,
	})
}

// Get handles GET /v1/revisions/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rev, err := h.repo.Get(r.Context(), id)
	if err != nil {
		apierrors.InternalError(w, "failed to get revision")
		return
	}
	if rev == nil {
		apierrors.NotFound(w, "revision not found")
		return
	}
	response.JSON(w, http.StatusOK, rev)
}
