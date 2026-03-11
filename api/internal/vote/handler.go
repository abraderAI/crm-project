package vote

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
	"github.com/abraderAI/crm-project/api/pkg/response"
)

// Handler provides HTTP handlers for Vote operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Vote handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Toggle handles POST .../threads/{thread}/vote.
// Toggles the authenticated user's vote on a thread.
func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "thread")

	uc := auth.GetUserContext(r.Context())
	if uc == nil {
		apierrors.Unauthorized(w, "authentication required")
		return
	}

	// Default role and tier — callers may extend this with RBAC lookup.
	role := models.RoleViewer
	billingTier := "free"

	result, err := h.service.Toggle(r.Context(), threadID, uc.UserID, role, billingTier)
	if err != nil {
		if err.Error() == "thread not found" {
			apierrors.NotFound(w, "thread not found")
			return
		}
		apierrors.InternalError(w, "failed to toggle vote")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetWeightTable handles GET .../vote/weights.
// Returns the current vote weight configuration.
func (h *Handler) GetWeightTable(w http.ResponseWriter, r *http.Request) {
	wc := h.service.GetWeightConfig()
	response.JSON(w, http.StatusOK, map[string]any{
		"role_weights":   wc.RoleWeights,
		"tier_bonuses":   wc.TierBonuses,
		"default_weight": wc.DefaultWeight,
	})
}
