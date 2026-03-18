package globalspace

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for global space thread endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new global space Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ListThreads handles GET /v1/global-spaces/{space}/threads.
// Accepts optional query params: mine=true, org_id, limit, cursor.
// Authentication is required; tier enforcement is handled client-side.
func (h *Handler) ListThreads(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")
	params := pagination.Parse(r)
	q := r.URL.Query()

	uc := auth.GetUserContext(r.Context())
	userID := ""
	if uc != nil {
		userID = uc.UserID
	}

	input := ListInput{
		SpaceSlug: spaceSlug,
		Params:    params,
		UserID:    userID,
		Mine:      q.Get("mine") == "true",
		OrgID:     q.Get("org_id"),
	}

	threads, pageInfo, err := h.service.ListThreads(r.Context(), input)
	if err != nil {
		apierrors.InternalError(w, "failed to list threads")
		return
	}

	response.JSON(w, http.StatusOK, response.ListResponse{
		Data:     threads,
		PageInfo: pageInfo,
	})
}

// createThreadRequest is the request body for POST /v1/global-spaces/{space}/threads.
type createThreadRequest struct {
	Title string  `json:"title"`
	Body  string  `json:"body"`
	OrgID *string `json:"org_id"`
}

// CreateThread handles POST /v1/global-spaces/{space}/threads.
// Requires authentication. Tier enforcement is handled client-side.
func (h *Handler) CreateThread(w http.ResponseWriter, r *http.Request) {
	spaceSlug := chi.URLParam(r, "space")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	var req createThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}
	if req.Title == "" {
		apierrors.ValidationError(w, "title is required", nil)
		return
	}

	t, err := h.service.CreateThread(r.Context(), spaceSlug, uc.UserID, CreateInput(req))
	if err != nil {
		switch err.Error() {
		case "title is required":
			apierrors.ValidationError(w, err.Error(), nil)
		case "board is locked":
			apierrors.Forbidden(w, "board is locked; new threads cannot be created")
		default:
			apierrors.InternalError(w, "failed to create thread")
		}
		return
	}

	response.Created(w, t)
}
