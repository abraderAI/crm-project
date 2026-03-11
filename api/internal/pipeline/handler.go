package pipeline

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for pipeline operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new pipeline handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// TransitionStage handles POST .../threads/{thread}/stage.
func (h *Handler) TransitionStage(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")
	if threadID == "" {
		apierrors.BadRequest(w, "thread identifier is required")
		return
	}

	uc := auth.GetUserContext(r.Context())
	userID := ""
	if uc != nil {
		userID = uc.UserID
	}

	var input TransitionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if input.Stage == "" {
		apierrors.ValidationError(w, "stage is required", nil)
		return
	}

	result, err := h.service.TransitionStage(r.Context(), threadID, input.Stage, userID)
	if err != nil {
		switch err.Error() {
		case "thread not found":
			apierrors.NotFound(w, "thread not found")
		default:
			apierrors.ValidationError(w, err.Error(), nil)
		}
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetStages handles GET /v1/orgs/{org}/pipeline/stages.
func (h *Handler) GetStages(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "org")

	stages := h.service.GetStages(r.Context(), orgID)
	response.JSON(w, http.StatusOK, map[string]any{"stages": stages})
}
