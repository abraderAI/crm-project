package search

import (
	"net/http"
	"strings"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for search operations.
type Handler struct {
	repo *Repository
}

// NewHandler creates a new search handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// Search handles GET /v1/search?q=&type=&scope=.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		apierrors.BadRequest(w, "query parameter 'q' is required")
		return
	}

	// Parse optional type filter (comma-separated).
	var entityTypes []string
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		entityTypes = strings.Split(typeParam, ",")
	}

	params := pagination.Parse(r)

	results, pageInfo, err := h.repo.Search(r.Context(), query, entityTypes, params)
	if err != nil {
		apierrors.InternalError(w, "search failed")
		return
	}

	if results == nil {
		results = []SearchResult{}
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     results,
		PageInfo: pageInfo,
	})
}
