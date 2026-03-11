package provision

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for provisioning operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new provisioning handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Provision handles POST .../threads/{thread}/provision.
func (h *Handler) Provision(w http.ResponseWriter, r *http.Request) {
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

	var input ProvisionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Input is optional — allow empty body.
		input = ProvisionInput{}
	}

	result, err := h.service.ProvisionCustomer(r.Context(), threadID, userID, input)
	if err != nil {
		switch err.Error() {
		case "thread not found":
			apierrors.NotFound(w, "thread not found")
		default:
			apierrors.ValidationError(w, err.Error(), nil)
		}
		return
	}

	response.Created(w, result)
}
